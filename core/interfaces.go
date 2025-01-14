package core

import "github.com/DerekBelloni/go-socket-server/data"

type RelayConnector interface {
	GetConnection(relayUrl string) (chan []byte, chan string, error)
	GetUserMetadata(relayUrl string, userHexKey string, metadataFinished chan<- string)
	GetUserNotes(relayUrl string, userHexKey string, notesFinished chan<- string)
	GetFollowList(relayUrl string, userHexKey string, followsFinished chan<- string)
	GetFollowListMetadata(relayUrl string, userHexKey string, pubKeys []string, subscriptionTracker SubscriptionTracker)
	GetSearchedAuthorMetadata(relayUrl string, authorPubKey string, searchKey string, subscriptionTracker SubscriptionTracker)
	RetrieveSearch(relayUrl string, search string, subscriptionTracker SubscriptionTracker, uuid string, pubkey string)
	SendNoteToRelay(relayUrl string, newNote data.NewNote)
}

type SubscriptionTracker interface {
	InFollowsMetadtaMapping(event []interface{}) (string, string, string, bool)
	InSearchEvent(event []interface{}, eventKind string) (string, bool)
	InSubscriptionMapping(event []interface{}) (string, bool)
	AddSearch(search string, uuid string, subscriptionId string, pubkey string)
	AddSubscription(subscriptionID string, userPubkey string, followsPubkey string, uuid string)
	FollowsMetadataSubscription(subscriptionID string, followsPubkey string, userPubkey string, subscriptionType string)
	RemoveSearch(search string)
}
