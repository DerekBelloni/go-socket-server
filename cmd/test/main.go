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
			fmt.Printf("New Note: %v\n", newNote)
		}(msg)
	}

	<-forever
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
	fmt.Printf("User hex key: %v\n", userHexKey)
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
	fmt.Printf("queue: %v\n", queue)
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

	var innerWg sync.WaitGroup
	// var outerWg sync.WaitGroup

	go func() {
		for relayUrl := range finished {
			fmt.Printf("Finished processing relay: %s\n", relayUrl)
		}
	}()

	// need to alter this check so I am finding the queue by its name ON the msgs
	// maybe check the msgs is not empty
	if queue.Name == "user_pub_key" {
		for d := range msgs {
			fmt.Printf("msg: %v\n", string(d.Body))
			go func(d amqp.Delivery) {
				userHexKey := string(d.Body)
				for _, url := range relayUrls {
					innerWg.Add(1)
					go func(url string) {
						defer innerWg.Done()
						relay.ConnectToRelay(url, finished, "user_metadata", userHexKey)
					}(url)
				}
				innerWg.Wait()
				metadataSetQueue(conn, userHexKey)
			}(d)
		}
	}
	// outerWg.Wait()

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func main() {
	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://nos.lol",
		// "wss://purplerelay.com",
		"wss://relay.primal.net",
		"wss://relay.nostr.band",
	}

	// finished := make(chan string)
	var forever chan struct{}

	// Queue: User Metadata
	go userMetadataQueue(relayUrls)

	// Queue: Posting a Note
	go createNote(relayUrls)

	<-forever
}
