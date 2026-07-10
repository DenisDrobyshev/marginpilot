// Package cogs reads AI cost of goods sold (COGS) from ClickHouse — the money
// side that metering accumulated per customer.
package cogs

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Reader queries priced usage from ClickHouse.
type Reader struct{ conn driver.Conn }

// NewReader opens a (lazy) ClickHouse connection.
func NewReader(addr string) (*Reader, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{Database: "default"},
	})
	if err != nil {
		return nil, err
	}
	return &Reader{conn: conn}, nil
}

// ByCustomer returns COGS (micros) per customer for a tenant in the current month.
func (r *Reader) ByCustomer(ctx context.Context, tenant string) (map[string]int64, error) {
	rows, err := r.conn.Query(ctx, `
SELECT customer_id, sum(cost_micros)
FROM usage_events
WHERE tenant_id = ? AND toYYYYMM(occurred_at) = ?
GROUP BY customer_id`, tenant, periodInt(time.Now()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int64)
	for rows.Next() {
		var customer string
		var micros uint64
		if err := rows.Scan(&customer, &micros); err != nil {
			return nil, err
		}
		out[customer] = int64(micros)
	}
	return out, rows.Err()
}

// Customer returns COGS (micros) for one customer in the current month.
func (r *Reader) Customer(ctx context.Context, tenant, customer string) (int64, error) {
	var micros uint64
	err := r.conn.QueryRow(ctx, `
SELECT sum(cost_micros)
FROM usage_events
WHERE tenant_id = ? AND customer_id = ? AND toYYYYMM(occurred_at) = ?`,
		tenant, customer, periodInt(time.Now())).Scan(&micros)
	if err != nil {
		return 0, err
	}
	return int64(micros), nil
}

func periodInt(t time.Time) uint32 {
	y, m, _ := t.UTC().Date()
	return uint32(y*100 + int(m))
}
