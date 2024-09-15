package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		fmt.Errorf("Error marshalling subscription request: %v\n ", err)
	}
	writeChan <- subscriptionRequestJSON
	test := <-eventChan
	if test != "" {
		metadataSet <- relayUrl
	}
}

func UserNotesSubscription(relayUrl string, userHexKey string, writeChan chan<- []byte, eventChan <-chan string, notesFinished chan<- string) {
	fmt.Println("subscription notes")
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
