package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

// TaskRepository handles task data persistence
type TaskRepository struct {
	db *DB
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// TaskSearchOptions defines options for searching tasks
type TaskSearchOptions struct {
	UserID           string              // Filter by user (creator or assignee)
	ListID           *string             // Filter by list
	Status           *models.TaskStatus  // Filter by status
	AssigneeID       *string             // Filter by assignee
	CreatorID        *string             // Filter by creator
	DueBefore        *time.Time          // Filter by due date
	DueAfter         *time.Time          // Filter by due date
	CompletedAfter   *time.Time          // Filter by completion date
	Priority         *int                // Filter by priority
	ParentTaskID     *string             // Filter by parent task
	HasDueDate       *bool               // Filter tasks with/without due dates
	Query            string              // Full-text search query
	Limit            int                 // Pagination limit
	Offset           int                 // Pagination offset
	OrderBy          string              // Order by field (created_at, updated_at, due_at, priority, title)
	OrderDirection   string              // Order direction (ASC, DESC)
}

// Create creates a new task in the database
func (r *TaskRepository) Create(task *models.Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	// Validate the task before inserting
	if err := task.Validate(); err != nil {
		return fmt.Errorf("task validation failed: %w", err)
	}

	query := `
		INSERT INTO tasks (
			id, title, description, creator_id, assignee_id, list_id,
			status, priority, estimated_minutes, due_at, completed_at,
			created_at, updated_at, metadata, recurrence_rule, parent_task_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		task.ID,
		task.Title,
		task.Description,
		task.CreatorID,
		task.AssigneeID,
		task.ListID,
		string(task.Status),
		task.Priority,
		task.EstimatedMinutes,
		task.DueAt,
		task.CompletedAt,
		task.CreatedAt,
		task.UpdatedAt,
		task.Metadata,
		task.RecurrenceRule,
		task.ParentTaskID,
	)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// GetByID retrieves a task by its ID
func (r *TaskRepository) GetByID(id string) (*models.Task, error) {
	if id == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	query := `
		SELECT id, title, description, creator_id, assignee_id, list_id,
		       status, priority, estimated_minutes, due_at, completed_at,
		       created_at, updated_at, metadata, recurrence_rule, parent_task_id
		FROM tasks 
		WHERE id = ?`

	task := &models.Task{}
	var statusStr string

	err := r.db.QueryRow(query, id).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.CreatorID,
		&task.AssigneeID,
		&task.ListID,
		&statusStr,
		&task.Priority,
		&task.EstimatedMinutes,
		&task.DueAt,
		&task.CompletedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.Metadata,
		&task.RecurrenceRule,
		&task.ParentTaskID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task by ID: %w", err)
	}

	task.Status = models.TaskStatus(statusStr)
	return task, nil
}

// Update updates an existing task
func (r *TaskRepository) Update(task *models.Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	// Validate the task before updating
	if err := task.Validate(); err != nil {
		return fmt.Errorf("task validation failed: %w", err)
	}

	// Update the timestamp
	task.UpdatedAt = time.Now()

	query := `
		UPDATE tasks 
		SET title = ?, description = ?, assignee_id = ?, list_id = ?,
		    status = ?, priority = ?, estimated_minutes = ?, due_at = ?, 
		    completed_at = ?, updated_at = ?, metadata = ?, recurrence_rule = ?,
		    parent_task_id = ?
		WHERE id = ?`

	result, err := r.db.Exec(query,
		task.Title,
		task.Description,
		task.AssigneeID,
		task.ListID,
		string(task.Status),
		task.Priority,
		task.EstimatedMinutes,
		task.DueAt,
		task.CompletedAt,
		task.UpdatedAt,
		task.Metadata,
		task.RecurrenceRule,
		task.ParentTaskID,
		task.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// Delete deletes a task from the database
func (r *TaskRepository) Delete(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	// Check if task has dependencies (other tasks depend on this one)
	var dependentCount int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM task_dependencies WHERE depends_on_task_id = ?
	`, taskID).Scan(&dependentCount)
	
	if err != nil {
		return fmt.Errorf("failed to check task dependencies: %w", err)
	}

	if dependentCount > 0 {
		return fmt.Errorf("cannot delete task: %d tasks depend on this task", dependentCount)
	}

	// Use transaction to delete task and its relationships
	tx, err := r.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete task dependencies
	_, err = tx.Exec(`DELETE FROM task_dependencies WHERE task_id = ?`, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task dependencies: %w", err)
	}

	// Delete task locations
	_, err = tx.Exec(`DELETE FROM task_locations WHERE task_id = ?`, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task locations: %w", err)
	}

	// Delete task assignments
	_, err = tx.Exec(`DELETE FROM task_assignments WHERE task_id = ?`, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task assignments: %w", err)
	}

	// Delete the task itself
	result, err := tx.Exec(`DELETE FROM tasks WHERE id = ?`, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return tx.Commit()
}

// Search searches tasks with various filters and full-text search
func (r *TaskRepository) Search(options TaskSearchOptions) ([]*models.Task, error) {
	var conditions []string
	var args []interface{}

	// Build base query
	baseQuery := `
		SELECT t.id, t.title, t.description, t.creator_id, t.assignee_id, t.list_id,
		       t.status, t.priority, t.estimated_minutes, t.due_at, t.completed_at,
		       t.created_at, t.updated_at, t.metadata, t.recurrence_rule, t.parent_task_id
	`

	var fromClause string
	if options.Query != "" {
		// Use full-text search
		fromClause = `
			FROM tasks t
			JOIN tasks_fts fts ON t.rowid = fts.rowid
		`
		conditions = append(conditions, "tasks_fts MATCH ?")
		args = append(args, options.Query)
	} else {
		fromClause = "FROM tasks t"
	}

	// Add user filter (tasks where user is creator or assignee)
	if options.UserID != "" {
		conditions = append(conditions, "(t.creator_id = ? OR t.assignee_id = ?)")
		args = append(args, options.UserID, options.UserID)
	}

	// Add specific creator filter
	if options.CreatorID != nil {
		conditions = append(conditions, "t.creator_id = ?")
		args = append(args, *options.CreatorID)
	}

	// Add specific assignee filter
	if options.AssigneeID != nil {
		conditions = append(conditions, "t.assignee_id = ?")
		args = append(args, *options.AssigneeID)
	}

	// Add list filter
	if options.ListID != nil {
		conditions = append(conditions, "t.list_id = ?")
		args = append(args, *options.ListID)
	}

	// Add status filter
	if options.Status != nil {
		conditions = append(conditions, "t.status = ?")
		args = append(args, string(*options.Status))
	}

	// Add priority filter
	if options.Priority != nil {
		conditions = append(conditions, "t.priority = ?")
		args = append(args, *options.Priority)
	}

	// Add parent task filter
	if options.ParentTaskID != nil {
		conditions = append(conditions, "t.parent_task_id = ?")
		args = append(args, *options.ParentTaskID)
	}

	// Add due date filters
	if options.DueBefore != nil {
		conditions = append(conditions, "t.due_at < ?")
		args = append(args, *options.DueBefore)
	}
	if options.DueAfter != nil {
		conditions = append(conditions, "t.due_at > ?")
		args = append(args, *options.DueAfter)
	}

	// Add completion date filter
	if options.CompletedAfter != nil {
		conditions = append(conditions, "t.completed_at > ?")
		args = append(args, *options.CompletedAfter)
	}

	// Add has due date filter
	if options.HasDueDate != nil {
		if *options.HasDueDate {
			conditions = append(conditions, "t.due_at IS NOT NULL")
		} else {
			conditions = append(conditions, "t.due_at IS NULL")
		}
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderClause := "ORDER BY t.created_at DESC" // Default ordering
	if options.OrderBy != "" {
		direction := "DESC"
		if options.OrderDirection == "ASC" {
			direction = "ASC"
		}
		
		// Validate order by field
		validOrderFields := map[string]bool{
			"created_at": true, "updated_at": true, "due_at": true,
			"priority": true, "title": true, "status": true,
		}
		if validOrderFields[options.OrderBy] {
			orderClause = fmt.Sprintf("ORDER BY t.%s %s", options.OrderBy, direction)
		}
	}

	// Build LIMIT clause
	limitClause := ""
	if options.Limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", options.Limit)
		if options.Offset > 0 {
			limitClause += fmt.Sprintf(" OFFSET %d", options.Offset)
		}
	}

	// Combine query parts
	query := baseQuery + fromClause + " " + whereClause + " " + orderClause + " " + limitClause

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		var statusStr string

		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.CreatorID,
			&task.AssigneeID,
			&task.ListID,
			&statusStr,
			&task.Priority,
			&task.EstimatedMinutes,
			&task.DueAt,
			&task.CompletedAt,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.Metadata,
			&task.RecurrenceRule,
			&task.ParentTaskID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}

		task.Status = models.TaskStatus(statusStr)
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task rows: %w", err)
	}

	return tasks, nil
}

