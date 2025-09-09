package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

type TaskList struct {
	ID          string          `db:"id" json:"id"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	OwnerID     string          `db:"owner_id" json:"owner_id"`
	IsShared    bool            `db:"is_shared" json:"is_shared"`
	Color       string          `db:"color" json:"color"`
	Icon        string          `db:"icon" json:"icon"`
	ParentID    *string         `db:"parent_id" json:"parent_id"`
	Position    int             `db:"position" json:"position"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
	Settings    json.RawMessage `db:"settings" json:"settings"`
}

var (
	hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

func NewTaskList(name, description, ownerID string) (*TaskList, error) {
	if err := validateListName(name); err != nil {
		return nil, err
	}

	now := time.Now()
	return &TaskList{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
		IsShared:    false,
		Color:       "#3B82F6",
		Icon:        "list",
		Position:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
		Settings:    json.RawMessage(`{}`),
	}, nil
}

func (tl *TaskList) SetName(name string) error {
	if err := validateListName(name); err != nil {
		return err
	}
	tl.Name = name
	tl.UpdatedAt = time.Now()
	return nil
}

func (tl *TaskList) SetDescription(description string) {
	tl.Description = description
	tl.UpdatedAt = time.Now()
}

func (tl *TaskList) SetColor(color string) error {
	if err := validateHexColor(color); err != nil {
		return err
	}
	tl.Color = color
	tl.UpdatedAt = time.Now()
	return nil
}

func (tl *TaskList) SetIcon(icon string) {
	tl.Icon = icon
	tl.UpdatedAt = time.Now()
}

func (tl *TaskList) SetPosition(position int) error {
	if position < 0 {
		return fmt.Errorf("position must be non-negative")
	}
	tl.Position = position
	tl.UpdatedAt = time.Now()
	return nil
}

func (tl *TaskList) SetParent(parentID string) error {
	if parentID == tl.ID {
		return fmt.Errorf("task list cannot be its own parent")
	}
	tl.ParentID = &parentID
	tl.UpdatedAt = time.Now()
	return nil
}

func (tl *TaskList) ClearParent() {
	tl.ParentID = nil
	tl.UpdatedAt = time.Now()
}

func (tl *TaskList) Share() {
	tl.IsShared = true
	tl.UpdatedAt = time.Now()
}

func (tl *TaskList) Unshare() {
	tl.IsShared = false
	tl.UpdatedAt = time.Now()
}

func (tl *TaskList) IsOwnedBy(userID string) bool {
	return tl.OwnerID == userID
}

func (tl *TaskList) HasParent() bool {
	return tl.ParentID != nil
}

func (tl *TaskList) Validate() error {
	if err := validateListName(tl.Name); err != nil {
		return err
	}

	if tl.OwnerID == "" {
		return fmt.Errorf("owner ID is required")
	}

	if err := validateHexColor(tl.Color); err != nil {
		return err
	}

	if tl.Position < 0 {
		return fmt.Errorf("position must be non-negative")
	}

	if tl.ParentID != nil && *tl.ParentID == tl.ID {
		return fmt.Errorf("task list cannot be its own parent")
	}

	return nil
}

func validateListName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(name) > 200 {
		return fmt.Errorf("name must not exceed 200 characters")
	}
	return nil
}

func validateHexColor(color string) error {
	if !hexColorRegex.MatchString(color) {
		return fmt.Errorf("color must be a valid hex color code (e.g., #3B82F6)")
	}
	return nil
}