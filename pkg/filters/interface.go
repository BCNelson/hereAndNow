package filters

import (
	"github.com/bcnelson/hereAndNow/pkg/models"
)

type FilterRule interface {
	Apply(ctx models.Context, task models.Task) (visible bool, reason string)
	Name() string
	Priority() int
}

type FilterResult struct {
	TaskID   string `json:"task_id"`
	Visible  bool   `json:"visible"`
	Reason   string `json:"reason"`
	FilterName string `json:"filter_name"`
}

type FilterEngine interface {
	AddRule(rule FilterRule)
	RemoveRule(name string)
	FilterTasks(ctx models.Context, tasks []models.Task) ([]models.Task, []FilterResult)
	GetAuditLog(taskID string, ctx models.Context) ([]FilterResult, error)
	ExplainTaskVisibility(ctx models.Context, task models.Task) TaskVisibilityExplanation
}

type FilterConfig struct {
	EnableLocationFilter   bool    `json:"enable_location_filter"`
	EnableTimeFilter      bool    `json:"enable_time_filter"`
	EnableDependencyFilter bool    `json:"enable_dependency_filter"`
	EnablePriorityFilter  bool    `json:"enable_priority_filter"`
	MaxDistanceMeters     float64 `json:"max_distance_meters"`
	MinEnergyLevel        int     `json:"min_energy_level"`
	DefaultPriorityWeight float64 `json:"default_priority_weight"`
}

type TaskVisibilityExplanation struct {
	TaskID        string              `json:"task_id"`
	TaskTitle     string              `json:"task_title"`
	IsVisible     bool                `json:"is_visible"`
	FilterResults []FilterExplanation `json:"filter_results"`
}

type FilterExplanation struct {
	FilterName string `json:"filter_name"`
	Passed     bool   `json:"passed"`
	Reason     string `json:"reason"`
	Priority   int    `json:"priority"`
}

var DefaultFilterConfig = FilterConfig{
	EnableLocationFilter:   true,
	EnableTimeFilter:      true,
	EnableDependencyFilter: true,
	EnablePriorityFilter:  true,
	MaxDistanceMeters:     5000.0,
	MinEnergyLevel:        1,
	DefaultPriorityWeight: 1.0,
}