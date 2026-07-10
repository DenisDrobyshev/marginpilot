package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/marginpilot/gateway/internal/domain"
)

const (
	ttl    = time.Hour
	prefix = "cache:resp:"
)

// Redis caches responses in Redis keyed by request fingerprint.
type Redis struct{ rdb *redis.Client }

// NewRedis builds a Redis-backed cache.
func NewRedis(addr string) *Redis {
	return &Redis{rdb: redis.NewClient(&redis.Options{Addr: addr})}
}

// Get returns the cached response for key, if present.
func (c *Redis) Get(ctx context.Context, key string) (domain.ChatResponse, bool, error) {
	b, err := c.rdb.Get(ctx, prefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.ChatResponse{}, false, nil
	}
	if err != nil {
		return domain.ChatResponse{}, false, err
	}
	var resp domain.ChatResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return domain.ChatResponse{}, false, err
	}
	return resp, true, nil
}

// Set stores a response under key with a TTL.
func (c *Redis) Set(ctx context.Context, key string, resp domain.ChatResponse) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, prefix+key, b, ttl).Err()
}

// Close releases the client.
func (c *Redis) Close() error { return c.rdb.Close() }
