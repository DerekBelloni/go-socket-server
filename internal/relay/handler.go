package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const maxTrendingIds = 25

var (
	mutex       sync.Mutex
	tempIds     map[string]string
	trendingIds map[string]string
	damusRelay  []string
	purpleRelay []string
	nosRelay    []string
	primalRelay []string
)

func init() {
	tempIds = make(map[string]string)
	trendingIds = make(map[string]string)
}

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
			// fmt.Println("Dumping rMessage structure:")

			if eventMap, ok := rMessage[2].(map[string]interface{}); ok {
				if kind, ok := eventMap["kind"].(float64); ok && int(kind) == 7 {
					if content, ok := eventMap["content"].(string); ok && string(content) != "-" {
						// spew.Dump(eventMap["tags"])
						// fmt.Println(eventMap["tags"])
						reactionEvents = append(reactionEvents, eventMap["tags"])
					}
				}
			} else {
				fmt.Println("rMessage[2] is not a map or is missing")
			}

			if firstElement, ok := rMessage[0].(string); ok && firstElement == "EOSE" {
				// batchJSON, err := json.Marshal(batch)
				if err != nil {
					log.Fatal("Error marshalling the batched relay data: ", err)
				}

				// redis.HandleRedis(batchJSON, relayUrl, finished)
				break
			}
		}

		batch = append(batch, message)
		retrieveReactionEvents(reactionEvents, relayUrl, finished)
	}
}

func extractETag(tag []interface{}) (bool, string) {
	fmt.Println("Checking tag:", tag) // Debug output

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

func processEvent(eventArray []interface{}, relayUrl string) {
	mutex.Lock()
	defer mutex.Unlock()
	for _, tag := range eventArray {
		tagArray, ok := tag.([]interface{})
		if !ok {
			fmt.Println("Skipping non-array tag:", tag) // Debug output
			continue
		}
		if isE, eventID := extractETag(tagArray); isE {
			eventID = strings.TrimSpace(eventID)
			if eventID != "" {
				// fmt.Println("Valid eventID found:", eventID, "from relay:", relayUrl) // Debug output
				tempIds[eventID] = relayUrl
			} else {
				fmt.Println("Empty or whitespace eventID ignored, original eventID:", eventID) // Debug output
			}
		} else {
			fmt.Println("Not an 'e' tag or invalid eventID, tagArray:", tagArray) // Debug output
		}
	}
}

func SplitReactionEvents() {
	mutex.Lock()
	defer mutex.Unlock()

	for eventID, relay := range trendingIds {
		switch r := relay; r {
		case "wss://relay.damus.io":
			damusRelay = append(damusRelay, eventID)
		case "wss://nos.lol":
			nosRelay = append(nosRelay, eventID)
		case "wss://purplerelay.com":
			purpleRelay = append(purpleRelay, eventID)
		case "wss://relay.primal.net":
			primalRelay = append(primalRelay, eventID)
		}
	}

	fmt.Println("primal relay: ", primalRelay)
}

func retrieveReactionEvents(reactionEvents []interface{}, relayUrl string, finished chan<- string) {
	fmt.Println("Starting retrieveReactionEvents for:", relayUrl)
	defer func() {
		fmt.Println("Finished retrieveReactionEvents for:", relayUrl) // Debug end
		finished <- relayUrl                                          // Signal that this relay has finished processing
	}()

	for _, event := range reactionEvents {
		if eventArray, ok := event.([]interface{}); ok {
			processEvent(eventArray, relayUrl)
		} else {
			fmt.Println("Invalid eventArray type:", event) // Debug output
		}
	}

	mutex.Lock()
	// fmt.Println("tempIds after processing:", tempIds) // Debug output

	for eventID, url := range tempIds {
		eventID = strings.TrimSpace(eventID)
		if eventID != "" {
			if len(trendingIds) < maxTrendingIds {
				trendingIds[eventID] = url
			} else {
				fmt.Println("Reached max limit of trendingIds")
				finished <- relayUrl
				break
			}
		} else {
			fmt.Println("Empty or whitespace eventID in tempIds:", eventID) // Debug output
		}
	}
	mutex.Unlock()

	// finished <- relayUrl

	// conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	// if err != nil {
	// 	log.Fatal("Dial error")
	// }

	// defer conn.Close()

	// subscriptionID, err := generateRandomString(16)
	// if err != nil {
	// 	log.Fatal("Error generating a subscription id: ", err)
	// }

	// subscriptionRequest := []interface{}{
	// 	"REQ",
	// 	subscriptionID,
	// 	map[string]interface{}{
	// 		"ids": trendingIds,
	// 	},
	// }

	// subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	// if err != nil {
	// 	log.Fatal("Error marshalling subscription request for trending events: ", err)
	// }

	// err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	// if err != nil {
	// 	log.Fatal("Error sending subscription request for trending events: ", err)
	// }

	// fmt.Println("Subscription request sent")

	// var trendingBatch []json.RawMessage

	// var log = logrus.New()

	// for {
	// 	_, message, err := conn.ReadMessage()
	// 	if err != nil {
	// 		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
	// 			log.WithFields(logrus.Fields{
	// 				"error": err,
	// 				"relay": relayUrl,
	// 			}).Error("WebSocket Error")
	// 			break
	// 		}
	// 		log.WithFields(logrus.Fields{
	// 			"error": err,
	// 			"relay": relayUrl,
	// 		}).Error("WebSocket read error")
	// 		break
	// 	}

	// 	if len(message) == 0 {
	// 		log.Warn("Received an empty message")
	// 		continue
	// 	}

	// 	var trendingMessage []interface{}
	// 	if err := json.Unmarshal(message, &trendingMessage); err != nil {
	// 		log.WithFields(logrus.Fields{
	// 			"error":   err,
	// 			"message": string(message),
	// 		}).Error("Error unmarshalling JSON")
	// 		continue
	// 	}

	// 	if len(trendingMessage) >= 3 {
	// 		if firstElement, ok := trendingMessage[0].(string); ok && firstElement == "EOSE" {
	// 			batchJSON, err := json.Marshal(trendingBatch)
	// 			if err != nil {
	// 				log.Fatal("Error marshalling the batched relay data: ", err)
	// 			}

	// 			redis.HandleRedis(batchJSON, relayUrl, finished)
	// 			break
	// 		}
	// 	}

	// 	trendingBatch = append(trendingBatch, message)
	// 	fmt.Println(len(trendingBatch))
	// }
}
