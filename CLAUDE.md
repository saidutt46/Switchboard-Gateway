# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Switchboard Gateway — a high-performance API Gateway written in Go with a Python FastAPI admin API and a React admin dashboard. Routes traffic, authenticates requests (API key / JWT), enforces rate limits, caches responses, and supports zero-downtime config updates via Redis pub/sub.

## Architecture

- **Gateway** (`cmd/gateway/`, `internal/`) — Go 1.21+. Reverse proxy with radix tree routing, plugin chain (auth → rate limit → cache → proxy), hot reload via Redis pub/sub.
- **Admin API** (`admin-api/`) — Python 3.11, FastAPI. CRUD REST API for services, routes, consumers, API keys, plugins. Publishes config changes to Redis.
- **Admin UI** (`admin-ui/`) — React 19, TypeScript, Vite, Tailwind CSS v4. Dashboard at port 4000 (Docker) or 5173 (dev). Proxies `/api` to admin API, `/gateway` to gateway.
- **Data stores** — PostgreSQL (config), Redis (cache, rate limiting, pub/sub), Kafka (logging).

## Workflow Rules

- **Always create a feature branch** with a descriptive prefix (`feature/`, `fix/`, `chore/`, `docs/`) before making changes. Never push or work directly on `main`.
- **Do not commit or push** unless the user has tested the changes manually or explicitly asks to commit.
- **Suggest commits at proper checkpoints** — don't batch all phases into one commit. Each logical unit of work should be a separate commit.
- **Branch protection is enabled on `main`** — all changes require a PR with at least 1 approving review.

## Commands

### Gateway (Go)

```bash
make build              # Build gateway binary
make run                # Build and run
make test               # Unit tests
make test-race          # Tests with race detection
make test-coverage      # Tests with coverage report
make fmt                # Format code
make vet                # Go vet
make lint               # golangci-lint
make ci                 # Full CI pipeline locally (fmt + vet + lint + security + test)
```

### Admin API (Python)

```bash
make admin-install      # Install dependencies (venv)
make admin-run          # Run with uvicorn (port 8000)
```

### Admin UI (React)

```bash
cd admin-ui
npm install             # Install dependencies
npm run dev             # Dev server with HMR (port 5173)
npm run build           # Production build
npm run test            # Unit tests (Vitest, 31 tests)
npm run test:e2e        # E2E tests (Playwright, 18 tests — requires services running)
```

Or via Makefile from project root:

```bash
make ui-install         # npm install
make ui-dev             # Dev server
make ui-build           # Production build
make ui-test            # Unit tests
make ui-test-e2e        # E2E tests
```

### Docker Services

```bash
docker compose up -d                  # Start full stack
docker compose up -d --build          # Rebuild and start
docker compose up -d --build admin-ui # Rebuild just the UI
docker compose down                   # Stop everything
docker compose ps                     # Status
docker compose logs -f gateway        # Tail gateway logs
```

Shell aliases (defined in ~/.zshrc):

```bash
csw         # cd to switchboard-gateway
sw-up       # Start switchboard services
sw-down     # Stop switchboard services
sw-ps       # Status
sw-logs     # Tail logs
```

### Database

```bash
make db-setup           # Initialize schema
make db-connect         # psql into the database
make db-reset           # Drop and recreate (destructive)
```

### Health Checks

```bash
curl http://localhost:8080/health   # Gateway
curl http://localhost:8000/health   # Admin API
curl http://localhost:4000          # Admin UI
```

## Ports

| Port | Service |
|------|---------|
| 4000 | Admin UI (Docker/nginx) |
| 5173 | Admin UI (Vite dev) |
| 5432 | PostgreSQL |
| 6379 | Redis |
| 8000 | Admin API |
| 8080 | Gateway |
| 8081 | Demo backend (httpbin) |
| 9092 | Kafka |

## Key Files

| File | Purpose |
|------|---------|
| `schema.sql` | Database schema + seed data |
| `docker-compose.yml` | Full development stack (8 services) |
| `admin-api/schemas.py` | Pydantic models — source of truth for API types |
| `admin-ui/src/api/types.ts` | TypeScript interfaces mirroring Pydantic schemas |
| `internal/plugin/builtin/` | Plugin implementations with full config options |
| `admin-api/routers/plugins.py` | `/plugins/available` endpoint (line ~270) |
| `.github/workflows/ci.yml` | CI pipeline |
| `.github/workflows/release.yml` | Release pipeline (builds binaries + 3 Docker images) |

## Plugin System

Plugins run in priority order (lower = first). Scopes: global, service, route, consumer. Built-in: `api-key-auth`, `jwt-auth`, `rate-limit`, `cache`, `cors`, `request-logger`. Rate-limit and cache plugins need `redis_url: "redis://redis:6379/0"` in their config when running in Docker.

## Commit Convention

```
feat:     New feature
fix:      Bug fix
chore:    Infrastructure, dependencies
docs:     Documentation
test:     Tests
refactor: Code restructuring
ci:       CI/CD changes
```
