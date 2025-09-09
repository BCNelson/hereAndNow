package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TaskLocation struct {
	ID         string    `db:"id" json:"id"`
	TaskID     string    `db:"task_id" json:"task_id"`
	LocationID string    `db:"location_id" json:"location_id"`
	IsRequired bool      `db:"is_required" json:"is_required"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func NewTaskLocation(taskID, locationID string, isRequired bool) (*TaskLocation, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	if locationID == "" {
		return nil, fmt.Errorf("location ID is required")
	}

	return &TaskLocation{
		ID:         uuid.New().String(),
		TaskID:     taskID,
		LocationID: locationID,
		IsRequired: isRequired,
		CreatedAt:  time.Now(),
	}, nil
}

func (tl *TaskLocation) SetRequired(required bool) {
	tl.IsRequired = required
}

func (tl *TaskLocation) BelongsToTask(taskID string) bool {
	return tl.TaskID == taskID
}

func (tl *TaskLocation) BelongsToLocation(locationID string) bool {
	return tl.LocationID == locationID
}

func (tl *TaskLocation) Validate() error {
	if tl.TaskID == "" {
		return fmt.Errorf("task ID is required")
	}

	if tl.LocationID == "" {
		return fmt.Errorf("location ID is required")
	}

	return nil
}