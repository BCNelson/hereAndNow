package sync

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
)

type CalendarSyncService struct {
	calendarRepo CalendarEventRepository
	httpClient   HTTPClient
}

type CalendarEventRepository interface {
	Create(event models.CalendarEvent) error
	Update(event models.CalendarEvent) error
	Delete(eventID string) error
	GetByExternalID(externalID string) (*models.CalendarEvent, error)
	GetByUserID(userID string) ([]models.CalendarEvent, error)
	GetEventsByUserIDAndTimeRange(userID string, start, end time.Time) ([]models.CalendarEvent, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type CalendarProvider interface {
	GetEvents(userID string, start, end time.Time) ([]ExternalEvent, error)
	CreateEvent(userID string, event ExternalEvent) (*ExternalEvent, error)
	UpdateEvent(userID string, eventID string, event ExternalEvent) (*ExternalEvent, error)
	DeleteEvent(userID string, eventID string) error
	ValidateCredentials(userID string) error
}

type ExternalEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location"`
	AllDay      bool      `json:"all_day"`
	Recurring   bool      `json:"recurring"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
}

func NewCalendarSyncService(calendarRepo CalendarEventRepository, httpClient HTTPClient) *CalendarSyncService {
	return &CalendarSyncService{
		calendarRepo: calendarRepo,
		httpClient:   httpClient,
	}
}

func (s *CalendarSyncService) SyncUserCalendar(userID string, provider CalendarProvider) (*SyncResult, error) {
	result := &SyncResult{
		UserID:    userID,
		StartTime: time.Now(),
		Created:   0,
		Updated:   0,
		Deleted:   0,
		Errors:    []string{},
	}

	if err := provider.ValidateCredentials(userID); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("credential validation failed: %v", err))
		return result, err
	}

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now().AddDate(0, 3, 0)

	externalEvents, err := provider.GetEvents(userID, start, end)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch events: %v", err))
		return result, err
	}

	existingEvents, err := s.calendarRepo.GetEventsByUserIDAndTimeRange(userID, start, end)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to get existing events: %v", err))
		return result, err
	}

	externalMap := make(map[string]ExternalEvent)
	for _, event := range externalEvents {
		externalMap[event.ID] = event
	}

	existingMap := make(map[string]models.CalendarEvent)
	for _, event := range existingEvents {
		if event.ExternalID != "" {
			existingMap[event.ExternalID] = event
		}
	}

	for externalID, externalEvent := range externalMap {
		if existingEvent, exists := existingMap[externalID]; exists {
			if s.shouldUpdateEvent(existingEvent, externalEvent) {
				updatedEvent := s.convertToInternalEvent(userID, externalEvent, &existingEvent.ID)
				if err := s.calendarRepo.Update(updatedEvent); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to update event %s: %v", externalID, err))
				} else {
					result.Updated++
				}
			}
		} else {
			newEvent := s.convertToInternalEvent(userID, externalEvent, nil)
			if err := s.calendarRepo.Create(newEvent); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to create event %s: %v", externalID, err))
			} else {
				result.Created++
			}
		}
	}

	for externalID, existingEvent := range existingMap {
		if _, exists := externalMap[externalID]; !exists {
			if err := s.calendarRepo.Delete(existingEvent.ID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to delete event %s: %v", externalID, err))
			} else {
				result.Deleted++
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

func (s *CalendarSyncService) CreateEventInExternalCalendar(userID string, eventID string, provider CalendarProvider) error {
	event, err := s.calendarRepo.GetByExternalID(eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	externalEvent := s.convertToExternalEvent(*event)
	
	createdEvent, err := provider.CreateEvent(userID, externalEvent)
	if err != nil {
		return fmt.Errorf("failed to create event in external calendar: %w", err)
	}

	event.ExternalID = createdEvent.ID
	event.LastSyncedAt = time.Now()

	if err := s.calendarRepo.Update(*event); err != nil {
		return fmt.Errorf("failed to update event with external ID: %w", err)
	}

	return nil
}

func (s *CalendarSyncService) GetUpcomingEvents(userID string, hours int) ([]models.CalendarEvent, error) {
	start := time.Now()
	end := start.Add(time.Duration(hours) * time.Hour)
	
	events, err := s.calendarRepo.GetEventsByUserIDAndTimeRange(userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming events: %w", err)
	}

	return events, nil
}

func (s *CalendarSyncService) FindAvailableTimeSlots(userID string, durationMinutes int, dayRange int) ([]TimeSlot, error) {
	start := time.Now()
	end := start.AddDate(0, 0, dayRange)
	
	events, err := s.calendarRepo.GetEventsByUserIDAndTimeRange(userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	slots := []TimeSlot{}
	duration := time.Duration(durationMinutes) * time.Minute

	current := start
	for current.Before(end) {
		dayStart := time.Date(current.Year(), current.Month(), current.Day(), 9, 0, 0, 0, current.Location())
		dayEnd := time.Date(current.Year(), current.Month(), current.Day(), 17, 0, 0, 0, current.Location())

		if current.After(dayStart) {
			dayStart = current
		}

		daySlots := s.findSlotsInDay(dayStart, dayEnd, events, duration)
		slots = append(slots, daySlots...)

		current = current.AddDate(0, 0, 1)
	}

	return slots, nil
}

func (s *CalendarSyncService) findSlotsInDay(dayStart, dayEnd time.Time, allEvents []models.CalendarEvent, duration time.Duration) []TimeSlot {
	dayEvents := []models.CalendarEvent{}
	for _, event := range allEvents {
		if (event.StartAt.After(dayStart) || event.StartAt.Equal(dayStart)) && 
		   event.StartAt.Before(dayEnd.Add(24*time.Hour)) {
			dayEvents = append(dayEvents, event)
		}
	}

	if len(dayEvents) == 0 {
		if dayEnd.Sub(dayStart) >= duration {
			return []TimeSlot{{Start: dayStart, End: dayStart.Add(duration)}}
		}
		return []TimeSlot{}
	}

	slots := []TimeSlot{}
	current := dayStart

	for _, event := range dayEvents {
		if event.StartAt.Sub(current) >= duration {
			slots = append(slots, TimeSlot{
				Start: current,
				End:   current.Add(duration),
			})
		}
		if event.EndAt.After(current) {
			current = event.EndAt
		}
	}

	if dayEnd.Sub(current) >= duration {
		slots = append(slots, TimeSlot{
			Start: current,
			End:   current.Add(duration),
		})
	}

	return slots
}

func (s *CalendarSyncService) convertToInternalEvent(userID string, external ExternalEvent, existingID *string) models.CalendarEvent {
	var id string
	if existingID != nil {
		id = *existingID
	} else {
		id = uuid.New().String()
	}

	return models.CalendarEvent{
		ID:           id,
		UserID:       userID,
		Title:        external.Title,
		StartAt:      external.StartTime,
		EndAt:        external.EndTime,
		Location:     stringPtr(external.Location),
		IsAllDay:     external.AllDay,
		ExternalID:   external.ID,
		ProviderID:   external.Source,
		LastSyncedAt: time.Now(),
	}
}

func (s *CalendarSyncService) convertToExternalEvent(internal models.CalendarEvent) ExternalEvent {
	var location string
	if internal.Location != nil {
		location = *internal.Location
	}

	return ExternalEvent{
		ID:          internal.ExternalID,
		Title:       internal.Title,
		Description: "",
		StartTime:   internal.StartAt,
		EndTime:     internal.EndAt,
		Location:    location,
		AllDay:      internal.IsAllDay,
		Source:      internal.ProviderID,
		URL:         "",
	}
}

func (s *CalendarSyncService) shouldUpdateEvent(existing models.CalendarEvent, external ExternalEvent) bool {
	existingLocation := ""
	if existing.Location != nil {
		existingLocation = *existing.Location
	}
	return existing.Title != external.Title ||
		!existing.StartAt.Equal(external.StartTime) ||
		!existing.EndAt.Equal(external.EndTime) ||
		existingLocation != external.Location ||
		existing.IsAllDay != external.AllDay
}

func (s *CalendarSyncService) ValidateCalendarAccess(userID string, provider CalendarProvider) error {
	return provider.ValidateCredentials(userID)
}

type SyncResult struct {
	UserID    string        `json:"user_id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Created   int           `json:"created"`
	Updated   int           `json:"updated"`
	Deleted   int           `json:"deleted"`
	Errors    []string      `json:"errors"`
}

type TimeSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type CalDAVProvider struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient HTTPClient
}

