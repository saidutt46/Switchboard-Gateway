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

## 🚀 Features

### ✅ Phase 3: Simple Reverse Proxy (Complete)
- HTTP reverse proxy with connection pooling
- Route matching (exact, parameters, wildcards)
- Request/response header forwarding
- Path parameter extraction
- Performance: 5,075 req/s sustained, p95 18.71ms

### ✅ Phase 5: Admin API & Hot Reload (Complete)
- **REST API for configuration management**
- **Zero-downtime config updates via Redis pub/sub**
- Auto-generated OpenAPI documentation
- 27 total API endpoints

#### Services Management
- Full CRUD operations for backend services
- Connection pooling configuration
- Load balancer type selection
- Service health tracking

#### Routes Management
- Dynamic route configuration
- Path-based routing (exact, parameters, wildcards)
- HTTP method filtering
- Host-based routing
- Hot reload support

#### Consumers & API Keys
- Consumer (API client) management
- Secure API key generation (SHA256 hashing)
- One-time key display
- Key enable/disable/revoke
- Key expiration support

#### Plugins System
- Global, service, route, and consumer-level plugins
- Priority-based execution order
- Available plugins:
  - Authentication: api-key-auth, jwt-auth, basic-auth
  - Traffic Control: rate-limit, ip-restriction
  - Performance: cache
  - Resilience: circuit-breaker, timeout
  - Transformation: cors, request-transform

#### Hot Reload
- Configuration changes apply in <200ms
- No gateway restart required
- Zero dropped requests
- All instances update simultaneously

---

## 📋 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+
- Python 3.11+ (for Admin API)
- PostgreSQL 15
- Redis 7
- Make

### 1. Start Infrastructure
```bash
# Start all services (PostgreSQL, Redis, Kafka, etc.)
make up

# Initialize database
make db-init

# Verify services are healthy
docker ps
```

### 2. Start Admin API
```bash
# Admin API runs in Docker
docker logs switchboard-admin-api -f

# Access OpenAPI docs
open http://localhost:8000/docs
```

### 3. Start Gateway
```bash
# Start gateway with hot reload
make run

# Should see:
# ✓ Database connection established
# ✓ Redis connection established
# ✓ Config watcher started - hot reload enabled! 🔥
# ✓ Gateway listening on :8080
```

### 4. Configure via Admin API
```bash
# Create a service
curl -X POST http://localhost:8000/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-api",
    "protocol": "http",
    "host": "api.example.com",
    "port": 8080
  }'

# Create a route
SERVICE_ID="<from-above-response>"
curl -X POST http://localhost:8000/routes \
  -H "Content-Type: application/json" \
  -d "{
    \"service_id\": \"$SERVICE_ID\",
    \"name\": \"my-route\",
    \"paths\": [\"/api/users\"],
    \"methods\": [\"GET\", \"POST\"]
  }"

# Gateway reloads automatically!
# Test immediately (no restart needed):
curl http://localhost:8080/api/users
```

---

## 🧪 Testing

### Run All Tests
```bash
# Unit tests
make test

# Integration tests
make test-integration

# Load tests
make load-test
```

### Manual Testing
```bash
# Comprehensive Admin API test suite
./tests/manual/test_admin_api.sh

# Test hot reload
./tests/manual/test_hot_reload.sh
```

---

## 📚 API Documentation

### Admin API
- **Base URL**: `http://localhost:8000`
- **Interactive Docs**: `http://localhost:8000/docs`
- **OpenAPI Spec**: `http://localhost:8000/openapi.json`

### Gateway
- **Base URL**: `http://localhost:8080`
- **Health Check**: `GET /health`
- **Ready Check**: `GET /ready`

### Endpoints Summary

**Services** (5 endpoints):
- `POST /services` - Create service
- `GET /services` - List services
- `GET /services/{id}` - Get service
- `PUT /services/{id}` - Update service
- `DELETE /services/{id}` - Delete service

**Routes** (6 endpoints):
- `POST /routes` - Create route
- `GET /routes` - List routes
- `GET /routes/{id}` - Get route
- `PUT /routes/{id}` - Update route
- `DELETE /routes/{id}` - Delete route
- `GET /routes/{id}/details` - Get route with service info

