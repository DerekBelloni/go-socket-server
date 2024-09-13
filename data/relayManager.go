package data

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type RelayManager struct {
	connections map[string]*websocket.Conn
	mutex       sync.RWMutex
	log         *logrus.Logger
	// readChans map[string]chan
	writeChans map[string]chan []byte
}

func NewRelayManager() *RelayManager {
	return &RelayManager{
		connections: make(map[string]*websocket.Conn),
		log:         logrus.New(),
	}
}

func (rm *RelayManager) GetConnection(relayUrl string) (*websocket.Conn, map[string]chan []byte, error) {
	rm.mutex.Lock()
	conn, exists := rm.connections[relayUrl]
	rm.mutex.Unlock()

	if exists && rm.isConnected(conn) {
		return conn, rm.writeChans, nil
	}

	return rm.createConnection(relayUrl)
}

func (rm *RelayManager) createConnection(relayUrl string) (*websocket.Conn, map[string]chan []byte, error) {
	rm.mutex.Lock()
	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	rm.mutex.Unlock()

	if err != nil {
		fmt.Printf("Error establishing a connection for relay: %v, %v\n", relayUrl, err)
		return nil, nil, err
	}

	rm.log.Info("Connection created for relay: %v\n", relayUrl)

	rm.connections[relayUrl] = conn
	return conn, rm.writeChans, nil
}

func (rm *RelayManager) monitorConnection(relayUrl string) {

}

func (rm *RelayManager) readLoop(relayUrl string) {

}

func (rm *RelayManager) writeLoop(relayUrl string) {

}

func (rm *RelayManager) CloseAllConnections() {

}

func (rm *RelayManager) isConnected(conn *websocket.Conn) bool {
	return true
}
