package cache

import (
	"context"
	"log"

	"github.com/CLAOJ/claoj/config"
	"github.com/redis/go-redis/v9"
)

var Client *redis.Client
var Ctx = context.Background()

// Connect initialises the Redis client.
func Connect() {
	cfg := config.C.Redis
	Client = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if _, err := Client.Ping(Ctx).Result(); err != nil {
		// The judge queue, live events, sessions and the audit log all live
		// in Redis; with several replicas there is no safe fallback.
		log.Fatalf("cache: Redis at %s unreachable: %v", cfg.Addr, err)
	}
	log.Printf("cache: connected to Redis at %s", cfg.Addr)
}

// Publish posts a message to a Redis pub/sub channel (matching Django's event_poster format).
func Publish(channel string, payload interface{}) error {
	return Client.Publish(Ctx, channel, payload).Err()
}
