// Package pricer contains adapters implementing consumer.Pricer.
package pricer

import (
	"context"

	"github.com/marginpilot/metering/internal/pricing"
)

// Builtin prices from the local table. Used when no rating service is wired,
// so metering works with no extra infrastructure.
type Builtin struct{}

// NewBuiltin constructs the built-in pricer.
func NewBuiltin() Builtin { return Builtin{} }

// Cost prices a call from the built-in catalog.
func (Builtin) Cost(_ context.Context, model string, in, out int) (int64, error) {
	return pricing.Cost(model, in, out), nil
}