**Consumers** (5 endpoints):
- `POST /consumers` - Create consumer
- `GET /consumers` - List consumers
- `GET /consumers/{id}` - Get consumer
- `PUT /consumers/{id}` - Update consumer
- `DELETE /consumers/{id}` - Delete consumer

**API Keys** (5 endpoints):
- `POST /consumers/{id}/keys` - Generate API key
- `GET /consumers/{id}/keys` - List API keys
- `DELETE /consumers/{id}/keys/{key_id}` - Revoke key
- `PATCH /consumers/{id}/keys/{key_id}/disable` - Disable key
- `PATCH /consumers/{id}/keys/{key_id}/enable` - Enable key

**Plugins** (6 endpoints):
- `POST /plugins` - Create plugin
- `GET /plugins` - List plugins
- `GET /plugins/{id}` - Get plugin
- `PUT /plugins/{id}` - Update plugin
- `DELETE /plugins/{id}` - Delete plugin
- `GET /plugins/available` - List available plugin types

---

## 🔥 Hot Reload Example
```bash
# Terminal 1: Start gateway
make run

# Terminal 2: Create a route via Admin API
curl -X POST http://localhost:8000/routes \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "...",
    "paths": ["/api/new-endpoint"],
    "methods": ["GET"]
  }'

# Terminal 1 shows:
# INFO: Config change detected: route created
# INFO: Reloading routes from database
# INFO: Routes reloaded successfully

# Terminal 2: Test immediately (no restart!)
curl http://localhost:8080/api/new-endpoint
# ✓ Works!
```

---

## 📊 Performance

### Phase 3 Benchmarks
- **Throughput**: 5,075 req/s sustained
- **Latency (p50)**: 4.44ms
- **Latency (p95)**: 18.71ms
- **Gateway Overhead**: ~2ms
- **Error Rate**: 0%

### Hot Reload Performance
- **Propagation Time**: <200ms
- **Downtime**: 0ms
- **Dropped Requests**: 0

---

## 🛠️ Development

### Project Structure
```
switchboard-gateway/
├── cmd/gateway/          # Gateway entry point
│   └── main.go
├── internal/             # Internal packages
│   ├── config/          # Configuration & watcher
│   ├── database/        # Database models & repository
│   ├── gateway/         # Gateway core logic
│   ├── health/          # Health checks
│   ├── logging/         # Structured logging
│   ├── proxy/           # HTTP proxy
│   └── router/          # Route matching
├── admin-api/           # Admin REST API (Python/FastAPI)
│   ├── app.py           # Main application
│   ├── database.py      # SQLAlchemy setup
│   ├── models.py        # Database models
│   ├── schemas.py       # Pydantic schemas
│   ├── events.py        # Redis pub/sub
│   └── routers/         # API endpoints
├── tests/               # Test suites
│   ├── manual/          # Manual test scripts
│   └── load/            # k6 load tests
└── docker-compose.yml   # Infrastructure
```

### Makefile Commands
```bash
make up              # Start infrastructure
make down            # Stop infrastructure
make run             # Start gateway
make test            # Run tests
make load-test       # Run k6 load tests
make db-init         # Initialize database
make logs            # View logs
```

---

## 🎯 Roadmap

### ✅ Completed
- [x] Phase 1: Project Foundation
- [x] Phase 2: Database & Basic Server
- [x] Phase 3: Simple Reverse Proxy
- [x] Phase 5: Admin API & Hot Reload

### 🚧 In Progress
- [ ] Phase 7: Plugin System Implementation
- [ ] Phase 8: Authentication Plugin
- [ ] Phase 9: Rate Limiting Plugin

### 📅 Planned
- [ ] Phase 10: Response Caching
- [ ] Phase 11: Load Balancing
- [ ] Phase 12: Circuit Breaker
- [ ] Phase 13: Health Checks
- [ ] Phase 14: Request Logging (Kafka)
- [ ] Phase 15: Monitoring (Prometheus/Grafana)

---

## 📝 License

APACHE License - see LICENSE file for details

---

## 🙏 Acknowledgments

Built with:
- [Go](https://golang.org/) - Gateway core
- [FastAPI](https://fastapi.tiangolo.com/) - Admin API
- [PostgreSQL](https://www.postgresql.org/) - Configuration storage
- [Redis](https://redis.io/) - Caching & pub/sub
- [Kafka](https://kafka.apache.org/) - Event streaming