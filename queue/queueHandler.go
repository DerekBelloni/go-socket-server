package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SearchEventPubkey struct {
	SearchEvent []interface{}
	PubKey      string
}

func ConsumeQueue(queueName string) (<-chan amqp091.Delivery, *amqp.Channel, *amqp.Connection, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open a channel: %v", err)
	}

	queue, err := channel.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	fmt.Printf("queue name: %v\n", queue.Name)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to declare queue: %v", err)
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
		return nil, nil, nil, fmt.Errorf("failed to consume queue: %v", err)
	}

	return msgs, channel, conn, err
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

func NotesQueue(notesEvent []interface{}, eventChan chan string) {
	queueName := "user_notes"
	notesEventJSON, err := json.Marshal(notesEvent)
	if err != nil {
		fmt.Printf("Error marshalling notes event into JSON: %v\n", err)
	}
	setQueue(queueName, notesEventJSON, nil)
}

func MetadataQueue(metadataEvent []interface{}, eventChan chan string) {
	queueName := "user_metadata"
	metadataEventJSON, err := json.Marshal(metadataEvent)
	if err != nil {
		fmt.Printf("Error marshalling metadata event into JSON: %v\n", err)
	}
	setQueue(queueName, metadataEventJSON, eventChan)
	// eventChan <- "done"
}

func FollowListQueue(followListEvent []interface{}, eventChan chan string) {
	queueName := "follow_list"
	followListEventJSON, err := json.Marshal(followListEvent)
	if err != nil {
		fmt.Printf("Error marshalling follow list event into JSON: %v\n", err)
	}
	setQueue(queueName, followListEventJSON, eventChan)
}

func SearchQueue(searchEvent []interface{}, subsriptionPubkey string, eventChan chan string) {
	queueName := "search_results"
	searchResultStruct := SearchEventPubkey{
		SearchEvent: searchEvent,
		PubKey:      subsriptionPubkey,
	}
	searchResultStructJson, err := json.Marshal(searchResultStruct)
	if err != nil {
		fmt.Printf("Error marshalling search event into JSON: %v\n", err)
	}
	setQueue(queueName, searchResultStructJson, eventChan)
}
