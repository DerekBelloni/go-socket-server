package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/DerekBelloni/go-socket-server/internal/data"
	"github.com/DerekBelloni/go-socket-server/internal/relay"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	pubKeyUUID     = make(map[string]string)
	pubKeyUUIDLock sync.RWMutex
)

// func createNote(relayUrls []string) {
// 	forever := make(chan struct{})
// 	noteFinished := make(chan string)

// 	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
// 	if err != nil {
// 		fmt.Println("Failed to connect to RabbitMQ", err)
// 	}
// 	defer conn.Close()

// 	channel, err := conn.Channel()
// 	if err != nil {
// 		fmt.Println("Failed to open a channel")
// 	}

// 	defer channel.Close()

// 	queue, err := channel.QueueDeclare(
// 		"new_note",
// 		false,
// 		false,
// 		false,
// 		false,
// 		nil,
// 	)

// 	if err != nil {
// 		fmt.Println("Failed to declare a queue", err)
// 	}

// 	msgs, err := channel.Consume(
// 		queue.Name,
// 		"",
// 		true,
// 		false,
// 		false,
// 		false,
// 		nil,
// 	)
// 	if err != nil {
// 		fmt.Println("Failed to register a consumer")
// 	}

// 	var wg sync.WaitGroup

// 	for msg := range msgs {
// 		go func(msg amqp.Delivery) {
// 			var newNote data.NewNote
// 			err := json.Unmarshal([]byte(msg.Body), &newNote)
// 			if err != nil {
// 				fmt.Printf("Error unmarshalling json: %v\n", err)
// 				return
// 			}

// 			for _, url := range relayUrls {
// 				wg.Add(1)
// 				go func(url string) {
// 					defer wg.Done()
// 					relay.SendNoteToRelay(url, newNote, noteFinished)
// 				}(url)
// 			}
// 		}(msg)
// 	}

// 	<-forever
// }

// func classifiedListings(relayUrls []string) {
// 	var innerWg sync.WaitGroup
// 	for _, url := range relayUrls {
// 		innerWg.Add(1)
// 		go func(relayUrl string) {
// 			defer innerWg.Done()
// 			relay.GetClassifiedListings(relayUrl)
// 		}(url)
// 		innerWg.Wait()
// 	}
// }

func createUserContext(userHexKey string) (context.Context, context.CancelFunc) {
	return context.WithCancel(context.WithValue(context.Background(), "userPubKey", userHexKey))
}

func userNotes(relayUrls []string, relayConnection *relay.RelayConnection, userHexKey string, notesFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(relayUrl string) {
			fmt.Println("in user notes")
			relayConnection.GetUserNotes(relayUrl, userHexKey, notesFinished)
		}(relayUrl)
	}
}

func userMetadata(relayUrls []string, relayConnection *relay.RelayConnection, userHexKey string, metadataFinished chan<- string) {
	fmt.Println("in user metadata")
	for _, relayUrl := range relayUrls {
		go func(url string) {
			relayConnection.GetUserMetadata(url, userHexKey, metadataFinished)
		}(relayUrl)
	}
}

func followList(relayUrls []string, relayConnection *relay.RelayConnection, userHexKey string, followsFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(relayUrl string) {
			relayConnection.GetFollowList(relayUrl, userHexKey, followsFinished)
		}(relayUrl)
	}
}

