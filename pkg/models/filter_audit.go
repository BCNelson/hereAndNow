package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type FilterAudit struct {
	ID            string          `db:"id" json:"id"`
	UserID        string          `db:"user_id" json:"user_id"`
	TaskID        string          `db:"task_id" json:"task_id"`
	ContextID     string          `db:"context_id" json:"context_id"`
	IsVisible     bool            `db:"is_visible" json:"is_visible"`
	Reasons       json.RawMessage `db:"reasons" json:"reasons"`
	PriorityScore float64         `db:"priority_score" json:"priority_score"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}

type FilterReason struct {
	Rule        string      `json:"rule"`
	Passed      bool        `json:"passed"`
	Details     string      `json:"details"`
	Score       float64     `json:"score,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
}

type FilterReasons []FilterReason

func NewFilterAudit(userID, taskID, contextID string, isVisible bool, reasons []FilterReason, priorityScore float64) (*FilterAudit, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	if contextID == "" {
		return nil, fmt.Errorf("context ID is required")
	}

	if priorityScore < 0 {
		return nil, fmt.Errorf("priority score cannot be negative")
	}

	reasonsJSON, err := json.Marshal(reasons)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reasons: %w", err)
	}

	return &FilterAudit{
		ID:            uuid.New().String(),
		UserID:        userID,
		TaskID:        taskID,
		ContextID:     contextID,
		IsVisible:     isVisible,
		Reasons:       reasonsJSON,
		PriorityScore: priorityScore,
		CreatedAt:     time.Now(),
	}, nil
}

func (fa *FilterAudit) GetReasons() ([]FilterReason, error) {
	var reasons []FilterReason
	if err := json.Unmarshal(fa.Reasons, &reasons); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reasons: %w", err)
	}
	return reasons, nil
}

func (fa *FilterAudit) SetReasons(reasons []FilterReason) error {
	reasonsJSON, err := json.Marshal(reasons)
	if err != nil {
		return fmt.Errorf("failed to marshal reasons: %w", err)
	}
	fa.Reasons = reasonsJSON
	return nil
}

func (fa *FilterAudit) AddReason(reason FilterReason) error {
	reasons, err := fa.GetReasons()
	if err != nil {
		return err
	}
	reasons = append(reasons, reason)
	return fa.SetReasons(reasons)
}

func (fa *FilterAudit) IsOwnedBy(userID string) bool {
	return fa.UserID == userID
}

func (fa *FilterAudit) BelongsToTask(taskID string) bool {
	return fa.TaskID == taskID
}

func (fa *FilterAudit) BelongsToContext(contextID string) bool {
	return fa.ContextID == contextID
}

func (fa *FilterAudit) GetFailingRules() ([]FilterReason, error) {
	reasons, err := fa.GetReasons()
	if err != nil {
		return nil, err
	}

	var failing []FilterReason
	for _, reason := range reasons {
		if !reason.Passed {
			failing = append(failing, reason)
		}
	}

	return failing, nil
}

func (fa *FilterAudit) GetPassingRules() ([]FilterReason, error) {
	reasons, err := fa.GetReasons()
	if err != nil {
		return nil, err
	}

	var passing []FilterReason
	for _, reason := range reasons {
		if reason.Passed {
			passing = append(passing, reason)
		}
	}

	return passing, nil
}

func (fa *FilterAudit) HasRule(ruleName string) bool {
	reasons, err := fa.GetReasons()
	if err != nil {
		return false
	}

	for _, reason := range reasons {
		if reason.Rule == ruleName {
			return true
		}
	}

	return false
}

func (fa *FilterAudit) GetRuleResult(ruleName string) (*FilterReason, error) {
	reasons, err := fa.GetReasons()
	if err != nil {
		return nil, err
	}

	for _, reason := range reasons {
		if reason.Rule == ruleName {
			return &reason, nil
		}
	}

	return nil, fmt.Errorf("rule '%s' not found in audit", ruleName)
}

func (fa *FilterAudit) CountFailingRules() (int, error) {
	failing, err := fa.GetFailingRules()
	if err != nil {
		return 0, err
	}
	return len(failing), nil
}

func (fa *FilterAudit) CountPassingRules() (int, error) {
	passing, err := fa.GetPassingRules()
	if err != nil {
		return 0, err
	}
	return len(passing), nil
}

func (fa *FilterAudit) GetSummary() (map[string]interface{}, error) {
	reasons, err := fa.GetReasons()
	if err != nil {
		return nil, err
	}

	failingCount := 0
	passingCount := 0
	totalScore := 0.0

	for _, reason := range reasons {
		if reason.Passed {
			passingCount++
		} else {
			failingCount++
		}
		totalScore += reason.Score
	}

	return map[string]interface{}{
		"total_rules":    len(reasons),
		"passing_rules":  passingCount,
		"failing_rules":  failingCount,
		"is_visible":     fa.IsVisible,
		"priority_score": fa.PriorityScore,
		"total_score":    totalScore,
		"created_at":     fa.CreatedAt,
	}, nil
}

func (fa *FilterAudit) Validate() error {
	if fa.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if fa.TaskID == "" {
		return fmt.Errorf("task ID is required")
	}

	if fa.ContextID == "" {
		return fmt.Errorf("context ID is required")
	}

	if fa.PriorityScore < 0 {
		return fmt.Errorf("priority score cannot be negative")
	}

	_, err := fa.GetReasons()
	if err != nil {
		return fmt.Errorf("invalid reasons JSON: %w", err)
	}

	return nil
}