// GetByUser returns all tasks for a user (as creator or assignee)
func (r *TaskRepository) GetByUser(userID string, limit, offset int) ([]*models.Task, error) {
	options := TaskSearchOptions{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	}
	return r.Search(options)
}

// GetByList returns all tasks in a specific list
func (r *TaskRepository) GetByList(listID string, limit, offset int) ([]*models.Task, error) {
	options := TaskSearchOptions{
		ListID: &listID,
		Limit:  limit,
		Offset: offset,
	}
	return r.Search(options)
}

// GetPendingTasks returns all pending tasks for a user
func (r *TaskRepository) GetPendingTasks(userID string, limit, offset int) ([]*models.Task, error) {
	status := models.TaskStatusPending
	options := TaskSearchOptions{
		UserID: userID,
		Status: &status,
		Limit:  limit,
		Offset: offset,
	}
	return r.Search(options)
}

// GetOverdueTasks returns overdue tasks for a user
func (r *TaskRepository) GetOverdueTasks(userID string, limit, offset int) ([]*models.Task, error) {
	now := time.Now()
	status := models.TaskStatusPending
	options := TaskSearchOptions{
		UserID:    userID,
		Status:    &status,
		DueBefore: &now,
		Limit:     limit,
		Offset:    offset,
		OrderBy:   "due_at",
		OrderDirection: "ASC",
	}
	return r.Search(options)
}

