package core

type RelayConnector interface {
	GetConnection(relayUrl string) (chan []byte, chan string, error)
	GetUserMetadata(relayUrl string, userHexKey string, metadataFinished chan<- string)
	GetUserNotes(relayUrl string, userHexKey string, notesFinished chan<- string)
	GetFollowList(relayUrl string, userHexKey string, followsFinished chan<- string)
	GetFollowListMetadata(relayUrl string, userHexKey string)
}
