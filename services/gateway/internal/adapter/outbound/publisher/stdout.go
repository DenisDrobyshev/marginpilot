// Package publisher contains outbound adapters implementing port.UsagePublisher.
package publisher

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/marginpilot/shared/events"
)

// Stdout logs usage events as JSON. It stands in for a Kafka/Redpanda producer
// during local development: swapping in the real producer means implementing
// this same port, with no change to the application core.
type Stdout struct{ log *slog.Logger }

// NewStdout constructs the stdout publisher.
func NewStdout(log *slog.Logger) *Stdout { return &Stdout{log: log} }

// Publish serialises the event and writes it to the structured log.
func (p *Stdout) Publish(_ context.Context, e events.UsageEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	p.log.Info("usage_event", "topic", events.TopicUsage, "payload", string(b))
	return nil
}
