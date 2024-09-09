package relay

import (
	"context"

	"github.com/DerekBelloni/go-socket-server/data"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var relayManager *data.RelayManager

func init() {
	relayManager = data.NewRelayManager()
}

func getConnection(relayUrl string) (*websocket.Conn, error) {
	return relayManager.GetConnection(relayUrl)
}

func GetUserMetadata(relayUrl string, finished chan<- string, mqMsgType string, userHexKey string) {
	// conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	log := logrus.WithField("relay", relayUrl)

	conn, err := getConnection(relayUrl)
	if err != nil {
		log.Error("Dial error: ", err)
	}
	// defer conn.Close()

	handleMetadata(conn, relayUrl, finished, userHexKey)
}

func SendNoteToRelay(relayUrl string, newNote data.NewNote, noteFinished chan<- string) {
	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	// conn, err := getConnection(relayUrl)
	log := logrus.WithField("relay", relayUrl)
	if err != nil {
		log.Error("Dial error: ", err)
	}
	defer conn.Close()

	handleNewNote(conn, relayUrl, newNote, noteFinished)
}

func GetUserNotes(ctx context.Context, cancel context.CancelFunc, relayUrl string, userHexKey string) {
	// conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	log := logrus.WithField("user notes, relay", relayUrl)
	conn, err := getConnection(relayUrl)
	if err != nil {
		log.Error("Dial error: ", err)
	}
	// defer conn.Close()

	handleUserNotes(ctx, cancel, conn, relayUrl, userHexKey)
}

func GetClassifiedListings(relayUrl string) {
	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	// conn, err := getConnection(relayUrl)
	log := logrus.WithField("relay", relayUrl)
	if err != nil {
		log.Error("Dial error: ", err)
	}
	defer conn.Close()

	handleClassifiedListings(conn, relayUrl)
}

func GetFollowList(ctx context.Context, cancel context.CancelFunc, relayUrl string, userHexKey string) {
	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	log := logrus.WithField("follow list, relay", relayUrl)
	// conn, err := getConnection(relayUrl)
	if err != nil {
		log.Error("Dial error: ", err)
	}
	defer conn.Close()

	handleFollowList(ctx, cancel, conn, relayUrl, userHexKey)
}
