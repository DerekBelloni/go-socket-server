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

func MetadataSubscription(relayUrl string, userHexKey string, writeChan chan<- []byte, eventChan <-chan string, metadataSet chan<- string) {
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		fmt.Printf("Error generating a subscription id: %v\n", err)
	}
	fmt.Printf("metadata subscription pubkey: %v\n\n", userHexKey)
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
	// test := <-eventChan
	// if test != "" {
	// 	metadataSet <- relayUrl
	// }

	// Wait for the metadata event or a timeout
	select {
	case test := <-eventChan:
		if test != "" {
			// Notify that metadata has been received
			metadataSet <- relayUrl
		}
	case <-time.After(10 * time.Second): // Timeout after 10 seconds
		fmt.Printf("Timeout waiting for metadata from relay: %s\n", relayUrl)
	}

	// Send the CLOSE message to terminate the subscription
	closeMessage := []interface{}{
		"CLOSE",
		subscriptionID,
	}
	closeMessageJSON, err := json.Marshal(closeMessage)
	if err != nil {
		fmt.Printf("Error marshalling close message: %v\n", err)
		return
	}

	// Send the CLOSE message to the relay
	writeChan <- closeMessageJSON
	fmt.Printf("Sent CLOSE message for subscription ID: %s\n", subscriptionID)
}
func UserNotesSubscription(relayUrl string, userHexKey string, writeChan chan<- []byte, eventChan <-chan string, notesFinished chan<- string) {
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
	notes := <-eventChan
	if notes != "" {
		notesFinished <- relayUrl
	}
}
func FollowListSubscription(relayUrl string, userHexKey string, writeChan chan<- []byte, eventChan <-chan string, followsFinished chan<- string) {
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
	follows := <-eventChan

	if follows != "" {
		followsFinished <- relayUrl
	}
}

func FollowListMetadataSubscription(relayUrl string, pubKeys []string, writeChan chan<- []byte, eventChan <-chan string) {
	for _, pubKey := range pubKeys {
		go func(pubKey string) {
			subscriptionID, err := generateRandomString(16)
			if err != nil {
				fmt.Printf("Error generating a subscription id: %v\n", err)
			}

			subscriptionRequest := []interface{}{
				"REQ",
				subscriptionID,
				map[string]interface{}{
					"kinds":   []int{0},
					"authors": []string{pubKey},
				},
			}

			subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
			if err != nil {
				fmt.Printf("Error marshalling subscription request: %v\n ", err)
			}
			writeChan <- subscriptionRequestJSON
		}(pubKey)
	}
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

func RetrieveSearchSubscription(relayUrl string, search string, writeChan chan<- []byte, eventChan <-chan string, subscriptionTracker core.SubscriptionTracker, uuid string, pubkey string) {
	go func() {
		subscriptionID, err := generateRandomString(16)
		if err != nil {
			fmt.Printf("Error generating a subscription id: %v\n", err)
		}
		subscriptionRequest := []interface{}{
			"REQ",
			subscriptionID,
			map[string]interface{}{
				"kind":   1,
				"limit":  30,
				"search": search,
			},
		}
		subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
		if err != nil {
			fmt.Printf("Error marshalling subscription request: %v\n ", err)
		}

		subscriptionTracker.AddSearch(search, uuid, subscriptionID, pubkey)
		writeChan <- subscriptionRequestJSON
	}()
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
}
