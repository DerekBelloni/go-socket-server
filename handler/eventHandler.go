package handler

import (
	"fmt"
)

func HandleEvent(eventData []interface{}, eventChan chan string) {
	if content, ok := eventData[2].(map[string]interface{}); ok {
		kind, _ := content["kind"].(float64)
		switch kind {
		case 0:
			metadataQueue(eventData, eventChan)
		case 1:
			notesQueue(eventData, eventChan)
		case 3:
			followListQueue(eventData, eventChan)
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
