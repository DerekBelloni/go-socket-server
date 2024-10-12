package handler

import (
	"context"
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

		fmt.Printf("Follows list for pubkey: %v\n%v\n", userHexKey, res)
	}

}
