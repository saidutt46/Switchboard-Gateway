# Switchboard Gateway

```
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   ███████╗██╗    ██╗██╗████████╗ ██████╗██╗  ██╗             ║
║   ██╔════╝██║    ██║██║╚══██╔══╝██╔════╝██║  ██║             ║
║   ███████╗██║ █╗ ██║██║   ██║   ██║     ███████║             ║
║   ╚════██║██║███╗██║██║   ██║   ██║     ██╔══██║             ║
║   ███████║╚███╔███╔╝██║   ██║   ╚██████╗██║  ██║             ║
║   ╚══════╝ ╚══╝╚══╝ ╚═╝   ╚═╝    ╚═════╝╚═╝  ╚═╝             ║
║                                                               ║
║              High-Performance API Gateway                     ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
```

**One gateway. All your APIs. Complete control.**

Switchboard is a lightweight, high-performance API Gateway that sits between your clients and microservices. Authenticate requests, enforce rate limits, cache responses, and route traffic—all with sub-millisecond overhead and zero downtime configuration updates.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/saidutt46/switchboard-gateway)](https://github.com/saidutt46/switchboard-gateway/releases)
[![CI](https://github.com/saidutt46/switchboard-gateway/actions/workflows/ci.yml/badge.svg)](https://github.com/saidutt46/switchboard-gateway/actions/workflows/ci.yml)

---

## Why Switchboard?

You have microservices. You need to:
- **Authenticate** every request before it hits your backend
- **Rate limit** abusive clients before they take you down
- **Cache** responses to reduce load and improve speed
- **Route** traffic to the right service based on path, method, or host
- **Change configuration** without restarting anything

Switchboard does all of this. Here's how it works.

---

## 🎬 See It In Action: A Complete Walkthrough

Let's say you're building an e-commerce platform with three microservices: Users, Products, and Orders. You want to expose them through a single API endpoint with authentication, rate limiting, and caching.

Here's exactly how you'd set that up with Switchboard.

### Step 0: Start Switchboard

```bash
# Clone and start everything
git clone https://github.com/saidutt46/switchboard-gateway.git
cd switchboard-gateway
docker-compose up -d
make run

# Gateway runs on :8080, Admin API on :8000
```

You now have two endpoints:
- `http://localhost:8080` — The Gateway (where clients send requests)
- `http://localhost:8000` — The Admin API (where you configure everything)

---

### Step 1: Register Your API Consumers

**Consumers are the applications or services that will call your API.** Think of them as your API clients—a mobile app, a partner integration, an internal service.

Every consumer gets tracked, authenticated, and can have individual rate limits.

```bash
# Register your mobile app as a consumer
curl -X POST http://localhost:8000/consumers \
  -H "Content-Type: application/json" \
  -d '{
    "username": "mobile-app-ios",
    "email": "ios-team@yourcompany.com"
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "username": "mobile-app-ios",
  "email": "ios-team@yourcompany.com",
  "created_at": "2024-01-15T10:30:00Z"
}
```

Now generate an API key for this consumer:

```bash
curl -X POST http://localhost:8000/consumers/550e8400-e29b-41d4-a716-446655440001/keys \
  -H "Content-Type: application/json" \
  -d '{"name": "production-key"}'
```

Response:
```json
{
  "id": "key-uuid-here",
  "name": "production-key",
  "key": "sk_live_a1b2c3d4e5f6..."
}
```

⚠️ **Save this key!** It's only shown once. This is what your mobile app will use to authenticate.

---

### Step 2: Define Your Backend Services

**Services are your actual microservices.** Tell Switchboard where they live.

```bash
# User Service
curl -X POST http://localhost:8000/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-service",
    "protocol": "http",
    "host": "users.internal",
    "port": 8001
  }'

# Product Service  
curl -X POST http://localhost:8000/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "product-service",
    "protocol": "http",
    "host": "products.internal",
    "port": 8002
  }'

# Order Service
curl -X POST http://localhost:8000/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "order-service",
    "protocol": "http",
    "host": "orders.internal",
    "port": 8003
  }'
```

You now have three services registered. But traffic can't reach them yet—we need routes.

---

### Step 3: Create Routes to Connect Everything

**Routes map incoming requests to backend services.** They define which paths go where.

```bash
# Route /api/users/* → user-service
curl -X POST http://localhost:8000/routes \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "<user-service-id>",
    "name": "user-routes",
    "paths": ["/api/users", "/api/users/:id"],
    "methods": ["GET", "POST", "PUT", "DELETE"]
  }'

# Route /api/products/* → product-service
curl -X POST http://localhost:8000/routes \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "<product-service-id>",
    "name": "product-routes",
    "paths": ["/api/products", "/api/products/:id", "/api/products/:id/reviews"],
    "methods": ["GET", "POST", "PUT", "DELETE"]
  }'

# Route /api/orders/* → order-service
curl -X POST http://localhost:8000/routes \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "<order-service-id>",
    "name": "order-routes",
    "paths": ["/api/orders", "/api/orders/:id"],
    "methods": ["GET", "POST"]
  }'
```

**Traffic now flows!** But anyone can access your API. Let's fix that.

```bash
# This works (but shouldn't without auth!)
curl http://localhost:8080/api/users
# → Returns user data
```

---

### Step 4: Lock It Down with Authentication

**Plugins add functionality to your gateway.** Let's require API keys for all requests.

```bash
curl -X POST http://localhost:8000/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-key-auth",
    "scope": "global",
    "config": {
      "key_header": "X-API-Key"
    },
    "enabled": true,
    "priority": 1
  }'
```

Now try accessing without a key:

```bash
curl http://localhost:8080/api/users
# → 401 Unauthorized: Missing API key
```

Use the key we generated earlier:

```bash
curl -H "X-API-Key: sk_live_a1b2c3d4e5f6..." http://localhost:8080/api/users
# → 200 OK: Returns user data
```

✅ **Authentication is live.** Only registered consumers with valid keys can access your API.

---

### Step 5: Protect Your Services with Rate Limiting

A single consumer shouldn't be able to hammer your API. Let's add rate limiting.

```bash
curl -X POST http://localhost:8000/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "name": "rate-limit",
    "scope": "global",
    "config": {
      "algorithm": "token-bucket",
      "limit": 100,
      "window": "1m",
      "identifier": "consumer_id"
    },
    "enabled": true,
    "priority": 10
  }'
```

Every consumer now gets **100 requests per minute**. The response headers tell them their quota:

```bash
curl -I -H "X-API-Key: sk_live_a1b2c3d4e5f6..." http://localhost:8080/api/users
```

```
HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1699123456
```

Exceed the limit and they get blocked:

```
HTTP/1.1 429 Too Many Requests
Retry-After: 45
```

✅ **Rate limiting is live.** Your backends are protected from abuse.

---

### Step 6: Speed Things Up with Caching

Product listings don't change every second. Let's cache them.

```bash
curl -X POST http://localhost:8000/plugins \
  -H "Content-Type: application/json" \
  -d '{
    "name": "cache",
    "scope": "service",
    "service_id": "<product-service-id>",
    "config": {
      "ttl_seconds": 300,
      "cache_methods": ["GET"],
      "bypass_paths": ["/api/products/flash-sale"]
    },
    "enabled": true,
    "priority": 5
  }'
```

First request hits the backend:

```bash
curl -I -H "X-API-Key: ..." http://localhost:8080/api/products
```
```
HTTP/1.1 200 OK
X-Cache: MISS
X-Response-Time: 150ms
```

Second request is served from cache:

```bash
curl -I -H "X-API-Key: ..." http://localhost:8080/api/products
```
```
HTTP/1.1 200 OK
X-Cache: HIT
X-Cache-TTL: 298
Age: 2
X-Response-Time: 17ms
```

✅ **Caching is live.** Product listings are now 9x faster.

---

### Step 7: The Complete Picture

Here's what you've built:

```
┌─────────────────────────────────────────────────────────────────┐
│                        YOUR CLIENTS                             │
│    Mobile App (iOS)  •  Mobile App (Android)  •  Partner API    │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ X-API-Key: sk_live_...
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SWITCHBOARD GATEWAY                          │
│                                                                 │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐        │
│  │   API Key    │ → │    Rate      │ → │    Cache     │ → Route│
│  │    Auth      │   │   Limiting   │   │   (Redis)    │        │
│  │  (priority 1)│   │ (priority 10)│   │ (priority 5) │        │
│  └──────────────┘   └──────────────┘   └──────────────┘        │
│                                                                 │
│  Routes:                                                        │
│    /api/users/*    → user-service:8001                         │
│    /api/products/* → product-service:8002 (cached!)            │
│    /api/orders/*   → order-service:8003                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                 ▼
       ┌────────────┐   ┌────────────┐   ┌────────────┐
       │   Users    │   │  Products  │   │   Orders   │
       │  Service   │   │  Service   │   │  Service   │
       └────────────┘   └────────────┘   └────────────┘
```

**Every request now:**
1. ✅ Authenticates via API key (rejects unauthorized requests)
2. ✅ Checks rate limit (protects backends from abuse)
3. ✅ Serves from cache if available (reduces latency)
4. ✅ Routes to the correct backend service
5. ✅ Tracks which consumer made the request

**And you can change any of this without restarting.** Hot reload updates configuration in under 200ms.

---

## 📊 Performance

Real benchmarks from load testing:

| Metric | Result |
|--------|--------|
| **Throughput** | 5,075 req/s sustained |
| **Latency (p50)** | 4.44ms |
| **Latency (p95)** | 18.71ms |
| **Gateway Overhead** | < 2ms |
| **Cache HIT** | 17ms |
| **Rate Limit Check** | < 1ms |
| **Hot Reload** | < 200ms |
| **Error Rate** | 0% |

---

## ✨ Features at a Glance

| Feature | Description |
|---------|-------------|
| 🚀 **High Performance** | Radix tree routing with O(log n) lookups |
| 👤 **Consumer Management** | Track and manage API clients |
| 🔐 **Authentication** | API Key and JWT support |
| ⚡ **Rate Limiting** | Token Bucket & Sliding Window algorithms |
| 💾 **Response Caching** | Redis-backed with smart cache keys |
| 🔄 **Hot Reload** | Zero-downtime config updates via Redis pub/sub |
| 🎛️ **Admin API** | Full REST API—no config files needed |
| 🔌 **Plugin System** | Extensible architecture for custom logic |
| 📊 **Structured Logging** | JSON logs with request tracing |

---

## 🔌 Available Plugins

| Plugin | What It Does | Scopes |
|--------|--------------|--------|
| `api-key-auth` | Validates API keys from header/query | Global, Service, Route |
| `jwt-auth` | Validates JWT tokens | Global, Service, Route |
| `rate-limit` | Enforces request quotas | Global, Service, Route, Consumer |
| `cache` | Caches responses in Redis | Global, Service, Route |
| `cors` | Adds CORS headers | Global, Service, Route |
| `request-logger` | Logs request/response details | Global |

**Plugin Priority:** Lower number = runs first. Auth should be 1, rate limiting 10, caching 5.

---

## 🏗️ Architecture

```
                                    ┌─────────────────┐
                                    │   Admin API     │
                                    │  (Python/Fast)  │
                                    │    :8000        │
                                    └────────┬────────┘
                                             │ CRUD Operations
                                             ▼
┌──────────┐    ┌─────────────────────────────────────────────────┐
│  Client  │───▶│           Switchboard Gateway (:8080)           │
└──────────┘    │                                                 │
                │  Request Flow:                                  │
                │  ┌────────┐  ┌────────┐  ┌────────┐  ┌───────┐ │
                │  │  Auth  │─▶│  Rate  │─▶│ Cache  │─▶│ Proxy │ │
                │  │ Plugin │  │ Limit  │  │ Plugin │  │       │ │
                │  └────────┘  └────────┘  └────────┘  └───────┘ │
                │                                                 │
                │  Config: Hot-reloaded from PostgreSQL + Redis   │
                └─────────────────────┬───────────────────────────┘
                                      │
              ┌───────────────────────┼───────────────────────┐
              ▼                       ▼                       ▼
       ┌────────────┐          ┌────────────┐          ┌────────────┐
       │  Service A │          │  Service B │          │  Service C │
       └────────────┘          └────────────┘          └────────────┘

Data Stores:
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ PostgreSQL  │    │    Redis    │    │    Kafka    │
│  • Config   │    │  • Cache    │    │   • Logs    │
│  • Routes   │    │  • Rate     │    │  • Events   │
│  • Consumers│    │    Limits   │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
```

---

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_PORT` | `8080` | Gateway listen port |
| `POSTGRES_DSN` | required | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection URL |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, console) |

### Example `.env`

```env
GATEWAY_PORT=8080
POSTGRES_DSN=postgres://switchboard:switchboard@localhost:5432/switchboard?sslmode=disable
REDIS_URL=redis://localhost:6379/0
LOG_LEVEL=info
ENVIRONMENT=production
```

---

## 🚀 Quick Start (TL;DR)

```bash
# 1. Start everything
git clone https://github.com/saidutt46/switchboard-gateway.git
cd switchboard-gateway
docker-compose up -d
make run

# 2. Create a consumer + API key
curl -X POST http://localhost:8000/consumers -H "Content-Type: application/json" \
  -d '{"username": "my-app"}'
# Note the consumer ID, then:
curl -X POST http://localhost:8000/consumers/<id>/keys -H "Content-Type: application/json" \
  -d '{"name": "my-key"}'
# Save the key!

# 3. Create a service
curl -X POST http://localhost:8000/services -H "Content-Type: application/json" \
  -d '{"name": "my-backend", "host": "localhost", "port": 3000}'

# 4. Create a route
curl -X POST http://localhost:8000/routes -H "Content-Type: application/json" \
  -d '{"service_id": "<service-id>", "paths": ["/api"], "methods": ["GET", "POST"]}'

# 5. Enable auth plugin
curl -X POST http://localhost:8000/plugins -H "Content-Type: application/json" \
  -d '{"name": "api-key-auth", "scope": "global", "config": {}, "enabled": true, "priority": 1}'

# 6. Test it!
curl -H "X-API-Key: <your-key>" http://localhost:8080/api
```

---

## 🧪 Testing

```bash
# Unit tests
make test

# Integration tests
make test-integration

# Load tests
k6 run tests/load/stress.js

# Manual test scripts
./tests/manual/test_admin_api.sh
./tests/manual/test_rate_limit.sh
./tests/manual/test_cache.sh
```

---

## 📁 Project Structure

```
switchboard-gateway/
├── cmd/gateway/           # Gateway entrypoint
├── internal/
│   ├── cache/             # Response caching
│   ├── config/            # Config & hot reload
│   ├── database/          # PostgreSQL repository
│   ├── plugin/            # Plugin system
│   │   └── builtin/       # Built-in plugins
│   ├── proxy/             # Reverse proxy
│   ├── ratelimit/         # Rate limiting
│   └── router/            # Radix tree routing
├── admin-api/             # Python Admin API
├── tests/                 # Test suites
├── docker-compose.yml     # Development stack
└── schema.sql             # Database schema
```

---

## 🗺️ Roadmap

**Completed:**
- [x] Core Gateway (Proxy, Routing, Config)
- [x] Admin API with full CRUD
- [x] Plugin System
- [x] API Key Authentication
- [x] JWT Authentication
- [x] Rate Limiting (Token Bucket, Sliding Window)
- [x] Response Caching
- [x] Hot Reload

**Coming Soon:**
- [ ] Load Balancing (Round Robin, Weighted, Least Connections)
- [ ] Circuit Breaker
- [ ] Active Health Checks
- [ ] Prometheus Metrics
- [ ] Kafka Request Logging
- [ ] WebSocket Support

---

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
git checkout -b feature/your-feature
make test
git commit -m "feat: your feature"
git push origin feature/your-feature
```

---

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

---

<p align="center">
  <b>Built for developers who want control without complexity.</b>
  <br><br>
  ⭐ Star this repo if Switchboard helps you!
</p>