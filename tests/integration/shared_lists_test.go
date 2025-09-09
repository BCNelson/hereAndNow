package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/hereandnow"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedTaskLists(t *testing.T) {
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
	taskListRepo := storage.NewTaskListRepository(db)
	listMemberRepo := storage.NewListMemberRepository(db)
	locationRepo := storage.NewLocationRepository(db)
	contextRepo := storage.NewContextRepository(db)

	// Create services
	contextService := hereandnow.NewContextService(contextRepo, locationRepo)
	filterEngine := filters.NewEngine()
	taskService := hereandnow.NewTaskService(taskRepo, filterEngine, contextService)
	listService := hereandnow.NewListService(taskListRepo, listMemberRepo, taskService)

	t.Run("Real-time collaboration on shared lists", func(t *testing.T) {
		ctx := context.Background()

		// Create multiple users
		alice := &models.User{
			ID:       uuid.New(),
			Email:    "alice@example.com",
			Name:     "Alice",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, alice, "password123")
		require.NoError(t, err)

		bob := &models.User{
			ID:       uuid.New(),
			Email:    "bob@example.com",
			Name:     "Bob",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, bob, "password123")
		require.NoError(t, err)

		charlie := &models.User{
			ID:       uuid.New(),
			Email:    "charlie@example.com",
			Name:     "Charlie",
			Timezone: "America/Los_Angeles",
		}
		err = userRepo.Create(ctx, charlie, "password123")
		require.NoError(t, err)

		// Alice creates a shared family shopping list
		shoppingList := &models.TaskList{
			ID:          uuid.New(),
			UserID:      alice.ID,
			Name:        "Family Shopping List",
			Description: "Shared grocery shopping for the family",
			IsShared:    true,
			CreatedAt:   time.Now(),
		}
		err = taskListRepo.Create(ctx, shoppingList)
		require.NoError(t, err)

		// Add Bob and Charlie as members
		bobMember := &models.ListMember{
			ID:         uuid.New(),
			ListID:     shoppingList.ID,
			UserID:     bob.ID,
			Role:       models.ListRoleEditor,
			InvitedBy:  alice.ID,
			JoinedAt:   time.Now(),
		}
		err = listMemberRepo.Add(ctx, bobMember)
		require.NoError(t, err)

		charlieMember := &models.ListMember{
			ID:         uuid.New(),
			ListID:     shoppingList.ID,
			UserID:     charlie.ID,
			Role:       models.ListRoleViewer,
			InvitedBy:  alice.ID,
			JoinedAt:   time.Now(),
		}
		err = listMemberRepo.Add(ctx, charlieMember)
		require.NoError(t, err)

		// Alice adds tasks to the shared list
		milkTask := &models.Task{
			ID:               uuid.New(),
			UserID:           alice.ID,
			ListID:           &shoppingList.ID,
			Title:            "Buy milk",
			EstimatedMinutes: 5,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, milkTask)
		require.NoError(t, err)

		breadTask := &models.Task{
			ID:               uuid.New(),
			UserID:           alice.ID,
			ListID:           &shoppingList.ID,
			Title:            "Buy bread",
			EstimatedMinutes: 5,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, breadTask)
		require.NoError(t, err)

		// Bob adds a task to the shared list
		eggsTask := &models.Task{
			ID:               uuid.New(),
			UserID:           bob.ID,
			ListID:           &shoppingList.ID,
			Title:            "Buy eggs",
			EstimatedMinutes: 5,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, eggsTask)
		require.NoError(t, err)

		// Test: All members can see tasks in shared list
		aliceTasks, err := listService.GetListTasks(ctx, alice.ID, shoppingList.ID)
		require.NoError(t, err)
		assert.Len(t, aliceTasks, 3, "Alice should see all 3 tasks")

		bobTasks, err := listService.GetListTasks(ctx, bob.ID, shoppingList.ID)
		require.NoError(t, err)
		assert.Len(t, bobTasks, 3, "Bob should see all 3 tasks")

		charlieTasks, err := listService.GetListTasks(ctx, charlie.ID, shoppingList.ID)
		require.NoError(t, err)
		assert.Len(t, charlieTasks, 3, "Charlie should see all 3 tasks")

		// Bob completes a task
		milkTask.Status = models.TaskStatusCompleted
		milkTask.CompletedAt = ptrTime(time.Now())
		milkTask.CompletedBy = &bob.ID
		err = taskRepo.Update(ctx, milkTask)
		require.NoError(t, err)

		// Test: All members see the updated status
		aliceTasks, err = listService.GetListTasks(ctx, alice.ID, shoppingList.ID)
		require.NoError(t, err)
		
		completedCount := 0
		for _, task := range aliceTasks {
			if task.Status == models.TaskStatusCompleted {
				completedCount++
			}
		}
		assert.Equal(t, 1, completedCount, "Alice should see 1 completed task")

		// Test: Charlie (viewer) cannot modify tasks
		// This would be enforced at the API/service layer
		canEdit := listService.CanEditList(ctx, charlie.ID, shoppingList.ID)
		assert.False(t, canEdit, "Charlie (viewer) should not be able to edit")

		canEditBob := listService.CanEditList(ctx, bob.ID, shoppingList.ID)
		assert.True(t, canEditBob, "Bob (editor) should be able to edit")
	})

	t.Run("Task assignment in shared lists", func(t *testing.T) {
		ctx := context.Background()

		// Create project team users
		manager := &models.User{
			ID:       uuid.New(),
			Email:    "manager@example.com",
			Name:     "Manager",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, manager, "password123")
		require.NoError(t, err)

		developer1 := &models.User{
			ID:       uuid.New(),
			Email:    "dev1@example.com",
			Name:     "Developer 1",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, developer1, "password123")
		require.NoError(t, err)

		developer2 := &models.User{
			ID:       uuid.New(),
			Email:    "dev2@example.com",
			Name:     "Developer 2",
			Timezone: "Europe/London",
		}
		err = userRepo.Create(ctx, developer2, "password123")
		require.NoError(t, err)

		// Manager creates a project task list
		projectList := &models.TaskList{
			ID:          uuid.New(),
			UserID:      manager.ID,
			Name:        "Sprint 1 Tasks",
			Description: "Tasks for the current sprint",
			IsShared:    true,
			CreatedAt:   time.Now(),
		}
		err = taskListRepo.Create(ctx, projectList)
		require.NoError(t, err)

		// Add developers as members
		dev1Member := &models.ListMember{
			ID:         uuid.New(),
			ListID:     projectList.ID,
			UserID:     developer1.ID,
			Role:       models.ListRoleEditor,
			InvitedBy:  manager.ID,
			JoinedAt:   time.Now(),
		}
		err = listMemberRepo.Add(ctx, dev1Member)
		require.NoError(t, err)

		dev2Member := &models.ListMember{
			ID:         uuid.New(),
			ListID:     projectList.ID,
			UserID:     developer2.ID,
			Role:       models.ListRoleEditor,
			InvitedBy:  manager.ID,
			JoinedAt:   time.Now(),
		}
		err = listMemberRepo.Add(ctx, dev2Member)
		require.NoError(t, err)

		// Manager creates and assigns tasks
		backendTask := &models.Task{
			ID:               uuid.New(),
			UserID:           manager.ID,
			ListID:           &projectList.ID,
			Title:            "Implement API endpoint",
			EstimatedMinutes: 240,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, backendTask)
		require.NoError(t, err)

		frontendTask := &models.Task{
			ID:               uuid.New(),
			UserID:           manager.ID,
			ListID:           &projectList.ID,
			Title:            "Create React component",
			EstimatedMinutes: 180,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, frontendTask)
		require.NoError(t, err)

		// Assign tasks to developers
		assignmentRepo := storage.NewTaskAssignmentRepository(db)

		backendAssignment := &models.TaskAssignment{
			ID:             uuid.New(),
			TaskID:         backendTask.ID,
			AssignedToID:   developer1.ID,
			AssignedByID:   manager.ID,
			Status:         models.AssignmentStatusAccepted,
			AssignedAt:     time.Now(),
		}
		err = assignmentRepo.Create(ctx, backendAssignment)
		require.NoError(t, err)

		frontendAssignment := &models.TaskAssignment{
			ID:             uuid.New(),
			TaskID:         frontendTask.ID,
			AssignedToID:   developer2.ID,
			AssignedByID:   manager.ID,
			Status:         models.AssignmentStatusPending,
			AssignedAt:     time.Now(),
		}
		err = assignmentRepo.Create(ctx, frontendAssignment)
		require.NoError(t, err)

		// Test: Developers see their assigned tasks
		dev1Assignments, err := assignmentRepo.GetUserAssignments(ctx, developer1.ID)
		require.NoError(t, err)
		assert.Len(t, dev1Assignments, 1, "Developer 1 should have 1 assignment")

		dev2Assignments, err := assignmentRepo.GetUserAssignments(ctx, developer2.ID)
		require.NoError(t, err)
		assert.Len(t, dev2Assignments, 1, "Developer 2 should have 1 assignment")

		// Developer 2 accepts the assignment
		frontendAssignment.Status = models.AssignmentStatusAccepted
		frontendAssignment.RespondedAt = ptrTime(time.Now())
		err = assignmentRepo.Update(ctx, frontendAssignment)
		require.NoError(t, err)

		// Test: Manager can track assignment statuses
		taskAssignments, err := assignmentRepo.GetTaskAssignments(ctx, backendTask.ID)
		require.NoError(t, err)
		assert.Len(t, taskAssignments, 1, "Backend task should have 1 assignment")
		assert.Equal(t, models.AssignmentStatusAccepted, taskAssignments[0].Status)

		frontendAssignments, err := assignmentRepo.GetTaskAssignments(ctx, frontendTask.ID)
		require.NoError(t, err)
		assert.Len(t, frontendAssignments, 1, "Frontend task should have 1 assignment")
		assert.Equal(t, models.AssignmentStatusAccepted, frontendAssignments[0].Status)
	})

	t.Run("Hierarchical shared lists", func(t *testing.T) {
		ctx := context.Background()

		// Create family members
		parent := &models.User{
			ID:       uuid.New(),
			Email:    "parent@example.com",
			Name:     "Parent",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, parent, "password123")
		require.NoError(t, err)

		child1 := &models.User{
			ID:       uuid.New(),
			Email:    "child1@example.com",
			Name:     "Child 1",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, child1, "password123")
		require.NoError(t, err)

		child2 := &models.User{
			ID:       uuid.New(),
			Email:    "child2@example.com",
			Name:     "Child 2",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, child2, "password123")
		require.NoError(t, err)

		// Create parent list: "Household Chores"
		householdList := &models.TaskList{
			ID:          uuid.New(),
			UserID:      parent.ID,
			Name:        "Household Chores",
			Description: "All family chores",
			IsShared:    true,
			CreatedAt:   time.Now(),
		}
		err = taskListRepo.Create(ctx, householdList)
		require.NoError(t, err)

		// Create child lists under household
		weeklyList := &models.TaskList{
			ID:          uuid.New(),
			UserID:      parent.ID,
			ParentID:    &householdList.ID,
			Name:        "Weekly Chores",
			Description: "Chores to do every week",
			IsShared:    true,
			CreatedAt:   time.Now(),
		}
		err = taskListRepo.Create(ctx, weeklyList)
		require.NoError(t, err)

		dailyList := &models.TaskList{
			ID:          uuid.New(),
			UserID:      parent.ID,
			ParentID:    &householdList.ID,
			Name:        "Daily Chores",
			Description: "Chores to do every day",
			IsShared:    true,
			CreatedAt:   time.Now(),
		}
		err = taskListRepo.Create(ctx, dailyList)
		require.NoError(t, err)

		// Add children as members to all lists
		for _, list := range []*models.TaskList{householdList, weeklyList, dailyList} {
			child1Member := &models.ListMember{
				ID:         uuid.New(),
				ListID:     list.ID,
				UserID:     child1.ID,
				Role:       models.ListRoleEditor,
				InvitedBy:  parent.ID,
				JoinedAt:   time.Now(),
			}
			err = listMemberRepo.Add(ctx, child1Member)
			require.NoError(t, err)

			child2Member := &models.ListMember{
				ID:         uuid.New(),
				ListID:     list.ID,
				UserID:     child2.ID,
				Role:       models.ListRoleEditor,
				InvitedBy:  parent.ID,
				JoinedAt:   time.Now(),
			}
			err = listMemberRepo.Add(ctx, child2Member)
			require.NoError(t, err)
		}

		// Add tasks to different levels
		vacuumTask := &models.Task{
			ID:               uuid.New(),
			UserID:           parent.ID,
			ListID:           &weeklyList.ID,
			Title:            "Vacuum living room",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, vacuumTask)
		require.NoError(t, err)

		dishesTask := &models.Task{
			ID:               uuid.New(),
			UserID:           parent.ID,
			ListID:           &dailyList.ID,
			Title:            "Wash dishes",
			EstimatedMinutes: 20,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, dishesTask)
		require.NoError(t, err)

		// Test: Get all lists for a user
		parentLists, err := listService.GetUserLists(ctx, parent.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(parentLists), 3, "Parent should see all 3 lists")

		child1Lists, err := listService.GetUserLists(ctx, child1.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(child1Lists), 3, "Child 1 should see all 3 shared lists")

		// Test: Get hierarchical structure
		childLists, err := taskListRepo.GetChildLists(ctx, householdList.ID)
		require.NoError(t, err)
		assert.Len(t, childLists, 2, "Household list should have 2 child lists")

		// Test: Tasks in child lists
		weeklyTasks, err := listService.GetListTasks(ctx, child1.ID, weeklyList.ID)
		require.NoError(t, err)
		assert.Len(t, weeklyTasks, 1, "Weekly list should have 1 task")

		dailyTasks, err := listService.GetListTasks(ctx, child2.ID, dailyList.ID)
		require.NoError(t, err)
		assert.Len(t, dailyTasks, 1, "Daily list should have 1 task")
	})

	t.Run("List member permissions", func(t *testing.T) {
		ctx := context.Background()

		// Create users with different roles
		owner := &models.User{
			ID:       uuid.New(),
			Email:    "owner@example.com",
			Name:     "Owner",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, owner, "password123")
		require.NoError(t, err)

		editor := &models.User{
			ID:       uuid.New(),
			Email:    "editor@example.com",
			Name:     "Editor",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, editor, "password123")
		require.NoError(t, err)

		viewer := &models.User{
			ID:       uuid.New(),
			Email:    "viewer@example.com",
			Name:     "Viewer",
			Timezone: "America/New_York",
		}
		err = userRepo.Create(ctx, viewer, "password123")
		require.NoError(t, err)

		// Owner creates a list
		projectList := &models.TaskList{
			ID:          uuid.New(),
			UserID:      owner.ID,
			Name:        "Project Tasks",
			Description: "Tasks with different permission levels",
			IsShared:    true,
			CreatedAt:   time.Now(),
		}
		err = taskListRepo.Create(ctx, projectList)
		require.NoError(t, err)

		// Add members with different roles
		editorMember := &models.ListMember{
			ID:         uuid.New(),
			ListID:     projectList.ID,
			UserID:     editor.ID,
			Role:       models.ListRoleEditor,
			InvitedBy:  owner.ID,
			JoinedAt:   time.Now(),
		}
		err = listMemberRepo.Add(ctx, editorMember)
		require.NoError(t, err)

		viewerMember := &models.ListMember{
			ID:         uuid.New(),
			ListID:     projectList.ID,
			UserID:     viewer.ID,
			Role:       models.ListRoleViewer,
			InvitedBy:  owner.ID,
			JoinedAt:   time.Now(),
		}
		err = listMemberRepo.Add(ctx, viewerMember)
		require.NoError(t, err)

		// Test permissions
		// Owner can do everything
		assert.True(t, listService.CanEditList(ctx, owner.ID, projectList.ID), "Owner can edit")
		assert.True(t, listService.CanDeleteList(ctx, owner.ID, projectList.ID), "Owner can delete")
		assert.True(t, listService.CanInviteMembers(ctx, owner.ID, projectList.ID), "Owner can invite")

		// Editor can edit but not delete or invite
		assert.True(t, listService.CanEditList(ctx, editor.ID, projectList.ID), "Editor can edit")
		assert.False(t, listService.CanDeleteList(ctx, editor.ID, projectList.ID), "Editor cannot delete")
		assert.False(t, listService.CanInviteMembers(ctx, editor.ID, projectList.ID), "Editor cannot invite")

		// Viewer can only view
		assert.False(t, listService.CanEditList(ctx, viewer.ID, projectList.ID), "Viewer cannot edit")
		assert.False(t, listService.CanDeleteList(ctx, viewer.ID, projectList.ID), "Viewer cannot delete")
		assert.False(t, listService.CanInviteMembers(ctx, viewer.ID, projectList.ID), "Viewer cannot invite")

		// Test: Editor creates a task
		editorTask := &models.Task{
			ID:               uuid.New(),
			UserID:           editor.ID,
			ListID:           &projectList.ID,
			Title:            "Task created by editor",
			EstimatedMinutes: 30,
			Status:           models.TaskStatusPending,
		}
		err = taskRepo.Create(ctx, editorTask)
		require.NoError(t, err)

		// All members should see the task
		tasks, err := listService.GetListTasks(ctx, viewer.ID, projectList.ID)
		require.NoError(t, err)
		assert.Len(t, tasks, 1, "Viewer should see task created by editor")
	})
}