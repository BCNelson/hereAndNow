package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseIntegration(t *testing.T) {
	t.Run("In-memory database operations", func(t *testing.T) {
		// Test with in-memory database
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		// Initialize schema
		err = storage.RunMigrations(db)
		require.NoError(t, err)

		// Test basic operations with all repositories
		testAllRepositories(t, db)
	})

	t.Run("File database operations", func(t *testing.T) {
		// Create temporary database file
		tempDir := os.TempDir()
		dbPath := filepath.Join(tempDir, "test_hereandnow.db")
		
		// Clean up any existing file
		os.Remove(dbPath)
		defer os.Remove(dbPath)

		// Test with file database
		db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
		require.NoError(t, err)
		defer db.Close()

		// Initialize schema
		err = storage.RunMigrations(db)
		require.NoError(t, err)

		// Test operations
		testAllRepositories(t, db)

		// Test database file persistence
		db.Close()

		// Reopen database and verify data persists
		db2, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
		require.NoError(t, err)
		defer db2.Close()

		// Verify data exists after reopening
		userRepo := storage.NewUserRepository(db2)
		ctx := context.Background()
		
		users, err := userRepo.GetAll(ctx)
		require.NoError(t, err)
		assert.Greater(t, len(users), 0, "Data should persist after reopening database")
	})

	t.Run("Concurrent operations", func(t *testing.T) {
		// Test concurrent read/write operations
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		err = storage.RunMigrations(db)
		require.NoError(t, err)

		userRepo := storage.NewUserRepository(db)
		taskRepo := storage.NewTaskRepository(db)
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "concurrent@example.com",
			Name:     "Concurrent User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Run concurrent operations
		numGoroutines := 10
		numTasksPerGoroutine := 20
		errors := make(chan error, numGoroutines)
		done := make(chan bool, numGoroutines)

		// Create tasks concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				defer func() { done <- true }()
				
				for j := 0; j < numTasksPerGoroutine; j++ {
					task := &models.Task{
						ID:               uuid.New(),
						UserID:           user.ID,
						Title:            fmt.Sprintf("Task %d-%d", routineID, j),
						EstimatedMinutes: 30,
						Status:           models.TaskStatusPending,
					}
					
					if err := taskRepo.Create(ctx, task); err != nil {
						errors <- err
						return
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-errors:
				t.Fatalf("Concurrent operation failed: %v", err)
			case <-done:
				// Success
			case <-time.After(30 * time.Second):
				t.Fatalf("Concurrent operations timed out")
			}
		}

		// Verify all tasks were created
		tasks, err := taskRepo.GetByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, numGoroutines*numTasksPerGoroutine, "All tasks should be created")
	})

	t.Run("Transaction rollback", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		err = storage.RunMigrations(db)
		require.NoError(t, err)

		ctx := context.Background()

		// Test transaction rollback on error
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		userRepo := storage.NewUserRepository(db)
		
		// Create user within transaction
		user := &models.User{
			ID:       uuid.New(),
			Email:    "tx-test@example.com",
			Name:     "Transaction Test",
			Timezone: "America/New_York",
		}
		
		// Use transaction-aware repository method if available
		err = userRepo.CreateWithTx(ctx, tx, user, "password123")
		require.NoError(t, err)

		// Verify user exists within transaction
		var count int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = ?", user.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "User should exist within transaction")

		// Rollback transaction
		err = tx.Rollback()
		require.NoError(t, err)

		// Verify user doesn't exist after rollback
		users, err := userRepo.GetAll(ctx)
		require.NoError(t, err)
		assert.Len(t, users, 0, "User should not exist after rollback")
	})

	t.Run("Schema migration", func(t *testing.T) {
		// Create database without running migrations
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		ctx := context.Background()

		// Verify tables don't exist initially
		var tableCount int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableCount)
		require.NoError(t, err)
		assert.Equal(t, 0, tableCount, "No tables should exist initially")

		// Run migrations
		err = storage.RunMigrations(db)
		require.NoError(t, err)

		// Verify all tables exist after migration
		expectedTables := []string{
			"users", "tasks", "task_lists", "locations", "task_locations",
			"task_dependencies", "calendar_events", "list_members",
			"task_assignments", "filter_audit", "contexts", "analytics",
		}

		for _, tableName := range expectedTables {
			var exists bool
			err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)", tableName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Table %s should exist after migration", tableName)
		}

		// Verify schema version tracking
		var version int
		err = db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version)
		require.NoError(t, err)
		assert.Greater(t, version, 0, "Schema version should be tracked")
	})

	t.Run("Full-text search functionality", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		err = storage.RunMigrations(db)
		require.NoError(t, err)

		userRepo := storage.NewUserRepository(db)
		taskRepo := storage.NewTaskRepository(db)
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "search@example.com",
			Name:     "Search User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create tasks with searchable content
		searchableTasks := []struct {
			title       string
			description string
		}{
			{"Buy groceries", "Need to get milk, eggs, and bread from the store"},
			{"Call doctor", "Schedule annual checkup appointment"},
			{"Fix database query", "Optimize the slow query in user reports"},
			{"Plan vacation", "Research destinations and book flights to Hawaii"},
			{"Team meeting", "Discuss project milestones and deliverables"},
		}

		for _, st := range searchableTasks {
			task := &models.Task{
				ID:               uuid.New(),
				UserID:           user.ID,
				Title:            st.title,
				Description:      st.description,
				EstimatedMinutes: 30,
				Status:           models.TaskStatusPending,
			}
			err = taskRepo.Create(ctx, task)
			require.NoError(t, err)
		}

		// Test full-text search
		searchTests := []struct {
			query    string
			expected []string
		}{
			{"groceries", []string{"Buy groceries"}},
			{"doctor appointment", []string{"Call doctor"}},
			{"database query", []string{"Fix database query"}},
			{"vacation Hawaii", []string{"Plan vacation"}},
			{"meeting project", []string{"Team meeting"}},
			{"milk eggs", []string{"Buy groceries"}},
		}

		for _, test := range searchTests {
			results, err := taskRepo.SearchTasks(ctx, user.ID, test.query)
			require.NoError(t, err, "Search should not error for query: %s", test.query)
			
			assert.Greater(t, len(results), 0, "Should find results for query: %s", test.query)
			
			// Verify expected results are found
			resultTitles := make([]string, len(results))
			for i, result := range results {
				resultTitles[i] = result.Title
			}
			
			for _, expectedTitle := range test.expected {
				assert.Contains(t, resultTitles, expectedTitle, "Should find '%s' for query: %s", expectedTitle, test.query)
			}
		}
	})

	t.Run("Spatial queries for locations", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		err = storage.RunMigrations(db)
		require.NoError(t, err)

		userRepo := storage.NewUserRepository(db)
		locationRepo := storage.NewLocationRepository(db)
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "spatial@example.com",
			Name:     "Spatial User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create locations in NYC area
		locations := []*models.Location{
			{
				ID:        uuid.New(),
				UserID:    user.ID,
				Name:      "Times Square",
				Latitude:  40.7580,
				Longitude: -73.9855,
				Radius:    100,
			},
			{
				ID:        uuid.New(),
				UserID:    user.ID,
				Name:      "Central Park",
				Latitude:  40.7829,
				Longitude: -73.9654,
				Radius:    500,
			},
			{
				ID:        uuid.New(),
				UserID:    user.ID,
				Name:      "Brooklyn Bridge",
				Latitude:  40.7061,
				Longitude: -73.9969,
				Radius:    200,
			},
			{
				ID:        uuid.New(),
				UserID:    user.ID,
				Name:      "Statue of Liberty",
				Latitude:  40.6892,
				Longitude: -74.0445,
				Radius:    300,
			},
		}

		for _, loc := range locations {
			err = locationRepo.Create(ctx, loc)
			require.NoError(t, err)
		}

		// Test spatial queries
		testPoint := struct {
			lat float64
			lng float64
		}{40.7505, -73.9934} // Near Times Square

		// Find locations within 1km
		nearbyLocations, err := locationRepo.GetLocationsWithinRadius(ctx, user.ID, testPoint.lat, testPoint.lng, 1000)
		require.NoError(t, err)

		// Should find Times Square and potentially Central Park
		assert.Greater(t, len(nearbyLocations), 0, "Should find nearby locations")
		
		foundTimesSquare := false
		for _, loc := range nearbyLocations {
			if loc.Name == "Times Square" {
				foundTimesSquare = true
			}
		}
		assert.True(t, foundTimesSquare, "Should find Times Square within 1km")

		// Test with smaller radius
		closeLocations, err := locationRepo.GetLocationsWithinRadius(ctx, user.ID, testPoint.lat, testPoint.lng, 500)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(closeLocations), len(nearbyLocations), "Smaller radius should return fewer or equal locations")
	})

	t.Run("Database constraints and validation", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		require.NoError(t, err)
		defer db.Close()

		err = storage.RunMigrations(db)
		require.NoError(t, err)

		userRepo := storage.NewUserRepository(db)
		taskRepo := storage.NewTaskRepository(db)
		ctx := context.Background()

		// Test unique constraints
		user1 := &models.User{
			ID:       uuid.New(),
			Email:    "duplicate@example.com",
			Name:     "User 1",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user1, "password123")
		require.NoError(t, err)

		// Try to create user with same email
		user2 := &models.User{
			ID:       uuid.New(),
			Email:    "duplicate@example.com", // Same email
			Name:     "User 2",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user2, "password123")
		assert.Error(t, err, "Should fail to create user with duplicate email")

		// Test foreign key constraints
		nonexistentUserID := uuid.New()
		task := &models.Task{
			ID:               uuid.New(),
			UserID:           nonexistentUserID, // Non-existent user
			Title:            "Orphan task",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, task)
		assert.Error(t, err, "Should fail to create task with non-existent user")

		// Test NOT NULL constraints
		invalidTask := &models.Task{
			ID:     uuid.New(),
			UserID: user1.ID,
			// Missing required Title field
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, invalidTask)
		assert.Error(t, err, "Should fail to create task without title")
	})
}

