// Package store holds the revenue side of the margin equation: customer
// subscriptions (plan + monthly revenue) in PostgreSQL. Billing owns this table;
// analytics reads it.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Subscription is a customer's plan and monthly revenue (micros of USD).
type Subscription struct {
	CustomerID    string
	Plan          string
	RevenueMicros int64
}

// Store is the PostgreSQL-backed subscription store.
type Store struct{ pool *pgxpool.Pool }

// New wraps a pgx pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Migrate creates the subscriptions table if absent.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS subscriptions (
    tenant_id      TEXT NOT NULL,
    customer_id    TEXT NOT NULL,
    plan           TEXT NOT NULL,
    revenue_micros BIGINT NOT NULL,
    PRIMARY KEY (tenant_id, customer_id)
)`)
	return err
}

// Seed inserts demo subscriptions so margin has a revenue side out of the box.
func (s *Store) Seed(ctx context.Context) error {
	rows := []struct {
		tenant, customer, plan string
		revenue                int64
	}{
		{"demo-tenant", "acme-inc", "pro", 99_000_000},         // $99/mo
		{"demo-tenant", "globex", "starter", 29_000_000},       // $29/mo, no usage yet
		{"northwind", "north-sales", "scale", 199_000_000},     // $199/mo
		{"northwind", "north-support", "starter", 49_000_000},  // $49/mo
	}
	for _, r := range rows {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO subscriptions (tenant_id, customer_id, plan, revenue_micros)
			 VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, customer_id) DO NOTHING`,
			r.tenant, r.customer, r.plan, r.revenue); err != nil {
			return err
		}
	}
	return nil
}

// List returns all subscriptions for a tenant.
func (s *Store) List(ctx context.Context, tenant string) ([]Subscription, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT customer_id, plan, revenue_micros FROM subscriptions WHERE tenant_id = $1 ORDER BY customer_id`,
		tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.CustomerID, &s.Plan, &s.RevenueMicros); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Get returns one subscription.
func (s *Store) Get(ctx context.Context, tenant, customer string) (Subscription, bool, error) {
	var sub Subscription
	err := s.pool.QueryRow(ctx,
		`SELECT customer_id, plan, revenue_micros FROM subscriptions WHERE tenant_id = $1 AND customer_id = $2`,
		tenant, customer).Scan(&sub.CustomerID, &sub.Plan, &sub.RevenueMicros)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, false, nil
	}
	if err != nil {
		return Subscription{}, false, err
	}
	return sub, true, nil
}
