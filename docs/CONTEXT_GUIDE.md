# Context 处理模式的决策与规范（go-service-scaffold）

本文档记录并统一本项目在上下文（Context）传递方面的架构决策与编码规范，旨在在保持解耦正确性的同时，最大化开发体验与效率。

---

## 1. 问题的提出：为什么 `c.Request.Context()` 让人困扰？

- **What**

```go
// 以 Handler: updateMe 为例（示意代码）
func (h *Handler) updateMe(c *gin.Context) {
    // 日志记录
    h.logger.Info(c.Request.Context(), "updating profile")

    // 业务调用
    if err := h.svc.UpdateProfile(c.Request.Context(), parseUpdateReq(c)); err != nil {
        h.logger.Error(c.Request.Context(), "update profile failed", err)
        renderErr(c, err)
        return
    }

    renderOK(c)
}
```

Handler 层需要反复显式地写 `c.Request.Context()`，在日志、服务调用、仓储访问等多个位置重复出现，既冗余又不直观。

- **Why**

从架构角度，我们希望业务逻辑、基础设施层与 `gin` 解耦，因此传递的应是标准库 `context.Context`。这在原则上是正确的；但在 Handler 的编写体验上显得啰嗦且降低可读性与心智愉悦度。

---

## 2. 模式一：严格引导（Strict Guidance）

- **What**

强制在所有调用点手动传递 `c.Request.Context()`；对错误用法（例如直接传入 `*gin.Context`）通过 `panic` 或断言失败快速暴露问题，以“惩罚式”方式统一风格。

- **Why**

- **优点**
  - 强制推行解耦原则，风格绝对统一。
  - 对团队具有“教育意义”，让上下文边界清晰明确。
- **缺点**
  - 开发者体验差、心智负担高，样板代码（boilerplate）显著增多。
  - 容易在重复性操作中引入疏漏，违背“低心智负担”的项目哲学。

---

## 3. 模式二：智能兼容（Smart Compatibility）

- **What**

在底层工具函数内部（如 `logger.FromContext`、`metrics.FromContext`、`trace.FromContext` 等）增加小幅“智能逻辑”，能够识别并兼容 `*gin.Context`，从中提取并统一为标准 `context.Context`，从而让上层调用可以直接传入 `c`。

- **Why**

- **优点**
  - 对上层开发者完全透明，极大提升便捷性与效率。
  - Handler 层可以自然地传入 `c`，减少样板代码，代码更直观。
- **缺点**
  - 底层存在“幕后”的智能处理，可能弱化对“解耦原则”的显式认知。
  - 需要在核心工具位置保持实现与测试的严谨性，以免掩盖真实问题。

---

## 4. 模式三：应用上下文（App Context）

- **要解决的核心问题**

  - 将“便利性”和“架构解耦”同时满足：对 Handler 与业务层提供极简 API，对框架保持零耦合。

- **What（示意）**

定义与框架无关的 `AppContext` 类型：

```go
// 伪代码示意：不依赖 gin，仅聚合项目所需能力
package appctx

type AppContext struct {
    Ctx    context.Context
    Logger *logger.Logger
    User   *auth.User // 示例：存放已解析的用户信息
    // 可选：保留原生 gin 上下文，供适配层/响应阶段使用（慎用）
    Gin    *gin.Context
    // ... 其他与框架无关的能力/元数据
}

// Handler 层签名
func (h *Handler) updateMe(c *appctx.AppContext) {
    c.Logger.Info("updating profile")
    if err := h.svc.UpdateProfile(c, payload); err != nil { /* ... */ }
}

// Service 层也以 AppContext 为第一参数（或至少能接收它）
func (s *Service) UpdateProfile(c *appctx.AppContext, req UpdateReq) error { /* ... */ return nil }
```

- **Why**

- **优点**
  - 极致便利：`c.Logger.Info(...)`、`svc.UpdateProfile(c, ...)` 等调用天然简洁。
  - 架构严格：`AppContext` 与 `gin` 无关，业务层可安心依赖自有类型。
- **缺点**
  - 需要引入新的上下文类型、构造与传递方式，存在落地与迁移成本。
  - 对三方库/通用签名（期望 `context.Context`）仍需桥接，也会形成双轨心智模型。

> 结论：作为成熟期的架构演进方向可行，但对脚手架“开箱即用、低门槛”的目标来说，初期成本偏高。

### 模式三的实现细节（wrap 与生命周期）

