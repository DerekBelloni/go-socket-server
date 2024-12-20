package core

import "github.com/DerekBelloni/go-socket-server/data"

type RelayConnector interface {
	GetConnection(relayUrl string) (chan []byte, chan string, error)
	GetUserMetadata(relayUrl string, userHexKey string, metadataFinished chan<- string)
	GetUserNotes(relayUrl string, userHexKey string, notesFinished chan<- string)
	GetFollowList(relayUrl string, userHexKey string, followsFinished chan<- string)
	GetFollowListMetadata(relayUrl string, userHexKey string, pubKeys []string)
	SendNoteToRelay(relayUrl string, newNote data.NewNote)
}

// think I can get rid of this
type SubscriptionTracker interface {
	InSearchEvent(event []interface{}) (string, bool)
	InSubscriptionMapping(event []interface{}) (string, bool)
	AddSearch(search string, uuid string, subscriptionId string, pubkey *string)
	AddSubscription(subscriptionID string, userPubkey string, followsPubkey string)
	RemoveSearch(search string)
}
