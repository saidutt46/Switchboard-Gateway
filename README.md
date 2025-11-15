# Switchboard Gateway

A learning project building a production-grade API Gateway in Go.

## What This Is

Personal project to learn distributed systems, Go, and infrastructure patterns by building a real API Gateway from scratch.

**Current Status:** Phase 2 Complete - Basic server with database connectivity

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Make (optional, for convenience)

### Setup
```bash
# Clone the repo
git clone https://github.com/saidutt46/switchboard-gateway.git
cd switchboard-gateway

# Install dependencies
make setup

# Start infrastructure (PostgreSQL, Redis, Kafka)
make up

# Run the gateway
make run
```

The gateway will start on `http://localhost:8080`

### Test It
```bash
# Health check
curl http://localhost:8080/health

# Ready check
curl http://localhost:8080/ready

# Root endpoint
curl http://localhost:8080/
```

## What's Built So Far

### Phase 1: Project Foundation ✅
- Complete project structure
- Docker Compose setup (PostgreSQL, Redis, Kafka)
- Database schema with migrations
- Development tooling (Makefile, .gitignore, etc.)

### Phase 2: Database & Basic Server ✅
- PostgreSQL connection pool
- Repository pattern for data access
- Environment-based configuration (envconfig)
- Structured logging (zerolog)
- HTTP server with health checks
- Graceful shutdown

## Development

### Common Commands
```bash
# Start everything
make start

# Stop everything
make stop

# View logs
make logs

# Run tests
make test

# Build binary
make build

# Format code
make fmt

# Check all services
make verify
```

### Configuration

Configuration via environment variables. Copy `.env.example` to `.env`:
```bash
cp .env.example .env
```

Key variables:
- `POSTGRES_DSN` - Database connection string
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `GATEWAY_PORT` - HTTP server port (default: 8080)

### Project Structure
```
switchboard-gateway/
├── cmd/gateway/          # Main application
├── internal/             # Private packages
│   ├── config/          # Configuration
│   ├── database/        # Database layer
│   ├── health/          # Health checks
│   └── logging/         # Structured logging
├── tests/               # Tests
├── docker-compose.yml   # Local infrastructure
└── schema.sql          # Database schema
```

## Database

### Connect to PostgreSQL
```bash
make db-connect
```

### Tables

- `services` - Backend microservices
- `routes` - Request routing rules
- `consumers` - API clients
- `api_keys` - Authentication credentials
- `plugins` - Modular features
- `service_targets` - Load balancing targets

## Testing
```bash
# Run all tests
make test

# With coverage
make test-coverage

# With race detector
make test-race
```

## What's Next

Building incrementally following the [ACTION_ITEMS](./docs/ACTION_ITEMS.md) plan.

**Next up:** Phase 3 - Simple Reverse Proxy

## Tech Stack

- **Language:** Go 1.25
- **Database:** PostgreSQL 15
- **Cache:** Redis 7
- **Events:** Kafka 7.5
- **Logging:** zerolog
- **Config:** envconfig

## License

Apache 2.0

---

**Learning in progress** 🚀