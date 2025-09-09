package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bcnelson/hereAndNow/internal/auth"
	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/hereandnow"
	"github.com/bcnelson/hereAndNow/pkg/models"
)

func handleTaskCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: task requires a subcommand")
		fmt.Println("Run 'hereandnow task --help' for usage")
		os.Exit(1)
	}

	if args[0] == "--help" || args[0] == "-h" {
		fmt.Printf(`Task Management Commands

USAGE:
    hereandnow task <SUBCOMMAND> [OPTIONS]

SUBCOMMANDS:
    add <title>         Create a new task
    list                List tasks (filtered by context)
    show <task-id>      Show task details
    update <task-id>    Update task information
    complete <task-id>  Mark task as complete
    delete <task-id>    Delete a task
    assign <task-id>    Assign task to user
    audit <task-id>     Show filtering audit trail
    search <query>      Search tasks by text

OPTIONS:
    --all               Show all tasks (override context filtering)
    --status <status>   Filter by status (pending|in_progress|completed|blocked)
    --priority <1-10>   Set task priority
    --estimate <mins>   Set estimated minutes
    --due <date>        Set due date (YYYY-MM-DD or YYYY-MM-DD HH:MM)
    --location <name>   Assign task to location
    --assignee <user>   Assign to user
    --depends-on <id>   Add task dependency
    --list <name>       Add to task list
    --help, -h          Show this help

EXAMPLES:
    # Add a simple task
    hereandnow task add "Buy milk"

    # Add task with location and time estimate
    hereandnow task add "Review reports" --location Office --estimate 60

    # Add task with dependency
    hereandnow task add "Send report" --depends-on draft-123 --priority 8

    # List current tasks (context filtered)
    hereandnow task list

    # List ALL tasks
    hereandnow task list --all

    # List only pending tasks
    hereandnow task list --status pending

    # Complete a task
    hereandnow task complete abc123

    # Show task audit trail
    hereandnow task audit abc123

    # Search tasks
    hereandnow task search "grocery"
`)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "add":
		executeTaskAdd(subArgs)
	case "list":
		executeTaskList(subArgs)
	case "show":
		executeTaskShow(subArgs)
	case "update":
		executeTaskUpdate(subArgs)
	case "complete":
		executeTaskComplete(subArgs)
	case "delete":
		executeTaskDelete(subArgs)
	case "assign":
		executeTaskAssign(subArgs)
	case "audit":
		executeTaskAudit(subArgs)
	case "search":
		executeTaskSearch(subArgs)
	default:
		fmt.Printf("Unknown task subcommand: %s\n", subcommand)
		fmt.Println("Run 'hereandnow task --help' for usage")
		os.Exit(1)
	}
}

