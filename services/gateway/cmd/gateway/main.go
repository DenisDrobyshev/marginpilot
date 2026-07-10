// Command gateway is the data-plane entrypoint: an OpenAI-compatible proxy that
// applies guardrails, serves a response cache, enforces budgets and emits a
// usage event per call. Outbound dependencies are selected by environment: with
// none set it runs fully in-process (echo provider, stdout publisher, allow-all
// budget, header caller, no cache, no guardrails) so it works with no infra.
package main

import (
	"context"
	"os"
	"strings"

	"github.com/marginpilot/gateway/internal/adapter/inbound/httpapi"
	"github.com/marginpilot/gateway/internal/adapter/outbound/budget"
	"github.com/marginpilot/gateway/internal/adapter/outbound/cache"
	"github.com/marginpilot/gateway/internal/adapter/outbound/guardrail"
	"github.com/marginpilot/gateway/internal/adapter/outbound/provider"
	"github.com/marginpilot/gateway/internal/adapter/outbound/publisher"
	"github.com/marginpilot/gateway/internal/adapter/outbound/resolver"
	"github.com/marginpilot/gateway/internal/app"
	"github.com/marginpilot/gateway/internal/port"
	"github.com/marginpilot/shared/config"
	"github.com/marginpilot/shared/httpx"
	"github.com/marginpilot/shared/logger"
)

func main() {
	log := logger.New("gateway")

	prov := provider.NewEcho()

	// Usage publisher: Kafka when configured, else stdout.
	var pub port.UsagePublisher
	if brokers := config.Get("KAFKA_BROKERS", ""); brokers != "" {
		kp := publisher.NewKafka(strings.Split(brokers, ","))
		defer func() { _ = kp.Close() }()
		pub = kp
		log.Info("usage publisher: kafka", "brokers", brokers)
	} else {
		pub = publisher.NewStdout(log)
		log.Info("usage publisher: stdout")
	}

	// Budget checker: budget gRPC service when configured, else allow-all.
	var bud port.BudgetChecker
	if target := config.Get("BUDGET_GRPC_TARGET", ""); target != "" {
		gb, err := budget.NewGRPC(target)
		if err != nil {
			log.Error("budget client init failed", "err", err)
			os.Exit(1)
		}
		defer func() { _ = gb.Close() }()
		bud = gb
		log.Info("budget checker: grpc", "target", target)
	} else {
		bud = budget.NewAllowAll()
		log.Info("budget checker: allow-all")
	}

	// Response cache: Redis when configured, else disabled.
	var cch port.Cache
	if addr := config.Get("REDIS_ADDR", ""); addr != "" {
		rc := cache.NewRedis(addr)
		defer func() { _ = rc.Close() }()
		cch = rc
		log.Info("cache: redis", "addr", addr)
	} else {
		cch = cache.NewNoop()
		log.Info("cache: disabled")
	}

	// Guardrails: policy when GUARDRAILS_MODE is redact|block, else off.
	var guard port.Guardrail
	switch mode := config.Get("GUARDRAILS_MODE", "off"); mode {
	case "redact", "block":
		guard = guardrail.NewPolicy(guardrail.Mode(mode),
			strings.Split(config.Get("GUARDRAILS_DENYLIST", ""), ","))
		log.Info("guardrails: on", "mode", mode)
	default:
		guard = guardrail.NewNoop()
		log.Info("guardrails: off")
	}

	// Caller resolver: identity gRPC service when configured, else header fallback.
	var res app.CallerResolver
	if target := config.Get("IDENTITY_GRPC_TARGET", ""); target != "" {
		ir, err := resolver.NewIdentity(target)
		if err != nil {
			log.Error("identity client init failed", "err", err)
			os.Exit(1)
		}
		defer func() { _ = ir.Close() }()
		res = ir
		log.Info("caller resolver: identity grpc", "target", target)
	} else {
		log.Info("caller resolver: header/demo fallback")
	}

	svc := app.New(prov, pub, bud, cch, guard, log)

	srv := httpx.New(config.Get("GATEWAY_ADDR", ":8080"), log)
	httpapi.New(svc, res, log).Register(srv.Mux())

	if err := srv.Run(context.Background()); err != nil {
		log.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}
