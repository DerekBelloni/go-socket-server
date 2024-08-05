package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

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

// I can probably hand in a connection type here as well
func handleRelayConnection(conn *websocket.Conn, relayUrl string, finished chan<- string, userHexKey string) {
	fmt.Printf("Hex key in handler: %v\n", userHexKey)
	defer conn.Close()

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
	}

	pubKeyHex := "9ba9ec6ed76fb4f4b94c27ae7932b3e8eacab0787162ba71d7d1b9c61d0e5c74"

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds":   []int{0},
			"authors": []string{pubKeyHex},
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

	fmt.Println("Metadata request sent")
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

		var metadataMessage []interface{}
		if err := json.Unmarshal(message, &metadataMessage); err != nil {
			log.WithFields(logrus.Fields{
				"error":   err,
				"message": string(message),
			}).Error("Error unmarshalling JSON")
			continue
		}

		var jsonMetadata []byte

		if len(metadataMessage) == 0 || metadataMessage[0] == "EOSE" {
			fmt.Println("Received an empty message or end of stream")
			continue
		}

		if metadataMessage[0] != "EOSE" {
			jsonMetadata, err = json.Marshal(metadataMessage)
		}

		if err != nil {
			log.Fatal("Error marshalling user metadata into JSON", err)
			break
		}

		redis.HandleMetaData(jsonMetadata, finished, relayUrl, pubKeyHex)
	}

	// var batch []json.RawMessage
	// var log = logrus.New()
	// // var reactionEvents []interface{}

	// eventChan := make(chan json.RawMessage, 1000)
	// doneChan := make(chan struct{})

	// go func() {
	// 	for event := range eventChan {
	// 		batch = append(batch, event)
	// 		if len(batch) >= 1000 {
	// 			batchJSON, err := json.Marshal(batch)
	// 			if err != nil {
	// 				log.Fatal("Error marshalling the batched relay data: ", err)
	// 			}
	// 			redis.HandleRedis(batchJSON, relayUrl, finished, "banana")
	// 			break
	// 		}
	// 	}
	// 	doneChan <- struct{}{}
	// }()
	// messageCount := 0

	// for messageCount < 1000 {
	// 	fmt.Printf("message count: %d", messageCount)
	// 	_, message, err := conn.ReadMessage()
	// 	if err != nil {
	// 		// this error handling can get its own method
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

	// 	var rMessage []interface{}
	// 	if err := json.Unmarshal(message, &rMessage); err != nil {
	// 		log.WithFields(logrus.Fields{
	// 			"error":   err,
	// 			"message": string(message),
	// 		}).Error("Error unmarshalling JSON")
	// 		continue
	// 	}

	// 	if len(rMessage) > 2 {
	// 		spew.Dump("rmessage: ", rMessage)
	// 		eventMap, ok := rMessage[2].(map[string]interface{})
	// 		if !ok {
	// 			continue
	// 		}

	// 		kind, ok := eventMap["kind"].(float64)
	// 		if !ok || int(kind) != 1 {
	// 			continue
	// 		}
	// 	}
	// }

	// close(eventChan)
	// <-doneChan
}

// func extractETag(tag []interface{}) (bool, string) {
// 	if len(tag) < 2 {
// 		return false, ""
// 	}

// 	key, ok := tag[0].(string)
// 	if !ok || key != "e" {
// 		return false, ""
// 	}

// 	eventID, ok := tag[1].(string)
// 	if !ok {
// 		return false, ""
// 	}

// 	return true, eventID
// }

// func processRelay(relaySlice *[]string, eventID string) bool {
// 	if len(*relaySlice) < 50 {
// 		*relaySlice = append(*relaySlice, eventID)
// 		return true
// 	}
// 	spew.Dump("relay slice: ", relaySlice)
// 	return false
// }

// func processEvent(eventArray []interface{}, relayUrl string) bool {
// 	for _, tag := range eventArray {
// 		tagArray, ok := tag.([]interface{})
// 		if !ok {
// 			fmt.Println("Skipping non-array tag:", tag)
// 			continue
// 		}
// 		if isE, eventID := extractETag(tagArray); isE {
// 			eventID = strings.TrimSpace(eventID)
// 			if eventID == "" {
// 				continue
// 			}
// 			switch relay := relayUrl; relay {
// 			case "wss://relay.damus.io":
// 				return processRelay(&damusRelay, eventID)
// 			case "wss://nos.lol":
// 				return processRelay(&nosRelay, eventID)
// 			case "wss://purplerelay.com":
// 				return processRelay(&purpleRelay, eventID)
// 			case "wss://relay.primal.net":
// 				return processRelay(&primalRelay, eventID)
// 			}
// 		}
// 	}
// 	return true
// }

// func parseReactionEvents(reactionEvents []interface{}, relayUrl string) bool {
// 	for _, event := range reactionEvents {
// 		if eventArray, ok := event.([]interface{}); ok {
// 			shouldContinue := processEvent(eventArray, relayUrl)
// 			if !shouldContinue {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// func retrieveReactionEvents(relayUrl string, finished chan<- string) {
// 	conn, _, err := websocket.DefaultDialer.Dial(relayUrl, nil)
// 	if err != nil {
// 		log.Fatal("Dial error: ", err)
// 	}

// 	defer conn.Close()

// 	subscriptionID, err := generateRandomString(16)
// 	if err != nil {
// 		log.Fatal("Error generating a subscription id: ", err)
// 	}

// 	var relaySlice *[]string

// 	switch relayUrl {
// 	case "wss://relay.damus.io":
// 		relaySlice = &damusRelay
// 	case "wss://nos.lol":
// 		relaySlice = &nosRelay
// 	case "wss://purplerelay.com":
// 		relaySlice = &purpleRelay
// 	case "wss://relay.primal.net":
// 		relaySlice = &primalRelay
// 	default:
// 		log.Printf("Uknown relay URL: %s\n", relayUrl)
// 		return
// 	}
// 	fmt.Printf("length: %d\n", len(damusRelay))
// 	subscriptionRequest := []interface{}{
// 		"REQ",
// 		subscriptionID,
// 		map[string]interface{}{
// 			"ids": relaySlice,
// 		},
// 	}

// 	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
// 	if err != nil {
// 		log.Fatal("Error sending subscription request: ", err)
// 	}

// 	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
// 	if err != nil {
// 		log.Fatal("Error sending subscription")
// 	}

// 	fmt.Println("Subscription request sent")

// 	var batch []json.RawMessage
// 	var log = logrus.New()

// 	for {
// 		_, message, err := conn.ReadMessage()
// 		if err != nil {
// 			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
// 				log.WithFields(logrus.Fields{
// 					"error": err,
// 				}).Error("WebSocket read error")
// 			}
// 			break
// 		}

// 		if len(message) == 0 {
// 			log.Warn("Received an empty message")
// 			continue
// 		}

// 		var rMessage []interface{}
// 		if err := json.Unmarshal(message, &rMessage); err != nil {
// 			log.WithFields(logrus.Fields{
// 				"error":   err,
// 				"message": string(message),
// 			}).Error("Error unmarshalling JSON")
// 			continue
// 		}

// 		if len(rMessage) > 2 {
// 			_, ok := rMessage[0].(string)
// 			if !ok {
// 				continue
// 			}

// 			batch = append(batch, message)
// 			batchJson, err := json.Marshal(batch)
// 			if err != nil {
// 				fmt.Println("Error marshalling the batched relay data for trending events: ", err)
// 			}
// 			if len(batchJson) >= 25 {
// 				redis.HandleRedis(batchJson, relayUrl, finished, "trending")
// 				break
// 			}
// 		}
// 	}
// }