func executeTaskAdd(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: task add requires title\n")
		fmt.Println("Usage: hereandnow task add <title> [OPTIONS]")
		os.Exit(1)
	}

	title := args[0]
	priority := 3
	estimate := (*int)(nil)
	dueDate := (*time.Time)(nil)
	location := ""
	assignee := ""
	dependsOn := ""
	listName := ""
	description := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--priority":
			if i+1 < len(args) {
				if p, err := strconv.Atoi(args[i+1]); err == nil && p >= 1 && p <= 10 {
					priority = p
					i++
				}
			}
		case "--estimate":
			if i+1 < len(args) {
				if e, err := strconv.Atoi(args[i+1]); err == nil {
					estimate = &e
					i++
				}
			}
		case "--due":
			if i+1 < len(args) {
				if due, err := parseDateTime(args[i+1]); err == nil {
					dueDate = &due
					i++
				}
			}
		case "--location":
			if i+1 < len(args) {
				location = args[i+1]
				i++
			}
		case "--assignee":
			if i+1 < len(args) {
				assignee = args[i+1]
				i++
			}
		case "--depends-on":
			if i+1 < len(args) {
				dependsOn = args[i+1]
				i++
			}
		case "--list":
			if i+1 < len(args) {
				listName = args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			}
		}
	}

	// Get current user (placeholder - would need session management)
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user. Please create a user first.\n")
		os.Exit(1)
	}

	// Initialize services
	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	// Parse dependencies
	var dependencies []hereandnow.TaskDependencyRequest
	if dependsOn != "" {
		dependencies = append(dependencies, hereandnow.TaskDependencyRequest{
			DependsOnTaskID: dependsOn,
			DependencyType:  models.DependencyTypeBlocks,
		})
	}

	// Find location ID if location name provided
	var locationIDs []string
	if location != "" {
		locationID, err := findLocationByName(location, userID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Location '%s' not found, task created without location\n", location)
		} else {
			locationIDs = append(locationIDs, locationID)
		}
	}

	// Find assignee ID if assignee provided
	var assigneeID *string
	if assignee != "" {
		aID, err := findUserByUsername(assignee)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: User '%s' not found, task not assigned\n", assignee)
		} else {
			assigneeID = &aID
		}
	}

	// Create task
	req := hereandnow.CreateTaskRequest{
		Title:            title,
		Description:      description,
		AssigneeID:       assigneeID,
		Priority:         priority,
		EstimatedMinutes: estimate,
		DueAt:            dueDate,
		LocationIDs:      locationIDs,
		Dependencies:     dependencies,
	}

	task, err := taskService.CreateTask(userID, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating task: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("Task created successfully: %s (ID: %s)", task.Title, task.ID))
}

func executeTaskList(args []string) {
	showAll := false
	status := ""

	for i, arg := range args {
		switch arg {
		case "--all":
			showAll = true
		case "--status":
			if i+1 < len(args) {
				status = args[i+1]
			}
		}
	}

	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	var tasks []models.Task

	if status != "" {
		// Filter by status
		taskStatus := models.TaskStatus(status)
		tasks, err = taskService.GetTasksByStatus(userID, taskStatus)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error retrieving tasks: %v\n", err)
			os.Exit(1)
		}
	} else if showAll {
		// Show all tasks
		config, _ := LoadConfig()
		db, _ := InitDatabase(config.Database.Path)
		defer db.Close()
		taskRepo := storage.NewTaskRepository(db)
		tasks, err = taskRepo.GetByUserID(userID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error retrieving tasks: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Show context-filtered tasks
		tasks, _, err = taskService.GetFilteredTasks(userID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error retrieving filtered tasks: %v\n", err)
			os.Exit(1)
		}
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, tasks)
}

func executeTaskShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: task show requires task ID\n")
		fmt.Println("Usage: hereandnow task show <task-id>")
		os.Exit(1)
	}

	taskID := args[0]

	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	task, err := taskService.GetTask(taskID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Task not found\n")
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, *task)
}

func executeTaskComplete(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: task complete requires task ID\n")
		fmt.Println("Usage: hereandnow task complete <task-id>")
		os.Exit(1)
	}

	taskID := args[0]
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	task, err := taskService.CompleteTask(taskID, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error completing task: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("Task completed: %s", task.Title))
}

func executeTaskUpdate(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: task update requires task ID\n")
		fmt.Println("Usage: hereandnow task update <task-id> [OPTIONS]")
		os.Exit(1)
	}

	taskID := args[0]
	var title, description *string
	var priority, estimate *int
	var dueDate *time.Time
	var status *models.TaskStatus

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--title":
			if i+1 < len(args) {
				title = &args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(args) {
				description = &args[i+1]
				i++
			}
		case "--priority":
			if i+1 < len(args) {
				if p, err := strconv.Atoi(args[i+1]); err == nil && p >= 1 && p <= 10 {
					priority = &p
					i++
				}
			}
		case "--estimate":
			if i+1 < len(args) {
				if e, err := strconv.Atoi(args[i+1]); err == nil {
					estimate = &e
					i++
				}
			}
		case "--due":
			if i+1 < len(args) {
				if due, err := parseDateTime(args[i+1]); err == nil {
					dueDate = &due
					i++
				}
			}
		case "--status":
			if i+1 < len(args) {
				s := models.TaskStatus(args[i+1])
				status = &s
				i++
			}
		}
	}

	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	req := hereandnow.UpdateTaskRequest{
		Title:            title,
		Description:      description,
		Priority:         priority,
		EstimatedMinutes: estimate,
		DueAt:            dueDate,
		Status:           status,
	}

	task, err := taskService.UpdateTask(taskID, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating task: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("Task updated: %s", task.Title))
}

