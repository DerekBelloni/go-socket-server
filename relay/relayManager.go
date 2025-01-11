// package relay

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/DerekBelloni/go-socket-server/core"
// 	"github.com/DerekBelloni/go-socket-server/event"
// 	"github.com/gorilla/websocket"
// )

// var (
// 	pongWait     = 10 * time.Second
// 	pingInterval = (pongWait * 9) / 10
// )

// type RelayManager struct {
// 	connections   map[string]*websocket.Conn
// 	subscriptions map[string]string
// 	eventChans    map[string]chan string
// 	readChans     map[string]chan []byte
// 	writeChans    map[string]chan []byte
// 	contexts      map[string]context.Context
// 	cancelFuncs   map[string]context.CancelFunc
// 	mutex         sync.RWMutex
// 	Connector     core.RelayConnector
// 	SearchTracker core.SubscriptionTracker
// }

// func NewRelayManager(connector core.RelayConnector, searchTracker core.SubscriptionTracker) *RelayManager {
// 	return &RelayManager{
// 		connections:   make(map[string]*websocket.Conn),
// 		subscriptions: make(map[string]string),
// 		eventChans:    make(map[string]chan string),
// 		readChans:     make(map[string]chan []byte),
// 		writeChans:    make(map[string]chan []byte),
// 		contexts:      make(map[string]context.Context),
// 		cancelFuncs:   make(map[string]context.CancelFunc),
// 		SearchTracker: searchTracker,
// 		Connector:     connector,
// 	}
// }

// func (rm *RelayManager) GetConnection(relayUrl string) (chan []byte, chan string, error) {
// 	rm.mutex.Lock()
// 	_, writeChan, _, eventChan, err := rm.getExistingConnection(relayUrl)
// 	rm.mutex.Unlock()

// 	if err == nil && rm.isConnected(relayUrl) {
// 		fmt.Println("in is connected conditional")
// 		return writeChan, eventChan, nil
// 	}

// 	return rm.createNewConnection(relayUrl)
// }

// func (rm *RelayManager) getExistingConnection(relayUrl string) (*websocket.Conn, chan []byte, chan []byte, chan string, error) {
// 	conn, connExists := rm.connections[relayUrl]
// 	writeChan, wChanExists := rm.writeChans[relayUrl]
// 	readChan, rChanExists := rm.readChans[relayUrl]
// 	eventChan, eChanExists := rm.eventChans[relayUrl]

// 	if !eChanExists || !wChanExists || !rChanExists || !connExists {
// 		return nil, nil, nil, nil, errors.New("no existing, valid connection")
// 	}

// 	return conn, writeChan, readChan, eventChan, nil
// }

// func (rm *RelayManager) createNewConnection(relayUrl string) (chan []byte, chan string, error) {
// 	fmt.Println("new connection created")
// 	conn, err := rm.createConnection(relayUrl)
// 	ctx, cancel := context.WithCancel(context.Background())

// 	if err != nil {
// 		return nil, nil, errors.New("could not establish new socket connection")
// 	}

// 	eventChan := make(chan string)
// 	readChan := make(chan []byte, 1000)
// 	writeChan := make(chan []byte)

// 	rm.eventChans[relayUrl] = eventChan
// 	rm.readChans[relayUrl] = readChan
// 	rm.writeChans[relayUrl] = writeChan
// 	rm.connections[relayUrl] = conn
// 	rm.contexts[relayUrl] = ctx
// 	rm.cancelFuncs[relayUrl] = cancel

// 	go rm.writeLoop(ctx, conn, relayUrl, writeChan)
// 	go rm.readLoop(ctx, conn, relayUrl, readChan)
// 	go rm.processReadChannel(ctx, readChan, relayUrl, eventChan)
// 	// go rm.connectionHeartbeat(relayUrl)
// 	return writeChan, eventChan, nil
// }

// func (rm *RelayManager) createConnection(relayUrl string) (*websocket.Conn, error) {
// 	dialer := websocket.Dialer{
// 		ReadBufferSize:  1024 * 1024,
// 		WriteBufferSize: 1024 * 1024,
// 	}

// 	conn, _, err := dialer.Dial(relayUrl, http.Header{})

// 	if err != nil {
// 		fmt.Printf("Error establishing a connection for relay: %v, %v\n", relayUrl, err)
// 		return nil, err
// 	}

// 	rm.connections[relayUrl] = conn
// 	return conn, nil
// }

