# rating (planned, Go)

Turns tokens into money. Maintains a versioned price catalog per provider/model and
converts each usage event into cost (AI COGS).

- provider price catalogs in PostgreSQL, effective-dated
- consumes usage events, writes priced rows to ClickHouse
- exposes cost lookups to Billing and Margin Analytics

Status: Phase 3 on the roadmap.
