package relay

import (
	"fmt"

	"github.com/DerekBelloni/go-socket-server/internal/data"
	"github.com/DerekBelloni/go-socket-server/internal/handler"
)

var relayManager *data.RelayManager

func init() {
	relayManager = data.NewRelayManager()
}

func initConnection(relayUrl string) (chan []byte, chan string, error) {
	return relayManager.GetConnection(relayUrl)
}

func GetUserMetadata(relayUrl string, userHexKey string, metadataFinished chan<- string) {
	writeChan, eventChan, err := initConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	handler.MetadataSubscription(relayUrl, userHexKey, writeChan, eventChan, metadataFinished)
}

func GetUserNotes(relayUrl string, userHexKey string, notesFinished chan<- string) {
	writeChan, eventChan, err := initConnection(relayUrl)
	fmt.Println("user notes in connection.go")
	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	handler.UserNotesSubscription(relayUrl, userHexKey, writeChan, eventChan, notesFinished)
}

func GetFollowList(relayUrl string, userHexKey string, followsFinished chan<- string) {
	writeChan, eventChan, err := initConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	handler.FollowListSubscription(relayUrl, userHexKey, writeChan, eventChan, followsFinished)
}

func GetFollowListMetadata(relayUrl string, pubKeys []string) {

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
