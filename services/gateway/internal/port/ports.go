// Package port declares the outbound interfaces (driven ports) the gateway
// core depends on. Adapters in internal/adapter/outbound implement them, so
// the core can be tested with fakes and infrastructure can be swapped freely.
package port

import (
	"context"

	"github.com/marginpilot/gateway/internal/domain"
	"github.com/marginpilot/shared/events"
)

// LLMProvider is an upstream model provider (OpenAI, Anthropic, a local model).
type LLMProvider interface {
	Name() string
	Complete(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error)
}

// UsagePublisher emits usage events to the backbone for metering and billing.
type UsagePublisher interface {
	Publish(ctx context.Context, e events.UsageEvent) error
}

// BudgetChecker answers, on the hot path, whether a customer may spend more.
// The real implementation is Redis-backed and must return within a few ms.
type BudgetChecker interface {
	Allow(ctx context.Context, tenantID, customerID string) (bool, error)
}
