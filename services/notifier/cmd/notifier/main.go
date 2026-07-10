// Command notifier fans out alert events (budget breaches, spend anomalies) to
// the log and an optional webhook. With KAFKA_BROKERS unset it runs health-only.
package main

import (
	"context"
	"strings"

	"github.com/marginpilot/notifier/internal/consumer"
	"github.com/marginpilot/shared/config"
	"github.com/marginpilot/shared/httpx"
	"github.com/marginpilot/shared/logger"
)

func main() {
	log := logger.New("notifier")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if brokers := config.Get("KAFKA_BROKERS", ""); brokers != "" {
		cons := consumer.New(strings.Split(brokers, ","), config.Get("NOTIFIER_WEBHOOK_URL", ""), log)
		defer func() { _ = cons.Close() }()
		go func() {
			if err := cons.Run(ctx); err != nil {
				log.Error("consumer stopped", "err", err)
			}
		}()
		log.Info("consumer started", "brokers", brokers)
	} else {
		log.Info("KAFKA_BROKERS unset — running health-only (no consumer)")
	}

	srv := httpx.New(config.Get("NOTIFIER_ADDR", ":8086"), log)
	if err := srv.Run(ctx); err != nil {
		log.Error("server exited with error", "err", err)
	}
}
