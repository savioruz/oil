# Oil

A production-grade Go REST API boilerplate with chi router, PostgreSQL, Redis, Kafka, and S3 storage. Designed for real-world services with clean architecture, compile-time dependency injection, and full observability out of the box.

## Tech Stack

| Category | Technology |
|---|---|
| Language | Go 1.27 |
| HTTP Router | chi v5 |
| Database | PostgreSQL with sqlx (read/write separation) |
| Cache | Redis / DragonflyDB |
| Message Queue | Kafka |
| Object Storage | AWS S3 (S3-compatible) |
| Authentication | External JWT validation (Better Auth) with lazy user profile sync |
| Authorization | Embedded RBAC via JSON permissions |
| Dependency Injection | Google Wire (compile-time) |
| Observability | OpenTelemetry + Jaeger (OTLP/gRPC) |
| API Documentation | Swagger / OpenAPI (swaggo) |
| Logging | zerolog (structured JSON) |
| Testing | testify + sqlmock + uber/mock |
| Hot Reload | air |

## Prerequisites

- Go 1.27+
- Node 22+ (for upgrading OpenAPI v2 to v3.1)
- Make
- Docker & Docker Compose

## Using this as a template

Oil is designed to be bootstrapped into a new, independent project without
forking or cloning. Two supported paths:

### Option 1: GitHub "Use this template"

Open [github.com/savioruz/oil](https://github.com/savioruz/oil), click
**Use this template** → **Create a new repository**, and pick a name. You get
a new repo with the same file structure and a clean git history (no fork
link, no upstream).

### Option 2: `gonew` (terminal / scriptable)

[gonew](https://pkg.go.dev/golang.org/x/tools/cmd/gonew) copies this module
and rewrites every `github.com/savioruz/oil/...` import path to your new
module path automatically:

```sh
go install golang.org/x/tools/cmd/gonew@latest
gonew github.com/savioruz/oil@latest github.com/<user>/<new-project>
cd new-project
```

The result has no oil git history, a correct `go.mod` module path, and all
imports already rewritten — no manual find-and-replace.

### Post-bootstrap checklist

1. **Generate project files**: `make generate` (regenerates `di/wire_gen.go`,
   `permissions/permissions.json`, and swagger docs) and `make generate.mock`
   (mocks). `make test` runs both as prerequisites.
2. **README.md**: replace the project name, tagline, and description.
3. **`.env.example`**: update secrets, URLs, and service names for your
   environment.
4. **CI workflows** (`.github/workflows/`): rename jobs/steps if they
   reference "oil" explicitly.
5. **LICENSE**: update the copyright holder if you keep the license.
6. **Commit the bootstrap** (including the generated files above) and push.

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

### 3. Setup RBAC
```bash
cp permissions/permissions.example.json permissions/permissions.json
```

Edit `permissions/permissions.json` with your handler spec.

### 4. Configure Kafka (if needed)

Kafka is optional. To enable it, configure SASL authentication:

```bash
cp kafka_jaas.conf.example kafka_jaas.conf
```

Edit `kafka_jaas.conf` with your credentials. **This file is in `.gitignore` — never commit it.**

### 5. Start infrastructure

```bash
docker-compose up -d
```

This starts PostgreSQL, Redis, Jaeger, Kafka, and Kafka UI locally.

| Service | Port |
|---|---|
| PostgreSQL | 5432 |
| Redis | 6379 |
| Jaeger (Badger) | 16686 |
| Kafka | 9092 |
| Kafka UI | 9080 |

### 6. Run migrations

```bash
make migrate.up
```

### 5. Start the server

```bash
make dev   # development mode with hot reload
```

The API is available at `http://localhost:8080`.  
Scalar Documentation is available at `http://localhost:8080/docs` (development only).

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
| `make modules name=<name>` | Scaffold a new module (model, repository, service) |
| `make docker.build` | Build the Docker image |
| `make docker.start` | Start Docker containers |
| `make docker.stop` | Stop Docker containers |
| `make clean` | Remove binary, mocks, and generated files |

## Project Structure

```
oil/
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
│   ├── modules/                # Business logic by module
│   │   ├── gallery/            # Image albums backed by S3
│   │   ├── todo/               # Todo CRUD (example module)
│   │   └── user/               # User model + repository
│   └── handlers/               # HTTP handlers (one per module)
├── migrations/postgres/        # Sequential SQL migration files
├── permissions/                # Embedded RBAC permissions JSON
├── shared/                     # Reusable packages
│   ├── cache/                  # Redis cache abstraction
│   ├── constant/               # App-wide constants and context keys
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

Oil follows a strict three-layer architecture within each module:

```
Handler (transport) → Service (module logic) → Repository (data access)
```

Each layer depends only on interfaces, making every layer independently testable and mockable. Google Wire generates the full dependency graph at compile time with no runtime reflection.

### Generic Repository

`shared/repository/Repository[T]` uses Go generics and `reflect` to auto-build SQL from struct tags. All modules get `Insert`, `Get`, `GetAll`, `Count`, `Update`, `Delete`, and transaction variants with zero boilerplate SQL. Read operations are always routed to the read replica and writes to the primary.

### Adding a New Module

```bash
make modules name=product
```

This scaffolds `internal/modules/product/` with the model, repository, and service packages. Wire up the new providers in `di/wire.go` and register your handler in `transport/http/router/`.

## Authentication

The API validates JWTs from an external auth service (Better Auth) via JWKS. No local token management.

- **Algorithm**: EdDSA/Ed25519 with OKP key type
- **JWKS endpoint**: fetched once at startup from `AUTH_SERVICE_URL`
- **Lazy sync**: user profile is created/linked on first API call

Send the JWT in the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

## Authorization

RBAC is driven by `permissions/permissions.json`, embedded into the binary at build time. Each route entry declares its allowed roles and whether to skip authentication entirely. The auth middleware evaluates role membership on every request with no database lookups.

## Rate Limiting

The rate limiter uses Redis as a sliding-window counter keyed by client IP + user agent. Response headers `X-Rate-Limit`, `X-Rate-Limit-Remaining`, and `X-Rate-Limit-Window` are set on every request. If Redis is unavailable, requests are allowed through (fail-open).

## Observability

Every handler, service, and repository creates an OpenTelemetry span via the `otel.Scope` abstraction. Traces are exported to Jaeger over OTLP/gRPC. The Jaeger UI is available at `http://localhost:16686` when running the dev stack.

## API Documentation

The project uses [Scalar](https://scalar.com/) for API documentation, generated from OpenAPI specs.

### Requirements

Install the Scalar CLI for OpenAPI 3.1 generation:

```bash
npm install -g @scalar/cli
```

### Generation

```bash
make generate
```

This generates:
- `docs/swagger.json` - Swagger 2.0 (OpenAPI v2)
- `docs/openapi.json` - OpenAPI 3.1
- `permissions/permissions.json` - RBAC permissions (initialize if not exists)

### Viewing Docs

In development, visit `http://localhost:8080/docs` to view the interactive API documentation.

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

## License

[MIT License](LICENSE)
