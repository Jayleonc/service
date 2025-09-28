package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/request"
	"github.com/Jayleonc/service/pkg/response"
)

// Handler exposes user endpoints.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Handler following the DI-oriented paradigm.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetRoutes returns the route declarations for the user feature.
func (h *Handler) GetRoutes() feature.ModuleRoutes {
	return feature.ModuleRoutes{
		PublicRoutes: []feature.RouteDefinition{
			{Path: "/users/register", Handler: h.register},
			{Path: "/users/login", Handler: h.login},
		},
		AuthenticatedRoutes: []feature.RouteDefinition{
			{Path: "/users/me/get", Handler: h.me},
			{Path: "/users/me/update", Handler: h.updateMe},
		},
		AdminRoutes: []feature.RouteDefinition{
			{Path: "/users/create", Handler: h.create},
			{Path: "/users/update", Handler: h.update},
			{Path: "/users/delete", Handler: h.delete},
			{Path: "/users/list", Handler: h.list},
			{Path: "/users/assign_roles", Handler: h.assignRoles},
		},
	}
}

func (h *Handler) register(c *gin.Context) {
	var req RegisterInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 3001, "invalid request payload")
		return
	}

	profile, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		code := 3002
		if err == ErrEmailExists {
			code = 3003
		}
		response.Error(c, status, code, err.Error())
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, profile)
}

func (h *Handler) login(c *gin.Context) {
	var req LoginInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 3011, "invalid request payload")
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		code := 3012
		status := http.StatusBadRequest
		if err == ErrInvalidCredentials {
			status = http.StatusUnauthorized
			code = 3013
		}
		response.Error(c, status, code, err.Error())
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
	session, ok := feature.AuthContextFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 3021, "missing session")
		return
	}

	profile, err := h.svc.Profile(c.Request.Context(), session.UserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 3022, err.Error())
		return
	}

	response.Success(c, profile)
}

func (h *Handler) updateMe(c *gin.Context) {
	session, ok := feature.AuthContextFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 3031, "missing session")
		return
	}

	var req UpdateProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 3032, "invalid request payload")
		return
	}

	profile, err := h.svc.UpdateProfile(c.Request.Context(), session.UserID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 3033, err.Error())
		return
	}

	response.Success(c, profile)
}

func (h *Handler) create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 3041, "invalid request payload")
		return
	}

	profile, err := h.svc.CreateUser(c.Request.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		code := 3042
		if err == ErrEmailExists {
			code = 3043
		}
		response.Error(c, status, code, err.Error())
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
		response.Error(c, http.StatusBadRequest, 3051, "invalid request payload")
		return
	}

	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 3052, "invalid user id")
		return
	}

	profile, err := h.svc.UpdateUser(c.Request.Context(), UpdateUserRequest{
		ID:    userID,
		Name:  payload.Name,
		Phone: payload.Phone,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, 3053, err.Error())
		return
	}

	response.Success(c, profile)
}

func (h *Handler) delete(c *gin.Context) {
	var payload struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, 3061, "invalid request payload")
		return
	}

	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 3062, "invalid user id")
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), DeleteUserRequest{ID: userID}); err != nil {
		response.Error(c, http.StatusBadRequest, 3063, err.Error())
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
		response.Error(c, http.StatusBadRequest, 3071, "invalid request payload")
		return
	}

	result, err := h.svc.ListUsers(c.Request.Context(), ListUsersRequest{
		Pagination: payload.Pagination,
		Name:       payload.Name,
		Email:      payload.Email,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 3072, err.Error())
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
		response.Error(c, http.StatusBadRequest, 3081, "invalid request payload")
		return
	}

	userID, err := uuid.Parse(payload.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 3082, "invalid user id")
		return
	}

	profile, err := h.svc.AssignRoles(c.Request.Context(), AssignRolesRequest{ID: userID, Roles: payload.Roles})
	if err != nil {
		response.Error(c, http.StatusBadRequest, 3083, err.Error())
		return
	}

	response.Success(c, profile)
}
