package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/DerekBelloni/go-socket-server/core"
	"github.com/DerekBelloni/go-socket-server/event"
	"github.com/gorilla/websocket"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

type RelayManager struct {
	connections   map[string]*websocket.Conn
	subscriptions map[string]string
	eventChans    map[string]chan string
	readChans     map[string]chan []byte
	writeChans    map[string]chan []byte
	contexts      map[string]context.Context
	cancelFuncs   map[string]context.CancelFunc
	mutex         sync.RWMutex
	Connector     core.RelayConnector
	SearchTracker core.SubscriptionTracker
}

func NewRelayManager(connector core.RelayConnector, searchTracker core.SubscriptionTracker) *RelayManager {
	return &RelayManager{
		connections:   make(map[string]*websocket.Conn),
		subscriptions: make(map[string]string),
		eventChans:    make(map[string]chan string),
		readChans:     make(map[string]chan []byte),
		writeChans:    make(map[string]chan []byte),
		contexts:      make(map[string]context.Context),
		cancelFuncs:   make(map[string]context.CancelFunc),
		SearchTracker: searchTracker,
		Connector:     connector,
	}
}

func (rm *RelayManager) GetConnection(relayUrl string) (chan []byte, chan string, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	_, writeChan, _, eventChan, err := rm.getExistingConnection(relayUrl)

	if err == nil && rm.isConnected(relayUrl) {
		return writeChan, eventChan, nil
	}

	return rm.createNewConnection(relayUrl)
}

func (rm *RelayManager) getExistingConnection(relayUrl string) (*websocket.Conn, chan []byte, chan []byte, chan string, error) {
	conn, connExists := rm.connections[relayUrl]
	writeChan, wChanExists := rm.writeChans[relayUrl]
	readChan, rChanExists := rm.readChans[relayUrl]
	eventChan, eChanExists := rm.eventChans[relayUrl]

	if !eChanExists || !wChanExists || !rChanExists || !connExists {
		return nil, nil, nil, nil, errors.New("no existing, valid connection")
	}

	return conn, writeChan, readChan, eventChan, nil
}

func (rm *RelayManager) createNewConnection(relayUrl string) (chan []byte, chan string, error) {
	conn, err := rm.createConnection(relayUrl)
	ctx, cancel := context.WithCancel(context.Background())

	if err != nil {
		return nil, nil, errors.New("could not establish new socket connection")
	}

	eventChan := make(chan string)
	readChan := make(chan []byte, 1000)
	writeChan := make(chan []byte)

	rm.eventChans[relayUrl] = eventChan
	rm.readChans[relayUrl] = readChan
	rm.writeChans[relayUrl] = writeChan
	rm.connections[relayUrl] = conn
	rm.contexts[relayUrl] = ctx
	rm.cancelFuncs[relayUrl] = cancel

	go rm.writeLoop(ctx, conn, relayUrl, writeChan)
	go rm.readLoop(ctx, conn, relayUrl, readChan)
	go rm.processReadChannel(ctx, readChan, relayUrl, eventChan)
	// go rm.connectionHeartbeat(relayUrl)
	return writeChan, eventChan, nil
}

func (rm *RelayManager) connectionHeartbeat(relayUrl string) {
	ticker := time.NewTicker(pingInterval)
	if conn, exists := rm.connections[relayUrl]; exists {
		<-ticker.C
		log.Println("ping")
		rm.mutex.Lock()
		if err := conn.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
			log.Println("Ping message error: ", err)
		}
		rm.mutex.Unlock()
	}

}

func (rm *RelayManager) createConnection(relayUrl string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		ReadBufferSize:  1024 * 1024,
		WriteBufferSize: 1024 * 1024,
	}

	conn, _, err := dialer.Dial(relayUrl, http.Header{})

	if err != nil {
		fmt.Printf("Error establishing a connection for relay: %v, %v\n", relayUrl, err)
		return nil, err
	}

	rm.connections[relayUrl] = conn
	return conn, nil
}

func (rm *RelayManager) deleteRelayMappings(relayUrl string) {
	delete(rm.eventChans, relayUrl)
	delete(rm.readChans, relayUrl)
	delete(rm.writeChans, relayUrl)
	delete(rm.connections, relayUrl)
	delete(rm.contexts, relayUrl)
	delete(rm.cancelFuncs, relayUrl)
}

func (rm *RelayManager) handleConnectionError(relayUrl string, err error) {
	switch {
	case websocket.IsUnexpectedCloseError(err):
		log.Printf("Connection closed unexpectedly: %v\n", err)
	case errors.Is(err, io.EOF):
		log.Printf("Connection closed normally\n")
	default:
		log.Printf("Connection error: %v\n", err)
	}
}

