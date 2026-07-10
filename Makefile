# In workspace mode `go build ./...` can't be run from the (module-less) root,
# so we list the workspace modules explicitly.
MODULES := ./proto/... ./shared/... ./services/gateway/... ./services/metering/... \
           ./services/budget/... ./services/identity/... ./services/rating/... \
           ./services/billing/... ./services/notifier/...

.PHONY: help tidy build test run-gateway run-metering run-budget run-identity run-rating run-billing run-notifier proto up down logs fmt vet

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

tidy: ## Sync workspace module requirements
	go work sync

build: ## Build all Go services
	go build $(MODULES)

vet: ## Static analysis
	go vet $(MODULES)

test: ## Run all Go tests
	go test $(MODULES)

run-gateway: ## Run the gateway locally (:8080)
	go run ./services/gateway/cmd/gateway

run-metering: ## Run the metering service locally (:8081)
	go run ./services/metering/cmd/metering

run-budget: ## Run the budget service locally (grpc :9101)
	go run ./services/budget/cmd/budget

run-identity: ## Run the identity service locally (grpc :9102)
	go run ./services/identity/cmd/identity

run-rating: ## Run the rating service locally (grpc :9103)
	go run ./services/rating/cmd/rating

run-billing: ## Run the billing service locally (:8085)
	go run ./services/billing/cmd/billing

run-notifier: ## Run the notifier service locally (:8086)
	go run ./services/notifier/cmd/notifier

proto: ## Regenerate gRPC code from .proto (needs protoc + plugins on PATH)
	cd proto && protoc --go_out=. --go_opt=module=github.com/marginpilot/contracts \
		--go-grpc_out=. --go-grpc_opt=module=github.com/marginpilot/contracts \
		budget/v1/budget.proto identity/v1/identity.proto

up: ## Build and start the full stack in Docker
	docker compose up --build -d

down: ## Stop the stack and remove volumes
	docker compose down -v

logs: ## Tail application service logs
	docker compose logs -f gateway metering budget identity rating billing notifier analytics

fmt: ## Format Go code
	go fmt $(MODULES)
