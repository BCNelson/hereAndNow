package contract

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTasksDelete validates DELETE /tasks/{taskId} endpoint against OpenAPI spec
func TestTasksDelete(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		taskId         string
		expectedStatus int
	}{
		{
			name:           "Valid task deletion",
			token:          "valid-jwt-token",
			taskId:         "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Task not found",
			token:          "valid-jwt-token",
			taskId:         "00000000-0000-0000-0000-000000000000",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid UUID format",
			token:          "valid-jwt-token",
			taskId:         "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Request without token",
			token:          "",
			taskId:         "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+tt.taskId, nil)
			
			// Add authorization header if token provided
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// This will fail until we implement the handler
			handler := getTaskDeleteHandler() // This function doesn't exist yet
			handler.ServeHTTP(rr, req)

			// Validate response
			assert.Equal(t, tt.expectedStatus, rr.Code)
			
			// For successful deletion, body should be empty (204 No Content)
			if tt.expectedStatus == http.StatusNoContent {
				assert.Empty(t, rr.Body.String())
			}
		})
	}
}

// getTaskDeleteHandler returns the handler for DELETE /tasks/{taskId}
func getTaskDeleteHandler() http.Handler {
	panic("getTaskDeleteHandler not implemented - implement in Phase 3.6 (T066)")
}