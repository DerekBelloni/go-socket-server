package handler

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ConsumeQueue(queueName string) (<-chan amqp.Delivery, *amqp.Connection) {
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
	return msgs, conn
}

func setQueue(queueName string, eventJson []byte, eventChan chan<- string) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		fmt.Printf("Failed to connect to RabbitMQ: %v\n", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		fmt.Printf("Failed to open channel: %v\n", err)
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
		fmt.Printf("Failed to declare queue: %v\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = channel.PublishWithContext(
		ctx,
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(eventJson),
		})

	if err != nil {
		fmt.Printf("Error publishing relay event to queue: %v\n", err)
	} else {
		fmt.Println("Event published to RabbitMQ")
		cancel()
	}
}

func notesQueue(notesEvent []interface{}, eventChan chan string) {
	queueName := "user_notes"
	notesEventJSON, err := json.Marshal(notesEvent)
	if err != nil {
		fmt.Printf("Error marshalling notes event into JSON: %v\n", err)
	}
	setQueue(queueName, notesEventJSON, nil)
}

func metadataQueue(metadataEvent []interface{}, eventChan chan string) {
	queueName := "user_metadata"
	metadataEventJSON, err := json.Marshal(metadataEvent)
	if err != nil {
		fmt.Printf("Error marshalling metadata event into JSON: %v\n", err)
	}
	setQueue(queueName, metadataEventJSON, eventChan)
	// eventChan <- "done"
}

func followListQueue(followListEvent []interface{}, eventChan chan string) {
	queueName := "follow_list"
	followListEventJSON, err := json.Marshal(followListEvent)
	if err != nil {
		fmt.Printf("Error marshalling follow list event into JSON: %v\n", err)
	}
	setQueue(queueName, followListEventJSON, eventChan)
}
