# Switchboard Gateway

A learning project building a production-grade API Gateway in Go.

## What This Is

Personal project to learn distributed systems, Go, and infrastructure patterns by building a real API Gateway from scratch.

**Current Status:** Phase 9 Complete - Rate Limiting with Token Bucket & Sliding Window algorithms

## Quick Start

### Prerequisites

- Go 1.21+
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

# Test rate limiting
for i in {1..15}; do curl -I http://localhost:8080/test/get 2>/dev/null | grep -E "HTTP|X-RateLimit"; done
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

### Phase 3: Simple Reverse Proxy ✅
- HTTP reverse proxy with connection pooling
- Route matching (exact, parameters, wildcards)
- Request/response header forwarding
- Path parameter extraction
- Performance: 5,075 req/s sustained, p95 18.71ms

### Phase 5: Admin API & Hot Reload ✅
- **REST API for configuration management**
- **Zero-downtime config updates via Redis pub/sub**
- Auto-generated OpenAPI documentation
- 27 total API endpoints
- Hot reload: <200ms propagation time

### Phase 9: Rate Limiting ✅ NEW
- **Two algorithms**: Token Bucket (burst-friendly) & Sliding Window (strict)
- **Identifier hierarchy**: consumer_id > api_key > ip_address
- **Standard headers**: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset
- **429 responses** with Retry-After header
- **Distributed state** via Redis (multi-instance support)
- **Hot reload** configuration changes
- **Production tested**: k6 load tests with 10+ concurrent users

---

## 🚀 Features

### 🔐 Authentication
Multiple authentication methods with flexible configuration:
#### API Key Authentication
- SHA256-hashed keys for security
- Support for header (`X-API-Key`) or query parameter (`apikey`)
- Redis caching (5-minute TTL)
- Consumer identification and context injection
- Configurable bypass paths
**Setup:**
```bash
# Create consumer
curl -X POST http://localhost:3000/api/consumers \
  -H "Content-Type: application/json" \
  -d '{"username": "partner-corp", "email": "api@partner.com"}'
# Generate API key (returned on consumer creation)
# Store the key hash in database
# Configure plugin
curl -X POST http://localhost:3000/api/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-key-auth",
    "scope": "global",
    "config": {
      "key_names": ["X-API-Key", "apikey"],
      "key_in_header": true,
      "key_in_query": true,
      "cache_ttl_seconds": 300
    },
    "priority": 5,
    "enabled": true
  }'
# Test
curl -H "X-API-Key: your-key-here" http://localhost:8080/api/endpoint
JWT Authentication
  • Token validation with configurable algorithms (HS256)
  • Claims validation (exp, nbf, iss, aud)
  • Support for Authorization Bearer header or cookies
  • Redis caching with TTL respecting token expiration
  • User context injection
Setup:
# Configure plugin
curl -X POST http://localhost:3000/api/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "name": "jwt-auth",
    "scope": "global",
    "config": {
      "secret_key": "your-secret-key",
      "algorithm": "HS256",
      "issuer": "your-issuer",
      "audience": "your-audience",
      "cache_ttl_seconds": 300
    },
    "priority": 6,
    "enabled": true
  }'
# Test
curl -H "Authorization: Bearer eyJhbGc..." http://localhost:8080/api/endpoint
Authentication Scopes
  • Global: All routes require authentication
  • Service: All routes of a specific service require authentication
  • Route: Specific routes require authentication (most flexible)
Example - Mixed Auth:
# Public documentation - no auth
# (Don't add auth plugin to /docs routes)
# User APIs - JWT required
# Add jwt-auth plugin with route scope to /api/users/* routes
# Partner webhooks - API Key required  
# Add api-key-auth plugin with route scope to /webhooks/* routes
```

### Rate Limiting (Phase 9)

#### Algorithms

| Algorithm | Best For | Characteristics |
|-----------|----------|-----------------|
| **Token Bucket** | Public APIs, Developer sandboxes | Gradual refill, allows bursts, forgiving |
| **Sliding Window** | Paid APIs, SLA enforcement | Strict limits, no bursts, compliance-ready |

#### Quick Example

```bash
# Configure global rate limit (10 requests/minute)
curl -X POST http://localhost:8000/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rate-limit",
    "scope": "global",
    "config": {
      "algorithm": "token-bucket",
      "limit": 10,
      "window": "1m"
    },
    "enabled": true,
    "priority": 10
  }'

# Test it
for i in {1..12}; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/test/get)
  echo "Request $i: HTTP $STATUS"
done

# Output:
# Request 1-10: HTTP 200
# Request 11-12: HTTP 429
```

#### Configuration Options

