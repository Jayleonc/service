package role

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Jayleonc/service/pkg/database"
	"github.com/Jayleonc/service/pkg/response"
)

// Handler demonstrates the lightweight, singleton-driven development path.
type Handler struct{}

type replaceRolesRequest struct {
	UserID string   `json:"user_id" binding:"required"`
	Roles  []string `json:"roles" binding:"required,min=1,dive,required"`
}

func (h *Handler) list(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 4010, "invalid request payload")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4011, "invalid user id")
		return
	}

	roles, err := List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 4012, err.Error())
		return
	}

	response.Success(c, gin.H{
		"user_id": userID,
		"roles":   roles,
	})
}

func (h *Handler) replace(c *gin.Context) {
	var req replaceRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 4022, "invalid request payload")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4021, "invalid user id")
		return
	}

	if err := Assign(c.Request.Context(), userID, req.Roles); err != nil {
		response.Error(c, http.StatusInternalServerError, 4023, err.Error())
		return
	}

	roles, err := List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 4024, err.Error())
		return
	}

	response.Success(c, gin.H{
		"user_id": userID,
		"roles":   roles,
	})
}

// NewHandler constructs a Handler without any explicit dependency wiring.
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes attaches the role endpoints to the shared API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/roles", h.create)
	rg.POST("/roles/get", h.list)
	rg.POST("/roles/replace", h.replace)
}

type createRoleRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (h *Handler) create(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 4001, "invalid request payload")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 4002, "invalid user id")
		return
	}

	db := database.Default()
	if db == nil {
		response.Error(c, http.StatusInternalServerError, 4003, "database not initialised")
		return
	}

	record := Role{ID: uuid.New(), UserID: userID, Role: req.Role, DateCreated: time.Now().UTC()}
	if err := db.WithContext(c.Request.Context()).Create(&record).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, 4004, err.Error())
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, gin.H{
		"id":           record.ID,
		"user_id":      record.UserID,
		"role":         record.Role,
		"date_created": record.DateCreated,
	})
}
