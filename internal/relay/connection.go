package relay

import (
	"log"

	"github.com/DerekBelloni/go-socket-server/data"
	"github.com/gorilla/websocket"
)

func ConnectToRelay(relayUrl string, finished chan<- string, mqMsgType string, userHexKey string) {
	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	if err != nil {
		log.Fatal("Dial error: ", err)
	}
	defer conn.Close()

	handleRelayConnection(conn, relayUrl, finished, userHexKey)
}

func SendNoteToRelay(relayUrl string, newNote data.NewNote, noteFinished chan<- string) {
	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	if err != nil {
		log.Fatal("Dial error: ", err)
	}
	defer conn.Close()

	handleNewNote(conn, relayUrl, newNote, noteFinished)
}
