package search

import (
	"fmt"
	"sync"
)

type SubscriptionTracker interface {
	AddSubscription()
	LookupSubscription()
	RemoveSubscription()
}

// specific

type SubscriptionInfo struct {
	UUID               string
	UserPubkey         string
	FollowsPubkey      string
	Type               string
	EmbeddedIdentifier string
	EventID            string
}

type SearchTrackerImpl struct {
	searchTrackerUUID      map[string]string
	newSubscriptionTracker map[string]SubscriptionInfo
	subscriptionTracker    map[string]string
	searchTrackerUUIDLOCK  sync.RWMutex
}

func NewSearchTrackerImpl() *SearchTrackerImpl {
	return &SearchTrackerImpl{
		searchTrackerUUID:      make(map[string]string),
		subscriptionTracker:    make(map[string]string),
		newSubscriptionTracker: make(map[string]SubscriptionInfo),
	}
}

func (st *SearchTrackerImpl) InSearchEvent(event []interface{}, eventKind string) (string, bool) {
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

func (st *SearchTrackerImpl) InFollowsMetadtaMapping(event []interface{}) (string, string, string, bool, string, string, string) {
	subscriptionID, ok := event[1].(string)
	if !ok {
		fmt.Println("Could not extract subscription ID from event")
	}

	var followsMetadataExists bool
	st.searchTrackerUUIDLOCK.Lock()
	defer st.searchTrackerUUIDLOCK.Unlock()

	subscriptionInfo := st.newSubscriptionTracker[subscriptionID]
	userPubkey := subscriptionInfo.UserPubkey
	followsPubkey := subscriptionInfo.FollowsPubkey
	susbscriptionType := subscriptionInfo.Type
	identifier := subscriptionInfo.EmbeddedIdentifier
	uuid := subscriptionInfo.UUID
	eventId := subscriptionInfo.EventID

	if followsPubkey != "" && userPubkey != "" && susbscriptionType == "followsMetadata" && identifier != "none" {
		followsMetadataExists = true
	} else {
		followsMetadataExists = false
	}
	return userPubkey, followsPubkey, susbscriptionType, followsMetadataExists, identifier, uuid, eventId

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

func (st *SearchTrackerImpl) FollowsMetadataSubscription(subscriptionID string, followsPubkey string, userPubkey string, subscriptionType string, uuid string, identifier string, eventId string) {
	if eventId == "" {
		eventId = "none"
	}

	if followsPubkey == "" {
		followsPubkey = "none"
	}

	if identifier == "" {
		identifier = "none"
	}

	if userPubkey == "" {
		userPubkey = "none"
	}

	if uuid == "" {
		uuid = "none"
	}

	subscriptionInfoStruct := SubscriptionInfo{
		UUID:               uuid,
		UserPubkey:         userPubkey,
		FollowsPubkey:      followsPubkey,
		Type:               subscriptionType,
		EmbeddedIdentifier: identifier,
		EventID:            eventId,
	}

	if subscriptionID != "" {
		st.searchTrackerUUIDLOCK.Lock()
		st.newSubscriptionTracker[subscriptionID] = subscriptionInfoStruct
		st.searchTrackerUUIDLOCK.Unlock()
	}
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

func (st *SearchTrackerImpl) AddSearch(search string, uuid string, subscriptionId string, pubkey string) {
	st.searchTrackerUUIDLOCK.Lock()
	defer st.searchTrackerUUIDLOCK.Unlock()

	if pubkey == "" {
		st.searchTrackerUUID[subscriptionId] = uuid
	} else {
		st.searchTrackerUUID[subscriptionId] = pubkey
	}
}

func (st *SearchTrackerImpl) RemoveSearch(search string) {
	//
}
