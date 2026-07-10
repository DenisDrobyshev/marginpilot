// Command billing aggregates priced usage (ClickHouse) against plan revenue
// (PostgreSQL) into per-customer invoices with margin, and exports usage to the
// billing provider. HTTP only — it is not on the request hot path.
package main

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/marginpilot/shared/config"
	"github.com/marginpilot/shared/httpx"
	"github.com/marginpilot/shared/logger"

	"github.com/marginpilot/billing/internal/cogs"
	"github.com/marginpilot/billing/internal/httpapi"
	"github.com/marginpilot/billing/internal/store"
	"github.com/marginpilot/billing/internal/stripe"
)

func main() {
	log := logger.New("billing")
	ctx := context.Background()

	dsn := config.Get("POSTGRES_DSN",
		"postgres://marginpilot:marginpilot@localhost:5432/marginpilot?sslmode=disable")
	pool, err := connect(dsn)
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	subs := store.New(pool)
	if err := subs.Migrate(ctx); err != nil {
		log.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	if err := subs.Seed(ctx); err != nil {
		log.Error("seed failed", "err", err)
	}

	cogsReader, err := cogs.NewReader(config.Get("CLICKHOUSE_ADDR", "localhost:9000"))
	if err != nil {
		log.Error("clickhouse open failed", "err", err)
		os.Exit(1)
	}

	srv := httpx.New(config.Get("BILLING_ADDR", ":8085"), log)
	httpapi.New(subs, cogsReader, stripe.NewMock(log), log).Register(srv.Mux())

	if err := srv.Run(ctx); err != nil {
		log.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func connect(dsn string) (*pgxpool.Pool, error) {
	var lastErr error
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		pool, err := pgxpool.New(ctx, dsn)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				cancel()
				return pool, nil
			}
			pool.Close()
		}
		cancel()
		lastErr = err
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}
