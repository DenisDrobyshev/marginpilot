// Command budget is the hot-path enforcement service: a gRPC BudgetService
// backed by Redis (spend counters + rate limiter), a health endpoint, and
// alert emission (to Kafka) when a request is blocked.
package main

import (
	"context"
	"net"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	budgetv1 "github.com/marginpilot/contracts/gen/budget/v1"
	"github.com/marginpilot/shared/config"
	"github.com/marginpilot/shared/httpx"
	"github.com/marginpilot/shared/logger"

	"github.com/marginpilot/budget/internal/adapter/alerter"
	"github.com/marginpilot/budget/internal/adapter/grpcserver"
	"github.com/marginpilot/budget/internal/adapter/redisstore"
	"github.com/marginpilot/budget/internal/enforcer"
)

func main() {
	log := logger.New("budget")

	rdb := redis.NewClient(&redis.Options{Addr: config.Get("REDIS_ADDR", "localhost:6379")})
	defer func() { _ = rdb.Close() }()

	defaultLimit := int64(config.GetInt("BUDGET_DEFAULT_LIMIT_MICROS", 5_000_000)) // $5.00
	enf := enforcer.New(redisstore.New(rdb), defaultLimit)

	// Alert producer: Kafka when configured, else disabled.
	var alertPub grpcserver.AlertPublisher
	if brokers := config.Get("KAFKA_BROKERS", ""); brokers != "" {
		ka := alerter.NewKafka(strings.Split(brokers, ","))
		defer func() { _ = ka.Close() }()
		alertPub = ka
		log.Info("alerts: kafka", "brokers", brokers)
	} else {
		log.Info("alerts: disabled")
	}

	grpcAddr := config.Get("BUDGET_GRPC_ADDR", ":9101")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc listen failed", "addr", grpcAddr, "err", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	budgetv1.RegisterBudgetServiceServer(gs, grpcserver.New(enf, alertPub, log))
	go func() {
		log.Info("grpc listening", "addr", grpcAddr)
		if err := gs.Serve(lis); err != nil {
			log.Error("grpc serve failed", "err", err)
		}
	}()

	srv := httpx.New(config.Get("BUDGET_ADDR", ":8082"), log)
	if err := srv.Run(context.Background()); err != nil {
		log.Error("server exited with error", "err", err)
	}
	gs.GracefulStop()
}
