package role

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/response"
)

// Handler 提供角色管理接口
type Handler struct {
	svc *Service
}

// NewHandler 创建角色 Handler
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetRoutes 返回角色功能的路由定义
func (h *Handler) GetRoutes() feature.ModuleRoutes {
	return feature.ModuleRoutes{
		AdminRoutes: []feature.RouteDefinition{
			{Path: "/roles/create", Handler: h.create},
			{Path: "/roles/update", Handler: h.update},
			{Path: "/roles/delete", Handler: h.delete},
			{Path: "/roles/list", Handler: h.list},
		},
	}
}

func (h *Handler) create(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 4001, "invalid request payload")
		return
	}

	role, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4002, err.Error())
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, role)
}

func (h *Handler) update(c *gin.Context) {
	var payload struct {
		ID          string  `json:"id" binding:"required"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, 4011, "invalid request payload")
		return
	}

	roleID, err := uuid.Parse(payload.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4012, "invalid role id")
		return
	}

	result, err := h.svc.Update(c.Request.Context(), UpdateRoleRequest{
		ID:          roleID,
		Name:        payload.Name,
		Description: payload.Description,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4013, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *Handler) delete(c *gin.Context) {
	var payload struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, 4021, "invalid request payload")
		return
	}

	roleID, err := uuid.Parse(payload.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4022, "invalid role id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), DeleteRoleRequest{ID: roleID}); err != nil {
		response.Error(c, http.StatusBadRequest, 4023, err.Error())
		return
	}

	response.Success(c, gin.H{"id": roleID})
}

func (h *Handler) list(c *gin.Context) {
	var payload struct{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, 4031, "invalid request payload")
		return
	}

	roles, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 4032, err.Error())
		return
	}

	response.Success(c, roles)
}
