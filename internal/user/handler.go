package user

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Jayleonc/service/internal/auth"
	"github.com/Jayleonc/service/internal/middleware"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/response"
)

// Handler exposes user endpoints.
type Handler struct {
	svc     *Service
	authSvc *auth.Service
}

// NewHandler constructs a Handler following the DI-oriented paradigm.
func NewHandler(svc *Service, authSvc *auth.Service) *Handler {
	return &Handler{svc: svc, authSvc: authSvc}
}

// RegisterRoutes registers user routes under the provided router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.POST("/register", h.register)
	users.POST("/login", h.login)

	authenticated := users.Group("")
	authenticated.Use(middleware.Authenticated(h.authSvc))
	authenticated.POST("/me/get", h.me)
	authenticated.POST("/me/update", h.updateMe)
	authenticated.POST("/roles/update", middleware.RBAC(constant.RoleAdmin), h.updateRoles)
}

type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone" binding:"omitempty"`
}

type registerResponse struct {
	User profileResponse `json:"user"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    int64           `json:"expires_in"`
	User         profileResponse `json:"user"`
}

type profileResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Phone       string    `json:"phone"`
	DateCreated string    `json:"date_created"`
	DateUpdated string    `json:"date_updated"`
}

type updateProfileRequest struct {
	Name  *string `json:"name" binding:"omitempty"`
	Phone *string `json:"phone" binding:"omitempty"`
}

type updateRolesRequest struct {
	UserID string   `json:"user_id" binding:"required"`
	Roles  []string `json:"roles" binding:"required,min=1,dive,required"`
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 3001, "invalid request payload")
		return
	}

	profile, err := h.svc.Register(c.Request.Context(), RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Phone:    req.Phone,
	})
	if err != nil {
		status := http.StatusBadRequest
		code := 3002
		if err == ErrEmailExists {
			code = 3003
		}
		response.Error(c, status, code, err.Error())
		return
	}

	response.SuccessWithStatus(c, http.StatusCreated, registerResponse{User: toProfileResponse(profile)})
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 3011, "invalid request payload")
		return
	}

	result, err := h.svc.Login(c.Request.Context(), LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
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

	response.Success(c, loginResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		ExpiresIn:    int64(result.Tokens.ExpiresIn.Seconds()),
		User:         toProfileResponse(result.Profile),
	})
}

func (h *Handler) me(c *gin.Context) {
	if err := bindOptionalJSON(c, &struct{}{}); err != nil {
		response.Error(c, http.StatusBadRequest, 3020, "invalid request payload")
		return
	}

	session, ok := auth.SessionFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 3021, "missing session")
		return
	}

	profile, err := h.svc.Profile(c.Request.Context(), session.UserID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 3022, err.Error())
		return
	}

	response.Success(c, toProfileResponse(profile))
}

func (h *Handler) updateMe(c *gin.Context) {
	session, ok := auth.SessionFromContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, 3031, "missing session")
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 3032, "invalid request payload")
		return
	}

	profile, err := h.svc.UpdateProfile(c.Request.Context(), session.UserID, UpdateProfileInput{
		Name:  req.Name,
		Phone: req.Phone,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, 3033, err.Error())
		return
	}

	response.Success(c, toProfileResponse(profile))
}

func (h *Handler) updateRoles(c *gin.Context) {
	var req updateRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 3044, "invalid request payload")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 3043, "invalid user id")
		return
	}

	profile, err := h.svc.UpdateRoles(c.Request.Context(), userID, req.Roles)
	if err != nil {
		status := http.StatusBadRequest
		code := 3045
		if err == ErrRolesRequired {
			code = 3046
		}
		response.Error(c, status, code, err.Error())
		return
	}

	response.Success(c, toProfileResponse(profile))
}

func toProfileResponse(profile Profile) profileResponse {
	return profileResponse{
		ID:          profile.ID,
		Name:        profile.Name,
		Email:       profile.Email,
		Roles:       profile.Roles,
		Phone:       profile.Phone,
		DateCreated: profile.DateCreated.Format(time.RFC3339),
		DateUpdated: profile.DateUpdated.Format(time.RFC3339),
	}
}

func bindOptionalJSON(c *gin.Context, dest any) error {
	if c.Request == nil || c.Request.Body == nil {
		return nil
	}
	if err := c.ShouldBindJSON(dest); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		var syntaxError *json.SyntaxError
		if errors.As(err, &syntaxError) {
			return err
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return err
		}
		// Gin wraps EOF errors, so perform string comparison as a fallback.
		if err.Error() == "EOF" {
			return nil
		}
		return err
	}
	return nil
}
