package tracking

import (
	"fmt"
	"sync"
)

type EmbeddedMetadata struct {
	Identifier  string
	Hex         string
	EventID     string
	UserContext UserContext
}

func (m EmbeddedMetadata) GetUserContext() UserContext {
	return m.UserContext
}

type EmbeddedTracker struct {
	subscriptions map[string]EmbeddedMetadata
	mutex         sync.RWMutex
}

func NewEmbeddedTracker() *EmbeddedTracker {
	return &EmbeddedTracker{
		subscriptions: make(map[string]EmbeddedMetadata),
	}
}

func (t *EmbeddedTracker) Track(subscriptionID string, metadata EmbeddedMetadata) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscriptions[subscriptionID] = metadata
	return nil
}

func (t *EmbeddedTracker) Lookup(event []interface{}) (EmbeddedMetadata, error) {
	subscriptionId, ok := event[1].(string)
	if !ok {
		fmt.Println("Could not extract the subscription id from the event in embedded lookup")
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	embeddedMetadata := t.subscriptions[subscriptionId]

	return embeddedMetadata, nil
}
