package filters

import (
	"fmt"
	"strings"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

type DependencyFilter struct {
	config           FilterConfig
	dependencyRepo   TaskDependencyRepository
	taskRepo         TaskRepository
}

type TaskDependencyRepository interface {
	GetDependenciesByTaskID(taskID string) ([]models.TaskDependency, error)
	GetDependentsByTaskID(taskID string) ([]models.TaskDependency, error)
}

type TaskRepository interface {
	GetByID(taskID string) (*models.Task, error)
	GetByStatus(userID string, status models.TaskStatus) ([]models.Task, error)
}

func NewDependencyFilter(config FilterConfig, dependencyRepo TaskDependencyRepository, taskRepo TaskRepository) *DependencyFilter {
	return &DependencyFilter{
		config:         config,
		dependencyRepo: dependencyRepo,
		taskRepo:       taskRepo,
	}
}

func (f *DependencyFilter) Name() string {
	return "dependency"
}

func (f *DependencyFilter) Priority() int {
	return 110
}

func (f *DependencyFilter) Apply(ctx models.Context, task models.Task) (visible bool, reason string) {
	if !f.config.EnableDependencyFilter {
		return true, "dependency filtering disabled"
	}

	dependencies, err := f.dependencyRepo.GetDependenciesByTaskID(task.ID)
	if err != nil {
		return false, fmt.Sprintf("error checking dependencies: %v", err)
	}

	if len(dependencies) == 0 {
		return true, "no dependencies"
	}

	hasCircularDep, circularReason := f.checkCircularDependencies(task.ID, make(map[string]bool))
	if hasCircularDep {
		return false, fmt.Sprintf("circular dependency detected: %s", circularReason)
	}

	unmetDependencies := []string{}
	for _, dep := range dependencies {
		dependentTask, err := f.taskRepo.GetByID(dep.DependsOnTaskID)
		if err != nil {
			unmetDependencies = append(unmetDependencies, fmt.Sprintf("unknown task %s", dep.DependsOnTaskID))
			continue
		}

		if !f.isDependencyMet(dep, *dependentTask) {
			unmetDependencies = append(unmetDependencies, f.formatUnmetDependency(dep, *dependentTask))
		}
	}

	if len(unmetDependencies) > 0 {
		return false, fmt.Sprintf("unmet dependencies: %s", strings.Join(unmetDependencies, ", "))
	}

	return true, fmt.Sprintf("all %d dependencies met", len(dependencies))
}

func (f *DependencyFilter) isDependencyMet(dep models.TaskDependency, dependentTask models.Task) bool {
	switch dep.DependencyType {
	case models.DependencyTypeBlocking:
		return dependentTask.Status == models.TaskStatusCompleted
	case models.DependencyTypeRelated:
		return dependentTask.Status == models.TaskStatusActive || 
			   dependentTask.Status == models.TaskStatusCompleted
	case models.DependencyTypeScheduled:
		return dependentTask.Status == models.TaskStatusCompleted
	default:
		return false
	}
}

func (f *DependencyFilter) formatUnmetDependency(dep models.TaskDependency, dependentTask models.Task) string {
	switch dep.DependencyType {
	case models.DependencyTypeBlocking:
		return fmt.Sprintf("'%s' must be completed first", dependentTask.Title)
	case models.DependencyTypeRelated:
		return fmt.Sprintf("'%s' must be started first", dependentTask.Title)
	case models.DependencyTypeScheduled:
		return fmt.Sprintf("'%s' must be completed according to schedule", dependentTask.Title)
	default:
		return fmt.Sprintf("'%s' has unknown dependency type", dependentTask.Title)
	}
}

func (f *DependencyFilter) checkCircularDependencies(taskID string, visited map[string]bool) (bool, string) {
	if visited[taskID] {
		return true, fmt.Sprintf("circular dependency involving task %s", taskID)
	}

	visited[taskID] = true

	dependencies, err := f.dependencyRepo.GetDependenciesByTaskID(taskID)
	if err != nil {
		return false, ""
	}

	for _, dep := range dependencies {
		hasCircular, reason := f.checkCircularDependencies(dep.DependsOnTaskID, visited)
		if hasCircular {
			return true, reason
		}
	}

	delete(visited, taskID)
	return false, ""
}

func (f *DependencyFilter) GetDependencyChain(taskID string) ([]models.Task, error) {
	chain := []models.Task{}
	visited := make(map[string]bool)
	
	err := f.buildDependencyChain(taskID, &chain, visited)
	if err != nil {
		return nil, err
	}

	return chain, nil
}

func (f *DependencyFilter) buildDependencyChain(taskID string, chain *[]models.Task, visited map[string]bool) error {
	if visited[taskID] {
		return fmt.Errorf("circular dependency detected for task %s", taskID)
	}

	visited[taskID] = true

	task, err := f.taskRepo.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	dependencies, err := f.dependencyRepo.GetDependenciesByTaskID(taskID)
	if err != nil {
		return fmt.Errorf("error getting dependencies for task %s: %v", taskID, err)
	}

	for _, dep := range dependencies {
		err := f.buildDependencyChain(dep.DependsOnTaskID, chain, visited)
		if err != nil {
			return err
		}
	}

	*chain = append(*chain, *task)
	delete(visited, taskID)

	return nil
}

func (f *DependencyFilter) CanStartTask(taskID string) (bool, []string, error) {
	dependencies, err := f.dependencyRepo.GetDependenciesByTaskID(taskID)
	if err != nil {
		return false, nil, fmt.Errorf("error checking dependencies: %v", err)
	}

	if len(dependencies) == 0 {
		return true, []string{}, nil
	}

	blockers := []string{}
	for _, dep := range dependencies {
		dependentTask, err := f.taskRepo.GetByID(dep.DependsOnTaskID)
		if err != nil {
			blockers = append(blockers, fmt.Sprintf("unknown task %s", dep.DependsOnTaskID))
			continue
		}

		if dep.DependencyType == models.DependencyTypeBlocking && 
		   dependentTask.Status != models.TaskStatusCompleted {
			blockers = append(blockers, dependentTask.Title)
		}
	}

	return len(blockers) == 0, blockers, nil
}