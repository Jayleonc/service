package role

import (
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"

        "github.com/Jayleonc/service/pkg/database"
)

type assignment struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID `gorm:"type:uuid;index"`
	Role        string
	DateCreated time.Time
}

func (a *assignment) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.DateCreated.IsZero() {
		a.DateCreated = time.Now().UTC()
	}
	return nil
}

// Handler demonstrates the lightweight, singleton-driven development path.
type Handler struct{}

// NewHandler constructs a Handler without any explicit dependency wiring.
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes attaches the role endpoints to the shared API group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/roles", h.create)
}

type createRoleRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (h *Handler) create(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	db := database.Default()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialised"})
		return
	}

	record := assignment{UserID: userID, Role: req.Role}
	if err := db.WithContext(c.Request.Context()).Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           record.ID,
		"user_id":      record.UserID,
		"role":         record.Role,
		"date_created": record.DateCreated,
	})
}