func executeTaskDelete(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: task delete requires task ID\n")
		fmt.Println("Usage: hereandnow task delete <task-id>")
		os.Exit(1)
	}

	taskID := args[0]

	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	if err := taskService.DeleteTask(taskID); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting task: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, "Task deleted successfully")
}

func executeTaskAssign(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: task assign requires task ID and username\n")
		fmt.Println("Usage: hereandnow task assign <task-id> <username>")
		os.Exit(1)
	}

	taskID := args[0]
	username := args[1]
	
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	assigneeID, err := findUserByUsername(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: User '%s' not found\n", username)
		os.Exit(1)
	}

	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	task, err := taskService.AssignTask(taskID, assigneeID, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error assigning task: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("Task assigned to %s: %s", username, task.Title))
}

func executeTaskAudit(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: task audit requires task ID\n")
		fmt.Println("Usage: hereandnow task audit <task-id>")
		os.Exit(1)
	}

	taskID := args[0]
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	explanation, err := taskService.ExplainTaskVisibility(taskID, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting task audit: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, *explanation)
}

func executeTaskSearch(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: task search requires query\n")
		fmt.Println("Usage: hereandnow task search <query>")
		os.Exit(1)
	}

	query := strings.Join(args, " ")
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	taskService, err := initTaskService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing task service: %v\n", err)
		os.Exit(1)
	}

	tasks, err := taskService.SearchTasks(userID, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching tasks: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, tasks)
}

// Helper functions

func initTaskService() (*hereandnow.TaskService, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		return nil, err
	}

	taskRepo := storage.NewTaskRepository(db)
	contextRepo := storage.NewContextRepository(db)
	dependencyRepo := storage.NewTaskDependencyRepository(db)
	taskLocationRepo := storage.NewTaskLocationRepository(db)
	filterEngine := filters.NewFilterEngine()

	return hereandnow.NewTaskService(taskRepo, contextRepo, dependencyRepo, taskLocationRepo, *filterEngine), nil
}

func getCurrentUserID() string {
	// In a real CLI application, this would check for a session file or config
	// For now, return the first user in the database
	config, err := LoadConfig()
	if err != nil {
		return ""
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		return ""
	}
	defer db.Close()

	userRepo := storage.NewUserRepository(db)
	users, err := userRepo.GetAll()
	if err != nil || len(users) == 0 {
		return ""
	}

	return users[0].ID
}

func findLocationByName(name, userID string) (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		return "", err
	}
	defer db.Close()

	locationRepo := storage.NewLocationRepository(db)
	locations, err := locationRepo.GetByUserID(userID)
	if err != nil {
		return "", err
	}

	for _, loc := range locations {
		if strings.EqualFold(loc.Name, name) {
			return loc.ID, nil
		}
	}

	return "", fmt.Errorf("location not found: %s", name)
}

func findUserByUsername(username string) (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		return "", err
	}
	defer db.Close()

	userRepo := storage.NewUserRepository(db)
	user, err := userRepo.GetByUsername(username)
	if err != nil {
		return "", err
	}

	return user.ID, nil
}

func parseDateTime(dateStr string) (time.Time, error) {
	// Try various date/time formats
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"01/02/2006",
		"01/02/2006 15:04",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}