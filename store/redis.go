package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

func HandleRedis(relayNotesJSON []byte, relayUrl string, finished chan<- string, batchType string) error {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	var redisKey string

	if batchType == "trending" {
		redisKey = relayUrl + "-trending"
	} else {
		redisKey = relayUrl
	}

	err := client.Set(ctx, redisKey, relayNotesJSON, 0).Err()
	if err != nil {
		fmt.Println("Error setting relays data to redis: ", err)
		return err
	}

	if batchType != "trending" {
		finished <- relayUrl
	}
	return nil
}

func HandleMetaData(userMetadataJSON []byte, finished chan<- string, relayUrl string, pubKeyHex string, conn *websocket.Conn) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	if pubKeyHex != "" {
		err := client.Set(ctx, pubKeyHex, userMetadataJSON, 0).Err()

		if err != nil {
			fmt.Println("Error setting user metadata to redis: ", err)
		}
	}
	conn.Close()
	finished <- relayUrl
}

func HandleFollowListPubKeys(userHexKey string) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	redisKey := userHexKey + ":" + "follows"
	if userHexKey != "" {
		res, err := client.Get(ctx, redisKey).Result()

		if err != nil {
			fmt.Printf("Couldn't retrieve follows list from Redis: %v\n", err)
		}

		fmt.Printf("Follows list for pubkey: %v\n\n%v\n\n", userHexKey, res)

		var followsEvent []interface{}
		err = json.Unmarshal([]byte(res), &followsEvent)
		if err != nil {
			fmt.Printf("Error unmarshalling follows event JSON from Redis")
		}

		if len(followsEvent) < 3 {
			fmt.Printf("Event not long enough")
		}

		content, ok := followsEvent[2].(map[string]interface{})
		if !ok {
			fmt.Printf("Could not extract content from event data")
		}

		tags, ok := content["tags"].([]interface{})
		if !ok {
			fmt.Printf("Could not extract tags from content")
		}

		fmt.Printf("tags: %v\n", tags)

		// content, ok := eventData[2].(map[string]interface{})
		// if !ok {
		// 	fmt.Printf("Could not extract content from event data")
		// }
		// if !ok {
		// 	fmt.Println("No content in metadata follows event in Redis")
		// }
		// fmt.Printf("unmarshalled follows content: %v\n")
	}
}
