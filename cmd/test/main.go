package main

import (
	"fmt"
	"log"

	"github.com/DerekBelloni/go-socket-server/internal/relay"
	amqp "github.com/rabbitmq/amqp091-go"
)

func userMetadataQueue(relayUrls []string, finished chan<- string) {
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

	var userHexKey []byte

	// routine for each queue
	if queue.Name != "user_pub_key" {
		for d := range msgs {
			userHexKey = d.Body
		}
	}

	for _, url := range relayUrls {
		relay.ConnectToRelay(url, finished, "user_metadata", userHexKey)
	}

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
}

func main() {
	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://nos.lol",
		"wss://purplerelay.com",
		"wss://relay.primal.net",
		"wss://relay.nostr.band",
	}

	finished := make(chan string)
	var forever chan struct{}

	// Queue: User Metadata
	go userMetadataQueue(relayUrls, finished)

	// Queue: Posting a Note

	<-forever

	// finished := make(chan string)
	// done := make(chan bool)
	// var wg sync.WaitGroup

	// ticker := time.NewTicker(10 * time.Minute)
	// defer ticker.Stop()

	// for _, relayUrl := range relayUrls {
	// 	wg.Add(1)
	// 	go func(relayUrl string) {
	// 		relay.ConnectToRelay(relayUrl, finished)
	// 		for {
	// 			select {
	// 			case <-ticker.C:
	// 				wg.Add(1)
	// 				go func() {
	// 					relay.ConnectToRelay(relayUrl, finished)
	// 				}()
	// 			case <-done:
	// 				return
	// 			}
	// 		}
	// 	}(relayUrl)
	// }

	// sig := make(chan os.Signal, 1)
	// signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// go func() {
	// 	<-sig
	// 	close(done)
	// 	wg.Wait()
	// 	os.Exit(0)
	// }()

	// for relayUrl := range finished {
	// 	fmt.Printf("Finished processing metadata for user: %s\n", relayUrl)
	// 	wg.Done()
	// }

	// wg.Wait()
}
