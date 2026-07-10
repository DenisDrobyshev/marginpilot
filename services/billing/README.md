# billing (planned, Go)

Aggregates priced usage into invoices and exports usage to the customer's billing
provider (Stripe first).

- period aggregation per customer/plan
- Stripe usage-record export + idempotent reconciliation
- source of truth for "revenue" in the margin calculation

Status: Phase 3 on the roadmap.
