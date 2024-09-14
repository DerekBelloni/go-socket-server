package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func MetadataSubscription(conn *websocket.Conn, relayUrl string, userHexKey string, writeChan chan<- []byte) {
	fmt.Println("hello there")
	subscriptionID, err := generateRandomString(16)
	if err != nil {
		fmt.Errorf("Error generating a subscription id: %v\n", err)
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
}

func UserNotesSubscription(conn *websocket.Conn, relayUrl string, userHexKey string, writeChan chan<- []byte) {

}
