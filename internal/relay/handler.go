package relay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/DerekBelloni/go-socket-server/data"
	"github.com/DerekBelloni/go-socket-server/internal/redis"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func clearWebSocketBuffer(conn *websocket.Conn) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
func handleFollowList(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, relayUrl string, userHexKey string) {
	// defer conn.Close()

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds": []int{3},
			"limit": 1,
			"tags": [][]string{
				{"p", userHexKey, relayUrl},
			},
		},
	}

	subscriptionRequestJSON, err := json.Marshal(subscriptionRequest)
	if err != nil {
		log.Fatal("Error marshalling subscription request: ", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	if err != nil {
		log.Fatal("Error marshalling subscription request: ", err)
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logrus.New().WithFields(logrus.Fields{
				"error": err,
				"relay": relayUrl,
			}).Warn("Error sending socket close message, follow list")
			conn.Close()
		}

		var followMessage []interface{}
		if err := json.Unmarshal(message, &followMessage); err != nil {
			logrus.New().WithFields(logrus.Fields{
				"error": err,
				"relay": relayUrl,
			}).Warn("Error sending socket close message, follow listwd")
			conn.Close()
		}

		fmt.Printf("follow list: %v, relay: %v\n", followMessage[0], relayUrl)

		if len(followMessage) <= 0 {
			continue
		}

		messageType, ok := followMessage[0].(string)
		if !ok {
			continue
		}

		switch messageType {
		case "EVENT":
			conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
			if err != nil {
				fmt.Println("Failed to connect to RabbitMQ", err)
			}
			defer conn.Close()

			channel, err := conn.Channel()
			if err != nil {
				fmt.Println("Failed to open a channel")
			}

			defer channel.Close()

			queue, err := channel.QueueDeclare(
				"follow_list",
				false,
				false,
				false,
				false,
				nil,
			)

			if err != nil {
				fmt.Println("Failed to declare a queue", err)
			}

			jsonUserNotes, err := json.Marshal(followMessage)
			if err != nil {
				fmt.Println("Failed to marshall follow list into JSON")
				continue
			}

			err = channel.PublishWithContext(ctx,
				"",
				queue.Name,
				false,
				false,
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        []byte(jsonUserNotes),
				})

			if err != nil {
				fmt.Printf("Failed to send follow list with rabbitmq: %v\n", err)
			} else {
				fmt.Printf("User notes sent to rabbitmq\n")
				cancel()
			}
		case "EOSE":
			fmt.Println("End of stored events")
			return

		case "NOTICE":
			fmt.Printf("Received NOTICE: %v\n", followMessage[1])
		default:
			fmt.Printf("Unknown message type: %s\n", messageType)
		}
	}
}

func handleClassifiedListings(conn *websocket.Conn, relayUrl string) {
	// defer conn.Close()

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Fatal("Error generating a subscription id: ", err)
	}

	subscriptionRequest := []interface{}{
		"REQ",
		subscriptionID,
		map[string]interface{}{
			"kinds": []int{30402},
			"limit": 100,
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
			logrus.New().WithFields(logrus.Fields{
				"error": err,
				"relay": relayUrl,
			}).Warn("Error sending socket close message")
			conn.Close()
		}

		var listingsMessage []interface{}
		if err := json.Unmarshal(message, &listingsMessage); err != nil {
			logrus.New().WithFields(logrus.Fields{
				"error": err,
				"relay": relayUrl,
			}).Warn("Error sending socket close message, classified listings")
			conn.Close()
		}

		spew.Dump("listings message: ", listingsMessage)
	}
}

func handleUserNotes(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, relayUrl string, userHexKey string) {
	// defer conn.Close()
	defer cancel()
	log := logrus.WithField("relay", relayUrl)

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Error("Error generating a subscription id: ", err)
	}

	// I can abstract subscriptions into methods on types I make for different nostr request types
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
		log.Error("Error marshalling subscription request: ", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	if err != nil {
		log.Error("Error sending subscription request, user notes: ", err)
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message from relay")
			break
		}

		var userNotes []interface{}
		if err := json.Unmarshal(message, &userNotes); err != nil {
			fmt.Println("Error unmarshalling relay message for user notes")
			continue
		}

		if len(userNotes) < 2 {
			continue
		}
		fmt.Printf("passed length check")

		messageType, ok := userNotes[0].(string)
		if !ok {
			continue
		}

		switch messageType {
		case "EVENT":
			conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
			if err != nil {
				fmt.Println("Failed to connect to RabbitMQ", err)
			}
			// defer conn.Close()

			channel, err := conn.Channel()
			if err != nil {
				fmt.Println("Failed to open a channel")
			}

			defer channel.Close()

			queue, err := channel.QueueDeclare(
				"user_notes",
				false,
				false,
				false,
				false,
				nil,
			)

			if err != nil {
				fmt.Println("Failed to declare a queue", err)
			}
			spew.Dump("user notes: ", userNotes)
			jsonUserNotes, err := json.Marshal(userNotes)
			if err != nil {
				fmt.Println("Failed to marshall user notes into JSON")
				continue
			}

			err = channel.PublishWithContext(ctx,
				"",
				queue.Name,
				false,
				false,
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        []byte(jsonUserNotes),
				})

			if err != nil {
				fmt.Printf("Failed to send notes with rabbitmq: %v\n", err)
			} else {
				fmt.Printf("User notes sent to rabbitmq\n")
				cancel()
			}

		case "EOSE":
			fmt.Println("End of stored events")
			return

		case "NOTICE":
			fmt.Printf("Received NOTICE: %v\n", userNotes[1])
		default:
			fmt.Printf("Unknown message type: %s\n", messageType)
		}
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
			}).Error("WebSocket read error, new note")
			break
		}
		var newNoteMsg []interface{}
		err = json.Unmarshal(message, &newNoteMsg)
		if err != nil {
			log.Fatal("Error unmarshalling json: ", err)
		}
		fmt.Printf("new message: %v\n", newNoteMsg)
	}
}

// I can probably hand in a connection type here as well
func handleRelayConnection(conn *websocket.Conn, relayUrl string, finished chan<- string, userHexKey string) {
	// defer conn.Close()
	log := logrus.WithField("relay", relayUrl)

	subscriptionID, err := generateRandomString(16)
	if err != nil {
		log.Error("Error generating a subscription id: ", err)
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
		log.Error("Error marshalling subscription request: ", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, subscriptionRequestJSON)
	if err != nil {
		log.Error("Error sending subscription request: ", err)
	}

	// the loop for connecting, reading and writing can probably be abstracted out into its own method
	if userHexKey != "" {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.WithFields(logrus.Fields{
					"error": err,
					"relay": relayUrl,
				}).Error("WebSocket read error, main relay connection")
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
			break
		}
	}
}
