package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CalendarEvent struct {
	ID           string          `db:"id" json:"id"`
	UserID       string          `db:"user_id" json:"user_id"`
	ProviderID   string          `db:"provider_id" json:"provider_id"`
	ExternalID   string          `db:"external_id" json:"external_id"`
	Title        string          `db:"title" json:"title"`
	StartAt      time.Time       `db:"start_at" json:"start_at"`
	EndAt        time.Time       `db:"end_at" json:"end_at"`
	Location     *string         `db:"location" json:"location"`
	IsAllDay     bool            `db:"is_all_day" json:"is_all_day"`
	IsBusy       bool            `db:"is_busy" json:"is_busy"`
	Metadata     json.RawMessage `db:"metadata" json:"metadata"`
	LastSyncedAt time.Time       `db:"last_synced_at" json:"last_synced_at"`
}

const (
	ProviderGoogle   = "google"
	ProviderOutlook  = "outlook"
	ProviderApple    = "apple"
	ProviderCalDAV   = "caldav"
)

func NewCalendarEvent(userID, providerID, externalID, title string, startAt, endAt time.Time) (*CalendarEvent, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	if providerID == "" {
		return nil, fmt.Errorf("provider ID is required")
	}

	if externalID == "" {
		return nil, fmt.Errorf("external ID is required")
	}

	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	if err := validateEventTimes(startAt, endAt); err != nil {
		return nil, err
	}

	now := time.Now()
	return &CalendarEvent{
		ID:           uuid.New().String(),
		UserID:       userID,
		ProviderID:   providerID,
		ExternalID:   externalID,
		Title:        title,
		StartAt:      startAt,
		EndAt:        endAt,
		IsAllDay:     false,
		IsBusy:       true,
		Metadata:     json.RawMessage(`{}`),
		LastSyncedAt: now,
	}, nil
}

func (ce *CalendarEvent) SetTitle(title string) error {
	if title == "" {
		return fmt.Errorf("title is required")
	}
	ce.Title = title
	return nil
}

func (ce *CalendarEvent) SetTimes(startAt, endAt time.Time) error {
	if err := validateEventTimes(startAt, endAt); err != nil {
		return err
	}
	ce.StartAt = startAt
	ce.EndAt = endAt
	return nil
}

func (ce *CalendarEvent) SetLocation(location string) {
	ce.Location = &location
}

func (ce *CalendarEvent) ClearLocation() {
	ce.Location = nil
}

func (ce *CalendarEvent) SetAllDay(isAllDay bool) {
	ce.IsAllDay = isAllDay
}

func (ce *CalendarEvent) SetBusy(isBusy bool) {
	ce.IsBusy = isBusy
}

func (ce *CalendarEvent) UpdateLastSyncedAt() {
	ce.LastSyncedAt = time.Now()
}

func (ce *CalendarEvent) Duration() time.Duration {
	return ce.EndAt.Sub(ce.StartAt)
}

func (ce *CalendarEvent) DurationMinutes() int {
	return int(ce.Duration().Minutes())
}

func (ce *CalendarEvent) IsOwnedBy(userID string) bool {
	return ce.UserID == userID
}

func (ce *CalendarEvent) IsFromProvider(providerID string) bool {
	return ce.ProviderID == providerID
}

func (ce *CalendarEvent) HasExternalID(externalID string) bool {
	return ce.ExternalID == externalID
}

func (ce *CalendarEvent) IsActive() bool {
	now := time.Now()
	return now.After(ce.StartAt) && now.Before(ce.EndAt)
}

func (ce *CalendarEvent) IsUpcoming() bool {
	return time.Now().Before(ce.StartAt)
}

func (ce *CalendarEvent) IsPast() bool {
	return time.Now().After(ce.EndAt)
}

func (ce *CalendarEvent) IsToday() bool {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	
	return ce.StartAt.Before(todayEnd) && ce.EndAt.After(todayStart)
}

func (ce *CalendarEvent) OverlapsWith(other *CalendarEvent) bool {
	return ce.StartAt.Before(other.EndAt) && ce.EndAt.After(other.StartAt)
}

func (ce *CalendarEvent) ConflictsWith(other *CalendarEvent) bool {
	return ce.OverlapsWith(other) && (ce.IsBusy || other.IsBusy)
}

func (ce *CalendarEvent) TimeUntilStart() time.Duration {
	return time.Until(ce.StartAt)
}

func (ce *CalendarEvent) TimeUntilEnd() time.Duration {
	return time.Until(ce.EndAt)
}

func (ce *CalendarEvent) Validate() error {
	if ce.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if ce.ProviderID == "" {
		return fmt.Errorf("provider ID is required")
	}

	if ce.ExternalID == "" {
		return fmt.Errorf("external ID is required")
	}

	if ce.Title == "" {
		return fmt.Errorf("title is required")
	}

	if err := validateEventTimes(ce.StartAt, ce.EndAt); err != nil {
		return err
	}

	return nil
}

func validateEventTimes(startAt, endAt time.Time) error {
	if startAt.After(endAt) || startAt.Equal(endAt) {
		return fmt.Errorf("start time must be before end time")
	}

	maxDuration := 7 * 24 * time.Hour // 7 days
	if endAt.Sub(startAt) > maxDuration {
		return fmt.Errorf("event duration cannot exceed 7 days")
	}

	return nil
}