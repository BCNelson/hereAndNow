package performance

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockAuditRepo implements FilterAuditRepository for testing
type MockAuditRepo struct{}

func (m *MockAuditRepo) SaveFilterResult(audit models.FilterAudit) error {
	return nil // No-op for benchmarking
}

func (m *MockAuditRepo) GetAuditLogByTaskID(taskID string, limit int) ([]models.FilterAudit, error) {
	return []models.FilterAudit{}, nil
}

func (m *MockAuditRepo) GetAuditLogByUserID(userID string, since time.Time, limit int) ([]models.FilterAudit, error) {
	return []models.FilterAudit{}, nil
}

// MockFilter implements FilterRule for benchmarking
type MockFilter struct {
	name       string
	priority   int
	passRate   float64 // Percentage of tasks that should pass (0.0-1.0)
	complexity int     // Simulated complexity (processing time)
}

func (f *MockFilter) Name() string {
	return f.name
}

func (f *MockFilter) Priority() int {
	return f.priority
}

func (f *MockFilter) Apply(ctx models.Context, task models.Task) (bool, string) {
	// Simulate processing complexity
	for i := 0; i < f.complexity; i++ {
		_ = fmt.Sprintf("processing_%d", i)
	}
	
	shouldPass := rand.Float64() < f.passRate
	reason := fmt.Sprintf("%s filter applied", f.name)
	if !shouldPass {
		reason = fmt.Sprintf("%s filter blocked task", f.name)
	}
	
	return shouldPass, reason
}

func generateTestTasks(count int) []models.Task {
	tasks := make([]models.Task, count)
	for i := 0; i < count; i++ {
		estimatedMinutes := rand.Intn(180) + 5 // 5-180 minutes
		priority := rand.Intn(5) + 1           // 1-5
		
		metadata := map[string]interface{}{
			"tags":        []string{fmt.Sprintf("tag_%d", i%10)},
			"category":    fmt.Sprintf("category_%d", i%5),
			"complexity":  rand.Intn(5) + 1,
		}
		metadataJSON, _ := json.Marshal(metadata)
		
		tasks[i] = models.Task{
			ID:               uuid.New().String(),
			Title:            fmt.Sprintf("Task %d", i),
			Description:      fmt.Sprintf("Test task description %d", i),
			CreatorID:        "test-user",
			Status:           models.TaskStatusPending,
			Priority:         priority,
			EstimatedMinutes: &estimatedMinutes,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Metadata:         metadataJSON,
		}
	}
	return tasks
}

func generateTestContext() models.Context {
	lat := 37.7749 + (rand.Float64()-0.5)*0.1  // San Francisco area
	lng := -122.4194 + (rand.Float64()-0.5)*0.1
	
	return models.Context{
		ID:                uuid.New().String(),
		UserID:            "test-user",
		Timestamp:         time.Now(),
		CurrentLatitude:   &lat,
		CurrentLongitude:  &lng,
		AvailableMinutes:  rand.Intn(240) + 15, // 15-240 minutes
		SocialContext:     models.SocialContextAlone,
		EnergyLevel:       rand.Intn(5) + 1, // 1-5
		WeatherCondition:  nil,
		TrafficLevel:      nil,
		Metadata:          json.RawMessage(`{}`),
	}
}

func setupFilterEngine() *filters.Engine {
	config := filters.DefaultFilterConfig
	auditRepo := &MockAuditRepo{}
	engine := filters.NewEngine(config, auditRepo)
	
	// Add realistic filter rules with varying complexity
	engine.AddRule(&MockFilter{
		name:       "location",
		priority:   100,
		passRate:   0.6, // 60% of tasks pass location filter
		complexity: 50,  // Medium complexity
	})
	
	engine.AddRule(&MockFilter{
		name:       "time",
		priority:   90,
		passRate:   0.75, // 75% of tasks pass time filter
		complexity: 20,   // Low complexity
	})
	
	engine.AddRule(&MockFilter{
		name:       "dependency",
		priority:   80,
		passRate:   0.85, // 85% of tasks pass dependency filter
		complexity: 100,  // High complexity (graph traversal)
	})
	
	engine.AddRule(&MockFilter{
		name:       "priority",
		priority:   70,
		passRate:   0.9, // 90% of tasks pass priority filter
		complexity: 10,  // Very low complexity
	})
	
	return engine
}

// BenchmarkFilterEngine_SmallDataset tests performance with realistic small dataset
func BenchmarkFilterEngine_SmallDataset(b *testing.B) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(10)
	ctx := generateTestContext()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.FilterTasks(ctx, tasks)
	}
}

// BenchmarkFilterEngine_MediumDataset tests performance with realistic medium dataset
func BenchmarkFilterEngine_MediumDataset(b *testing.B) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(100)
	ctx := generateTestContext()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.FilterTasks(ctx, tasks)
	}
}

