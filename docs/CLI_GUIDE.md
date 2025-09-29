# Project CLI 使用指南

本指南介绍了脚手架工具提供的核心命令，帮助开发者快速初始化项目并生成特性模块骨架。

## `make init-project`

`make init-project` 会运行 `project-cli init`，用于在克隆模板项目后第一次修改 Go Module 路径。

执行流程：

1. 脚本会读取当前 `go.mod` 中的 module 名称，并在终端显示。
2. 工具提示输入新的 module 路径（例如 `github.com/company/project`）。
3. 当输入的路径与当前路径不同且合法时，工具会遍历项目中的源码文件并批量替换旧的路径。
4. 终端会输出被更新的文件数量，并提示开发者在 Makefile 中移除或注释 `init-project` 目标，避免重复执行。

## `make new-feature`

`make new-feature` 会运行交互式的 `project-cli new-feature` 向导，根据回答生成模块目录、样板代码和注册逻辑。完整流程如下：

1. **特性名称**：要求输入使用小写字母、数字或下划线组成的名称（例如 `billing`、`user_profile`）。名称会同时用于包名、目录名以及模块注册信息。
2. **特性类型**：可选择 `Simple` 或 `Structured`。
   - `Simple` 生成最小化的 handler 与 register 文件，适合无数据库依赖的轻量模块。
   - `Structured` 额外生成 repository 与 service 层的文件骨架，为需要分层设计的模块提供模板。
3. **是否生成 RBAC 权限声明**：新增的可选项，默认回答为“否”。
   - 选择“否”时，生成的 handler 路由与旧版保持一致，不包含任何权限字段。
   - 选择“是”时，handler 中的每个 `feature.RouteDefinition` 都会自动带上 `RequiredPermission` 占位符，格式为 `<模块名>:<路由名>`（例如 `billing:ping`）。这样可以直接对接启用了 RBAC 的项目配置。

向导执行完成后，脚手架会将新模块注册到 `internal/app/bootstrap.go`，确保模块在应用启动时被正确加载。

> 小贴士：如需再次运行向导，可直接执行 `make new-feature`，脚手架会提示并阻止重复生成已有模块。
