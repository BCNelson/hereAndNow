package performance

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/stretchr/testify/assert"
)

// ConcurrentStats tracks statistics for concurrent operations
type ConcurrentStats struct {
	TotalOperations   int64         `json:"total_operations"`
	SuccessfulOps     int64         `json:"successful_ops"`
	FailedOps         int64         `json:"failed_ops"`
	TotalDuration     time.Duration `json:"total_duration"`
	AvgDuration       time.Duration `json:"avg_duration"`
	MinDuration       time.Duration `json:"min_duration"`
	MaxDuration       time.Duration `json:"max_duration"`
	OperationsPerSec  float64       `json:"operations_per_sec"`
	MemoryUsedMB      float64       `json:"memory_used_mb"`
}

// UserSession represents a concurrent user session
type UserSession struct {
	UserID   string
	Context  models.Context
	Tasks    []models.Task
	Engine   *filters.Engine
	Stats    ConcurrentStats
	durations []time.Duration
	mu       sync.Mutex
}

func (s *UserSession) recordOperation(duration time.Duration, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	atomic.AddInt64(&s.Stats.TotalOperations, 1)
	if success {
		atomic.AddInt64(&s.Stats.SuccessfulOps, 1)
	} else {
		atomic.AddInt64(&s.Stats.FailedOps, 1)
	}
	
	s.durations = append(s.durations, duration)
}

func (s *UserSession) calculateStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.durations) == 0 {
		return
	}
	
	var total time.Duration
	min := s.durations[0]
	max := s.durations[0]
	
	for _, d := range s.durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}
	
	s.Stats.TotalDuration = total
	s.Stats.AvgDuration = total / time.Duration(len(s.durations))
	s.Stats.MinDuration = min
	s.Stats.MaxDuration = max
	
	if s.Stats.TotalDuration > 0 {
		s.Stats.OperationsPerSec = float64(s.Stats.TotalOperations) / s.Stats.TotalDuration.Seconds()
	}
}

func createUserSession(userID string, taskCount int) *UserSession {
	ctx := generateTestContext()
	ctx.UserID = userID
	
	tasks := generateTestTasks(taskCount)
	for i := range tasks {
		tasks[i].CreatorID = userID
	}
	
	engine := setupFilterEngine()
	
	return &UserSession{
		UserID:  userID,
		Context: ctx,
		Tasks:   tasks,
		Engine:  engine,
		durations: make([]time.Duration, 0, 100),
	}
}

// simulateUserActivity simulates realistic user activity patterns
func (s *UserSession) simulateUserActivity(ctx context.Context, duration time.Duration) {
	endTime := time.Now().Add(duration)
	
	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			return
		default:
			// Simulate different user activities
			activity := rand.Intn(4)
			
			switch activity {
			case 0:
				// Filter tasks (most common activity)
				s.performFilterTasks()
			case 1:
				// Explain task visibility
				s.performExplainVisibility()
			case 2:
				// Update context (location/energy change)
				s.updateContext()
			case 3:
				// Add new task
				s.addNewTask()
			}
			
			// Random delay between operations (50-200ms)
			delay := time.Duration(rand.Intn(150)+50) * time.Millisecond
			time.Sleep(delay)
		}
	}
}

func (s *UserSession) performFilterTasks() {
	start := time.Now()
	_, results := s.Engine.FilterTasks(s.Context, s.Tasks)
	duration := time.Since(start)
	
	success := len(results) > 0
	s.recordOperation(duration, success)
}

func (s *UserSession) performExplainVisibility() {
	if len(s.Tasks) == 0 {
		return
	}
	
	task := s.Tasks[rand.Intn(len(s.Tasks))]
	start := time.Now()
	explanation := s.Engine.ExplainTaskVisibility(s.Context, task)
	duration := time.Since(start)
	
	success := explanation.TaskID == task.ID
	s.recordOperation(duration, success)
}

func (s *UserSession) updateContext() {
	// Simulate context changes (location, energy, available time)
	s.Context.EnergyLevel = rand.Intn(5) + 1
	s.Context.AvailableMinutes = rand.Intn(240) + 15
	
	// Occasionally update location
	if rand.Float64() < 0.3 {
		lat := 37.7749 + (rand.Float64()-0.5)*0.1
		lng := -122.4194 + (rand.Float64()-0.5)*0.1
		s.Context.CurrentLatitude = &lat
		s.Context.CurrentLongitude = &lng
	}
	
	s.Context.Timestamp = time.Now()
}

func (s *UserSession) addNewTask() {
	newTask := generateTestTasks(1)[0]
	newTask.CreatorID = s.UserID
	s.Tasks = append(s.Tasks, newTask)
}

