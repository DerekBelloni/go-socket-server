package tracking

import (
	"fmt"
	"sync"
)

type NPubMetadata struct {
	Identifier  string
	Hex         string
	UserContext UserContext
}

func (np NPubMetadata) GetUserContext() UserContext {
	return np.UserContext
}

type NPubMetadataTracker struct {
	subscriptions map[string]NPubMetadata
	mutex         sync.RWMutex
}

func NewNPubMetadataTracker() *NPubMetadataTracker {
	return &NPubMetadataTracker{
		subscriptions: make(map[string]NPubMetadata),
	}
}

func (np *NPubMetadataTracker) Track(subscriptionID string, metadata NPubMetadata) error {
	np.mutex.Lock()
	defer np.mutex.Unlock()
	np.subscriptions[subscriptionID] = metadata
	return nil
}

func (np *NPubMetadataTracker) Lookup(event []interface{}) (NPubMetadata, error) {
	subscriptionId, ok := event[1].(string)
	if !ok {
		fmt.Println("Could not extract the subscription id from the event in embedded lookup")
	}
	np.mutex.Lock()
	defer np.mutex.Unlock()

	npubMetadata := np.subscriptions[subscriptionId]

	return npubMetadata, nil
}