// func (rm *RelayManager) deleteRelayMappings(relayUrl string) {
// 	delete(rm.eventChans, relayUrl)
// 	delete(rm.readChans, relayUrl)
// 	delete(rm.writeChans, relayUrl)
// 	delete(rm.connections, relayUrl)
// 	delete(rm.contexts, relayUrl)
// 	delete(rm.cancelFuncs, relayUrl)
// }

// func (rm *RelayManager) handleConnectionError(relayUrl string, err error, errSource string) {
// 	switch {
// 	case websocket.IsUnexpectedCloseError(err):
// 		log.Printf("Connection closed unexpectedly: %v, error source: %v\n", err, errSource)
// 	case errors.Is(err, io.EOF):
// 		log.Printf("Connection closed normally, error source: %v\n", errSource)
// 	default:
// 		log.Printf("Connection error: %v, error source: %v\n", err, errSource)
// 	}
// }

// func (rm *RelayManager) handleChannelClosue(relayUrl string) {
// 	rm.mutex.Lock()
// 	cancelFunc, exists := rm.cancelFuncs[relayUrl]
// 	rm.mutex.Unlock()

// 	if exists {
// 		cancelFunc()
// 	}

// 	rm.CloseConnection(relayUrl)
// }

// func (rm *RelayManager) pingHandler(ctx context.Context, conn *websocket.Conn, relayUrl string) {
// 	ticker := time.NewTicker(pingInterval)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			rm.mutex.Lock()
// 			rm.CloseConnection(relayUrl)
// 			rm.mutex.Unlock()
// 			return
// 		case <-ticker.C:
// 			// log.Printf("Sending PING to %s", relayUrl) // Add logging
// 			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
// 				rm.mutex.Lock()
// 				rm.handleConnectionError(relayUrl, err, "pingHandler")
// 				rm.CloseConnection(relayUrl)
// 				rm.mutex.Unlock()
// 				return
// 			}
// 		}
// 	}
// }

// func (rm *RelayManager) pongHandler(conn *websocket.Conn) error {
// 	conn.SetPongHandler(func(string) error {
// 		// log.Printf("Received PONG") // Add logging
// 		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
// 			// log.Printf("Error setting read deadline in pong handler: %v", err) // Add logging
// 			return err
// 		}
// 		// log.Printf("Successfully set new read deadline")
// 		return nil
// 	})
// 	return nil
// }

// func (rm *RelayManager) readLoop(ctx context.Context, conn *websocket.Conn, relayUrl string, readChan chan []byte) {
// 	log.Printf("[%s] Starting new read loop", relayUrl)

// 	// go rm.pingHandler(ctx, conn, relayUrl)
// 	// rm.pongHandler(conn)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			log.Printf("[%s] Context cancelled, closing connection", relayUrl)
// 			rm.CloseConnection(relayUrl)
// 			return
// 		default:
// 			rm.mutex.Lock()
// 			messageType, reader, err := conn.NextReader()
// 			rm.mutex.Unlock()
// 			if err != nil {
// 				rm.handleConnectionError(relayUrl, err, "readLoop")
// 				rm.CloseConnection(relayUrl)

// 				rm.mutex.Lock()
// 				remainingConns := len(rm.connections)
// 				rm.mutex.Unlock()
// 				log.Printf("[%s] Connection closed, remaining connections: %d", relayUrl, remainingConns)
// 				return
// 			}

// 			if messageType == websocket.TextMessage {
// 				message, err := io.ReadAll(reader)
// 				fmt.Printf("message in read loop %v\n", string(message))
// 				if err != nil {
// 					log.Printf("Error reading message from relay: %v, error: %v\n", relayUrl, err)
// 					continue
// 				}

// 				var jsonMessage []json.RawMessage
// 				if err := json.Unmarshal(message, &jsonMessage); err != nil {
// 					log.Printf("Error parsing JSON from relay: %v, error: %v\n", relayUrl, err)
// 					continue
// 				}
// 				select {
// 				case readChan <- message:
// 				default:
// 					var msg []interface{}
// 					_ = json.Unmarshal(message, &msg)
// 					fmt.Printf("[DROP] %s: Dropped message type: %v, buffer size: %d/100\n",
// 						relayUrl, msg[0], len(readChan))
// 				}
// 			} else if messageType == websocket.BinaryMessage {
// 				log.Printf("Received unexpected binary message from relay: %v\n", relayUrl)
// 			}
// 		}
// 	}
// }

