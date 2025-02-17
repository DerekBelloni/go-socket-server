package tracking

import "sync"

type FollowsMetadata struct {
	FollowsPubkey string
	UserContext   UserContext
}

func (f FollowsMetadata) GetUserContext() UserContext {
	return f.UserContext
}

type FollowsTracker struct {
	subscriptions map[string]FollowsMetadata
	mutex         sync.RWMutex
}

func NewFollowsTracker() *FollowsTracker {
	return &FollowsTracker{
		subscriptions: make(map[string]FollowsMetadata),
	}
}

func (f *FollowsTracker) Track(subscriptionID string, metadata FollowsMetadata) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.subscriptions[subscriptionID] = metadata
	return nil
}

func (f *FollowsMetadata) Lookup(event []interface{}) (FollowsMetadata, error) {
	return FollowsMetadata{}, nil
}
