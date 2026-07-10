package pricing_test

import (
	"testing"

	"github.com/marginpilot/metering/internal/pricing"
)

func TestCost(t *testing.T) {
	// 1000 in * 0.15 + 1000 out * 0.60 = 150 + 600 = 750 micros.
	if got := pricing.Cost("gpt-4o-mini", 1000, 1000); got != 750 {
		t.Errorf("gpt-4o-mini cost = %d, want 750", got)
	}
	// Unknown model uses the fallback: 1000*1.0 + 1000*3.0 = 4000 micros.
	if got := pricing.Cost("mystery-model", 1000, 1000); got != 4000 {
		t.Errorf("fallback cost = %d, want 4000", got)
	}
}
