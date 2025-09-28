# Feature Development Guide

This service template embraces two complementary development paradigms so that teams can choose the right balance between delivery speed and long-term maintainability. Both styles plug into a shared application bootstrap layer implemented in [`internal/server`](internal/server), which assembles infrastructure in `bootstrap.go`, exposes lifecycle management in `app.go`, and routes requests through `router.go` before dispatching them via the feature contracts declared in [`internal/feature`](internal/feature).

## Why two paradigms?

- **Structured / Dependency Injection (DI)** keeps dependencies explicit and testable. It is ideal for complex, business-critical features where lifecycle control, migrations, and robust testing matter most. The `auth` feature is the canonical example.
- **Simple / Singleton-first** favours rapid iteration by leaning on globally initialised infrastructure such as `database.Default()` or `logger.Default()`. It is perfect for thin CRUD endpoints, admin utilities, or experiments where speed outweighs ceremony. The `role` feature showcases this approach.

Supporting both patterns lets new contributors start fast while giving senior engineers a clear path for extracting durable, well-factored domains when requirements grow.

## Choosing the right paradigm

Run through this quick checklist whenever you create a feature:

- [ ] Does the feature touch core business workflows or sensitive data paths?
- [ ] Will the module require migrations, background jobs, or integration tests?
- [ ] Do you expect complex dependency graphs (multiple repositories, third-party clients, etc.)?
- [ ] Is long-term ownership shared across teams or does it demand strict boundaries?
- [ ] Do you need fine-grained observability (custom metrics, tracing spans, structured logs)?

If you answered “yes” to most questions, start with the **structured/DI** template. A majority of “no” answers means the **simple/singleton** template is usually sufficient—refactor to the structured pattern later if the feature evolves.

## Structured pattern (auth feature)

The `auth` module highlights explicit wiring from repository to handler, making dependencies easy to follow and mock.

```go
// internal/auth/register.go
func Register(ctx context.Context, deps feature.Dependencies) error {
        if deps.DB == nil {
                return fmt.Errorf("auth feature requires a database instance")
        }
        if deps.Router == nil {
                return fmt.Errorf("auth feature requires the route registrar")
        }

        repo := NewRepository(deps.DB)
        if err := repo.Migrate(ctx); err != nil {
                return fmt.Errorf("run migrations: %w", err)
        }

        svc := NewService(repo)
        handler := NewHandler(svc)
        deps.Router.RegisterModule("/auth", handler.GetRoutes())

        if deps.Logger != nil {
                deps.Logger.Info("auth feature initialised", "pattern", "structured")
        }

        return nil
}
```

Dependencies flow in a single direction: `NewRepository` → `NewService` → `NewHandler`. Each layer is easy to test in isolation and only receives what it needs. The feature depends on the shared router registrar exposed by the bootstrap layer, but everything else stays local.

## Simple pattern (role feature)

The `role` feature keeps the surface area tiny by calling global singletons directly from the handler.

```go
// internal/role/handler.go
func (h *Handler) create(c *gin.Context) {
        var req createRoleRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        userID, err := uuid.Parse(req.UserID)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
                return
        }

        db := database.Default()
        if db == nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialised"})
                return
        }

        record := assignment{UserID: userID, Role: req.Role}
        if err := db.WithContext(c.Request.Context()).Create(&record).Error; err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusCreated, gin.H{
                "id":           record.ID,
                "user_id":      record.UserID,
                "role":         record.Role,
                "date_created": record.DateCreated,
        })
}
```

There is no explicit wiring layer: the handler reaches for `database.Default()` when it needs persistence and returns early if the service is not ready. This keeps code generation small and accelerates prototyping.

## Scaffolding new features

Use the `make new-feature` automation to eliminate repetitive setup. The command launches an interactive wizard, so no flags are required:

```bash
make new-feature
```

The generator will:

1. Create `internal/<name>/` with either the DI (repository/service/handler/register) or singleton (handler/register) template.
2. Wire the feature into [`internal/app/bootstrap.go`](internal/app/bootstrap.go) so it is registered automatically during bootstrap.
3. Leave the bootstrap, router, and middleware untouched—new features plug directly into the existing lifecycle via `feature.Dependencies`.

After scaffolding, fill in repository methods, flesh out services, and replace placeholder routes with real logic. The rest of the application (configuration, database, logger, metrics, telemetry) is already available through `feature.Dependencies` or singleton helpers.
