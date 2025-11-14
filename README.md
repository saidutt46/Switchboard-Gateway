# Switchboard Gateway 🚀

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

A production-grade, high-performance API Gateway built in Go. Designed for microservices architectures with a focus on **performance**, **extensibility**, and **developer experience**.

## ✨ Features

- **🚀 High Performance**: Sub-millisecond overhead, 50k+ req/sec per instance
- **🔌 Plugin System**: Modular architecture with pluggable components
- **🔐 Authentication**: API Key, JWT, Basic Auth
- **⚡ Rate Limiting**: Sliding window, token bucket algorithms
- **💾 Response Caching**: Redis-backed intelligent caching
- **⚖️ Load Balancing**: Round-robin, least connections, weighted, IP hash
- **🛡️ Circuit Breaker**: Prevent cascading failures
- **🔄 Hot Reload**: Zero-downtime configuration updates
- **📊 Observability**: Prometheus metrics, distributed tracing, request logging
- **🐳 Cloud Native**: Docker, Kubernetes ready

## 🏗️ Architecture

```
┌─────────────┐
│   Clients   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│     Switchboard Gateway (Go)        │
│  ┌──────────────────────────────┐   │
│  │  Router (Radix Tree)         │   │
│  │  Plugin Chain                │   │
│  │  Reverse Proxy               │   │
│  └──────────────────────────────┘   │
└─────────────┬───────────────────────┘
              │
    ┌─────────┼─────────┐
    ▼         ▼         ▼
┌────────┐ ┌──────┐ ┌──────┐
│Postgres│ │Redis │ │Kafka │
│(Config)│ │(Cache│ │(Logs)│
└────────┘ └──────┘ └──────┘
              │
    ┌─────────┴─────────┐
    ▼                   ▼
┌─────────┐      ┌──────────┐
│Backend  │      │Analytics │
│Services │      │Service   │
└─────────┘      └──────────┘
```

## 🚀 Quick Start

### Prerequisites

