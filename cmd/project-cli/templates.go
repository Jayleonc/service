package main

const modelTemplate = `package {{.Package}}

import (
"github.com/google/uuid"

"github.com/Jayleonc/service/pkg/model"
)

// {{.EntityName}} 定义了 {{.DisplayName}} 实体的数据库结构。
type {{.EntityName}} struct {
        ID          uuid.UUID ` + "`gorm:\"type:uuid;primaryKey\"`" + `
        Name        string    ` + "`gorm:\"size:255\" json:\"name\"`" + `
        model.Base
}

// TableName overrides the default gorm table name.
func ({{.EntityName}}) TableName() string {
return "{{.Name}}"
}
`

const simpleHandlerTemplate = `package {{.Package}}

import (
        "errors"
        "io"
        "net/http"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
"gorm.io/gorm"

{{- if .EnableRBAC}}
"github.com/Jayleonc/service/internal/rbac"
{{- end}}
"github.com/Jayleonc/service/internal/feature"
"github.com/Jayleonc/service/pkg/database"
"github.com/Jayleonc/service/pkg/ginx/paginator"
"github.com/Jayleonc/service/pkg/ginx/request"
"github.com/Jayleonc/service/pkg/ginx/response"
"github.com/Jayleonc/service/pkg/xerr"
)

// CreateInput 定义创建 {{.DisplayName}} 的请求体。
type CreateInput struct {
        Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

// GetByIDInput 定义根据 ID 查询 {{.DisplayName}} 的请求体。
type GetByIDInput struct {
        ID string ` + "`json:\"id\" binding:\"required\"`" + `
}

// UpdateInput 定义更新 {{.DisplayName}} 的请求体。
type UpdateInput struct {
        ID   string ` + "`json:\"id\" binding:\"required\"`" + `
        Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

// DeleteInput 定义删除 {{.DisplayName}} 的请求体。
type DeleteInput struct {
        ID string ` + "`json:\"id\" binding:\"required\"`" + `
}

// ListQuery 描述列表查询可用的分页和过滤参数。
type ListQuery struct {
        Page     int    ` + "`json:\"page\"`" + `
        PageSize int    ` + "`json:\"pageSize\"`" + `
        OrderBy  string ` + "`json:\"orderBy\"`" + `
        Name     string ` + "`json:\"name\"`" + `
}

type Handler struct{}

func NewHandler() *Handler {
return &Handler{}
}

// GetRoutes 在这里定义模块的相对路由，无需包含模块名。
func (h *Handler) GetRoutes() feature.ModuleRoutes {
return feature.ModuleRoutes{
AuthenticatedRoutes: []feature.RouteDefinition{
{{- if .EnableRBAC}}
                {Path: "create", Handler: h.create, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionCreate)},
                {Path: "get_by_id", Handler: h.getByID, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionRead)},
                {Path: "update", Handler: h.update, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionUpdate)},
                {Path: "delete", Handler: h.delete, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionDelete)},
                {Path: "list", Handler: h.list, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionList)},
{{- else}}
                {Path: "create", Handler: h.create},
                {Path: "get_by_id", Handler: h.getByID},
                {Path: "update", Handler: h.update},
                {Path: "delete", Handler: h.delete},
                {Path: "list", Handler: h.list},
{{- end}}
},
}
}

func (h *Handler) create(c *gin.Context) {
var input CreateInput
if err := c.ShouldBindJSON(&input); err != nil {
response.Error(c, http.StatusBadRequest,
        xerr.ErrBadRequest.WithMessage("invalid request payload"))
return
}

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError,
        xerr.ErrInternalServer.WithMessage("database is not initialised"))
return
}

        record := &{{.EntityName}}{
                ID:   uuid.New(),
                Name: input.Name,
        }

if err := db.WithContext(c.Request.Context()).Create(record).Error; err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.SuccessWithStatus(c, http.StatusCreated, record)
}

func (h *Handler) getByID(c *gin.Context) {
        var input GetByIDInput
if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid request payload"))
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid resource identifier"))
                return
        }

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError,
        xerr.ErrInternalServer.WithMessage("database is not initialised"))
return
}

var record {{.EntityName}}
if err := db.WithContext(c.Request.Context()).First(&record, "id = ?", id).Error; err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
response.Error(c, http.StatusNotFound,
        xerr.ErrNotFound.WithMessage("{{.EntityVar}} not found"))
return
}
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, record)
}

func (h *Handler) update(c *gin.Context) {
        var input UpdateInput
if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid request payload"))
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid resource identifier"))
                return
        }

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError,
        xerr.ErrInternalServer.WithMessage("database is not initialised"))
return
}

ctx := c.Request.Context()
var record {{.EntityName}}
        if err := db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
                if errors.Is(err, gorm.ErrRecordNotFound) {
                        response.Error(c, http.StatusNotFound,
                                xerr.ErrNotFound.WithMessage("{{.EntityVar}} not found"))
                        return
                }
                response.Error(c, http.StatusInternalServerError, err)
                return
        }

        record.Name = input.Name

if err := db.WithContext(ctx).Save(&record).Error; err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, record)
}

func (h *Handler) delete(c *gin.Context) {
        var input DeleteInput
if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid request payload"))
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid resource identifier"))
                return
        }

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError,
        xerr.ErrInternalServer.WithMessage("database is not initialised"))
return
}

if err := db.WithContext(c.Request.Context()).Delete(&{{.EntityName}}{}, "id = ?", id).Error; err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, gin.H{"id": id})
}

func (h *Handler) list(c *gin.Context) {
        var query ListQuery
if err := c.ShouldBindJSON(&query); err != nil {
                if !errors.Is(err, io.EOF) {
                        response.Error(c, http.StatusBadRequest,
                                xerr.ErrBadRequest.WithMessage("invalid request payload"))
                        return
                }
}

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError,
        xerr.ErrInternalServer.WithMessage("database is not initialised"))
return
}

pageReq := request.Pagination{
                Page:     query.Page,
                PageSize: query.PageSize,
                OrderBy:  query.OrderBy,
}

session := db.WithContext(c.Request.Context()).Model(&{{.EntityName}}{})
if query.Name != "" {
session = session.Where("name LIKE ?", "%"+query.Name+"%")
}

result, err := paginator.Paginate[{{.EntityName}}](session, &pageReq)
if err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, result)
}
`

