// Package spend defines the Redis key convention for per-customer spend
// counters. Metering (writer) and budget (reader) both import it so they can
// never drift on the key format. Spend is tracked in micros of USD
// (1_000_000 micros = 1 USD) and bucketed per calendar month.
package spend

import (
	"fmt"
	"time"
)

// Period returns the monthly bucket label for t (UTC), e.g. "202607".
func Period(t time.Time) string {
	return t.UTC().Format("200601")
}

// Key returns the Redis key holding the accumulated spend (micros) for a
// customer in the month containing t.
func Key(tenant, customer string, t time.Time) string {
	return fmt.Sprintf("spend:%s:%s:%s", tenant, customer, Period(t))
}
