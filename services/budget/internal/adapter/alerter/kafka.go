// Package alerter publishes alert events to the backbone for the notifier.
package alerter

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"github.com/marginpilot/shared/events"
)

// Kafka publishes alert events to the alerts topic.
type Kafka struct{ w *kafka.Writer }

// NewKafka builds an alert producer.
func NewKafka(brokers []string) *Kafka {
	return &Kafka{w: &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  events.TopicAlerts,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
	}}
}

// Publish writes one alert event.
func (k *Kafka) Publish(ctx context.Context, e events.AlertEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return k.w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(e.TenantID + ":" + e.CustomerID),
		Value: b,
	})
}

// Close flushes and closes the writer.
func (k *Kafka) Close() error { return k.w.Close() }
