package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ContextHandler struct {
	contextService ContextService
}

type ContextUpdateRequest struct {
	CurrentLatitude   *float64 `json:"current_latitude"`
	CurrentLongitude  *float64 `json:"current_longitude"`
	CurrentLocationID *string  `json:"current_location_id"`
	AvailableMinutes  *int     `json:"available_minutes"`
	SocialContext     *string  `json:"social_context"`
	EnergyLevel       *int     `json:"energy_level"`
	WeatherCondition  *string  `json:"weather_condition"`
	TrafficLevel      *string  `json:"traffic_level"`
}

func NewContextHandler(contextService ContextService) *ContextHandler {
	return &ContextHandler{
		contextService: contextService,
	}
}

// GetContext handles GET /context - get current user context
func (h *ContextHandler) GetContext(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	context, err := h.contextService.GetCurrentContext(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get current context",
		})
		return
	}

	c.JSON(http.StatusOK, context)
}

// UpdateContext handles POST /context - update user context
func (h *ContextHandler) UpdateContext(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	var req ContextUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Get current context to update
	context, err := h.contextService.GetCurrentContext(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get current context",
		})
		return
	}

	// Apply updates
	if req.CurrentLatitude != nil && req.CurrentLongitude != nil {
		if err := context.SetCurrentPosition(*req.CurrentLatitude, *req.CurrentLongitude); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid coordinates",
				Details: err.Error(),
			})
			return
		}
	} else if req.CurrentLatitude == nil && req.CurrentLongitude == nil {
		// Both nil means clear position
		context.ClearCurrentPosition()
	}

	if req.CurrentLocationID != nil {
		if *req.CurrentLocationID == "" {
			context.ClearCurrentLocation()
		} else {
			context.SetCurrentLocation(*req.CurrentLocationID)
		}
	}

	if req.AvailableMinutes != nil {
		if err := context.SetAvailableMinutes(*req.AvailableMinutes); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid available minutes",
				Details: err.Error(),
			})
			return
		}
	}

	if req.SocialContext != nil {
		if err := context.SetSocialContext(*req.SocialContext); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid social context",
				Details: err.Error(),
			})
			return
		}
	}

	if req.EnergyLevel != nil {
		if err := context.SetEnergyLevel(*req.EnergyLevel); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid energy level",
				Details: err.Error(),
			})
			return
		}
	}

	if req.WeatherCondition != nil {
		if *req.WeatherCondition == "" {
			context.ClearWeatherCondition()
		} else {
			if err := context.SetWeatherCondition(*req.WeatherCondition); err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "Invalid weather condition",
					Details: err.Error(),
				})
				return
			}
		}
	}

	if req.TrafficLevel != nil {
		if *req.TrafficLevel == "" {
			context.ClearTrafficLevel()
		} else {
			if err := context.SetTrafficLevel(*req.TrafficLevel); err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "Invalid traffic level",
					Details: err.Error(),
				})
				return
			}
		}
	}

	// Update context
	updatedContext, err := h.contextService.UpdateContext(*context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to update context",
		})
		return
	}

	c.JSON(http.StatusOK, updatedContext)
}