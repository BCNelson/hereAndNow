package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskService    TaskService
	contextService ContextService
}

type TaskService interface {
	GetFilteredTasks(userID string, filters TaskFilters) (*TaskListResponse, error)
	CreateTask(task models.Task) (*models.Task, error)
	GetTaskByID(taskID string, userID string) (*models.Task, error)
	UpdateTask(task models.Task) (*models.Task, error)
	DeleteTask(taskID string, userID string) error
	AssignTask(taskID string, assigneeID string, assignedBy string, message string) error
	CompleteTask(taskID string, userID string) (*models.Task, error)
	GetTaskAudit(taskID string, userID string) ([]models.FilterAudit, error)
	CreateTaskFromNaturalLanguage(input string, userID string) (*models.Task, error)
}

type ContextService interface {
	GetCurrentContext(userID string) (*models.Context, error)
	UpdateContext(context models.Context) (*models.Context, error)
}

type TaskFilters struct {
	Status      string
	AssigneeID  string
	ListID      string
	ShowAll     bool
	Limit       int
	Offset      int
}

type TaskListResponse struct {
	Tasks   []models.Task   `json:"tasks"`
	Total   int             `json:"total"`
	Context models.Context  `json:"context"`
}

type TaskCreateRequest struct {
	Title            string    `json:"title" binding:"required"`
	Description      string    `json:"description"`
	ListID           string    `json:"list_id"`
	Priority         int       `json:"priority"`
	EstimatedMinutes *int      `json:"estimated_minutes"`
	DueAt            *time.Time `json:"due_at"`
	LocationIDs      []string  `json:"location_ids"`
	DependencyIDs    []string  `json:"dependency_ids"`
}

type TaskUpdateRequest struct {
	Title            *string    `json:"title"`
	Description      *string    `json:"description"`
	Status           *string    `json:"status"`
	Priority         *int       `json:"priority"`
	EstimatedMinutes *int       `json:"estimated_minutes"`
	DueAt            *time.Time `json:"due_at"`
}

type TaskAssignRequest struct {
	AssigneeID string `json:"assignee_id" binding:"required"`
	Message    string `json:"message"`
}

type NaturalLanguageRequest struct {
	Input     string `json:"input" binding:"required"`
	InputType string `json:"input_type"`
}

func NewTaskHandler(taskService TaskService, contextService ContextService) *TaskHandler {
	return &TaskHandler{
		taskService:    taskService,
		contextService: contextService,
	}
}

// GetTasks handles GET /tasks - get filtered tasks for current context
func (h *TaskHandler) GetTasks(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	// Parse query parameters
	filters := TaskFilters{
		Status:     c.Query("status"),
		AssigneeID: c.Query("assignee_id"),
		ListID:     c.Query("list_id"),
		ShowAll:    c.Query("show_all") == "true",
		Limit:      50, // Default
		Offset:     0,  // Default
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Validate status filter
	if filters.Status != "" {
		validStatuses := []string{"pending", "active", "completed", "cancelled", "blocked"}
		valid := false
		for _, status := range validStatuses {
			if filters.Status == status {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid status filter",
			})
			return
		}
	}

	// Get filtered tasks
	response, err := h.taskService.GetFilteredTasks(userID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get tasks",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreateTask handles POST /tasks
func (h *TaskHandler) CreateTask(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	var req TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Create task model
	task := models.Task{
		Title:       req.Title,
		Description: req.Description,
		CreatorID:   user.ID,
		ListID:      &req.ListID,
		Status:      models.TaskStatusPending,
		Priority:    req.Priority,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.EstimatedMinutes != nil {
		task.EstimatedMinutes = req.EstimatedMinutes
	}

	if req.DueAt != nil {
		task.DueAt = req.DueAt
	}

	// Create task
	createdTask, err := h.taskService.CreateTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to create task",
		})
		return
	}

	c.JSON(http.StatusCreated, createdTask)
}

// GetTask handles GET /tasks/{taskId}
func (h *TaskHandler) GetTask(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Task ID is required",
		})
		return
	}

	task, err := h.taskService.GetTaskByID(taskID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Task not found",
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// UpdateTask handles PATCH /tasks/{taskId}
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Task ID is required",
		})
		return
	}

	// Get existing task
	task, err := h.taskService.GetTaskByID(taskID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Task not found",
		})
		return
	}

	var req TaskUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Apply updates
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = models.TaskStatus(*req.Status)
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.EstimatedMinutes != nil {
		task.EstimatedMinutes = req.EstimatedMinutes
	}
	if req.DueAt != nil {
		task.DueAt = req.DueAt
	}

	task.UpdatedAt = time.Now()

	// Update task
	updatedTask, err := h.taskService.UpdateTask(*task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to update task",
		})
		return
	}

	c.JSON(http.StatusOK, updatedTask)
}

// DeleteTask handles DELETE /tasks/{taskId}
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Task ID is required",
		})
		return
	}

	if err := h.taskService.DeleteTask(taskID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to delete task",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// AssignTask handles POST /tasks/{taskId}/assign
func (h *TaskHandler) AssignTask(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Task ID is required",
		})
		return
	}

	var req TaskAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	if err := h.taskService.AssignTask(taskID, req.AssigneeID, userID, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to assign task",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task assigned successfully"})
}

// CompleteTask handles POST /tasks/{taskId}/complete
func (h *TaskHandler) CompleteTask(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Task ID is required",
		})
		return
	}

	task, err := h.taskService.CompleteTask(taskID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to complete task",
		})
		return
	}

	c.JSON(http.StatusOK, task)
}

// GetTaskAudit handles GET /tasks/{taskId}/audit
func (h *TaskHandler) GetTaskAudit(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Task ID is required",
		})
		return
	}

	audit, err := h.taskService.GetTaskAudit(taskID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get task audit",
		})
		return
	}

	c.JSON(http.StatusOK, audit)
}

// CreateTaskNatural handles POST /tasks/natural
func (h *TaskHandler) CreateTaskNatural(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	var req NaturalLanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	task, err := h.taskService.CreateTaskFromNaturalLanguage(req.Input, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to create task from natural language",
		})
		return
	}

	c.JSON(http.StatusCreated, task)
}