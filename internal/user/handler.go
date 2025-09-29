package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/ginx/request"
	"github.com/Jayleonc/service/pkg/ginx/response"
)

// Handler 对外提供用户模块的 HTTP 接口。
type Handler struct {
	svc *Service
}

// NewHandler 依赖注入用户服务后返回处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetRoutes 声明用户模块所需的路由。
func (h *Handler) GetRoutes() feature.ModuleRoutes {
	return feature.ModuleRoutes{
		PublicRoutes: []feature.RouteDefinition{
			{Path: "register", Handler: h.register},
			{Path: "login", Handler: h.login},
		},
		AuthenticatedRoutes: []feature.RouteDefinition{
			{Path: "me/get", Handler: h.me},
			{Path: "me/update", Handler: h.updateMe},
			{Path: "create", Handler: h.create, RequiredPermission: "user:create"},
			{Path: "update", Handler: h.update, RequiredPermission: "user:update"},
			{Path: "delete", Handler: h.delete, RequiredPermission: "user:delete"},
			{Path: "list", Handler: h.list, RequiredPermission: "user:list"},
			{Path: "assign_roles", Handler: h.assignRoles, RequiredPermission: "user:assign_roles"},
		},
	}
}

func (h *Handler) register(c *gin.Context) {
	var req RegisterInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrRegisterInvalidPayload)
		return
	}

	profile, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			response.Error(c, http.StatusBadRequest, ErrEmailExists)
			return
		}
		response.Error(c, http.StatusBadRequest, ErrRegisterFailed)
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, profile)
}

func (h *Handler) login(c *gin.Context) {
	var req LoginInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrLoginInvalidPayload)
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, ErrInvalidCredentials)
			return
		}
		response.Error(c, http.StatusBadRequest, ErrLoginFailed)
		return
	}

	response.Success(c, gin.H{
		"access_token":  result.Tokens.AccessToken,
		"refresh_token": result.Tokens.RefreshToken,
		"expires_in":    int64(result.Tokens.ExpiresIn.Seconds()),
		"user":          result.Profile,
	})
}

func (h *Handler) me(c *gin.Context) {
	session := feature.MustGetAuthContext(c)

	profile, err := h.svc.Profile(c.Request.Context(), session.UserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, ErrProfileLookupFailed)
		return
	}

	response.Success(c, profile)
}

func (h *Handler) updateMe(c *gin.Context) {
	session := feature.MustGetAuthContext(c)

	var req UpdateProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrUpdateMeInvalidBody)
		return
	}

	profile, err := h.svc.UpdateProfile(c.Request.Context(), session.UserID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrUpdateProfileFailed)
		return
	}

	response.Success(c, profile)
}

func (h *Handler) create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrCreateInvalidPayload)
		return
	}

	profile, err := h.svc.CreateUser(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			response.Error(c, http.StatusBadRequest, ErrEmailExists)
			return
		}
		response.Error(c, http.StatusBadRequest, ErrCreateFailed)
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, profile)
}

func (h *Handler) update(c *gin.Context) {
	var payload struct {
		ID    string  `json:"id" binding:"required"`
		Name  *string `json:"name"`
		Phone *string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, ErrUpdateInvalidPayload)
		return
	}

	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrInvalidUserID)
		return
	}

	profile, err := h.svc.UpdateUser(c.Request.Context(), UpdateUserRequest{
		ID:    userID,
		Name:  payload.Name,
		Phone: payload.Phone,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrUpdateUserFailed)
		return
	}

	response.Success(c, profile)
}

func (h *Handler) delete(c *gin.Context) {
	var payload struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, ErrDeleteInvalidPayload)
		return
	}

	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrDeleteInvalidUserID)
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), DeleteUserRequest{ID: userID}); err != nil {
		response.Error(c, http.StatusBadRequest, ErrDeleteUserFailed)
		return
	}

	response.Success(c, gin.H{"id": userID})
}

func (h *Handler) list(c *gin.Context) {
	var payload struct {
		Pagination request.Pagination `json:"pagination"`
		Name       string             `json:"name"`
		Email      string             `json:"email"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, ErrListInvalidPayload)
		return
	}

	result, err := h.svc.ListUsers(c.Request.Context(), ListUsersRequest{
		Pagination: payload.Pagination,
		Name:       payload.Name,
		Email:      payload.Email,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, ErrListUsersFailed)
		return
	}

	response.Success(c, result)
}

func (h *Handler) assignRoles(c *gin.Context) {
	var payload struct {
		ID    string   `json:"id" binding:"required"`
		Roles []string `json:"roles" binding:"required,min=1,dive,required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, ErrAssignInvalidPayload)
		return
	}

	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrAssignInvalidUserID)
		return
	}

	profile, err := h.svc.AssignRoles(c.Request.Context(), AssignRolesRequest{ID: userID, Roles: payload.Roles})
	if err != nil {
		response.Error(c, http.StatusBadRequest, ErrAssignRolesFailed)
		return
	}

	response.Success(c, profile)
}
