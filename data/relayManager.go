package data

import (
	"errors"
	"fmt"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/websocket"
)

type RelayManager struct {
	connections map[string]*websocket.Conn
	mutex       sync.RWMutex
	writeChans  map[string]chan []byte
}

func NewRelayManager() *RelayManager {
	return &RelayManager{
		connections: make(map[string]*websocket.Conn),
		writeChans:  make(map[string]chan []byte),
	}
}

func (rm *RelayManager) GetConnection(relayUrl string) (*websocket.Conn, chan []byte, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	conn, writeChan, err := rm.getExistingConnection(relayUrl)

	if err == nil {
		go rm.writeLoop(relayUrl, writeChan)
		return conn, writeChan, nil
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

func (rm *RelayManager) createNewConnection(relayUrl string) (*websocket.Conn, chan []byte, error) {
	conn, err := rm.createConnection(relayUrl)

	if err != nil {
		return nil, nil, errors.New("could not establish new socket connection")
	}

	writeChan := make(chan []byte)
	rm.writeChans[relayUrl] = writeChan
	rm.connections[relayUrl] = conn

	go rm.writeLoop(relayUrl, writeChan)
	return conn, writeChan, nil
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

func (rm *RelayManager) readLoop(relayUrl string) {

}

func (rm *RelayManager) writeLoop(relayUrl string, writeChan <-chan []byte) {
	fmt.Println("in write loop")
	for {
		for msg := range writeChan {
			spew.Dump("message: ", msg)
		}
	}
}

func (rm *RelayManager) CloseAllConnections() {

}

func (rm *RelayManager) isConnected(conn *websocket.Conn) bool {
	return true
}
