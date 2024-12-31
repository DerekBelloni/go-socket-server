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

	var searchKeyExists bool

	st.searchTrackerUUIDLOCK.Lock()
	searchKey := st.searchTrackerUUID[subscriptionID]
	st.searchTrackerUUIDLOCK.Unlock()

	if searchKey != "" {
		searchKeyExists = true
	} else {
		searchKeyExists = false
	}

	return searchKey, searchKeyExists
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

func (st *SearchTrackerImpl) AddSubscription(subscriptionID string, userPubkey string, followsPubkey string, uuid string) {

	if userPubkey == "" {
		st.searchTrackerUUIDLOCK.Lock()
		st.subscriptionTracker[subscriptionID] = uuid
		st.searchTrackerUUIDLOCK.Unlock()
	} else {
		st.searchTrackerUUIDLOCK.Lock()
		st.subscriptionTracker[subscriptionID] = userPubkey
		st.searchTrackerUUIDLOCK.Unlock()
	}
}

func (st *SearchTrackerImpl) AddSearch(search string, uuid string, subscriptionId string, pubkey *string) {
	st.searchTrackerUUIDLOCK.Lock()
	defer st.searchTrackerUUIDLOCK.Unlock()

	if pubkey == nil {
		fmt.Printf("UUID IN ADD SEARCH!!!: %v\n", uuid)
		st.searchTrackerUUID[subscriptionId] = uuid
	} else if *pubkey != "" {
		fmt.Printf("In add search, uuid: %v\n\n pubkey: %v\n\n", uuid, pubkey)
		st.searchTrackerUUID[subscriptionId] = *pubkey
	}
}

func (st *SearchTrackerImpl) RemoveSearch(search string) {
	//
}
