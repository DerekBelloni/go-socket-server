package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	fmt.Printf("Random bytesL %v\n", bytes)

	return hex.EncodeToString(bytes), nil
}

func main() {
	relayUrl := "wss://relay.damus.io"

	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	if err != nil {
		log.Fatal("Dial error: ", err)
	}
	defer conn.Close()

	// Generate a unique subscription ID
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
	}

	// Set the request
	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{},
	}

	// Marshal the request to JSON
	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		log.Fatal("Error marshalling subscription request: ", err)
	}

	// Send the subscription request to the relay
	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	if err != nil {
		log.Fatal("Error sending subscription request: ", err)
	}
	fmt.Println("Subscription request sent")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error: ", err)
			break
		}

		log.Printf("Received message: %s", message)
	}
}
