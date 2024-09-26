package core

import "context"

type RelayConnector interface {
	GetConnection(relayUrl string) (chan []byte, chan string, error)
	GetUserMetadata(ctx context.Context, string, userHexKey string, metadataFinished chan<- string)
	GetUserNotes(ctx context.Context, relayUrl string, userHexKey string, notesFinished chan<- string)
	GetFollowList(ctx context.Context, relayUrl string, userHexKey string, followsFinished chan<- string)
	GetFollowListMetadata(ctx context.Context, relayUrl string, pubKeys []string)
}
