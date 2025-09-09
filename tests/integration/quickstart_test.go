package integration

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QuickstartTestSuite validates all scenarios from quickstart.md using the public library
type QuickstartTestSuite struct {
	filterEngine *filters.Engine
	testUserID   string
	locations    map[string]*models.Location
	tasks        []*models.Task
	context      *models.Context
}

func TestQuickstartValidation(t *testing.T) {
	suite := setupQuickstartTest(t)

	t.Run("Prerequisites", suite.testPrerequisites)
	t.Run("BasicUsage", suite.testBasicUsage)
	t.Run("LocationFiltering", suite.testLocationFiltering)
	t.Run("TimeFiltering", suite.testTimeFiltering)
	t.Run("TaskDependencies", suite.testTaskDependencies)
	t.Run("SystemVerification", suite.testSystemVerification)
}

func setupQuickstartTest(t interface{}) *QuickstartTestSuite {
	// Create mock audit repository
	auditRepo := &MockFilterAuditRepository{}
	
	// Create filter engine with quickstart-compatible configuration
	config := filters.FilterConfig{
		EnableLocationFilter:   true,
		EnableTimeFilter:      true,
		EnableDependencyFilter: true,
		EnablePriorityFilter:  true,
		MaxDistanceMeters:     200.0,
		MinEnergyLevel:        1,
		DefaultPriorityWeight: 1.0,
	}
	
	filterEngine := filters.NewEngine(config, auditRepo)
	filterEngine.AddRule(&filters.LocationFilter{})
	filterEngine.AddRule(&filters.TimeFilter{})
	filterEngine.AddRule(&filters.DependencyFilter{})
	filterEngine.AddRule(&filters.PriorityFilter{})

	// Create test user
	testUserID := "quickstart-user-123"
	
	// Create initial context (San Francisco coordinates)
	lat := 37.7749
	lng := -122.4194
	availMin := 60
	energy := 4
	social := "alone"
	
	context := &models.Context{
		ID:               "context-1",
		UserID:           testUserID,
		Timestamp:        time.Now(),
		CurrentLatitude:  &lat,
		CurrentLongitude: &lng,
		AvailableMinutes: availMin,
		EnergyLevel:      energy,
		SocialContext:    social,
	}

	// Create test locations
	locations := make(map[string]*models.Location)
	
	homeLocation, _ := models.NewLocation(testUserID, "Home", "123 Home St", 37.7749, -122.4194, 100)
	locations["home"] = homeLocation
	
	officeLocation, _ := models.NewLocation(testUserID, "Office", "456 Work Ave", 37.7858, -122.4065, 200)
	locations["office"] = officeLocation

	return &QuickstartTestSuite{
		filterEngine: filterEngine,
		testUserID:   testUserID,
		locations:    locations,
		tasks:        []*models.Task{},
		context:      context,
	}
}

// testPrerequisites validates system requirements mentioned in quickstart
func (s *QuickstartTestSuite) testPrerequisites(t *testing.T) {
	t.Log("Testing prerequisites from quickstart.md")

	// Test that core models can be created (validates Go installation and basic functionality)
	user, err := models.NewUser("testuser", "test@example.com", "Test User", "America/New_York")
	assert.NoError(t, err, "Should be able to create user model")
	assert.NotNil(t, user, "User model should be created")
	
	// Validate user model
	assert.NoError(t, user.Validate(), "User should be valid")
	
	// Test that locations can be created with coordinates
	location, err := models.NewLocation(s.testUserID, "Test Location", "123 Test St", 37.7749, -122.4194, 100)
	assert.NoError(t, err, "Should be able to create location model")
	assert.True(t, location.IsOwnedBy(s.testUserID), "Location should be owned by test user")
	
	// Test memory and performance (basic operation should be fast)
	start := time.Now()
	for i := 0; i < 100; i++ {
		task, _ := models.NewTask(fmt.Sprintf("Task %d", i), "Test task", s.testUserID)
		_ = task.Validate()
	}
	duration := time.Since(start)
	assert.Less(t, duration, time.Millisecond*100, "Creating 100 tasks should take less than 100ms")
}

