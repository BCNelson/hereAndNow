package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/api"
	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIIntegration(t *testing.T) {
	// Setup test database and server
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = storage.RunMigrations(db)
	require.NoError(t, err)

	// Initialize repositories
	userRepo := storage.NewUserRepository(db)
	taskRepo := storage.NewTaskRepository(db)
	locationRepo := storage.NewLocationRepository(db)
	contextRepo := storage.NewContextRepository(db)

	// Create API handlers
	apiHandlers := &api.Handlers{
		UserRepo:     userRepo,
		TaskRepo:     taskRepo,
		LocationRepo: locationRepo,
		ContextRepo:  contextRepo,
	}

	// Setup router
	router := mux.NewRouter()
	api.SetupRoutes(router, apiHandlers)
	
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("Complete authentication flow", func(t *testing.T) {
		// Register new user
		registerPayload := map[string]interface{}{
			"email":    "apitest@example.com",
			"name":     "API Test User",
			"password": "securepassword123",
			"timezone": "America/New_York",
		}
		
		registerBody, _ := json.Marshal(registerPayload)
		resp, err := http.Post(server.URL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(registerBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var registerResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&registerResp)
		require.NoError(t, err)
		assert.Contains(t, registerResp, "user")
		assert.Contains(t, registerResp, "token")

		// Extract token for authenticated requests
		token := registerResp["token"].(string)

		// Login with same credentials
		loginPayload := map[string]interface{}{
			"email":    "apitest@example.com",
			"password": "securepassword123",
		}
		
		loginBody, _ := json.Marshal(loginPayload)
		resp, err = http.Post(server.URL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(loginBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var loginResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&loginResp)
		require.NoError(t, err)
		assert.Contains(t, loginResp, "token")

		// Use token for authenticated request
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		client := &http.Client{}
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var userResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&userResp)
		require.NoError(t, err)
		assert.Equal(t, "apitest@example.com", userResp["email"])
		assert.Equal(t, "API Test User", userResp["name"])

		// Logout
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify token is invalidated
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Complete task management flow", func(t *testing.T) {
		// Create authenticated user first
		token := createAuthenticatedUser(t, server, "taskuser@example.com", "Task User")

		client := &http.Client{}

		// Create a location first
		locationPayload := map[string]interface{}{
			"name":      "Coffee Shop",
			"latitude":  40.7128,
			"longitude": -74.0060,
			"radius":    100,
		}
		
		locationBody, _ := json.Marshal(locationPayload)
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/locations", bytes.NewBuffer(locationBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var locationResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&locationResp)
		require.NoError(t, err)
		locationID := locationResp["id"].(string)

		// Create a task
		taskPayload := map[string]interface{}{
			"title":             "Buy coffee beans",
			"description":       "Get medium roast beans from local coffee shop",
			"estimatedMinutes":  30,
			"priority":          "medium",
			"locationIds":       []string{locationID},
		}
		
		taskBody, _ := json.Marshal(taskPayload)
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/tasks", bytes.NewBuffer(taskBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var taskResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&taskResp)
		require.NoError(t, err)
		taskID := taskResp["id"].(string)
		assert.Equal(t, "Buy coffee beans", taskResp["title"])

		// Get specific task
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/tasks/"+taskID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var taskDetailResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&taskDetailResp)
		require.NoError(t, err)
		assert.Equal(t, "Buy coffee beans", taskDetailResp["title"])
		assert.Contains(t, taskDetailResp, "locations")

		// Update task
		updatePayload := map[string]interface{}{
			"title":             "Buy premium coffee beans",
			"description":       "Get premium single-origin beans",
			"estimatedMinutes":  45,
		}
		
		updateBody, _ := json.Marshal(updatePayload)
		req, _ = http.NewRequest("PATCH", server.URL+"/api/v1/tasks/"+taskID, bytes.NewBuffer(updateBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updatedTaskResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&updatedTaskResp)
		require.NoError(t, err)
		assert.Equal(t, "Buy premium coffee beans", updatedTaskResp["title"])
		assert.Equal(t, float64(45), updatedTaskResp["estimatedMinutes"])

		// Set user context near the location
		contextPayload := map[string]interface{}{
			"currentLatitude":  40.7128,
			"currentLongitude": -74.0060,
			"availableMinutes": 60,
			"energyLevel":      "high",
			"socialContext":    "alone",
		}
		
		contextBody, _ := json.Marshal(contextPayload)
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/context", bytes.NewBuffer(contextBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Get filtered tasks (should include the coffee task since we're at the location)
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/tasks", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tasksResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&tasksResp)
		require.NoError(t, err)
		assert.Contains(t, tasksResp, "tasks")
		
		tasks := tasksResp["tasks"].([]interface{})
		assert.Greater(t, len(tasks), 0, "Should return filtered tasks")

		// Complete the task
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/tasks/"+taskID+"/complete", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var completeResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&completeResp)
		require.NoError(t, err)
		assert.Equal(t, "completed", completeResp["status"])

		// Delete the task
		req, _ = http.NewRequest("DELETE", server.URL+"/api/v1/tasks/"+taskID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify task is deleted/soft-deleted
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/tasks/"+taskID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Context and location management", func(t *testing.T) {
		token := createAuthenticatedUser(t, server, "location@example.com", "Location User")
		client := &http.Client{}

		// Create multiple locations
		locations := []map[string]interface{}{
			{
				"name":      "Home",
				"latitude":  40.7128,
				"longitude": -74.0060,
				"radius":    100,
			},
			{
				"name":      "Office",
				"latitude":  40.7580,
				"longitude": -73.9855,
				"radius":    50,
			},
			{
				"name":      "Gym",
				"latitude":  40.7505,
				"longitude": -73.9934,
				"radius":    25,
			},
		}

		createdLocations := make([]string, 0)

		for _, loc := range locations {
			locationBody, _ := json.Marshal(loc)
			req, _ := http.NewRequest("POST", server.URL+"/api/v1/locations", bytes.NewBuffer(locationBody))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var locationResp map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&locationResp)
			require.NoError(t, err)
			createdLocations = append(createdLocations, locationResp["id"].(string))
		}

		// Get all locations
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/locations", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var locationsResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&locationsResp)
		require.NoError(t, err)
		
		locationsList := locationsResp["locations"].([]interface{})
		assert.Len(t, locationsList, 3, "Should return all created locations")

		// Update context to be at home
		contextPayload := map[string]interface{}{
			"currentLatitude":   40.7128,
			"currentLongitude":  -74.0060,
			"currentLocationId": createdLocations[0], // Home
			"availableMinutes":  90,
			"energyLevel":       "high",
			"socialContext":     "family",
		}
		
		contextBody, _ := json.Marshal(contextPayload)
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/context", bytes.NewBuffer(contextBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Get current context
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/context", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var contextResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&contextResp)
		require.NoError(t, err)
		assert.Equal(t, float64(40.7128), contextResp["currentLatitude"])
		assert.Equal(t, createdLocations[0], contextResp["currentLocationId"])
		assert.Equal(t, "family", contextResp["socialContext"])
	})

	t.Run("Natural language task creation", func(t *testing.T) {
		token := createAuthenticatedUser(t, server, "nlp@example.com", "NLP User")
		client := &http.Client{}

		// Create a known location first
		locationPayload := map[string]interface{}{
			"name":      "grocery store",
			"latitude":  40.7260,
			"longitude": -73.9897,
			"radius":    200,
		}
		
		locationBody, _ := json.Marshal(locationPayload)
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/locations", bytes.NewBuffer(locationBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Test natural language parsing
		nlpPayload := map[string]interface{}{
			"text": "buy milk when at grocery store",
		}
		
		nlpBody, _ := json.Marshal(nlpPayload)
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/tasks/natural", bytes.NewBuffer(nlpBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var nlpResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&nlpResp)
		require.NoError(t, err)
		
		assert.Contains(t, nlpResp["title"].(string), "buy milk")
		assert.Contains(t, nlpResp, "locations")
		
		locations := nlpResp["locations"].([]interface{})
		assert.Greater(t, len(locations), 0, "Should detect location from text")
	})

	t.Run("Task filtering and search", func(t *testing.T) {
		token := createAuthenticatedUser(t, server, "search@example.com", "Search User")
		client := &http.Client{}

		// Create multiple tasks with different attributes
		tasks := []map[string]interface{}{
			{
				"title":             "Buy groceries",
				"description":       "Get milk, eggs, and bread",
				"estimatedMinutes":  30,
				"priority":          "medium",
			},
			{
				"title":             "Urgent: Fix production bug",
				"description":       "Critical issue with user authentication",
				"estimatedMinutes":  120,
				"priority":          "critical",
			},
			{
				"title":             "Weekly team meeting",
				"description":       "Discuss project progress and goals",
				"estimatedMinutes":  60,
				"priority":          "low",
			},
		}

		for _, task := range tasks {
			taskBody, _ := json.Marshal(task)
			req, _ := http.NewRequest("POST", server.URL+"/api/v1/tasks", bytes.NewBuffer(taskBody))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}

		// Test filtering by priority
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/tasks?priority=critical", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var priorityResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&priorityResp)
		require.NoError(t, err)
		
		priorityTasks := priorityResp["tasks"].([]interface{})
		assert.Len(t, priorityTasks, 1, "Should return 1 critical task")

		// Test search by text
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/tasks?search=groceries", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		require.NoError(t, err)
		
		searchTasks := searchResp["tasks"].([]interface{})
		assert.Greater(t, len(searchTasks), 0, "Should find tasks matching 'groceries'")

		// Test filtering by time estimate
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/tasks?maxMinutes=60", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var timeResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&timeResp)
		require.NoError(t, err)
		
		timeTasks := timeResp["tasks"].([]interface{})
		// Should return tasks that take 60 minutes or less
		for _, taskInterface := range timeTasks {
			task := taskInterface.(map[string]interface{})
			assert.LessOrEqual(t, task["estimatedMinutes"].(float64), float64(60))
		}
	})

	t.Run("Error handling and validation", func(t *testing.T) {
		token := createAuthenticatedUser(t, server, "error@example.com", "Error User")
		client := &http.Client{}

		// Test invalid task creation
		invalidTaskPayload := map[string]interface{}{
			// Missing required fields
			"description":      "Task without title",
			"estimatedMinutes": -10, // Negative minutes
		}
		
		taskBody, _ := json.Marshal(invalidTaskPayload)
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/tasks", bytes.NewBuffer(taskBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Test accessing non-existent task
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/tasks/"+uuid.New().String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// Test unauthorized access
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/tasks", nil)
		// No Authorization header
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		// Test malformed JSON
		req, _ = http.NewRequest("POST", server.URL+"/api/v1/tasks", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func createAuthenticatedUser(t *testing.T, server *httptest.Server, email, name string) string {
	registerPayload := map[string]interface{}{
		"email":    email,
		"name":     name,
		"password": "testpassword123",
		"timezone": "America/New_York",
	}
	
	registerBody, _ := json.Marshal(registerPayload)
	resp, err := http.Post(server.URL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(registerBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	require.NoError(t, err)

	return registerResp["token"].(string)
}