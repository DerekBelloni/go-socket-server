package tracking

import (
	"fmt"
	"sync"
)

type EmbeddedMetadata struct {
	Identifier  string
	Hex         string
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
	// need to check it first for relay urls as well
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscriptions[subscriptionID] = metadata
	fmt.Printf("Embedded tracker mapping: %v\n", t.subscriptions[subscriptionID])
	return nil
}

func (t *EmbeddedTracker) Lookup(event []interface{}) (EmbeddedMetadata, error) {
	return EmbeddedMetadata{}, nil
}
