package user

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"github.com/DerekBelloni/go-socket-server/core"
	"github.com/DerekBelloni/go-socket-server/data"
	"github.com/DerekBelloni/go-socket-server/relay"
	"github.com/DerekBelloni/go-socket-server/search"
	"github.com/DerekBelloni/go-socket-server/store"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Service struct {
	relayConnection *relay.RelayConnection
	searchTracker   core.SearchTracker
	relayUrls       []string
	pubKeyUUID      map[string]string
	pubKeyUUIDLock  sync.RWMutex
	searchUUID      map[string]string
	searchUUIDLock  sync.RWMutex
}

func NewService(relayConnection *relay.RelayConnection, relayUrls []string, searchTracker *search.SearchTrackerImpl) *Service {
	return &Service{
		relayConnection: relayConnection,
		relayUrls:       relayUrls,
		searchTracker:   searchTracker,
		pubKeyUUID:      make(map[string]string),
		searchUUID:      make(map[string]string),
	}
}

// Probably need to incorporate contexts
// func createUserContext(userHexKey string) (context.Context, context.CancelFunc) {
// 	return context.WithCancel(context.WithValue(context.Background(), "userPubKey", userHexKey))
// }

func (s *Service) checkAndUpdatePubkeyUUID(userHexKey, uuid string) bool {
	s.pubKeyUUIDLock.Lock()
	defer s.pubKeyUUIDLock.Unlock()

	existingUUID, exists := s.pubKeyUUID[userHexKey]

	if exists && existingUUID == uuid {
		return false
	}

	s.pubKeyUUID[userHexKey] = uuid
	return true
}

func (s *Service) createNote(note data.NewNote) {
	for _, relayUrl := range s.relayUrls {
		go func(relayUrl string) {
			s.relayConnection.SendNoteToRelay(relayUrl, note)
		}(relayUrl)
	}
}

func (s *Service) followList(relayUrls []string, userHexKey string, followsFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(relayUrl string) {
			s.relayConnection.GetFollowList(relayUrl, userHexKey, followsFinished)
		}(relayUrl)
	}
}

func (s *Service) followsMetadata(userHexKey string) {
	redisClient := store.NewRedisClient()
	pubKeys := redisClient.HandleFollowListPubKeys(userHexKey)

	for _, relayUrl := range s.relayUrls {
		go func(relayUrl string) {
			s.relayConnection.GetFollowListMetadata(relayUrl, userHexKey, pubKeys)
		}(relayUrl)
	}
}

func (s *Service) retrieveSearch(search string, uuid string, pubkey *string) {
	relayUrl := "wss://relay.nostr.band/"
	go func() {
		s.relayConnection.RetrieveSearch(relayUrl, search, s.searchTracker, uuid, pubkey)
	}()
}

func (s *Service) userNotes(relayUrls []string, userHexKey string, notesFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(relayUrl string) {
			s.relayConnection.GetUserNotes(relayUrl, userHexKey, notesFinished)
		}(relayUrl)
	}
}

func (s *Service) userMetadata(relayUrls []string, userHexKey string, metadataFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(url string) {
			s.relayConnection.GetUserMetadata(url, userHexKey, metadataFinished)
		}(relayUrl)
	}
}

func (s *Service) StartMetadataQueue() {
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

				if !s.checkAndUpdatePubkeyUUID(userHexKey, uuid) {
					continue
				}

				s.userMetadata(s.relayUrls, userHexKey, metadataFinished)
				<-metadataFinished
				fmt.Println("past user metadata channel")

				s.userNotes(s.relayUrls, userHexKey, notesFinished)
				<-notesFinished
				fmt.Println("Completed user notes processing")

				s.followList(s.relayUrls, userHexKey, followsFinished)
				<-followsFinished
				fmt.Println("Completed follow list processing")
			}
		}
	}()

	log.Printf("[*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func (s *Service) StartFollowsMetadataQueue() {
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
					continue
				}

				userHexKey := parts[0]
				uuid := parts[1]

				s.pubKeyUUIDLock.Lock()
				existingUUID, exists := s.pubKeyUUID[userHexKey]
				if exists && existingUUID == uuid {
					s.pubKeyUUIDLock.Unlock()
					fmt.Printf("Mapping already exists for Pubkey: %s. Skipping processing\n", userHexKey)
					continue
				}

				s.pubKeyUUID[userHexKey] = uuid
				s.pubKeyUUIDLock.Unlock()

				s.followsMetadata(userHexKey)
			}
		}
	}()
	log.Printf("[*] Waiting for follows list messages. To exit press CTRL+C")
	<-forever
}

