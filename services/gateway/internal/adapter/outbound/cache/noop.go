// Package cache contains outbound adapters implementing port.Cache.
package cache

import (
	"context"

	"github.com/marginpilot/gateway/internal/domain"
)

// Noop is a disabled cache: always a miss. Used when no Redis is configured.
type Noop struct{}

// NewNoop constructs the no-op cache.
func NewNoop() Noop { return Noop{} }

// Get always misses.
func (Noop) Get(context.Context, string) (domain.ChatResponse, bool, error) {
	return domain.ChatResponse{}, false, nil
}

// Set is a no-op.
func (Noop) Set(context.Context, string, domain.ChatResponse) error { return nil }
