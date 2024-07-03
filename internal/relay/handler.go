package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/DerekBelloni/go-socket-server/internal/redis"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func handleRelayConnection(conn *websocket.Conn, relayUrl string, finished chan<- string) {
	defer conn.Close()

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
	}

	start := time.Now()

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds": []int{0, 1, 7},
			"since": start.Add(-24 * time.Hour).Unix(),
			"until": time.Now().Unix(),
			"limit": 2000,
		},
	}

	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		log.Fatal("Error marshalling subscription request: ", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	if err != nil {
		log.Fatal("Error sending subscription request: ", err)
	}

	fmt.Println("Subscription request sent")

	var batch []json.RawMessage

	var log = logrus.New()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WithFields(logrus.Fields{
					"error": err,
				}).Error("WebSocket Error")
				break
			}
			log.WithFields(logrus.Fields{
				"error": err,
			}).Error("WebSocket read error")
			break
		}

		if len(message) == 0 {
			log.Warn("Received an empty message")
			continue
		}

		var rMessage []interface{}
		if err := json.Unmarshal(message, &rMessage); err != nil {
			log.WithFields(logrus.Fields{
				"error":   err,
				"message": string(message),
			}).Error("Error unmarshalling JSON")
			continue
		}

		if len(rMessage) >= 3 {
			fmt.Println("Dumping rMessage structure:")

			if eventMap, ok := rMessage[2].(map[string]interface{}); ok {
				if kind, ok := eventMap["kind"].(float64); ok && int(kind) == 7 {
					if content, ok := eventMap["content"].(string); ok && string(content) != "-" {
						// spew.Dump(rMessage[2])
						fmt.Println(rMessage[2])
					}
				}
			} else {
				fmt.Println("rMessage[2] is not a map or is missing")
			}

			if firstElement, ok := rMessage[0].(string); ok && firstElement == "EOSE" {
				batchJSON, err := json.Marshal(batch)
				if err != nil {
					log.Fatal("Error marshalling the batched relay data: ", err)
				}

				redis.HandleRedis(batchJSON, relayUrl, finished)
				break
			}
		}

		batch = append(batch, message)
	}

	// func retrieveReactionEvents(batch ) {

	// }
}
