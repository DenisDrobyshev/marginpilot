// Command rating serves the model price catalog over gRPC, backed by PostgreSQL.
package main

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	ratingv1 "github.com/marginpilot/contracts/gen/rating/v1"
	"github.com/marginpilot/shared/config"
	"github.com/marginpilot/shared/httpx"
	"github.com/marginpilot/shared/logger"

	"github.com/marginpilot/rating/internal/catalog"
	"github.com/marginpilot/rating/internal/grpcserver"
)

func main() {
	log := logger.New("rating")

	dsn := config.Get("POSTGRES_DSN",
		"postgres://marginpilot:marginpilot@localhost:5432/marginpilot?sslmode=disable")
	pool, err := connect(dsn)
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	store := catalog.New(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		log.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	if err := store.Seed(ctx); err != nil {
		log.Error("seed failed", "err", err)
	}

	grpcAddr := config.Get("RATING_GRPC_ADDR", ":9103")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc listen failed", "addr", grpcAddr, "err", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	ratingv1.RegisterRatingServiceServer(gs, grpcserver.New(store))
	go func() {
		log.Info("grpc listening", "addr", grpcAddr)
		if err := gs.Serve(lis); err != nil {
			log.Error("grpc serve failed", "err", err)
		}
	}()

	srv := httpx.New(config.Get("RATING_ADDR", ":8084"), log)
	if err := srv.Run(ctx); err != nil {
		log.Error("server exited with error", "err", err)
	}
	gs.GracefulStop()
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