func (rm *RelayManager) handleChannelClosue(relayUrl string) {
	rm.mutex.Lock()
	cancelFunc, exists := rm.cancelFuncs[relayUrl]
	rm.mutex.Unlock()

	if exists {
		cancelFunc()
	}

	rm.CloseConnection(relayUrl)
}

func (rm *RelayManager) pongHandler(relayUrl string) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	err := rm.connections[relayUrl].SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		fmt.Printf("error setting read dead line for relay connection: %v\n", err)
	}

	rm.connections[relayUrl].SetPongHandler(func(string) error {
		log.Println("pong")
		return rm.connections[relayUrl].SetReadDeadline(time.Now().Add(pongWait))
	})

	return nil
}

func (rm *RelayManager) readLoop(ctx context.Context, conn *websocket.Conn, relayUrl string, readChan chan []byte) {
	// go rm.pongHandler(relayUrl)
	defer conn.Close()

	err := rm.connections[relayUrl].SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		fmt.Printf("error setting read dead line for relay connection: %v\n", err)
	}

	rm.connections[relayUrl].SetPongHandler(func(string) error {
		log.Println("pong")
		return rm.connections[relayUrl].SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		messageType, reader, err := conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error getting next reader from relay: %v, error: %v\n", relayUrl, err)
				rm.CloseConnection(relayUrl)
			}
			break
		}

		if messageType == websocket.TextMessage {
			message, err := io.ReadAll(reader)
			if err != nil {
				log.Printf("Error reading message from relay: %v, error: %v\n", relayUrl, err)
				continue
			}

			var jsonMessage []json.RawMessage
			if err := json.Unmarshal(message, &jsonMessage); err != nil {
				log.Printf("Error parsing JSON from relay: %v, error: %v\n", relayUrl, err)
				continue
			}
			select {
			case readChan <- message:
			default:
				var msg []interface{}
				_ = json.Unmarshal(message, &msg)
				fmt.Printf("[DROP] %s: Dropped message type: %v, buffer size: %d/100\n",
					relayUrl, msg[0], len(readChan))
			}
		} else if messageType == websocket.BinaryMessage {
			log.Printf("Received unexpected binary message from relay: %v\n", relayUrl)
		}
	}
}

func (rm *RelayManager) processReadChannel(ctx context.Context, readChan <-chan []byte, relayUrl string, eventChan chan string) {
	for msg := range readChan {
		var relayMessage []interface{}
		err := json.Unmarshal(msg, &relayMessage)

		if err != nil {
			log.Printf("Error unmarshalling relay message: %v\n", err)
			continue
		}

		rm.processMessage(relayMessage, relayUrl, eventChan)
	}
}

func (rm *RelayManager) processMessage(relayMessage []interface{}, relayUrl string, eventChan chan string) {
	if len(relayMessage) == 0 {
		return
	}

	relayMsgType := relayMessage[0]

	switch relayMsgType {
	case "EVENT":
		event.HandleEvent(relayMessage, eventChan, rm.Connector, relayUrl, rm.SearchTracker)
	case "NOTICE":
		event.HandleNotice(relayMessage)
	case "EOSE":
		event.HandleEOSE(relayMessage, relayUrl, eventChan)
	default:
		fmt.Printf("Unknown message type received: %v\n", relayMsgType)
	}
}

func (rm *RelayManager) writeLoop(ctx context.Context, conn *websocket.Conn, relayUrl string, writeChan <-chan []byte) {
	defer func() {
		log.Printf("Write loop ending for %s: ", relayUrl)
	}()

	for {
		select {
		case <-ctx.Done():
			rm.CloseConnection(relayUrl)
			return
		case msg, ok := <-writeChan:
			if !ok {
				rm.handleChannelClosue(relayUrl)
				return
			}

			rm.mutex.Lock()
			err := conn.WriteMessage(websocket.TextMessage, msg)
			rm.mutex.Unlock()

			if err != nil {
				rm.CloseConnection(relayUrl)
				rm.handleConnectionError(relayUrl, err)
				return
			}
		}
	}
}

func (rm *RelayManager) CloseConnection(relayUrl string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if relayConn, exists := rm.connections[relayUrl]; exists {
		// Try to send close message, but don't block too long
		relayConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		// Close the connection regardless of whether the message sent
		relayConn.Close()
		rm.deleteRelayMappings(relayUrl)
	}
}

func (rm *RelayManager) isConnected(relayUrl string) bool {
	conn, connExists := rm.connections[relayUrl]

	if !connExists {
		return false
	}

	return conn.UnderlyingConn() == nil
}
