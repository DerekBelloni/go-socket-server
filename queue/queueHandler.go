package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DerekBelloni/go-socket-server/data"
	"github.com/rabbitmq/amqp091-go"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SearchEvent struct {
	Event     []interface{}
	SearchKey string
}

type FollowsEvent struct {
	FollowsEvent []interface{}
	UserPubkey   *string
	UUID         *string
	Type         string
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

func setQueue(queueName string, eventJson []byte) {
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
		true,
		false,
		false,
		nil,
	)

	if err != nil {
		fmt.Printf("Failed to declare queue: %v\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fmt.Printf("queue being publsihed: %v\n\n", queueName)
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
		cancel()
	}
}

func NotesQueue(notesEvent []interface{}, eventChan chan string, followsPubkey string) {
	queueName := "user_notes"

	if followsPubkey == "" {
		notesEventJSON, err := json.Marshal(notesEvent)
		if err != nil {
			fmt.Printf("Error marshalling notes event into JSON: %v\n", err)
		}
		setQueue(queueName, notesEventJSON)
		// eventChan <- "done"
	} else {
		followsEventStruct := FollowsEvent{
			FollowsEvent: notesEvent,
		}
		followsEventJSON, err := json.Marshal(&followsEventStruct)
		if err != nil {
			fmt.Printf("Error marshalling follow event into JSON")
		}
		setQueue(queueName, followsEventJSON)
	}
}

// This using the new NostrEvent which is alreadu packaged
func NewNotesQueue(event data.EventMessage, eventChan chan string) {
	queueName := "user_notes"

	notesEventJson, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("Error marshalling notes event into JSON: %v\n", err)
	}
	setQueue(queueName, notesEventJson)
}

func FollowsMetadataQueue(metadataEvent []interface{}, userPubkey string, followsPubkey string) {
	queueName := "follows_metadata"
	fmt.Printf("metadata event in queue: %v\n\n\n", metadataEvent)
	metadataEventJSON, err := json.Marshal(metadataEvent)
	if err != nil {
		fmt.Printf("Error marshalling metadata event into JSON: %v\n", err)
	}
	setQueue(queueName, metadataEventJSON)
}

// need a seperate method for follows metadata
func MetadataQueue(metadataEvent []interface{}, eventChan chan string) {
	queueName := "user_metadata"
	fmt.Printf("metadata event in queue: %v\n\n\n", metadataEvent)
	metadataEventJSON, err := json.Marshal(metadataEvent)
	if err != nil {
		fmt.Printf("Error marshalling metadata event into JSON: %v\n", err)
	}
	setQueue(queueName, metadataEventJSON)
	eventChan <- "done"
}

func FollowListQueue(followListEvent []interface{}, eventChan chan string) {
	queueName := "follow_list"
	followListEventJSON, err := json.Marshal(followListEvent)
	if err != nil {
		fmt.Printf("Error marshalling follow list event into JSON: %v\n", err)
	}

	setQueue(queueName, followListEventJSON)
}

func SearchQueue(searchEvent []interface{}, searchKey string, eventChan chan string) {
	queueName := "search_results"
	searchResultStruct := SearchEvent{
		Event:     searchEvent,
		SearchKey: searchKey,
	}
	searchResultStructJson, err := json.Marshal(searchResultStruct)
	if err != nil {
		fmt.Printf("Error marshalling search event into JSON: %v\n", err)
	}
	setQueue(queueName, searchResultStructJson)
}

func AuthorMetadataQueue(metadataEvent []interface{}, searchKey string) {
	queueName := "author_metadata"

	searchEvent := SearchEvent{
		Event:     metadataEvent,
		SearchKey: searchKey,
	}
	searchEventJSON, err := json.Marshal(searchEvent)
	if err != nil {
		fmt.Printf("Error marshalling search event into JSONL %v\n", err)
	}
	setQueue(queueName, searchEventJSON)
}