// testBasicUsage tests CLI-equivalent operations using the library
func (s *QuickstartTestSuite) testBasicUsage(t *testing.T) {
	t.Log("Testing basic usage scenarios from quickstart.md")

	// Test 1: Add a task quickly (equivalent to "hereandnow task add")
	task, err := models.NewTask("Buy milk when at grocery store", "Quick grocery run", s.testUserID)
	require.NoError(t, err)
	require.NotNil(t, task)
	
	// Set task properties like CLI would
	err = task.SetPriority(3)
	require.NoError(t, err)
	
	s.tasks = append(s.tasks, task)
	assert.Equal(t, "Buy milk when at grocery store", task.Title)
	assert.Equal(t, 3, task.Priority)

	// Test 2: Create a location (equivalent to "hereandnow location add")
	groceryStore, err := models.NewLocation(
		s.testUserID,
		"Grocery Store", 
		"789 Market St",
		37.7849, // Close to home location
		-122.4094, 
		150,
	)
	require.NoError(t, err)
	s.locations["grocery"] = groceryStore
	
	// Verify location properties
	assert.Equal(t, "Grocery Store", groceryStore.Name)
	assert.Equal(t, 150, groceryStore.Radius)
	assert.True(t, groceryStore.IsOwnedBy(s.testUserID))

	// Test 3: Update context (equivalent to "hereandnow context update")
	s.context.CurrentLatitude = &groceryStore.Latitude
	s.context.CurrentLongitude = &groceryStore.Longitude
	s.context.Timestamp = time.Now()
	
	assert.Equal(t, groceryStore.Latitude, *s.context.CurrentLatitude)
	assert.Equal(t, groceryStore.Longitude, *s.context.CurrentLongitude)

	// Test 4: Complete a task (equivalent to "hereandnow task complete")
	err = task.SetStatus(models.TaskStatusCompleted)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusCompleted, task.Status)
	assert.True(t, task.IsCompleted())
}

// testLocationFiltering tests location-based filtering from quickstart manual test workflow
func (s *QuickstartTestSuite) testLocationFiltering(t *testing.T) {
	t.Log("Testing location-based filtering from quickstart.md")

	// Create a task that requires office location
	officeTask, err := models.NewTask("Review quarterly reports", "Office work task", s.testUserID)
	require.NoError(t, err)
	err = officeTask.SetPriority(2)
	require.NoError(t, err)
	err = officeTask.SetEstimatedMinutes(60)
	require.NoError(t, err)
	
	s.tasks = append(s.tasks, officeTask)

	// Test 1: Simulate being at home - task should be filtered out by location
	homeContext := &models.Context{
		ID:               "context-home",
		UserID:           s.testUserID,
		Timestamp:        time.Now(),
		CurrentLatitude:  &s.locations["home"].Latitude,
		CurrentLongitude: &s.locations["home"].Longitude,
		AvailableMinutes: 120, // Plenty of time
		EnergyLevel:      4,   // High energy
	}

	// Mock that office task requires office location
	mockLocationFilter := &MockLocationFilter{
		RequiredLocations: map[string]*models.Location{
			officeTask.ID: s.locations["office"],
		},
	}
	
	visible, reason := mockLocationFilter.Apply(*homeContext, *officeTask)
	assert.False(t, visible, "Office task should not be visible when at home")
	assert.Contains(t, strings.ToLower(reason), "location", "Reason should mention location")

	// Test 2: Simulate being at office - task should be visible
	officeContext := &models.Context{
		ID:               "context-office",
		UserID:           s.testUserID,
		Timestamp:        time.Now(),
		CurrentLatitude:  &s.locations["office"].Latitude,
		CurrentLongitude: &s.locations["office"].Longitude,
		AvailableMinutes: 120,
		EnergyLevel:      4,
	}

	visible, reason = mockLocationFilter.Apply(*officeContext, *officeTask)
	assert.True(t, visible, "Office task should be visible when at office")
	assert.Contains(t, strings.ToLower(reason), "within", "Reason should indicate proximity")
}