const simpleRegisterTemplate = `package {{.Package}}

import (
"context"
"fmt"

"github.com/Jayleonc/service/internal/feature"
"github.com/Jayleonc/service/pkg/database"
)

func Register(ctx context.Context, deps *feature.Dependencies) error {
        // 校验当前模块所需的依赖是否已经注入。
        if err := deps.Require("DB", "Router"); err != nil {
                return fmt.Errorf("{{.Name}} feature dependencies: %w", err)
        }

        // 读取默认数据库连接，保证基础 CRUD 能正确操作数据。
        db := database.Default()
        if db == nil {
                return fmt.Errorf("{{.Name}} feature requires a configured database connection")
        }

        // 自动迁移当前模块所需的数据表结构。
        if err := db.WithContext(ctx).AutoMigrate(&{{.EntityName}}{}); err != nil {
                return fmt.Errorf("migrate {{.Name}} tables: %w", err)
        }

        // 构建处理器并注册模块路由。
        handler := NewHandler()
        // 模块的路由将统一挂载到 /v1/{{ .FeatureName }} 路径下。
        deps.Router.RegisterModule("{{ .FeatureName }}", handler.GetRoutes())

        // 记录模块初始化日志，便于排查问题。
        if deps.Logger != nil {
                deps.Logger.InfoContext(ctx, "{{.LogMessage}}", "pattern", "simple-crud")
        }

        // 返回 nil 表示初始化成功。
        return nil
}
`

const structuredRepositoryTemplate = `package {{.Package}}

import (
"context"

"github.com/google/uuid"
"gorm.io/gorm"

"github.com/Jayleonc/service/pkg/ginx/paginator"
"github.com/Jayleonc/service/pkg/ginx/request"
"github.com/Jayleonc/service/pkg/ginx/response"
)

type Repository struct {
db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
return &Repository{db: db}
}

// ListOptions 描述分页查询可用的过滤条件。
type ListOptions struct {
Pagination request.Pagination
Name       string
}

func (r *Repository) Migrate(ctx context.Context) error {
if r.db == nil {
return gorm.ErrInvalidDB
}
return r.db.WithContext(ctx).AutoMigrate(&{{.EntityName}}{})
}

func (r *Repository) Create(ctx context.Context, entity *{{.EntityName}}) error {
return r.db.WithContext(ctx).Create(entity).Error
}

func (r *Repository) Update(ctx context.Context, entity *{{.EntityName}}) error {
return r.db.WithContext(ctx).Save(entity).Error
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
return r.db.WithContext(ctx).Delete(&{{.EntityName}}{}, "id = ?", id).Error
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*{{.EntityName}}, error) {
var entity {{.EntityName}}
if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
return nil, err
}
return &entity, nil
}

func (r *Repository) List(ctx context.Context, opts ListOptions) (*response.PageResult[{{.EntityName}}], error) {
query := r.db.WithContext(ctx).Model(&{{.EntityName}}{})
if opts.Name != "" {
query = query.Where("name LIKE ?", "%"+opts.Name+"%")
}

pagination := opts.Pagination
return paginator.Paginate[{{.EntityName}}](query, &pagination)
}
`

