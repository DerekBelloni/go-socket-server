package user

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/DerekBelloni/go-socket-server/relay"
	"github.com/DerekBelloni/go-socket-server/store"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Service struct {
	relayConnection *relay.RelayConnection
	relayUrls       []string
	pubKeyUUID      map[string]string
	pubKeyUUIDLock  sync.RWMutex
}

func NewService(relayConnection *relay.RelayConnection, relayUrls []string) *Service {
	return &Service{
		relayConnection: relayConnection,
		relayUrls:       relayUrls,
		pubKeyUUID:      make(map[string]string),
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

func (s *Service) followsMetadata(userHexKey string) {
	redisClient := store.NewRedisClient()
	redisClient.HandleFollowListPubKeys(userHexKey)
	// for _, relayUrl := range relyaUrls {
	// 	go func(relayUrl string) {
	// 		s.relayConnection.GetFollowListMetadata(relayUrl, userHexKey, pubKeys)
	// 	}(relayUrl)
	// }
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

func (s *Service) followList(relayUrls []string, userHexKey string, followsFinished chan<- string) {
	for _, relayUrl := range relayUrls {
		go func(relayUrl string) {
			s.relayConnection.GetFollowList(relayUrl, userHexKey, followsFinished)
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
