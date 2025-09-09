package hereandnow

import (
	"fmt"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo         TaskRepository
	contextRepo      ContextRepository
	dependencyRepo   TaskDependencyRepository
	taskLocationRepo TaskLocationRepository
	filterEngine     filters.FilterEngine
}

type TaskRepository interface {
	Create(task models.Task) error
	GetByID(taskID string) (*models.Task, error)
	GetByUserID(userID string) ([]models.Task, error)
	GetByStatus(userID string, status models.TaskStatus) ([]models.Task, error)
	Update(task models.Task) error
	Delete(taskID string) error
	GetByListID(listID string) ([]models.Task, error)
	Search(userID string, query string) ([]models.Task, error)
}

type ContextRepository interface {
	GetLatestByUserID(userID string) (*models.Context, error)
	Create(context models.Context) error
}

type TaskDependencyRepository interface {
	Create(dependency models.TaskDependency) error
	GetDependenciesByTaskID(taskID string) ([]models.TaskDependency, error)
	GetDependentsByTaskID(taskID string) ([]models.TaskDependency, error)
	Delete(dependentTaskID, dependsOnTaskID string) error
}

type TaskLocationRepository interface {
	Create(taskLocation models.TaskLocation) error
	GetLocationsByTaskID(taskID string) ([]models.Location, error)
	Delete(taskID, locationID string) error
}

func NewTaskService(
	taskRepo TaskRepository,
	contextRepo ContextRepository,
	dependencyRepo TaskDependencyRepository,
	taskLocationRepo TaskLocationRepository,
	filterEngine filters.FilterEngine,
) *TaskService {
	return &TaskService{
		taskRepo:         taskRepo,
		contextRepo:      contextRepo,
		dependencyRepo:   dependencyRepo,
		taskLocationRepo: taskLocationRepo,
		filterEngine:     filterEngine,
	}
}

func (s *TaskService) CreateTask(userID string, req CreateTaskRequest) (*models.Task, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid task request: %w", err)
	}

	task := models.Task{
		ID:               uuid.New().String(),
		Title:            req.Title,
		Description:      req.Description,
		CreatorID:        userID,
		AssigneeID:       req.AssigneeID,
		ListID:           req.ListID,
		Status:           models.TaskStatusPending,
		Priority:         req.Priority,
		EstimatedMinutes: req.EstimatedMinutes,
		DueAt:            req.DueAt,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         req.Metadata,
		RecurrenceRule:   req.RecurrenceRule,
		ParentTaskID:     req.ParentTaskID,
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	if err := s.addTaskLocations(task.ID, req.LocationIDs); err != nil {
		return nil, fmt.Errorf("failed to add task locations: %w", err)
	}

	if err := s.addTaskDependencies(task.ID, req.Dependencies); err != nil {
		return nil, fmt.Errorf("failed to add task dependencies: %w", err)
	}

	return &task, nil
}

func (s *TaskService) GetFilteredTasks(userID string) ([]models.Task, []filters.FilterResult, error) {
	allTasks, err := s.taskRepo.GetByUserID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user tasks: %w", err)
	}

	context, err := s.contextRepo.GetLatestByUserID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user context: %w", err)
	}

	filteredTasks, filterResults := s.filterEngine.FilterTasks(*context, allTasks)
	
	return filteredTasks, filterResults, nil
}

func (s *TaskService) GetTask(taskID string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	return task, nil
}

func (s *TaskService) UpdateTask(taskID string, req UpdateTaskRequest) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
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
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.AssigneeID != nil {
		task.AssigneeID = req.AssigneeID
	}

	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(*task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

func (s *TaskService) CompleteTask(taskID string, userID string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	if task.Status == models.TaskStatusCompleted {
		return task, nil
	}

	completedAt := time.Now()
	task.Status = models.TaskStatusCompleted
	task.CompletedAt = &completedAt
	task.UpdatedAt = completedAt

	if err := s.taskRepo.Update(*task); err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}

	return task, nil
}

func (s *TaskService) AssignTask(taskID string, assigneeID string, assignerID string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	task.AssigneeID = &assigneeID
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.Update(*task); err != nil {
		return nil, fmt.Errorf("failed to assign task: %w", err)
	}

	return task, nil
}