func NewCalDAVProvider(baseURL, username, password string, httpClient HTTPClient) *CalDAVProvider {
	return &CalDAVProvider{
		BaseURL:    baseURL,
		Username:   username,
		Password:   password,
		HTTPClient: httpClient,
	}
}

func (p *CalDAVProvider) GetEvents(userID string, start, end time.Time) ([]ExternalEvent, error) {
	reqBody := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
    <D:prop>
        <D:getetag />
        <C:calendar-data />
    </D:prop>
    <C:filter>
        <C:comp-filter name="VCALENDAR">
            <C:comp-filter name="VEVENT">
                <C:time-range start="%s" end="%s"/>
            </C:comp-filter>
        </C:comp-filter>
    </C:filter>
</C:calendar-query>`, start.Format("20060102T150405Z"), end.Format("20060102T150405Z"))

	req, err := http.NewRequest("REPORT", p.BaseURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.Username, p.Password)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "1")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CalDAV request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("CalDAV server returned status %d", resp.StatusCode)
	}

	return []ExternalEvent{}, nil
}

func (p *CalDAVProvider) CreateEvent(userID string, event ExternalEvent) (*ExternalEvent, error) {
	eventID := uuid.New().String()
	icalData := p.convertToICalendar(event)

	eventURL := fmt.Sprintf("%s/%s.ics", p.BaseURL, eventID)
	req, err := http.NewRequest("PUT", eventURL, strings.NewReader(icalData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.Username, p.Password)
	req.Header.Set("Content-Type", "text/calendar")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CalDAV create request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CalDAV server returned status %d", resp.StatusCode)
	}

	event.ID = eventID
	return &event, nil
}

func (p *CalDAVProvider) UpdateEvent(userID string, eventID string, event ExternalEvent) (*ExternalEvent, error) {
	icalData := p.convertToICalendar(event)

	eventURL := fmt.Sprintf("%s/%s.ics", p.BaseURL, eventID)
	req, err := http.NewRequest("PUT", eventURL, strings.NewReader(icalData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.Username, p.Password)
	req.Header.Set("Content-Type", "text/calendar")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CalDAV update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("CalDAV server returned status %d", resp.StatusCode)
	}

	return &event, nil
}

func (p *CalDAVProvider) DeleteEvent(userID string, eventID string) error {
	eventURL := fmt.Sprintf("%s/%s.ics", p.BaseURL, eventID)
	req, err := http.NewRequest("DELETE", eventURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.Username, p.Password)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("CalDAV delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("CalDAV server returned status %d", resp.StatusCode)
	}

	return nil
}

func (p *CalDAVProvider) ValidateCredentials(userID string) error {
	req, err := http.NewRequest("OPTIONS", p.BaseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.Username, p.Password)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("CalDAV validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid credentials")
	}

	return nil
}

func (p *CalDAVProvider) convertToICalendar(event ExternalEvent) string {
	return fmt.Sprintf(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Here and Now//EN
BEGIN:VEVENT
UID:%s
DTSTART:%s
DTEND:%s
SUMMARY:%s
DESCRIPTION:%s
LOCATION:%s
END:VEVENT
END:VCALENDAR`,
		event.ID,
		event.StartTime.Format("20060102T150405Z"),
		event.EndTime.Format("20060102T150405Z"),
		event.Title,
		event.Description,
		event.Location)
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}