func testAllRepositories(t *testing.T, db *sql.DB) {
	ctx := context.Background()

	// Initialize all repositories
	userRepo := storage.NewUserRepository(db)
	taskRepo := storage.NewTaskRepository(db)
	locationRepo := storage.NewLocationRepository(db)
	contextRepo := storage.NewContextRepository(db)
	calendarRepo := storage.NewCalendarEventRepository(db)

	// Create test user
	user := &models.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Name:     "Test User",
		Timezone: "America/New_York",
	}
	err := userRepo.Create(ctx, user, "password123")
	require.NoError(t, err)

	// Test user operations
	retrievedUser, err := userRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, retrievedUser.Email)

	// Create test location
	location := &models.Location{
		ID:        uuid.New(),
		UserID:    user.ID,
		Name:      "Test Location",
		Latitude:  40.7128,
		Longitude: -74.0060,
		Radius:    100,
	}
	err = locationRepo.Create(ctx, location)
	require.NoError(t, err)

	// Create test task
	task := &models.Task{
		ID:               uuid.New(),
		UserID:           user.ID,
		Title:            "Test Task",
		Description:      "Test Description",
		EstimatedMinutes: 60,
		Status:           models.TaskStatusPending,
	}
	err = taskRepo.Create(ctx, task)
	require.NoError(t, err)

	// Test task-location relationship
	err = taskRepo.AddLocation(ctx, task.ID, location.ID)
	require.NoError(t, err)

	taskLocations, err := taskRepo.GetTaskLocations(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, taskLocations, 1)
	assert.Equal(t, location.Name, taskLocations[0].Name)

	// Create test context
	testContext := &models.Context{
		ID:                uuid.New(),
		UserID:            user.ID,
		CurrentLatitude:   40.7128,
		CurrentLongitude:  -74.0060,
		CurrentLocationID: &location.ID,
		AvailableMinutes:  120,
		EnergyLevel:       models.EnergyLevelHigh,
		SocialContext:     models.SocialContextAlone,
		CreatedAt:         time.Now(),
	}
	err = contextRepo.Create(ctx, testContext)
	require.NoError(t, err)

	// Create test calendar event
	now := time.Now()
	calendarEvent := &models.CalendarEvent{
		ID:          uuid.New(),
		UserID:      user.ID,
		Title:       "Test Meeting",
		StartTime:   now.Add(1 * time.Hour),
		EndTime:     now.Add(2 * time.Hour),
		IsBlocking:  true,
		CalendarID:  "primary",
		ExternalID:  "test-event-123",
		LastSynced:  now,
	}
	err = calendarRepo.Create(ctx, calendarEvent)
	require.NoError(t, err)

	// Verify all data exists
	users, err := userRepo.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, users, 1)

	tasks, err := taskRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	locations, err := locationRepo.GetByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, locations, 1)

	contexts, err := contextRepo.GetLatestByUserID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, testContext.ID, contexts.ID)

	events, err := calendarRepo.GetEventsInRange(ctx, user.ID, now, now.Add(3*time.Hour))
	require.NoError(t, err)
	assert.Len(t, events, 1)
}