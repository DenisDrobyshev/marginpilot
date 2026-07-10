# notifier (planned, Go)

Delivers alerts: budget thresholds crossed, spend anomalies detected, margin below
target. Fan-out to email, Slack and webhooks.

- consumes budget/analytics events
- per-tenant channel config and dedup/throttling
- at-least-once delivery with retries

Status: Phase 2–4 on the roadmap.
