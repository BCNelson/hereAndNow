package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTasksList validates GET /tasks endpoint against OpenAPI spec
func TestTasksList(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		queryParams    string
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid request with context filtering",
			token:          "valid-jwt-token",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				// Validate response schema
				assert.Contains(t, response, "tasks")
				assert.Contains(t, response, "total")
				assert.Contains(t, response, "context")

				// Validate tasks array
				tasks, ok := response["tasks"].([]interface{})
				require.True(t, ok, "tasks should be array")

				// If tasks exist, validate Task schema
				for _, taskInterface := range tasks {
					task, ok := taskInterface.(map[string]interface{})
					require.True(t, ok, "each task should be object")

					// Validate required Task fields
					assert.Contains(t, task, "id")
					assert.Contains(t, task, "title")
					assert.Contains(t, task, "status")
					assert.Contains(t, task, "created_at")
					assert.Contains(t, task, "creator_id")

					// Validate status enum
					if status, ok := task["status"].(string); ok {
						validStatuses := []string{"pending", "active", "completed", "cancelled", "blocked"}
						assert.Contains(t, validStatuses, status)
					}

					// Validate priority if present
					if priority, ok := task["priority"]; ok && priority != nil {
						pVal := priority.(float64)
						assert.GreaterOrEqual(t, pVal, 1.0)
						assert.LessOrEqual(t, pVal, 5.0)
					}
				}

				// Validate total is integer
				assert.IsType(t, float64(0), response["total"])

				// Validate context object
				context, ok := response["context"].(map[string]interface{})
				require.True(t, ok, "context should be object")
				assert.Contains(t, context, "user_id")
				assert.Contains(t, context, "timestamp")
			},
		},
		{
			name:           "Filter by status",
			token:          "valid-jwt-token",
			queryParams:    "status=pending",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				tasks := response["tasks"].([]interface{})
				for _, taskInterface := range tasks {
					task := taskInterface.(map[string]interface{})
					assert.Equal(t, "pending", task["status"])
				}
			},
		},
		{
			name:           "Pagination with limit and offset",
			token:          "valid-jwt-token",
			queryParams:    "limit=10&offset=5",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				tasks := response["tasks"].([]interface{})
				assert.LessOrEqual(t, len(tasks), 10) // Should respect limit
			},
		},
		{
			name:           "Show all tasks (no context filtering)",
			token:          "valid-jwt-token",
			queryParams:    "show_all=true",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				assert.Contains(t, response, "tasks")
				assert.Contains(t, response, "total")
			},
		},
		{
			name:           "Request without token",
			token:          "",
			queryParams:    "",
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
		{
			name:           "Invalid status filter",
			token:          "valid-jwt-token",
			queryParams:    "status=invalid_status",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				if err == nil {
					assert.Contains(t, response, "error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create URL with query parameters
			url := "/api/v1/tasks"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, url, nil)
			
			// Add authorization header if token provided
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// This will fail until we implement the handler
			// Call the handler (not implemented yet - this MUST fail)
			handler := getTasksListHandler() // This function doesn't exist yet
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

// getTasksListHandler returns the handler for GET /tasks
// This function doesn't exist yet and MUST be implemented in Phase 3.6
func getTasksListHandler() http.Handler {
	// This will cause the test to fail - exactly what we want for TDD
	panic("getTasksListHandler not implemented - implement in Phase 3.6 (T062)")
}