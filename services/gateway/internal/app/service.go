// Package app is the gateway application core. It orchestrates the outbound
// ports and contains the business rules; it imports no transport or driver code.
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/marginpilot/gateway/internal/domain"
	"github.com/marginpilot/gateway/internal/port"
	"github.com/marginpilot/shared/events"
)

// ErrBudgetExceeded is returned when the caller is over their spend limit.
var ErrBudgetExceeded = errors.New("budget exceeded")

// Caller identifies who is behind a request. The inbound adapter resolves it
// from the virtual API key (via the Identity service) before calling Proxy.
type Caller struct {
	TenantID   string
	CustomerID string
	Feature    string
}

// Service wires the three outbound ports into the request path.
type Service struct {
	provider  port.LLMProvider
	publisher port.UsagePublisher
	budget    port.BudgetChecker
	log       *slog.Logger
}

// New constructs the core from its dependencies.
func New(p port.LLMProvider, pub port.UsagePublisher, b port.BudgetChecker, log *slog.Logger) *Service {
	return &Service{provider: p, publisher: pub, budget: b, log: log}
}

// Proxy is the full request path: enforce budget, call the provider, then emit
// a usage event. Emission is best-effort — a backbone hiccup must never fail a
// user request, so a publish error is logged and swallowed.
func (s *Service) Proxy(ctx context.Context, c Caller, req domain.ChatRequest) (domain.ChatResponse, error) {
	allowed, err := s.budget.Allow(ctx, c.TenantID, c.CustomerID)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	if !allowed {
		return domain.ChatResponse{}, ErrBudgetExceeded
	}

	start := time.Now()
	resp, err := s.provider.Complete(ctx, req)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	latency := time.Since(start)

	evt := events.UsageEvent{
		EventID:      newID(),
		TenantID:     c.TenantID,
		CustomerID:   c.CustomerID,
		Feature:      c.Feature,
		Provider:     s.provider.Name(),
		Model:        resp.Model,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		LatencyMS:    latency.Milliseconds(),
		OccurredAt:   time.Now().UTC(),
	}
	if err := s.publisher.Publish(ctx, evt); err != nil {
		s.log.Error("publish usage event failed", "event_id", evt.EventID, "err", err)
	}
	return resp, nil
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