- **wrap 函数（从 gin.Context 生成 AppContext）**

```go
// 典型形态一：返回可选的清理函数
func FromGin(c *gin.Context) (*appctx.AppContext, func()) {
    base := c.Request.Context()
    // 聚合本项目需要的能力与数据
    ac := &appctx.AppContext{
        Ctx:    base,                   // 继承请求上下文，保留取消/超时
        Logger: logger.FromContext(base),
        User:   auth.FromContext(base), // 例如从中间件放入的用户信息
        Gin:    c,                      // 可选：暴露 gin.Context（仅适配/响应层使用）
    }
    // 可选：派生子 context 用于请求内的统一取消/超时控制
    ctx, cancel := context.WithCancel(base)
    ac.Ctx = ctx
    // cleanup: 归还资源、flush 缓冲、结束 span 等
    cleanup := func() { cancel() }
    return ac, cleanup
}

// 典型形态二：纯构造（无清理函数）
func Wrap(ctx context.Context, deps Deps) *appctx.AppContext {
    return &appctx.AppContext{
        Ctx:    ctx,
        Logger: deps.Logger,
        User:   deps.User,
    }
}
```

- **集成位置与生命周期**

  - 在 `gin` 中间件或 Handler 入口处调用 `FromGin(c)` 构造 `AppContext`，`defer cleanup()` 保证资源回收。
  - 将 `AppContext` 透传到 Service/Repo：`svc.UpdateProfile(ac, ...)`。
  - 访问第三方库（要求 `context.Context`）时使用 `ac.Ctx`：`client.Do(ac.Ctx, ...)`。

- **互操作策略**
  - Web 框架相关能力留在适配层（中间件/适配器），`AppContext` 仅暴露与业务相关的通用能力。
  - 若需要 `gin.Context`，仅在最外层适配，不向业务/仓储泄漏。

---

## 5. 决策与最终方案

- **What**

我们最终选择：**模式二：智能兼容（Smart Compatibility）**。

需要注释掉 pkg/observe/logger/logger.go,ensureContext() 的内容逻辑

- **Why**

在一个以效率与开发者体验为首要目标的脚手架中，我们倾向以底层少量“智能代码”换取上层业务的极致简洁，从而让团队成员“无摩擦”地达成正确做法。

### 何时优先选择模式一或模式三

- **优先模式一（严格引导）**

  - 团队规模大、代码审查严格，需要“强一致性与可教化”的组织目标。
  - 你希望以明确错误（如 `panic`/lint 规则）来阻止不当用法，形成稳定的工程文化。

- **考虑模式三（应用上下文）**
  - 需要在“极致便利”与“严格解耦”之间取得长期平衡，且能接受引入新类型与迁移成本。
  - 希望在 DSL/业务 API 中呈现最小心智模型（`c.Logger.Info(...)`、`svc.Do(ac, ...)`）。

---

## 6. 学习要点（Learning Notes）

- **上下文统一思想**：无论 Handler 传入 `*gin.Context` 还是 `context.Context`，底层都会在某处统一为标准 `context.Context`，以保持与 Web 框架解耦。
- **调用风格对比**：文中的示例（Before/After）仅用于理解不同风格带来的心智负担差异，不构成改造建议。
- **模式选择的本质**：
  - 模式一强调“显式与纪律”，更具教育意义与一致性。
  - 模式二强调“便利与体验”，以少量底层智能换取上层简洁。
  - 模式三通过 `AppContext` 统一承载业务所需能力，保持对框架的最小泄漏。

---

## 7. 结论与编码规范（Standards）

- **What（规范）**

  - Handler 层编写时，开发者可以直接将 `*gin.Context` 传递给所有需要上下文的底层函数（如 `logger.Info`、`metrics.Inc`、`svc.DoSomething` 等）。
  - 所有新建的、对外接收 `context.Context` 的底层函数，必须在入口处调用 `ensureContext` 将来参标准化。

- **Why（目的）**

  - 统一最佳实践，降低心智负担，避免样板代码，提升团队整体开发效率。


---

## 附录：术语对照

- `context.Context`：Go 标准库上下文接口。
- `*gin.Context`：`gin-gonic` 框架的请求上下文；承载 HTTP 请求与中间件数据。
- `AppContext`：本文第三种模式中的项目自定义上下文类型，独立于 Web 框架。

我觉得模式一更好，就是麻烦一点，注意使用 gin 里 Request.Context() 就可以了，如果你使用 project-cli 生成的代码就是模式一。
