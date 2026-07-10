package app_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/marginpilot/gateway/internal/app"
	"github.com/marginpilot/gateway/internal/domain"
	"github.com/marginpilot/shared/events"
)

type fakeProvider struct{}

func (fakeProvider) Name() string { return "fake" }
func (fakeProvider) Complete(context.Context, domain.ChatRequest) (domain.ChatResponse, error) {
	return domain.ChatResponse{
		Model: "fake-1",
		Usage: domain.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
	}, nil
}

type fakePublisher struct{ got []events.UsageEvent }

func (f *fakePublisher) Publish(_ context.Context, e events.UsageEvent) error {
	f.got = append(f.got, e)
	return nil
}

type fakeBudget struct{ allow bool }

func (b fakeBudget) Allow(context.Context, string, string) (bool, error) { return b.allow, nil }

// fakeCache can be primed with a hit.
type fakeCache struct {
	hit  bool
	resp domain.ChatResponse
	sets int
}

func (c *fakeCache) Get(context.Context, string) (domain.ChatResponse, bool, error) {
	return c.resp, c.hit, nil
}
func (c *fakeCache) Set(context.Context, string, domain.ChatResponse) error {
	c.sets++
	return nil
}

// fakeGuardrail optionally blocks.
type fakeGuardrail struct{ block bool }

func (g fakeGuardrail) Check(r domain.ChatRequest) (domain.ChatRequest, error) {
	if g.block {
		return domain.ChatRequest{}, app.ErrBlocked
	}
	return r, nil
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestProxy_EmitsUsageWhenWithinBudget(t *testing.T) {
	pub := &fakePublisher{}
	cache := &fakeCache{}
	svc := app.New(fakeProvider{}, pub, fakeBudget{allow: true}, cache, fakeGuardrail{}, testLogger())

	_, err := svc.Proxy(context.Background(),
		app.Caller{TenantID: "t1", CustomerID: "c1", Feature: "chat"}, domain.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("want 1 usage event, got %d", len(pub.got))
	}
	if pub.got[0].OutputTokens != 5 || pub.got[0].TenantID != "t1" {
		t.Errorf("unexpected usage event: %+v", pub.got[0])
	}
	if cache.sets != 1 {
		t.Errorf("want response cached once, got %d", cache.sets)
	}
}

func TestProxy_BlocksWhenOverBudget(t *testing.T) {
	pub := &fakePublisher{}
	svc := app.New(fakeProvider{}, pub, fakeBudget{allow: false}, &fakeCache{}, fakeGuardrail{}, testLogger())

	_, err := svc.Proxy(context.Background(), app.Caller{}, domain.ChatRequest{})
	if !errors.Is(err, app.ErrBudgetExceeded) {
		t.Fatalf("want ErrBudgetExceeded, got %v", err)
	}
	if len(pub.got) != 0 {
		t.Errorf("expected no usage event on budget block, got %d", len(pub.got))
	}
}

func TestProxy_CacheHitSkipsProviderAndBudget(t *testing.T) {
	pub := &fakePublisher{}
	cached := domain.ChatResponse{Model: "cached-1"}
	// Budget denies, but a cache hit must bypass it entirely.
	svc := app.New(fakeProvider{}, pub, fakeBudget{allow: false},
		&fakeCache{hit: true, resp: cached}, fakeGuardrail{}, testLogger())

	resp, err := svc.Proxy(context.Background(), app.Caller{TenantID: "t1"}, domain.ChatRequest{})
	if err != nil {
		t.Fatalf("unexpected error on cache hit: %v", err)
	}
	if resp.Model != "cached-1" {
		t.Errorf("want cached response, got %+v", resp)
	}
	if len(pub.got) != 1 || pub.got[0].Provider != "cache" || pub.got[0].OutputTokens != 0 {
		t.Errorf("cache hit should emit a zero-cost 'cache' usage event, got %+v", pub.got)
	}
}

func TestProxy_GuardrailBlocks(t *testing.T) {
	pub := &fakePublisher{}
	svc := app.New(fakeProvider{}, pub, fakeBudget{allow: true},
		&fakeCache{}, fakeGuardrail{block: true}, testLogger())

	_, err := svc.Proxy(context.Background(), app.Caller{}, domain.ChatRequest{})
	if !errors.Is(err, app.ErrBlocked) {
		t.Fatalf("want ErrBlocked, got %v", err)
	}
	if len(pub.got) != 0 {
		t.Errorf("blocked request must not emit usage, got %d", len(pub.got))
	}
}
