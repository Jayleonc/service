# Auth Service Toolbox

The Auth Service Toolbox is a Go service starter kit that demonstrates how to assemble
production-ready building blocks without forcing a single dependency management strategy.
It keeps the application layer focused on behaviour while extracting reusable
infrastructure components into a portable toolbox. The result is a codebase that is easy
to reason about, simple to extend, and ready to power new services beyond authentication.

## Architectural Intent

This project applies four guiding principles:

1. **Separate application from infrastructure** – the `cmd/` tree only wires components
together. Reusable capabilities such as logging, configuration loading, telemetry, and
database connectivity live in `pkg/` and can be imported by any service.
2. **Expose dual dependency paradigms** – every infrastructure component offers both a
constructor (dependency injection path) and a global accessor (singleton path). Teams can
choose the style that best matches the scenario.
3. **Focus on a universal core** – the repository ships with an authentication service and
user management workflow, a feature set relevant to almost every product.
4. **Documentation as a product** – this README is the blueprint describing how the
architecture fits together and how to extend it responsibly.

## Project Layout

```
cmd/
  auth/        # Application assembly – orchestrates the auth service
internal/
  handler/     # HTTP handlers and router composition
  middleware/  # Cross-cutting Gin middleware
  repository/  # Data access for users
  service/     # Domain logic for user management
pkg/
  auth/        # JWT manager with DI + singleton access
  config/      # Configuration loader and global access
  database/    # GORM Postgres connection helpers
  logger/      # Structured logging helpers
  metrics/     # Prometheus registry helpers
  server/      # HTTP server factory
  telemetry/   # OpenTelemetry initialisation helpers
```

`cmd/auth/main.go` now reads like an assembly script. It wires configuration, logging,
telemetry, persistence, and the HTTP surface without embedding the construction details of
those pieces. Each toolbox package is reusable across future services.

## Dual Dependency Paradigms

Every infrastructure package exposes two complementary usage patterns.

### Structured / Dependency Injection Path

Use explicit constructors when you need clarity, testability, or custom lifecycles.

```go
cfg, _ := config.Load(ctx, nil)
log := logger.New(logger.Config{Level: cfg.Log.Level})
authManager, _ := auth.NewManager(auth.Config{Secret: cfg.Auth.Secret})
db, _ := database.New(database.Config{Host: cfg.Database.Host})
repo := repository.NewUserRepository(db)
svc := service.NewUserService(repo)
```

Each dependency is passed to the next layer, making relationships explicit and easy to
mock in unit tests.

### Simple / Singleton Path

When speed matters more than indirection, initialise the toolbox once and use global
helpers anywhere.

```go
config.Init(ctx, nil)
logger.Init(logger.Config{Level: "info"})
database.Init(database.Config{Host: "localhost"})
auth.Init(auth.Config{Secret: "supersecret"})

func CreateSession(userID string) (string, error) {
        mgr := auth.Default()
        return mgr.GenerateToken(userID, []string{"user"})
}
```

The singleton path keeps peripheral code lightweight while still being threadsafe. Both
approaches coexist, so teams can mix and match as complexity evolves.

## Running the Auth Service

1. Provision dependencies (for example via Docker Compose) and export the required
   environment variables. Only a PostgreSQL database is required to boot the API.
2. Install Go 1.22 or newer.
3. Start the service:

```bash
cd cmd/auth
go run .
```

The server listens on `0.0.0.0:3000` by default. Update configuration via environment
variables prefixed with `AUTH_` (e.g. `AUTH_DATABASE_HOST`, `AUTH_LOG_LEVEL`).

## Extending the Toolbox

- Add new reusable infrastructure by creating another package under `pkg/` with both DI
  and singleton entry points.
- Keep new domain logic inside `internal/` packages and expose HTTP or gRPC handlers that
  orchestrate the toolbox dependencies.
- Create additional services by adding new folders under `cmd/` that assemble the
  components they require.

## Observability

- Logging is provided by `pkg/logger`, with defaults configured through `AUTH_LOG_LEVEL`
  and `AUTH_LOG_PRETTY`.
- Metrics are exposed via the `/metrics` endpoint when a Prometheus registry is wired.
- Tracing is optional: enable it with `AUTH_TELEMETRY_ENABLED=true` and point
  `AUTH_TELEMETRY_ENDPOINT` to an OTLP endpoint.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
