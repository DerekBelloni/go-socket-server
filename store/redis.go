package store

import (
	"context"
	"encoding/json"
	"fmt"

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

func ExtractPubKeys(tags []interface{}) (string, bool) {
	var pubKeys []string

	tagSlice, ok := tag.([]interface{})
	if !ok || len(tagSlice) < 2 {
		return "", false
	}
	pubKey, ok := tagSlice[1].(string)
	if !ok {
		return "", false
	}
	return pubKey, true
}

// func ExtractPubKeys(tag interface{}) (string, bool) {
// 	tagSlice, ok := tag.([]interface{})
// 	if !ok || len(tagSlice) < 2 {
// 		return "", false
// 	}
// 	pubKey, ok := tagSlice[1].(string)
// 	if !ok {
// 		return "", false
// 	}
// 	return pubKey, true
// }

func (r *RedisClient) HandleFollowListPubKeys(userHexKey string) []string {
	redisKey := userHexKey + ":" + "follows"
	// var pubKeys []string

	if userHexKey != "" {
		res, err := r.client.Get(r.ctx, redisKey).Result()

		if err != nil {
			fmt.Printf("Couldn't retrieve follows list from Redis: %v\n", err)
		}

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

		pubKeys, ok := ExtractPubKeys(tags)

		// for _, tag := range tags {
		// 	pubKey, ok := ExtractPubKeys(tag)
		// 	if !ok {
		// 		continue
		// 	}
		// 	pubKeys = append(pubKeys, pubKey)
		// }

		fmt.Printf("pubkeys: %v\n", pubKeys)
	}
	return pubKeys
}
