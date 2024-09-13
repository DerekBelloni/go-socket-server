package relay

import "github.com/gorilla/websocket"

func MetadataSubscription(conn *websocket.Conn, relayUrl string, userHexKey string, writeChans map[string]chan []byte) {

}

func UserNotesSubscription(conn *websocket.Conn, relayUrl string, userHexKey string, writeChans map[string]chan []byte) {

}
