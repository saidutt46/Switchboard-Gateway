.PHONY: help setup up down restart logs clean test build run dev db-migrate db-reset kafka-topics verify

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

setup: ## Initial project setup (go mod download)
	@echo "🔧 Setting up Switchboard Gateway..."
	go mod download
	go mod verify
	@echo "✅ Dependencies downloaded"

up: ## Start all services (PostgreSQL, Redis, Kafka)
	@echo "🚀 Starting all services..."
	docker-compose up -d
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 10
	@make verify

down: ## Stop all services
	@echo "🛑 Stopping all services..."
	docker-compose down

restart: ## Restart all services
	@echo "🔄 Restarting services..."
	docker-compose restart

logs: ## Show logs from all services
	docker-compose logs -f

logs-gateway: ## Show logs from gateway only
	docker-compose logs -f gateway

logs-postgres: ## Show logs from PostgreSQL
	docker-compose logs -f postgres

logs-redis: ## Show logs from Redis
	docker-compose logs -f redis

logs-kafka: ## Show logs from Kafka
	docker-compose logs -f kafka

clean: ## Stop and remove all containers, volumes, and networks
	@echo "🧹 Cleaning up..."
	docker-compose down -v
	rm -rf vendor/
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
		echo "✅ Database reset complete"; \
	fi

db-migrate: ## Run database migrations (currently: schema.sql)
	@echo "📦 Running migrations..."
	docker exec -i switchboard-postgres psql -U switchboard -d switchboard < schema.sql
	@echo "✅ Migrations complete"

db-query: ## Run a custom SQL query (usage: make db-query SQL="SELECT * FROM services")
	docker exec -it switchboard-postgres psql -U switchboard -d switchboard -c "$(SQL)"

# ============================================================================
# Redis Operations
# ============================================================================

redis-cli: ## Connect to Redis CLI
	docker exec -it switchboard-redis redis-cli

redis-flush: ## Flush all Redis data (WARNING: clears cache!)
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
	docker exec switchboard-kafka kafka-topics --bootstrap-server localhost:9092 \
		--create --if-not-exists --topic gateway.config.changes \
		--partitions 1 --replication-factor 1 --config cleanup.policy=compact
	@echo "✅ Topics created"

kafka-consume-requests: ## Consume request logs from Kafka
	docker exec -it switchboard-kafka kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic gateway.requests \
		--from-beginning

# ============================================================================
# Development
# ============================================================================

build: ## Build the gateway binary
	@echo "🔨 Building gateway..."
	go build -o bin/gateway cmd/gateway/main.go
	@echo "✅ Build complete: bin/gateway"

run: ## Run the gateway locally
	@echo "🚀 Starting gateway..."
	go run cmd/gateway/main.go

dev: up ## Start services and run gateway in development mode
	@echo "🔧 Development mode - watching for changes..."
	@# TODO: Add air or other hot reload tool
	go run cmd/gateway/main.go

test: ## Run all tests
	@echo "🧪 Running tests..."
	go test ./... -v -cover

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	go test ./internal/... -v -short

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	go test ./tests/integration/... -v

test-coverage: ## Generate test coverage report
	@echo "📊 Generating coverage report..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# ============================================================================
# Code Quality
# ============================================================================

lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "💅 Formatting code..."
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...

# ============================================================================
# Verification
# ============================================================================

verify: ## Verify all services are running
	@echo "🔍 Verifying services..."
	@echo -n "PostgreSQL: "
	@docker exec switchboard-postgres pg_isready -U switchboard > /dev/null 2>&1 && echo "✅ Running" || echo "❌ Not running"
	@echo -n "Redis: "
	@docker exec switchboard-redis redis-cli ping > /dev/null 2>&1 && echo "✅ Running" || echo "❌ Not running"
	@echo -n "Kafka: "
	@docker exec switchboard-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1 && echo "✅ Running" || echo "❌ Not running"
	@echo -n "Demo Backend: "
	@curl -s http://localhost:8081/status > /dev/null 2>&1 && echo "✅ Running" || echo "❌ Not running"

health: ## Check health of all services
	@echo "🏥 Health check..."
	@echo "PostgreSQL:"
	@docker exec switchboard-postgres psql -U switchboard -d switchboard -c "SELECT version();" || true
	@echo ""
	@echo "Redis:"
	@docker exec switchboard-redis redis-cli INFO server | grep redis_version || true
	@echo ""
	@echo "Kafka:"
	@docker exec switchboard-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 | head -n 1 || true

# ============================================================================
# Demo & Testing
# ============================================================================

demo: ## Insert demo data into database
	@echo "📝 Inserting demo data..."
	@docker exec -i switchboard-postgres psql -U switchboard -d switchboard <<-EOSQL
		-- Additional demo service
		INSERT INTO services (name, protocol, host, port) VALUES
		('product-service', 'http', 'demo-backend', 80)
		ON CONFLICT (name) DO NOTHING;
		
		-- Additional demo route
		INSERT INTO routes (service_id, name, paths, methods) VALUES
		((SELECT id FROM services WHERE name = 'product-service'), 
		 'product-api', 
		 ARRAY['/api/products', '/api/products/:id'],
		 ARRAY['GET', 'POST', 'PUT', 'DELETE'])
		ON CONFLICT DO NOTHING;
	EOSQL
	@echo "✅ Demo data inserted"

test-proxy: ## Test the proxy with a simple request
	@echo "🧪 Testing proxy..."
	curl -v http://localhost:8080/api/users

test-auth: ## Test API key authentication
	@echo "🔐 Testing authentication..."
	@echo "Without key (should fail):"
	curl -i http://localhost:8080/api/users
	@echo ""
	@echo "With key (should succeed):"
	curl -i -H "X-API-Key: test-key-12345" http://localhost:8080/api/users

# ============================================================================
# Docker Operations
# ============================================================================

docker-build: ## Build Docker image for gateway
	@echo "🐳 Building Docker image..."
	docker build -t switchboard-gateway:latest .
	@echo "✅ Image built: switchboard-gateway:latest"

docker-push: ## Push Docker image to registry (requires LOGIN)
	@echo "📤 Pushing to registry..."
	docker push switchboard-gateway:latest

# ============================================================================
# Documentation
# ============================================================================

docs: ## Generate API documentation
	@echo "📚 Generating documentation..."
	@# TODO: Add swagger/godoc generation
	@echo "⚠️  Documentation generation not yet implemented"

# ============================================================================
# Default Target
# ============================================================================

.DEFAULT_GOAL := help