func followsMetadata(relyaUrls []string, relayConnection *relay.RelayConnection, userHexKey string) {
	for _, relayUrl := range relyaUrls {
		go func(relayUrl string) {
			relayConnection.GetFollowListMetadata(relayUrl, userHexKey)
		}(relayUrl)
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

func metadataSetQueue(conn *amqp.Connection, userHexKey string) {
	channel, err := conn.Channel()
	if err != nil {
		fmt.Println("Failed to open a channel")
	}

	defer channel.Close()

	queue, err := channel.QueueDeclare(
		"metadata_set",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		fmt.Println("Failed to declare a queue", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = channel.PublishWithContext(ctx,
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(userHexKey),
		})

	if err != nil {
		fmt.Println("Failed to publish message")
	} else {
		fmt.Println("User metadata sent to RabbitMQ")
	}
}

func userMetadataQueue(relayUrls []string, relayConnection *relay.RelayConnection) {
	forever := make(chan struct{})
	metadataFinished := make(chan string)
	notesFinished := make(chan string)
	followsFinished := make(chan string)

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
		"user_pub_key",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		fmt.Println("Failed to declare a queue", err)
	}

	go func() {
		for {
			msgs, err := channel.Consume(
				queue.Name,
				"",
				true,
				false,
				false,
				false,
				nil,
			)

			if err != nil {
				fmt.Println("Failed to register a consumer")
				return
			}

			for d := range msgs {
				userHexKeyUUID := string(d.Body)
				parts := strings.Split(userHexKeyUUID, ":")
				if len(parts) != 2 {
					continue
				}

				userHexKey := parts[0]
				uuid := parts[1]

				pubKeyUUIDLock.Lock()
				existingUUID, exists := pubKeyUUID[userHexKey]
				if exists && existingUUID == uuid {
					pubKeyUUIDLock.Unlock()
					fmt.Printf("Mapping already exists for Pubkey: %s. Skipping processing\n", userHexKey)
					continue
				}

				pubKeyUUID[userHexKey] = uuid
				pubKeyUUIDLock.Unlock()

				userMetadata(relayUrls, relayConnection, userHexKey, metadataFinished)
				<-metadataFinished
				fmt.Println("past user metadata channel")

				userNotes(relayUrls, relayConnection, userHexKey, notesFinished)
				<-notesFinished
				fmt.Println("Completed user notes processing")

				followList(relayUrls, relayConnection, userHexKey, followsFinished)
				<-followsFinished
				fmt.Println("Completed follow list processing")
			}
		}
	}()

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func userFollowsMetadataQueue(relayUrls []string, relayConnection *relay.RelayConnection) {
	forever := make(chan struct{})
	queueName := "follow_list_metadata"

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
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		fmt.Println("Failed to declare a queue", err)
	}

	go func() {
		for {
			msgs, err := channel.Consume(
				queue.Name,
				"",
				true,
				false,
				false,
				false,
				nil,
			)

			if err != nil {
				fmt.Println("Failed to register a consumer")
				return
			}

			for d := range msgs {
				userHexKeyUUID := string(d.Body)
				parts := strings.Split(userHexKeyUUID, ":")
				if len(parts) != 2 {
					fmt.Println("here")
					continue
				}

				userHexKey := parts[0]
				uuid := parts[1]

				pubKeyUUIDLock.Lock()
				existingUUID, exists := pubKeyUUID[userHexKey]
				if exists && existingUUID == uuid {
					pubKeyUUIDLock.Unlock()
					fmt.Printf("Mapping already exists for Pubkey: %s. Skipping processing\n", userHexKey)
					continue
				}

				pubKeyUUID[userHexKey] = uuid
				pubKeyUUIDLock.Unlock()

				followsMetadata(relayUrls, relayConnection, userHexKey)
			}
		}
	}()
	log.Printf("[*] Waiting for follows list messages. To exit press CTRL+C")
	<-forever
}

func main() {
	relayManager := data.NewRelayManager(nil)
	relayConnection := relay.NewRelayConnection(relayManager)
	relayManager.Connector = relayConnection

	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://nos.lol",
		"wss://purplerelay.com",
		"wss://relay.primal.net",
		"wss://relay.nostr.band",
	}

	var forever chan struct{}

	go userMetadataQueue(relayUrls, relayConnection)

	go userFollowsMetadataQueue(relayUrls, relayConnection)

	// Queue: Posting a Note
	// go createNote(relayUrls)
	// go classifiedListings(relayUrls)

	<-forever
}
