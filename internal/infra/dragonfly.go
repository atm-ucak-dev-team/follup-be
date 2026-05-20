package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/redis/go-redis/v9"
)

// NewDragonflyClient creates a new DragonflyDB client
func NewDragonflyClient(cfg *domain.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.DragonflyAddr,
		Password: cfg.DragonflyPassword,
		DB:       cfg.DragonflyDB,
	})
}

// Ping checks if the DragonflyDB connection is alive
func Ping(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}

// Close gracefully closes the DragonflyDB connection
func Close(client *redis.Client) error {
	return client.Close()
}

// JSONSet stores a JSON-serialized value at the given key
func JSONSet(ctx context.Context, client *redis.Client, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return client.Set(ctx, key, data, 0).Err()
}

// JSONGet retrieves and JSON-deserializes a value from the given key
func JSONGet(ctx context.Context, client *redis.Client, key string, dest interface{}) error {
	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key '%s' not found", key)
		}
		return fmt.Errorf("failed to get key '%s': %w", key, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// JSONSetWithTTL stores a JSON-serialized value with a time-to-live
func JSONSetWithTTL(ctx context.Context, client *redis.Client, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return client.Set(ctx, key, data, ttl).Err()
}

// JSONSetNX stores a JSON-serialized value only if the key doesn't exist
func JSONSetNX(ctx context.Context, client *redis.Client, key string, value interface{}) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	return client.SetNX(ctx, key, data, 0).Result()
}

// JSONDelete removes a key from DragonflyDB
func JSONDelete(ctx context.Context, client *redis.Client, key string) error {
	return client.Del(ctx, key).Err()
}

// JSONExists checks if a key exists in DragonflyDB
func JSONExists(ctx context.Context, client *redis.Client, key string) (bool, error) {
	count, err := client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return count > 0, nil
}

// JSONGetMultiple retrieves multiple keys and returns them as a map
func JSONGetMultiple(ctx context.Context, client *redis.Client, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Use pipeline for better performance
	pipe := client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	result := make(map[string][]byte)
	for i, cmd := range cmds {
		data, err := cmd.Bytes()
		if err != nil {
			if err == redis.Nil {
				continue // Key doesn't exist, skip it
			}
			return nil, fmt.Errorf("failed to get key '%s': %w", keys[i], err)
		}
		result[keys[i]] = data
	}

	return result, nil
}

// JSONScan searches for keys matching a pattern and returns their values
func JSONScan(ctx context.Context, client *redis.Client, pattern string, dest interface{}, scanCount int64) ([]string, error) {
	var keys []string
	var cursor uint64

	for {
		var batch []string
		var err error

		batch, cursor, err = client.Scan(ctx, cursor, pattern, scanCount).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan keys: %w", err)
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}
