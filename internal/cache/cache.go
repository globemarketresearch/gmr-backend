package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

var Client *redis.Client
var ctx = context.Background()
var sfGroup singleflight.Group

func Connect(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	_, err := Client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connected successfully")
	return nil
}

func Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, ttl).Err()
}

func Get(key string, dest interface{}) error {
	data, err := Client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func Delete(key string) error {
	return Client.Del(ctx, key).Err()
}

func DeletePattern(pattern string) error {
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = Client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := Client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}

// GetOrSet retrieves data from cache, or executes the provided function if cache miss
// Uses singleflight to prevent cache stampede
func GetOrSet(key string, dest interface{}, ttl time.Duration, fn func() (interface{}, error)) error {
	// Try to get from cache first
	err := Get(key, dest)
	if err == nil {
		return nil
	}

	// Cache miss - use singleflight to prevent stampede
	v, err, _ := sfGroup.Do(key, func() (interface{}, error) {
		// Double-check cache (another goroutine might have filled it)
		err := Get(key, dest)
		if err == nil {
			return dest, nil
		}

		// Execute the function to get fresh data
		data, err := fn()
		if err != nil {
			return nil, err
		}

		// Cache the result
		if err := Set(key, data, ttl); err != nil {
			log.Printf("Failed to cache data for key %s: %v", key, err)
		}

		return data, nil
	})

	if err != nil {
		return err
	}

	// Marshal and unmarshal to populate dest
	jsonData, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, dest)
}

func Close() error {
	if Client == nil {
		return nil
	}
	return Client.Close()
}
