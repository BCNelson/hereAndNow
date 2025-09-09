package api

import (
	"net/http"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	locationService LocationService
}

type LocationService interface {
	GetLocationsByUserID(userID string) ([]models.Location, error)
	CreateLocation(location models.Location) (*models.Location, error)
	GetLocationByID(locationID string) (*models.Location, error)
	UpdateLocation(location models.Location) (*models.Location, error)
	DeleteLocation(locationID string, userID string) error
}

type LocationCreateRequest struct {
	Name      string   `json:"name" binding:"required"`
	Address   string   `json:"address"`
	Latitude  float64  `json:"latitude" binding:"required"`
	Longitude float64  `json:"longitude" binding:"required"`
	Radius    int      `json:"radius"`
	Category  string   `json:"category"`
	PlaceID   *string  `json:"place_id"`
}

func NewLocationHandler(locationService LocationService) *LocationHandler {
	return &LocationHandler{
		locationService: locationService,
	}
}

// GetLocations handles GET /locations - get user's saved locations
func (h *LocationHandler) GetLocations(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	locations, err := h.locationService.GetLocationsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get locations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"locations": locations,
		"total":     len(locations),
	})
}

// CreateLocation handles POST /locations - create new location with GPS validation
func (h *LocationHandler) CreateLocation(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	var req LocationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Set default radius if not provided
	if req.Radius == 0 {
		req.Radius = 100 // Default 100 meters
	}

	// Set default category if not provided
	if req.Category == "" {
		req.Category = "other"
	}

	// Create location model using NewLocation constructor for validation
	location, err := models.NewLocation(user.ID, req.Name, req.Address, req.Latitude, req.Longitude, req.Radius)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid location data",
			Details: err.Error(),
		})
		return
	}

	// Set optional fields
	location.Category = req.Category
	location.PlaceID = req.PlaceID

	// Create location
	createdLocation, err := h.locationService.CreateLocation(*location)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to create location",
		})
		return
	}

	c.JSON(http.StatusCreated, createdLocation)
}