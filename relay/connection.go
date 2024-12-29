package relay

import (
	"fmt"

	"github.com/DerekBelloni/go-socket-server/core"
	"github.com/DerekBelloni/go-socket-server/data"
	"github.com/DerekBelloni/go-socket-server/subscriptions"
)

// I dont need this struct to hold a relayManager
type RelayConnection struct {
	relayManager *RelayManager
}

func NewRelayConnection(manager *RelayManager) *RelayConnection {
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
func (rc *RelayConnection) GetFollowListMetadata(relayUrl string, userHexKey string, pubKeys []string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	subscriptions.FollowListMetadataSubscription(relayUrl, pubKeys, writeChan, eventChan)
}

func (rc *RelayConnection) GetFollowsNotes(relayUrl string, userPubkey string, followsPubkey string, subscriptionTracker core.SubscriptionTracker, uuid string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	subscriptions.FollowsNotesSubscription(relayUrl, userPubkey, followsPubkey, subscriptionTracker, writeChan, eventChan, uuid)
}

func (rc *RelayConnection) GetSearchedAuthorMetadata(relayUrl string, authorPubkey string, userPubkey string, uuid string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	subscriptions.SearchedAuthorMetadata(relayUrl, authorPubkey, writeChan, eventChan)
}

func (rc *RelayConnection) SendNoteToRelay(relayUrl string, newNote data.NewNote) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)
	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}

	subscriptions.CreateNoteEvent(relayUrl, newNote, writeChan, eventChan)
}

func (rc *RelayConnection) RetrieveSearch(relayUrl string, search string, subscriptionTracker core.SubscriptionTracker, uuid string, pubkey *string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)
	if err != nil {
		fmt.Printf("Dial error: %v\n", err)
	}
	subscriptions.RetrieveSearchSubscription(relayUrl, search, writeChan, eventChan, subscriptionTracker, uuid, pubkey)
}