// GetSubtasks returns all subtasks for a parent task
func (r *TaskRepository) GetSubtasks(parentTaskID string) ([]*models.Task, error) {
	options := TaskSearchOptions{
		ParentTaskID: &parentTaskID,
		OrderBy:      "created_at",
		OrderDirection: "ASC",
	}
	return r.Search(options)
}

// FullTextSearch performs a full-text search on task titles and descriptions
func (r *TaskRepository) FullTextSearch(userID, query string, limit, offset int) ([]*models.Task, error) {
	options := TaskSearchOptions{
		UserID: userID,
		Query:  query,
		Limit:  limit,
		Offset: offset,
	}
	return r.Search(options)
}

// Count returns the total number of tasks matching the search options
func (r *TaskRepository) Count(options TaskSearchOptions) (int, error) {
	var conditions []string
	var args []interface{}

	// Build query conditions (similar to Search method)
	var fromClause string
	if options.Query != "" {
		fromClause = `
			FROM tasks t
			JOIN tasks_fts fts ON t.rowid = fts.rowid
		`
		conditions = append(conditions, "tasks_fts MATCH ?")
		args = append(args, options.Query)
	} else {
		fromClause = "FROM tasks t"
	}

	if options.UserID != "" {
		conditions = append(conditions, "(t.creator_id = ? OR t.assignee_id = ?)")
		args = append(args, options.UserID, options.UserID)
	}

	if options.CreatorID != nil {
		conditions = append(conditions, "t.creator_id = ?")
		args = append(args, *options.CreatorID)
	}

	if options.AssigneeID != nil {
		conditions = append(conditions, "t.assignee_id = ?")
		args = append(args, *options.AssigneeID)
	}

	if options.ListID != nil {
		conditions = append(conditions, "t.list_id = ?")
		args = append(args, *options.ListID)
	}

	if options.Status != nil {
		conditions = append(conditions, "t.status = ?")
		args = append(args, string(*options.Status))
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := "SELECT COUNT(*) " + fromClause + " " + whereClause

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return count, nil
}

// UpdateStatus updates a task's status with timestamp tracking
func (r *TaskRepository) UpdateStatus(taskID string, status models.TaskStatus) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	var completedAt *time.Time
	if status == models.TaskStatusCompleted {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE tasks 
		SET status = ?, completed_at = ?, updated_at = ?
		WHERE id = ?`

	result, err := r.db.Exec(query, string(status), completedAt, time.Now(), taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// UpdateMetadata updates a task's metadata
func (r *TaskRepository) UpdateMetadata(taskID string, metadata map[string]interface{}) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `UPDATE tasks SET metadata = ?, updated_at = ? WHERE id = ?`
	_, err = r.db.Exec(query, metadataJSON, time.Now(), taskID)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// Exists checks if a task exists by ID
func (r *TaskRepository) Exists(taskID string) (bool, error) {
	if taskID == "" {
		return false, fmt.Errorf("task ID cannot be empty")
	}

	var count int
	query := `SELECT COUNT(*) FROM tasks WHERE id = ?`
	
	err := r.db.QueryRow(query, taskID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check task existence: %w", err)
	}

	return count > 0, nil
}