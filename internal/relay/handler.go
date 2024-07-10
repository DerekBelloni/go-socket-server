package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/DerekBelloni/go-socket-server/internal/redis"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var (
	damusRelay  []string
	purpleRelay []string
	nosRelay    []string
	primalRelay []string
)

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// I think I can pass in the filters I want depending on where it is being called from (reactions vs default events, etc)
func createSubscriptionRequest() []interface{} {
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
			"limit": 100,
		},
	}
	return subscriptionRequest
}

// I can probably hand in a connection type here as well
func handleRelayConnection(conn *websocket.Conn, relayUrl string, finished chan<- string) {
	defer conn.Close()

	subscriptionRequest := createSubscriptionRequest()

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
					"relay": relayUrl,
				}).Error("WebSocket Error")
				break
			}
			log.WithFields(logrus.Fields{
				"error": err,
				"relay": relayUrl,
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

		var reactionEvents []interface{}
		if len(rMessage) >= 3 {
			eventMap, ok := rMessage[2].(map[string]interface{})
			if !ok {
				continue
			}

			kind, ok := eventMap["kind"].(float64)
			if !ok || int(kind) != 7 {
				continue
			}

			content, ok := eventMap["content"].(string)
			if !ok || string(content) == "-" {
				continue
			}

			reactionEvents = append(reactionEvents, eventMap["tags"])

			if firstElement, ok := rMessage[0].(string); ok && firstElement == "EOSE" {
				batchJSON, err := json.Marshal(batch)
				if err != nil {
					log.Fatal("Error marshalling the batched relay data: ", err)
				}
				redis.HandleRedis(batchJSON, relayUrl, finished, "")
				break
			}
		}

		batch = append(batch, message)
		shouldContinue := parseReactionEvents(reactionEvents, relayUrl, finished)
		if !shouldContinue {
			log.WithFields(logrus.Fields{
				"relay": relayUrl,
			}).Info("Collected enough events, stopping processing")
			break
		}
	}
	retrieveReactionEvents(relayUrl, finished)
}

func extractETag(tag []interface{}) (bool, string) {
	if len(tag) < 2 {
		return false, ""
	}

	key, ok := tag[0].(string)
	if !ok || key != "e" {
		return false, ""
	}

	eventID, ok := tag[1].(string)
	if !ok {
		return false, ""
	}

	return true, eventID
}

func processRelay(relaySlice *[]string, relay string, finished chan<- string, eventID string) bool {
	if len(*relaySlice) < 25 {
		*relaySlice = append(*relaySlice, eventID)
		return true
	}
	// if len(*relaySlice) == 25 {
	// 	finished <- relay
	// }
	return false
}

func processEvent(eventArray []interface{}, relayUrl string, finished chan<- string) bool {
	for _, tag := range eventArray {
		tagArray, ok := tag.([]interface{})
		if !ok {
			fmt.Println("Skipping non-array tag:", tag)
			continue
		}
		if isE, eventID := extractETag(tagArray); isE {
			eventID = strings.TrimSpace(eventID)
			if eventID == "" {
				continue
			}
			switch relay := relayUrl; relay {
			case "wss://relay.damus.io":
				return processRelay(&damusRelay, relay, finished, eventID)
			case "wss://nos.lol":
				return processRelay(&nosRelay, relay, finished, eventID)
			case "wss://purplerelay.com":
				return processRelay(&purpleRelay, relay, finished, eventID)
			case "wss://relay.primal.net":
				return processRelay(&primalRelay, relay, finished, eventID)
			}
		}
	}
	return true
}

func parseReactionEvents(reactionEvents []interface{}, relayUrl string, finished chan<- string) bool {
	for _, event := range reactionEvents {
		if eventArray, ok := event.([]interface{}); ok {
			shouldContinue := processEvent(eventArray, relayUrl, finished)
			if !shouldContinue {
				return false
			}
		}
	}
	return true
}

func retrieveReactionEvents(relayUrl string, finished chan<- string) {
	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	if err != nil {
		log.Fatal("Dial error: ", err)
	}

	defer conn.Close()

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
	}

	var relaySlice *[]string

	switch relayUrl {
	case "wss://relay.damus.io":
		relaySlice = &damusRelay
	case "wss://nos.lol":
		relaySlice = &nosRelay
	case "wss://purplerelay.com":
		relaySlice = &purpleRelay
	case "wss://relay.primal.net":
		relaySlice = &primalRelay
	default:
		log.Printf("Uknown relay URL: %s", relayUrl)
		return
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"ids": relaySlice,
		},
	}

	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		log.Fatal("Error sending subscription request: ", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	if err != nil {
		log.Fatal("Error sending subscription")
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
				}).Error("WebSocket read error")
			}
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

		if len(rMessage) > 3 {
			firstElement, ok := rMessage[0].(string)
			if !ok || firstElement != "EOSE" {
				continue
			}
			batchJson, err := json.Marshal(batch)
			if err != nil {
				fmt.Println("Error marshalling the batched relay data for trending events: ", err)
			}
			redis.HandleRedis(batchJson, relayUrl, finished, "trending")
		}
		batch = append(batch, message)
	}
}
