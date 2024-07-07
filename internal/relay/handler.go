package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
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

func isRootTag(tag []interface{}) (bool, string) {
	if len(tag) < 4 {
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

	relation, ok := tag[3].(string)
	if !ok || relation != "root" {
		return false, ""
	}

	return true, eventID
}

func processEvent(event []interface{}) []string {
	var eventIDs []string

	// type assert the tagArray, if ok then set the isRoot, eventId by calling to isRootTag method
	for _, tag := range event {
		tagArray, ok := tag.([]interface{})
		if !ok {
			continue
		}
		if isRoot, eventID := isRootTag(tagArray); isRoot {
			// spew.Dump("event id: ", eventID)
			eventIDs = append(eventIDs, eventID)
		}
	}
	// spew.Dump("Event ids: ", eventIDs)
	return eventIDs
}

func retrieveReactionEvents(reactionEvents []interface{}, relayUrl string, finished chan<- string) {
	trendingEventIDs := make([]string, 0)

	for _, event := range reactionEvents {
		if eventArray, ok := event.([]interface{}); ok {
			eventIDs := processEvent(eventArray)
			spew.Dump(eventIDs)
			for _, id := range eventIDs {
				if id != "" {
					trendingEventIDs = append(trendingEventIDs, id)
				}
			}
		}
	}

	fmt.Printf("Trending event IDs for %s: %d\n", relayUrl, len(trendingEventIDs))

	if len(trendingEventIDs) == 0 {
		fmt.Println("No trending event IDs found for", relayUrl)
		return
	}

	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
	if err != nil {
		log.Fatal("Dial error")
	}

	defer conn.Close()

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"ids": trendingEventIDs,
		},
	}

	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		log.Fatal("Error marshalling subscription request for trending events: ", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	if err != nil {
		log.Fatal("Error sending subscription request for trending events: ", err)
	}

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