const structuredServiceTemplate = `package {{.Package}}

import (
"context"
"fmt"

"github.com/google/uuid"

"github.com/Jayleonc/service/pkg/ginx/request"
"github.com/Jayleonc/service/pkg/ginx/response"
)

type Service struct {
repo *Repository
}

func NewService(repo *Repository) *Service {
return &Service{repo: repo}
}

// CreateParams 定义创建 {{.DisplayName}} 所需的业务参数。
type CreateParams struct {
Name string
}

// UpdateParams 定义更新 {{.DisplayName}} 所需的业务参数。
type UpdateParams struct {
ID   uuid.UUID
Name string
}

// ListParams 描述查询 {{.DisplayName}} 列表时可用的选项。
type ListParams struct {
Pagination request.Pagination
Name       string
}

func (s *Service) Create(ctx context.Context, params CreateParams) (*{{.EntityName}}, error) {
if s.repo == nil {
return nil, fmt.Errorf("repository is not configured")
}

entity := &{{.EntityName}}{
ID:   uuid.New(),
Name: params.Name,
}

if err := s.repo.Create(ctx, entity); err != nil {
return nil, err
}

return entity, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*{{.EntityName}}, error) {
if s.repo == nil {
return nil, fmt.Errorf("repository is not configured")
}

return s.repo.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, params UpdateParams) (*{{.EntityName}}, error) {
if s.repo == nil {
return nil, fmt.Errorf("repository is not configured")
}

record, err := s.repo.FindByID(ctx, params.ID)
if err != nil {
return nil, err
}

record.Name = params.Name

if err := s.repo.Update(ctx, record); err != nil {
return nil, err
}

return record, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
if s.repo == nil {
return fmt.Errorf("repository is not configured")
}

return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, params ListParams) (*response.PageResult[{{.EntityName}}], error) {
if s.repo == nil {
return nil, fmt.Errorf("repository is not configured")
}

return s.repo.List(ctx, ListOptions{
Pagination: params.Pagination,
Name:       params.Name,
})
}
`

