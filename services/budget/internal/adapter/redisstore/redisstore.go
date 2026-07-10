// Package redisstore implements enforcer.Store on Redis: spend counters (shared
// with metering via shared/spend), per-customer limits, and a fixed-window rate
// limiter.
package redisstore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/marginpilot/shared/spend"
)

const (
	rateWindow = time.Minute
	rateMax    = 120 // requests per window per customer
)

// Store is a Redis-backed enforcer.Store.
type Store struct{ rdb *redis.Client }

// New wraps a redis client.
func New(rdb *redis.Client) *Store { return &Store{rdb: rdb} }

// Spend reads the accumulated spend counter metering increments.
func (s *Store) Spend(ctx context.Context, tenant, customer string) (int64, error) {
	return s.getInt(ctx, spend.Key(tenant, customer, time.Now()))
}

// Limit reads the configured limit (0 when unset).
func (s *Store) Limit(ctx context.Context, tenant, customer string) (int64, error) {
	return s.getInt(ctx, limitKey(tenant, customer))
}

// SetLimit persists the configured limit.
func (s *Store) SetLimit(ctx context.Context, tenant, customer string, micros int64) error {
	return s.rdb.Set(ctx, limitKey(tenant, customer), micros, 0).Err()
}

// AllowRate implements a fixed-window counter: INCR a per-window key, set its
// TTL on first hit, and compare against the cap.
func (s *Store) AllowRate(ctx context.Context, tenant, customer string) (bool, error) {
	bucket := time.Now().Unix() / int64(rateWindow.Seconds())
	key := fmt.Sprintf("rl:%s:%s:%d", tenant, customer, bucket)
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		_ = s.rdb.Expire(ctx, key, rateWindow).Err()
	}
	return n <= rateMax, nil
}

func (s *Store) getInt(ctx context.Context, key string) (int64, error) {
	v, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(v, 10, 64)
}

func limitKey(tenant, customer string) string {
	return fmt.Sprintf("limit:%s:%s", tenant, customer)
}
