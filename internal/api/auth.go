package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/bcnelson/hereAndNow/internal/auth"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *auth.AuthService
}

func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string      `json:"token"`
	User      models.User `json:"user"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type ErrorResponse struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	authReq := auth.LoginRequest{
		Email:    req.Username, // Using Email field to pass username/email 
		Password: req.Password,
	}

	loginResp, err := h.authService.Login(authReq, userAgent, ipAddress)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid credentials",
			})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Authentication failed",
			})
		}
		return
	}

	response := LoginResponse{
		Token:     loginResp.Token,
		User:      loginResp.User,
		ExpiresAt: loginResp.ExpiresAt,
	}

	c.JSON(http.StatusOK, response)
}

// Logout handles POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authorization header required",
		})
		return
	}

	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Invalid authorization header format",
		})
		return
	}

	token := tokenParts[1]

	if err := h.authService.Logout(token); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Logout failed",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// AuthMiddleware validates JWT tokens and sets user context
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Authorization header required",
			})
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]
		user, err := h.authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user in context for downstream handlers
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	}
}

// GetCurrentUser returns the authenticated user from context
func GetCurrentUser(c *gin.Context) (*models.User, error) {
	user, exists := c.Get("user")
	if !exists {
		return nil, http.ErrNoCookie
	}

	u, ok := user.(*models.User)
	if !ok {
		return nil, http.ErrNoCookie
	}

	return u, nil
}

// GetCurrentUserID returns the authenticated user ID from context
func GetCurrentUserID(c *gin.Context) (string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", http.ErrNoCookie
	}

	id, ok := userID.(string)
	if !ok {
		return "", http.ErrNoCookie
	}

	return id, nil
}