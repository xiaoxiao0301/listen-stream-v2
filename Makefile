# Listen Stream Server Makefile

.PHONY: help proto-gen proto-clean test cover lint fmt build clean run-auth run-proxy docker-build docker-push migrate-up migrate-down sqlc-gen mock-gen check-deps

# Default target
.DEFAULT_GOAL := help

# Variables
PROTO_DIR := server/shared/proto
PROTO_OUT_DIR := server/shared/proto
GO_PACKAGES := $(shell go list ./... | grep -v /proto/)

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(COLOR_BOLD)Usage:$(COLOR_RESET)\n  make $(COLOR_BLUE)<target>$(COLOR_RESET)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(COLOR_BLUE)%-15s$(COLOR_RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Protobuf

proto-gen: ## Generate Go code from .proto files
	@echo "$(COLOR_GREEN)Generating protobuf code...$(COLOR_RESET)"
	@protoc --version || (echo "$(COLOR_YELLOW)protoc not found. Install with: brew install protobuf$(COLOR_RESET)" && exit 1)
	@which protoc-gen-go > /dev/null || (echo "$(COLOR_YELLOW)protoc-gen-go not found. Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest$(COLOR_RESET)" && exit 1)
	@which protoc-gen-go-grpc > /dev/null || (echo "$(COLOR_YELLOW)protoc-gen-go-grpc not found. Install with: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest$(COLOR_RESET)" && exit 1)
	@protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(PROTO_OUT_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/auth/v1/*.proto \
		$(PROTO_DIR)/user/v1/*.proto \
		$(PROTO_DIR)/sync/v1/*.proto \
		$(PROTO_DIR)/admin/v1/*.proto
	@echo "$(COLOR_GREEN)✓ Protobuf code generated successfully$(COLOR_RESET)"

proto-clean: ## Remove generated protobuf files
	@echo "$(COLOR_YELLOW)Cleaning generated protobuf files...$(COLOR_RESET)"
	@find $(PROTO_DIR) -name "*.pb.go" -type f -delete
	@echo "$(COLOR_GREEN)✓ Protobuf files cleaned$(COLOR_RESET)"

##@ Development

run: ## Run current service (requires SERVICE env var, e.g., make run SERVICE=auth-svc)
	@if [ -z "$(SERVICE)" ]; then \
		echo "$(COLOR_YELLOW)Usage: make run SERVICE=auth-svc$(COLOR_RESET)"; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)Starting $(SERVICE)...$(COLOR_RESET)"
	@cd server/services/$(SERVICE) && go run cmd/main.go

run-auth: ## Run auth-svc
	@$(MAKE) run SERVICE=auth-svc

run-proxy: ## Run proxy-svc
	@$(MAKE) run SERVICE=proxy-svc

run-user: ## Run user-svc
	@$(MAKE) run SERVICE=user-svc

run-sync: ## Run sync-svc
	@$(MAKE) run SERVICE=sync-svc

run-admin: ## Run admin-svc
	@$(MAKE) run SERVICE=admin-svc

##@ Testing

test: ## Run all tests
	@echo "$(COLOR_GREEN)Running tests...$(COLOR_RESET)"
	@go test -v -race -timeout 5m $(GO_PACKAGES)

test-short: ## Run short tests (skip integration tests)
	@echo "$(COLOR_GREEN)Running short tests...$(COLOR_RESET)"
	@go test -v -short -race -timeout 2m $(GO_PACKAGES)

cover: ## Run tests with coverage
	@echo "$(COLOR_GREEN)Running tests with coverage...$(COLOR_RESET)"
	@go test -v -race -coverprofile=coverage.out -covermode=atomic $(GO_PACKAGES)
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)✓ Coverage report generated: coverage.html$(COLOR_RESET)"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

benchmark: ## Run benchmarks
	@echo "$(COLOR_GREEN)Running benchmarks...$(COLOR_RESET)"
	@go test -v -bench=. -benchmem -run=^# $(GO_PACKAGES)

##@ Code Quality

lint: ## Run golangci-lint
	@echo "$(COLOR_GREEN)Running linter...$(COLOR_RESET)"
	@which golangci-lint > /dev/null || (echo "$(COLOR_YELLOW)golangci-lint not found. Install from: https://golangci-lint.run/usage/install/$(COLOR_RESET)" && exit 1)
	@golangci-lint run --timeout 5m ./...

fmt: ## Format Go code
	@echo "$(COLOR_GREEN)Formatting code...$(COLOR_RESET)"
	@go fmt ./...
	@goimports -w -local github.com/listen-stream server/ || echo "$(COLOR_YELLOW)goimports not found, skipping. Install with: go install golang.org/x/tools/cmd/goimports@latest$(COLOR_RESET)"

vet: ## Run go vet
	@echo "$(COLOR_GREEN)Running go vet...$(COLOR_RESET)"
	@go vet ./...

##@ Build

build: ## Build all services
	@echo "$(COLOR_GREEN)Building all services...$(COLOR_RESET)"
	@mkdir -p bin
	@for service in auth-svc proxy-svc user-svc sync-svc admin-svc; do \
		echo "Building $$service..."; \
		cd server/services/$$service && go build -o ../../../bin/$$service cmd/main.go && cd ../../..; \
	done
	@echo "$(COLOR_GREEN)✓ All services built successfully$(COLOR_RESET)"

build-service: ## Build a specific service (requires SERVICE env var)
	@if [ -z "$(SERVICE)" ]; then \
		echo "$(COLOR_YELLOW)Usage: make build-service SERVICE=auth-svc$(COLOR_RESET)"; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)Building $(SERVICE)...$(COLOR_RESET)"
	@mkdir -p bin
	@cd server/services/$(SERVICE) && go build -o ../../../bin/$(SERVICE) cmd/main.go
	@echo "$(COLOR_GREEN)✓ $(SERVICE) built successfully$(COLOR_RESET)"

clean: ## Clean build artifacts
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "$(COLOR_GREEN)✓ Build artifacts cleaned$(COLOR_RESET)"

##@ Docker

docker-build: ## Build Docker images for all services
	@echo "$(COLOR_GREEN)Building Docker images...$(COLOR_RESET)"
	@for service in auth-svc proxy-svc user-svc sync-svc admin-svc; do \
		echo "Building $$service Docker image..."; \
		docker build -f server/services/$$service/Dockerfile -t listen-stream/$$service:latest .; \
	done
	@echo "$(COLOR_GREEN)✓ Docker images built successfully$(COLOR_RESET)"

docker-push: ## Push Docker images to registry
	@echo "$(COLOR_GREEN)Pushing Docker images...$(COLOR_RESET)"
	@if [ -z "$(REGISTRY)" ]; then \
		echo "$(COLOR_YELLOW)Usage: make docker-push REGISTRY=your-registry.com$(COLOR_RESET)"; \
		exit 1; \
	fi
	@for service in auth-svc proxy-svc user-svc sync-svc admin-svc; do \
		echo "Pushing $$service..."; \
		docker tag listen-stream/$$service:latest $(REGISTRY)/listen-stream/$$service:latest; \
		docker push $(REGISTRY)/listen-stream/$$service:latest; \
	done
	@echo "$(COLOR_GREEN)✓ Docker images pushed successfully$(COLOR_RESET)"

##@ Database

migrate-up: ## Run database migrations (up)
	@echo "$(COLOR_GREEN)Running database migrations...$(COLOR_RESET)"
	@which migrate > /dev/null || (echo "$(COLOR_YELLOW)migrate not found. Install from: https://github.com/golang-migrate/migrate$(COLOR_RESET)" && exit 1)
	@migrate -path server/migrations -database "$(DATABASE_URL)" up
	@echo "$(COLOR_GREEN)✓ Migrations applied successfully$(COLOR_RESET)"

migrate-down: ## Rollback database migrations (down)
	@echo "$(COLOR_YELLOW)Rolling back database migrations...$(COLOR_RESET)"
	@migrate -path server/migrations -database "$(DATABASE_URL)" down
	@echo "$(COLOR_GREEN)✓ Migrations rolled back$(COLOR_RESET)"

migrate-create: ## Create a new migration file (requires NAME env var)
	@if [ -z "$(NAME)" ]; then \
		echo "$(COLOR_YELLOW)Usage: make migrate-create NAME=create_users_table$(COLOR_RESET)"; \
		exit 1; \
	fi
	@migrate create -ext sql -dir server/migrations -seq $(NAME)
	@echo "$(COLOR_GREEN)✓ Migration files created$(COLOR_RESET)"

sqlc-gen: ## Generate sqlc code
	@echo "$(COLOR_GREEN)Generating sqlc code...$(COLOR_RESET)"
	@which sqlc > /dev/null || (echo "$(COLOR_YELLOW)sqlc not found. Install from: https://docs.sqlc.dev/en/latest/overview/install.html$(COLOR_RESET)" && exit 1)
	@cd server && sqlc generate
	@echo "$(COLOR_GREEN)✓ SQLC code generated successfully$(COLOR_RESET)"

##@ Code Generation

mock-gen: ## Generate mock files
	@echo "$(COLOR_GREEN)Generating mocks...$(COLOR_RESET)"
	@which mockgen > /dev/null || (echo "$(COLOR_YELLOW)mockgen not found. Install with: go install github.com/golang/mock/mockgen@latest$(COLOR_RESET)" && exit 1)
	@go generate ./...
	@echo "$(COLOR_GREEN)✓ Mocks generated successfully$(COLOR_RESET)"

##@ Dependencies

deps: ## Download and tidy dependencies
	@echo "$(COLOR_GREEN)Downloading dependencies...$(COLOR_RESET)"
	@cd server && go mod download
	@cd server && go mod tidy
	@echo "$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)"

check-deps: ## Check for outdated dependencies
	@echo "$(COLOR_GREEN)Checking for outdated dependencies...$(COLOR_RESET)"
	@cd server && go list -u -m all

##@ Deployment

deploy-staging: ## Deploy to staging environment
	@echo "$(COLOR_GREEN)Deploying to staging...$(COLOR_RESET)"
	@kubectl apply -f deployments/k8s/staging/
	@echo "$(COLOR_GREEN)✓ Deployed to staging$(COLOR_RESET)"

deploy-prod: ## Deploy to production environment
	@echo "$(COLOR_GREEN)Deploying to production...$(COLOR_RESET)"
	@kubectl apply -f deployments/k8s/production/
	@echo "$(COLOR_GREEN)✓ Deployed to production$(COLOR_RESET)"

##@ Utilities

install-tools: ## Install required development tools
	@echo "$(COLOR_GREEN)Installing development tools...$(COLOR_RESET)"
	@echo "Installing protoc plugins..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installing code generation tools..."
	@go install github.com/golang/mock/mockgen@latest
	@echo "Installing linter..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(COLOR_GREEN)✓ All tools installed$(COLOR_RESET)"

init: install-tools proto-gen deps ## Initialize project (install tools, generate code, download deps)
	@echo "$(COLOR_GREEN)✓ Project initialized successfully$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Next steps:$(COLOR_RESET)"
	@echo "  1. Start services: docker-compose -f docker-compose.local.yml up -d"
	@echo "  2. Run migrations: make migrate-up DATABASE_URL=postgresql://postgres:postgres123@localhost:5432/listen_stream?sslmode=disable"
	@echo "  3. Start a service: make run-auth"

##@ Infrastructure (Phase 7)

INFRA_DIR := infra
INFRA_COMPOSE := $(INFRA_DIR)/docker-compose.yml
SERVICES_COMPOSE := $(INFRA_DIR)/docker-compose.services.yml

infra-up: ## Start full infrastructure (Consul + OTel/Jaeger + Prometheus/Grafana + ELK)
	@echo "$(COLOR_GREEN)Starting Listen Stream infrastructure...$(COLOR_RESET)"
	@cp -n $(INFRA_DIR)/.env.example $(INFRA_DIR)/.env 2>/dev/null || true
	@docker compose -f $(INFRA_COMPOSE) --env-file $(INFRA_DIR)/.env up -d
	@echo ""
	@echo "$(COLOR_GREEN)✓ Infrastructure started$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Service URLs:$(COLOR_RESET)"
	@echo "  Consul UI:    http://localhost:8500"
	@echo "  Jaeger UI:    http://localhost:16686"
	@echo "  Prometheus:   http://localhost:9090"
	@echo "  Grafana:      http://localhost:3000  (admin / listen_stream_admin_2026)"
	@echo "  AlertManager: http://localhost:9093"
	@echo "  Kibana:       http://localhost:5601"
	@echo "  Elasticsearch:http://localhost:9200"

infra-down: ## Stop all infrastructure containers
	@echo "$(COLOR_YELLOW)Stopping infrastructure...$(COLOR_RESET)"
	@docker compose -f $(INFRA_COMPOSE) down
	@echo "$(COLOR_GREEN)✓ Infrastructure stopped$(COLOR_RESET)"

infra-clean: ## Stop and remove all infrastructure data (DESTRUCTIVE)
	@echo "$(COLOR_YELLOW)WARNING: This will delete all infrastructure data!$(COLOR_RESET)"
	@read -p "Are you sure? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	@docker compose -f $(INFRA_COMPOSE) down -v --remove-orphans
	@echo "$(COLOR_GREEN)✓ Infrastructure data cleaned$(COLOR_RESET)"

infra-logs: ## Tail logs from all infrastructure containers
	@docker compose -f $(INFRA_COMPOSE) logs -f --tail=50

infra-status: ## Show status of all infrastructure containers
	@docker compose -f $(INFRA_COMPOSE) ps

infra-consul-status: ## Show Consul cluster members
	@echo "$(COLOR_GREEN)Consul cluster members:$(COLOR_RESET)"
	@docker exec listen-stream-consul-1 consul members

infra-consul-kv: ## List all Consul KV keys
	@docker exec listen-stream-consul-1 consul kv get -recurse listen-stream/

infra-consul-reinit: ## Re-run Consul KV initialization
	@echo "$(COLOR_GREEN)Re-initializing Consul KV...$(COLOR_RESET)"
	@docker run --rm --network listen-stream-backend \
		-v $(PWD)/$(INFRA_DIR)/consul/init-kv.sh:/init-kv.sh \
		curlimages/curl:latest /bin/sh /init-kv.sh http://consul-server-1:8500
	@echo "$(COLOR_GREEN)✓ Consul KV re-initialized$(COLOR_RESET)"

infra-es-status: ## Show Elasticsearch cluster health
	@curl -s http://localhost:9200/_cluster/health | python3 -m json.tool 2>/dev/null || \
		curl -s http://localhost:9200/_cluster/health

infra-es-indices: ## List Elasticsearch indices
	@curl -s http://localhost:9200/_cat/indices?v

infra-prometheus-reload: ## Reload Prometheus configuration
	@curl -s -X POST http://localhost:9090/-/reload
	@echo "$(COLOR_GREEN)✓ Prometheus configuration reloaded$(COLOR_RESET)"

services-up: ## Start Go services (requires infra to be running)
	@echo "$(COLOR_GREEN)Starting Go services...$(COLOR_RESET)"
	@docker compose \
		-f $(INFRA_COMPOSE) \
		-f $(SERVICES_COMPOSE) \
		--env-file $(INFRA_DIR)/.env \
		up -d auth-svc proxy-svc user-svc sync-svc admin-svc
	@echo "$(COLOR_GREEN)✓ Go services started$(COLOR_RESET)"

services-down: ## Stop Go services
	@docker compose \
		-f $(INFRA_COMPOSE) \
		-f $(SERVICES_COMPOSE) \
		down auth-svc proxy-svc user-svc sync-svc admin-svc

up: infra-up services-up ## Start everything (infra + services)
	@echo "$(COLOR_GREEN)✓ Full stack started$(COLOR_RESET)"

down: ## Stop everything
	@docker compose \
		-f $(INFRA_COMPOSE) \
		-f $(SERVICES_COMPOSE) \
		down

