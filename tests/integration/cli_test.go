package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIIntegration(t *testing.T) {
	// Build the CLI binary for testing
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)

	// Setup test database file
	tempDir := os.TempDir()
	testDBPath := filepath.Join(tempDir, "test_cli.db")
	defer os.Remove(testDBPath)

	t.Run("CLI initialization and basic commands", func(t *testing.T) {
		// Test help command
		cmd := exec.Command(binaryPath, "--help")
		output, err := cmd.Output()
		require.NoError(t, err)
		
		helpText := string(output)
		assert.Contains(t, helpText, "hereandnow", "Help should contain program name")
		assert.Contains(t, helpText, "COMMANDS", "Help should list commands")
		assert.Contains(t, helpText, "task", "Help should mention task commands")
		assert.Contains(t, helpText, "user", "Help should mention user commands")

		// Test version command
		cmd = exec.Command(binaryPath, "--version")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		versionText := string(output)
		assert.Contains(t, versionText, "version", "Version should contain version info")

		// Test init command
		cmd = exec.Command(binaryPath, "init", "--database", testDBPath)
		output, err = cmd.Output()
		require.NoError(t, err)
		
		initText := string(output)
		assert.Contains(t, initText, "initialized", "Init should confirm initialization")

		// Verify database was created
		assert.FileExists(t, testDBPath, "Database file should be created")
	})

	t.Run("User management commands", func(t *testing.T) {
		// Create user
		cmd := exec.Command(binaryPath, "user", "create", 
			"--database", testDBPath,
			"--email", "cli-user@example.com",
			"--name", "CLI Test User",
			"--password", "testpassword123")
		output, err := cmd.Output()
		require.NoError(t, err)
		
		createOutput := string(output)
		assert.Contains(t, createOutput, "created", "Should confirm user creation")
		assert.Contains(t, createOutput, "cli-user@example.com", "Should show user email")

		// List users
		cmd = exec.Command(binaryPath, "user", "list", "--database", testDBPath)
		output, err = cmd.Output()
		require.NoError(t, err)
		
		listOutput := string(output)
		assert.Contains(t, listOutput, "CLI Test User", "Should list created user")
		assert.Contains(t, listOutput, "cli-user@example.com", "Should show user email")

		// Update user
		cmd = exec.Command(binaryPath, "user", "update",
			"--database", testDBPath,
			"--email", "cli-user@example.com",
			"--name", "Updated CLI User")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		updateOutput := string(output)
		assert.Contains(t, updateOutput, "updated", "Should confirm user update")

		// Verify update
		cmd = exec.Command(binaryPath, "user", "list", "--database", testDBPath)
		output, err = cmd.Output()
		require.NoError(t, err)
		
		verifyOutput := string(output)
		assert.Contains(t, verifyOutput, "Updated CLI User", "Should show updated name")
	})

	t.Run("Task management commands", func(t *testing.T) {
		// First ensure we have a user
		setupUser(t, binaryPath, testDBPath)

		// Add a task
		cmd := exec.Command(binaryPath, "task", "add",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--title", "Buy groceries",
			"--description", "Need milk, eggs, and bread",
			"--minutes", "30")
		output, err := cmd.Output()
		require.NoError(t, err)
		
		addOutput := string(output)
		assert.Contains(t, addOutput, "added", "Should confirm task addition")
		assert.Contains(t, addOutput, "Buy groceries", "Should show task title")

		// List tasks
		cmd = exec.Command(binaryPath, "task", "list",
			"--database", testDBPath,
			"--user", "cli-user@example.com")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		listOutput := string(output)
		assert.Contains(t, listOutput, "Buy groceries", "Should list added task")
		assert.Contains(t, listOutput, "pending", "Should show task status")
		assert.Contains(t, listOutput, "30m", "Should show estimated time")

		// Add task with priority
		cmd = exec.Command(binaryPath, "task", "add",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--title", "Fix urgent bug",
			"--priority", "high",
			"--minutes", "120")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		priorityOutput := string(output)
		assert.Contains(t, priorityOutput, "added", "Should confirm high priority task")

		// List with format options
		cmd = exec.Command(binaryPath, "task", "list",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--format", "json")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		jsonOutput := string(output)
		assert.Contains(t, jsonOutput, `"title"`, "JSON output should contain title field")
		assert.Contains(t, jsonOutput, `"Buy groceries"`, "JSON should contain task")

		// Get first task ID for further operations
		taskID := extractTaskIDFromList(t, binaryPath, testDBPath)

		// Complete a task
		cmd = exec.Command(binaryPath, "task", "complete",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--id", taskID)
		output, err = cmd.Output()
		require.NoError(t, err)
		
		completeOutput := string(output)
		assert.Contains(t, completeOutput, "completed", "Should confirm task completion")

		// Verify completion
		cmd = exec.Command(binaryPath, "task", "list",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--status", "completed")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		completedListOutput := string(output)
		assert.Contains(t, completedListOutput, "completed", "Should show completed tasks")
	})

	t.Run("Location management commands", func(t *testing.T) {
		// Ensure user exists
		setupUser(t, binaryPath, testDBPath)

		// Add location
		cmd := exec.Command(binaryPath, "location", "add",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--name", "Home",
			"--latitude", "40.7128",
			"--longitude", "-74.0060",
			"--radius", "100")
		output, err := cmd.Output()
		require.NoError(t, err)
		
		addLocationOutput := string(output)
		assert.Contains(t, addLocationOutput, "added", "Should confirm location addition")
		assert.Contains(t, addLocationOutput, "Home", "Should show location name")

		// Add another location
		cmd = exec.Command(binaryPath, "location", "add",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--name", "Office",
			"--latitude", "40.7580",
			"--longitude", "-73.9855",
			"--radius", "50")
		output, err = cmd.Output()
		require.NoError(t, err)

		// List locations
		cmd = exec.Command(binaryPath, "location", "list",
			"--database", testDBPath,
			"--user", "cli-user@example.com")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		listLocationsOutput := string(output)
		assert.Contains(t, listLocationsOutput, "Home", "Should list home location")
		assert.Contains(t, listLocationsOutput, "Office", "Should list office location")
		assert.Contains(t, listLocationsOutput, "40.7128", "Should show coordinates")

		// Update location
		cmd = exec.Command(binaryPath, "location", "update",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--name", "Home",
			"--radius", "150")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		updateLocationOutput := string(output)
		assert.Contains(t, updateLocationOutput, "updated", "Should confirm location update")
	})

	t.Run("Context management commands", func(t *testing.T) {
		// Ensure user and locations exist
		setupUser(t, binaryPath, testDBPath)
		setupLocations(t, binaryPath, testDBPath)

		// Set context
		cmd := exec.Command(binaryPath, "context", "update",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--latitude", "40.7128",
			"--longitude", "-74.0060",
			"--available", "90",
			"--energy", "high",
			"--social", "alone")
		output, err := cmd.Output()
		require.NoError(t, err)
		
		contextOutput := string(output)
		assert.Contains(t, contextOutput, "updated", "Should confirm context update")

		// Show context
		cmd = exec.Command(binaryPath, "context", "show",
			"--database", testDBPath,
			"--user", "cli-user@example.com")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		showContextOutput := string(output)
		assert.Contains(t, showContextOutput, "40.7128", "Should show current latitude")
		assert.Contains(t, showContextOutput, "90", "Should show available minutes")
		assert.Contains(t, showContextOutput, "high", "Should show energy level")
		assert.Contains(t, showContextOutput, "alone", "Should show social context")
	})

	t.Run("Server commands", func(t *testing.T) {
		// Test migrate command
		cmd := exec.Command(binaryPath, "migrate",
			"--database", testDBPath)
		output, err := cmd.Output()
		require.NoError(t, err)
		
		migrateOutput := string(output)
		assert.Contains(t, migrateOutput, "migration", "Should mention migration")

		// Test server start (run briefly then stop)
		serverCmd := exec.Command(binaryPath, "serve",
			"--database", testDBPath,
			"--port", "8081")
		
		// Start server in background
		err = serverCmd.Start()
		require.NoError(t, err)

		// Give server time to start
		time.Sleep(2 * time.Second)

		// Kill server
		err = serverCmd.Process.Kill()
		require.NoError(t, err)

		// Wait for process to finish
		serverCmd.Wait()
	})

	t.Run("Filtered task listing with context", func(t *testing.T) {
		// Setup complete environment
		setupUser(t, binaryPath, testDBPath)
		setupLocations(t, binaryPath, testDBPath)
		setupTasks(t, binaryPath, testDBPath)

		// Set context at home
		cmd := exec.Command(binaryPath, "context", "update",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--latitude", "40.7128",
			"--longitude", "-74.0060",
			"--available", "60",
			"--energy", "high")
		_, err := cmd.Output()
		require.NoError(t, err)

		// List filtered tasks
		cmd = exec.Command(binaryPath, "task", "list",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--filter")
		output, err := cmd.Output()
		require.NoError(t, err)
		
		filteredOutput := string(output)
		// Should show tasks available in current context
		assert.NotEmpty(t, strings.TrimSpace(filteredOutput), "Should return filtered tasks")

		// Set context with limited time
		cmd = exec.Command(binaryPath, "context", "update",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--latitude", "40.7128",
			"--longitude", "-74.0060",
			"--available", "15", // Very limited time
			"--energy", "low")
		_, err = cmd.Output()
		require.NoError(t, err)

		// List filtered tasks with limited time
		cmd = exec.Command(binaryPath, "task", "list",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--filter")
		output, err = cmd.Output()
		require.NoError(t, err)
		
		limitedOutput := string(output)
		// Should show fewer or different tasks due to time constraint
		assert.NotEqual(t, filteredOutput, limitedOutput, "Results should differ with different context")
	})

	t.Run("Error handling and validation", func(t *testing.T) {
		// Test invalid database path
		cmd := exec.Command(binaryPath, "task", "list",
			"--database", "/invalid/path/db.sqlite",
			"--user", "test@example.com")
		output, err := cmd.CombinedOutput()
		assert.Error(t, err, "Should fail with invalid database path")
		
		errorOutput := string(output)
		assert.Contains(t, strings.ToLower(errorOutput), "error", "Should show error message")

		// Test missing required flags
		cmd = exec.Command(binaryPath, "task", "add") // Missing database and user
		output, err = cmd.CombinedOutput()
		assert.Error(t, err, "Should fail with missing required flags")
		
		missingFlagOutput := string(output)
		assert.Contains(t, strings.ToLower(missingFlagOutput), "required", "Should mention required flags")

		// Test invalid user
		cmd = exec.Command(binaryPath, "task", "list",
			"--database", testDBPath,
			"--user", "nonexistent@example.com")
		output, err = cmd.CombinedOutput()
		assert.Error(t, err, "Should fail with nonexistent user")

		// Test invalid task ID
		cmd = exec.Command(binaryPath, "task", "complete",
			"--database", testDBPath,
			"--user", "cli-user@example.com",
			"--id", "invalid-uuid")
		output, err = cmd.CombinedOutput()
		assert.Error(t, err, "Should fail with invalid task ID")
	})

	t.Run("Configuration file handling", func(t *testing.T) {
		// Test with config file
		configDir := filepath.Join(tempDir, ".hereandnow")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)
		defer os.RemoveAll(configDir)

		configPath := filepath.Join(configDir, "config.yaml")
		configContent := fmt.Sprintf(`database: %s
user: cli-user@example.com
format: json
`, testDBPath)

		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Test command using config file
		cmd := exec.Command(binaryPath, "task", "list", "--config", configPath)
		output, err := cmd.Output()
		require.NoError(t, err)
		
		configOutput := string(output)
		// Should use JSON format from config
		assert.Contains(t, configOutput, `"title"`, "Should use JSON format from config")
	})
}

func buildCLIBinary(t *testing.T) string {
	// Get the project root directory
	wd, err := os.Getwd()
	require.NoError(t, err)
	
	// Go up to project root (assuming we're in tests/integration)
	projectRoot := filepath.Join(wd, "..", "..")
	
	// Build the CLI binary
	binaryPath := filepath.Join(os.TempDir(), "hereandnow-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/hereandnow")
	cmd.Dir = projectRoot
	
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build CLI binary: %s", string(output))
	
	return binaryPath
}

func setupUser(t *testing.T, binaryPath, dbPath string) {
	cmd := exec.Command(binaryPath, "user", "create",
		"--database", dbPath,
		"--email", "cli-user@example.com",
		"--name", "CLI Test User",
		"--password", "testpassword123")
	_, err := cmd.Output()
	require.NoError(t, err, "Failed to setup test user")
}

func setupLocations(t *testing.T, binaryPath, dbPath string) {
	locations := [][]string{
		{"Home", "40.7128", "-74.0060", "100"},
		{"Office", "40.7580", "-73.9855", "50"},
		{"Grocery Store", "40.7260", "-73.9897", "200"},
	}

	for _, loc := range locations {
		cmd := exec.Command(binaryPath, "location", "add",
			"--database", dbPath,
			"--user", "cli-user@example.com",
			"--name", loc[0],
			"--latitude", loc[1],
			"--longitude", loc[2],
			"--radius", loc[3])
		_, err := cmd.Output()
		require.NoError(t, err, "Failed to setup location: %s", loc[0])
	}
}

func setupTasks(t *testing.T, binaryPath, dbPath string) {
	tasks := [][]string{
		{"Buy milk", "Quick grocery run", "15", "low"},
		{"Weekly planning", "Plan next week's tasks", "45", "medium"},
		{"Deep work session", "Focus on important project", "120", "high"},
		{"Quick email check", "Respond to urgent emails", "10", "low"},
	}

	for _, task := range tasks {
		cmd := exec.Command(binaryPath, "task", "add",
			"--database", dbPath,
			"--user", "cli-user@example.com",
			"--title", task[0],
			"--description", task[1],
			"--minutes", task[2],
			"--priority", task[3])
		_, err := cmd.Output()
		require.NoError(t, err, "Failed to setup task: %s", task[0])
	}
}

func extractTaskIDFromList(t *testing.T, binaryPath, dbPath string) string {
	cmd := exec.Command(binaryPath, "task", "list",
		"--database", dbPath,
		"--user", "cli-user@example.com",
		"--format", "json")
	output, err := cmd.Output()
	require.NoError(t, err)

	outputStr := string(output)
	// Extract first task ID from JSON output
	// This is a simple extraction - in a real implementation,
	// we'd parse the JSON properly
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, `"id"`) {
			parts := strings.Split(line, `"`)
			if len(parts) >= 4 {
				// Return the UUID-like string
				id := parts[3]
				if len(id) > 30 { // Basic UUID length check
					return id
				}
			}
		}
	}
	
	t.Fatal("Could not extract task ID from list output")
	return ""
}