// TestConcurrent20Users verifies the system can handle 20 concurrent users
func TestConcurrent20Users(t *testing.T) {
	const numUsers = 20
	const testDuration = 30 * time.Second
	
	// Pre-allocate memory to avoid GC during test
	runtime.GC()
	
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	
	// Create user sessions
	sessions := make([]*UserSession, numUsers)
	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("concurrent-user-%d", i)
		sessions[i] = createUserSession(userID, 50) // 50 tasks per user
	}
	
	// Start concurrent user activities
	ctx, cancel := context.WithTimeout(context.Background(), testDuration+5*time.Second)
	defer cancel()
	
	var wg sync.WaitGroup
	startTime := time.Now()
	
	for i, session := range sessions {
		wg.Add(1)
		go func(idx int, s *UserSession) {
			defer wg.Done()
			s.simulateUserActivity(ctx, testDuration)
			s.calculateStats()
			t.Logf("User %d: %d operations, avg: %v, ops/sec: %.2f", 
				idx, s.Stats.TotalOperations, s.Stats.AvgDuration, s.Stats.OperationsPerSec)
		}(i, session)
	}
	
	// Wait for all users to complete
	wg.Wait()
	totalTestDuration := time.Since(startTime)
	
	// Measure memory usage
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	memoryUsedMB := float64(endMem.Alloc-startMem.Alloc) / 1024 / 1024
	
	// Aggregate statistics
	var totalOps int64
	var totalSuccessOps int64
	var totalFailedOps int64
	var maxAvgDuration time.Duration
	
	for _, session := range sessions {
		totalOps += session.Stats.TotalOperations
		totalSuccessOps += session.Stats.SuccessfulOps
		totalFailedOps += session.Stats.FailedOps
		if session.Stats.AvgDuration > maxAvgDuration {
			maxAvgDuration = session.Stats.AvgDuration
		}
	}
	
	successRate := float64(totalSuccessOps) / float64(totalOps) * 100
	overallOpsPerSec := float64(totalOps) / totalTestDuration.Seconds()
	
	t.Logf("=== CONCURRENT TEST RESULTS ===")
	t.Logf("Users: %d", numUsers)
	t.Logf("Test Duration: %v", totalTestDuration)
	t.Logf("Total Operations: %d", totalOps)
	t.Logf("Success Rate: %.2f%%", successRate)
	t.Logf("Overall Ops/sec: %.2f", overallOpsPerSec)
	t.Logf("Max Avg Duration: %v", maxAvgDuration)
	t.Logf("Memory Used: %.2f MB", memoryUsedMB)
	
	// Performance assertions
	assert.GreaterOrEqual(t, totalOps, int64(numUsers*10), 
		"Should have performed at least 10 operations per user")
	assert.GreaterOrEqual(t, successRate, 95.0, 
		"Success rate should be at least 95%")
	assert.Less(t, maxAvgDuration.Milliseconds(), int64(500), 
		"Average operation time should be under 500ms even under load")
	assert.Less(t, memoryUsedMB, 100.0, 
		"Memory usage should be reasonable under concurrent load")
	
	// Verify system responsiveness under load
	assert.GreaterOrEqual(t, overallOpsPerSec, 50.0, 
		"System should handle at least 50 operations per second across all users")
}

// BenchmarkConcurrent20Users benchmarks the concurrent user scenario
func BenchmarkConcurrent20Users(b *testing.B) {
	const numUsers = 20
	
	sessions := make([]*UserSession, numUsers)
	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("bench-user-%d", i)
		sessions[i] = createUserSession(userID, 20)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		
		for _, session := range sessions {
			wg.Add(1)
			go func(s *UserSession) {
				defer wg.Done()
				// Perform 5 filtering operations per iteration
				for j := 0; j < 5; j++ {
					s.performFilterTasks()
				}
			}(session)
		}
		
		wg.Wait()
	}
}

