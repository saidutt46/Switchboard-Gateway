# Changelog

All notable changes to Switchboard API Gateway will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Prometheus metrics endpoints
- Circuit breaker plugin
- WebSocket support

### Changed
- Improved rate limiting performance

### Fixed
- Memory leak in connection pooling

---

## [0.7.0] - 2024-11-25

### Added
- **2-Package Docker Architecture:** Separate Gateway (Go) and Admin API (Python) images
- **CI/CD Pipeline:** GitHub Actions workflow for automated testing and releases
- **Release Automation:** Multi-platform binary builds (Linux, macOS, Windows - amd64/arm64)
- **Docker Registry:** Published images to ghcr.io
- **Comprehensive Makefile:** 60+ targets for development, testing, deployment
- **Health Checks:** Proper healthcheck endpoints with dependency verification
- **Enhanced Dockerfile:** Multi-stage build with security best practices

### Changed
- Dockerfile renamed from `Dockerfile.gateway` to `Dockerfile.Gateway`
- Admin API now includes curl for healthcheck
- Improved docker-compose.yml with proper service dependencies

### Fixed
- Docker image naming (lowercase repository names for registry compatibility)
- Admin API healthcheck failure (missing curl)
- golangci-lint configuration (v2 schema)
- Race condition in rate limiting tests

### Security
- Added gosec security scanning to CI pipeline
- Implemented non-root user in Docker images
- Added OCI standard labels to images

---

## [0.6.0] - 2024-11-20

### Added
- **Plugin System:** Priority-based plugin execution
- **Rate Limiting:** Token bucket and sliding window algorithms
- **Response Caching:** Redis-backed caching with cache key generation
- **Authentication:** API Key and JWT auth plugins
- **CORS Plugin:** Configurable CORS headers

### Changed
- Refactored router to use radix tree (O(log n) lookups)
- Improved hot reload mechanism via Redis pub/sub

### Fixed
- Memory leak in plugin chain execution
- Race condition in configuration reload

---

## [0.5.0] - 2024-11-10

### Added
- **Hot Reload:** Zero-downtime configuration updates via Redis pub/sub
- **Admin API:** FastAPI-based REST API for configuration management
- **Database Repository:** PostgreSQL repository pattern implementation

### Changed
- Configuration moved from JSON files to PostgreSQL
- In-memory configuration with hot reload

---

## [0.4.0] - 2024-11-01

### Added
- **Reverse Proxy:** HTTP reverse proxy with connection pooling
- **Route Matching:** Path, method, and host-based routing
- **Path Parameters:** Support for :id style parameters
- **Wildcard Routes:** Support for * wildcard matching

---

## [0.3.0] - 2024-10-20

### Added
- **PostgreSQL Integration:** Schema and models
- **Redis Integration:** Connection pooling
- **Structured Logging:** zerolog implementation
- **Graceful Shutdown:** Signal handling

---

## [0.2.0] - 2024-10-10

### Added
- Basic HTTP server
- Configuration loading
- Health check endpoint

---

## [0.1.0] - 2024-10-01

### Added
- Initial project structure
- Go modules setup
- Basic CLI

---

## Release Notes

### How to Upgrade

#### From 0.6.x to 0.7.0

**Docker Deployment:**
```bash
# Pull new images
docker pull ghcr.io/saidutt46/switchboard-gateway/gateway:v0.7.0
docker pull ghcr.io/saidutt46/switchboard-gateway/admin-api:v0.7.0

# Update docker-compose.yml to use new images
docker-compose up -d
```

**Binary Deployment:**
```bash
# Download from GitHub releases
wget https://github.com/saidutt46/Switchboard-Gateway/releases/download/v0.7.0/gateway-linux-amd64

# Replace old binary
sudo systemctl stop switchboard-gateway
sudo mv gateway-linux-amd64 /usr/local/bin/switchboard-gateway
sudo systemctl start switchboard-gateway
```

**Breaking Changes:** None

**Database Migrations:** None required

---

[Unreleased]: https://github.com/saidutt46/Switchboard-Gateway/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/saidutt46/Switchboard-Gateway/releases/tag/v0.7.0
[0.6.0]: https://github.com/saidutt46/Switchboard-Gateway/releases/tag/v0.6.0