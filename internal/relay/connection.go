package relay

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

func ConnectToRelay(relayUrl string) {
	fmt.Printf("landing in the connection.go file: %s\n", relayUrl)

	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	if err != nil {
		log.Fatal("Dial error: ", err)
	}
	defer conn.Close()

	handleRelayConnection(conn, relayUrl)
}