// testTimeFiltering tests time-based filtering scenarios
func (s *QuickstartTestSuite) testTimeFiltering(t *testing.T) {
	t.Log("Testing time-based filtering from quickstart.md")

	// Create a quick task (5 minutes)
	quickTask, err := models.NewTask("Quick email check", "5 minute task", s.testUserID)
	require.NoError(t, err)
	err = quickTask.SetEstimatedMinutes(5)
	require.NoError(t, err)
	err = quickTask.SetPriority(3)
	require.NoError(t, err)
	
	s.tasks = append(s.tasks, quickTask)

	// Test 1: 10 minutes available - should be visible
	contextWithTime := &models.Context{
		ID:               "context-time-10",
		UserID:           s.testUserID,
		Timestamp:        time.Now(),
		AvailableMinutes: 10,
		EnergyLevel:      3,
	}

	timeFilter := &filters.TimeFilter{}
	visible, reason := timeFilter.Apply(*contextWithTime, *quickTask)
	assert.True(t, visible, "Quick task should be visible with 10 minutes available")
	assert.Contains(t, strings.ToLower(reason), "available", "Reason should mention available time")

	// Test 2: Only 3 minutes available - should be hidden
	contextLimitedTime := &models.Context{
		ID:               "context-time-3",
		UserID:           s.testUserID,
		Timestamp:        time.Now(),
		AvailableMinutes: 3,
		EnergyLevel:      3,
	}

	visible, reason = timeFilter.Apply(*contextLimitedTime, *quickTask)
	assert.False(t, visible, "Quick task should be hidden with only 3 minutes available")
	assert.Contains(t, strings.ToLower(reason), "insufficient", "Reason should indicate insufficient time")
}

// testTaskDependencies tests dependency-based filtering
func (s *QuickstartTestSuite) testTaskDependencies(t *testing.T) {
	t.Log("Testing task dependencies from quickstart.md")

	// Create first task (draft)
	draftTask, err := models.NewTask("Write report draft", "Initial draft of the report", s.testUserID)
	require.NoError(t, err)
	err = draftTask.SetPriority(2)
	require.NoError(t, err)
	s.tasks = append(s.tasks, draftTask)

	// Create dependent task (review)
	reviewTask, err := models.NewTask("Review report", "Review the completed draft", s.testUserID)
	require.NoError(t, err)
	err = reviewTask.SetPriority(2)
	require.NoError(t, err)
	s.tasks = append(s.tasks, reviewTask)

	// Mock dependency filter that knows about the relationship
	mockDependencyFilter := &MockDependencyFilter{
		Dependencies: map[string][]string{
			reviewTask.ID: {draftTask.ID}, // Review depends on draft
		},
		CompletedTasks: make(map[string]bool),
	}

	basicContext := &models.Context{
		ID:               "context-deps",
		UserID:           s.testUserID,
		Timestamp:        time.Now(),
		AvailableMinutes: 120,
		EnergyLevel:      4,
	}

	// Test 1: Draft task should be visible (no dependencies)
	visible, reason := mockDependencyFilter.Apply(*basicContext, *draftTask)
	assert.True(t, visible, "Draft task should be visible (no dependencies)")
	assert.Contains(t, strings.ToLower(reason), "no dependencies", "Reason should indicate no dependencies")

	// Test 2: Review task should be hidden (dependency not completed)
	visible, reason = mockDependencyFilter.Apply(*basicContext, *reviewTask)
	assert.False(t, visible, "Review task should be hidden (dependency not completed)")
	assert.Contains(t, strings.ToLower(reason), "pending", "Reason should mention pending dependencies")

	// Test 3: Complete the draft task
	mockDependencyFilter.CompletedTasks[draftTask.ID] = true

	// Test 4: Review task should now be visible
	visible, reason = mockDependencyFilter.Apply(*basicContext, *reviewTask)
	assert.True(t, visible, "Review task should now be visible after dependency completion")
	assert.Contains(t, strings.ToLower(reason), "satisfied", "Reason should indicate dependencies satisfied")
}

