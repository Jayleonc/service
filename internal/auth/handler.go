package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/response"
)

// Handler exposes the authentication refresh endpoint.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Handler following the DI-oriented paradigm.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetRoutes 声明认证功能的路由
func (h *Handler) GetRoutes() feature.ModuleRoutes {
	return feature.ModuleRoutes{
		PublicRoutes: []feature.RouteDefinition{
			{Path: "/auth/refresh", Handler: h.refresh},
		},
	}
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type refreshResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, ErrRefreshInvalidPayload)
		return
	}

	tokens, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			response.Error(c, http.StatusUnauthorized, ErrInvalidRefreshToken)
			return
		}
		response.Error(c, http.StatusInternalServerError, ErrRefreshFailed)
		return
	}

	response.Success(c, refreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(tokens.ExpiresIn.Seconds()),
	})
}
