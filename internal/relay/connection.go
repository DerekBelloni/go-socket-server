package relay

import (
	"fmt"

	"github.com/DerekBelloni/go-socket-server/data"
)

var relayManager *data.RelayManager

func init() {
	relayManager = data.NewRelayManager()
}

func initConnection(relayUrl string) (chan []byte, error) {
	return relayManager.GetConnection(relayUrl)
}

func GetUserMetadata(relayUrl string, finished chan<- string, mqMsgType string, userHexKey string, metadataSet chan<- string) {
	fmt.Print("in connection.go\n")

	writeChan, err := initConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}
	MetadataSubscription(relayUrl, userHexKey, writeChan)
	// handleMetadata(conn, relayUrl, finished, userHexKey, metadataSet)
}

// func SendNoteToRelay(relayUrl string, newNote data.NewNote, noteFinished chan<- string) {
// 	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
// 	// conn, err := getConnection(relayUrl)
// 	log := logrus.WithField("relay", relayUrl)
// 	if err != nil {
// 		log.Error("Dial error: ", err)
// 	}
// 	defer conn.Close()

// 	handleNewNote(conn, relayUrl, newNote, noteFinished)
// }

// func GetUserNotes(ctx context.Context, cancel context.CancelFunc, relayUrl string, userHexKey string, notesFinished chan<- string) {
// 	log := logrus.WithField("user notes, relay", relayUrl)
// 	conn, writeChan, err := getConnection(relayUrl)
// 	if err != nil {
// 		log.Error("Dial error: ", err)
// 	}

// 	UserNotesSubscription(conn, relayUrl, userHexKey, writeChan)
// 	// handleUserNotes(ctx, cancel, conn, relayUrl, userHexKey, notesFinished)
// }

// func GetClassifiedListings(relayUrl string) {
// 	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
// 	// conn, err := getConnection(relayUrl)
// 	log := logrus.WithField("relay", relayUrl)
// 	if err != nil {
// 		log.Error("Dial error: ", err)
// 	}
// 	defer conn.Close()

// 	handleClassifiedListings(conn, relayUrl)
// }

// func GetFollowList(ctx context.Context, cancel context.CancelFunc, relayUrl string, userHexKey string) {
// 	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
// 	log := logrus.WithField("follow list, relay", relayUrl)
// 	// conn, err := getConnection(relayUrl)
// 	if err != nil {
// 		log.Error("Dial error: ", err)
// 	}
// 	defer conn.Close()

// 	handleFollowList(ctx, cancel, conn, relayUrl, userHexKey)
// }
