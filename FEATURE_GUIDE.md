# 功能开发指南

该服务模板同时支持两套互补的开发范式，方便团队在交付速度与长期可维护性之间取得平衡。无论采用哪种方式，最终都会接入 [`internal/server`](internal/server) 中的通用启动流程：`bootstrap.go` 负责组装基础设施，`app.go` 暴露生命周期管理能力，`router.go` 则在根据 [`internal/feature`](internal/feature) 下定义的契约完成路由与分发。

## 为什么要提供两种范式？

- **结构化 / 依赖注入（DI）模式**：依赖关系透明可控，便于测试，非常适合业务复杂、生命周期长或对数据安全要求高的功能，例如 `auth` 模块。
- **轻量化 / 单例优先模式**：依赖全局初始化的基础设施（如 `database.Default()` 或 `logger.Default()`）推进迭代，适合 CRUD、管理工具或以速度为先的实验性功能，`role` 模块即为示例。

双轨并行让新成员可以快速落地需求，同时也为后续提炼领域能力、构建稳定可靠的模块提供清晰的迁移路径。

## 如何选择合适的范式

在创建新功能时，可以先回答以下问题：

- [ ] 功能是否涉及核心业务流程或敏感数据路径？
- [ ] 是否需要数据库迁移、后台任务或集成测试？
- [ ] 是否会出现复杂的依赖图（多个仓储、第三方客户端等）？
- [ ] 是否需要多个团队共同长期维护，需要严格的边界？
- [ ] 是否需要精细的可观测性（自定义指标、Tracing、结构化日志）？

如果大多数问题的答案为“是”，建议直接使用 **结构化/DI 模板**。若多为“否”，则可以先选择 **轻量化/单例模板**，后续一旦复杂度提升再重构为结构化模式即可。

## 结构化模式示例（auth 模块）

`auth` 模块清晰地展示了从仓储到处理器的显式依赖注入过程，便于追踪与 Mock：

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

依赖沿着单一方向流转：`NewRepository` → `NewService` → `NewHandler`。每一层都易于独立测试，仅接收所需能力。功能模块依赖启动层提供的路由注册器，其余资源均保持局部自治。

## 轻量化模式示例（role 模块）

`role` 模块通过直接使用全局单例，尽可能压缩样板代码：

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
                "id":        record.ID,
                "userId":    record.UserID,
                "role":      record.Role,
                "createdAt": record.CreatedAt,
        })
}
```

该模式不需要显式装配层，处理器直接调用 `database.Default()` 并在不可用时快速失败，最大程度压缩生成代码，适合快速原型。

## 新功能脚手架

执行 `make new-feature` 可以自动化创建模板，避免重复劳动。命令提供交互式向导，无需额外参数：

```bash
make new-feature
```

代码生成器会完成以下工作：

1. 在 `internal/<name>/` 下创建结构化（repository/service/handler/register）或单例（handler/register）模板。
2. 将功能自动注册到 [`internal/app/bootstrap.go`](internal/app/bootstrap.go)，在应用启动时自动加载。
3. 保持启动、路由和中间件逻辑不变——新功能通过 `feature.Dependencies` 或单例助手直接接入现有生命周期。

脚手架完成后，只需补全仓储方法、完善服务逻辑并用真实路由替换占位符。配置、数据库、日志、指标、链路追踪等基础能力已经通过 `feature.Dependencies` 或全局单例提供，可即刻复用。
