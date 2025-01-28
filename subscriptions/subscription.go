package subscriptions

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DerekBelloni/go-socket-server/core"
	"github.com/DerekBelloni/go-socket-server/data"
)

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func MetadataSubscription(relayUrl string, userHexKey string, writeChan chan<- []byte, eventChan <-chan string) {
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		fmt.Printf("Error generating a subscription id: %v\n", err)
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds":   []int{0},
			"authors": []string{userHexKey},
		},
	}
	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		fmt.Printf("Error marshalling subscription request: %v\n ", err)
	}
	writeChan <- subscriptionRequestJSON

	closeSubscription(subscriptionID, writeChan)
}

func UserNotesSubscription(relayUrl string, userHexKey string, writeChan chan<- []byte, eventChan <-chan string) {
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		fmt.Printf("Error generating a subscription id: %v\n", err)
	}
	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds":   []int{1},
			"authors": []string{userHexKey},
			"limit":   100,
		},
	}
	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		fmt.Printf("Error marshalling subscription request: %v\n ", err)
	}

	writeChan <- subscriptionRequestJSON

	closeSubscription(subscriptionID, writeChan)
}

func FollowListSubscription(relayUrl string, userHexKey string, writeChan chan<- []byte, eventChan <-chan string) {
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		fmt.Printf("Error generating a subscription id: %v\n", err)
	}
	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds":   []int{3},
			"authors": []string{userHexKey},
			"limit":   1,
		},
	}
	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		fmt.Printf("Error marshalling subscription request: %v\n ", err)
	}

	writeChan <- subscriptionRequestJSON
	closeSubscription(subscriptionID, writeChan)
}

func FollowListMetadataSubscription(relayUrl string, pubKeys []string, userHexKey string, writeChan chan<- []byte, eventChan <-chan string, subscriptionTracker core.SubscriptionTracker) {
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		fmt.Printf("Error generating a subscription id: %v\n", err)
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds":   []int{0},
			"authors": pubKeys,
		},
	}

	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		fmt.Printf("Error marshalling subscription request: %v\n ", err)
	}
	subscriptionType := "followsMetadata"
	subscriptionTracker.FollowsMetadataSubscription(subscriptionID, "test", userHexKey, subscriptionType)
	writeChan <- subscriptionRequestJSON

	closeSubscription(subscriptionID, writeChan)
}

func FollowsNotesSubscription(relayUrl string, userPubkey string, followsPubkey string, subscriptionTracker core.SubscriptionTracker, writeChan chan<- []byte, eventChan <-chan string, uuid string) {
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		fmt.Printf("Error generating a subscription id: %v\n", err)
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds":   []int{1},
			"authors": []string{followsPubkey},
			"limit":   25,
		},
	}

	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		fmt.Printf("Error marshalling subscription request: %v\n", err)
	}

	subscriptionTracker.AddSubscription(subscriptionID, userPubkey, followsPubkey, uuid)
	writeChan <- subscriptionRequestJSON

	closeSubscription(subscriptionID, writeChan)
}

func CreateNoteEvent(relayUrl string, newNote data.NewNote, writeChan chan<- []byte, eventChan <-chan string) {
	event := data.NostrEvent{
		PubKey:    newNote.PubHexKey,
		CreatedAt: time.Now().Unix(),
		Kind:      newNote.Kind,
		Tags:      [][]string{},
		Content:   newNote.Content,
	}

	if err := event.GenerateId(); err != nil {
		fmt.Printf("Error generating an event id: %v\n", err)
	}

	if err := event.SignEvent(newNote.PrivHexKey); err != nil {
		fmt.Printf("Error signing the event: %v\n", err)
	}

	eventMessage := []interface{}{
		"EVENT",
		event,
	}

	jsonBytes, err := json.Marshal(eventMessage)
	if err != nil {
		fmt.Printf("Error marshalling event: %v\n ", err)
	}
	writeChan <- jsonBytes
}

// need to close these subscriptions for sure
func RetrieveSearchSubscription(relayUrl string, search string, writeChan chan<- []byte, eventChan <-chan string, subscriptionTracker core.SubscriptionTracker, uuid string, pubkey string) {
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		fmt.Printf("Error generating a subscription id: %v\n", err)
	}
	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kind":   1,
			"limit":  10,
			"search": search,
		},
	}
	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		fmt.Printf("Error marshalling subscription request: %v\n ", err)
	}

	subscriptionTracker.AddSearch(search, uuid, subscriptionID, pubkey)
	writeChan <- subscriptionRequestJSON

	closeSubscription(subscriptionID, writeChan)
}

func SearchedAuthorMetadata(relayUrl string, authorPubkey string, searchKey string, subscriptionTracker core.SubscriptionTracker, writeChan chan<- []byte, eventChan <-chan string) {
	var uuid string
	var userPubkey string
	if _, err := hex.DecodeString(searchKey); err == nil {
		userPubkey = searchKey
	} else {
		uuid = searchKey
	}

	subscriptionID, err := generateRandomString(16)

	if err != nil {
		fmt.Printf("Error generating a subscription id: %v\n", err)
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds":   []int{0},
			"authors": []string{authorPubkey},
		},
	}

	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		fmt.Printf("Error marshalling subscription request: %v\n", err)
	}

	subscriptionTracker.AddSearch("author", uuid, subscriptionID, userPubkey)
	writeChan <- subscriptionRequestJSON
	closeSubscription(subscriptionID, writeChan)
	// time.Sleep(5 * time.Second)

	// closeMessage := []interface{}{
	// 	"CLOSE",
	// 	subscriptionID,
	// }
	// closeMessageJSON, err := json.Marshal(closeMessage)
	// if err != nil {
	// 	fmt.Printf("Error marshalling close message: %v\n", err)
	// 	return
	// }

	// writeChan <- closeMessageJSON
}

func closeSubscription(subscriptionID string, writeChan chan<- []byte) {
	time.Sleep(5 * time.Second)

	closeMessage := []interface{}{
		"CLOSE",
		subscriptionID,
	}

	closeMessageJSON, err := json.Marshal(closeMessage)
	if err != nil {
		fmt.Printf("Error marshalling close message: %v\n", err)
		return
	}

	writeChan <- closeMessageJSON
}
