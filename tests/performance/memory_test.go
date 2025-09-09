package performance

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/stretchr/testify/assert"
)

// MemoryProfile represents memory usage at a specific point
type MemoryProfile struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	HeapSysMB    float64 `json:"heap_sys_mb"`
	StackInUseMB float64 `json:"stack_inuse_mb"`
	Timestamp    time.Time
}

func captureMemoryProfile() MemoryProfile {
	runtime.GC() // Force garbage collection for accurate reading
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return MemoryProfile{
		AllocMB:      float64(m.Alloc) / 1024 / 1024,
		TotalAllocMB: float64(m.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(m.Sys) / 1024 / 1024,
		NumGC:        m.NumGC,
		HeapAllocMB:  float64(m.HeapAlloc) / 1024 / 1024,
		HeapSysMB:    float64(m.HeapSys) / 1024 / 1024,
		StackInUseMB: float64(m.StackInuse) / 1024 / 1024,
		Timestamp:    time.Now(),
	}
}

// TestMemoryFootprintRequirement verifies the <50MB memory footprint requirement
func TestMemoryFootprintRequirement(t *testing.T) {
	// Baseline memory measurement
	baseline := captureMemoryProfile()
	t.Logf("Baseline memory: %.2f MB allocated, %.2f MB system", baseline.AllocMB, baseline.SysMB)
	
	// Create realistic workload components
	engine := setupFilterEngine()
	
	// Generate realistic data sets
	smallDataset := generateTestTasks(100)   // Typical user task count
	mediumDataset := generateTestTasks(500)  // Power user task count
	largeDataset := generateTestTasks(1000)  // Stress test dataset
	
	contexts := make([]models.Context, 10)
	for i := range contexts {
		contexts[i] = generateTestContext()
	}
	
	postSetup := captureMemoryProfile()
	setupMemoryMB := postSetup.AllocMB - baseline.AllocMB
	t.Logf("Memory after setup: %.2f MB (delta: +%.2f MB)", postSetup.AllocMB, setupMemoryMB)
	
	// Test memory usage with different workloads
	t.Run("SmallWorkload", func(t *testing.T) {
		preTest := captureMemoryProfile()
		
		// Perform operations
		for i := 0; i < 100; i++ {
			ctx := contexts[i%len(contexts)]
			_, _ = engine.FilterTasks(ctx, smallDataset)
		}
		
		postTest := captureMemoryProfile()
		workloadMemoryMB := postTest.AllocMB - preTest.AllocMB
		
		t.Logf("Small workload memory: %.2f MB (delta: +%.2f MB)", postTest.AllocMB, workloadMemoryMB)
		assert.Less(t, postTest.AllocMB, 25.0, "Small workload should use less than 25MB")
	})
	
	t.Run("MediumWorkload", func(t *testing.T) {
		runtime.GC()
		preTest := captureMemoryProfile()
		
		// Perform operations with medium dataset
		for i := 0; i < 50; i++ {
			ctx := contexts[i%len(contexts)]
			_, _ = engine.FilterTasks(ctx, mediumDataset)
		}
		
		postTest := captureMemoryProfile()
		workloadMemoryMB := postTest.AllocMB - preTest.AllocMB
		
		t.Logf("Medium workload memory: %.2f MB (delta: +%.2f MB)", postTest.AllocMB, workloadMemoryMB)
		assert.Less(t, postTest.AllocMB, 40.0, "Medium workload should use less than 40MB")
	})
	
	t.Run("LargeWorkload", func(t *testing.T) {
		runtime.GC()
		preTest := captureMemoryProfile()
		
		// Perform operations with large dataset
		for i := 0; i < 20; i++ {
			ctx := contexts[i%len(contexts)]
			_, _ = engine.FilterTasks(ctx, largeDataset)
		}
		
		postTest := captureMemoryProfile()
		workloadMemoryMB := postTest.AllocMB - preTest.AllocMB
		
		t.Logf("Large workload memory: %.2f MB (delta: +%.2f MB)", postTest.AllocMB, workloadMemoryMB)
		assert.Less(t, postTest.AllocMB, 50.0, "Large workload should use less than 50MB - REQUIREMENT")
	})
	
	// Final memory check
	runtime.GC()
	final := captureMemoryProfile()
	totalMemoryMB := final.AllocMB - baseline.AllocMB
	
	t.Logf("=== MEMORY FOOTPRINT RESULTS ===")
	t.Logf("Baseline: %.2f MB", baseline.AllocMB)
	t.Logf("Final: %.2f MB", final.AllocMB)
	t.Logf("Total Memory Used: %.2f MB", totalMemoryMB)
	t.Logf("System Memory: %.2f MB", final.SysMB)
	t.Logf("Heap Allocated: %.2f MB", final.HeapAllocMB)
	t.Logf("GC Runs: %d", final.NumGC - baseline.NumGC)
	
	// Verify the core requirement
	assert.Less(t, final.AllocMB, 50.0, 
		"Application should use less than 50MB of memory - CORE REQUIREMENT")
}

// TestMemoryLeakDetection verifies no memory leaks during sustained operation
func TestMemoryLeakDetection(t *testing.T) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(200)
	ctx := generateTestContext()
	
	const numRounds = 10
	const operationsPerRound = 100
	
	memoryReadings := make([]float64, numRounds)
	
	for round := 0; round < numRounds; round++ {
		runtime.GC()
		
		// Perform operations
		for i := 0; i < operationsPerRound; i++ {
			_, _ = engine.FilterTasks(ctx, tasks)
			
			// Occasionally update context to simulate real usage
			if i%20 == 0 {
				ctx.EnergyLevel = (ctx.EnergyLevel % 5) + 1
				ctx.AvailableMinutes = ctx.AvailableMinutes + 10
			}
		}
		
		runtime.GC()
		afterRound := captureMemoryProfile()
		memoryReadings[round] = afterRound.AllocMB
		
		t.Logf("Round %d: %.2f MB (operations: %d)", round+1, afterRound.AllocMB, operationsPerRound)
	}
	
	// Analyze memory trend
	firstThreeAvg := (memoryReadings[0] + memoryReadings[1] + memoryReadings[2]) / 3
	lastThreeAvg := (memoryReadings[numRounds-3] + memoryReadings[numRounds-2] + memoryReadings[numRounds-1]) / 3
	
	memoryGrowthMB := lastThreeAvg - firstThreeAvg
	growthPercentage := (memoryGrowthMB / firstThreeAvg) * 100
	
	t.Logf("Memory growth analysis:")
	t.Logf("First 3 rounds avg: %.2f MB", firstThreeAvg)
	t.Logf("Last 3 rounds avg: %.2f MB", lastThreeAvg)
	t.Logf("Growth: %.2f MB (%.1f%%)", memoryGrowthMB, growthPercentage)
	
	// Memory should not grow significantly (allowing for some variance)
	assert.Less(t, growthPercentage, 20.0, 
		"Memory growth should be less than 20% indicating no significant leaks")
	assert.Less(t, lastThreeAvg, 50.0, 
		"Final memory usage should still be under 50MB")
}

