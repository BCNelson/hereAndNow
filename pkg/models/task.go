package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID               string          `db:"id" json:"id"`
	Title            string          `db:"title" json:"title"`
	Description      string          `db:"description" json:"description"`
	CreatorID        string          `db:"creator_id" json:"creator_id"`
	AssigneeID       *string         `db:"assignee_id" json:"assignee_id"`
	ListID           *string         `db:"list_id" json:"list_id"`
	Status           TaskStatus      `db:"status" json:"status"`
	Priority         int             `db:"priority" json:"priority"`
	EstimatedMinutes *int            `db:"estimated_minutes" json:"estimated_minutes"`
	DueAt            *time.Time      `db:"due_at" json:"due_at"`
	CompletedAt      *time.Time      `db:"completed_at" json:"completed_at"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
	Metadata         json.RawMessage `db:"metadata" json:"metadata"`
	RecurrenceRule   *string         `db:"recurrence_rule" json:"recurrence_rule"`
	ParentTaskID     *string         `db:"parent_task_id" json:"parent_task_id"`
}

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusActive    TaskStatus = "active"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusBlocked   TaskStatus = "blocked"
)

func NewTask(title, description, creatorID string) (*Task, error) {
	if err := validateTitle(title); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Task{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		CreatorID:   creatorID,
		Status:      TaskStatusPending,
		Priority:    3,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    json.RawMessage(`{}`),
	}, nil
}

func (t *Task) SetStatus(status TaskStatus) error {
	if err := t.validateStatusTransition(status); err != nil {
		return err
	}

	t.Status = status
	t.UpdatedAt = time.Now()

	if status == TaskStatusCompleted {
		now := time.Now()
		t.CompletedAt = &now
	} else if t.CompletedAt != nil && status != TaskStatusCompleted {
		t.CompletedAt = nil
	}

	return nil
}

func (t *Task) SetPriority(priority int) error {
	if priority < 1 || priority > 5 {
		return fmt.Errorf("priority must be between 1 and 5")
	}
	t.Priority = priority
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Task) SetEstimatedMinutes(minutes int) error {
	if minutes <= 0 {
		return fmt.Errorf("estimated minutes must be positive")
	}
	t.EstimatedMinutes = &minutes
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Task) Assign(userID string) error {
	t.AssigneeID = &userID
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Task) Unassign() {
	t.AssigneeID = nil
	t.UpdatedAt = time.Now()
}

func (t *Task) SetDueDate(dueAt time.Time) {
	t.DueAt = &dueAt
	t.UpdatedAt = time.Now()
}

func (t *Task) ClearDueDate() {
	t.DueAt = nil
	t.UpdatedAt = time.Now()
}

func (t *Task) IsOverdue() bool {
	return t.DueAt != nil && t.DueAt.Before(time.Now()) && t.Status != TaskStatusCompleted
}

func (t *Task) IsCompleted() bool {
	return t.Status == TaskStatusCompleted
}

func (t *Task) IsCancelled() bool {
	return t.Status == TaskStatusCancelled
}

func (t *Task) IsActive() bool {
	return t.Status == TaskStatusActive || t.Status == TaskStatusPending
}

func (t *Task) Validate() error {
	if err := validateTitle(t.Title); err != nil {
		return err
	}

	if t.CreatorID == "" {
		return fmt.Errorf("creator ID is required")
	}

	if t.Priority < 1 || t.Priority > 5 {
		return fmt.Errorf("priority must be between 1 and 5")
	}

	if t.EstimatedMinutes != nil && *t.EstimatedMinutes <= 0 {
		return fmt.Errorf("estimated minutes must be positive")
	}

	if !isValidTaskStatus(t.Status) {
		return fmt.Errorf("invalid task status: %s", t.Status)
	}

	return nil
}

func (t *Task) validateStatusTransition(newStatus TaskStatus) error {
	if !isValidTaskStatus(newStatus) {
		return fmt.Errorf("invalid task status: %s", newStatus)
	}

	switch t.Status {
	case TaskStatusPending:
		if newStatus != TaskStatusActive && newStatus != TaskStatusCancelled && newStatus != TaskStatusBlocked {
			return fmt.Errorf("cannot transition from pending to %s", newStatus)
		}
	case TaskStatusActive:
		if newStatus != TaskStatusCompleted && newStatus != TaskStatusBlocked && newStatus != TaskStatusCancelled {
			return fmt.Errorf("cannot transition from active to %s", newStatus)
		}
	case TaskStatusBlocked:
		if newStatus != TaskStatusActive && newStatus != TaskStatusCancelled {
			return fmt.Errorf("cannot transition from blocked to %s", newStatus)
		}
	case TaskStatusCompleted, TaskStatusCancelled:
		return fmt.Errorf("cannot transition from %s status", t.Status)
	}

	return nil
}

func validateTitle(title string) error {
	if len(title) == 0 {
		return fmt.Errorf("title is required")
	}
	if len(title) > 500 {
		return fmt.Errorf("title must not exceed 500 characters")
	}
	return nil
}

func isValidTaskStatus(status TaskStatus) bool {
	switch status {
	case TaskStatusPending, TaskStatusActive, TaskStatusCompleted, TaskStatusCancelled, TaskStatusBlocked:
		return true
	default:
		return false
	}
}