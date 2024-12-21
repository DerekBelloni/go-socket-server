package search

import (
	"fmt"
	"sync"
)

type SearchTrackerImpl struct {
	searchTrackerUUID     map[string]string
	subscriptionTracker   map[string]string
	searchTrackerUUIDLOCK sync.RWMutex
}

func NewSearchTrackerImpl() *SearchTrackerImpl {
	return &SearchTrackerImpl{
		searchTrackerUUID:   make(map[string]string),
		subscriptionTracker: make(map[string]string),
	}
}

func (st *SearchTrackerImpl) InSearchEvent(event []interface{}) (string, bool) {
	subscriptionID, ok := event[1].(string)
	if !ok {
		fmt.Println("Could not extract subscription ID from event")
	}

	var pubKeyExists bool

	st.searchTrackerUUIDLOCK.Lock()
	subscriptionPubkey := st.searchTrackerUUID[subscriptionID]
	st.searchTrackerUUIDLOCK.Unlock()

	if subscriptionPubkey != "" {
		pubKeyExists = true
	} else {
		pubKeyExists = false
	}

	return subscriptionPubkey, pubKeyExists
}

func (st *SearchTrackerImpl) InSubscriptionMapping(event []interface{}) (string, bool) {
	subscriptionID, ok := event[1].(string)
	if !ok {
		fmt.Println("Could not extract subscription ID from event")
	}

	var pubKeyExists bool

	st.searchTrackerUUIDLOCK.Lock()
	subscriptionPubkey := st.subscriptionTracker[subscriptionID]
	st.searchTrackerUUIDLOCK.Unlock()

	if subscriptionPubkey != "" {
		pubKeyExists = true
	} else {
		pubKeyExists = false
	}

	return subscriptionPubkey, pubKeyExists
}

func (st *SearchTrackerImpl) AddSubscription(subscriptionID string, userPubkey string, followsPubkey string) {
	st.searchTrackerUUIDLOCK.Lock()
	st.subscriptionTracker[subscriptionID] = userPubkey
	st.searchTrackerUUIDLOCK.Unlock()
}

func (st *SearchTrackerImpl) AddSearch(search string, uuid string, subscriptionId string, pubkey *string) {
	if pubkey == nil {
		st.searchTrackerUUIDLOCK.Lock()
		st.searchTrackerUUID[subscriptionId] = uuid
		st.searchTrackerUUIDLOCK.Unlock()
	} else {
		st.searchTrackerUUIDLOCK.Lock()
		st.searchTrackerUUID[subscriptionId] = *pubkey
		st.searchTrackerUUIDLOCK.Unlock()
	}
}

func (st *SearchTrackerImpl) RemoveSearch(search string) {
	//
}
