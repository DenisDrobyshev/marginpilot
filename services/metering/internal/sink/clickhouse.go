// Package sink persists priced usage rows to ClickHouse for OLAP and margin
// analytics.
package sink

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/marginpilot/shared/events"
)

// ClickHouse is the analytics sink.
type ClickHouse struct{ conn driver.Conn }

// NewClickHouse opens a (lazy) connection to ClickHouse over the native protocol.
func NewClickHouse(addr string) (*ClickHouse, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{Database: "default"},
	})
	if err != nil {
		return nil, err
	}
	return &ClickHouse{conn: conn}, nil
}

// EnsureSchema creates the usage_events table if it does not exist.
func (c *ClickHouse) EnsureSchema(ctx context.Context) error {
	return c.conn.Exec(ctx, `
CREATE TABLE IF NOT EXISTS usage_events (
    event_id      String,
    tenant_id     String,
    customer_id   String,
    feature       String,
    provider      String,
    model         String,
    input_tokens  UInt32,
    output_tokens UInt32,
    cost_micros   UInt64,
    latency_ms    UInt32,
    occurred_at   DateTime64(3, 'UTC')
) ENGINE = MergeTree
ORDER BY (tenant_id, customer_id, occurred_at)`)
}

// Insert writes one priced usage row.
func (c *ClickHouse) Insert(ctx context.Context, e events.UsageEvent, costMicros int64) error {
	batch, err := c.conn.PrepareBatch(ctx, "INSERT INTO usage_events")
	if err != nil {
		return err
	}
	if err := batch.Append(
		e.EventID, e.TenantID, e.CustomerID, e.Feature, e.Provider, e.Model,
		uint32(e.InputTokens), uint32(e.OutputTokens), uint64(costMicros),
		uint32(e.LatencyMS), e.OccurredAt,
	); err != nil {
		return err
	}
	return batch.Send()
}

// Ping verifies connectivity.
func (c *ClickHouse) Ping(ctx context.Context) error { return c.conn.Ping(ctx) }

// Close releases the connection.
func (c *ClickHouse) Close() error { return c.conn.Close() }
