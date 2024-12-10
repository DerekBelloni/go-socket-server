package search

import (
	"fmt"
	"sync"
)

type SearchTrackerImpl struct {
	searchTrackerUUID     map[string]string
	searchTrackerUUIDLOCK sync.RWMutex
}

func NewSearchTrackerImpl() *SearchTrackerImpl {
	return &SearchTrackerImpl{
		searchTrackerUUID: make(map[string]string),
	}
}

func (st *SearchTrackerImpl) InSearchEvent(event map[string]interface{}) bool {
	return false
}

func (st *SearchTrackerImpl) AddSearch(search string, uuid string, subscriptionId string) {
	// st.searchTrackerUUID[]
	fmt.Printf("subscriptionId: %v\n", subscriptionId)
}

func (st *SearchTrackerImpl) RemoveSearch(search string) {

}