// func (rm *RelayManager) processReadChannel(ctx context.Context, readChan <-chan []byte, relayUrl string, eventChan chan string) {
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			rm.CloseConnection(relayUrl)
// 			return
// 		case msg, ok := <-readChan:
// 			if !ok {
// 				rm.handleChannelClosue(relayUrl)
// 				return
// 			}
// 			var relayMessage []interface{}
// 			err := json.Unmarshal(msg, &relayMessage)

// 			if err != nil {
// 				log.Printf("Error unmarshalling relay message: %v\n", err)
// 				continue
// 			}

// 			rm.processMessage(relayMessage, relayUrl, eventChan)
// 		}
// 	}
// }

// func (rm *RelayManager) processMessage(relayMessage []interface{}, relayUrl string, eventChan chan string) {
// 	if len(relayMessage) == 0 {
// 		return
// 	}

// 	relayMsgType := relayMessage[0]

// 	switch relayMsgType {
// 	case "EVENT":
// 		event.HandleEvent(relayMessage, eventChan, rm.Connector, relayUrl, rm.SearchTracker)
// 	case "NOTICE":
// 		event.HandleNotice(relayMessage)
// 	case "EOSE":
// 		event.HandleEOSE(relayMessage, relayUrl, eventChan)
// 	default:
// 		fmt.Printf("Unknown message type received: %v\n", relayMsgType)
// 	}
// }

// func (rm *RelayManager) writeLoop(ctx context.Context, conn *websocket.Conn, relayUrl string, writeChan <-chan []byte) {
// 	defer func() {
// 		log.Printf("Write loop ending for %s: ", relayUrl)
// 	}()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			rm.CloseConnection(relayUrl)
// 			return
// 		case msg, ok := <-writeChan:
// 			if !ok {
// 				rm.handleChannelClosue(relayUrl)
// 				return
// 			}
// 			fmt.Printf("message in write loop: %v\n", string(msg))
// 			rm.mutex.Lock()
// 			err := conn.WriteMessage(websocket.TextMessage, msg)
// 			rm.mutex.Unlock()

// 			if err != nil {
// 				rm.CloseConnection(relayUrl)
// 				rm.handleConnectionError(relayUrl, err, "writeLoop")
// 				return
// 			}
// 		}
// 	}
// }

// func (rm *RelayManager) CloseConnection(relayUrl string) {
// 	rm.mutex.Lock()
// 	defer rm.mutex.Unlock()
// 	fmt.Printf("closing client connection: %v\n", relayUrl)
// 	if relayConn, exists := rm.connections[relayUrl]; exists {
// 		relayConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
// 		relayConn.Close()
// 		rm.deleteRelayMappings(relayUrl)
// 	}
// }

// func (rm *RelayManager) isConnected(relayUrl string) bool {
// 	rm.mutex.Lock()
// 	defer rm.mutex.Unlock()
// 	conn, connExists := rm.connections[relayUrl]

// 	if !connExists {
// 		return false
// 	}

// 	return conn.UnderlyingConn() != nil
// }

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
	mutex         sync.RWMutex
	eventChans    map[string]chan string
	readChans     map[string]chan []byte
	writeChans    map[string]chan []byte
	//
	contexts      map[string]context.Context
	cancelFuncs   map[string]context.CancelFunc
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
		//
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

	go rm.writeLoop(conn, relayUrl, writeChan)
	go rm.readLoop(conn, relayUrl, readChan)
	go rm.processReadChannel(readChan, relayUrl, eventChan)
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

func (rm *RelayManager) readLoop(conn *websocket.Conn, relayUrl string, readChan chan []byte) {
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
				// fmt.Printf("[READ] %s: Message queued, buffer size: %d/100\n", relayUrl, len(readChan))
			default:
				// Log enough to identify what we're dropping
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

func (rm *RelayManager) processReadChannel(readChan <-chan []byte, relayUrl string, eventChan chan string) {
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

func (rm *RelayManager) writeLoop(conn *websocket.Conn, relayUrl string, writeChan <-chan []byte) {
	for {
		for msg := range writeChan {
			rm.mutex.Lock()
			err := conn.WriteMessage(websocket.TextMessage, msg)
			rm.mutex.Unlock()
			if err != nil {
				fmt.Printf("Error sending subscription request to relay: %v\n", err)
				break
			}
		}
	}
}

func (rm *RelayManager) CloseConnection(relayUrl string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if relayConn, exists := rm.connections[relayUrl]; exists {
		relayConn.Close()
		delete(rm.connections, relayUrl)
	}
}

func (rm *RelayManager) isConnected(relayUrl string) bool {
	conn, connExists := rm.connections[relayUrl]
	if !connExists {
		return false
	}

	return conn.UnderlyingConn() != nil
}
