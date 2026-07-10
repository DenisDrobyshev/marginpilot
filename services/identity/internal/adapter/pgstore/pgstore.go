// Package pgstore stores virtual API keys in PostgreSQL. Keys are stored only as
// SHA-256 hashes; the plaintext key is never persisted.
package pgstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record is the caller a virtual key resolves to.
type Record struct {
	TenantID   string
	CustomerID string
	Feature    string
}

// Store is a PostgreSQL-backed key store.
type Store struct{ pool *pgxpool.Pool }

// New wraps a pgx pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Migrate creates the schema if absent.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS virtual_keys (
    key_hash    TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    feature     TEXT NOT NULL DEFAULT 'default',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	return err
}

// Seed inserts a key mapping if it does not already exist, so the stack is
// usable out of the box with a known demo key.
func (s *Store) Seed(ctx context.Context, apiKey, tenant, customer, feature string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO virtual_keys (key_hash, tenant_id, customer_id, feature)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (key_hash) DO NOTHING`,
		Hash(apiKey), tenant, customer, feature)
	return err
}

// Resolve looks up the caller behind an API key.
func (s *Store) Resolve(ctx context.Context, apiKey string) (Record, bool, error) {
	var r Record
	err := s.pool.QueryRow(ctx,
		`SELECT tenant_id, customer_id, feature FROM virtual_keys WHERE key_hash = $1`,
		Hash(apiKey)).Scan(&r.TenantID, &r.CustomerID, &r.Feature)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	return r, true, nil
}

// Hash returns the hex SHA-256 of an API key.
func Hash(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}
