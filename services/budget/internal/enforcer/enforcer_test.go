package enforcer_test

import (
	"context"
	"testing"

	"github.com/marginpilot/budget/internal/enforcer"
)

type fakeStore struct {
	spend    int64
	limit    int64
	rateOK   bool
	setLimit int64
}

func (f *fakeStore) Spend(context.Context, string, string) (int64, error) { return f.spend, nil }
func (f *fakeStore) Limit(context.Context, string, string) (int64, error) { return f.limit, nil }
func (f *fakeStore) AllowRate(context.Context, string, string) (bool, error) {
	return f.rateOK, nil
}
func (f *fakeStore) SetLimit(_ context.Context, _, _ string, m int64) error {
	f.setLimit = m
	return nil
}

func TestAllow(t *testing.T) {
	tests := []struct {
		name       string
		store      fakeStore
		defLimit   int64
		wantAllow  bool
		wantReason string
	}{
		{"within budget", fakeStore{spend: 1_000_000, limit: 5_000_000, rateOK: true}, 0, true, ""},
		{"over budget", fakeStore{spend: 6_000_000, limit: 5_000_000, rateOK: true}, 0, false, "budget"},
		{"rate limited", fakeStore{spend: 0, limit: 5_000_000, rateOK: false}, 0, false, "rate_limit"},
		{"uses default limit", fakeStore{spend: 500_000, limit: 0, rateOK: true}, 1_000_000, true, ""},
		{"default exceeded", fakeStore{spend: 2_000_000, limit: 0, rateOK: true}, 1_000_000, false, "budget"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := enforcer.New(&tc.store, tc.defLimit)
			d, err := e.Allow(context.Background(), "t", "c")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.Allowed != tc.wantAllow || d.Reason != tc.wantReason {
				t.Errorf("got {allowed=%v reason=%q}, want {allowed=%v reason=%q}",
					d.Allowed, d.Reason, tc.wantAllow, tc.wantReason)
			}
		})
	}
}
