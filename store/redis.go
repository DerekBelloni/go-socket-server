package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient() *RedisClient {
	return &RedisClient{
		client: redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}),
		ctx: context.Background(),
	}
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

func (r *RedisClient) HandleFollowListPubKeys(userHexKey string) {
	// client := redis.NewClient(&redis.Options{
	// 	Addr:     "localhost:6379",
	// 	Password: "",
	// 	DB:       0,
	// })

	// ctx := context.Background()

	redisKey := userHexKey + ":" + "follows"
	if userHexKey != "" {
		res, err := r.client.Get(r.ctx, redisKey).Result()

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
