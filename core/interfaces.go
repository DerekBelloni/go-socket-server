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

type SearchTracker interface {
	InSearchEvent(event map[string]interface{}) bool
	AddSearch(search string, uuid string)
	RemoveSearch(search string)
}
