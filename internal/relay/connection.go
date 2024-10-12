package relay

import (
	"fmt"

	"github.com/DerekBelloni/go-socket-server/internal/data"
	"github.com/DerekBelloni/go-socket-server/internal/handler"
	"github.com/DerekBelloni/go-socket-server/internal/subscriptions"
)

type RelayConnection struct {
	relayManager *data.RelayManager
}

func NewRelayConnection(manager *data.RelayManager) *RelayConnection {
	return &RelayConnection{relayManager: manager}
}

func (rc *RelayConnection) GetConnection(relayUrl string) (chan []byte, chan string, error) {
	return rc.relayManager.GetConnection(relayUrl)
}

func (rc *RelayConnection) GetUserMetadata(relayUrl string, userHexKey string, metadataFinished chan<- string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	subscriptions.MetadataSubscription(relayUrl, userHexKey, writeChan, eventChan, metadataFinished)
}

func (rc *RelayConnection) GetUserNotes(relayUrl string, userHexKey string, notesFinished chan<- string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)
	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	subscriptions.UserNotesSubscription(relayUrl, userHexKey, writeChan, eventChan, notesFinished)
}

func (rc *RelayConnection) GetFollowList(relayUrl string, userHexKey string, followsFinished chan<- string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	subscriptions.FollowListSubscription(relayUrl, userHexKey, writeChan, eventChan, followsFinished)
}

// add the user key context here
func (rc *RelayConnection) GetFollowListMetadata(relayUrl string, userHexKey string) {
	handler.HandleFollowListPubKeys(userHexKey)

	// writeChan, eventChan, err := rc.GetConnection(relayUrl)

	// if err != nil {
	// 	fmt.Printf("Diale error: %v\n", err)
	// }

	// subscriptions.FollowListMetadataSubscription(relayUrl, pubKeys, writeChan, eventChan)
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
