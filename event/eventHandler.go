package event

import (
	"fmt"

	"github.com/DerekBelloni/go-socket-server/core"
	"github.com/DerekBelloni/go-socket-server/data"
	"github.com/DerekBelloni/go-socket-server/queue"
)

var callbackRelays = []string{
	"wss://relay.damus.io",
	"wss://relay.primal.net",
	"wss://relay.nostr.band",
}

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

// this is not being used
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

func ExtractAuthorsPubkey(content map[string]interface{}) string {
	pubkey, ok := content["pubkey"].(string)

	if !ok {
		return ""
	}

	return pubkey
}

func PackageEvent(eventData []interface{}, userPubkey string, eventType string, uuid string) data.EventMessage {
	message := data.EventMessage{
		UserPubkey: &userPubkey,
		UUID:       &uuid,
	}
	fmt.Printf("uuid in package event: %v\n", uuid)
	if userPubkey != "" {
		if eventType == "follows" {
			message.Event = data.FollowsEvent{
				Data: eventData,
			}
		}
	}

	return message
}

func HandleEvent(eventData []interface{}, eventChan chan string, connector core.RelayConnector, relayUrl string, subscriptionTracker core.SubscriptionTracker) {
	content, ok := eventData[2].(map[string]interface{})
	if !ok {
		fmt.Println("Could not extract content from event data")
		return
	}

	kind, ok := content["kind"].(float64)
	if !ok {
		fmt.Println("Could not extract kind from content")
	}

	switch kind {
	case 0:
		searchKey, searchKeyExists := subscriptionTracker.InSearchEvent(eventData, "0")
		// need to parse which is a follows metadata, use the search tracker
		if !searchKeyExists {
			queue.MetadataQueue(eventData, eventChan)
		} else {
			queue.AuthorMetadataQueue(eventData, searchKey)
		}
	case 1:
		searchKey, searchKeyExists := subscriptionTracker.InSearchEvent(eventData, "1")
		// this is only for follower notes at the moment
		subscriptionPubkey, subscriptionExists := subscriptionTracker.InSubscriptionMapping(eventData)

		if !searchKeyExists && !subscriptionExists {
			queue.NotesQueue(eventData, eventChan, "")
		} else if !searchKeyExists && subscriptionExists {
			eventMessage := PackageEvent(eventData, subscriptionPubkey, "follows", "")
			queue.NewNotesQueue(eventMessage, eventChan)
		} else {
			authorPubkey := ExtractAuthorsPubkey(content)
			for _, relayUrl := range callbackRelays {
				go func(relayUrl string) {
					connector.GetSearchedAuthorMetadata(relayUrl, authorPubkey, searchKey, subscriptionTracker)
				}(relayUrl)
			}
			queue.SearchQueue(eventData, searchKey, eventChan)
		}
	case 3:
		queue.FollowListQueue(eventData, eventChan)
	}
}

func HandleNotice(noticeData []interface{}, relayURL string) {
	fmt.Printf("Notice received: %v, relay url: %v\n", noticeData, relayURL)
}

func HandleEOSE(eoseData []interface{}, relayUrl string, eventChan chan<- string) {
	fmt.Printf("EOSE received: %v, %v\n\n", eoseData, relayUrl)
}
