package handler

import (
	"fmt"

	"github.com/DerekBelloni/go-socket-server/internal/core"
)

func ExtractPubKey(tag interface{}) (string, bool) {
	tagSlice, ok := tag.([]interface{})
	if !ok || len(tagSlice) < 2 {
		return "", false
	}

	pubKey, ok := tagSlice[1].(string)
	if !ok {
		return "", false
	}

	return pubKey, true
}

func PrepareFollowsPubKeys(content map[string]interface{}, eventChan chan<- string) []string {
	tags, ok := content["tags"].([]interface{})
	if !ok {
		return nil
	}

	var pubKeys []string
	for _, tag := range tags {
		pubKey, ok := ExtractPubKey(tag)
		if !ok {
			continue
		}
		pubKeys = append(pubKeys, pubKey)
	}
	return pubKeys
}

func HandleEvent(eventData []interface{}, eventChan chan string, connector core.RelayConnector, relayUrl string) {
	if content, ok := eventData[2].(map[string]interface{}); ok {
		kind, _ := content["kind"].(float64)
		switch kind {
		case 0:
			fmt.Printf("case 0 event data: %v\n", eventData)
			metadataQueue(eventData, eventChan)
		case 1:
			notesQueue(eventData, eventChan)
		case 3:
			// pull the ids out of the follows, go get metadata for each one
			// followListQueue(eventData, eventChan)
			pubKeys := PrepareFollowsPubKeys(content, eventChan)
			connector.GetFollowListMetadata(relayUrl, pubKeys)
		}
	}

}

func HandleNotice(noticeData []interface{}) {
	fmt.Printf("Notice received: %v\n", noticeData)
}

func HandleEOSE(eoseData []interface{}, relayUrl string, eventChan chan<- string) {
	fmt.Printf("EOSE received: %v, %v\n\n", eoseData, relayUrl)
	eventChan <- relayUrl
}
