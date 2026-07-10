// Package consumer reads usage events off the backbone, prices them via a
// Pricer, writes the raw row to ClickHouse and increments the shared Redis
// spend counter the budget service reads.
package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"github.com/marginpilot/metering/internal/sink"
	"github.com/marginpilot/shared/events"
	"github.com/marginpilot/shared/spend"
)

const spendTTL = 45 * 24 * time.Hour

// Pricer converts token usage into cost in micros. Implemented by a built-in
// table or a gRPC client to the rating service.
type Pricer interface {
	Cost(ctx context.Context, model string, inputTokens, outputTokens int) (int64, error)
}

// Consumer wires the backbone reader to the sinks.
type Consumer struct {
	reader *kafka.Reader
	ch     *sink.ClickHouse
	rdb    *redis.Client
	pricer Pricer
	log    *slog.Logger
}

// New builds a consumer for the usage topic.
func New(brokers []string, ch *sink.ClickHouse, rdb *redis.Client, pricer Pricer, log *slog.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   events.TopicUsage,
		GroupID: "metering",
	})
	return &Consumer{reader: r, ch: ch, rdb: rdb, pricer: pricer, log: log}
}

// Run consumes until ctx is cancelled. Offsets commit only after a message is
// handled, giving at-least-once delivery; the ClickHouse insert dedupes on
// EventID at the analytics layer.
func (c *Consumer) Run(ctx context.Context) error {
	c.log.Info("consuming", "topic", events.TopicUsage)
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.log.Error("read message failed", "err", err)
			continue
		}
		if err := c.handle(ctx, m.Value); err != nil {
			c.log.Error("handle event failed", "err", err)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, payload []byte) error {
	var e events.UsageEvent
	if err := json.Unmarshal(payload, &e); err != nil {
		return err
	}

	cost, err := c.pricer.Cost(ctx, e.Model, e.InputTokens, e.OutputTokens)
	if err != nil {
		return err
	}

	if err := c.ch.Insert(ctx, e, cost); err != nil {
		return err
	}

	key := spend.Key(e.TenantID, e.CustomerID, e.OccurredAt)
	if err := c.rdb.IncrBy(ctx, key, cost).Err(); err != nil {
		return err
	}
	_ = c.rdb.Expire(ctx, key, spendTTL).Err()

	c.log.Info("metered",
		"customer", e.CustomerID, "model", e.Model, "cost_micros", cost)
	return nil
}

// Close stops the reader.
func (c *Consumer) Close() error { return c.reader.Close() }
