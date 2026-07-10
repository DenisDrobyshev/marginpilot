// Package catalog is the versioned price catalog: model -> micros of USD per
// million tokens, stored in PostgreSQL. Cost is computed with integer math to
// avoid floating-point drift on money.
package catalog

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the PostgreSQL-backed catalog.
type Store struct{ pool *pgxpool.Pool }

// New wraps a pgx pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Migrate creates the catalog table if absent.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS model_prices (
    model                  TEXT PRIMARY KEY,
    input_micros_per_mtok  BIGINT NOT NULL,
    output_micros_per_mtok BIGINT NOT NULL,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
	return err
}

var seed = []struct {
	model   string
	in, out int64
}{
	{"gpt-4o-mini", 150_000, 600_000},
	{"gpt-4o", 2_500_000, 10_000_000},
	{"claude-haiku-4-5", 800_000, 4_000_000},
	{"echo-1", 100_000, 100_000},
	{"default", 1_000_000, 3_000_000},
}

// Seed inserts the initial catalog if rows are missing.
func (s *Store) Seed(ctx context.Context) error {
	for _, p := range seed {
		if _, err := s.pool.Exec(ctx,
			`INSERT INTO model_prices (model, input_micros_per_mtok, output_micros_per_mtok)
			 VALUES ($1, $2, $3) ON CONFLICT (model) DO NOTHING`,
			p.model, p.in, p.out); err != nil {
			return err
		}
	}
	return nil
}

// Price returns the cost of a call in micros, using the "default" row for
// unknown models.
func (s *Store) Price(ctx context.Context, model string, inTok, outTok int64) (int64, error) {
	inPM, outPM, err := s.rates(ctx, model)
	if err != nil {
		return 0, err
	}
	return (inTok*inPM + outTok*outPM) / 1_000_000, nil
}

func (s *Store) rates(ctx context.Context, model string) (int64, int64, error) {
	var in, out int64
	err := s.pool.QueryRow(ctx,
		`SELECT input_micros_per_mtok, output_micros_per_mtok FROM model_prices WHERE model = $1`,
		model).Scan(&in, &out)
	if errors.Is(err, pgx.ErrNoRows) {
		if model == "default" {
			return 0, 0, errors.New("catalog: default price row missing")
		}
		return s.rates(ctx, "default")
	}
	if err != nil {
		return 0, 0, err
	}
	return in, out, nil
}
