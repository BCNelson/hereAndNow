package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/hereandnow"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/bcnelson/hereAndNow/pkg/sync"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalendarIntegration(t *testing.T) {
	// Setup test database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Initialize database schema
	err = storage.RunMigrations(db)
	require.NoError(t, err)

	// Create repositories
	userRepo := storage.NewUserRepository(db)
	taskRepo := storage.NewTaskRepository(db)
	calendarRepo := storage.NewCalendarEventRepository(db)
	locationRepo := storage.NewLocationRepository(db)
	contextRepo := storage.NewContextRepository(db)

	// Create services
	contextService := hereandnow.NewContextService(contextRepo, locationRepo)
	filterEngine := filters.NewEngine()
	taskService := hereandnow.NewTaskService(taskRepo, filterEngine, contextService)
	calendarService := sync.NewCalendarService(calendarRepo)

	t.Run("Calendar events affect task availability", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "calendar-user@example.com",
			Name:     "Calendar User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		now := time.Now()

		// Create tasks with different time requirements
		shortTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Quick phone call",
			EstimatedMinutes: 15,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, shortTask)
		require.NoError(t, err)

		mediumTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Review documents",
			EstimatedMinutes: 45,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, mediumTask)
		require.NoError(t, err)

		longTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Complete project report",
			EstimatedMinutes: 120,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, longTask)
		require.NoError(t, err)

		// Scenario 1: Free morning with meeting in afternoon
		morningMeeting := &models.CalendarEvent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Title:       "Team Standup",
			StartTime:   now.Add(3 * time.Hour),
			EndTime:     now.Add(4 * time.Hour),
			IsBlocking:  true,
			CalendarID:  "primary",
			ExternalID:  "google-standup-123",
			LastSynced:  now,
		}
		err = calendarRepo.Create(ctx, morningMeeting)
		require.NoError(t, err)

		// User has 3 hours until meeting
		morningContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 240, // Claims 4 hours but calendar says otherwise
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextWork,
			CreatedAt:        now,
		}
		err = contextService.UpdateContext(ctx, morningContext)
		require.NoError(t, err)

		// Should see all tasks that fit in 3-hour window
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Long task (2 hours) should be visible in 3-hour window
		taskTitles := extractTitles(tasks)
		assert.Contains(t, taskTitles, "Quick phone call")
		assert.Contains(t, taskTitles, "Review documents")
		assert.Contains(t, taskTitles, "Complete project report")

		// Scenario 2: Back-to-back meetings with small gaps
		meeting1 := &models.CalendarEvent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Title:       "Client Call",
			StartTime:   now.Add(30 * time.Minute),
			EndTime:     now.Add(90 * time.Minute),
			IsBlocking:  true,
			CalendarID:  "primary",
			ExternalID:  "google-client-456",
			LastSynced:  now,
		}
		err = calendarRepo.Create(ctx, meeting1)
		require.NoError(t, err)

		meeting2 := &models.CalendarEvent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Title:       "Design Review",
			StartTime:   now.Add(100 * time.Minute),
			EndTime:     now.Add(160 * time.Minute),
			IsBlocking:  true,
			CalendarID:  "primary",
			ExternalID:  "google-design-789",
			LastSynced:  now,
		}
		err = calendarRepo.Create(ctx, meeting2)
		require.NoError(t, err)

		// User has only 30 minutes before first meeting
		busyContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 180,
			EnergyLevel:      models.EnergyLevelMedium,
			SocialContext:    models.SocialContextWork,
			CreatedAt:        now,
		}
		err = contextService.UpdateContext(ctx, busyContext)
		require.NoError(t, err)

		// Should only see tasks that fit in 30-minute window
		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Only short task (15 min) fits before meeting
		shortTaskVisible := false
		for _, task := range tasks {
			if task.Title == "Quick phone call" {
				shortTaskVisible = true
			}
			// Medium and long tasks shouldn't be visible
			assert.NotEqual(t, "Review documents", task.Title)
			assert.NotEqual(t, "Complete project report", task.Title)
		}
		assert.True(t, shortTaskVisible, "Short task should be visible in 30-min window")
	})

	t.Run("Non-blocking calendar events", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "optional-meeting@example.com",
			Name:     "Optional Meeting User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		now := time.Now()

		// Create a task
		task := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Work on presentation",
			EstimatedMinutes: 60,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Create non-blocking (optional) event
		optionalMeeting := &models.CalendarEvent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Title:       "Optional: Lunch & Learn",
			StartTime:   now.Add(30 * time.Minute),
			EndTime:     now.Add(90 * time.Minute),
			IsBlocking:  false, // Non-blocking event
			CalendarID:  "primary",
			ExternalID:  "google-optional-111",
			LastSynced:  now,
		}
		err = calendarRepo.Create(ctx, optionalMeeting)
		require.NoError(t, err)

		// User context
		userContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 120,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextWork,
			CreatedAt:        now,
		}
		err = contextService.UpdateContext(ctx, userContext)
		require.NoError(t, err)

		// Task should still be visible despite optional meeting
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 1, "Task should be visible with non-blocking event")
		assert.Equal(t, "Work on presentation", tasks[0].Title)
	})

	t.Run("Calendar sync updates", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "sync-test@example.com",
			Name:     "Sync Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		now := time.Now()

		// Initial calendar state: One meeting
		initialMeeting := &models.CalendarEvent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Title:       "Morning Sync",
			StartTime:   now.Add(2 * time.Hour),
			EndTime:     now.Add(3 * time.Hour),
			IsBlocking:  true,
			CalendarID:  "primary",
			ExternalID:  "google-sync-001",
			LastSynced:  now,
		}
		err = calendarRepo.Create(ctx, initialMeeting)
		require.NoError(t, err)

		// Simulate calendar sync adding new events
		newEvents := []*models.CalendarEvent{
			{
				ID:          uuid.New(),
				UserID:      user.ID,
				Title:       "Urgent: Customer Issue",
				StartTime:   now.Add(30 * time.Minute),
				EndTime:     now.Add(60 * time.Minute),
				IsBlocking:  true,
				CalendarID:  "primary",
				ExternalID:  "google-urgent-002",
				LastSynced:  now.Add(5 * time.Minute),
			},
			{
				ID:          uuid.New(),
				UserID:      user.ID,
				Title:       "1:1 with Manager",
				StartTime:   now.Add(4 * time.Hour),
				EndTime:     now.Add(5 * time.Hour),
				IsBlocking:  true,
				CalendarID:  "primary",
				ExternalID:  "google-1on1-003",
				LastSynced:  now.Add(5 * time.Minute),
			},
		}

		for _, event := range newEvents {
			err = calendarRepo.Create(ctx, event)
			require.NoError(t, err)
		}

		// Test: Get events for time window
		startWindow := now
		endWindow := now.Add(6 * time.Hour)
		events, err := calendarRepo.GetEventsInRange(ctx, user.ID, startWindow, endWindow)
		require.NoError(t, err)

		assert.Len(t, events, 3, "Should have 3 events in 6-hour window")

		// Test: Update existing event (meeting moved earlier)
		initialMeeting.StartTime = now.Add(1 * time.Hour)
		initialMeeting.EndTime = now.Add(2 * time.Hour)
		initialMeeting.LastSynced = now.Add(10 * time.Minute)
		err = calendarRepo.Update(ctx, initialMeeting)
		require.NoError(t, err)

		// Test: Delete cancelled event
		err = calendarRepo.Delete(ctx, newEvents[1].ID)
		require.NoError(t, err)

		events, err = calendarRepo.GetEventsInRange(ctx, user.ID, startWindow, endWindow)
		require.NoError(t, err)
		assert.Len(t, events, 2, "Should have 2 events after deletion")
	})

	t.Run("Recurring events", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "recurring@example.com",
			Name:     "Recurring Event User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		now := time.Now()
		// Round to start of day for consistency
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		// Create recurring daily standup for the week
		for i := 0; i < 5; i++ { // Monday to Friday
			standup := &models.CalendarEvent{
				ID:           uuid.New(),
				UserID:       user.ID,
				Title:        "Daily Standup",
				StartTime:    today.Add(time.Duration(i*24)*time.Hour + 9*time.Hour), // 9 AM each day
				EndTime:      today.Add(time.Duration(i*24)*time.Hour + 9*time.Hour + 30*time.Minute),
				IsBlocking:   true,
				CalendarID:   "primary",
				ExternalID:   fmt.Sprintf("google-standup-day%d", i),
				RecurringID:  ptrString("standup-series"),
				LastSynced:   now,
			}
			err = calendarRepo.Create(ctx, standup)
			require.NoError(t, err)
		}

		// Test: Get all instances of recurring event
		recurringEvents, err := calendarRepo.GetRecurringEvents(ctx, user.ID, "standup-series")
		require.NoError(t, err)
		assert.Len(t, recurringEvents, 5, "Should have 5 standup instances")

		// Test: Check for conflicts with recurring events
		// Try to schedule something during tomorrow's standup
		tomorrowStandup := today.Add(24*time.Hour + 9*time.Hour)
		hasConflict, err := calendarRepo.HasConflict(ctx, user.ID, tomorrowStandup, tomorrowStandup.Add(30*time.Minute))
		require.NoError(t, err)
		assert.True(t, hasConflict, "Should detect conflict with recurring standup")

		// No conflict after standup
		afterStandup := tomorrowStandup.Add(45 * time.Minute)
		hasConflict, err = calendarRepo.HasConflict(ctx, user.ID, afterStandup, afterStandup.Add(30*time.Minute))
		require.NoError(t, err)
		assert.False(t, hasConflict, "No conflict after standup ends")
	})

	t.Run("All-day events", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "allday@example.com",
			Name:     "All Day Event User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		tomorrow := today.Add(24 * time.Hour)

		// Create all-day events
		vacation := &models.CalendarEvent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Title:       "Vacation Day",
			StartTime:   tomorrow,
			EndTime:     tomorrow.Add(24 * time.Hour),
			IsBlocking:  true,
			IsAllDay:    true,
			CalendarID:  "primary",
			ExternalID:  "google-vacation-001",
			LastSynced:  now,
		}
		err = calendarRepo.Create(ctx, vacation)
		require.NoError(t, err)

		holiday := &models.CalendarEvent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Title:       "Public Holiday",
			StartTime:   today.Add(7 * 24 * time.Hour), // Next week
			EndTime:     today.Add(8 * 24 * time.Hour),
			IsBlocking:  false, // Holidays might not block tasks
			IsAllDay:    true,
			CalendarID:  "holidays",
			ExternalID:  "google-holiday-001",
			LastSynced:  now,
		}
		err = calendarRepo.Create(ctx, holiday)
		require.NoError(t, err)

		// Test: Tasks blocked on vacation day
		vacationContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 480, // Full day
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        tomorrow.Add(10 * time.Hour), // During vacation
		}
		err = contextService.UpdateContext(ctx, vacationContext)
		require.NoError(t, err)

		// Create a work task
		workTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Finish work project",
			EstimatedMinutes: 120,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, workTask)
		require.NoError(t, err)

		// Work task should not be visible on vacation day
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 0, "No work tasks visible on vacation day")
	})
}

func ptrString(s string) *string {
	return &s
}