```json
{
  "algorithm": "token-bucket",        // or "sliding-window"
  "limit": 1000,                      // requests per window
  "window": "1m",                     // 1s, 1m, 1h, 24h
  "identifier": "auto",               // auto, consumer_id, api_key, ip
  "headers": true,                    // add X-RateLimit-* headers
  "response_code": 429,               // HTTP status when limited
  "response_message": "Rate limit exceeded"
}
```

#### Response Headers

```
X-RateLimit-Limit: 10                # Max requests allowed
X-RateLimit-Remaining: 7             # Requests remaining
X-RateLimit-Reset: 1734567890        # Unix timestamp when limit resets
Retry-After: 45                      # Seconds to wait (on 429)
```

#### Identifier Strategy

Rate limits are enforced using a **priority hierarchy**:

1. **consumer_id** (from auth plugin) - Most specific
2. **api_key** (from X-API-Key header) - Per API key
3. **ip** (from X-Forwarded-For) - Fallback

**Auto mode** tries each in order until one is found.

#### Scopes

```sql
-- Global: All requests
INSERT INTO plugins (name, scope, config, enabled) 
VALUES ('rate-limit', 'global', '{"limit": 1000, "window": "1m"}', true);

-- Service: All routes for a service
INSERT INTO plugins (name, scope, service_id, config, enabled) 
VALUES ('rate-limit', 'service', '<service-id>', '{"limit": 5000, "window": "1h"}', true);

-- Route: Specific route only
INSERT INTO plugins (name, scope, route_id, config, enabled) 
VALUES ('rate-limit', 'route', '<route-id>', '{"limit": 100, "window": "1m"}', true);
```

#### Performance

**Load Test Results** (k6):
- ✅ **Burst Test**: 10 requests allowed, 5 denied (100% accurate)
- ✅ **Sustained Load**: 38 requests over 3.5min (Token Bucket refill working)
- ✅ **Concurrent Users**: Handles 10+ users correctly
- ✅ **Headers**: 100% of responses include rate limit headers
- ✅ **Latency**: P95 < 1.5s (mostly upstream)

### Admin API & Hot Reload

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
  - **Rate Limiting**: Token Bucket & Sliding Window
  - CORS with preflight support
  - Request Logger with structured logging

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

# Add rate limiting
curl -X POST http://localhost:8000/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rate-limit",
    "scope": "global",
    "config": {
      "algorithm": "token-bucket",
      "limit": 100,
      "window": "1m"
    },
    "enabled": true,
    "priority": 10
  }'

# Gateway reloads automatically!
# Test immediately (no restart needed):
curl -I http://localhost:8080/api/users
```

---

## 🧪 Testing

### Run All Tests
```bash
# Unit tests
make test

# Integration tests
make test-integration

# Load tests (rate limiting)
k6 run tests/load/rate_limit_burst_simple.js
k6 run tests/load/rate_limit_realistic.js
```

### Manual Testing
```bash
# Comprehensive Admin API test suite
./tests/manual/test_admin_api.sh

# Test hot reload
./tests/manual/test_hot_reload.sh

# Test rate limiting
./tests/manual/test_rate_limit.sh
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

**Plugins** (6 endpoints):
- `POST /plugins` - Create plugin
- `GET /plugins` - List plugins
- `GET /plugins/{id}` - Get plugin
- `PUT /plugins/{id}` - Update plugin
- `DELETE /plugins/{id}` - Delete plugin
- `GET /plugins/available` - List available plugin types

---

## 📊 Performance

### Phase 3 Benchmarks
- **Throughput**: 5,075 req/s sustained
- **Latency (p50)**: 4.44ms
- **Latency (p95)**: 18.71ms
- **Gateway Overhead**: ~2ms
- **Error Rate**: 0%

### Phase 9 Benchmarks (Rate Limiting)
- **Burst Test**: 10/15 requests allowed (67% - Token Bucket)
- **Sustained Load**: 38 requests over 3.5min (gradual refill working)
- **Concurrent Users**: 10+ users handled correctly
- **Latency Impact**: P95 < 1.5s (minimal overhead)
- **Accuracy**: 100% enforcement

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
│   ├── plugin/          # Plugin system
│   │   └── builtin/    # Built-in plugins (rate-limit, cors, etc.)
│   ├── proxy/           # HTTP proxy
│   ├── ratelimit/       # Rate limiting algorithms
│   │   ├── token_bucket.go
│   │   ├── sliding_window.go
│   │   └── redis_store.go
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
│       ├── rate_limit_burst_simple.js
│       ├── rate_limit_realistic.js
│       └── rate_limit_token_bucket.js
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
- [x] Phase 9: Rate Limiting (Token Bucket & Sliding Window)

### 🚧 Planned
- [ ] Phase 8: Authentication Plugin
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
- [k6](https://k6.io/) - Load testing

**gvs46**