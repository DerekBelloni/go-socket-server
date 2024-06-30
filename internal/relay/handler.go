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
			"kinds": []int{0, 1},
			"since": start.Add(-1 * time.Hour).Unix(),
			"until": time.Now().Unix(),
			"limit": 500,
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

		if len(rMessage) > 0 {
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
}
