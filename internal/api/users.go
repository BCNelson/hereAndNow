package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userRepo UserRepository
}

type UserRepository interface {
	GetByID(userID string) (*models.User, error)
	Update(user *models.User) error
}

func NewUserHandler(userRepo UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

type UserResponse struct {
	ID          string          `json:"id"`
	Username    string          `json:"username"`
	Email       string          `json:"email"`
	DisplayName string          `json:"display_name"`
	TimeZone    string          `json:"timezone"`
	CreatedAt   time.Time       `json:"created_at"`
	Settings    json.RawMessage `json:"settings"`
}

type UserUpdateRequest struct {
	DisplayName *string         `json:"display_name,omitempty"`
	TimeZone    *string         `json:"timezone,omitempty"`
	Settings    json.RawMessage `json:"settings,omitempty"`
}

// GetMe handles GET /users/me
func (h *UserHandler) GetMe(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	response := UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		TimeZone:    user.TimeZone,
		CreatedAt:   user.CreatedAt,
		Settings:    user.Settings,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateMe handles PATCH /users/me
func (h *UserHandler) UpdateMe(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	var req UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Update fields if provided
	updated := false
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
		updated = true
	}

	if req.TimeZone != nil {
		// Validate timezone
		if _, err := time.LoadLocation(*req.TimeZone); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid timezone",
				Details: err.Error(),
			})
			return
		}
		user.TimeZone = *req.TimeZone
		updated = true
	}

	if req.Settings != nil {
		// Validate JSON
		var settings map[string]interface{}
		if err := json.Unmarshal(req.Settings, &settings); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid settings format",
				Details: err.Error(),
			})
			return
		}
		user.Settings = req.Settings
		updated = true
	}

	if updated {
		user.UpdatedAt = time.Now()
		if err := h.userRepo.Update(user); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to update user",
			})
			return
		}
	}

	response := UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		TimeZone:    user.TimeZone,
		CreatedAt:   user.CreatedAt,
		Settings:    user.Settings,
	}

	c.JSON(http.StatusOK, response)
}