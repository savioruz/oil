# Oil

A production-grade Go REST API boilerplate with chi router, PostgreSQL, Redis, Kafka, and S3 storage. Designed for real-world services with clean architecture, compile-time dependency injection, and full observability out of the box.

## Tech Stack

| Category | Technology |
|---|---|
| Language | Go 1.26 |
| HTTP Router | chi v5 |
| Database | PostgreSQL with sqlx (read/write separation) |
| Cache | Redis / DragonflyDB |
| Message Queue | Kafka |
| Object Storage | AWS S3 (S3-compatible) |
| Authentication | JWT (access + refresh tokens, Redis-backed revocation) |
| Authorization | Embedded RBAC via JSON permissions |
| Dependency Injection | Google Wire (compile-time) |
| Observability | OpenTelemetry + Jaeger (OTLP/gRPC) |
| API Documentation | Swagger / OpenAPI (swaggo) |
| Logging | zerolog (structured JSON) |
| Testing | testify + sqlmock + uber/mock |
| Hot Reload | air |

## Prerequisites

- Go 1.26+
- Make
- Docker & Docker Compose

## Getting Started

### 1. Clone and setup

```bash
git clone https://github.com/savioruz/oil.git
cd oil
```

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env` with your settings. All variables are documented with defaults in `.env.example`.

### 3. Start infrastructure

```bash
docker-compose up -d
```

This starts PostgreSQL, Redis, Jaeger, Kafka, and Kafka UI locally.

| Service | Port |
|---|---|
| PostgreSQL | 5432 |
| Redis | 6379 |
| Jaeger UI | 16686 |
| Kafka | 9092 |
| Kafka UI | 9080 |

### 4. Run migrations

```bash
make migrate.up
```

### 5. Start the server

```bash
make dev   # development mode with hot reload
```

The API is available at `http://localhost:8080`.  
Swagger UI is available at `http://localhost:8080/swagger/index.html` (development only).

## Make Commands

| Command | Description |
|---|---|
| `make dev` | Run with hot reload (air) |
| `make run` | Run the application |
| `make build` | Build the binary (`engine`) |
| `make test` | Run all tests |
| `make coverage` | Run tests and generate coverage report |
| `make coverage.view` | Open coverage report in browser |
| `make lint` | Run golangci-lint with auto-fix |
| `make generate` | Generate Swagger docs and Wire DI code |
| `make generate.mock` | Generate mocks with mockgen |
| `make migrate.up` | Run all pending migrations |
| `make migrate.down` | Roll back one migration |
| `make migrate.step-up` | Apply one migration |
| `make migrate.drop` | Drop all migrations |
| `make migrate.create name=<name>` | Create a new migration file |
| `make domains name=<name>` | Scaffold a new domain (model, repository, service) |
| `make docker.build` | Build the Docker image |
| `make docker.start` | Start Docker containers |
| `make docker.stop` | Stop Docker containers |
| `make clean` | Remove binary, mocks, and generated files |

## Project Structure

```
oil/
├── api/                        # Vercel serverless entry point
├── cmd/
│   ├── app/                    # Main server binary
│   └── migrate/                # Standalone migration CLI
├── config/                     # Config struct (envconfig + godotenv)
├── deployments/                # Production Docker Compose files
├── di/                         # Google Wire wiring (wire.go + wire_gen.go)
├── docs/                       # Generated Swagger output
├── helper/                     # Migration runner helpers
├── infras/                     # Infrastructure adapters
│   ├── jwt/                    # JWT sign, verify, revoke, refresh
│   ├── kafka/                  # Kafka producer + consumer
│   ├── otel/                   # OpenTelemetry tracer + scope abstraction
│   ├── postgres/               # Read/write PostgreSQL connections
│   ├── redis/                  # Redis client setup
│   └── s3/                     # S3 upload, presign, delete
├── internal/
│   ├── domains/                # Business logic by domain
│   │   ├── auth/               # Register, login, refresh, change-password
│   │   ├── gallery/            # Image albums backed by S3
│   │   ├── todo/               # Todo CRUD (example domain)
│   │   └── user/               # User model + repository
│   └── handlers/               # HTTP handlers (one per domain)
├── migrations/postgres/        # Sequential SQL migration files
├── permissions/                # Embedded RBAC permissions JSON
├── shared/                     # Reusable packages
│   ├── cache/                  # Redis cache abstraction
│   ├── constant/               # App-wide constants and context keys
│   ├── cookie/                 # Refresh token cookie helpers
│   ├── dto/                    # QueryParams and filter builders
│   ├── errkey/                 # Typed error keys
│   ├── failure/                # Typed HTTP error types
│   ├── logger/                 # zerolog initialization
│   ├── model/                  # Shared audit metadata struct
│   ├── password/               # bcrypt hash + verify
│   ├── repository/             # Generic repository[T] with reflection-driven SQL
│   ├── timezone/               # UTC time helpers
│   └── validator/              # JSON decode + go-playground/validator
└── transport/http/
    ├── http.go                 # Server setup and graceful shutdown
    ├── middleware/             # OTel tracing, JWT auth, RBAC, rate limiting
    ├── response/               # Typed JSON response helpers
    └── router/                 # Route registration
```

## Architecture

Oil follows a strict three-layer architecture within each domain:

```
Handler (transport) → Service (domain logic) → Repository (data access)
```

Each layer depends only on interfaces, making every layer independently testable and mockable. Google Wire generates the full dependency graph at compile time with no runtime reflection.

### Generic Repository

`shared/repository/Repository[T]` uses Go generics and `reflect` to auto-build SQL from struct tags. All domains get `Insert`, `Get`, `GetAll`, `Count`, `Update`, `Delete`, and transaction variants with zero boilerplate SQL. Read operations are always routed to the read replica and writes to the primary.

### Adding a New Domain

```bash
make domains name=product
```

This scaffolds `internal/domains/product/` with the model, repository, and service packages. Wire up the new providers in `di/wire.go` and register your handler in `transport/http/router/`.

## Authentication

The API uses JWT access + refresh token pairs with Redis-backed token lifecycle management:

- **Access token**: 15 minutes (configurable)
- **Refresh token**: 7 days (configurable)
- **Revocation**: tokens are blacklisted in Redis on logout; all sessions can be revoked at once
- **Rotation**: refresh tokens are single-use; a new pair is issued on every refresh

Send the access token in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

The refresh token is delivered and accepted via an `HttpOnly` cookie.

## Authorization

RBAC is driven by `permissions/permissions.json`, embedded into the binary at build time. Each route entry declares its allowed roles and whether to skip authentication entirely. The auth middleware evaluates role membership on every request with no database lookups.

## Rate Limiting

The rate limiter uses Redis as a sliding-window counter keyed by client IP + user agent. Response headers `X-Rate-Limit`, `X-Rate-Limit-Remaining`, and `X-Rate-Limit-Window` are set on every request. If Redis is unavailable, requests are allowed through (fail-open).

## Observability

Every handler, service, and repository creates an OpenTelemetry span via the `otel.Scope` abstraction. Traces are exported to Jaeger over OTLP/gRPC. The Jaeger UI is available at `http://localhost:16686` when running the dev stack.

## Deployment

### Docker (binary)

Multi-stage build producing a minimal `scratch` image:

```bash
make build
docker build -t oil:latest .
```

Production compose (app only, references external services):

```bash
docker-compose -f deployments/app.yml up -d
```

### Vercel (serverless)

The same codebase deploys to Vercel without changes. `api/index.go` exposes the standard `Handler(w, r)` function and `vercel.json` routes all traffic through it.

## License

[MIT License](LICENSE)
