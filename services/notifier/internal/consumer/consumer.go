// Package consumer reads alert events off the backbone and delivers them: always
// to the structured log, and to a webhook when NOTIFIER_WEBHOOK_URL is set.
package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/marginpilot/shared/events"
)

// Consumer delivers alerts.
type Consumer struct {
	reader  *kafka.Reader
	webhook string
	client  *http.Client
	log     *slog.Logger
}

// New builds a consumer for the alerts topic.
func New(brokers []string, webhook string, log *slog.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   events.TopicAlerts,
		GroupID: "notifier",
	})
	return &Consumer{
		reader:  r,
		webhook: webhook,
		client:  &http.Client{Timeout: 5 * time.Second},
		log:     log,
	}
}

// Run consumes until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	c.log.Info("consuming", "topic", events.TopicAlerts, "webhook", c.webhook != "")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.log.Error("read message failed", "err", err)
			continue
		}
		c.handle(ctx, m.Value)
	}
}

func (c *Consumer) handle(ctx context.Context, payload []byte) {
	var a events.AlertEvent
	if err := json.Unmarshal(payload, &a); err != nil {
		c.log.Error("bad alert payload", "err", err)
		return
	}

	c.log.Info("alert",
		"type", a.Type, "severity", a.Severity,
		"tenant", a.TenantID, "customer", a.CustomerID, "message", a.Message)

	if c.webhook == "" {
		return
	}
	if err := c.postWebhook(ctx, payload); err != nil {
		c.log.Error("webhook delivery failed", "err", err)
	}
}

func (c *Consumer) postWebhook(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhook, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New("webhook returned " + resp.Status)
	}
	return nil
}

// Close stops the reader.
func (c *Consumer) Close() error { return c.reader.Close() }
