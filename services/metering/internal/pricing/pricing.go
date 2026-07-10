// Package pricing converts token usage into cost in micros of USD. It is a
// stand-in for the future Rating service: a small built-in catalog with a
// sensible default so metering can price events today.
package pricing

// price is micros of USD per single token.
type price struct{ in, out float64 }

// catalog maps model -> micros per token.
// e.g. $0.15 per 1M input tokens = 0.15 micros/token.
var catalog = map[string]price{
	"gpt-4o-mini":      {in: 0.15, out: 0.60},
	"gpt-4o":           {in: 2.50, out: 10.00},
	"claude-haiku-4-5": {in: 0.80, out: 4.00},
	"echo-1":           {in: 0.10, out: 0.10},
}

var fallback = price{in: 1.00, out: 3.00}

// Cost returns the cost of one call in micros of USD (1_000_000 micros = $1).
func Cost(model string, inputTokens, outputTokens int) int64 {
	p, ok := catalog[model]
	if !ok {
		p = fallback
	}
	c := float64(inputTokens)*p.in + float64(outputTokens)*p.out
	return int64(c + 0.5)
}
