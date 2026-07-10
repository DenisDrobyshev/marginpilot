// Package events defines the cross-service event contracts carried on the
// Kafka/Redpanda backbone. Keeping them in the shared module makes the
// producer (gateway) and consumers (metering, billing) agree by construction.
package events

import "time"

// TopicUsage is the backbone topic every billable LLM call lands on.
const TopicUsage = "usage.events.v1"

// TopicAlerts carries operational alerts (budget breaches, spend anomalies) to
// the notifier for fan-out.
const TopicAlerts = "alerts.v1"

// AlertEvent is emitted when something needs a human's attention.
type AlertEvent struct {
	AlertID    string    `json:"alert_id"`
	Type       string    `json:"type"`     // "budget", "rate_limit", "anomaly"
	Severity   string    `json:"severity"` // "info", "warning", "critical"
	TenantID   string    `json:"tenant_id"`
	CustomerID string    `json:"customer_id"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurred_at"`
}

// UsageEvent is the canonical record emitted for a single LLM call. It is the
// single source of truth for metering, rating and margin analytics, so it
// carries both the business dimensions (tenant, customer, feature) and the
// technical facts (provider, model, tokens).
type UsageEvent struct {
	EventID      string    `json:"event_id"`
	TenantID     string    `json:"tenant_id"`
	CustomerID   string    `json:"customer_id"`
	Feature      string    `json:"feature"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	LatencyMS    int64     `json:"latency_ms"`
	OccurredAt   time.Time `json:"occurred_at"`
}
