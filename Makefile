.PHONY: help setup up down restart logs clean test build run verify

# ============================================================================
# Switchboard Gateway - Makefile
# ============================================================================

help: ## Show this help message
	@echo "Switchboard Gateway - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ============================================================================
# Setup & Infrastructure
# ============================================================================

setup: ## Install Go dependencies
	@echo "🔧 Installing dependencies..."
	go mod download
	go mod tidy
	go mod verify
	@echo "✅ Dependencies installed"

up: ## Start all services (PostgreSQL, Redis, Kafka)
	@echo "🚀 Starting services..."
	docker-compose up -d
	@echo "⏳ Waiting for services..."
	@sleep 10
	@make verify

down: ## Stop all services
	@echo "🛑 Stopping services..."
	docker-compose down

restart: ## Restart all services
	@make down
	@make up

logs: ## Show logs from all services
	docker-compose logs -f

logs-postgres: ## Show PostgreSQL logs
	docker-compose logs -f postgres

logs-redis: ## Show Redis logs
	docker-compose logs -f redis

logs-kafka: ## Show Kafka logs
	docker-compose logs -f kafka

clean: ## Stop and remove all containers and volumes
	@echo "🧹 Cleaning up..."
	docker-compose down -v
	rm -rf bin/
	@echo "✅ Cleanup complete"

# ============================================================================
# Database Operations
# ============================================================================

db-connect: ## Connect to PostgreSQL via psql
	docker exec -it switchboard-postgres psql -U switchboard -d switchboard

db-reset: ## Reset database (WARNING: destroys all data!)
	@echo "⚠️  WARNING: This will destroy all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down postgres; \
		docker volume rm switchboard-gateway_postgres_data 2>/dev/null || true; \
		docker-compose up -d postgres; \
		sleep 5; \
		echo "✅ Database reset complete"; \
	fi

db-query: ## Run a SQL query (usage: make db-query SQL="SELECT * FROM services")
	docker exec -it switchboard-postgres psql -U switchboard -d switchboard -c "$(SQL)"

db-restore: ## Restore sample data to database
	@echo "🔄 Restoring sample data..."
	docker exec -i switchboard-postgres psql -U switchboard -d switchboard < schema.sql
	@echo "✅ Sample data restored"

db-setup-test: ## Setup database for testing with go-httpbin
	@echo "🔧 Setting up test routes..."
	@cat tests/manual/setup_test_routes.sql | docker exec -i switchboard-postgres psql -U switchboard -d switchboard
	@echo "✅ Test routes configured"

db-init: db-restore db-setup-test ## Initialize database with schema and test data

# ============================================================================
# Redis Operations
# ============================================================================

redis-cli: ## Connect to Redis CLI
	docker exec -it switchboard-redis redis-cli

redis-flush: ## Flush all Redis data
	@echo "⚠️  Flushing Redis cache..."
	docker exec -it switchboard-redis redis-cli FLUSHALL
	@echo "✅ Redis flushed"

# ============================================================================
# Kafka Operations
# ============================================================================

kafka-topics: ## List all Kafka topics
	docker exec switchboard-kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-create-topics: ## Create required Kafka topics
	@echo "📋 Creating Kafka topics..."
	docker exec switchboard-kafka kafka-topics --bootstrap-server localhost:9092 \
		--create --if-not-exists --topic gateway.requests \
		--partitions 6 --replication-factor 1
	docker exec switchboard-kafka kafka-topics --bootstrap-server localhost:9092 \
		--create --if-not-exists --topic gateway.errors \
		--partitions 3 --replication-factor 1
	@echo "✅ Topics created"

# ============================================================================
# Gateway Operations
# ============================================================================

run: ## Run the gateway (loads .env automatically)
	@echo "🚀 Starting Switchboard Gateway..."
	go run cmd/gateway/main.go

run-dev: up run ## Start services and run gateway

build: ## Build the gateway binary
	@echo "🔨 Building gateway..."
	@mkdir -p bin
	go build -o bin/gateway cmd/gateway/main.go
	@echo "✅ Binary created: bin/gateway"

build-prod: ## Build production binary with version info
	@echo "🔨 Building production binary..."
	@mkdir -p bin
	go build -ldflags "-X main.Version=0.2.0 -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S') -X main.GitCommit=$(shell git rev-parse --short HEAD)" -o bin/gateway cmd/gateway/main.go
	@echo "✅ Production binary created: bin/gateway"

install: build ## Install binary to $GOPATH/bin
	@echo "📦 Installing gateway..."
	go install cmd/gateway/main.go
	@echo "✅ Installed to $(shell go env GOPATH)/bin/gateway"

# ============================================================================
# Testing
# ============================================================================

test: ## Run all tests
	@echo "🧪 Running tests..."
	go test ./... -v

test-coverage: ## Generate test coverage report
	@echo "📊 Generating coverage report..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

test-race: ## Run tests with race detector
	@echo "🏁 Running tests with race detector..."
	go test ./... -race -v

test-router: ## Test router package
	@echo "🧪 Testing router..."
	go test ./internal/router/... -v

test-proxy: ## Test proxy package
	@echo "🧪 Testing proxy..."
	go test ./internal/proxy/... -v

test-phase3: test-router test-proxy ## Test all Phase 3 components
	@echo "✅ Phase 3 tests complete"

# ============================================================================
# Code Quality
# ============================================================================

fmt: ## Format code
	@echo "💅 Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...
	@echo "✅ Vet passed"

lint: ## Run linter (requires golangci-lint)
	@echo "🔍 Running linter..."
	@which golangci-lint > /dev/null || (echo "❌ golangci-lint not installed. Run: brew install golangci-lint"; exit 1)
	golangci-lint run
	@echo "✅ Lint passed"

# ============================================================================
# Testing & Verification
# ============================================================================

load-test: ## Run load test with k6 (requires k6 installed)
	@echo "🔥 Running load test..."
	@which k6 > /dev/null || (echo "❌ k6 not installed. Run: brew install k6"; exit 1)
	k6 run tests/load/simple.js

# ============================================================================
# Verification & Health Checks
# ============================================================================

verify: ## Verify all services are running
	@echo "🔍 Verifying services..."
	@echo -n "PostgreSQL: "
	@docker exec switchboard-postgres pg_isready -U switchboard > /dev/null 2>&1 && echo "✅" || echo "❌"
	@echo -n "Redis: "
	@docker exec switchboard-redis redis-cli ping > /dev/null 2>&1 && echo "✅" || echo "❌"
	@echo -n "Kafka: "
	@docker exec switchboard-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1 && echo "✅" || echo "❌"
	@echo -n "Demo Backend: "
	@curl -s http://localhost:8081/status > /dev/null 2>&1 && echo "✅" || echo "❌"

health: ## Check gateway health endpoint
	@echo "🏥 Checking gateway health..."
	@curl -s http://localhost:8080/health | jq '.' || echo "❌ Gateway not running or jq not installed"

ready: ## Check gateway ready endpoint
	@echo "✅ Checking gateway readiness..."
	@curl -s http://localhost:8080/ready | jq '.' || echo "❌ Gateway not running or jq not installed"

# ============================================================================
# Quick Start
# ============================================================================

start: up run ## Quick start: Start services and run gateway

stop: ## Stop gateway and services
	@echo "🛑 Stopping everything..."
	@pkill -f "go run cmd/gateway/main.go" 2>/dev/null || true
	@make down

# ============================================================================
# Default Target
# ============================================================================

.DEFAULT_GOAL := help