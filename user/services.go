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
	"github.com/DerekBelloni/go-socket-server/queue"
	"github.com/DerekBelloni/go-socket-server/relay"
	"github.com/DerekBelloni/go-socket-server/search"
	"github.com/DerekBelloni/go-socket-server/store"
)

type Service struct {
	relayConnection *relay.RelayConnection
	searchTracker   core.SearchTracker
	relayUrls       []string
	pubKeyUUID      map[string]string
	pubKeyUUIDLock  sync.RWMutex
}

func NewService(relayConnection *relay.RelayConnection, relayUrls []string, searchTracker *search.SearchTrackerImpl) *Service {
	return &Service{
		relayConnection: relayConnection,
		relayUrls:       relayUrls,
		searchTracker:   searchTracker,
		pubKeyUUID:      make(map[string]string),
	}
}

// Probably need to incorporate contexts
// Especially now the I know I get the subscription id back on events from the relays I think it is a great idea

// func createUserContext(userHexKey string) (context.Context, context.CancelFunc) {
// 	return context.WithCancel(context.WithValue(context.Background(), "userPubKey", userHexKey))
// }

func (s *Service) checkAndUpdateUUID(userHexKey, uuid string, search string) bool {
	s.pubKeyUUIDLock.Lock()
	defer s.pubKeyUUIDLock.Unlock()

	var existingUUID string
	var exists bool

	if userHexKey != "" {
		existingUUID, exists = s.pubKeyUUID[userHexKey]
	} else {
		existingUUID, exists = s.pubKeyUUID[search]
	}

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

	queueName := "user_pub_key"

	msgs, channel, conn, err := queue.ConsumeQueue(queueName)
	if err != nil {
		fmt.Printf("Error consuming messages from the %v queue, %v\n", queueName, err)
		return
	}

	defer channel.Close()
	defer conn.Close()

	go func() {
		for {
			for d := range msgs {
				userHexKeyUUID := string(d.Body)
				parts := strings.Split(userHexKeyUUID, ":")
				if len(parts) != 2 {
					continue
				}

				userHexKey := parts[0]
				uuid := parts[1]

				if !s.checkAndUpdateUUID(userHexKey, uuid, "") {
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

	msgs, channel, conn, err := queue.ConsumeQueue(queueName)

	if err != nil {
		fmt.Printf("Error consuming messages from the %v queue, %v\n", queueName, err)
		return
	}

	defer channel.Close()
	defer conn.Close()

	go func() {
		for {
			for d := range msgs {
				userHexKeyUUID := string(d.Body)
				parts := strings.Split(userHexKeyUUID, ":")
				if len(parts) != 2 {
					continue
				}

				userHexKey := parts[0]
				uuid := parts[1]

				if !s.checkAndUpdateUUID(userHexKey, uuid, "") {
					continue
				}

				s.followsMetadata(userHexKey)
			}
		}
	}()
	log.Printf("[*] Waiting for follows list messages. To exit press CTRL+C")
	<-forever
}

func (s *Service) StartCreateNoteQueue() {
	forever := make(chan struct{})
	queueName := "new_note"
	msgs, channel, conn, err := queue.ConsumeQueue(queueName)
	if err != nil {
		fmt.Printf("Error consuming messages from the %v queue, %v\n", queueName, err)
		return
	}

	defer channel.Close()
	defer conn.Close()

	go func() {
		for {
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

				if !s.checkAndUpdateUUID(userHexKey, uuid, "") {
					continue
				}

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
	queueName := "search"
	msgs, channel, conn, err := queue.ConsumeQueue(queueName)
	if err != nil {
		fmt.Printf("Error consuming message from the %v queue, %v\n", queueName, err)
		return
	}

	defer channel.Close()
	defer conn.Close()

	go func() {
		for {
			for d := range msgs {
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

				if !s.checkAndUpdateUUID(*pubkey, uuid, search) {
					fmt.Printf("Mapping already exists for search: %s. Skipping processing\n", search)
					continue
				}

				s.retrieveSearch(search, uuid, pubkey)
			}
		}
	}()
	<-forever
}

func (s *Service) StartFollowsNotesQueue() {
	forever := make(chan struct{})
	queueName := "follow_notes"
	msgs, channel, conn, err := queue.ConsumeQueue(queueName)
	if err != nil {
		fmt.Printf("Error consumingn message from the %v queue, %v\n", queueName, err)
	}

	defer channel.Close()
	defer conn.Close()

	go func() {
		for {
			for d := range msgs {
				userHexKeyUUID := string(d.Body)
				parts := strings.Split(userHexKeyUUID, ":")
				if len(parts) != 2 {
					continue
				}

				userHexKey := parts[0]
				uuid := parts[1]

				if !s.checkAndUpdateUUID(userHexKey, uuid, "") {
					continue
				}

			}
		}
	}()

	<-forever
}
