// Package stripe exports usage to the customer's billing provider. The real
// adapter (guarded by STRIPE_API_KEY) would push usage records to Stripe; this
// mock logs what it would send, so the flow is demonstrable without an account.
package stripe

import (
	"context"
	"log/slog"
)

// Exporter is the billing-provider port.
type Exporter interface {
	ExportUsage(ctx context.Context, tenant, customer string, costMicros int64) error
}

// Mock logs instead of calling Stripe.
type Mock struct{ log *slog.Logger }

// NewMock builds the mock exporter.
func NewMock(log *slog.Logger) *Mock { return &Mock{log: log} }

// ExportUsage records what would be pushed to Stripe.
func (m *Mock) ExportUsage(_ context.Context, tenant, customer string, costMicros int64) error {
	m.log.Info("stripe export (mock)",
		"tenant", tenant, "customer", customer, "usage_micros", costMicros,
		"note", "wire a real adapter behind STRIPE_API_KEY to push usage records")
	return nil
}
