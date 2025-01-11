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

func ExtractPubKeys(tags []interface{}) ([]string, bool) {
	if len(tags) == 0 {
		return nil, false
	}

	pubKeys := make([]string, 0, len(tags))

	for _, tag := range tags {
		tagSlice, ok := tag.([]interface{})
		if !ok || len(tagSlice) < 2 || tagSlice[0] != "p" {
			continue
		}
		pubKey, ok := tagSlice[1].(string)
		if !ok || pubKey == "" {
			continue
		}

		pubKeys = append(pubKeys, pubKey)
	}

	if len(pubKeys) == 0 {
		return nil, false
	}

	return pubKeys, true
}

func (r *RedisClient) HandleFollowListPubKeys(userHexKey string) []string {
	redisKey := userHexKey + ":" + "follows"

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
		if !ok {
			fmt.Printf("Could not extract pubkeys from tags")
		}

		return pubKeys
	}
	return nil
}