// TestMemoryUnderLoad verifies memory behavior under concurrent load
func TestMemoryUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory load test in short mode")
	}
	
	const numUsers = 10
	const testDuration = 10 * time.Second
	
	baseline := captureMemoryProfile()
	
	sessions := make([]*UserSession, numUsers)
	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("memory-load-user-%d", i)
		sessions[i] = createUserSession(userID, 100)
	}
	
	postSetup := captureMemoryProfile()
	setupMemoryMB := postSetup.AllocMB - baseline.AllocMB
	t.Logf("Memory after session setup: %.2f MB (delta: +%.2f MB)", postSetup.AllocMB, setupMemoryMB)
	
	// Memory readings during load
	memoryReadings := make([]MemoryProfile, 0)
	done := make(chan bool, numUsers)
	
	// Start memory monitoring
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for i := 0; i < int(testDuration.Seconds()); i++ {
			<-ticker.C
			profile := captureMemoryProfile()
			memoryReadings = append(memoryReadings, profile)
		}
	}()
	
	// Start user sessions
	for _, session := range sessions {
		go func(s *UserSession) {
			endTime := time.Now().Add(testDuration)
			for time.Now().Before(endTime) {
				s.performFilterTasks()
				time.Sleep(50 * time.Millisecond)
			}
			done <- true
		}(session)
	}
	
	// Wait for all sessions to complete
	for i := 0; i < numUsers; i++ {
		<-done
	}
	
	// Final memory measurement
	runtime.GC()
	final := captureMemoryProfile()
	peakMemory := final.AllocMB
	
	// Find peak memory during test
	for _, reading := range memoryReadings {
		if reading.AllocMB > peakMemory {
			peakMemory = reading.AllocMB
		}
	}
	
	t.Logf("=== MEMORY UNDER LOAD RESULTS ===")
	t.Logf("Baseline: %.2f MB", baseline.AllocMB)
	t.Logf("Peak during load: %.2f MB", peakMemory)
	t.Logf("Final after load: %.2f MB", final.AllocMB)
	t.Logf("Memory growth under load: %.2f MB", peakMemory-baseline.AllocMB)
	
	assert.Less(t, peakMemory, 50.0, 
		"Peak memory usage under load should be less than 50MB")
	assert.Less(t, final.AllocMB, 50.0, 
		"Final memory after load should be less than 50MB")
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(100)
	ctx := generateTestContext()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = engine.FilterTasks(ctx, tasks)
	}
}

