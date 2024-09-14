package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

type RelayManager struct {
	connections map[string]*websocket.Conn
	mutex       sync.RWMutex
	writeChans  map[string]chan []byte
	readChans   map[string]chan []byte
}

func NewRelayManager() *RelayManager {
	return &RelayManager{
		connections: make(map[string]*websocket.Conn),
		writeChans:  make(map[string]chan []byte),
		readChans:   make(map[string]chan []byte),
	}
}

func (rm *RelayManager) GetConnection(relayUrl string) (chan []byte, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	conn, writeChan, err := rm.getExistingConnection(relayUrl)

	if err == nil {
		go rm.writeLoop(conn, relayUrl, writeChan)
		go rm.readLoop(conn, relayUrl)
		return writeChan, nil
	}

	return rm.createNewConnection(relayUrl)
}

func (rm *RelayManager) getExistingConnection(relayUrl string) (*websocket.Conn, chan []byte, error) {
	conn, connExists := rm.connections[relayUrl]
	writeChan, chanExists := rm.writeChans[relayUrl]

	if !chanExists || !connExists {
		return nil, nil, errors.New("no existing, valid connection")
	}

	return conn, writeChan, nil
}

func (rm *RelayManager) createNewConnection(relayUrl string) (chan []byte, error) {
	conn, err := rm.createConnection(relayUrl)

	if err != nil {
		return nil, errors.New("could not establish new socket connection")
	}

	writeChan := make(chan []byte)
	readChan := make(chan []byte)
	rm.readChans[relayUrl] = readChan
	rm.writeChans[relayUrl] = writeChan
	rm.connections[relayUrl] = conn

	go rm.writeLoop(conn, relayUrl, writeChan)
	go rm.readLoop(conn, relayUrl)
	return writeChan, nil
}

func (rm *RelayManager) createConnection(relayUrl string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)

	if err != nil {
		fmt.Printf("Error establishing a connection for relay: %v, %v\n", relayUrl, err)
		return nil, err
	}

	rm.connections[relayUrl] = conn
	return conn, nil
}

func (rm *RelayManager) monitorConnection(relayUrl string) {

}

func (rm *RelayManager) readLoop(conn *websocket.Conn, relayUrl string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Error reading message from relay: %v, error: %v\n", relayUrl, err)
			continue
		}
		var metadata []interface{}
		if err := json.Unmarshal(msg, &metadata); err != nil {
			fmt.Printf("Error unmarshalling relay message")
			continue
		}
		fmt.Printf("metadata: %v\n", metadata...)
	}
}

func (rm *RelayManager) writeLoop(conn *websocket.Conn, relayUrl string, writeChan <-chan []byte) {
	for {
		for msg := range writeChan {
			err := conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				fmt.Printf("Error sending subscription request to relay: %v\n", err)
				continue
			}
		}
	}
}

func (rm *RelayManager) CloseAllConnections() {

}

func (rm *RelayManager) isConnected(conn *websocket.Conn) bool {
	return true
}
