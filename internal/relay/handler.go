package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/DerekBelloni/go-socket-server/data"
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

func handleUserNotes(conn *websocket.Conn, relayUrl string, userHexKey string) {
	defer conn.Close()

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds":   []int{1},
			"authors": []string{userHexKey},
			"since":   time.Now().Add(-24 * time.Hour).Unix(), // Last 24 hours
			"limit":   100,
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

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Apple")
			break
		}

		var userNotes []interface{}
		if err := json.Unmarshal(message, &userNotes); err != nil {
			fmt.Println("Banana")
			continue
		}

		fmt.Printf("user notes: %v\n", userNotes)
	}
}

func handleNewNote(conn *websocket.Conn, relayUrl string, newNote data.NewNote, noteFinished chan<- string) {
	// what are you doing here?
	defer func() {
		noteFinished <- relayUrl

		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "Websocket closure"))
		if err != nil {
			logrus.New().WithFields(logrus.Fields{
				"error": err,
				"relay": relayUrl,
			}).Warn("Error sending socket close message")
			conn.Close()
		}
	}()

	timestamp := time.Now().Unix()

	newNoteEvent := data.NostrEvent{
		PubKey:    newNote.PubHexKey,
		CreatedAt: timestamp,
		Kind:      newNote.Kind,
		Tags:      [][]string{},
		Content:   newNote.Content,
	}

	var idWg sync.WaitGroup

	idWg.Add(1)
	go func(note data.NostrEvent) {
		defer idWg.Done()
		err := newNoteEvent.GenerateId()
		if err != nil {
			log.Fatal("Error generating event ID: ", err)
			return
		}
	}(newNoteEvent)
	idWg.Wait()

	err := newNoteEvent.SignEvent(newNote.PrivHexKey)
	if err != nil {
		log.Fatal("Error signing the event: ", err)
	}

	isValid, err := newNoteEvent.VerifySignature()
	if err != nil {
		log.Fatal("Error verifying signature: ", err)
	}
	if !isValid {
		log.Fatal("Generated signature is not valid")
	}

	eventWrapper := []interface{}{"EVENT", newNoteEvent}

	newNoteJson, err := json.Marshal(eventWrapper)
	if err != nil {
		log.Fatal("Error marshalling the event into JSON: ", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, newNoteJson)
	if err != nil {
		log.Fatal("Error sending subscription request: ", err)
	}

	var log = logrus.New()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
				"relay": relayUrl,
			}).Error("WebSocket read error")
			break
		}
		var newNoteMsg []interface{}
		err = json.Unmarshal(message, &newNoteMsg)
		if err != nil {
			log.Fatal("Error unmarshalling json: ", err)
		}
		fmt.Printf("new message: %v\n", newNoteMsg)
		// need to send something back through the channel to indicate the event was susccessful or not
	}
}

// I can probably hand in a connection type here as well
func handleRelayConnection(conn *websocket.Conn, relayUrl string, finished chan<- string, userHexKey string) {
	defer conn.Close()

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
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
		log.Fatal("Error marshalling subscription request: ", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	if err != nil {
		log.Fatal("Error sending subscription request: ", err)
	}

	var log = logrus.New()

	// the loop for connecting, reading and writing can probably be abstracted out into its own method
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			// if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			// 	log.WithFields(logrus.Fields{
			// 		"error": err,
			// 		"relay": relayUrl,
			// 	}).Error("WebSocket Error")
			// 	break
			// }
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

		redis.HandleMetaData(jsonMetadata, finished, relayUrl, userHexKey, conn)
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
