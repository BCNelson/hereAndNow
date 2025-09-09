package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTasksGet validates GET /tasks/{taskId} endpoint against OpenAPI spec
func TestTasksGet(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		taskId         string
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid task retrieval",
			token:          "valid-jwt-token",
			taskId:         "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var task map[string]interface{}
				err := json.Unmarshal(body, &task)
				require.NoError(t, err)

				// Validate complete Task schema
				assert.Contains(t, task, "id")
				assert.Contains(t, task, "title")
				assert.Contains(t, task, "description")
				assert.Contains(t, task, "creator_id")
				assert.Contains(t, task, "status")
				assert.Contains(t, task, "priority")
				assert.Contains(t, task, "created_at")
				assert.Contains(t, task, "updated_at")
				assert.Contains(t, task, "locations")
				assert.Contains(t, task, "dependencies")

				// Validate task ID matches request
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", task["id"])

				// Validate status enum
				if status, ok := task["status"].(string); ok {
					validStatuses := []string{"pending", "active", "completed", "cancelled", "blocked"}
					assert.Contains(t, validStatuses, status)
				}

				// Validate priority range
				if priority, ok := task["priority"]; ok && priority != nil {
					pVal := priority.(float64)
					assert.GreaterOrEqual(t, pVal, 1.0)
					assert.LessOrEqual(t, pVal, 5.0)
				}

				// Validate locations array
				if locations, ok := task["locations"].([]interface{}); ok {
					for _, locInterface := range locations {
						location := locInterface.(map[string]interface{})
						assert.Contains(t, location, "id")
						assert.Contains(t, location, "name")
						assert.Contains(t, location, "latitude")
						assert.Contains(t, location, "longitude")
					}
				}

				// Validate dependencies array
				if dependencies, ok := task["dependencies"].([]interface{}); ok {
					for _, dep := range dependencies {
						// Should be UUID strings
						assert.IsType(t, "", dep)
						depId := dep.(string)
						assert.Len(t, depId, 36) // UUID format
					}
				}
			},
		},
		{
			name:           "Task not found",
			token:          "valid-jwt-token",
			taskId:         "00000000-0000-0000-0000-000000000000",
			expectedStatus: http.StatusNotFound,
			validateBody:   nil,
		},
		{
			name:           "Invalid UUID format",
			token:          "valid-jwt-token",
			taskId:         "invalid-uuid",
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
			taskId:         "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
		{
			name:           "Request with invalid token",
			token:          "invalid-token",
			taskId:         "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+tt.taskId, nil)
			
			// Add authorization header if token provided
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// This will fail until we implement the handler
			// Call the handler (not implemented yet - this MUST fail)
			handler := getTaskGetHandler() // This function doesn't exist yet
			handler.ServeHTTP(rr, req)

			// Validate response
			assert.Equal(t, tt.expectedStatus, rr.Code)
			
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
				if tt.validateBody != nil {
					tt.validateBody(t, rr.Body.Bytes())
				}
			}
		})
	}
}

// getTaskGetHandler returns the handler for GET /tasks/{taskId}
// This function doesn't exist yet and MUST be implemented in Phase 3.6
func getTaskGetHandler() http.Handler {
	// This will cause the test to fail - exactly what we want for TDD
	panic("getTaskGetHandler not implemented - implement in Phase 3.6 (T064)")
}