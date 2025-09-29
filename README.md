# 面向特性的 Go 应用脚手架

本仓库收录了我们“特性自注册”和“双重开发范式”蓝图的最终形态。它是一个可运行的 Go 服务，展示了如何在拥有明确主见的应用核心之上，构建能够在不触碰启动流程的情况下自行注册的功能模块。模板刻意保持紧凑，让团队既能高速迭代，又能维持长期维护所需的清晰结构。

## 架构总览

```
cmd/service/            # 入口 - 创建应用上下文
internal/server/        # 应用核心、启动逻辑与特性清单
internal/auth/          # 结构化/依赖注入示例特性（用户管理）
internal/rbac/          # 结构化 RBAC 特性（角色与权限管理）
internal/middleware/    # 共享的 Gin 中间件
pkg/                    # 可复用的基础设施（配置、数据库、认证、观测等）
```

执行流程如下：

1. `cmd/service/main.go` 创建可取消的上下文、调用启动器并负责优雅停机。
2. `internal/server/bootstrap.go` 组装共享基础设施（配置、日志、数据库、观测、HTTP 路由器）并遍历特性清单。
3. 清单中的每个条目都暴露一个 `Register` 函数，接收共享依赖并在公共路由器上挂载路由。
4. `internal/server/app.go` 封装 HTTP 服务器生命周期（`Start`、`Shutdown`），让入口函数保持声明式。

## 特性清单

`internal/app/bootstrap.go` 是启用特性的唯一事实来源。新增特性意味着：

1. 在 `internal/` 下创建一个包含 `Register(context.Context, feature.Dependencies) error` 函数的包。
2. 将该注册函数追加到 `Features` 切片中，并附上描述性名称。
3. （可选）导出更多初始化日志，方便后续阅读。

由于启动器只是遍历这份列表，随着应用扩展，主函数可以保持原样。

## 双重开发范式

两个示例特性展示了如何在同一个应用中平衡速度与结构。

### 结构化 / 依赖注入路径 —— `internal/auth`

认证特性体现了“企业级”路线。`register.go` 负责装配仓储、服务和 HTTP 处理器。仓储拥有迁移和数据访问逻辑，服务层集中校验与领域编排，处理器暴露 REST 接口。这种风格强调显式依赖，适合复杂且变化频繁的业务场景。

核心文件：

- `internal/auth/repository.go` —— 持久化模型、迁移与错误转换。
- `internal/auth/service.go` —— 校验与领域编排。
- `internal/auth/handler.go` —— `/v1/users` 的 HTTP 契约。
- `internal/auth/register.go` —— 自包含的依赖装配。

### 细粒度 RBAC —— `internal/rbac`

RBAC 特性展示了如何在结构化范式下构建高度内聚的授权系统。模块集中定义角色与权限模型，提供仓储、服务、HTTP 处理器以及声明式的权限中间件。通过统一的 `/v1/rbac` API，前端可以动态管理角色、权限及其关联关系，同时所有业务路由只需声明所需权限即可获得保护。

核心文件：

- `internal/rbac/model.go` —— 角色与权限的统一数据模型。
- `internal/rbac/service.go` —— 角色/权限业务逻辑与种子数据。
- `internal/rbac/handler.go` —— 完整的 RBAC 管理 API。
- `internal/rbac/middleware.go` —— 基于权限的动态路由防护。
- `internal/rbac/register.go` —— 模块装配与依赖注册。

认证与 RBAC 模块共享同一个路由器，彼此之间不会将关注点泄漏到启动器中。

## 核心架构决策

### 主键策略：UUID vs 自增ID

脚手架默认为核心实体（如用户、角色）采用 UUID 作为主键。UUID 可以在应用层提前生成，彻底避免了“先插入再拿到自增 ID”所带来的事务耦合和跨服务协调难题，同时具备更好的安全性与分布式环境友好性。我们推荐使用顺序更友好的 UUID v7 生成方案，以获得更优的数据库索引性能。此脚手架的用户可以根据自身业务需求，自行修改为主键自增策略。

### 开发范式：结构化(DI) vs 简单(Singleton)

“简单范式”带来极高的开发效率，但可测试性与长期扩展性会受到限制。建议将其用于功能简单、需求稳定、几乎不需要未来演进的纯 CRUD 场景或快速原型验证。当涉及核心业务流程、逻辑复杂或需要长期维护的功能时，应优先选择“结构化(DI)范式”，以保持代码的可维护性与可演化性。

## 快速开始

1. 克隆本仓库并进入项目目录。
2. 运行交互式初始化工具，为你的组织重新命名 Go 模块路径：

   ```bash
   make init-project
   ```

3. 使用向导式生成器脚手架第一个特性：

   ```bash
   make new-feature
   ```

4. 运行服务：

   ```bash
   go run ./cmd/service
   ```

服务默认监听 `0.0.0.0:3000`。健康检查位于 `/health`，Prometheus 指标位于 `/metrics`，示例 API 则暴露在 `/v1`（通过启动阶段初始化的 JWT 管理器进行认证）。

## 扩展模板

1. **创建特性** —— 在 `internal/` 下添加目录并实现 `Register` 函数。
2. **加入清单** —— 向 `internal/app/bootstrap.go` 追加清单条目。
3. **选择合适范式** —— 可以像 `internal/auth` 或 `internal/rbac` 那样显式装配，也可以按需实现更轻量的单例模式。根据特性选择最合适的方式，两种范式可以在同一应用中并存。

借助这套流程，新增特性只需修改两个位置：特性自身目录与清单。

## 可观测性与基础设施

- 日志、指标、数据库访问、JWT 管理与观测功能位于 `pkg/`。每个包都同时提供构造器风格（`New*`）与单例风格（`Init`、`Default`）的辅助方法，让模块可以自由选择更顺手的模式。
- 请求日志、异常恢复、指标采集、认证等中间件位于 `internal/middleware/`，由共享路由器自动应用。

## 许可证

基于 Apache License 2.0 发布，详见 [LICENSE](LICENSE)。
