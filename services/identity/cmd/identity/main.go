// Command identity resolves virtual API keys into tenant/customer identities.
// It exposes a gRPC IdentityService backed by PostgreSQL, plus a health endpoint.
package main

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	identityv1 "github.com/marginpilot/contracts/gen/identity/v1"
	"github.com/marginpilot/shared/config"
	"github.com/marginpilot/shared/httpx"
	"github.com/marginpilot/shared/logger"

	"github.com/marginpilot/identity/internal/adapter/grpcserver"
	"github.com/marginpilot/identity/internal/adapter/pgstore"
)

func main() {
	log := logger.New("identity")

	dsn := config.Get("POSTGRES_DSN",
		"postgres://marginpilot:marginpilot@localhost:5432/marginpilot?sslmode=disable")

	pool, err := connect(dsn)
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	store := pgstore.New(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		log.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	// Seed demo keys so the stack is usable end to end. sk-demo -> demo-tenant,
	// sk-north -> a second tenant (northwind) for the dashboard's tenant switcher.
	for _, s := range []struct{ key, tenant, customer string }{
		{"sk-demo", "demo-tenant", "acme-inc"},
		{"sk-north", "northwind", "north-sales"},
	} {
		if err := store.Seed(ctx, s.key, s.tenant, s.customer, "default"); err != nil {
			log.Error("seed failed", "key", s.key, "err", err)
		}
	}

	grpcAddr := config.Get("IDENTITY_GRPC_ADDR", ":9102")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc listen failed", "addr", grpcAddr, "err", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(gs, grpcserver.New(store))
	go func() {
		log.Info("grpc listening", "addr", grpcAddr)
		if err := gs.Serve(lis); err != nil {
			log.Error("grpc serve failed", "err", err)
		}
	}()

	srv := httpx.New(config.Get("IDENTITY_ADDR", ":8083"), log)
	if err := srv.Run(ctx); err != nil {
		log.Error("server exited with error", "err", err)
	}
	gs.GracefulStop()
}

// connect retries the initial connection so the service tolerates Postgres
// still starting up in docker-compose.
func connect(dsn string) (*pgxpool.Pool, error) {
	var lastErr error
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		pool, err := pgxpool.New(ctx, dsn)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
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
