package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/DerekBelloni/go-socket-server/internal/core"
	"github.com/DerekBelloni/go-socket-server/internal/handler"
	"github.com/gorilla/websocket"
)

type RelayManager struct {
	connections    map[string]*websocket.Conn
	mutex          sync.RWMutex
	eventChans     map[string]chan string
	readChans      map[string]chan []byte
	writeChans     map[string]chan []byte
	noRelaysActive chan string
	activeRelays   []string
	Connector      core.RelayConnector
}

func NewRelayManager(connector core.RelayConnector) *RelayManager {
	return &RelayManager{
		connections:  make(map[string]*websocket.Conn),
		eventChans:   make(map[string]chan string),
		readChans:    make(map[string]chan []byte),
		writeChans:   make(map[string]chan []byte),
		activeRelays: make([]string, 10),
		Connector:    connector,
	}
}

func (rm *RelayManager) GetConnection(relayUrl string) (chan []byte, chan string, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	conn, writeChan, readChan, eventChan, err := rm.getExistingConnection(relayUrl)

	if err == nil && rm.isConnected(relayUrl) {
		rm.activeRelays = append(rm.activeRelays, relayUrl)
		go rm.writeLoop(conn, relayUrl, writeChan)
		go rm.readLoop(conn, relayUrl, readChan)
		go rm.processReadChannel(readChan, relayUrl, eventChan)
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
	readChan := make(chan []byte)
	writeChan := make(chan []byte)

	rm.eventChans[relayUrl] = eventChan
	rm.readChans[relayUrl] = readChan
	rm.writeChans[relayUrl] = writeChan
	rm.connections[relayUrl] = conn
	rm.activeRelays = append(rm.activeRelays, relayUrl)

	go rm.writeLoop(conn, relayUrl, writeChan)
	go rm.readLoop(conn, relayUrl, readChan)
	go rm.processReadChannel(readChan, relayUrl, eventChan)
	return writeChan, eventChan, nil
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

func (rm *RelayManager) monitorConnection(relayUrl string) {

}

func (rm *RelayManager) readLoop(conn *websocket.Conn, relayUrl string, readChan chan []byte) {
	defer func() {
		rm.connections[relayUrl].Close()
	}()

	for {
		messageType, reader, err := conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error getting next reader from relay: %v, error: %v\n", relayUrl, err)
				// close connection
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
				fmt.Println("Message read to read channel")
			default:
				fmt.Printf("Read channel is full, discarding message from: %v\n\n", relayUrl)
			}
		} else if messageType == websocket.BinaryMessage {
			log.Printf("Received unexpected binary message from relay: %v\n", relayUrl)
		}
		time.Sleep(2 * time.Second)
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
		handler.HandleEvent(relayMessage, eventChan, rm.Connector, relayUrl)
	case "NOTICE":
		handler.HandleNotice(relayMessage)
	case "EOSE":
		handler.HandleEOSE(relayMessage, relayUrl, eventChan)
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

	return conn.UnderlyingConn() == nil
}
