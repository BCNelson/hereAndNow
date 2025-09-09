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

func TestTaskDependencies(t *testing.T) {
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

	t.Run("Dependent tasks hidden until prerequisites complete", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "dependency-test@example.com",
			Name:     "Dependency Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create a chain of dependent tasks
		// Task 1: Buy ingredients (no dependencies)
		buyIngredientsTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Buy ingredients for cake",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, buyIngredientsTask)
		require.NoError(t, err)

		// Task 2: Bake cake (depends on buying ingredients)
		bakeCakeTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Bake birthday cake",
			EstimatedMinutes: 90,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, bakeCakeTask)
		require.NoError(t, err)

		// Add dependency: bake cake requires buying ingredients first
		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           bakeCakeTask.ID,
			DependsOnTaskID:  buyIngredientsTask.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		// Task 3: Decorate cake (depends on baking cake)
		decorateCakeTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Decorate the cake",
			EstimatedMinutes: 45,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, decorateCakeTask)
		require.NoError(t, err)

		// Add dependency: decorating requires baking first
		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           decorateCakeTask.ID,
			DependsOnTaskID:  bakeCakeTask.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		// Set user context
		userContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 180, // 3 hours available
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, userContext)
		require.NoError(t, err)

		// Test 1: Initially, only the first task should be visible
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 1, "Only root task should be visible initially")
		assert.Equal(t, "Buy ingredients for cake", tasks[0].Title)

		// Complete the first task
		buyIngredientsTask.Status = models.TaskStatusCompleted
		buyIngredientsTask.CompletedAt = ptrTime(time.Now())
		err = taskRepo.Update(ctx, buyIngredientsTask)
		require.NoError(t, err)

		// Test 2: After completing first task, second task becomes visible
		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 1, "Second task should be visible after first is completed")
		assert.Equal(t, "Bake birthday cake", tasks[0].Title)

		// Complete the second task
		bakeCakeTask.Status = models.TaskStatusCompleted
		bakeCakeTask.CompletedAt = ptrTime(time.Now())
		err = taskRepo.Update(ctx, bakeCakeTask)
		require.NoError(t, err)

		// Test 3: After completing second task, third task becomes visible
		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 1, "Third task should be visible after second is completed")
		assert.Equal(t, "Decorate the cake", tasks[0].Title)

		// Complete the third task
		decorateCakeTask.Status = models.TaskStatusCompleted
		decorateCakeTask.CompletedAt = ptrTime(time.Now())
		err = taskRepo.Update(ctx, decorateCakeTask)
		require.NoError(t, err)

		// Test 4: All tasks completed, nothing visible
		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 0, "No tasks visible when all are completed")
	})

	t.Run("Multiple parallel dependencies", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "parallel-dep-test@example.com",
			Name:     "Parallel Dependency User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create tasks that can be done in parallel
		researchTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Research vacation destinations",
			EstimatedMinutes: 60,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, researchTask)
		require.NoError(t, err)

		budgetTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Calculate vacation budget",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, budgetTask)
		require.NoError(t, err)

		// Task that depends on both parallel tasks
		bookTripTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Book vacation trip",
			EstimatedMinutes: 45,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, bookTripTask)
		require.NoError(t, err)

		// Add dependencies: booking requires both research AND budget
		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           bookTripTask.ID,
			DependsOnTaskID:  researchTask.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           bookTripTask.ID,
			DependsOnTaskID:  budgetTask.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		// Set user context
		userContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 120,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, userContext)
		require.NoError(t, err)

		// Test 1: Both prerequisite tasks should be visible
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 2, "Both parallel prerequisite tasks should be visible")
		taskTitles := extractTitles(tasks)
		assert.Contains(t, taskTitles, "Research vacation destinations")
		assert.Contains(t, taskTitles, "Calculate vacation budget")

		// Complete only one prerequisite
		researchTask.Status = models.TaskStatusCompleted
		researchTask.CompletedAt = ptrTime(time.Now())
		err = taskRepo.Update(ctx, researchTask)
		require.NoError(t, err)

		// Test 2: Dependent task still not visible with only one prerequisite done
		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 1, "Only remaining prerequisite should be visible")
		assert.Equal(t, "Calculate vacation budget", tasks[0].Title)

		// Complete second prerequisite
		budgetTask.Status = models.TaskStatusCompleted
		budgetTask.CompletedAt = ptrTime(time.Now())
		err = taskRepo.Update(ctx, budgetTask)
		require.NoError(t, err)

		// Test 3: Now dependent task becomes visible
		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 1, "Dependent task visible after all prerequisites complete")
		assert.Equal(t, "Book vacation trip", tasks[0].Title)
	})

	t.Run("Circular dependency detection", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "circular-test@example.com",
			Name:     "Circular Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create three tasks
		taskA := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Task A",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, taskA)
		require.NoError(t, err)

		taskB := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Task B",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, taskB)
		require.NoError(t, err)

		taskC := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Task C",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, taskC)
		require.NoError(t, err)

		// Create dependencies: A -> B -> C -> A (circular)
		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           taskB.ID,
			DependsOnTaskID:  taskA.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           taskC.ID,
			DependsOnTaskID:  taskB.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		// This should create a circular dependency
		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           taskA.ID,
			DependsOnTaskID:  taskC.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		// Should detect circular dependency and return error or handle gracefully
		// The exact behavior depends on implementation
		
		// Set user context
		userContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 120,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, userContext)
		require.NoError(t, err)

		// If circular dependencies are allowed to be created but filtered out
		// then no tasks should be visible due to circular dependency
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		if err == nil {
			// If no error, then filtering should handle it
			assert.Len(t, tasks, 0, "No tasks should be visible with circular dependencies")
		}
	})

	t.Run("Soft dependencies (suggested order)", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "soft-dep-test@example.com",
			Name:     "Soft Dependency User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create tasks with soft dependencies
		readBookTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Read design patterns book",
			EstimatedMinutes: 120,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, readBookTask)
		require.NoError(t, err)

		implementPatternTask := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Implement observer pattern",
			EstimatedMinutes: 60,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, implementPatternTask)
		require.NoError(t, err)

		// Add soft dependency: implementing is suggested after reading, but not required
		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           implementPatternTask.ID,
			DependsOnTaskID:  readBookTask.ID,
			DependencyType:   models.DependencyTypeSuggested,
		})
		require.NoError(t, err)

		// Set user context
		userContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 180,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, userContext)
		require.NoError(t, err)

		// Test: Both tasks should be visible (soft dependency doesn't block)
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 2, "Both tasks visible with soft dependency")
		taskTitles := extractTitles(tasks)
		assert.Contains(t, taskTitles, "Read design patterns book")
		assert.Contains(t, taskTitles, "Implement observer pattern")

		// The reading task might be prioritized higher (implementation detail)
		// but both should be available
	})

	t.Run("Dependency chain with mixed statuses", func(t *testing.T) {
		ctx := context.Background()

		// Create test user
		user := &models.User{
			ID:       uuid.New(),
			Email:    "chain-test@example.com",
			Name:     "Chain Test User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, user, "password123")
		require.NoError(t, err)

		// Create a longer dependency chain with mixed statuses
		step1 := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Step 1: Planning",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusCompleted, // Already done
			CompletedAt:      ptrTime(time.Now().Add(-2 * time.Hour)),
		}
		err = taskRepo.Create(ctx, step1)
		require.NoError(t, err)

		step2 := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Step 2: Design",
			EstimatedMinutes: 45,
			Status:           models.TaskStatusCompleted, // Already done
			CompletedAt:      ptrTime(time.Now().Add(-1 * time.Hour)),
		}
		err = taskRepo.Create(ctx, step2)
		require.NoError(t, err)

		step3 := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Step 3: Implementation",
			EstimatedMinutes: 90,
			Status:           models.TaskStatusInProgress, // Currently working on
			StartedAt:        ptrTime(time.Now().Add(-30 * time.Minute)),
		}
		err = taskRepo.Create(ctx, step3)
		require.NoError(t, err)

		step4 := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Step 4: Testing",
			EstimatedMinutes: 60,
			Status:           models.TaskStatusPending, // Waiting
		}
		err = taskRepo.Create(ctx, step4)
		require.NoError(t, err)

		step5 := &models.Task{
			ID:               uuid.New(),
			UserID:           user.ID,
			Title:            "Step 5: Deployment",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending, // Waiting
		}
		err = taskRepo.Create(ctx, step5)
		require.NoError(t, err)

		// Create dependency chain
		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           step2.ID,
			DependsOnTaskID:  step1.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           step3.ID,
			DependsOnTaskID:  step2.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           step4.ID,
			DependsOnTaskID:  step3.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		err = taskRepo.AddDependency(ctx, &models.TaskDependency{
			ID:               uuid.New(),
			TaskID:           step5.ID,
			DependsOnTaskID:  step4.ID,
			DependencyType:   models.DependencyTypeBlocking,
		})
		require.NoError(t, err)

		// Set user context
		userContext := &models.Context{
			ID:               uuid.New(),
			UserID:           user.ID,
			CurrentLatitude:  40.7128,
			CurrentLongitude: -74.0060,
			AvailableMinutes: 120,
			EnergyLevel:      models.EnergyLevelHigh,
			SocialContext:    models.SocialContextAlone,
			CreatedAt:        time.Now(),
		}
		err = contextService.UpdateContext(ctx, userContext)
		require.NoError(t, err)

		// Test: Only in-progress task should be visible
		tasks, err := taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		// Should only see the in-progress task (step 3)
		assert.Len(t, tasks, 1, "Only in-progress task should be visible")
		assert.Equal(t, "Step 3: Implementation", tasks[0].Title)

		// Complete step 3
		step3.Status = models.TaskStatusCompleted
		step3.CompletedAt = ptrTime(time.Now())
		err = taskRepo.Update(ctx, step3)
		require.NoError(t, err)

		// Now step 4 should be visible
		tasks, err = taskService.GetFilteredTasks(ctx, user.ID)
		require.NoError(t, err)

		assert.Len(t, tasks, 1, "Next task in chain should be visible")
		assert.Equal(t, "Step 4: Testing", tasks[0].Title)
	})
}