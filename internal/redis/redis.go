package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func HandleRedis(relayNotesJSON []byte, relayUrl string) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	err := client.Set(ctx, relayUrl, relayNotesJSON, 0).Err()
	if err != nil {
		fmt.Println("Error setting relays data to redis: ", err)
		return
	}
}
