package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DerekBelloni/go-socket-server/internal/relay"
	amqp "github.com/rabbitmq/amqp091-go"
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

func userNotes(relayUrls []string, userHexKey string, notesFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(relayUrl string) {
			relay.GetUserNotes(relayUrl, userHexKey, notesFinished)
		}(relayUrl)
	}
}

func userMetadata(relayUrls []string, userHexKey string, metadataFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(url string) {
			relay.GetUserMetadata(url, userHexKey, metadataFinished)
		}(relayUrl)
	}
}

func followList(relayUrls []string, userHexKey string, followsFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(relayUrl string) {
			relay.GetFollowList(relayUrl, userHexKey, followsFinished)
		}(relayUrl)
	}
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

func userMetadataQueue(relayUrls []string) {
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

	for d := range msgs {
		go func(d amqp.Delivery) {
			userHexKey := string(d.Body)
			if userHexKey != "" {
				userMetadata(relayUrls, userHexKey, metadataFinished)
				<-metadataFinished

				userNotes(relayUrls, userHexKey, notesFinished)
				<-notesFinished
				fmt.Println("past notes finsished channel")

				followList(relayUrls, userHexKey, followsFinished)
				<-followsFinished
			}
		}(d)
	}

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func main() {
	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://nos.lol",
		"wss://purplerelay.com",
		"wss://relay.primal.net",
		"wss://relay.nostr.band",
	}

	var forever chan struct{}

	go userMetadataQueue(relayUrls)

	// Queue: Posting a Note
	// go createNote(relayUrls)
	// go classifiedListings(relayUrls)

	<-forever
}
