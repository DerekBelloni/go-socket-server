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
	relayConnection     *relay.RelayConnection
	subscriptionTracker core.SubscriptionTracker
	relayUrls           []string
	pubKeyUUID          map[string]string
	pubKeyUUIDLock      sync.RWMutex
}

func NewService(relayConnection *relay.RelayConnection, relayUrls []string, searchTracker *search.SearchTrackerImpl) *Service {
	return &Service{
		relayConnection:     relayConnection,
		relayUrls:           relayUrls,
		subscriptionTracker: searchTracker,
		pubKeyUUID:          make(map[string]string),
	}
}

// func createUserContext(userHexKey string) (context.Context, context.CancelFunc) {
// 	return context.WithCancel(context.WithValue(context.Background(), "userPubKey", userHexKey))
// }

func (s *Service) checkAndUpdateUUID(userHexKey string, uuid string, search string) bool {
	fmt.Printf("uuid in service layer mapping: %v\n", uuid)
	s.pubKeyUUIDLock.Lock()
	defer s.pubKeyUUIDLock.Unlock()

	var key string
	if userHexKey != "" {
		key = userHexKey
	} else {
		key = search
	}

	// this needs to be updated because if a logged in user does multiple searches it will return false because it will match on their pubkey
	existingUUID, exists := s.pubKeyUUID[key]
	if exists && existingUUID == uuid {
		return false
	}

	s.pubKeyUUID[key] = uuid
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

func (s *Service) followsNotes(userPubKey string, followPubKey string, uuid string) {
	for _, relayUrl := range s.relayUrls {
		go func(relayUrl string) {
			s.relayConnection.GetFollowsNotes(relayUrl, userPubKey, followPubKey, s.subscriptionTracker, uuid)
		}(relayUrl)
	}
}

func (s *Service) retrieveSearch(search string, uuid string, pubkey string) {
	relayUrl := "wss://relay.nostr.band/"
	go func() {
		s.relayConnection.RetrieveSearch(relayUrl, search, s.subscriptionTracker, uuid, pubkey)
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
				fmt.Printf("here we are\n")
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

				var pubkey string
				if len(parts) > 2 && parts[2] != "" {
					pubkey = parts[2]
				}

				if !s.checkAndUpdateUUID(pubkey, uuid, search) {
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
		fmt.Printf("Error conusming message from the %v queue, %v\n", queueName, err)
	}

	defer channel.Close()
	defer conn.Close()

	go func() {
		for {
			for d := range msgs {
				userHexKeyUUID := string(d.Body)
				parts := strings.Split(userHexKeyUUID, ":")
				if len(parts) != 3 {
					continue
				}

				userPubkey := parts[0]
				followPubkey := parts[1]
				uuid := parts[2]

				if !s.checkAndUpdateUUID(userPubkey, uuid, "") {
					continue
				}

				fmt.Printf("user pubkey: %v\n, follows pubkey: %v\n", userPubkey, followPubkey)

				s.followsNotes(userPubkey, followPubkey, uuid)
			}
		}
	}()

	<-forever
}
