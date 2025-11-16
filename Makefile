# Switchboard API Gateway - Makefile
# Complete build and development automation

.PHONY: help build run test clean docker fmt lint vet deps dev db-setup db-migrate db-reset services-up services-down logs admin stress plugin-test coverage benchmark

# Variables
BINARY_NAME=gateway
MAIN_PATH=./cmd/gateway
BUILD_DIR=./build
VERSION?=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Colors for output
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m
COLOR_BLUE=\033[34m

##@ Help

help: ## Display this help
	@echo "$(COLOR_BOLD)Switchboard API Gateway - Makefile Commands$(COLOR_RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make $(COLOR_BLUE)<target>$(COLOR_RESET)\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(COLOR_BLUE)%-20s$(COLOR_RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

dev: ## Run gateway in development mode with hot reload (requires air)
	@echo "$(COLOR_GREEN)Starting gateway in development mode...$(COLOR_RESET)"
	@air || (echo "$(COLOR_YELLOW)air not installed. Install with: go install github.com/cosmtrek/air@latest$(COLOR_RESET)" && go run $(MAIN_PATH))

run: build ## Build and run the gateway
	@echo "$(COLOR_GREEN)Running gateway...$(COLOR_RESET)"
	@./$(BUILD_DIR)/$(BINARY_NAME)

run-debug: build ## Run gateway with debug logging
	@echo "$(COLOR_GREEN)Running gateway with debug logging...$(COLOR_RESET)"
	@LOG_LEVEL=debug ./$(BUILD_DIR)/$(BINARY_NAME)

##@ Build

build: ## Build the gateway binary
	@echo "$(COLOR_GREEN)Building gateway...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(COLOR_RESET)"

build-linux: ## Build for Linux (production)
	@echo "$(COLOR_GREEN)Building for Linux...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ Linux build complete$(COLOR_RESET)"

build-mac: ## Build for macOS
	@echo "$(COLOR_GREEN)Building for macOS...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin $(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ macOS build complete$(COLOR_RESET)"

build-windows: ## Build for Windows
	@echo "$(COLOR_GREEN)Building for Windows...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "$(COLOR_GREEN)✓ Windows build complete$(COLOR_RESET)"

build-all: build-linux build-mac build-windows ## Build for all platforms

##@ Testing

test: ## Run all unit tests
	@echo "$(COLOR_GREEN)Running unit tests...$(COLOR_RESET)"
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(COLOR_GREEN)Running tests with coverage...$(COLOR_RESET)"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)✓ Coverage report: coverage.html$(COLOR_RESET)"

test-race: ## Run tests with race detection
	@echo "$(COLOR_GREEN)Running tests with race detection...$(COLOR_RESET)"
	@go test -race ./...

test-router: ## Test router package only
	@echo "$(COLOR_GREEN)Testing router package...$(COLOR_RESET)"
	@go test -v ./internal/router

test-plugin: ## Test plugin system
	@echo "$(COLOR_GREEN)Testing plugin system...$(COLOR_RESET)"
	@go test -v ./internal/plugin
	@go test -v ./internal/plugin/builtin

test-proxy: ## Test reverse proxy
	@echo "$(COLOR_GREEN)Testing reverse proxy...$(COLOR_RESET)"
	@go test -v ./internal/proxy

##@ Load Testing (k6)

stress-smoke: services-up ## Run smoke test (quick validation)
	@echo "$(COLOR_GREEN)Running smoke test...$(COLOR_RESET)"
	@k6 run tests/load/smoke.js

stress-pool: services-up ## Run connection pool test
	@echo "$(COLOR_GREEN)Running connection pool test...$(COLOR_RESET)"
	@k6 run tests/load/connection_pool.js

stress-full: services-up ## Run full gateway validation test
	@echo "$(COLOR_GREEN)Running full gateway validation...$(COLOR_RESET)"
	@k6 run tests/load/gateway_validation.js

stress-all: stress-smoke stress-pool stress-full ## Run all load tests sequentially

stress-custom: ## Run custom k6 test (usage: make stress-custom FILE=mytest.js VUS=50 DURATION=2m)
	@echo "$(COLOR_GREEN)Running custom test: $(FILE)...$(COLOR_RESET)"
	@k6 run $(if $(VUS),--vus $(VUS)) $(if $(DURATION),--duration $(DURATION)) tests/load/$(FILE)

##@ Code Quality

fmt: ## Format Go code
	@echo "$(COLOR_GREEN)Formatting code...$(COLOR_RESET)"
	@go fmt ./...
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

lint: ## Run golangci-lint
	@echo "$(COLOR_GREEN)Running linter...$(COLOR_RESET)"
	@golangci-lint run || (echo "$(COLOR_YELLOW)golangci-lint not installed. Install with: brew install golangci-lint$(COLOR_RESET)" && exit 1)

vet: ## Run go vet
	@echo "$(COLOR_GREEN)Running go vet...$(COLOR_RESET)"
	@go vet ./...

check: fmt vet lint test ## Run all code quality checks

##@ Dependencies

deps: ## Download Go dependencies
	@echo "$(COLOR_GREEN)Downloading dependencies...$(COLOR_RESET)"
	@go mod download
	@echo "$(COLOR_GREEN)✓ Dependencies downloaded$(COLOR_RESET)"

deps-tidy: ## Tidy Go dependencies
	@echo "$(COLOR_GREEN)Tidying dependencies...$(COLOR_RESET)"
	@go mod tidy
	@echo "$(COLOR_GREEN)✓ Dependencies tidied$(COLOR_RESET)"

deps-verify: ## Verify Go dependencies
	@echo "$(COLOR_GREEN)Verifying dependencies...$(COLOR_RESET)"
	@go mod verify
	@echo "$(COLOR_GREEN)✓ Dependencies verified$(COLOR_RESET)"

deps-update: ## Update all dependencies to latest
	@echo "$(COLOR_GREEN)Updating dependencies...$(COLOR_RESET)"
	@go get -u ./...
	@go mod tidy
	@echo "$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)"

##@ Docker Services

services-up: ## Start all Docker services (PostgreSQL, Redis, Kafka)
	@echo "$(COLOR_GREEN)Starting Docker services...$(COLOR_RESET)"
	@docker-compose up -d
	@echo "$(COLOR_GREEN)✓ Services started$(COLOR_RESET)"
	@make services-status

services-down: ## Stop all Docker services
	@echo "$(COLOR_GREEN)Stopping Docker services...$(COLOR_RESET)"
	@docker-compose down
	@echo "$(COLOR_GREEN)✓ Services stopped$(COLOR_RESET)"

services-restart: ## Restart all Docker services
	@echo "$(COLOR_GREEN)Restarting Docker services...$(COLOR_RESET)"
	@docker-compose restart
	@echo "$(COLOR_GREEN)✓ Services restarted$(COLOR_RESET)"

services-status: ## Show status of Docker services
	@echo "$(COLOR_BLUE)Docker Services Status:$(COLOR_RESET)"
	@docker-compose ps

services-logs: ## Tail logs from all services
	@docker-compose logs -f

services-clean: ## Stop and remove all containers and volumes
	@echo "$(COLOR_YELLOW)⚠️  This will delete all data!$(COLOR_RESET)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		echo "$(COLOR_GREEN)✓ Services cleaned$(COLOR_RESET)"; \
	fi

##@ Database

db-connect: ## Connect to PostgreSQL database
	@docker exec -it switchboard-postgres psql -U switchboard -d switchboard

db-setup: ## Initialize database schema
	@echo "$(COLOR_GREEN)Setting up database...$(COLOR_RESET)"
	@docker exec -i switchboard-postgres psql -U switchboard -d switchboard < schema.sql
	@echo "$(COLOR_GREEN)✓ Database schema created$(COLOR_RESET)"

db-test-data: ## Load test data (routes, services, plugins)
	@echo "$(COLOR_GREEN)Loading test data...$(COLOR_RESET)"
	@docker exec -i switchboard-postgres psql -U switchboard -d switchboard < tests/manual/setup_test_routes.sql
	@docker exec -i switchboard-postgres psql -U switchboard -d switchboard < tests/manual/setup_test_plugins.sql
	@echo "$(COLOR_GREEN)✓ Test data loaded$(COLOR_RESET)"

db-reset: ## Reset database (drop and recreate)
	@echo "$(COLOR_YELLOW)⚠️  This will delete all data!$(COLOR_RESET)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -i switchboard-postgres psql -U switchboard -d switchboard -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"; \
		make db-setup; \
		echo "$(COLOR_GREEN)✓ Database reset$(COLOR_RESET)"; \
	fi

db-backup: ## Backup database to file
	@echo "$(COLOR_GREEN)Backing up database...$(COLOR_RESET)"
	@docker exec switchboard-postgres pg_dump -U switchboard switchboard > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(COLOR_GREEN)✓ Database backed up$(COLOR_RESET)"

db-restore: ## Restore database from backup (usage: make db-restore FILE=backup.sql)
	@echo "$(COLOR_GREEN)Restoring database from $(FILE)...$(COLOR_RESET)"
	@docker exec -i switchboard-postgres psql -U switchboard -d switchboard < $(FILE)
	@echo "$(COLOR_GREEN)✓ Database restored$(COLOR_RESET)"

db-query: ## Run SQL query (usage: make db-query SQL="SELECT * FROM routes")
	@docker exec -it switchboard-postgres psql -U switchboard -d switchboard -c "$(SQL)"

##@ Redis

redis-connect: ## Connect to Redis CLI
	@docker exec -it switchboard-redis redis-cli

redis-flush: ## Flush all Redis data (WARNING: destructive)
	@echo "$(COLOR_YELLOW)⚠️  This will delete all Redis data!$(COLOR_RESET)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -it switchboard-redis redis-cli FLUSHALL; \
		echo "$(COLOR_GREEN)✓ Redis flushed$(COLOR_RESET)"; \
	fi

redis-monitor: ## Monitor Redis commands in real-time
	@docker exec -it switchboard-redis redis-cli MONITOR

redis-stats: ## Show Redis statistics
	@docker exec -it switchboard-redis redis-cli INFO stats

##@ Admin API

admin-install: ## Install Admin API dependencies
	@echo "$(COLOR_GREEN)Installing Admin API dependencies...$(COLOR_RESET)"
	@cd admin-api && python3 -m venv venv && . venv/bin/activate && pip install -r requirements.txt
	@echo "$(COLOR_GREEN)✓ Admin API dependencies installed$(COLOR_RESET)"

admin-run: ## Run Admin API
	@echo "$(COLOR_GREEN)Starting Admin API...$(COLOR_RESET)"
	@cd admin-api && . venv/bin/activate && uvicorn main:app --reload --host 0.0.0.0 --port 8000

admin-test: ## Test Admin API endpoints
	@echo "$(COLOR_GREEN)Testing Admin API...$(COLOR_RESET)"
	@./tests/manual/test_admin_api.sh

##@ Plugin Development

plugin-list: ## List all registered plugins
	@echo "$(COLOR_BLUE)Registered Plugins:$(COLOR_RESET)"
	@docker exec -i switchboard-postgres psql -U switchboard -d switchboard -c "SELECT name, scope, priority, enabled FROM plugins ORDER BY priority;"

plugin-add-logger: ## Add request logger plugin (global)
	@echo "$(COLOR_GREEN)Adding request logger plugin...$(COLOR_RESET)"
	@docker exec -i switchboard-postgres psql -U switchboard -d switchboard << EOF
	INSERT INTO plugins (name, scope, config, priority, enabled) \
	VALUES ('request-logger', 'global', '{"critical":false,"log_headers":true,"log_query_params":true,"excluded_paths":["/health","/ready"]}'::jsonb, 1, true) \
	ON CONFLICT (name, scope, COALESCE(service_id::text, ''), COALESCE(route_id::text, '')) DO NOTHING;
	EOF
	@echo "$(COLOR_GREEN)✓ Request logger plugin added$(COLOR_RESET)"

plugin-add-cors: ## Add CORS plugin (global)
	@echo "$(COLOR_GREEN)Adding CORS plugin...$(COLOR_RESET)"
	@docker exec -i switchboard-postgres psql -U switchboard -d switchboard << EOF
	INSERT INTO plugins (name, scope, config, priority, enabled) \
	VALUES ('cors', 'global', '{"critical":false,"allowed_origins":["*"],"allowed_methods":["GET","POST","PUT","DELETE","PATCH","OPTIONS"],"allowed_headers":["Content-Type","Authorization"],"allow_credentials":false,"max_age":86400}'::jsonb, 5, true) \
	ON CONFLICT (name, scope, COALESCE(service_id::text, ''), COALESCE(route_id::text, '')) DO NOTHING;
	EOF
	@echo "$(COLOR_GREEN)✓ CORS plugin added$(COLOR_RESET)"

plugin-reload: ## Trigger hot reload of plugins
	@echo "$(COLOR_GREEN)Triggering plugin reload...$(COLOR_RESET)"
	@docker exec -it switchboard-redis redis-cli PUBLISH gateway:config:changes '{"entity_type":"plugin","entity_id":"*","action":"reload"}'
	@echo "$(COLOR_GREEN)✓ Reload signal sent$(COLOR_RESET)"

##@ Docker

docker-build: ## Build Docker image
	@echo "$(COLOR_GREEN)Building Docker image...$(COLOR_RESET)"
	@docker build -t switchboard-gateway:$(VERSION) .
	@docker tag switchboard-gateway:$(VERSION) switchboard-gateway:latest
	@echo "$(COLOR_GREEN)✓ Docker image built: switchboard-gateway:$(VERSION)$(COLOR_RESET)"

docker-run: ## Run gateway in Docker container
	@echo "$(COLOR_GREEN)Running gateway in Docker...$(COLOR_RESET)"
	@docker run -d \
		--name switchboard-gateway \
		-p 8080:8080 \
		--env-file .env \
		switchboard-gateway:latest
	@echo "$(COLOR_GREEN)✓ Gateway container started$(COLOR_RESET)"

docker-stop: ## Stop gateway Docker container
	@docker stop switchboard-gateway || true
	@docker rm switchboard-gateway || true
	@echo "$(COLOR_GREEN)✓ Gateway container stopped$(COLOR_RESET)"

##@ Performance

benchmark: ## Run Go benchmarks
	@echo "$(COLOR_GREEN)Running benchmarks...$(COLOR_RESET)"
	@go test -bench=. -benchmem ./...

profile-cpu: ## Profile CPU usage
	@echo "$(COLOR_GREEN)Profiling CPU (30 seconds)...$(COLOR_RESET)"
	@go test -cpuprofile=cpu.prof -bench=. ./...
	@go tool pprof -http=:8081 cpu.prof

profile-mem: ## Profile memory usage
	@echo "$(COLOR_GREEN)Profiling memory...$(COLOR_RESET)"
	@go test -memprofile=mem.prof -bench=. ./...
	@go tool pprof -http=:8081 mem.prof

##@ Cleanup

clean: ## Remove build artifacts
	@echo "$(COLOR_GREEN)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f *.prof
	@echo "$(COLOR_GREEN)✓ Cleaned$(COLOR_RESET)"

clean-all: clean services-clean ## Clean everything (build + Docker)

##@ Quick Start

quickstart: services-up db-setup db-test-data build run ## Complete quickstart (setup + run)

reset: services-down clean services-up db-setup db-test-data ## Reset everything

##@ Version

version: ## Show version information
	@echo "$(COLOR_BOLD)Switchboard API Gateway$(COLOR_RESET)"
	@echo "Version:    $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"