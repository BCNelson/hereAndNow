package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/hereandnow"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeBasedFiltering(t *testing.T) {
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
	locationRepo := storage.NewLocationRepository(db)
	contextRepo := storage.NewContextRepository(db)

	// Create services
	contextService := hereandnow.NewContextService(contextRepo, locationRepo)
	filterEngine := filters.NewEngine()
	taskService := hereandnow.NewTaskService(taskRepo, filterEngine, contextService)

	t.Run("Tasks filtered by available time windows", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "time-test@example.com",
			Name:     "Time Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create tasks with different time requirements
		quickTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Send quick email",
			EstimatedMinutes: 5,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, quickTask)
		require.NoError(t, err)

		mediumTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Review document",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, mediumTask)
		require.NoError(t, err)

		longTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Write project proposal",
			EstimatedMinutes: 120,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, longTask)
		require.NoError(t, err)

		focusTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Deep work: Algorithm design",
			EstimatedMinutes: 90,
			RequiresFocus:    true,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, focusTask)
		require.NoError(t, err)

		// Test 1: Only 10 minutes available
		shortContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 10,
			EnergyLevel:      models.EnergyLevelMedium,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, shortContext)
		require.NoError(t, err)

		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should only see quick task (5 minutes)
		assert.Len(t, tasks, 1, "With 10 minutes, should only see quick task")
		assert.Equal(t, "Send quick email", tasks[0].Title)

		// Test 2: 45 minutes available
		mediumContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 45,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, mediumContext)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should see quick and medium tasks
		assert.Len(t, tasks, 2, "With 45 minutes, should see 2 tasks")
		taskTitles := extractTitles(tasks)
		assert.Contains(t, taskTitles, "Send quick email")
		assert.Contains(t, taskTitles, "Review document")

		// Test 3: 3 hours available
		longContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 180,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, longContext)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should see all tasks
		assert.Len(t, tasks, 4, "With 180 minutes, should see all 4 tasks")
		taskTitles = extractTitles(tasks)
		assert.Contains(t, taskTitles, "Send quick email")
		assert.Contains(t, taskTitles, "Review document")
		assert.Contains(t, taskTitles, "Write project proposal")
		assert.Contains(t, taskTitles, "Deep work: Algorithm design")
	})

	t.Run("Calendar events affect task availability", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "calendar-test@example.com",
			Name:     "Calendar Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create a task that requires 60 minutes
		task := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Prepare presentation",
			EstimatedMinutes: 60,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Create calendar event repository
		calendarRepo := storage.NewCalendarEventRepository(db)

		// Add calendar event in 30 minutes
		now := time.Now()
		meetingStart := now.Add(30 * time.Minute)
		meetingEnd := meetingStart.Add(1 * time.Hour)

		calendarEvent := &models.CalendarEvent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Title:       "Team Meeting",
			StartTime:   meetingStart,
			EndTime:     meetingEnd,
			IsBlocking:  true,
			CalendarID:  "primary",
			ExternalID:  "google-event-123",
			LastSynced:  now,
		}
		err = calendarRepo.Create(ctx, calendarEvent)
		require.NoError(t, err)

		// Test: User has meeting in 30 minutes
		contextWithMeeting := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 120, // Would normally have 2 hours
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextWork,
			CreatedAt:        now,
		}
		err = contextService.UpdateContext(ctx, contextWithMeeting)
		require.NoError(t, err)

		// Calculate actual available time considering calendar
		actualAvailable := contextService.GetAvailableTimeUntilNextEvent(ctx, user.ID, now)
		
		// Should only have 30 minutes until meeting
		assert.LessOrEqual(t, actualAvailable, 30)

		// Task requiring 60 minutes should not be visible
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 0, "Task requiring 60 minutes not visible with meeting in 30 minutes")

		// Test: After the meeting
		afterMeetingContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 120,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextWork,
			CreatedAt:        meetingEnd.Add(5 * time.Minute),
		}
		err = contextService.UpdateContext(ctx, afterMeetingContext)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 1, "After meeting, task should be visible")
		assert.Equal(t, "Prepare presentation", tasks[0].Title)
	})

	t.Run("Time windows with deadlines", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "deadline-test@example.com",
			Name:     "Deadline Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		now := time.Now()

		// Create tasks with different deadlines
		urgentTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Urgent: Submit report",
			EstimatedMinutes: 45,
			DueDate:          ptrTime(now.Add(2 * time.Hour)),
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, urgentTask)
		require.NoError(t, err)

		tomorrowTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Tomorrow: Review contract",
			EstimatedMinutes: 60,
			DueDate:          ptrTime(now.Add(24 * time.Hour)),
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, tomorrowTask)
		require.NoError(t, err)

		nextWeekTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Next week: Plan workshop",
			EstimatedMinutes: 90,
			DueDate:          ptrTime(now.Add(7 * 24 * time.Hour)),
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, nextWeekTask)
		require.NoError(t, err)

		// Test: Limited time, prioritize by deadline
		limitedContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 50, // Only enough for urgent task
			EnergyLevel:      models.EnergyLevelMedium,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        now,
		}
		err = contextService.UpdateContext(ctx, limitedContext)
		require.NoError(t, err)

		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should prioritize urgent task
		assert.Greater(t, len(tasks), 0, "Should have at least one task")
		if len(tasks) > 0 {
			assert.Equal(t, "Urgent: Submit report", tasks[0].Title, "Urgent task should be first")
		}
	})

	t.Run("Focus time requirements", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "focus-test@example.com",
			Name:     "Focus Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create tasks with focus requirements
		focusTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Write code: Complex algorithm",
			EstimatedMinutes: 60,
			RequiresFocus:    true,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, focusTask)
		require.NoError(t, err)

		noFocusTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Respond to emails",
			EstimatedMinutes: 30,
			RequiresFocus:    false,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, noFocusTask)
		require.NoError(t, err)

		// Test 1: In distracting environment (with family)
		distractedContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 90,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextFamily, // Distracting
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, distractedContext)
		require.NoError(t, err)

		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should only see non-focus task
		assert.Len(t, tasks, 1, "In distracting environment, should only see non-focus task")
		assert.Equal(t, "Respond to emails", tasks[0].Title)

		// Test 2: In quiet environment (alone)
		quietContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 90,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone, // Good for focus
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, quietContext)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should see both tasks
		assert.Len(t, tasks, 2, "In quiet environment, should see both tasks")
		taskTitles := extractTitles(tasks)
		assert.Contains(t, taskTitles, "Write code: Complex algorithm")
		assert.Contains(t, taskTitles, "Respond to emails")
	})
}

func ptrTime(t time.Time) *time.Time {
	return &t
}