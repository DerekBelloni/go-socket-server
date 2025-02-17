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

func (rc *RelayConnection) GetUserMetadata(relayUrl string, userHexKey string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v, method: %v:\n", err, "getUserMetadata")
	}

	subscriptions.MetadataSubscription(relayUrl, userHexKey, writeChan, eventChan)
}

func (rc *RelayConnection) GetUserNotes(relayUrl string, userHexKey string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)
	if err != nil {
		fmt.Printf("Dial error: %v, method: %v:\n", err, "getUserNotes")
	}

	subscriptions.UserNotesSubscription(relayUrl, userHexKey, writeChan, eventChan)
}

func (rc *RelayConnection) GetFollowList(relayUrl string, userHexKey string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v, method: %v:\n", err, "getFollowList")
	}

	subscriptions.FollowListSubscription(relayUrl, userHexKey, writeChan, eventChan)
}

// add the user key context here
func (rc *RelayConnection) GetFollowListMetadata(relayUrl string, userHexKey string, pubKeys []string, subscriptionTracker core.SubscriptionTracker) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v, method: %v:\n", err, "getFollowListMetadata")
	}

	subscriptions.FollowListMetadataSubscription(relayUrl, pubKeys, userHexKey, writeChan, eventChan, subscriptionTracker)
}

func (rc *RelayConnection) GetFollowsNotes(relayUrl string, userPubkey string, followsPubkey string, subscriptionTracker core.SubscriptionTracker, uuid string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v, method: %v, relay: %v\n", err, "getFollowsNotes", relayUrl)
	}

	subscriptions.FollowsNotesSubscription(relayUrl, userPubkey, followsPubkey, subscriptionTracker, writeChan, eventChan, uuid)
}

func (rc *RelayConnection) GetSearchedAuthorMetadata(relayUrl string, authorPubkey string, searchKey string, subscriptionTracker core.SubscriptionTracker) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)

	if err != nil {
		fmt.Printf("Dial error: %v, method: %v:\n", err, "getSearchedAuthorMetadata")
	}

	subscriptions.SearchedAuthorMetadata(relayUrl, authorPubkey, searchKey, subscriptionTracker, writeChan, eventChan)
}

func (rc *RelayConnection) SendNoteToRelay(relayUrl string, newNote data.NewNote) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)
	if err != nil {
		fmt.Printf("Dial error: %v, method: %v:\n", err, "sendNoteToRelay")
	}

	subscriptions.CreateNoteEvent(relayUrl, newNote, writeChan, eventChan)
}

func (rc *RelayConnection) RetrieveSearch(relayUrl string, search string, subscriptionTracker core.SubscriptionTracker, uuid string, pubkey string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)
	if err != nil {
		fmt.Printf("Dial error: %v, method: %v:\n", err, "retrieveSearch")
	}
	subscriptions.RetrieveSearchSubscription(relayUrl, search, writeChan, eventChan, subscriptionTracker, uuid, pubkey)
}

func (rc *RelayConnection) RetrieveEmbeddedEntity(eventId string, hex string, identifier string, relayUrl string, uuid string) {
	writeChan, eventChan, err := rc.GetConnection(relayUrl)
	if err != nil {
		fmt.Printf("Dial error: %v, method: %v\n", err, "retrieveEmeddedEntity")
	}
	trackerManager := rc.relayManager.TrackerManager
	subscriptions.RetrieveEmbeddedEntity(eventId, hex, identifier, relayUrl, uuid, writeChan, eventChan, trackerManager)
}