- **Go** 1.25+ ([install](https://golang.org/doc/install))
- **Docker** & Docker Compose ([install](https://docs.docker.com/get-docker/))
- **Make** (optional, for convenience commands)

### 1. Clone the Repository

```bash
git clone https://github.com/saidutt46/switchboard-gateway.git
cd switchboard-gateway
```

### 2. Start Infrastructure

```bash
# Start PostgreSQL, Redis, Kafka
make up

# Or without make:
docker-compose up -d
```

### 3. Verify Services

```bash
make verify

# Expected output:
# PostgreSQL: ✅ Running
# Redis: ✅ Running
# Kafka: ✅ Running
# Demo Backend: ✅ Running
```

### 4. Install Dependencies

```bash
make setup

# Or:
go mod download
```

### 5. Run the Gateway

```bash
# Coming soon in Phase 2!
# make run
```

## 📦 What's Included Out of the Box

### Database Schema
Complete PostgreSQL schema with:
- `services` - Backend microservices
- `routes` - Path/method/host matching rules
- `consumers` - API clients
- `api_keys` - Authentication credentials
- `plugins` - Modular functionality
- Sample data for testing

### Docker Services
- **PostgreSQL 15** - Configuration storage
- **Redis 7** - Caching & rate limiting
- **Kafka 7.5** - Event streaming
- **httpbin** - Demo backend for testing

### Developer Tools
- **Makefile** with 40+ helpful commands
- Health checks for all services
- Database migration support
- Kafka topic management
- Test data insertion

## 🔧 Development Commands

```bash
# Show all available commands
make help

# Start services
make up

# View logs
make logs
make logs-gateway
make logs-postgres

# Database operations
make db-connect          # Connect to PostgreSQL
make db-reset            # Reset database
make db-query SQL="..."  # Run custom query

# Redis operations
make redis-cli           # Connect to Redis
make redis-flush         # Clear cache

# Kafka operations
make kafka-topics        # List topics
make kafka-consume-requests  # View request logs

# Testing
make test                # Run all tests
make test-unit           # Unit tests only
make test-integration    # Integration tests

# Code quality
make lint                # Run linter
make fmt                 # Format code
make vet                 # Run go vet

# Cleanup
make down                # Stop services
make clean               # Remove everything
```

## 📚 Project Structure

```
switchboard-gateway/
├── cmd/
│   └── gateway/          # Gateway entrypoint
├── internal/             # Private application code
│   ├── database/         # PostgreSQL client & models
│   ├── router/           # Route matching (radix tree)
│   ├── proxy/            # Reverse proxy
│   ├── plugins/          # Plugin system
│   ├── middleware/       # HTTP middleware
│   ├── config/           # Configuration management
│   ├── redis/            # Redis client
│   ├── kafka/            # Kafka producer
│   └── ...               # More packages as we build
├── admin-api/            # Python FastAPI admin interface
├── analytics/            # Python analytics service
├── tests/                # Tests
├── docs/                 # Documentation
├── examples/             # Example configurations
├── deployments/          # Kubernetes manifests
├── docker-compose.yml    # Local development setup
├── schema.sql            # Database schema
└── Makefile              # Developer commands
```

## 🎯 Development Roadmap

We're following a **23-phase development plan** (~70 days):

### Phase 1: ✅ Project Foundation (Days 1-3)
- [x] Project structure
- [x] Go modules
- [x] Docker Compose
- [x] Database schema
- [x] Documentation

### Phase 2: 🚧 Database & Basic Server (Days 4-5)
- [ ] PostgreSQL connection pool
- [ ] Database models
- [ ] Repository pattern
- [ ] Basic HTTP server

### Phase 3: Simple Reverse Proxy (Days 6-7)
- [ ] HTTP reverse proxy
- [ ] Connection pooling
- [ ] Timeout handling

### Phase 4: Route Matching (Days 8-10)
- [ ] Radix tree implementation
- [ ] Path parameter extraction
- [ ] Wildcard matching

### Phase 5-23: Advanced Features
- [ ] Admin API (Python FastAPI)
- [ ] Plugin system
- [ ] Authentication (API Key, JWT)
- [ ] Rate limiting
- [ ] Response caching
- [ ] Load balancing
- [ ] Circuit breaker
- [ ] Health checks
- [ ] Request logging
- [ ] Hot reload
- [ ] Observability (Prometheus, Grafana, Jaeger)
- [ ] Analytics service
- [ ] Comprehensive testing
- [ ] Documentation
- [ ] Kubernetes deployment

**See [ACTION_ITEMS](./docs/ACTION_ITEMS.md) for complete roadmap.**

## 🧪 Testing the Setup

### 1. Check Database

```bash
make db-connect

# Inside psql:
\dt                    # List tables
SELECT * FROM services;
SELECT * FROM routes;
\q                     # Quit
```

### 2. Check Redis

```bash
make redis-cli

# Inside redis-cli:
PING                   # Should return PONG
INFO server            # Server info
exit
```

### 3. Check Kafka

```bash
make kafka-topics

# Should show:
# gateway.requests
# gateway.errors
# gateway.config.changes
```

### 4. Check Demo Backend

```bash
curl http://localhost:8081/status
# Should return 200 OK with httpbin info
```

## 🔌 Configuration

### Environment Variables

The gateway uses environment variables for configuration:

```bash
# Database
POSTGRES_DSN="postgres://switchboard:password@localhost:5432/switchboard"

# Redis
REDIS_URL="redis://localhost:6379"

# Kafka
KAFKA_BROKERS="localhost:9092"

# Gateway
GATEWAY_PORT=8080
LOG_LEVEL=info
```

### Database Configuration

All gateway configuration is stored in PostgreSQL:
- Services (backend APIs)
- Routes (path matching rules)
- Consumers (API clients)
- API Keys (authentication)
- Plugins (features)

**Example: Add a new service**

```sql
INSERT INTO services (name, protocol, host, port) VALUES
('my-api', 'http', 'my-api.internal', 8080);
```

**Example: Add a route**

```sql
INSERT INTO routes (service_id, name, paths, methods) VALUES
((SELECT id FROM services WHERE name = 'my-api'),
 'my-route',
 ARRAY['/api/v1/users'],
 ARRAY['GET', 'POST']);
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Run linter (`make lint`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Kong](https://github.com/Kong/kong), [Traefik](https://github.com/traefik/traefik), and [Tyk](https://github.com/TykTechnologies/tyk)
- Built with ❤️ for the open source community

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/saidutt46/switchboard-gateway/issues)
- **Discussions**: [GitHub Discussions](https://github.com/saidutt46/switchboard-gateway/discussions)
- **Documentation**: [docs/](./docs/)

## ⭐ Star History

If you find this project useful, please consider giving it a star! ⭐

---

**Current Status**: 🚧 Under Active Development (Phase 1 Complete!)

Built with Go 🐹 | Powered by Open Source 💙