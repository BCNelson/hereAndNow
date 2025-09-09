package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/hereandnow"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskAssignmentWorkflow(t *testing.T) {
	// Setup test database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Initialize database schema
	err = storage.RunMigrations(db)
	require.NoError(t, err)

	// Create repositories
	userRepo := storage.NewUserRepository(db)
	taskRepo := storage.NewTaskRepository(db)
	assignmentRepo := storage.NewTaskAssignmentRepository(db)
	notificationRepo := storage.NewNotificationRepository(db)

	// Create services
	assignmentService := hereandnow.NewAssignmentService(assignmentRepo, taskRepo, notificationRepo)

	t.Run("Complete delegation workflow", func(t *testing.T) {
		ctx := context.Background()

		// Create users
		manager := &models.User{
			ID:       uuid.New(),
			Email:    "manager@example.com",
			Name:     "Manager",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, manager, "password123")
		require.NoError(t, err)

		employee1 := &models.User{
			ID:       uuid.New(),
			Email:    "employee1@example.com",
			Name:     "Employee 1",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, employee1, "password123")
		require.NoError(t, err)

		employee2 := &models.User{
			ID:       uuid.New(),
			Email:    "employee2@example.com",
			Name:     "Employee 2",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, employee2, "password123")
		require.NoError(t, err)

		// Manager creates a task
		task := &models.Task{
			ID:               uuid.New(),
			UserID:           manager.ID,
			Title:            "Prepare quarterly report",
			Description:      "Compile Q4 sales data and create presentation",
			EstimatedMinutes: 240,
			Priority:         models.PriorityHigh,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Step 1: Manager assigns task to employee1
		assignment, err := assignmentService.AssignTask(ctx, task.ID, employee1.ID, manager.ID, "Please complete by EOW")
		require.NoError(t, err)
		assert.Equal(t, models.AssignmentStatusPending, assignment.Status)

		// Verify notification was created for employee1
		notifications, err := notificationRepo.GetUserNotifications(ctx, employee1.ID, false)
		require.NoError(t, err)
		assert.Len(t, notifications, 1, "Employee1 should have 1 notification")
		assert.Contains(t, notifications[0].Message, "assigned you a task")

		// Step 2: Employee1 views the assignment
		assignments, err := assignmentService.GetUserAssignments(ctx, employee1.ID)
		require.NoError(t, err)
		assert.Len(t, assignments, 1, "Employee1 should see 1 assignment")
		assert.Equal(t, "Prepare quarterly report", assignments[0].Task.Title)

		// Step 3: Employee1 accepts the assignment
		err = assignmentService.RespondToAssignment(ctx, assignment.ID, employee1.ID, models.AssignmentStatusAccepted, "Will complete by Thursday")
		require.NoError(t, err)

		// Verify status update
		updatedAssignment, err := assignmentRepo.GetByID(ctx, assignment.ID)
		require.NoError(t, err)
		assert.Equal(t, models.AssignmentStatusAccepted, updatedAssignment.Status)
		assert.NotNil(t, updatedAssignment.RespondedAt)
		assert.Equal(t, "Will complete by Thursday", updatedAssignment.ResponseNote)

		// Manager gets notification of acceptance
		managerNotifications, err := notificationRepo.GetUserNotifications(ctx, manager.ID, false)
		require.NoError(t, err)
		assert.Greater(t, len(managerNotifications), 0, "Manager should have notification")

		// Step 4: Employee1 starts working on the task
		task.Status = models.TaskStatusInProgress
		task.StartedAt = ptrTime(time.Now())
		err = taskRepo.Update(ctx, task)
		require.NoError(t, err)

		// Step 5: Employee1 completes the task
		err = assignmentService.CompleteAssignedTask(ctx, task.ID, employee1.ID, "Report completed and uploaded to shared drive")
		require.NoError(t, err)

		// Verify task completion
		completedTask, err := taskRepo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, models.TaskStatusCompleted, completedTask.Status)
		assert.NotNil(t, completedTask.CompletedAt)
		assert.Equal(t, employee1.ID, *completedTask.CompletedBy)

		// Verify assignment completion
		completedAssignment, err := assignmentRepo.GetByID(ctx, assignment.ID)
		require.NoError(t, err)
		assert.Equal(t, models.AssignmentStatusCompleted, completedAssignment.Status)
		assert.NotNil(t, completedAssignment.CompletedAt)

		// Manager gets completion notification
		finalNotifications, err := notificationRepo.GetUserNotifications(ctx, manager.ID, false)
		require.NoError(t, err)
		assert.Greater(t, len(finalNotifications), 1, "Manager should have completion notification")
	})

	t.Run("Assignment rejection workflow", func(t *testing.T) {
		ctx := context.Background()

		// Create users
		lead := &models.User{
			ID:       uuid.New(),
			Email:    "lead@example.com",
			Name:     "Team Lead",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, lead, "password123")
		require.NoError(t, err)

		developer := &models.User{
			ID:       uuid.New(),
			Email:    "developer@example.com",
			Name:     "Developer",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, developer, "password123")
		require.NoError(t, err)

		// Lead creates a task
		task := &models.Task{
			ID:               uuid.New(),
			UserID:           lead.ID,
			Title:            "Implement new API endpoint",
			EstimatedMinutes: 480,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Lead assigns task to developer
		assignment, err := assignmentService.AssignTask(ctx, task.ID, developer.ID, lead.ID, "Need this for the sprint")
		require.NoError(t, err)

		// Developer rejects the assignment
		err = assignmentService.RespondToAssignment(ctx, assignment.ID, developer.ID, models.AssignmentStatusRejected, "Already at capacity this sprint")
		require.NoError(t, err)

		// Verify rejection
		rejectedAssignment, err := assignmentRepo.GetByID(ctx, assignment.ID)
		require.NoError(t, err)
		assert.Equal(t, models.AssignmentStatusRejected, rejectedAssignment.Status)
		assert.Equal(t, "Already at capacity this sprint", rejectedAssignment.ResponseNote)

		// Task should revert to unassigned
		unassignedTask, err := taskRepo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, models.TaskStatusPending, unassignedTask.Status)

		// Lead gets rejection notification
		leadNotifications, err := notificationRepo.GetUserNotifications(ctx, lead.ID, false)
		require.NoError(t, err)
		assert.Greater(t, len(leadNotifications), 0, "Lead should have rejection notification")
	})

	t.Run("Reassignment workflow", func(t *testing.T) {
		ctx := context.Background()

		// Create users
		supervisor := &models.User{
			ID:       uuid.New(),
			Email:    "supervisor@example.com",
			Name:     "Supervisor",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, supervisor, "password123")
		require.NoError(t, err)

		worker1 := &models.User{
			ID:       uuid.New(),
			Email:    "worker1@example.com",
			Name:     "Worker 1",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, worker1, "password123")
		require.NoError(t, err)

		worker2 := &models.User{
			ID:       uuid.New(),
			Email:    "worker2@example.com",
			Name:     "Worker 2",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, worker2, "password123")
		require.NoError(t, err)

		// Create task
		task := &models.Task{
			ID:               uuid.New(),
			UserID:           supervisor.ID,
			Title:            "Update documentation",
			EstimatedMinutes: 120,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Initial assignment to worker1
		assignment1, err := assignmentService.AssignTask(ctx, task.ID, worker1.ID, supervisor.ID, "Please update the API docs")
		require.NoError(t, err)

		// Worker1 accepts
		err = assignmentService.RespondToAssignment(ctx, assignment1.ID, worker1.ID, models.AssignmentStatusAccepted, "On it")
		require.NoError(t, err)

		// Worker1 becomes unavailable, supervisor reassigns to worker2
		err = assignmentService.CancelAssignment(ctx, assignment1.ID, supervisor.ID, "Worker1 out sick")
		require.NoError(t, err)

		// Verify cancellation
		cancelledAssignment, err := assignmentRepo.GetByID(ctx, assignment1.ID)
		require.NoError(t, err)
		assert.Equal(t, models.AssignmentStatusCancelled, cancelledAssignment.Status)

		// Reassign to worker2
		assignment2, err := assignmentService.AssignTask(ctx, task.ID, worker2.ID, supervisor.ID, "Reassigning due to availability")
		require.NoError(t, err)
		assert.NotEqual(t, assignment1.ID, assignment2.ID)

		// Worker2 accepts and completes
		err = assignmentService.RespondToAssignment(ctx, assignment2.ID, worker2.ID, models.AssignmentStatusAccepted, "Will do")
		require.NoError(t, err)

		err = assignmentService.CompleteAssignedTask(ctx, task.ID, worker2.ID, "Documentation updated")
		require.NoError(t, err)

		// Verify final state
		finalTask, err := taskRepo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, models.TaskStatusCompleted, finalTask.Status)
		assert.Equal(t, worker2.ID, *finalTask.CompletedBy)
	})

	t.Run("Assignment with deadline tracking", func(t *testing.T) {
		ctx := context.Background()

		// Create users
		projectManager := &models.User{
			ID:       uuid.New(),
			Email:    "pm@example.com",
			Name:     "Project Manager",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, projectManager, "password123")
		require.NoError(t, err)

		contractor := &models.User{
			ID:       uuid.New(),
			Email:    "contractor@example.com",
			Name:     "Contractor",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, contractor, "password123")
		require.NoError(t, err)

		now := time.Now()
		deadline := now.Add(48 * time.Hour)

		// Create task with deadline
		task := &models.Task{
			ID:               uuid.New(),
			UserID:           projectManager.ID,
			Title:            "Critical bug fix",
			EstimatedMinutes: 180,
			Priority:         models.PriorityCritical,
			DueDate:          &deadline,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Assign with deadline emphasis
		assignment, err := assignmentService.AssignTask(ctx, task.ID, contractor.ID, projectManager.ID, "URGENT: Due in 48 hours")
		require.NoError(t, err)
		assert.NotNil(t, assignment.DueDate)
		assert.Equal(t, deadline.Unix(), assignment.DueDate.Unix())

		// Contractor accepts
		err = assignmentService.RespondToAssignment(ctx, assignment.ID, contractor.ID, models.AssignmentStatusAccepted, "Will prioritize")
		require.NoError(t, err)

		// Simulate work in progress updates
		progressUpdates := []struct {
			hours    int
			progress int
			note     string
		}{
			{6, 25, "Identified root cause"},
			{12, 50, "Fix implemented, testing"},
			{18, 75, "Tests passing, preparing deploy"},
			{24, 100, "Deployed to production"},
		}

		for _, update := range progressUpdates {
			updateTime := now.Add(time.Duration(update.hours) * time.Hour)
			
			// Update assignment progress
			err = assignmentService.UpdateProgress(ctx, assignment.ID, contractor.ID, update.progress, update.note, updateTime)
			require.NoError(t, err)

			// Project manager gets progress notification
			notifications, err := notificationRepo.GetUserNotifications(ctx, projectManager.ID, false)
			require.NoError(t, err)
			assert.Greater(t, len(notifications), 0, "PM should get progress updates")
		}

		// Complete before deadline
		completionTime := now.Add(24 * time.Hour)
		err = assignmentService.CompleteAssignedTaskAt(ctx, task.ID, contractor.ID, "Bug fixed and deployed", completionTime)
		require.NoError(t, err)

		// Verify on-time completion
		finalAssignment, err := assignmentRepo.GetByID(ctx, assignment.ID)
		require.NoError(t, err)
		assert.True(t, finalAssignment.CompletedAt.Before(deadline), "Completed before deadline")
	})

	t.Run("Bulk assignment to team", func(t *testing.T) {
		ctx := context.Background()

		// Create team lead and team members
		teamLead := &models.User{
			ID:       uuid.New(),
			Email:    "teamlead@example.com",
			Name:     "Team Lead",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, teamLead, "password123")
		require.NoError(t, err)

		// Create team members
		teamMembers := make([]*models.User, 3)
		for i := 0; i < 3; i++ {
			teamMembers[i] = &models.User{
				ID:       uuid.New(),
				Email:    fmt.Sprintf("member%d@example.com", i+1),
				Name:     fmt.Sprintf("Team Member %d", i+1),
				Timezone: "America/New_York",
			}
			err = userRepo.Create(ctx, teamMembers[i], "password123")
			require.NoError(t, err)
		}

		// Create multiple tasks for distribution
		tasks := make([]*models.Task, 5)
		taskTitles := []string{
			"Code review PR #123",
			"Update unit tests",
			"Fix linting errors",
			"Document API changes",
			"Deploy to staging",
		}

		for i, title := range taskTitles {
			tasks[i] = &models.Task{
				ID:               uuid.New(),
				UserID:           teamLead.ID,
				Title:            title,
				EstimatedMinutes: 60 + (i * 30),
				Status:           models.TaskStatusPending,
			}
			err = taskRepo.Create(ctx, tasks[i])
			require.NoError(t, err)
		}

		// Distribute tasks among team members
		assignments := make([]*models.TaskAssignment, 0)
		for i, task := range tasks {
			assignee := teamMembers[i%len(teamMembers)] // Round-robin assignment
			assignment, err := assignmentService.AssignTask(ctx, task.ID, assignee.ID, teamLead.ID, "Part of sprint tasks")
			require.NoError(t, err)
			assignments = append(assignments, assignment)
		}

		// Verify distribution
		for i, member := range teamMembers {
			memberAssignments, err := assignmentService.GetUserAssignments(ctx, member.ID)
			require.NoError(t, err)
			
			expectedCount := 2
			if i == 2 { // Last member gets fewer in this distribution
				expectedCount = 1
			}
			assert.Len(t, memberAssignments, expectedCount, "Member %d should have correct number of assignments", i+1)
		}

		// Team members accept their assignments
		for i, assignment := range assignments {
			assignee := teamMembers[i%len(teamMembers)]
			err = assignmentService.RespondToAssignment(ctx, assignment.ID, assignee.ID, models.AssignmentStatusAccepted, "Accepted")
			require.NoError(t, err)
		}

		// Track team progress
		teamProgress, err := assignmentService.GetTeamAssignmentStats(ctx, teamLead.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, teamProgress.TotalAssigned)
		assert.Equal(t, 5, teamProgress.Accepted)
		assert.Equal(t, 0, teamProgress.Completed)
	})

	t.Run("Assignment permissions and validation", func(t *testing.T) {
		ctx := context.Background()

		// Create users with different roles
		owner := &models.User{
			ID:       uuid.New(),
			Email:    "owner@example.com",
			Name:     "Task Owner",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, owner, "password123")
		require.NoError(t, err)

		otherUser := &models.User{
			ID:       uuid.New(),
			Email:    "other@example.com",
			Name:     "Other User",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, otherUser, "password123")
		require.NoError(t, err)

		assignee := &models.User{
			ID:       uuid.New(),
			Email:    "assignee@example.com",
			Name:     "Assignee",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, assignee, "password123")
		require.NoError(t, err)

		// Owner creates task
		task := &models.Task{
			ID:               uuid.New(),
			UserID:           owner.ID,
			Title:            "Private task",
			EstimatedMinutes: 60,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, task)
		require.NoError(t, err)

		// Test: Only owner can assign their own tasks
		_, err = assignmentService.AssignTask(ctx, task.ID, assignee.ID, otherUser.ID, "Trying to assign someone else's task")
		assert.Error(t, err, "Non-owner should not be able to assign task")

		// Owner successfully assigns
		assignment, err := assignmentService.AssignTask(ctx, task.ID, assignee.ID, owner.ID, "Please complete")
		require.NoError(t, err)

		// Test: Can't assign already assigned task
		_, err = assignmentService.AssignTask(ctx, task.ID, otherUser.ID, owner.ID, "Double assignment")
		assert.Error(t, err, "Should not be able to assign already assigned task")

		// Test: Only assignee can accept/reject
		err = assignmentService.RespondToAssignment(ctx, assignment.ID, otherUser.ID, models.AssignmentStatusAccepted, "Not my assignment")
		assert.Error(t, err, "Non-assignee should not be able to respond")

		// Assignee successfully responds
		err = assignmentService.RespondToAssignment(ctx, assignment.ID, assignee.ID, models.AssignmentStatusAccepted, "Will do")
		require.NoError(t, err)

		// Test: Can't complete someone else's assigned task
		err = assignmentService.CompleteAssignedTask(ctx, task.ID, otherUser.ID, "Trying to complete")
		assert.Error(t, err, "Non-assignee should not be able to complete")

		// Assignee successfully completes
		err = assignmentService.CompleteAssignedTask(ctx, task.ID, assignee.ID, "Completed")
		require.NoError(t, err)
	})
}

