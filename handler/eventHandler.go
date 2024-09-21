package handler

import (
	"fmt"
)

func extractPubKey(tag interface{}) (string, bool) {

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

func prepareFollowsPubKeys(content map[string]interface{}, eventChan chan<- string) []strings {
	tags, ok := content["tags"].([]interface{})
	if !ok {
		return
	}

	var pubKeys []string
	for _, tag := range tags {
		pubKey, ok := extractPubKey(tag)
		if !ok {
			continue
		}
		pubKeys = append(pubKeys, pubKey)
	}
	return pubKeys
}

func HandleEvent(eventData []interface{}, eventChan chan string) {
	if content, ok := eventData[2].(map[string]interface{}); ok {
		kind, _ := content["kind"].(float64)
		switch kind {
		case 0:
			metadataQueue(eventData, eventChan)
		case 1:
			notesQueue(eventData, eventChan)
		case 3:
			// pull the ids out of the follows, go get metadata for each one
			followListQueue(eventData, eventChan)
			// pubKeys := prepareFollowsPubKeys(content, eventChan)
			// handler.Su
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
