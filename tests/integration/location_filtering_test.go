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

func TestLocationBasedFiltering(t *testing.T) {
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

	// Test scenario: User has tasks at different locations
	t.Run("Tasks appear/disappear based on GPS location", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "location-test@example.com",
			Name:     "Location Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create locations
		homeLocation := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "Home",
			Latitude:  40.7128,
			Longitude: -74.0060,
			Radius:    100, // 100 meters
		}
		err = locationRepo.Create(ctx, homeLocation)
		require.NoError(t, err)

		officeLocation := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "Office",
			Latitude:  40.7580,
			Longitude: -73.9855,
			Radius:    50, // 50 meters
		}
		err = locationRepo.Create(ctx, officeLocation)
		require.NoError(t, err)

		groceryStore := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "Grocery Store",
			Latitude:  40.7260,
			Longitude: -73.9897,
			Radius:    200, // 200 meters
		}
		err = locationRepo.Create(ctx, groceryStore)
		require.NoError(t, err)

		// Create tasks with location requirements
		homeTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Water the plants",
			EstimatedMinutes: 10,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, homeTask)
		require.NoError(t, err)
		err = taskRepo.AddLocation(ctx, homeTask.ID, homeLocation.ID)
		require.NoError(t, err)

		officeTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Submit TPS reports",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, officeTask)
		require.NoError(t, err)
		err = taskRepo.AddLocation(ctx, officeTask.ID, officeLocation.ID)
		require.NoError(t, err)

		groceryTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Buy milk and eggs",
			EstimatedMinutes: 20,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, groceryTask)
		require.NoError(t, err)
		err = taskRepo.AddLocation(ctx, groceryTask.ID, groceryStore.ID)
		require.NoError(t, err)

		anywhereTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Call mom",
			EstimatedMinutes: 15,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, anywhereTask)
		require.NoError(t, err)
		// No location requirement - can be done anywhere

		// Test 1: User at home
		homeContext := &models.Context{
			ID:                uuid.New(),
			UserID:            user.ID,
			CurrentLatitude:   40.7128,
			CurrentLongitude:  -74.0060,
			CurrentLocationID: &homeLocation.ID,
			AvailableMinutes:  60,
			EnergyLevel:       models.EnergyLevelHigh,
			SocialContext:     models.SocialContextAlone,
			CreatedAt:         time.Now(),
		}
		err = contextService.UpdateContext(ctx, homeContext)
		require.NoError(t, err)

		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should see home task and anywhere task
		assert.Len(t, tasks, 2, "At home, should see 2 tasks")
		taskTitles := extractTitles(tasks)
		assert.Contains(t, taskTitles, "Water the plants")
		assert.Contains(t, taskTitles, "Call mom")

		// Test 2: User at office
		officeContext := &models.Context{
			ID:                uuid.New(),
			UserID:            user.ID,
			CurrentLatitude:   40.7580,
			CurrentLongitude:  -73.9855,
			CurrentLocationID: &officeLocation.ID,
			AvailableMinutes:  120,
			EnergyLevel:       models.EnergyLevelHigh,
			SocialContext:     models.SocialContextWork,
			CreatedAt:         time.Now(),
		}
		err = contextService.UpdateContext(ctx, officeContext)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should see office task and anywhere task
		assert.Len(t, tasks, 2, "At office, should see 2 tasks")
		taskTitles = extractTitles(tasks)
		assert.Contains(t, taskTitles, "Submit TPS reports")
		assert.Contains(t, taskTitles, "Call mom")

		// Test 3: User at grocery store
		groceryContext := &models.Context{
			ID:                uuid.New(),
			UserID:            user.ID,
			CurrentLatitude:   40.7260,
			CurrentLongitude:  -73.9897,
			CurrentLocationID: &groceryStore.ID,
			AvailableMinutes:  30,
			EnergyLevel:       models.EnergyLevelMedium,
			SocialContext:     models.SocialContextAlone,
			CreatedAt:         time.Now(),
		}
		err = contextService.UpdateContext(ctx, groceryContext)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should see grocery task and anywhere task
		assert.Len(t, tasks, 2, "At grocery store, should see 2 tasks")
		taskTitles = extractTitles(tasks)
		assert.Contains(t, taskTitles, "Buy milk and eggs")
		assert.Contains(t, taskTitles, "Call mom")

		// Test 4: User at random location (not near any saved location)
		randomContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.6892, // Central Park
			CurrentLongitude: -74.0445,
			AvailableMinutes: 60,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, randomContext)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should only see anywhere task
		assert.Len(t, tasks, 1, "At random location, should only see 1 task")
		assert.Equal(t, "Call mom", tasks[0].Title)
	})

	t.Run("Tasks respect location radius", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "radius-test@example.com",
			Name:     "Radius Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create location with small radius (10 meters)
		preciseLocation := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "Precise Spot",
			Latitude:  40.7128,
			Longitude: -74.0060,
			Radius:    10, // Very small radius
		}
		err = locationRepo.Create(ctx, preciseLocation)
		require.NoError(t, err)

		// Create task at precise location
		preciseTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Find the hidden treasure",
			EstimatedMinutes: 5,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, preciseTask)
		require.NoError(t, err)
		err = taskRepo.AddLocation(ctx, preciseTask.ID, preciseLocation.ID)
		require.NoError(t, err)

		// Test: User exactly at location
		exactContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 60,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, exactContext)
		require.NoError(t, err)

		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 1, "At exact location, should see task")

		// Test: User 20 meters away (outside radius)
		// Approximately 0.0002 degrees = 20 meters at this latitude
		nearbyContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7130, // ~20 meters north
			CurrentLongitude: -74.0060,
			AvailableMinutes: 60,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, nearbyContext)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 0, "20 meters away from 10m radius, should not see task")
	})

	t.Run("Multiple valid locations for single task", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "multi-location@example.com",
			Name:     "Multi Location User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create multiple pharmacy locations
		pharmacy1 := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "CVS Downtown",
			Latitude:  40.7128,
			Longitude: -74.0060,
			Radius:    100,
		}
		err = locationRepo.Create(ctx, pharmacy1)
		require.NoError(t, err)

		pharmacy2 := &models.Location{
			ID:        uuid.New(),
			UserID:    user.ID,
			Name:      "Walgreens Midtown",
			Latitude:  40.7580,
			Longitude: -73.9855,
			Radius:    100,
		}
		err = locationRepo.Create(ctx, pharmacy2)
		require.NoError(t, err)

		// Create task that can be done at either pharmacy
		medicineTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Pick up prescription",
			EstimatedMinutes: 10,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, medicineTask)
		require.NoError(t, err)
		err = taskRepo.AddLocation(ctx, medicineTask.ID, pharmacy1.ID)
		require.NoError(t, err)
		err = taskRepo.AddLocation(ctx, medicineTask.ID, pharmacy2.ID)
		require.NoError(t, err)

		// Test: User at first pharmacy
		context1 := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 30,
			EnergyLevel:      models.EnergyLevelMedium,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, context1)
		require.NoError(t, err)

		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 1, "At pharmacy 1, should see medicine task")
		assert.Equal(t, "Pick up prescription", tasks[0].Title)

		// Test: User at second pharmacy
		context2 := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7580,
			CurrentLongitude: -73.9855,
			AvailableMinutes: 30,
			EnergyLevel:      models.EnergyLevelMedium,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, context2)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 1, "At pharmacy 2, should see medicine task")
		assert.Equal(t, "Pick up prescription", tasks[0].Title)

		// Test: User at neither pharmacy
		context3 := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.6892, // Different location
			CurrentLongitude: -74.0445,
			AvailableMinutes: 30,
			EnergyLevel:      models.EnergyLevelMedium,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, context3)
		require.NoError(t, err)

		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 0, "Away from both pharmacies, should not see medicine task")
	})
}

func extractTitles(tasks []*models.Task) []string {
	titles := make([]string, len(tasks))
	for i, task := range tasks {
		titles[i] = task.Title
	}
	return titles
}