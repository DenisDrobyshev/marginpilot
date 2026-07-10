// Package app is the gateway application core. It orchestrates the outbound
// ports and contains the business rules; it imports no transport or driver code.
package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/marginpilot/gateway/internal/domain"
	"github.com/marginpilot/gateway/internal/port"
	"github.com/marginpilot/shared/events"
)

// ErrBudgetExceeded is returned when the caller is over their spend limit.
var ErrBudgetExceeded = errors.New("budget exceeded")

// ErrBlocked is returned when a guardrail rejects the request.
var ErrBlocked = errors.New("request blocked by guardrail")

// Caller identifies who is behind a request. The inbound adapter resolves it
// from the virtual API key (via the Identity service) before calling Proxy.
type Caller struct {
	TenantID   string
	CustomerID string
	Feature    string
}

// Service wires the outbound ports into the request path.
type Service struct {
	provider  port.LLMProvider
	publisher port.UsagePublisher
	budget    port.BudgetChecker
	cache     port.Cache
	guardrail port.Guardrail
	log       *slog.Logger
}

// New constructs the core from its dependencies.
func New(p port.LLMProvider, pub port.UsagePublisher, b port.BudgetChecker, c port.Cache, g port.Guardrail, log *slog.Logger) *Service {
	return &Service{provider: p, publisher: pub, budget: b, cache: c, guardrail: g, log: log}
}

// Proxy is the full request path:
//  1. guardrail — inspect/redact/block the request;
//  2. cache — a hit returns instantly, costs nothing, consumes no budget;
//  3. budget — enforce the customer's spend limit;
//  4. provider — call upstream;
//  5. cache-fill + usage emission (best-effort).
func (s *Service) Proxy(ctx context.Context, c Caller, req domain.ChatRequest) (domain.ChatResponse, error) {
	req, err := s.guardrail.Check(req)
	if err != nil {
		return domain.ChatResponse{}, err
	}

	key := cacheKey(req)
	if resp, ok, err := s.cache.Get(ctx, key); err != nil {
		s.log.Warn("cache get failed", "err", err)
	} else if ok {
		// Cache hit: no provider call, no budget consumed. Record a zero-cost
		// usage event tagged "cache" so metering counts the hit at $0.
		s.emit(ctx, c, resp, "cache", 0, 0, 0)
		return resp, nil
	}

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

	if err := s.cache.Set(ctx, key, resp); err != nil {
		s.log.Warn("cache set failed", "err", err)
	}

	s.emit(ctx, c, resp, s.provider.Name(),
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, latency.Milliseconds())
	return resp, nil
}

// emit publishes a usage event; a publish failure is logged, never fatal.
func (s *Service) emit(ctx context.Context, c Caller, resp domain.ChatResponse, provider string, in, out int, latencyMS int64) {
	evt := events.UsageEvent{
		EventID:      newID(),
		TenantID:     c.TenantID,
		CustomerID:   c.CustomerID,
		Feature:      c.Feature,
		Provider:     provider,
		Model:        resp.Model,
		InputTokens:  in,
		OutputTokens: out,
		LatencyMS:    latencyMS,
		OccurredAt:   time.Now().UTC(),
	}
	if err := s.publisher.Publish(ctx, evt); err != nil {
		s.log.Error("publish usage event failed", "event_id", evt.EventID, "err", err)
	}
}

// cacheKey fingerprints a request by model + normalized messages.
func cacheKey(req domain.ChatRequest) string {
	var b strings.Builder
	b.WriteString(req.Model)
	b.WriteByte('\n')
	for _, m := range req.Messages {
		b.WriteString(m.Role)
		b.WriteByte(':')
		b.WriteString(strings.TrimSpace(m.Content))
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
