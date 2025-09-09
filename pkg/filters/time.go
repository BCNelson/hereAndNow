package filters

import (
	"fmt"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

type TimeFilter struct {
	config         FilterConfig
	calendarRepo   CalendarEventRepository
}

type CalendarEventRepository interface {
	GetEventsByUserIDAndTimeRange(userID string, start, end time.Time) ([]models.CalendarEvent, error)
}

func NewTimeFilter(config FilterConfig, calendarRepo CalendarEventRepository) *TimeFilter {
	return &TimeFilter{
		config:       config,
		calendarRepo: calendarRepo,
	}
}

func (f *TimeFilter) Name() string {
	return "time"
}

func (f *TimeFilter) Priority() int {
	return 90
}

func (f *TimeFilter) Apply(ctx models.Context, task models.Task) (visible bool, reason string) {
	if !f.config.EnableTimeFilter {
		return true, "time filtering disabled"
	}

	if task.EstimatedMinutes == nil {
		return true, "task has no time estimate"
	}

	estimatedMinutes := *task.EstimatedMinutes
	availableMinutes := ctx.AvailableMinutes

	if estimatedMinutes <= 0 {
		return true, "task has no time requirement"
	}

	if availableMinutes <= 0 {
		return false, "no available time in current context"
	}

	if estimatedMinutes > availableMinutes {
		return false, fmt.Sprintf("task needs %d minutes but only %d available", 
			estimatedMinutes, availableMinutes)
	}

	hasConflict, conflictReason := f.checkCalendarConflicts(ctx, task)
	if hasConflict {
		return false, conflictReason
	}

	energyRequired := f.estimateEnergyRequirement(task)
	if energyRequired > ctx.EnergyLevel {
		return false, fmt.Sprintf("task requires energy level %d but current level is %d", 
			energyRequired, ctx.EnergyLevel)
	}

	return true, fmt.Sprintf("task fits in %d minute window (needs %d)", 
		availableMinutes, estimatedMinutes)
}

func (f *TimeFilter) checkCalendarConflicts(ctx models.Context, task models.Task) (bool, string) {
	if task.EstimatedMinutes == nil {
		return false, ""
	}

	now := ctx.Timestamp
	taskEndTime := now.Add(time.Duration(*task.EstimatedMinutes) * time.Minute)

	events, err := f.calendarRepo.GetEventsByUserIDAndTimeRange(
		ctx.UserID, 
		now.Add(-5*time.Minute),
		taskEndTime.Add(5*time.Minute),
	)
	if err != nil {
		return false, fmt.Sprintf("unable to check calendar: %v", err)
	}

	for _, event := range events {
		if f.isTimeOverlapping(now, taskEndTime, event.StartAt, event.EndAt) {
			return true, fmt.Sprintf("conflicts with calendar event: %s", event.Title)
		}
	}

	return false, ""
}

func (f *TimeFilter) isTimeOverlapping(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

func (f *TimeFilter) estimateEnergyRequirement(task models.Task) int {
	baseEnergy := 1

	if task.EstimatedMinutes != nil {
		minutes := *task.EstimatedMinutes
		switch {
		case minutes > 120:
			baseEnergy = 4
		case minutes > 60:
			baseEnergy = 3
		case minutes > 30:
			baseEnergy = 2
		default:
			baseEnergy = 1
		}
	}

	if task.Priority >= 8 {
		baseEnergy++
	}

	if baseEnergy > 5 {
		baseEnergy = 5
	}

	return baseEnergy
}

func (f *TimeFilter) GetNextAvailableTimeSlot(ctx models.Context, task models.Task) (*time.Time, error) {
	if task.EstimatedMinutes == nil {
		return nil, fmt.Errorf("task has no time estimate")
	}

	now := ctx.Timestamp
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	
	estimatedDuration := time.Duration(*task.EstimatedMinutes) * time.Minute

	events, err := f.calendarRepo.GetEventsByUserIDAndTimeRange(ctx.UserID, now, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("unable to check calendar: %v", err)
	}

	if len(events) == 0 {
		return &now, nil
	}

	for i := 0; i < len(events); i++ {
		var slotEnd time.Time
		if i == 0 {
			slotEnd = events[i].StartAt
		} else {
			slotEnd = events[i].StartAt
		}
		
		slotStart := now
		if i > 0 {
			slotStart = events[i-1].EndAt
		}

		if slotEnd.Sub(slotStart) >= estimatedDuration {
			return &slotStart, nil
		}
	}

	lastEventEnd := events[len(events)-1].EndAt
	if endOfDay.Sub(lastEventEnd) >= estimatedDuration {
		return &lastEventEnd, nil
	}

	return nil, fmt.Errorf("no available time slot found for task duration")
}