package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/hereandnow"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNaturalLanguageParsing(t *testing.T) {
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

	// Create services
	nlpService := hereandnow.NewNaturalLanguageService(taskRepo, locationRepo)

	t.Run("Parse location-based tasks", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "nlp-test@example.com",
			Name:     "NLP Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create known locations
		groceryStore := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "grocery store",
			Latitude:  40.7260,
			Longitude: -73.9897,
			Radius:    200,
		}
		err = locationRepo.Create(ctx, groceryStore)
		require.NoError(t, err)

		office := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "office",
			Latitude:  40.7580,
			Longitude: -73.9855,
			Radius:    50,
		}
		err = locationRepo.Create(ctx, office)
		require.NoError(t, err)

		home := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "home",
			Latitude:  40.7128,
			Longitude: -74.0060,
			Radius:    100,
		}
		err = locationRepo.Create(ctx, home)
		require.NoError(t, err)

		// Test cases for location parsing
		testCases := []struct {
			input            string
			expectedTitle    string
			expectedLocation string
			hasLocation      bool
		}{
			{
				input:            "buy milk when at grocery store",
				expectedTitle:    "buy milk",
				expectedLocation: "grocery store",
				hasLocation:      true,
			},
			{
				input:            "pick up dry cleaning on the way home",
				expectedTitle:    "pick up dry cleaning",
				expectedLocation: "home",
				hasLocation:      true,
			},
			{
				input:            "submit report when I get to the office",
				expectedTitle:    "submit report",
				expectedLocation: "office",
				hasLocation:      true,
			},
			{
				input:            "call mom",
				expectedTitle:    "call mom",
				expectedLocation: "",
				hasLocation:      false,
			},
			{
				input:            "buy groceries at the store",
				expectedTitle:    "buy groceries",
				expectedLocation: "grocery store",
				hasLocation:      true,
			},
		}

		for _, tc := range testCases {
			task, err := nlpService.ParseTaskFromText(ctx, user.ID, tc.input)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedTitle, task.Title, "Title should match for: %s", tc.input)

			if tc.hasLocation {
				locations, err := taskRepo.GetTaskLocations(ctx, task.ID)
				require.NoError(t, err)
				assert.Greater(t, len(locations), 0, "Should have location for: %s", tc.input)
				
				if len(locations) > 0 {
					assert.Contains(t, locations[0].Name, tc.expectedLocation, "Location should match for: %s", tc.input)
				}
			}
		}
	})

	t.Run("Parse time-based tasks", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "time-nlp@example.com",
			Name:     "Time NLP User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		now := time.Now()

		// Test cases for time parsing
		testCases := []struct {
			input           string
			expectedTitle   string
			hasDueDate      bool
			expectedDueTime string // relative description
		}{
			{
				input:           "finish report by tomorrow",
				expectedTitle:   "finish report",
				hasDueDate:      true,
				expectedDueTime: "tomorrow",
			},
			{
				input:           "call dentist today",
				expectedTitle:   "call dentist",
				hasDueDate:      true,
				expectedDueTime: "today",
			},
			{
				input:           "submit taxes by April 15",
				expectedTitle:   "submit taxes",
				hasDueDate:      true,
				expectedDueTime: "specific date",
			},
			{
				input:           "meeting at 3pm",
				expectedTitle:   "meeting",
				hasDueDate:      true,
				expectedDueTime: "specific time",
			},
			{
				input:           "weekly team sync every Monday",
				expectedTitle:   "weekly team sync",
				hasDueDate:      true,
				expectedDueTime: "recurring",
			},
			{
				input:           "read book",
				expectedTitle:   "read book",
				hasDueDate:      false,
				expectedDueTime: "",
			},
		}

		for _, tc := range testCases {
			task, err := nlpService.ParseTaskFromText(ctx, user.ID, tc.input)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedTitle, task.Title, "Title should match for: %s", tc.input)

			if tc.hasDueDate {
				assert.NotNil(t, task.DueDate, "Should have due date for: %s", tc.input)
				
				if task.DueDate != nil {
					switch tc.expectedDueTime {
					case "today":
						assert.Equal(t, now.Day(), task.DueDate.Day(), "Should be today")
					case "tomorrow":
						tomorrow := now.Add(24 * time.Hour)
						assert.Equal(t, tomorrow.Day(), task.DueDate.Day(), "Should be tomorrow")
					}
				}
			} else {
				assert.Nil(t, task.DueDate, "Should not have due date for: %s", tc.input)
			}
		}
	})

	t.Run("Parse priority and energy requirements", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "priority-nlp@example.com",
			Name:     "Priority NLP User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Test cases for priority parsing
		testCases := []struct {
			input            string
			expectedTitle    string
			expectedPriority models.Priority
			expectedEnergy   models.EnergyLevel
		}{
			{
				input:            "urgent: fix production bug",
				expectedTitle:    "fix production bug",
				expectedPriority: models.PriorityHigh,
				expectedEnergy:   models.EnergyLevelHigh,
			},
			{
				input:            "high priority: review contracts",
				expectedTitle:    "review contracts",
				expectedPriority: models.PriorityHigh,
				expectedEnergy:   models.EnergyLevelMedium,
			},
			{
				input:            "low energy: sort emails",
				expectedTitle:    "sort emails",
				expectedPriority: models.PriorityLow,
				expectedEnergy:   models.EnergyLevelLow,
			},
			{
				input:            "important meeting with CEO",
				expectedTitle:    "meeting with CEO",
				expectedPriority: models.PriorityHigh,
				expectedEnergy:   models.EnergyLevelHigh,
			},
			{
				input:            "quick task: update spreadsheet",
				expectedTitle:    "update spreadsheet",
				expectedPriority: models.PriorityMedium,
				expectedEnergy:   models.EnergyLevelLow,
			},
		}

		for _, tc := range testCases {
			task, err := nlpService.ParseTaskFromText(ctx, user.ID, tc.input)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedTitle, task.Title, "Title should match for: %s", tc.input)
			assert.Equal(t, tc.expectedPriority, task.Priority, "Priority should match for: %s", tc.input)
			assert.Equal(t, tc.expectedEnergy, task.RequiredEnergy, "Energy should match for: %s", tc.input)
		}
	})

	t.Run("Parse complex compound tasks", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "complex-nlp@example.com",
			Name:     "Complex NLP User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create locations
		gym := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "gym",
			Latitude:  40.7200,
			Longitude: -73.9900,
			Radius:    100,
		}
		err = locationRepo.Create(ctx, gym)
		require.NoError(t, err)

		// Complex test cases
		testCases := []struct {
			input              string
			expectedTitle      string
			expectedSubtasks   []string
			expectedLocation   string
			expectedDuration   int
			requiresFocus      bool
		}{
			{
				input:            "urgent: prepare presentation for tomorrow's meeting at the office, needs 2 hours of focus time",
				expectedTitle:    "prepare presentation",
				expectedSubtasks: []string{},
				expectedLocation: "office",
				expectedDuration: 120,
				requiresFocus:    true,
			},
			{
				input:            "workout at gym for 45 minutes then buy protein shake",
				expectedTitle:    "workout",
				expectedSubtasks: []string{"buy protein shake"},
				expectedLocation: "gym",
				expectedDuration: 45,
				requiresFocus:    false,
			},
			{
				input:            "review and sign documents (30 min) when I have quiet time",
				expectedTitle:    "review and sign documents",
				expectedSubtasks: []string{},
				expectedLocation: "",
				expectedDuration: 30,
				requiresFocus:    true,
			},
		}

		for _, tc := range testCases {
			task, err := nlpService.ParseTaskFromText(ctx, user.ID, tc.input)
			require.NoError(t, err)

			assert.Contains(t, task.Title, tc.expectedTitle, "Title should contain expected text for: %s", tc.input)
			
			if tc.expectedDuration > 0 {
				assert.Equal(t, tc.expectedDuration, task.EstimatedMinutes, "Duration should match for: %s", tc.input)
			}
			
			assert.Equal(t, tc.requiresFocus, task.RequiresFocus, "Focus requirement should match for: %s", tc.input)

			if tc.expectedLocation != "" {
				locations, err := taskRepo.GetTaskLocations(ctx, task.ID)
				require.NoError(t, err)
				assert.Greater(t, len(locations), 0, "Should have location for: %s", tc.input)
			}
		}
	})

	t.Run("Parse recurring tasks", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "recurring-nlp@example.com",
			Name:     "Recurring NLP User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Test cases for recurring tasks
		testCases := []struct {
			input              string
			expectedTitle      string
			expectedRecurrence string
		}{
			{
				input:              "water plants every Monday and Thursday",
				expectedTitle:      "water plants",
				expectedRecurrence: "weekly",
			},
			{
				input:              "daily standup at 9am",
				expectedTitle:      "standup",
				expectedRecurrence: "daily",
			},
			{
				input:              "monthly report on the first Friday",
				expectedTitle:      "report",
				expectedRecurrence: "monthly",
			},
			{
				input:              "gym every other day",
				expectedTitle:      "gym",
				expectedRecurrence: "custom",
			},
			{
				input:              "weekly review every Sunday evening",
				expectedTitle:      "review",
				expectedRecurrence: "weekly",
			},
		}

		for _, tc := range testCases {
			task, err := nlpService.ParseTaskFromText(ctx, user.ID, tc.input)
			require.NoError(t, err)

			assert.Contains(t, task.Title, tc.expectedTitle, "Title should contain expected text for: %s", tc.input)
			
			// Check if recurrence pattern was detected
			if tc.expectedRecurrence != "" {
				// Recurrence information would be stored in task metadata or a separate field
				assert.NotNil(t, task.RecurrencePattern, "Should have recurrence pattern for: %s", tc.input)
			}
		}
	})

	t.Run("Handle ambiguous input", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "ambiguous-nlp@example.com",
			Name:     "Ambiguous NLP User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Test cases with ambiguous input
		testCases := []struct {
			input         string
			shouldSucceed bool
			expectedError string
		}{
			{
				input:         "",
				shouldSucceed: false,
				expectedError: "empty input",
			},
			{
				input:         "do the thing",
				shouldSucceed: true, // Should create task with vague title
				expectedError: "",
			},
			{
				input:         "!!!###$$$",
				shouldSucceed: false,
				expectedError: "invalid input",
			},
			{
				input:         "a" + string(make([]byte, 1000)), // Very long input
				shouldSucceed: false,
				expectedError: "input too long",
			},
		}

		for _, tc := range testCases {
			task, err := nlpService.ParseTaskFromText(ctx, user.ID, tc.input)
			
			if tc.shouldSucceed {
				assert.NoError(t, err, "Should succeed for: %s", tc.input)
				assert.NotNil(t, task, "Should create task for: %s", tc.input)
			} else {
				assert.Error(t, err, "Should fail for: %s", tc.input)
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError, "Error should contain expected message")
				}
			}
		}
	})
}