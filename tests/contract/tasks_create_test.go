package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTasksCreate validates POST /tasks endpoint against OpenAPI spec
func TestTasksCreate(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		requestBody    interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:  "Valid task creation - minimal",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"title": "Test Task",
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, body []byte) {
				var task map[string]interface{}
				err := json.Unmarshal(body, &task)
				require.NoError(t, err)

				// Validate created Task schema
				assert.Contains(t, task, "id")
				assert.Contains(t, task, "title")
				assert.Contains(t, task, "creator_id")
				assert.Contains(t, task, "status")
				assert.Contains(t, task, "created_at")

				assert.Equal(t, "Test Task", task["title"])
				assert.Equal(t, "pending", task["status"]) // Default status
			},
		},
		{
			name:  "Valid task creation - full details",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"title":             "Complete Project",
				"description":       "Finish the task management project",
				"priority":          3,
				"estimated_minutes": 120,
				"due_at":           "2024-12-31T23:59:59Z",
				"location_ids":     []string{"550e8400-e29b-41d4-a716-446655440000"},
				"dependency_ids":   []string{"550e8400-e29b-41d4-a716-446655440001"},
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, body []byte) {
				var task map[string]interface{}
				err := json.Unmarshal(body, &task)
				require.NoError(t, err)

				assert.Equal(t, "Complete Project", task["title"])
				assert.Equal(t, "Finish the task management project", task["description"])
				assert.Equal(t, float64(3), task["priority"])
				assert.Equal(t, float64(120), task["estimated_minutes"])
				assert.Equal(t, "2024-12-31T23:59:59Z", task["due_at"])

				// Validate relationships are included
				if locations, ok := task["locations"].([]interface{}); ok {
					assert.NotEmpty(t, locations)
				}
				if dependencies, ok := task["dependencies"].([]interface{}); ok {
					assert.NotEmpty(t, dependencies)
				}
			},
		},
		{
			name:  "Task with list assignment",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"title":   "List Task",
				"list_id": "550e8400-e29b-41d4-a716-446655440002",
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, body []byte) {
				var task map[string]interface{}
				err := json.Unmarshal(body, &task)
				require.NoError(t, err)

				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440002", task["list_id"])
			},
		},
		{
			name:           "Missing required title",
			token:          "valid-jwt-token",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				if err == nil {
					assert.Contains(t, response, "error")
				}
			},
		},
		{
			name:  "Invalid priority (too high)",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"title":    "Invalid Priority Task",
				"priority": 10,
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				if err == nil {
					assert.Contains(t, response, "error")
				}
			},
		},
		{
			name:  "Invalid priority (too low)",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"title":    "Invalid Priority Task",
				"priority": 0,
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				if err == nil {
					assert.Contains(t, response, "error")
				}
			},
		},
		{
			name:  "Invalid due date format",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"title":  "Invalid Date Task",
				"due_at": "invalid-date",
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				if err == nil {
					assert.Contains(t, response, "error")
				}
			},
		},
		{
			name:           "Request without token",
			token:          "",
			requestBody:    map[string]string{"title": "Test Task"},
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
		{
			name:           "Request with invalid token",
			token:          "invalid-token",
			requestBody:    map[string]string{"title": "Test Task"},
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			
			// Add authorization header if token provided
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// This will fail until we implement the handler
			// Call the handler (not implemented yet - this MUST fail)
			handler := getTasksCreateHandler() // This function doesn't exist yet
			handler.ServeHTTP(rr, req)

			// Validate response
			assert.Equal(t, tt.expectedStatus, rr.Code)
			
			if tt.expectedStatus == http.StatusCreated {
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
				if tt.validateBody != nil {
					tt.validateBody(t, rr.Body.Bytes())
				}
			}
		})
	}
}

// getTasksCreateHandler returns the handler for POST /tasks
// This function doesn't exist yet and MUST be implemented in Phase 3.6
func getTasksCreateHandler() http.Handler {
	// This will cause the test to fail - exactly what we want for TDD
	panic("getTasksCreateHandler not implemented - implement in Phase 3.6 (T063)")
}