const structuredHandlerTemplate = `package {{.Package}}

import (
        "errors"
        "io"
        "net/http"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
"gorm.io/gorm"

{{- if .EnableRBAC}}
"github.com/Jayleonc/service/internal/rbac"
{{- end}}
"github.com/Jayleonc/service/internal/feature"
"github.com/Jayleonc/service/pkg/ginx/request"
"github.com/Jayleonc/service/pkg/ginx/response"
"github.com/Jayleonc/service/pkg/xerr"
)

type Handler struct {
service *Service
}

func NewHandler(service *Service) *Handler {
return &Handler{service: service}
}

// CreateInput 定义创建 {{.DisplayName}} 的请求结构。
type CreateInput struct {
        Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

// GetByIDInput 定义根据 ID 查询 {{.DisplayName}} 的请求结构。
type GetByIDInput struct {
        ID string ` + "`json:\"id\" binding:\"required\"`" + `
}

// UpdateInput 定义更新 {{.DisplayName}} 的请求结构。
type UpdateInput struct {
        ID   string ` + "`json:\"id\" binding:\"required\"`" + `
        Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

// DeleteInput 定义删除 {{.DisplayName}} 的请求结构。
type DeleteInput struct {
        ID string ` + "`json:\"id\" binding:\"required\"`" + `
}

// ListQuery 描述查询参数。
type ListQuery struct {
        Page     int    ` + "`json:\"page\"`" + `
        PageSize int    ` + "`json:\"pageSize\"`" + `
        OrderBy  string ` + "`json:\"orderBy\"`" + `
        Name     string ` + "`json:\"name\"`" + `
}

// GetRoutes 在这里定义模块的相对路由，无需包含模块名。
func (h *Handler) GetRoutes() feature.ModuleRoutes {
return feature.ModuleRoutes{
AuthenticatedRoutes: []feature.RouteDefinition{
{{- if .EnableRBAC}}
                {Path: "create", Handler: h.create, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionCreate)},
                {Path: "get_by_id", Handler: h.getByID, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionRead)},
                {Path: "update", Handler: h.update, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionUpdate)},
                {Path: "delete", Handler: h.delete, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionDelete)},
                {Path: "list", Handler: h.list, RequiredPermission: rbac.PermissionKey(rbac.Resource{{.PascalName}}, rbac.ActionList)},
{{- else}}
                {Path: "create", Handler: h.create},
                {Path: "get_by_id", Handler: h.getByID},
                {Path: "update", Handler: h.update},
                {Path: "delete", Handler: h.delete},
                {Path: "list", Handler: h.list},
{{- end}}
},
}
}

func (h *Handler) create(c *gin.Context) {
var input CreateInput
if err := c.ShouldBindJSON(&input); err != nil {
response.Error(c, http.StatusBadRequest,
        xerr.ErrBadRequest.WithMessage("invalid request payload"))
return
}

        entity, err := h.service.Create(c.Request.Context(), CreateParams{
                Name: input.Name,
        })
if err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.SuccessWithStatus(c, http.StatusCreated, entity)
}

func (h *Handler) getByID(c *gin.Context) {
        var input GetByIDInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid request payload"))
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid resource identifier"))
                return
        }

record, err := h.service.GetByID(c.Request.Context(), id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
response.Error(c, http.StatusNotFound,
        xerr.ErrNotFound.WithMessage("{{.EntityVar}} not found"))
return
}
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, record)
}

func (h *Handler) update(c *gin.Context) {
        var input UpdateInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid request payload"))
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid resource identifier"))
                return
        }

        record, err := h.service.Update(c.Request.Context(), UpdateParams{
                ID:   id,
                Name: input.Name,
        })
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
response.Error(c, http.StatusNotFound,
        xerr.ErrNotFound.WithMessage("{{.EntityVar}} not found"))
return
}
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, record)
}

func (h *Handler) delete(c *gin.Context) {
        var input DeleteInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid request payload"))
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest,
                        xerr.ErrBadRequest.WithMessage("invalid resource identifier"))
                return
        }

        if err := h.service.Delete(c.Request.Context(), id); err != nil {
                response.Error(c, http.StatusInternalServerError, err)
                return
        }

response.Success(c, gin.H{"id": id})
}

func (h *Handler) list(c *gin.Context) {
        var query ListQuery
        if err := c.ShouldBindJSON(&query); err != nil {
                if !errors.Is(err, io.EOF) {
                        response.Error(c, http.StatusBadRequest,
                                xerr.ErrBadRequest.WithMessage("invalid request payload"))
                        return
                }
        }

result, err := h.service.List(c.Request.Context(), ListParams{
Pagination: request.Pagination{
Page:     query.Page,
PageSize: query.PageSize,
OrderBy:  query.OrderBy,
},
Name: query.Name,
})
if err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, result)
}
`

const structuredRegisterTemplate = `package {{.Package}}

import (
"context"
"fmt"

"github.com/Jayleonc/service/internal/feature"
)

func Register(ctx context.Context, deps *feature.Dependencies) error {
        // 校验当前模块所需的依赖是否已经注入。
        if err := deps.Require("DB", "Router"); err != nil {
                return fmt.Errorf("{{.Name}} feature dependencies: %w", err)
        }

        // 构建仓储层，承载数据库读写。
        repo := NewRepository(deps.DB)
        // 自动迁移当前模块所需的数据表结构。
        if err := repo.Migrate(ctx); err != nil {
                return fmt.Errorf("migrate {{.Name}} tables: %w", err)
        }

        // 初始化业务服务并装配 HTTP 处理器。
        service := NewService(repo)
        handler := NewHandler(service)

        // 模块的路由将统一挂载到 /v1/{{ .FeatureName }} 路径下。
        deps.Router.RegisterModule("{{ .FeatureName }}", handler.GetRoutes())

        // 记录模块初始化日志，便于排查问题。
        if deps.Logger != nil {
                deps.Logger.InfoContext(ctx, "{{.LogMessage}}", "pattern", "structured-crud")
        }

        // 返回 nil 表示初始化成功。
        return nil
}
`

const errorsTemplate = `package {{.Package}}

// {{.DisplayName}} 模块错误码范围：{{.SuggestedErrorCode}}-{{.SuggestedErrorCodeEnd}}
//
// 提示：为 '{{.Name}}' 模块定义的业务错误码，建议从 {{.SuggestedErrorCode}} 开始，以避免与通用错误码或其他模块冲突。
// 使用示例：
// // var (
// //     ErrExampleFailure = xerr.New({{.SuggestedErrorCode}}, "示例错误描述")
// // )
`
