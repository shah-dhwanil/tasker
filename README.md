# Tasker

A Task Management REST API built in Go. Provides CRUD operations for todos and categories with user isolation via Clerk authentication, parent-child todo relationships with automatic status synchronization, full-text search, pagination, sorting, filtering, and soft-delete support.

## Features

- **Todo management** — Create, read, update, soft-delete todos with status, priority, category, and parent-child relationships
- **Category management** — Create, read, update, soft-delete categories
- **Parent-child status sync** — Auto-completes/reactivates parent todos based on children status
- **Full-text search** — PostgreSQL `tsvector`-based search across todo titles and descriptions
- **Pagination & filtering** — Cursor-less pagination with filtering by status, priority, category, parent, and text search; sorting by any field
- **Observability** — New Relic APM instrumentation and structured logging with zap
- **Clean architecture** — Handler → Service → Repository layered design with dependency injection

## Tech Stack

| Language | Go 1.25.5 |
|----------|-----------|
| HTTP Framework | [Echo v4](https://github.com/labstack/echo) |
| Database | PostgreSQL via [pgx v5](https://github.com/jackc/pgx) |
| Auth | [Clerk SDK v2](https://github.com/clerk/clerk-sdk-go) |
| Migrations | [tern v2](https://github.com/jackc/tern) |
| Validation | [go-playground/validator v10](https://github.com/go-playground/validator) |
| Logging | [zap](https://go.uber.org/zap) |
| APM | [New Relic Go Agent v3](https://github.com/newrelic/go-agent) |
| Config | [koanf v2](https://github.com/knadh/koanf) |
| API Docs | [swaggo/swag v2](https://github.com/swaggo/swag) |
| Testing | [testify](https://github.com/stretchr/testify) |

## Getting Started

### Prerequisites

- Go 1.25.5+
- PostgreSQL database

### Installation

```bash
git clone https://github.com/shah-dhwanil/tasker.git
cd tasker
```

### Configuration

Copy the example configuration and set environment variables:

```bash
cp .env.example .env
```

Required environment variables (loaded from `.env` by godotenv):

| Variable | Description |
|----------|-------------|
| `TASKER_POSTGRES.DSN` | PostgreSQL connection string |
| `TASKER_CLERK.SECRET_KEY` | Clerk secret key |
| `TASKER_NEW_RELIC.LICENSE_KEY` | New Relic license key |

Configuration is loaded from `config.yaml` and overridden by `TASKER_*` environment variables.

### Running

```bash
go run cmd/main.go
```

The server starts on port 8000 by default (configurable in `config.yaml`).

Swagger UI is available at `http://localhost:8000/swagger/index.html`.

### Testing

```bash
go test ./...
go test -v ./internal/service/...
go test -v ./internal/repository/...
go test -v ./internal/handler/...
```

Tests use `.env.test` with a separate database and Clerk test JWT. The test setup creates the full schema, runs tests, then tears down.

### Building

```bash
go build -o tasker cmd/main.go
```

## API

All endpoints require a `Bearer <token>` Authorization header (Clerk JWT).

### Health

| Method | Path |
|--------|------|
| GET | `/health` |

### Categories

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/categories` | Create category |
| GET | `/api/v1/categories` | List categories (paginated) |
| GET | `/api/v1/categories/:categoryId` | Get category by ID |
| PATCH | `/api/v1/categories/:categoryId` | Update category |
| DELETE | `/api/v1/categories/:categoryId` | Soft-delete category |

### Todos

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/todos` | Create todo |
| GET | `/api/v1/todos` | List todos (paginated, filterable, searchable, sortable) |
| GET | `/api/v1/todos/:todoId` | Get todo by ID (includes children) |
| PATCH | `/api/v1/todos/:todoId` | Update todo |
| DELETE | `/api/v1/todos/:todoId` | Soft-delete todo |

## Project Structure

```
cmd/
  main.go               # Application entry point
internal/
  app/                  # Server setup and dependency wiring
  config/               # Configuration loading (koanf)
  database/             # pgx pool, migrations, query utilities
  error_handler/        # AppError → HTTP response mapping
  errors/               # Domain error types
  handler/              # HTTP handlers (generic Handle[T] wrapper)
  middleware/           # Auth, CORS, logging, rate limiter, New Relic, request ID
  normalization/        # Request normalization interface
  observability/        # Logging and New Relic setup
  repository/           # Database queries with dynamic builders
  routes/               # Route registration
  schema/               # Domain types, DTOs, request/response schemas
  service/              # Business logic layer
  testing/              # Test helpers and setup/teardown
  validation/           # Struct validation
docs/                   # Swagger generated docs
```

## Architecture

The API follows a clean layered architecture:

```
Handler → Service → Repository → PostgreSQL
```

- **Handlers** — Bind request data, validate, normalize, call service, return response
- **Services** — Business logic, ownership checks, parent-child status sync, parallel queries
- **Repositories** — Database access with dynamic query building and error mapping

Key patterns: generics-based `Handle[T]` wrapper, `Nullable[T]` for partial updates, `errgroup` for concurrent queries, soft-deletes with `is_deleted` flag.

## License

GNU Lesser General Public License v2.1
