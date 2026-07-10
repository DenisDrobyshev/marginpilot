package publisher

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"github.com/marginpilot/shared/events"
)

// Kafka publishes usage events to the backbone. It keys messages by
// tenant:customer so a customer's events keep per-partition order.
type Kafka struct{ w *kafka.Writer }

// NewKafka builds a producer for the usage topic.
func NewKafka(brokers []string) *Kafka {
	return &Kafka{w: &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  events.TopicUsage,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
	}}
}

// Publish serialises and writes one event.
func (p *Kafka) Publish(ctx context.Context, e events.UsageEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(e.TenantID + ":" + e.CustomerID),
		Value: b,
	})
}

// Close flushes and closes the writer.
func (p *Kafka) Close() error { return p.w.Close() }
