package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService AnalyticsService
}

type AnalyticsService interface {
	GetAnalyticsByUserIDAndDateRange(userID string, startDate, endDate time.Time) ([]models.Analytics, error)
	GetAnalyticsByUserIDAndDate(userID string, date time.Time) (*models.Analytics, error)
	CreateAnalytics(analytics models.Analytics) (*models.Analytics, error)
	UpdateAnalytics(analytics models.Analytics) (*models.Analytics, error)
	GetProductivitySummary(userID string, startDate, endDate time.Time) (*ProductivitySummary, error)
}

type ProductivitySummary struct {
	Period              string                   `json:"period"`
	StartDate           string                   `json:"start_date"`
	EndDate             string                   `json:"end_date"`
	TotalTasks          int                      `json:"total_tasks"`
	TasksCompleted      int                      `json:"tasks_completed"`
	TasksCancelled      int                      `json:"tasks_cancelled"`
	AverageCompletionRate float64               `json:"average_completion_rate"`
	AverageProductivityScore float64           `json:"average_productivity_score"`
	TotalMinutesEstimated int                   `json:"total_minutes_estimated"`
	TotalMinutesActual   int                    `json:"total_minutes_actual"`
	EstimationAccuracy   float64                `json:"estimation_accuracy"`
	LocationChanges      int                    `json:"location_changes"`
	DailyBreakdown      []map[string]interface{} `json:"daily_breakdown"`
	Trends              map[string]interface{}   `json:"trends"`
}

func NewAnalyticsHandler(analyticsService AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetAnalytics handles GET /analytics - get productivity analytics with date ranges
func (h *AnalyticsHandler) GetAnalytics(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	// Parse query parameters
	period := c.DefaultQuery("period", "week") // day, week, month, year, custom
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	summary := c.DefaultQuery("summary", "true") == "true"

	// Calculate date range based on period
	var startDate, endDate time.Time
	now := time.Now()

	if period == "custom" && startDateStr != "" && endDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid start_date format",
				Details: "Use YYYY-MM-DD format",
			})
			return
		}

		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid end_date format",
				Details: "Use YYYY-MM-DD format",
			})
			return
		}
	} else {
		switch period {
		case "day":
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endDate = startDate.Add(24 * time.Hour).Add(-time.Second)
		case "week":
			// Start of week (Monday)
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7 // Sunday = 7
			}
			startDate = now.AddDate(0, 0, -weekday+1)
			startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
			endDate = startDate.AddDate(0, 0, 7).Add(-time.Second)
		case "month":
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
		case "year":
			startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
			endDate = startDate.AddDate(1, 0, 0).Add(-time.Second)
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid period",
				Details: "Valid periods: day, week, month, year, custom",
			})
			return
		}
	}

	// Validate date range
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "End date must be after start date",
		})
		return
	}

	// Check if date range is too large (max 1 year)
	maxDuration := 365 * 24 * time.Hour
	if endDate.Sub(startDate) > maxDuration {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Date range too large (maximum 1 year)",
		})
		return
	}

	if summary {
		// Get productivity summary
		productivitySummary, err := h.analyticsService.GetProductivitySummary(userID, startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to get productivity summary",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"period":     period,
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
			"summary":    productivitySummary,
		})
	} else {
		// Get detailed analytics
		analytics, err := h.analyticsService.GetAnalyticsByUserIDAndDateRange(userID, startDate, endDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to get analytics",
			})
			return
		}

		// Parse limit for detailed view
		limit := 100 // Default
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
				limit = parsedLimit
			}
		}

		// Limit results if needed
		if len(analytics) > limit {
			analytics = analytics[:limit]
		}

		c.JSON(http.StatusOK, gin.H{
			"period":     period,
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
			"analytics":  analytics,
			"total":      len(analytics),
		})
	}
}