package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TaskDependency struct {
	ID              string         `db:"id" json:"id"`
	TaskID          string         `db:"task_id" json:"task_id"`
	DependsOnTaskID string         `db:"depends_on_task_id" json:"depends_on_task_id"`
	DependencyType  DependencyType `db:"dependency_type" json:"dependency_type"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
}

type DependencyType string

const (
	DependencyTypeBlocking  DependencyType = "blocking"
	DependencyTypeRelated   DependencyType = "related"
	DependencyTypeScheduled DependencyType = "scheduled"
)

func NewTaskDependency(taskID, dependsOnTaskID string, dependencyType DependencyType) (*TaskDependency, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	if dependsOnTaskID == "" {
		return nil, fmt.Errorf("depends on task ID is required")
	}

	if taskID == dependsOnTaskID {
		return nil, fmt.Errorf("task cannot depend on itself")
	}

	if !isValidDependencyType(dependencyType) {
		return nil, fmt.Errorf("invalid dependency type: %s", dependencyType)
	}

	return &TaskDependency{
		ID:              uuid.New().String(),
		TaskID:          taskID,
		DependsOnTaskID: dependsOnTaskID,
		DependencyType:  dependencyType,
		CreatedAt:       time.Now(),
	}, nil
}

func (td *TaskDependency) SetType(dependencyType DependencyType) error {
	if !isValidDependencyType(dependencyType) {
		return fmt.Errorf("invalid dependency type: %s", dependencyType)
	}
	td.DependencyType = dependencyType
	return nil
}

func (td *TaskDependency) IsBlocking() bool {
	return td.DependencyType == DependencyTypeBlocking
}

func (td *TaskDependency) IsRelated() bool {
	return td.DependencyType == DependencyTypeRelated
}

func (td *TaskDependency) IsScheduled() bool {
	return td.DependencyType == DependencyTypeScheduled
}

func (td *TaskDependency) BelongsToTask(taskID string) bool {
	return td.TaskID == taskID
}

func (td *TaskDependency) DependsOnTask(taskID string) bool {
	return td.DependsOnTaskID == taskID
}

func (td *TaskDependency) InvolvesTask(taskID string) bool {
	return td.TaskID == taskID || td.DependsOnTaskID == taskID
}

func (td *TaskDependency) WouldCreateCircularDependency(otherTaskID, otherDependsOnTaskID string) bool {
	return (td.TaskID == otherDependsOnTaskID && td.DependsOnTaskID == otherTaskID)
}

func (td *TaskDependency) Validate() error {
	if td.TaskID == "" {
		return fmt.Errorf("task ID is required")
	}

	if td.DependsOnTaskID == "" {
		return fmt.Errorf("depends on task ID is required")
	}

	if td.TaskID == td.DependsOnTaskID {
		return fmt.Errorf("task cannot depend on itself")
	}

	if !isValidDependencyType(td.DependencyType) {
		return fmt.Errorf("invalid dependency type: %s", td.DependencyType)
	}

	return nil
}

func isValidDependencyType(dependencyType DependencyType) bool {
	switch dependencyType {
	case DependencyTypeBlocking, DependencyTypeRelated, DependencyTypeScheduled:
		return true
	default:
		return false
	}
}