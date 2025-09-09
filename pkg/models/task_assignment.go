package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TaskAssignment struct {
	ID              string           `db:"id" json:"id"`
	TaskID          string           `db:"task_id" json:"task_id"`
	AssignedBy      string           `db:"assigned_by" json:"assigned_by"`
	AssignedTo      string           `db:"assigned_to" json:"assigned_to"`
	AssignedAt      time.Time        `db:"assigned_at" json:"assigned_at"`
	Status          AssignmentStatus `db:"status" json:"status"`
	ResponseAt      *time.Time       `db:"response_at" json:"response_at"`
	ResponseMessage *string          `db:"response_message" json:"response_message"`
}

type AssignmentStatus string

const (
	AssignmentStatusPending  AssignmentStatus = "pending"
	AssignmentStatusAccepted AssignmentStatus = "accepted"
	AssignmentStatusRejected AssignmentStatus = "rejected"
)

func NewTaskAssignment(taskID, assignedBy, assignedTo string) (*TaskAssignment, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	if assignedBy == "" {
		return nil, fmt.Errorf("assigned by user ID is required")
	}

	if assignedTo == "" {
		return nil, fmt.Errorf("assigned to user ID is required")
	}

	if assignedBy == assignedTo {
		return nil, fmt.Errorf("user cannot assign task to themselves")
	}

	return &TaskAssignment{
		ID:         uuid.New().String(),
		TaskID:     taskID,
		AssignedBy: assignedBy,
		AssignedTo: assignedTo,
		AssignedAt: time.Now(),
		Status:     AssignmentStatusPending,
	}, nil
}

func (ta *TaskAssignment) Accept(message *string) error {
	if ta.Status != AssignmentStatusPending {
		return fmt.Errorf("can only accept pending assignments")
	}

	now := time.Now()
	ta.Status = AssignmentStatusAccepted
	ta.ResponseAt = &now
	ta.ResponseMessage = message

	return nil
}

func (ta *TaskAssignment) Reject(message *string) error {
	if ta.Status != AssignmentStatusPending {
		return fmt.Errorf("can only reject pending assignments")
	}

	now := time.Now()
	ta.Status = AssignmentStatusRejected
	ta.ResponseAt = &now
	ta.ResponseMessage = message

	return nil
}

func (ta *TaskAssignment) IsPending() bool {
	return ta.Status == AssignmentStatusPending
}

func (ta *TaskAssignment) IsAccepted() bool {
	return ta.Status == AssignmentStatusAccepted
}

func (ta *TaskAssignment) IsRejected() bool {
	return ta.Status == AssignmentStatusRejected
}

func (ta *TaskAssignment) HasResponse() bool {
	return ta.ResponseAt != nil
}

func (ta *TaskAssignment) IsAssignedTo(userID string) bool {
	return ta.AssignedTo == userID
}

func (ta *TaskAssignment) WasAssignedBy(userID string) bool {
	return ta.AssignedBy == userID
}

func (ta *TaskAssignment) BelongsToTask(taskID string) bool {
	return ta.TaskID == taskID
}

func (ta *TaskAssignment) InvolvesUser(userID string) bool {
	return ta.AssignedBy == userID || ta.AssignedTo == userID
}

func (ta *TaskAssignment) PendingDuration() *time.Duration {
	if ta.HasResponse() {
		duration := ta.ResponseAt.Sub(ta.AssignedAt)
		return &duration
	}
	duration := time.Since(ta.AssignedAt)
	return &duration
}

func (ta *TaskAssignment) IsOverdue(timeoutHours int) bool {
	if ta.HasResponse() || timeoutHours <= 0 {
		return false
	}
	timeout := time.Duration(timeoutHours) * time.Hour
	return time.Since(ta.AssignedAt) > timeout
}

func (ta *TaskAssignment) CanRespond(userID string) bool {
	return ta.IsPending() && ta.IsAssignedTo(userID)
}

func (ta *TaskAssignment) CanCancel(userID string) bool {
	return ta.IsPending() && ta.WasAssignedBy(userID)
}

func (ta *TaskAssignment) Validate() error {
	if ta.TaskID == "" {
		return fmt.Errorf("task ID is required")
	}

	if ta.AssignedBy == "" {
		return fmt.Errorf("assigned by user ID is required")
	}

	if ta.AssignedTo == "" {
		return fmt.Errorf("assigned to user ID is required")
	}

	if ta.AssignedBy == ta.AssignedTo {
		return fmt.Errorf("user cannot assign task to themselves")
	}

	if !isValidAssignmentStatus(ta.Status) {
		return fmt.Errorf("invalid assignment status: %s", ta.Status)
	}

	if ta.Status != AssignmentStatusPending && ta.ResponseAt == nil {
		return fmt.Errorf("response time is required for non-pending assignments")
	}

	if ta.Status == AssignmentStatusPending && ta.ResponseAt != nil {
		return fmt.Errorf("response time should not be set for pending assignments")
	}

	return nil
}

func isValidAssignmentStatus(status AssignmentStatus) bool {
	switch status {
	case AssignmentStatusPending, AssignmentStatusAccepted, AssignmentStatusRejected:
		return true
	default:
		return false
	}
}