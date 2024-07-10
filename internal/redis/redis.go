package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func HandleRedis(relayNotesJSON []byte, relayUrl string, finished chan<- string, batchType string) error {
	fmt.Println("in redis")
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	var redisKey string

	if batchType == "trending" {
		redisKey = relayUrl + "-trending"
		fmt.Printf("redis key: %s", redisKey)
	} else {
		redisKey = relayUrl
	}

	// err := client.Set(ctx, relayUrl, relayNotesJSON, 0).Err()
	err := client.Set(ctx, redisKey, relayNotesJSON, 0).Err()
	if err != nil {
		fmt.Println("Error setting relays data to redis: ", err)
		return err
	}

	if batchType == "trending" {
		fmt.Println("batch type: ", batchType)
		finished <- relayUrl
	}
	return nil
}
