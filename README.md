# Modular Go Application Scaffold

This repository captures the final iteration of our "modular self-registration" and "dual development paradigm" blueprint. It is a runnable Go service whose structure demonstrates how to combine an opinionated application core with modules that register themselves without touching the bootstrap logic. The template is intentionally compact so that teams can move fast while keeping a clear path for long-term maintenance.

## Architecture Overview

```
cmd/service/            # Entry point – creates the application context
internal/server/        # Application core, bootstrap logic and module manifest
internal/auth/          # Structured/DI example module (user management)
internal/role/          # Simple/Singleton example module (role assignments)
internal/middleware/    # Shared Gin middleware
pkg/                    # Reusable infrastructure (config, DB, auth, telemetry, ...)
```

The execution flow is as follows:

1. `cmd/service/main.go` creates a cancellable context, invokes the bootstrapper, and manages graceful shutdown.
2. `internal/server/bootstrap.go` assembles shared infrastructure (config, logger, database, telemetry, HTTP router) and iterates the module manifest.
3. Each entry in `internal/server/modules.go` exposes a `Register` function that receives shared dependencies and mounts routes on the shared router.
4. `internal/server/app.go` encapsulates the HTTP server lifecycle (`Start`, `Shutdown`) so the entry point stays declarative.

## Module Manifest

`internal/server/modules.go` is the single source of truth for enabled modules. Adding a module means:

1. Creating a package under `internal/` with a `Register(context.Context, server.ModuleDeps) error` function.
2. Appending the register function to the `Modules` slice alongside a descriptive name.
3. (Optional) Exporting additional setup logs to guide future readers.

Because the bootstrapper simply loops over this list, the main function remains untouched as the application grows.

## Dual Development Paradigms

Two modules demonstrate how to balance speed and structure inside the same application.

### Structured / Dependency-Injection Path – `internal/auth`

The authentication module models the "enterprise" path. `register.go` wires a repository, service, and HTTP handler. The repository owns migrations and data access logic, the service layer centralises validation, and the handler exposes REST endpoints. This style favours explicit dependencies and is ideal for complex, high-change domains.

Key files:

- `internal/auth/repository.go` – persistence model, migrations, and error translation.
- `internal/auth/service.go` – validation and domain orchestration.
- `internal/auth/handler.go` – HTTP contract for `/v1/users`.
- `internal/auth/register.go` – self-contained dependency graph assembly.

### Simple / Singleton Path – `internal/role`

The role module embraces the "move fast" path. `register.go` fetches the globally initialised database (set up by the bootstrapper), runs a lightweight migration, and mounts a single handler. The handler itself directly touches GORM to insert rows for `POST /v1/roles`. No repository or service layer is introduced—the logic stays inside the handler for maximum velocity when requirements are small and well understood.

Key files:

- `internal/role/handler.go` – inline model definitions plus the request handler using `database.Default()`.
- `internal/role/register.go` – minimal bootstrap, perfect for quick CRUD style features.

Both modules share the same router instance and live side-by-side without leaking concerns into the bootstrapper.

## Quick Start

1. Install Go 1.22 or newer.
2. Ensure PostgreSQL is available and export configuration through the `AUTH_` environment variables (defaults point to `localhost:5432`).
3. Run the service:

   ```bash
   go run ./cmd/service
   ```

The server starts on `0.0.0.0:3000` by default. A health probe is available at `/health`, Prometheus metrics at `/metrics`, and the demo APIs under `/v1` (authenticated by the JWT manager initialised during bootstrap).

## Extending the Template

1. **Create a module** – add a folder under `internal/` and implement a `Register` function.
2. **Add it to the manifest** – append an entry to `internal/server/modules.go`.
3. **Use the paradigm that fits** – wire explicit constructors (like `internal/auth`) or lean on global singletons (like `internal/role`). You can mix approaches within the same application depending on the feature.

With this workflow, a new module can be added by touching only two locations: its own directory and the manifest.

## Observability and Infrastructure

- Logging, metrics, database access, JWT management, and telemetry live in `pkg/`. Each package exposes both constructor-style (`New*`) and singleton-style (`Init`, `Default`) helpers so modules can choose the ergonomics they need.
- Middleware such as request logging, panic recovery, metrics collection, and authentication live in `internal/middleware/` and are automatically applied by the shared router.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
