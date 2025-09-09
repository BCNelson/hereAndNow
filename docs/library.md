# Here and Now Go Library Documentation

The Here and Now Go library provides a complete context-aware task management solution that can be embedded in any Go application. This library implements intelligent task filtering based on location, time, dependencies, and other contextual factors.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Components](#core-components)
- [Task Management](#task-management)
- [Context Management](#context-management)
- [Filtering Engine](#filtering-engine)
- [Data Models](#data-models)
- [Advanced Usage](#advanced-usage)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Installation

Add the library to your Go project:

```bash
go get github.com/bcnelson/hereAndNow
```

## Quick Start

Here's a minimal example to get started with the library:

```go
package main

import (
    "log"
    "time"
    
    "github.com/bcnelson/hereAndNow/pkg/hereandnow"
    "github.com/bcnelson/hereAndNow/pkg/filters"
    "github.com/bcnelson/hereAndNow/pkg/models"
)

func main() {
    // Initialize repositories (implement these interfaces)
    taskRepo := &MyTaskRepository{}
    contextRepo := &MyContextRepository{}
    dependencyRepo := &MyTaskDependencyRepository{}
    taskLocationRepo := &MyTaskLocationRepository{}
    auditRepo := &MyFilterAuditRepository{}
    
    // Create filter engine
    config := filters.FilterConfig{
        EnableLocationFilter:   true,
        EnableTimeFilter:      true,
        EnableDependencyFilter: true,
        EnablePriorityFilter:  true,
        LocationRadiusMeters:  100,
        MinEnergyLevel:        1,
    }
    
    filterEngine := filters.NewEngine(config, auditRepo)
    
    // Add filter rules
    filterEngine.AddRule(&filters.LocationFilter{})
    filterEngine.AddRule(&filters.TimeFilter{})
    filterEngine.AddRule(&filters.DependencyFilter{})
    filterEngine.AddRule(&filters.PriorityFilter{})
    
    // Create task service
    taskService := hereandnow.NewTaskService(
        taskRepo,
        contextRepo,
        dependencyRepo,
        taskLocationRepo,
        filterEngine,
    )
    
    // Create a task
    task, err := taskService.CreateTask("user-123", hereandnow.CreateTaskRequest{
        Title:            "Buy groceries",
        Description:      "Pick up milk, bread, and eggs",
        Priority:         3,
        EstimatedMinutes: &[]int{30}[0],
        LocationIDs:      []string{"grocery-store-location"},
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Get filtered tasks based on current context
    tasks, filterResults, err := taskService.GetFilteredTasks("user-123")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Found %d visible tasks", len(tasks))
    for _, result := range filterResults {
        log.Printf("Task %s: %s - %s", result.TaskID, result.FilterName, result.Reason)
    }
}
```

## Core Components

### Task Service (`hereandnow.TaskService`)

The central service for task management operations:

```go
type TaskService struct {
    taskRepo         TaskRepository
    contextRepo      ContextRepository
    dependencyRepo   TaskDependencyRepository
    taskLocationRepo TaskLocationRepository
    filterEngine     filters.FilterEngine
}
```

**Key Methods:**
- `CreateTask(userID string, req CreateTaskRequest) (*models.Task, error)`
- `GetFilteredTasks(userID string) ([]models.Task, []filters.FilterResult, error)`
- `UpdateTask(taskID string, req UpdateTaskRequest) (*models.Task, error)`
- `CompleteTask(taskID string, userID string) (*models.Task, error)`
- `ExplainTaskVisibility(taskID, userID string) (*filters.TaskVisibilityExplanation, error)`

### Context Service (`hereandnow.ContextService`)

Manages user context for filtering decisions:

```go
type ContextService struct {
    contextRepo ContextRepository
    locationRepo LocationRepository
}
```

**Key Methods:**
- `UpdateContext(userID string, req UpdateContextRequest) (*models.Context, error)`
- `GetCurrentContext(userID string) (*models.Context, error)`
- `DetectLocationChanges(userID string, lat, lng float64) ([]models.Location, error)`

### Filter Engine (`filters.Engine`)

The core filtering system that determines task visibility:

```go
type Engine struct {
    rules       []FilterRule
    auditRepo   FilterAuditRepository
    config      FilterConfig
}
```

**Key Methods:**
- `FilterTasks(ctx models.Context, tasks []models.Task) ([]models.Task, []FilterResult)`
- `AddRule(rule FilterRule)`
- `ExplainTaskVisibility(ctx models.Context, task models.Task) TaskVisibilityExplanation`

## Task Management

### Creating Tasks

```go
func createTaskExample(taskService *hereandnow.TaskService) {
    req := hereandnow.CreateTaskRequest{
        Title:            "Complete project proposal",
        Description:      "Draft and review the Q4 project proposal document",
        Priority:         2,                    // 1-10 scale
        EstimatedMinutes: &[]int{120}[0],       // 2 hours
        DueAt:            &[]time.Time{time.Now().Add(48 * time.Hour)}[0],
        LocationIDs:      []string{"office-location", "home-office-location"},
        Dependencies: []hereandnow.TaskDependencyRequest{
            {
                DependsOnTaskID: "research-task-id",
                DependencyType:  models.DependencyTypeFinishToStart,
            },
        },
    }
    
    task, err := taskService.CreateTask("user-123", req)
    if err != nil {
        log.Printf("Failed to create task: %v", err)
        return
    }
    
    log.Printf("Created task %s: %s", task.ID, task.Title)
}
```

### Updating Tasks

```go
func updateTaskExample(taskService *hereandnow.TaskService, taskID string) {
    newPriority := 1  // High priority
    newStatus := models.TaskStatusActive
    
    req := hereandnow.UpdateTaskRequest{
        Priority: &newPriority,
        Status:   &newStatus,
    }
    
    task, err := taskService.UpdateTask(taskID, req)
    if err != nil {
        log.Printf("Failed to update task: %v", err)
        return
    }
    
    log.Printf("Updated task %s status to %s", task.ID, task.Status)
}
```

### Context-Aware Task Retrieval

The library's core feature is intelligent task filtering based on context:

```go
func getRelevantTasks(taskService *hereandnow.TaskService, userID string) {
    // Get tasks filtered by current context
    tasks, filterResults, err := taskService.GetFilteredTasks(userID)
    if err != nil {
        log.Printf("Failed to get filtered tasks: %v", err)
        return
    }
    
    log.Printf("Context shows %d relevant tasks out of all user tasks", len(tasks))
    
    // Examine why each task was shown or hidden
    for _, result := range filterResults {
        status := "HIDDEN"
        if result.Visible {
            status = "VISIBLE"
        }
        log.Printf("Task %s [%s]: %s - %s", 
            result.TaskID, status, result.FilterName, result.Reason)
    }
}
```

## Context Management

Context determines which tasks are relevant based on the user's current situation:

```go
func updateUserContext(contextService *hereandnow.ContextService, userID string) {
    req := hereandnow.UpdateContextRequest{
        CurrentLatitude:  &[]float64{40.7128}[0],  // NYC
        CurrentLongitude: &[]float64{-74.0060}[0],
        AvailableMinutes: &[]int{60}[0],           // 1 hour available
        EnergyLevel:      &[]int{4}[0],            // High energy (1-5 scale)
        SocialContext:    &[]string{"alone"}[0],
    }
    
    context, err := contextService.UpdateContext(userID, req)
    if err != nil {
        log.Printf("Failed to update context: %v", err)
        return
    }
    
    log.Printf("Updated context for user %s at location (%.4f, %.4f)", 
        userID, context.CurrentLatitude, context.CurrentLongitude)
}
```

### Location Detection

```go
func handleLocationChange(contextService *hereandnow.ContextService, userID string, lat, lng float64) {
    // Detect if user entered any defined locations
    locations, err := contextService.DetectLocationChanges(userID, lat, lng)
    if err != nil {
        log.Printf("Failed to detect location changes: %v", err)
        return
    }
    
    if len(locations) > 0 {
        for _, location := range locations {
            log.Printf("User entered location: %s (%.4f, %.4f, %dm radius)", 
                location.Name, location.Latitude, location.Longitude, location.Radius)
        }
        
        // Update context will automatically trigger task filtering
        contextService.UpdateContext(userID, hereandnow.UpdateContextRequest{
            CurrentLatitude:  &lat,
            CurrentLongitude: &lng,
        })
    }
}
```

## Filtering Engine

### Built-in Filter Rules

The library provides several built-in filter rules:

#### 1. Location Filter

Shows tasks only when the user is within the required location radius:

```go
type LocationFilter struct {
    maxDistanceMeters float64
}

func (f *LocationFilter) Apply(ctx models.Context, task models.Task) (bool, string) {
    // Implementation checks if user is within task location radius
    // Returns (true, "Within 150m of required location") or
    //         (false, "User 2.3km away from required location")
}
```

#### 2. Time Filter

Shows tasks only when there's sufficient available time:

```go
type TimeFilter struct {
    bufferMinutes int
}

func (f *TimeFilter) Apply(ctx models.Context, task models.Task) (bool, string) {
    // Compares available time vs estimated task duration
    // Returns (true, "60 min available >= 30 min estimated") or
    //         (false, "Only 15 min available < 45 min estimated")
}
```

#### 3. Dependency Filter

Shows tasks only when prerequisites are completed:

```go
type DependencyFilter struct{}

func (f *DependencyFilter) Apply(ctx models.Context, task models.Task) (bool, string) {
    // Checks if all dependent tasks are completed
    // Returns (true, "All 2 dependencies completed") or
    //         (false, "Waiting for 'Research phase' to complete")
}
```

#### 4. Priority Filter

Adjusts task visibility based on available energy and task priority:

```go
type PriorityFilter struct{}

func (f *PriorityFilter) Apply(ctx models.Context, task models.Task) (bool, string) {
    // Matches task energy requirements with user energy level
    // Returns (true, "Energy level 4 >= task requirement 2") or
    //         (false, "Energy level 2 < high-energy task requirement 4")
}
```

### Custom Filter Rules

Create custom filters by implementing the `FilterRule` interface:

```go
type FilterRule interface {
    Apply(ctx models.Context, task models.Task) (bool, string)
    Name() string
    Priority() int
}

// Example: Weather-based filter
type WeatherFilter struct{}

func (f *WeatherFilter) Apply(ctx models.Context, task models.Task) (bool, string) {
    // Custom logic based on weather conditions
    if task.Category == "outdoor" && ctx.WeatherCondition == "rain" {
        return false, "Outdoor task hidden due to rainy weather"
    }
    return true, "Weather conditions suitable"
}

func (f *WeatherFilter) Name() string     { return "weather" }
func (f *WeatherFilter) Priority() int   { return 80 }

// Add to engine
filterEngine.AddRule(&WeatherFilter{})
```

### Filter Configuration

Configure filtering behavior:

```go
config := filters.FilterConfig{
    EnableLocationFilter:   true,
    EnableTimeFilter:      true,
    EnableDependencyFilter: true,
    EnablePriorityFilter:  true,
    LocationRadiusMeters:  200,    // Expand location matching radius
    MinEnergyLevel:        2,      // Hide low-energy tasks when energy < 2
    TimeBufferMinutes:     10,     // Require 10 min buffer beyond estimated time
}

filterEngine.UpdateConfig(config)
```

## Data Models

### Task Model

```go
type Task struct {
    ID               string          `db:"id" json:"id"`
    Title            string          `db:"title" json:"title"`
    Description      string          `db:"description" json:"description"`
    CreatorID        string          `db:"creator_id" json:"creator_id"`
    AssigneeID       *string         `db:"assignee_id" json:"assignee_id"`
    ListID           *string         `db:"list_id" json:"list_id"`
    Status           TaskStatus      `db:"status" json:"status"`
    Priority         int             `db:"priority" json:"priority"`
    EstimatedMinutes *int            `db:"estimated_minutes" json:"estimated_minutes"`
    DueAt            *time.Time      `db:"due_at" json:"due_at"`
    CompletedAt      *time.Time      `db:"completed_at" json:"completed_at"`
    CreatedAt        time.Time       `db:"created_at" json:"created_at"`
    UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
    Metadata         json.RawMessage `db:"metadata" json:"metadata"`
    RecurrenceRule   *string         `db:"recurrence_rule" json:"recurrence_rule"`
    ParentTaskID     *string         `db:"parent_task_id" json:"parent_task_id"`
}
```

**Creating and validating tasks:**

```go
// Create a new task with validation
task, err := models.NewTask("Complete documentation", "Write library docs", "user-123")
if err != nil {
    log.Printf("Invalid task: %v", err)
    return
}

// Validate existing task
if err := task.Validate(); err != nil {
    log.Printf("Task validation failed: %v", err)
    return
}

// Update task properties with validation
err = task.SetPriority(1)  // High priority
if err != nil {
    log.Printf("Invalid priority: %v", err)
}

err = task.SetEstimatedMinutes(45)
if err != nil {
    log.Printf("Invalid estimated minutes: %v", err)
}
```

### Location Model

```go
type Location struct {
    ID        string          `db:"id" json:"id"`
    UserID    string          `db:"user_id" json:"user_id"`
    Name      string          `db:"name" json:"name"`
    Address   string          `db:"address" json:"address"`
    Latitude  float64         `db:"latitude" json:"latitude"`
    Longitude float64         `db:"longitude" json:"longitude"`
    Radius    int             `db:"radius" json:"radius"`
    Category  string          `db:"category" json:"category"`
    PlaceID   *string         `db:"place_id" json:"place_id"`
    Metadata  json.RawMessage `db:"metadata" json:"metadata"`
    CreatedAt time.Time       `db:"created_at" json:"created_at"`
    UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}
```

**Working with locations:**

```go
// Create a location
location, err := models.NewLocation(
    "user-123",
    "Home Office", 
    "123 Main St, City, ST 12345",
    40.7128,    // latitude
    -74.0060,   // longitude
    100,        // radius in meters
)
if err != nil {
    log.Printf("Invalid location: %v", err)
    return
}

// Check if user is within location
if location.IsWithinRadius(40.7130, -74.0062) {
    log.Printf("User is within %s location", location.Name)
}

// Calculate distance
distance := location.DistanceFrom(40.7200, -74.0100)
log.Printf("Distance to %s: %.2f meters", location.Name, distance)
```

### Context Model

```go
type Context struct {
    ID                   string     `db:"id" json:"id"`
    UserID               string     `db:"user_id" json:"user_id"`
    Timestamp            time.Time  `db:"timestamp" json:"timestamp"`
    CurrentLatitude      *float64   `db:"current_latitude" json:"current_latitude"`
    CurrentLongitude     *float64   `db:"current_longitude" json:"current_longitude"`
    CurrentLocationID    *string    `db:"current_location_id" json:"current_location_id"`
    AvailableMinutes     *int       `db:"available_minutes" json:"available_minutes"`
    SocialContext        *string    `db:"social_context" json:"social_context"`
    EnergyLevel          *int       `db:"energy_level" json:"energy_level"`
    WeatherCondition     *string    `db:"weather_condition" json:"weather_condition"`
    TrafficLevel         *string    `db:"traffic_level" json:"traffic_level"`
    DeviceContext        *string    `db:"device_context" json:"device_context"`
    EnvironmentNoise     *int       `db:"environment_noise" json:"environment_noise"`
    LastCalendarSync     *time.Time `db:"last_calendar_sync" json:"last_calendar_sync"`
    ActiveCalendarEvents int        `db:"active_calendar_events" json:"active_calendar_events"`
    Metadata             json.RawMessage `db:"metadata" json:"metadata"`
}
```

## Advanced Usage

### Audit and Transparency

Get detailed explanations for filtering decisions:

```go
func explainTaskVisibility(taskService *hereandnow.TaskService, taskID, userID string) {
    explanation, err := taskService.ExplainTaskVisibility(taskID, userID)
    if err != nil {
        log.Printf("Failed to explain task visibility: %v", err)
        return
    }
    
    log.Printf("Task '%s' is %s", explanation.TaskTitle, 
        map[bool]string{true: "VISIBLE", false: "HIDDEN"}[explanation.IsVisible])
    
    for _, result := range explanation.FilterResults {
        status := "PASS"
        if !result.Passed {
            status = "FAIL"
        }
        log.Printf("  %s [%s]: %s", result.FilterName, status, result.Reason)
    }
}
```

### Performance Monitoring

Track filter performance and statistics:

```go
func analyzeFilterPerformance(filterEngine *filters.Engine, ctx models.Context, tasks []models.Task) {
    stats := filterEngine.GetFilterStats(ctx, tasks)
    
    log.Printf("Filter Performance:")
    log.Printf("  Total tasks: %d", stats.TotalTasks)
    log.Printf("  Visible tasks: %d", stats.VisibleTasks)
    log.Printf("  Filter efficiency: %.1f%%", 
        float64(stats.VisibleTasks)/float64(stats.TotalTasks)*100)
    
    for filterName, filterStats := range stats.FilterResults {
        log.Printf("  %s filter:", filterName)
        log.Printf("    Visible: %d, Hidden: %d", filterStats.TasksVisible, filterStats.TasksHidden)
        
        for reason, count := range filterStats.Reasons {
            log.Printf("    '%s': %d tasks", reason, count)
        }
    }
}
```

### Real-time Updates

Handle context changes and task updates:

```go
func handleRealTimeUpdates(taskService *hereandnow.TaskService, contextService *hereandnow.ContextService) {
    userID := "user-123"
    
    // Simulate location change
    newContext := hereandnow.UpdateContextRequest{
        CurrentLatitude:  &[]float64{40.7589}[0],  // Times Square
        CurrentLongitude: &[]float64{-73.9851}[0],
        AvailableMinutes: &[]int{30}[0],
    }
    
    context, err := contextService.UpdateContext(userID, newContext)
    if err != nil {
        log.Printf("Failed to update context: %v", err)
        return
    }
    
    // Get updated task list
    tasks, _, err := taskService.GetFilteredTasks(userID)
    if err != nil {
        log.Printf("Failed to get updated tasks: %v", err)
        return
    }
    
    log.Printf("Context change resulted in %d visible tasks", len(tasks))
    
    // Notify UI or external systems about task list changes
    notifyTaskListChanged(userID, tasks)
}

func notifyTaskListChanged(userID string, tasks []models.Task) {
    // Implementation depends on your notification system
    // Could be WebSocket, SSE, message queue, etc.
}
```

## Best Practices

### 1. Repository Implementation

Implement repositories with proper error handling and performance considerations:

```go
type SQLiteTaskRepository struct {
    db *sql.DB
}

func (r *SQLiteTaskRepository) Create(task models.Task) error {
    query := `
        INSERT INTO tasks (id, title, description, creator_id, status, priority, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `
    
    _, err := r.db.Exec(query, 
        task.ID, task.Title, task.Description, task.CreatorID,
        task.Status, task.Priority, task.CreatedAt, task.UpdatedAt)
    
    if err != nil {
        return fmt.Errorf("failed to insert task: %w", err)
    }
    
    return nil
}

func (r *SQLiteTaskRepository) GetByUserID(userID string) ([]models.Task, error) {
    query := `
        SELECT id, title, description, creator_id, status, priority, 
               estimated_minutes, due_at, completed_at, created_at, updated_at
        FROM tasks 
        WHERE creator_id = ? OR assignee_id = ?
        ORDER BY priority DESC, created_at DESC
    `
    
    rows, err := r.db.Query(query, userID, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to query tasks: %w", err)
    }
    defer rows.Close()
    
    var tasks []models.Task
    for rows.Next() {
        var task models.Task
        err := rows.Scan(
            &task.ID, &task.Title, &task.Description, &task.CreatorID,
            &task.Status, &task.Priority, &task.EstimatedMinutes,
            &task.DueAt, &task.CompletedAt, &task.CreatedAt, &task.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan task: %w", err)
        }
        tasks = append(tasks, task)
    }
    
    return tasks, nil
}
```

### 2. Context Updates

Update context efficiently and handle edge cases:

```go
func efficientContextUpdate(contextService *hereandnow.ContextService, userID string, lat, lng float64) {
    // Get current context to avoid unnecessary updates
    currentContext, err := contextService.GetCurrentContext(userID)
    if err != nil {
        log.Printf("Failed to get current context: %v", err)
        return
    }
    
    // Check if location changed significantly (>10 meters)
    if currentContext.CurrentLatitude != nil && currentContext.CurrentLongitude != nil {
        distance := haversineDistance(
            *currentContext.CurrentLatitude, *currentContext.CurrentLongitude,
            lat, lng,
        )
        
        if distance < 10 { // Less than 10 meters movement
            return // Skip update
        }
    }
    
    // Update with new location
    req := hereandnow.UpdateContextRequest{
        CurrentLatitude:  &lat,
        CurrentLongitude: &lng,
    }
    
    _, err = contextService.UpdateContext(userID, req)
    if err != nil {
        log.Printf("Failed to update context: %v", err)
    }
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
    // Implementation of Haversine formula
    const R = 6371000 // Earth radius in meters
    // ... formula implementation
    return R * c
}
```

### 3. Error Handling

Implement comprehensive error handling:

```go
func robustTaskOperation(taskService *hereandnow.TaskService) {
    userID := "user-123"
    
    // Use context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Wrap operations with error handling
    tasks, err := safeGetFilteredTasks(ctx, taskService, userID)
    if err != nil {
        log.Printf("Task operation failed: %v", err)
        return
    }
    
    log.Printf("Successfully retrieved %d tasks", len(tasks))
}

func safeGetFilteredTasks(ctx context.Context, taskService *hereandnow.TaskService, userID string) ([]models.Task, error) {
    // Channel to receive result
    type result struct {
        tasks []models.Task
        err   error
    }
    
    resultChan := make(chan result, 1)
    
    // Run operation in goroutine
    go func() {
        tasks, _, err := taskService.GetFilteredTasks(userID)
        resultChan <- result{tasks: tasks, err: err}
    }()
    
    // Wait for result or timeout
    select {
    case res := <-resultChan:
        return res.tasks, res.err
    case <-ctx.Done():
        return nil, fmt.Errorf("operation timed out: %w", ctx.Err())
    }
}
```

### 4. Memory Management

For applications processing many tasks:

```go
func processLargeTaskSet(taskService *hereandnow.TaskService, userID string) {
    const batchSize = 100
    
    // Process tasks in batches to control memory usage
    offset := 0
    for {
        tasks, err := taskService.GetTasksPaginated(userID, batchSize, offset)
        if err != nil {
            log.Printf("Failed to get tasks batch: %v", err)
            break
        }
        
        if len(tasks) == 0 {
            break // No more tasks
        }
        
        // Process batch
        for _, task := range tasks {
            processTask(task)
        }
        
        offset += batchSize
        
        // Yield to scheduler
        runtime.Gosched()
    }
}

func processTask(task models.Task) {
    // Your task processing logic
    log.Printf("Processing task: %s", task.Title)
}
```

## Examples

### Complete Application Example

Here's a complete example showing how to build a context-aware task manager:

```go
package main

import (
    "database/sql"
    "log"
    "time"
    
    _ "github.com/mattn/go-sqlite3"
    "github.com/bcnelson/hereAndNow/pkg/hereandnow"
    "github.com/bcnelson/hereAndNow/pkg/filters"
    "github.com/bcnelson/hereAndNow/pkg/models"
)

func main() {
    // Initialize database
    db, err := sql.Open("sqlite3", "tasks.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Initialize repositories
    taskRepo := NewSQLiteTaskRepository(db)
    contextRepo := NewSQLiteContextRepository(db)
    dependencyRepo := NewSQLiteTaskDependencyRepository(db)
    taskLocationRepo := NewSQLiteTaskLocationRepository(db)
    auditRepo := NewSQLiteFilterAuditRepository(db)
    locationRepo := NewSQLiteLocationRepository(db)
    
    // Create services
    filterEngine := createFilterEngine(auditRepo)
    taskService := hereandnow.NewTaskService(
        taskRepo, contextRepo, dependencyRepo, taskLocationRepo, filterEngine)
    contextService := hereandnow.NewContextService(contextRepo, locationRepo)
    
    // Demo usage
    runTaskManagerDemo(taskService, contextService)
}

func createFilterEngine(auditRepo filters.FilterAuditRepository) *filters.Engine {
    config := filters.FilterConfig{
        EnableLocationFilter:   true,
        EnableTimeFilter:      true,
        EnableDependencyFilter: true,
        EnablePriorityFilter:  true,
        LocationRadiusMeters:  100,
        MinEnergyLevel:        1,
        TimeBufferMinutes:     5,
    }
    
    engine := filters.NewEngine(config, auditRepo)
    engine.AddRule(&filters.LocationFilter{})
    engine.AddRule(&filters.TimeFilter{})
    engine.AddRule(&filters.DependencyFilter{})
    engine.AddRule(&filters.PriorityFilter{})
    
    return engine
}

func runTaskManagerDemo(taskService *hereandnow.TaskService, contextService *hereandnow.ContextService) {
    userID := "demo-user"
    
    // 1. Create some locations
    homeLocation, _ := models.NewLocation(userID, "Home", "123 Home St", 40.7128, -74.0060, 100)
    officeLocation, _ := models.NewLocation(userID, "Office", "456 Work Ave", 40.7589, -73.9851, 150)
    
    // 2. Create tasks
    log.Println("Creating tasks...")
    
    homeTask, _ := taskService.CreateTask(userID, hereandnow.CreateTaskRequest{
        Title:            "Do laundry",
        Description:      "Wash and fold clothes",
        Priority:         3,
        EstimatedMinutes: &[]int{60}[0],
        LocationIDs:      []string{homeLocation.ID},
    })
    
    officeTask, _ := taskService.CreateTask(userID, hereandnow.CreateTaskRequest{
        Title:            "Attend team meeting",
        Description:      "Weekly sync with development team",
        Priority:         1,
        EstimatedMinutes: &[]int{30}[0],
        LocationIDs:      []string{officeLocation.ID},
    })
    
    // 3. Simulate being at home with limited time
    log.Println("Simulating context: At home, 45 minutes available")
    
    contextService.UpdateContext(userID, hereandnow.UpdateContextRequest{
        CurrentLatitude:  &homeLocation.Latitude,
        CurrentLongitude: &homeLocation.Longitude,
        AvailableMinutes: &[]int{45}[0],
        EnergyLevel:      &[]int{3}[0],
        SocialContext:    &[]string{"alone"}[0],
    })
    
    // 4. Get filtered tasks
    tasks, filterResults, _ := taskService.GetFilteredTasks(userID)
    
    log.Printf("Visible tasks: %d", len(tasks))
    for _, task := range tasks {
        log.Printf("  - %s", task.Title)
    }
    
    // 5. Show filtering explanations
    log.Println("Filter explanations:")
    for _, result := range filterResults {
        status := "VISIBLE"
        if !result.Visible {
            status = "HIDDEN"
        }
        log.Printf("  %s [%s]: %s", result.TaskID, status, result.Reason)
    }
    
    // 6. Explain specific task visibility
    explanation, _ := taskService.ExplainTaskVisibility(homeTask.ID, userID)
    log.Printf("Detailed explanation for '%s':", explanation.TaskTitle)
    for _, filterResult := range explanation.FilterResults {
        log.Printf("  %s: %s", filterResult.FilterName, filterResult.Reason)
    }
}
```

This comprehensive documentation provides everything needed to effectively use the Here and Now Go library for building context-aware task management applications. The library handles the complexity of intelligent filtering while providing transparency and flexibility for different use cases.

---

**Generated:** 2025-09-09  
**Library Version:** 1.0.0  
**Go Version:** 1.21+