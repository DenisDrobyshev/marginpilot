// Command metering consumes usage events, prices them (via rating or a built-in
// catalog), persists rows to ClickHouse and increments the Redis spend counters
// the budget service reads. With KAFKA_BROKERS unset it runs health-only, so
// `go run` works with no infra.
package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/marginpilot/metering/internal/adapter/pricer"
	"github.com/marginpilot/metering/internal/consumer"
	"github.com/marginpilot/metering/internal/sink"
	"github.com/marginpilot/shared/config"
	"github.com/marginpilot/shared/httpx"
	"github.com/marginpilot/shared/logger"
)

func main() {
	log := logger.New("metering")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if brokers := config.Get("KAFKA_BROKERS", ""); brokers != "" {
		startConsumer(ctx, log, brokers)
	} else {
		log.Info("KAFKA_BROKERS unset — running health-only (no consumer)")
	}

	srv := httpx.New(config.Get("METERING_ADDR", ":8081"), log)
	if err := srv.Run(ctx); err != nil {
		log.Error("server exited with error", "err", err)
	}
}

func startConsumer(ctx context.Context, log *logger.Logger, brokers string) {
	ch, err := sink.NewClickHouse(config.Get("CLICKHOUSE_ADDR", "localhost:9000"))
	if err != nil {
		log.Error("clickhouse open failed", "err", err)
		os.Exit(1)
	}

	// ClickHouse may still be starting in docker-compose; retry the schema.
	for attempt := 1; ; attempt++ {
		if err := ch.EnsureSchema(ctx); err == nil {
			break
		} else if attempt >= 15 {
			log.Error("clickhouse schema failed", "err", err)
			os.Exit(1)
		} else {
			log.Info("waiting for clickhouse", "attempt", attempt)
			time.Sleep(2 * time.Second)
		}
	}

	// Pricer: rating service when configured, else the built-in catalog.
	pr := pricerFrom(log)

	rdb := redis.NewClient(&redis.Options{Addr: config.Get("REDIS_ADDR", "localhost:6379")})
	cons := consumer.New(strings.Split(brokers, ","), ch, rdb, pr, log)
	go func() {
		if err := cons.Run(ctx); err != nil {
			log.Error("consumer stopped", "err", err)
		}
	}()
	log.Info("consumer started", "brokers", brokers)
}

func pricerFrom(log *logger.Logger) consumer.Pricer {
	if target := config.Get("RATING_GRPC_TARGET", ""); target != "" {
		rp, err := pricer.NewGRPC(target)
		if err != nil {
			log.Error("rating client init failed", "err", err)
			os.Exit(1)
		}
		log.Info("pricer: rating grpc", "target", target)
		return rp
	}
	log.Info("pricer: builtin")
	return pricer.NewBuiltin()
}
