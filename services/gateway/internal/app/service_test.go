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

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestProxy_EmitsUsageWhenWithinBudget(t *testing.T) {
	pub := &fakePublisher{}
	svc := app.New(fakeProvider{}, pub, fakeBudget{allow: true}, testLogger())

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
}

func TestProxy_BlocksWhenOverBudget(t *testing.T) {
	pub := &fakePublisher{}
	svc := app.New(fakeProvider{}, pub, fakeBudget{allow: false}, testLogger())

	_, err := svc.Proxy(context.Background(), app.Caller{}, domain.ChatRequest{})
	if !errors.Is(err, app.ErrBudgetExceeded) {
		t.Fatalf("want ErrBudgetExceeded, got %v", err)
	}
	if len(pub.got) != 0 {
		t.Errorf("expected no usage event on budget block, got %d", len(pub.got))
	}
}
