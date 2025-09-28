package role

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/database"
	"github.com/Jayleonc/service/pkg/ginx/response"
)

// Handler 提供角色管理接口，直接使用全局数据库单例。
type Handler struct{}

// NewHandler 创建角色 Handler。
func NewHandler() *Handler {
	return &Handler{}
}

// GetRoutes 返回角色功能的路由定义。
func (h *Handler) GetRoutes() feature.ModuleRoutes {
	return feature.ModuleRoutes{
		AdminRoutes: []feature.RouteDefinition{
			{Path: "create", Handler: h.create},
			{Path: "update", Handler: h.update},
			{Path: "delete", Handler: h.delete},
			{Path: "list", Handler: h.list},
		},
	}
}

// CreateRoleRequest 定义创建角色的请求体。
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateRoleRequest 定义更新角色的请求体。
type UpdateRoleRequest struct {
	ID          string  `json:"id" binding:"required"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// DeleteRoleRequest 定义删除角色的请求体。
type DeleteRoleRequest struct {
	ID string `json:"id" binding:"required"`
}

func (h *Handler) create(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrCreateInvalidPayload)
		return
	}

	db, err := defaultDB()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, ErrCreateDatabaseUnavailable)
		return
	}

	name := normalizeRoleName(req.Name)
	if name == "" {
		response.Error(c, http.StatusBadRequest, ErrRoleNameRequired)
		return
	}

	role := &Role{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
	}

	if err := db.WithContext(c.Request.Context()).Create(role).Error; err != nil {
		response.Error(c, http.StatusBadRequest, ErrCreateFailed)
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, role)
}

func (h *Handler) update(c *gin.Context) {
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrUpdateInvalidPayload)
		return
	}

	roleID, err := uuid.Parse(req.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidRoleID)
		return
	}

	db, err := defaultDB()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, ErrUpdateDatabaseUnavailable)
		return
	}

	record, err := findRoleByID(c.Request.Context(), db, roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, ErrRoleNotFound)
			return
		}
		response.Error(c, http.StatusInternalServerError, ErrLoadRoleFailed)
		return
	}

	if req.Name != nil {
		normalized := normalizeRoleName(*req.Name)
		if normalized == "" {
			response.Error(c, http.StatusBadRequest, ErrUpdateRoleNameEmpty)
			return
		}
		record.Name = normalized
	}
	if req.Description != nil {
		record.Description = strings.TrimSpace(*req.Description)
	}

	if err := db.WithContext(c.Request.Context()).Save(record).Error; err != nil {
		response.Error(c, http.StatusBadRequest, ErrUpdateFailed)
		return
	}

	response.Success(c, record)
}

func (h *Handler) delete(c *gin.Context) {
	var req DeleteRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrDeleteInvalidPayload)
		return
	}

	roleID, err := uuid.Parse(req.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrDeleteInvalidRoleID)
		return
	}

	db, err := defaultDB()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, ErrDeleteDatabaseUnavailable)
		return
	}

	if err := db.WithContext(c.Request.Context()).Delete(&Role{}, "id = ?", roleID).Error; err != nil {
		response.Error(c, http.StatusBadRequest, ErrDeleteFailed)
		return
	}

	response.Success(c, gin.H{"id": roleID})
}

func (h *Handler) list(c *gin.Context) {
	db, err := defaultDB()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, ErrListDatabaseUnavailable)
		return
	}

	var roles []Role
	if err := db.WithContext(c.Request.Context()).Order("created_at ASC").Find(&roles).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, ErrListFailed)
		return
	}

	response.Success(c, roles)
}

// Migrate 执行角色模型迁移。
func Migrate(ctx context.Context) error {
	db, err := defaultDB()
	if err != nil {
		return err
	}
	return db.WithContext(ctx).AutoMigrate(&Role{})
}

// EnsureDefaultRoles 确保默认角色存在。
func EnsureDefaultRoles(ctx context.Context, defaults map[string]string) error {
	db, err := defaultDB()
	if err != nil {
		return err
	}

	for name, desc := range defaults {
		normalized := normalizeRoleName(name)
		if normalized == "" {
			continue
		}

		var existing Role
		err := db.WithContext(ctx).First(&existing, "name = ?", normalized).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				role := &Role{ID: uuid.Must(uuid.NewV7()), Name: normalized, Description: desc}
				if err := db.WithContext(ctx).Create(role).Error; err != nil {
					return err
				}
				continue
			}
			return err
		}
	}
	return nil
}

// FindRolesByNames 根据名称集合获取角色列表。
func FindRolesByNames(ctx context.Context, names []string) ([]Role, error) {
	db, err := defaultDB()
	if err != nil {
		return nil, err
	}

	normalized := normalizeList(names)
	if len(normalized) == 0 {
		return nil, nil
	}

	var records []Role
	if err := db.WithContext(ctx).
		Where("name IN ?", normalized).
		Order("created_at ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func findRoleByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*Role, error) {
	var record Role
	if err := db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func normalizeRoleName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

func normalizeList(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(names))
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := normalizeRoleName(name)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func defaultDB() (*gorm.DB, error) {
	db := database.Default()
	if db == nil {
		return nil, errors.New("role: database not initialised")
	}
	return db, nil
}
