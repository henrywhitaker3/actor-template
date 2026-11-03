# Actor service template

A starter service for building durable, actor-based Go applications. It brings together:

- [Francis](https://github.com/italypaleale/francis) for the local actor runtime
- an HTTP API with OpenAPI enabled through Windowframe
- GORM with SQLite for the zero-configuration default and PostgreSQL for deployed environments
- OpenTelemetry tracing, Prometheus metrics, structured logging, and health checks
- Docker Compose services for PostgreSQL, Jaeger, and the Francis dashboard

The repository is deliberately a skeleton: it starts the runtime and HTTP servers, but registers no domain models, actors, or API routes yet.

## Prerequisites

- Docker and Docker Compose
- [mise](https://mise.jdx.dev/), which installs the pinned Go toolchain and project tools

Install the tools once after cloning:

```sh
mise install
```

## Start locally

The simplest local setup uses the default SQLite database. Create a `.env` file containing a runtime pre-shared key and an application name:

```dotenv
APP_NAME=my-service
RUNTIME_PSK=replace-with-a-random-value
```

Generate a suitable key with:

```sh
mise run psk
```

Start the service and its supporting development services:

```sh
mise run serve
```

`serve` first runs `mise run dev`, which starts PostgreSQL, Jaeger, and the Francis dashboard in Docker. The Go process then runs on the host and, unless you set `DB_DRIVER` and `DB_URL`, stores its data in `db.sqlite` in the project directory.

Once running, the service exposes:

| Address | Purpose |
| --- | --- |
| `http://localhost:8080` | Application HTTP API |
| `http://localhost:8082/healthz` | Process health check |
| `http://localhost:8082/readyz` | Ready after the actor runtime is ready |
| `http://localhost:8082/metrics` | Prometheus metrics |
| `http://localhost:16686` | Jaeger UI |
| `http://localhost:7402` | Francis dashboard |

The HTTP server enables OpenAPI support. Routes and their OpenAPI operations appear once your application registers them.

### Use PostgreSQL while running Go on the host

The checked-in `.env.example` is intended for the `app` Docker Compose service, where `postgres` is a resolvable container hostname. For a host-run process, use `localhost` instead:

```dotenv
APP_NAME=my-service
RUNTIME_PSK=replace-with-a-random-value
DB_DRIVER=postgres
DB_URL=postgres://api:api@localhost:5432/api?sslmode=disable
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

Then run `mise run serve`. Jaeger receives OTLP/HTTP traces on port `4318`.

## Run with Docker Compose

Copy the provided environment file and set the two required application values:

```sh
cp .env.example .env
```

Add `APP_NAME` and `RUNTIME_PSK` to `.env`, then build and start the full stack:

```sh
docker compose up --build
```

In this mode the app connects to the Compose PostgreSQL service. Compose publishes the application API on `8080` and the actor runtime management port on `8081` only when `RUNTIME_MANAGEMENT_TOKENS` is configured. The management API itself listens on container port `7401`; adjust the Compose port mapping if you need host access to it.

## Configuration

Configuration is read from environment variables; a `.env` file is loaded automatically for local development.

| Variable | Default | Description |
| --- | --- | --- |
| `APP_NAME` | — | Service name used in API and telemetry metadata. |
| `APP_URL` | empty | Public API URL used when generating OpenAPI metadata. |
| `RUNTIME_HOST` | machine hostname | Hostname advertised by the Francis runtime. |
| `RUNTIME_PSK` | — | Pre-shared key used by runtime peers. Generate one with `mise run psk`. |
| `RUNTIME_MANAGEMENT_TOKENS` | empty | Comma-separated tokens that enable the runtime management API. |
| `RUNTIME_MANAGEMENT_ALLOWED_ORIGINS` | empty | Comma-separated origins allowed by the management API. |
| `DB_DRIVER` | `sqlite` | Database driver: `sqlite` or `postgres`. |
| `DB_URL` | `db.sqlite` | SQLite path or PostgreSQL connection URL. |
| `LOG_LEVEL` | `info` | Structured log level. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | library default | OTLP collector endpoint for traces. |

Keep `.env` out of source control. In particular, treat `RUNTIME_PSK` and management tokens as secrets.

## How the application starts

`main.go` coordinates startup and shutdown:

1. `bootstrap.Bootstrap` loads configuration, configures logging and telemetry, opens the database, and creates the Francis runtime and HTTP server.
2. The actor runtime starts on UDP port `7157` and signals readiness.
3. After that signal, the HTTP API starts on port `8080` and `/readyz` starts returning `200`.
4. A separate metadata server starts immediately on port `8082` for health checks and metrics.
5. On shutdown, the HTTP and metadata servers are stopped gracefully and the runtime context is cancelled.

The bootstrap options in `main.go` are the extension points:

- `RegisterModelFuncs` runs database model setup, such as GORM migrations.
- `RegisterActorFuncs` receives the constructed application, including the Francis host and GORM database, and registers actor factories.
- `RegisterAPIRoutes` receives the HTTP server and shared dependencies and registers application routes.

`internal/database` also includes `BaseModel` for common UUID/timestamp fields and a generic `Pagination` helper. `internal/testutils` provides a disposable, ready-to-use Francis host for actor tests.

## Make this template yours

Replace the Go module path before adding application code:

```sh
mise run rename github.com/your-org/your-service
mise exec -- go mod tidy
```

The rename task updates the Go source imports and `go.mod`. Update the example paths in this README and any repository metadata to match your new service.

Add your models, actor registrations, and route registrars, then pass them to `bootstrap.Bootstrap` in `main.go`. Keeping those registrations explicit makes startup order and dependencies visible in one place.

## Development commands

| Command | Description |
| --- | --- |
| `mise run dev` | Start PostgreSQL, Jaeger, and the Francis dashboard. |
| `mise run serve` | Start dependencies and run the Go service. |
| `mise run psk` | Generate a random runtime pre-shared key. |
| `mise run test` | Run all Go tests with `gotestsum`. |
| `mise run coverage` | Write an HTML coverage report to `reports/coverage.html`. |
| `mise run release` | Create and publish a version/tag and GitHub release from Conventional Commit history. |

`mise run release` pushes Git tags and commits and creates a GitHub release, so run it only from a clean, reviewed branch with the required GitHub access.
