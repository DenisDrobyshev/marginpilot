# budget (planned, Go)

Hot-path budget and policy enforcement. Answers the gateway's `Allow(tenant, customer)`
check in a few milliseconds and applies rate limits.

- Redis-backed token buckets + hierarchical budgets (org → team → customer)
- gRPC server implementing the gateway's `BudgetChecker` port
- consumes metering counters to know current spend; emits alerts near thresholds

Status: interface defined in `services/gateway/internal/port`. Implementation is
Phase 2 on the roadmap.