// testSystemVerification simulates the "hereandnow doctor" command validation
func (s *QuickstartTestSuite) testSystemVerification(t *testing.T) {
	t.Log("Testing system verification equivalent to 'hereandnow doctor'")

	checks := map[string]func() error{
		"Filter Engine": func() error {
			if s.filterEngine == nil {
				return fmt.Errorf("filter engine not initialized")
			}
			return nil
		},
		"Location Services": func() error {
			for name, location := range s.locations {
				if err := location.Validate(); err != nil {
					return fmt.Errorf("location '%s' invalid: %w", name, err)
				}
			}
			return nil
		},
		"Task Validation": func() error {
			for i, task := range s.tasks {
				if err := task.Validate(); err != nil {
					return fmt.Errorf("task %d invalid: %w", i, err)
				}
			}
			return nil
		},
		"Context Management": func() error {
			if s.context == nil {
				return fmt.Errorf("no context available")
			}
			if s.context.UserID != s.testUserID {
				return fmt.Errorf("context user ID mismatch")
			}
			return nil
		},
		"Filter Configuration": func() error {
			config := s.filterEngine.GetConfig()
			if !config.EnableLocationFilter {
				return fmt.Errorf("location filtering disabled")
			}
			if !config.EnableTimeFilter {
				return fmt.Errorf("time filtering disabled")
			}
			return nil
		},
	}

	for checkName, checkFunc := range checks {
		t.Run(checkName, func(t *testing.T) {
			err := checkFunc()
			if err == nil {
				t.Logf("✓ %s: OK", checkName)
			} else {
				t.Errorf("✗ %s: %v", checkName, err)
			}
			assert.NoError(t, err, fmt.Sprintf("System check '%s' should pass", checkName))
		})
	}
}

// Benchmark test to ensure performance requirements from quickstart
func BenchmarkQuickstartOperations(b *testing.B) {
	suite := setupQuickstartTest(b)

	b.Run("TaskCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			task, err := models.NewTask(fmt.Sprintf("Benchmark task %d", i), "Test task", suite.testUserID)
			if err != nil {
				b.Fatal(err)
			}
			err = task.SetPriority(3)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LocationDistance", func(b *testing.B) {
		location := suite.locations["home"]
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Calculate distance to nearby point
			_ = location.DistanceFrom(37.7750, -122.4195)
		}
	})

	b.Run("FilterApplication", func(b *testing.B) {
		if len(suite.tasks) == 0 {
			task, _ := models.NewTask("Benchmark filter task", "Test task", suite.testUserID)
			suite.tasks = append(suite.tasks, task)
		}
		
		timeFilter := &filters.TimeFilter{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = timeFilter.Apply(*suite.context, *suite.tasks[0])
		}
	})
}

// Mock implementations for testing

type MockFilterAuditRepository struct{}

func (m *MockFilterAuditRepository) SaveFilterResult(audit models.FilterAudit) error {
	return nil
}

func (m *MockFilterAuditRepository) GetAuditLogByTaskID(taskID string, limit int) ([]models.FilterAudit, error) {
	return []models.FilterAudit{}, nil
}

func (m *MockFilterAuditRepository) GetAuditLogByUserID(userID string, since time.Time, limit int) ([]models.FilterAudit, error) {
	return []models.FilterAudit{}, nil
}

type MockLocationFilter struct {
	RequiredLocations map[string]*models.Location
}

func (m *MockLocationFilter) Apply(ctx models.Context, task models.Task) (bool, string) {
	requiredLocation, exists := m.RequiredLocations[task.ID]
	if !exists {
		return true, "No location requirement"
	}
	
	if ctx.CurrentLatitude == nil || ctx.CurrentLongitude == nil {
		return false, "User location unknown"
	}
	
	distance := requiredLocation.DistanceFrom(*ctx.CurrentLatitude, *ctx.CurrentLongitude)
	if distance <= float64(requiredLocation.Radius) {
		return true, fmt.Sprintf("Within %.0fm of required location '%s'", distance, requiredLocation.Name)
	}
	
	return false, fmt.Sprintf("User %.0fm away from required location '%s' (max %dm)", 
		distance, requiredLocation.Name, requiredLocation.Radius)
}

func (m *MockLocationFilter) Name() string     { return "location" }
func (m *MockLocationFilter) Priority() int   { return 90 }

type MockDependencyFilter struct {
	Dependencies   map[string][]string // taskID -> list of dependency task IDs
	CompletedTasks map[string]bool     // taskID -> completed status
}

func (m *MockDependencyFilter) Apply(ctx models.Context, task models.Task) (bool, string) {
	dependencies, exists := m.Dependencies[task.ID]
	if !exists || len(dependencies) == 0 {
		return true, "No dependencies"
	}
	
	for _, depTaskID := range dependencies {
		if !m.CompletedTasks[depTaskID] {
			return false, fmt.Sprintf("Waiting for dependency '%s' to complete", depTaskID)
		}
	}
	
	return true, fmt.Sprintf("All %d dependencies satisfied", len(dependencies))
}

func (m *MockDependencyFilter) Name() string     { return "dependency" }
func (m *MockDependencyFilter) Priority() int   { return 95 }