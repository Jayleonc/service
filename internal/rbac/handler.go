package rbac

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/ginx/response"
)

// Handler exposes management APIs for the RBAC module.
type Handler struct {
	svc ServiceContract
}

// ServiceContract 定义了 Handler 所需的服务层能力。
type ServiceContract interface {
	CreateRole(ctx context.Context, input CreateRoleInput) (*Role, error)
	UpdateRole(ctx context.Context, input UpdateRoleInput) (*Role, error)
	DeleteRole(ctx context.Context, input DeleteRoleInput) error
	ListRoles(ctx context.Context) ([]Role, error)
	AssignPermissions(ctx context.Context, input AssignRolePermissionsInput) (*Role, error)
	GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error)
	CreatePermission(ctx context.Context, input CreatePermissionInput) (*Permission, error)
	UpdatePermission(ctx context.Context, input UpdatePermissionInput) (*Permission, error)
	DeletePermission(ctx context.Context, input DeletePermissionInput) error
	ListPermissions(ctx context.Context) ([]Permission, error)
}

// NewHandler constructs a handler with the provided service dependency.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

var _ ServiceContract = (*Service)(nil)

// GetRoutes declares the RBAC management endpoints.
func (h *Handler) GetRoutes() feature.ModuleRoutes {
	return feature.ModuleRoutes{
		AuthenticatedRoutes: []feature.RouteDefinition{
			{Path: "role/create", Handler: h.createRole, RequiredPermission: PermissionKey(ResourceRBACRole, ActionCreate)},
			{Path: "role/update", Handler: h.updateRole, RequiredPermission: PermissionKey(ResourceRBACRole, ActionUpdate)},
			{Path: "role/delete", Handler: h.deleteRole, RequiredPermission: PermissionKey(ResourceRBACRole, ActionDelete)},
			{Path: "role/list", Handler: h.listRoles, RequiredPermission: PermissionKey(ResourceRBACRole, ActionList)},
			{Path: "role/assign_permissions", Handler: h.assignRolePermissions, RequiredPermission: PermissionKey(ResourceRBACRole, ActionAssignPermissions)},
			{Path: "role/get_permissions", Handler: h.getRolePermissions, RequiredPermission: PermissionKey(ResourceRBACRole, ActionViewPermissions)},
			{Path: "permission/create", Handler: h.createPermission, RequiredPermission: PermissionKey(ResourceRBACPermission, ActionCreate)},
			{Path: "permission/update", Handler: h.updatePermission, RequiredPermission: PermissionKey(ResourceRBACPermission, ActionUpdate)},
			{Path: "permission/delete", Handler: h.deletePermission, RequiredPermission: PermissionKey(ResourceRBACPermission, ActionDelete)},
			{Path: "permission/list", Handler: h.listPermissions, RequiredPermission: PermissionKey(ResourceRBACPermission, ActionList)},
		},
	}
}

func (h *Handler) createRole(c *gin.Context) {
	var req CreateRoleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	role, err := h.svc.CreateRole(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, role)
}

func (h *Handler) updateRole(c *gin.Context) {
	var req struct {
		ID          string `json:"id" binding:"required"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	roleID, err := uuid.Parse(req.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	updated, err := h.svc.UpdateRole(c.Request.Context(), UpdateRoleInput{ID: roleID, Name: req.Name, Description: req.Description})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, ErrResourceNotFound)
			return
		}
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	response.Success(c, updated)
}

func (h *Handler) deleteRole(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	roleID, err := uuid.Parse(req.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	if err := h.svc.DeleteRole(c.Request.Context(), DeleteRoleInput{ID: roleID}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, ErrResourceNotFound)
			return
		}
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	response.Success(c, gin.H{"id": roleID})
}

func (h *Handler) listRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, roles)
}

func (h *Handler) assignRolePermissions(c *gin.Context) {
	var req struct {
		RoleID      string   `json:"roleId" binding:"required"`
		Permissions []string `json:"permissions" binding:"required,min=1,dive,required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	role, err := h.svc.AssignPermissions(c.Request.Context(), AssignRolePermissionsInput{RoleID: roleID, Permissions: req.Permissions})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, ErrResourceNotFound)
			return
		}
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	response.Success(c, role)
}

func (h *Handler) getRolePermissions(c *gin.Context) {
	var req struct {
		RoleID string `json:"roleId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	permissions, err := h.svc.GetRolePermissions(c.Request.Context(), roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, ErrResourceNotFound)
			return
		}
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	response.Success(c, gin.H{"roleId": roleID, "permissions": permissions})
}

func (h *Handler) createPermission(c *gin.Context) {
	var req CreatePermissionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	permission, err := h.svc.CreatePermission(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, permission)
}

func (h *Handler) updatePermission(c *gin.Context) {
	var req struct {
		ID          string `json:"id" binding:"required"`
		Resource    string `json:"resource"`
		Action      string `json:"action"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	permissionID, err := uuid.Parse(req.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	updated, err := h.svc.UpdatePermission(c.Request.Context(), UpdatePermissionInput{ID: permissionID, Resource: req.Resource, Action: req.Action, Description: req.Description})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, ErrResourceNotFound)
			return
		}
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	response.Success(c, updated)
}

func (h *Handler) deletePermission(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	permissionID, err := uuid.Parse(req.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidPayload)
		return
	}

	if err := h.svc.DeletePermission(c.Request.Context(), DeletePermissionInput{ID: permissionID}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, ErrResourceNotFound)
			return
		}
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	response.Success(c, gin.H{"id": permissionID})
}

func (h *Handler) listPermissions(c *gin.Context) {
	permissions, err := h.svc.ListPermissions(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.Success(c, permissions)
}