func (s *Service) StartCreateNoteQueue() {
	forever := make(chan struct{})

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
			}

			for d := range msgs {
				var result map[string]interface{}
				var newNote data.NewNote

				err := json.Unmarshal([]byte(d.Body), &result)
				if err != nil {
					fmt.Printf("Error unmarshalling json: %v\n", err)
					continue
				}

				userHexKeyUUID, ok := result["pubKeyUUID"].(string)
				if !ok {
					fmt.Println("Invalid or missing content")
					continue
				}

				parts := strings.Split(userHexKeyUUID, ":")
				if len(parts) != 2 {
					continue
				}

				userHexKey := parts[0]
				uuid := parts[1]
				// I have abstracted this functionality out, call that
				s.pubKeyUUIDLock.Lock()
				existingUUID, exists := s.pubKeyUUID[userHexKey]
				if exists && existingUUID == uuid {
					s.pubKeyUUIDLock.Unlock()
					fmt.Printf("Mapping already exists for Pubkey: %s. Skipping processing\n", userHexKeyUUID)
					continue
				}

				s.pubKeyUUID[userHexKey] = uuid
				s.pubKeyUUIDLock.Unlock()

				newNote.Content, ok = result["content"].(string)
				if !ok {
					fmt.Println("Invalid or missing content")
					continue
				}
				fmt.Printf("type of kind: %v\n", reflect.TypeOf(result["kind"]))
				newNote.PubHexKey, ok = result["pubHexKey"].(string)
				if !ok {
					fmt.Println("")
					continue
				}

				newNote.PrivHexKey, ok = result["privHexKey"].(string)
				if !ok {
					fmt.Println("Invalid or missing private hex key")
					continue
				}

				newNote.Kind, ok = result["kind"].(float64)
				if !ok {
					fmt.Println("Invalid or missing event kind")
					continue
				}

				s.createNote(newNote)
			}
		}
	}()

	<-forever
}

func (s *Service) StartSearchQueue() {
	forever := make(chan struct{})

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		fmt.Println("2. Failed to connect to RabbitMQ", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		fmt.Println("4. Failed to open a channel")
	}

	queue, err := channel.QueueDeclare(
		"search",
		false,
		false,
		false,
		false,
		nil,
	)

	go func() {
		fmt.Println("7. Starting consumer goroutine")
		for {
			fmt.Println("8. Waiting for messages")
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
				fmt.Println("9. Failed to register a consumer", err)
			}

			for d := range msgs {
				fmt.Println("11. Received message:", string(d.Body))
				searchUUID := string(d.Body)
				parts := strings.Split(searchUUID, ":")
				if len(parts) < 2 {
					continue
				}

				search := parts[0]
				uuid := parts[1]

				var pubkey *string
				if len(parts) > 2 && parts[2] != "" {
					pubkeyStr := parts[2]
					pubkey = &pubkeyStr
				}

				s.searchUUIDLock.Lock()
				existingUUID, exists := s.searchUUID[search]
				if exists && existingUUID == uuid {
					s.searchUUIDLock.Unlock()
					fmt.Printf("Mapping already exists for search: %s. Skipping processing\n", search)
					continue
				}

				s.searchUUID[search] = uuid
				s.searchUUIDLock.Unlock()

				s.retrieveSearch(search, uuid, pubkey)
			}
		}
	}()
	<-forever
}