// TestMemoryProfileGeneration generates memory profiles for analysis
func TestMemoryProfileGeneration(t *testing.T) {
	if os.Getenv("MEMORY_PROFILE") != "1" {
		t.Skip("Set MEMORY_PROFILE=1 to generate memory profiles")
	}
	
	// Create CPU and memory profiles
	cpuFile, err := os.Create("cpu_profile.prof")
	if err != nil {
		t.Fatal(err)
	}
	defer cpuFile.Close()
	
	memFile, err := os.Create("mem_profile.prof")
	if err != nil {
		t.Fatal(err)
	}
	defer memFile.Close()
	
	// Start CPU profiling
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		t.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	
	// Run workload
	engine := setupFilterEngine()
	tasks := generateTestTasks(1000)
	
	for i := 0; i < 1000; i++ {
		ctx := generateTestContext()
		_, _ = engine.FilterTasks(ctx, tasks)
	}
	
	// Write memory profile
	runtime.GC()
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		t.Fatal(err)
	}
	
	t.Log("Memory and CPU profiles generated:")
	t.Log("  cpu_profile.prof - analyze with: go tool pprof cpu_profile.prof")
	t.Log("  mem_profile.prof - analyze with: go tool pprof mem_profile.prof")
}

// TestGarbageCollectionBehavior verifies GC behavior under load
func TestGarbageCollectionBehavior(t *testing.T) {
	engine := setupFilterEngine()
	tasks := generateTestTasks(500)
	
	// Initial GC stats
	var initialStats runtime.MemStats
	runtime.ReadMemStats(&initialStats)
	
	// Perform memory-intensive operations
	for i := 0; i < 200; i++ {
		ctx := generateTestContext()
		_, results := engine.FilterTasks(ctx, tasks)
		
		// Force some additional allocations
		_ = fmt.Sprintf("iteration_%d_results_%d", i, len(results))
		
		if i%50 == 0 {
			var midStats runtime.MemStats
			runtime.ReadMemStats(&midStats)
			t.Logf("Iteration %d: GC runs: %d, Memory: %.2f MB", 
				i, midStats.NumGC, float64(midStats.Alloc)/1024/1024)
		}
	}
	
	// Final GC stats
	runtime.GC()
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)
	
	gcRuns := finalStats.NumGC - initialStats.NumGC
	avgGCPause := finalStats.PauseTotalNs - initialStats.PauseTotalNs
	if gcRuns > 0 {
		avgGCPause = avgGCPause / uint64(gcRuns)
	}
	
	t.Logf("=== GARBAGE COLLECTION ANALYSIS ===")
	t.Logf("GC runs during test: %d", gcRuns)
	t.Logf("Average GC pause: %.2f ms", float64(avgGCPause)/1e6)
	t.Logf("Final memory: %.2f MB", float64(finalStats.Alloc)/1024/1024)
	t.Logf("Total allocations: %.2f MB", float64(finalStats.TotalAlloc-initialStats.TotalAlloc)/1024/1024)
	
	// GC should be reasonable
	assert.Less(t, float64(avgGCPause)/1e6, 10.0, 
		"Average GC pause should be less than 10ms")
	assert.Less(t, float64(finalStats.Alloc)/1024/1024, 50.0, 
		"Final memory should be less than 50MB")
}

// TestMemoryEfficiencyOfFilters tests memory efficiency of individual filters
func TestMemoryEfficiencyOfFilters(t *testing.T) {
	config := filters.DefaultFilterConfig
	auditRepo := &MockAuditRepo{}
	
	tests := []struct {
		filterName string
		setup      func(*filters.Engine)
	}{
		{
			filterName: "LocationFilter",
			setup: func(e *filters.Engine) {
				e.AddRule(&MockFilter{name: "location", priority: 100, passRate: 0.8, complexity: 50})
			},
		},
		{
			filterName: "TimeFilter", 
			setup: func(e *filters.Engine) {
				e.AddRule(&MockFilter{name: "time", priority: 90, passRate: 0.7, complexity: 20})
			},
		},
		{
			filterName: "DependencyFilter",
			setup: func(e *filters.Engine) {
				e.AddRule(&MockFilter{name: "dependency", priority: 80, passRate: 0.9, complexity: 100})
			},
		},
	}
	
	tasks := generateTestTasks(200)
	ctx := generateTestContext()
	
	for _, test := range tests {
		t.Run(test.filterName, func(t *testing.T) {
			runtime.GC()
			before := captureMemoryProfile()
			
			engine := filters.NewEngine(config, auditRepo)
			test.setup(engine)
			
			// Run multiple filtering operations
			for i := 0; i < 100; i++ {
				_, _ = engine.FilterTasks(ctx, tasks)
			}
			
			runtime.GC()
			after := captureMemoryProfile()
			
			memoryUsedMB := after.AllocMB - before.AllocMB
			t.Logf("%s memory usage: %.2f MB", test.filterName, memoryUsedMB)
			
			assert.Less(t, memoryUsedMB, 20.0, 
				"Individual filter should use less than 20MB")
		})
	}
}