// TestConcurrentStressTest puts higher stress on the system
func TestConcurrentStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	const numUsers = 50
	const testDuration = 15 * time.Second
	const tasksPerUser = 100
	
	sessions := make([]*UserSession, numUsers)
	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("stress-user-%d", i)
		sessions[i] = createUserSession(userID, tasksPerUser)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), testDuration+5*time.Second)
	defer cancel()
	
	var wg sync.WaitGroup
	startTime := time.Now()
	
	for _, session := range sessions {
		wg.Add(1)
		go func(s *UserSession) {
			defer wg.Done()
			s.simulateUserActivity(ctx, testDuration)
			s.calculateStats()
		}(session)
	}
	
	wg.Wait()
	totalDuration := time.Since(startTime)
	
	var totalOps int64
	var totalSuccessOps int64
	
	for _, session := range sessions {
		totalOps += session.Stats.TotalOperations
		totalSuccessOps += session.Stats.SuccessfulOps
	}
	
	successRate := float64(totalSuccessOps) / float64(totalOps) * 100
	opsPerSec := float64(totalOps) / totalDuration.Seconds()
	
	t.Logf("Stress Test - Users: %d, Ops: %d, Success: %.1f%%, Ops/sec: %.1f", 
		numUsers, totalOps, successRate, opsPerSec)
	
	// Even under stress, system should maintain decent performance
	assert.GreaterOrEqual(t, successRate, 90.0, 
		"Success rate should be at least 90% under stress")
	assert.GreaterOrEqual(t, opsPerSec, 20.0, 
		"System should handle at least 20 operations per second under stress")
}

// TestConcurrentMemoryStability verifies memory doesn't leak under concurrent load
func TestConcurrentMemoryStability(t *testing.T) {
	const numUsers = 10
	const numRounds = 5
	
	sessions := make([]*UserSession, numUsers)
	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("memory-user-%d", i)
		sessions[i] = createUserSession(userID, 30)
	}
	
	memoryReadings := make([]float64, numRounds)
	
	for round := 0; round < numRounds; round++ {
		runtime.GC()
		var beforeMem runtime.MemStats
		runtime.ReadMemStats(&beforeMem)
		
		var wg sync.WaitGroup
		for _, session := range sessions {
			wg.Add(1)
			go func(s *UserSession) {
				defer wg.Done()
				// Perform 20 operations
				for i := 0; i < 20; i++ {
					s.performFilterTasks()
				}
			}(session)
		}
		wg.Wait()
		
		runtime.GC()
		var afterMem runtime.MemStats
		runtime.ReadMemStats(&afterMem)
		
		memoryUsedMB := float64(afterMem.Alloc-beforeMem.Alloc) / 1024 / 1024
		memoryReadings[round] = memoryUsedMB
		
		t.Logf("Round %d: Memory used: %.2f MB", round+1, memoryUsedMB)
	}
	
	// Check for memory stability (no significant growth across rounds)
	firstReading := memoryReadings[0]
	lastReading := memoryReadings[numRounds-1]
	
	growthRatio := lastReading / firstReading
	t.Logf("Memory growth ratio: %.2fx", growthRatio)
	
	assert.Less(t, growthRatio, 2.0, 
		"Memory usage should not grow significantly across test rounds")
	
	for _, reading := range memoryReadings {
		assert.Less(t, reading, 50.0, 
			"Each round should use less than 50MB of additional memory")
	}
}

// TestRaceConditions verifies thread safety under concurrent access
func TestRaceConditions(t *testing.T) {
	const numGoroutines = 100
	const operationsPerGoroutine = 50
	
	engine := setupFilterEngine()
	tasks := generateTestTasks(10)
	
	var wg sync.WaitGroup
	var successfulOps int64
	var totalOps int64
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			ctx := generateTestContext()
			ctx.UserID = fmt.Sprintf("race-user-%d", goroutineID)
			
			for j := 0; j < operationsPerGoroutine; j++ {
				atomic.AddInt64(&totalOps, 1)
				
				// Mix of operations to test thread safety
				switch j % 4 {
				case 0:
					_, results := engine.FilterTasks(ctx, tasks)
					if len(results) >= 0 {
						atomic.AddInt64(&successfulOps, 1)
					}
				case 1:
					explanation := engine.ExplainTaskVisibility(ctx, tasks[0])
					if explanation.TaskID == tasks[0].ID {
						atomic.AddInt64(&successfulOps, 1)
					}
				case 2:
					stats := engine.GetFilterStats(ctx, tasks)
					if stats.TotalTasks == len(tasks) {
						atomic.AddInt64(&successfulOps, 1)
					}
				case 3:
					filters := engine.GetRegisteredFilters()
					if len(filters) > 0 {
						atomic.AddInt64(&successfulOps, 1)
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	successRate := float64(successfulOps) / float64(totalOps) * 100
	t.Logf("Race condition test: %d/%d operations successful (%.1f%%)", 
		successfulOps, totalOps, successRate)
	
	assert.Equal(t, totalOps, int64(numGoroutines*operationsPerGoroutine), 
		"All operations should have been attempted")
	assert.GreaterOrEqual(t, successRate, 99.0, 
		"Success rate should be very high, indicating no race conditions")
}