func (s *TaskService) DeleteTask(taskID string) error {
	dependencies, err := s.dependencyRepo.GetDependentsByTaskID(taskID)
	if err != nil {
		return fmt.Errorf("failed to check task dependencies: %w", err)
	}

	if len(dependencies) > 0 {
		return fmt.Errorf("cannot delete task with %d dependent tasks", len(dependencies))
	}

	if err := s.taskRepo.Delete(taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

func (s *TaskService) SearchTasks(userID string, query string) ([]models.Task, error) {
	tasks, err := s.taskRepo.Search(userID, query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return tasks, nil
}

func (s *TaskService) GetTasksByStatus(userID string, status models.TaskStatus) ([]models.Task, error) {
	tasks, err := s.taskRepo.GetByStatus(userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by status: %w", err)
	}

	return tasks, nil
}

func (s *TaskService) ExplainTaskVisibility(taskID string, userID string) (*filters.TaskVisibilityExplanation, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	context, err := s.contextRepo.GetLatestByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	explanation := s.filterEngine.ExplainTaskVisibility(*context, *task)
	return &explanation, nil
}

func (s *TaskService) GetAuditLog(taskID string, userID string) ([]filters.FilterResult, error) {
	context, err := s.contextRepo.GetLatestByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	auditLog, err := s.filterEngine.GetAuditLog(taskID, *context)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return auditLog, nil
}

func (s *TaskService) addTaskLocations(taskID string, locationIDs []string) error {
	for _, locationID := range locationIDs {
		taskLocation := models.TaskLocation{
			ID:         uuid.New().String(),
			TaskID:     taskID,
			LocationID: locationID,
			CreatedAt:  time.Now(),
		}
		
		if err := s.taskLocationRepo.Create(taskLocation); err != nil {
			return fmt.Errorf("failed to add location %s: %w", locationID, err)
		}
	}
	return nil
}

func (s *TaskService) addTaskDependencies(taskID string, dependencies []TaskDependencyRequest) error {
	for _, dep := range dependencies {
		taskDep := models.TaskDependency{
			ID:               uuid.New().String(),
			TaskID:           taskID,
			DependsOnTaskID:  dep.DependsOnTaskID,
			DependencyType:   dep.DependencyType,
			CreatedAt:        time.Now(),
		}
		
		if err := s.dependencyRepo.Create(taskDep); err != nil {
			return fmt.Errorf("failed to add dependency %s: %w", dep.DependsOnTaskID, err)
		}
	}
	return nil
}

type CreateTaskRequest struct {
	Title            string                    `json:"title"`
	Description      string                    `json:"description"`
	AssigneeID       *string                   `json:"assignee_id"`
	ListID           *string                   `json:"list_id"`
	Priority         int                       `json:"priority"`
	EstimatedMinutes *int                      `json:"estimated_minutes"`
	DueAt            *time.Time                `json:"due_at"`
	Metadata         []byte                    `json:"metadata"`
	RecurrenceRule   *string                   `json:"recurrence_rule"`
	ParentTaskID     *string                   `json:"parent_task_id"`
	LocationIDs      []string                  `json:"location_ids"`
	Dependencies     []TaskDependencyRequest   `json:"dependencies"`
}

type UpdateTaskRequest struct {
	Title            *string            `json:"title"`
	Description      *string            `json:"description"`
	Priority         *int               `json:"priority"`
	EstimatedMinutes *int               `json:"estimated_minutes"`
	DueAt            *time.Time         `json:"due_at"`
	Status           *models.TaskStatus `json:"status"`
	AssigneeID       *string            `json:"assignee_id"`
}

type TaskDependencyRequest struct {
	DependsOnTaskID string                     `json:"depends_on_task_id"`
	DependencyType  models.DependencyType      `json:"dependency_type"`
}

func (r CreateTaskRequest) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if r.Priority < 1 || r.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}
	if r.EstimatedMinutes != nil && *r.EstimatedMinutes < 0 {
		return fmt.Errorf("estimated minutes cannot be negative")
	}
	return nil
}