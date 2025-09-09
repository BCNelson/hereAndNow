package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Analytics struct {
	ID               string          `db:"id" json:"id"`
	UserID           string          `db:"user_id" json:"user_id"`
	Date             time.Time       `db:"date" json:"date"`
	TasksCreated     int             `db:"tasks_created" json:"tasks_created"`
	TasksCompleted   int             `db:"tasks_completed" json:"tasks_completed"`
	TasksCancelled   int             `db:"tasks_cancelled" json:"tasks_cancelled"`
	MinutesEstimated int             `db:"minutes_estimated" json:"minutes_estimated"`
	MinutesActual    int             `db:"minutes_actual" json:"minutes_actual"`
	LocationChanges  int             `db:"location_changes" json:"location_changes"`
	Metadata         json.RawMessage `db:"metadata" json:"metadata"`
}

type AnalyticsMetadata struct {
	MostUsedLocation   string            `json:"most_used_location,omitempty"`
	PeakProductiveHour int               `json:"peak_productive_hour,omitempty"`
	AvgTaskPriority    float64           `json:"avg_task_priority,omitempty"`
	CompletionRate     float64           `json:"completion_rate,omitempty"`
	EstimationAccuracy float64           `json:"estimation_accuracy,omitempty"`
	CategoryBreakdown  map[string]int    `json:"category_breakdown,omitempty"`
	TimeSpentByHour    map[string]int    `json:"time_spent_by_hour,omitempty"`
	CustomMetrics      map[string]interface{} `json:"custom_metrics,omitempty"`
}

func NewAnalytics(userID string, date time.Time) (*Analytics, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	return &Analytics{
		ID:               uuid.New().String(),
		UserID:           userID,
		Date:             dateOnly,
		TasksCreated:     0,
		TasksCompleted:   0,
		TasksCancelled:   0,
		MinutesEstimated: 0,
		MinutesActual:    0,
		LocationChanges:  0,
		Metadata:         json.RawMessage(`{}`),
	}, nil
}

func (a *Analytics) IncrementTasksCreated() {
	a.TasksCreated++
}

func (a *Analytics) IncrementTasksCompleted() {
	a.TasksCompleted++
}

func (a *Analytics) IncrementTasksCancelled() {
	a.TasksCancelled++
}

func (a *Analytics) AddEstimatedMinutes(minutes int) {
	if minutes > 0 {
		a.MinutesEstimated += minutes
	}
}

func (a *Analytics) AddActualMinutes(minutes int) {
	if minutes > 0 {
		a.MinutesActual += minutes
	}
}

func (a *Analytics) IncrementLocationChanges() {
	a.LocationChanges++
}

func (a *Analytics) GetMetadata() (*AnalyticsMetadata, error) {
	var metadata AnalyticsMetadata
	if err := json.Unmarshal(a.Metadata, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	return &metadata, nil
}

func (a *Analytics) SetMetadata(metadata *AnalyticsMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	a.Metadata = metadataJSON
	return nil
}

func (a *Analytics) CalculateCompletionRate() float64 {
	totalTasks := a.TasksCreated + a.TasksCompleted
	if totalTasks == 0 {
		return 0.0
	}
	return float64(a.TasksCompleted) / float64(totalTasks)
}

func (a *Analytics) CalculateEstimationAccuracy() float64 {
	if a.MinutesEstimated == 0 || a.MinutesActual == 0 {
		return 0.0
	}

	estimated := float64(a.MinutesEstimated)
	actual := float64(a.MinutesActual)
	
	difference := estimated - actual
	if difference < 0 {
		difference = -difference
	}
	
	accuracy := 1.0 - (difference / estimated)
	if accuracy < 0 {
		accuracy = 0.0
	}
	
	return accuracy
}

func (a *Analytics) GetTotalTasks() int {
	return a.TasksCreated + a.TasksCompleted + a.TasksCancelled
}

func (a *Analytics) IsOwnedBy(userID string) bool {
	return a.UserID == userID
}

func (a *Analytics) IsForDate(date time.Time) bool {
	targetDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	return a.Date.Equal(targetDate)
}

func (a *Analytics) IsToday() bool {
	return a.IsForDate(time.Now())
}

func (a *Analytics) IsYesterday() bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return a.IsForDate(yesterday)
}

func (a *Analytics) IsThisWeek() bool {
	now := time.Now()
	_, week := now.ISOWeek()
	_, analyticsWeek := a.Date.ISOWeek()
	return week == analyticsWeek && now.Year() == a.Date.Year()
}

func (a *Analytics) IsThisMonth() bool {
	now := time.Now()
	return now.Month() == a.Date.Month() && now.Year() == a.Date.Year()
}

func (a *Analytics) GetProductivityScore() float64 {
	if a.GetTotalTasks() == 0 {
		return 0.0
	}

	completionRate := a.CalculateCompletionRate()
	estimationAccuracy := a.CalculateEstimationAccuracy()
	
	efficiencyBonus := 0.0
	if a.MinutesActual > 0 && a.MinutesEstimated > 0 {
		if a.MinutesActual <= a.MinutesEstimated {
			efficiencyBonus = 0.2
		}
	}

	score := (completionRate * 0.6) + (estimationAccuracy * 0.3) + efficiencyBonus
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

func (a *Analytics) GenerateSummary() map[string]interface{} {
	return map[string]interface{}{
		"date":                a.Date.Format("2006-01-02"),
		"total_tasks":         a.GetTotalTasks(),
		"tasks_created":       a.TasksCreated,
		"tasks_completed":     a.TasksCompleted,
		"tasks_cancelled":     a.TasksCancelled,
		"minutes_estimated":   a.MinutesEstimated,
		"minutes_actual":      a.MinutesActual,
		"location_changes":    a.LocationChanges,
		"completion_rate":     a.CalculateCompletionRate(),
		"estimation_accuracy": a.CalculateEstimationAccuracy(),
		"productivity_score":  a.GetProductivityScore(),
	}
}

func (a *Analytics) Validate() error {
	if a.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if a.TasksCreated < 0 {
		return fmt.Errorf("tasks created cannot be negative")
	}

	if a.TasksCompleted < 0 {
		return fmt.Errorf("tasks completed cannot be negative")
	}

	if a.TasksCancelled < 0 {
		return fmt.Errorf("tasks cancelled cannot be negative")
	}

	if a.MinutesEstimated < 0 {
		return fmt.Errorf("estimated minutes cannot be negative")
	}

	if a.MinutesActual < 0 {
		return fmt.Errorf("actual minutes cannot be negative")
	}

	if a.LocationChanges < 0 {
		return fmt.Errorf("location changes cannot be negative")
	}

	_, err := a.GetMetadata()
	if err != nil {
		return fmt.Errorf("invalid metadata JSON: %w", err)
	}

	return nil
}