// BenchmarkFilterEngine_LargeDataset tests performance with realistic large dataset
func BenchmarkFilterEngine_LargeDataset(b *testing.B) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(1000)
	ctx := generateTestContext()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.FilterTasks(ctx, tasks)
	}
}

// BenchmarkFilterEngine_SingleTaskFiltering tests individual task filtering performance
func BenchmarkFilterEngine_SingleTaskFiltering(b *testing.B) {
	engine := setupFilterEngine()
	task := generateTestTasks(1)[0]
	ctx := generateTestContext()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.FilterTasks(ctx, []models.Task{task})
	}
}

// TestFilterPerformanceRequirement verifies sub-100ms filtering requirement
func TestFilterPerformanceRequirement(t *testing.T) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(500) // Realistic user task count
	ctx := generateTestContext()
	
	// Warm up
	for i := 0; i < 10; i++ {
		_, _ = engine.FilterTasks(ctx, tasks)
	}
	
	// Measure performance over multiple runs
	const numRuns = 50
	var totalDuration time.Duration
	
	for i := 0; i < numRuns; i++ {
		start := time.Now()
		visibleTasks, results := engine.FilterTasks(ctx, tasks)
		duration := time.Since(start)
		totalDuration += duration
		
		// Validate basic functionality
		assert.LessOrEqual(t, len(visibleTasks), len(tasks))
		assert.Equal(t, len(tasks)*4, len(results)) // 4 filters applied to each task
	}
	
	avgDuration := totalDuration / numRuns
	
	t.Logf("Average filtering time for %d tasks: %v", len(tasks), avgDuration)
	t.Logf("Performance requirement: sub-100ms")
	
	// Verify sub-100ms requirement
	assert.Less(t, avgDuration.Milliseconds(), int64(100),
		"Filtering should complete in under 100ms, but took %v", avgDuration)
}

// TestFilterEngineMemoryUsage tests memory efficiency
func TestFilterEngineMemoryUsage(t *testing.T) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(1000)
	ctx := generateTestContext()
	
	// Force garbage collection before measuring
	runtime.GC()
	
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	// Perform filtering operations
	for i := 0; i < 100; i++ {
		_, _ = engine.FilterTasks(ctx, tasks)
	}
	
	runtime.ReadMemStats(&m2)
	
	memoryUsedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	t.Logf("Memory used for filtering operations: %.2f MB", memoryUsedMB)
	
	// Memory usage should be reasonable for filtering operations
	assert.Less(t, memoryUsedMB, 10.0, "Filtering should not use excessive memory")
}

// BenchmarkFilterEngine_WorstCase tests performance under worst-case conditions
func BenchmarkFilterEngine_WorstCase(b *testing.B) {
	config := filters.DefaultFilterConfig
	auditRepo := &MockAuditRepo{}
	engine := filters.NewEngine(config, auditRepo)
	
	// Add high-complexity filters that mostly fail
	engine.AddRule(&MockFilter{
		name:       "complex_location",
		priority:   100,
		passRate:   0.1, // Only 10% pass
		complexity: 200, // High complexity
	})
	
	engine.AddRule(&MockFilter{
		name:       "complex_dependency",
		priority:   90,
		passRate:   0.1, // Only 10% pass
		complexity: 300, // Very high complexity
	})
	
	tasks := generateTestTasks(100)
	ctx := generateTestContext()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.FilterTasks(ctx, tasks)
	}
}

// BenchmarkFilterEngine_BestCase tests performance under best-case conditions
func BenchmarkFilterEngine_BestCase(b *testing.B) {
	config := filters.DefaultFilterConfig
	auditRepo := &MockAuditRepo{}
	engine := filters.NewEngine(config, auditRepo)
	
	// Add simple filters that mostly pass
	engine.AddRule(&MockFilter{
		name:       "simple_filter",
		priority:   100,
		passRate:   0.95, // 95% pass
		complexity: 1,    // Minimal complexity
	})
	
	tasks := generateTestTasks(1000)
	ctx := generateTestContext()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.FilterTasks(ctx, tasks)
	}
}

// BenchmarkFilterEngineExplainVisibility tests explanation performance
func BenchmarkFilterEngineExplainVisibility(b *testing.B) {
	engine := setupFilterEngine()
	task := generateTestTasks(1)[0]
	ctx := generateTestContext()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.ExplainTaskVisibility(ctx, task)
	}
}

// TestFilterConsistency verifies filtering results are consistent
func TestFilterConsistency(t *testing.T) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(50)
	ctx := generateTestContext()
	
	// Run filtering multiple times and verify results are consistent
	var firstResults []models.Task
	for i := 0; i < 10; i++ {
		visibleTasks, _ := engine.FilterTasks(ctx, tasks)
		
		if i == 0 {
			firstResults = visibleTasks
		} else {
			// Results should be consistent (same filter rules, same context)
			assert.Equal(t, len(firstResults), len(visibleTasks),
				"Filter results should be consistent across runs")
		}
	}
}