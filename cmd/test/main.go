package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/DerekBelloni/go-socket-server/data"
	"github.com/DerekBelloni/go-socket-server/internal/relay"
	amqp "github.com/rabbitmq/amqp091-go"
)

func createNote(relayUrls []string) {
	forever := make(chan struct{})
	noteFinished := make(chan string)

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
		"new_note",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		fmt.Println("Failed to declare a queue", err)
	}

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
	}

	var wg sync.WaitGroup

	for msg := range msgs {
		go func(msg amqp.Delivery) {
			var newNote data.NewNote
			err := json.Unmarshal([]byte(msg.Body), &newNote)
			if err != nil {
				fmt.Printf("Error unmarshalling json: %v\n", err)
				return
			}

			for _, url := range relayUrls {
				wg.Add(1)
				go func(url string) {
					defer wg.Done()
					relay.SendNoteToRelay(url, newNote, noteFinished)
				}(url)
			}
		}(msg)
	}

	<-forever
}

func userNotes(relayUrls []string, userHexKey string, notesFinished chan<- string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, url := range relayUrls {
		var noteWg sync.WaitGroup
		noteWg.Add(1)
		go func(url string) {
			defer noteWg.Done()
			relay.GetUserNotes(ctx, cancel, url, userHexKey, notesFinished)
		}(url)
	}
}

func metadataSetQueue(conn *amqp.Connection, userHexKey string) {
	fmt.Println("apple")
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
		fmt.Println("User metadata stashed in Redis")
	}
}

func userMetadataQueue(relayUrls []string) {
	forever := make(chan struct{})
	finished := make(chan string)
	notesFinished := make(chan string)
	metadataSet := make(chan string)

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
	}

	// var metaWg sync.WaitGroup
	var notesWg sync.WaitGroup

	go func() {
		for relayUrl := range finished {
			fmt.Printf("Finished processing metadata for relay: %s\n", relayUrl)
			// metaWg.Done()
		}
	}()
	go func() {
		for relayUrl := range notesFinished {
			fmt.Printf("Finished processing user notes relay: %s\n", relayUrl)
			notesWg.Done()
		}
	}()

	for d := range msgs {
		go func(d amqp.Delivery) {
			userHexKey := string(d.Body)
			for _, url := range relayUrls {
				go func(url string, conn *amqp.Connection) {
					relay.GetUserMetadata(url, finished, "user_metadata", userHexKey, metadataSet)
				}(url, conn)
			}

			<-metadataSet
			metadataSetQueue(conn, userHexKey)

			for _, url := range relayUrls {
				go func(url string, conn *amqp.Connection) {
					userNotes(relayUrls, userHexKey, notesFinished)
				}(url, conn)
			}

			<-notesFinished
			// followList(relayUrls, userHexKey)
		}(d)
	}

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func classifiedListings(relayUrls []string) {
	var innerWg sync.WaitGroup
	for _, url := range relayUrls {
		innerWg.Add(1)
		go func(relayUrl string) {
			defer innerWg.Done()
			relay.GetClassifiedListings(relayUrl)
		}(url)
		innerWg.Wait()
	}
}

func followList(relayUrls []string, userHexKey string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	for _, relayUrl := range relayUrls {
		wg.Add(1)
		go func(relayUrl string) {
			defer wg.Done()
			relay.GetFollowList(ctx, cancel, relayUrl, userHexKey)
		}(relayUrl)
	}
}

func main() {
	relayUrls := []string{
		// "wss://relay.damus.io",
		"wss://nos.lol",
		// "wss://purplerelay.com",
		"wss://relay.primal.net",
		"wss://relay.nostr.band",
	}

	var forever chan struct{}

	// Queue: User Metadata
	go userMetadataQueue(relayUrls)
	// Queue: Posting a Note
	// go createNote(relayUrls)

	// go classifiedListings(relayUrls)

	// not sure if I need a blocking channel here
	<-forever
}
