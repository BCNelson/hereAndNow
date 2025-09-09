package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUsersMe validates GET /users/me endpoint against OpenAPI spec
func TestUsersMe(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid request with token",
			token:          "valid-jwt-token",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var user map[string]interface{}
				err := json.Unmarshal(body, &user)
				require.NoError(t, err)

				// Validate User schema according to OpenAPI spec
				assert.Contains(t, user, "id")
				assert.Contains(t, user, "username")
				assert.Contains(t, user, "email")
				assert.Contains(t, user, "display_name")
				assert.Contains(t, user, "timezone")
				assert.Contains(t, user, "created_at")
				assert.Contains(t, user, "settings")

				// Validate field types
				assert.IsType(t, "", user["id"])
				assert.IsType(t, "", user["username"])
				assert.IsType(t, "", user["email"])
				assert.IsType(t, "", user["display_name"])
				assert.IsType(t, "", user["timezone"])
				assert.IsType(t, "", user["created_at"])
				
				// settings can be object or null
				if user["settings"] != nil {
					assert.IsType(t, map[string]interface{}{}, user["settings"])
				}

				// Validate email format (basic check)
				email, ok := user["email"].(string)
				if ok && email != "" {
					assert.Contains(t, email, "@")
				}

				// Validate UUID format for id (basic check)
				id, ok := user["id"].(string)
				if ok {
					assert.Len(t, id, 36) // UUID length with hyphens
				}
			},
		},
		{
			name:           "Request without token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
		{
			name:           "Request with invalid token",
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			
			// Add authorization header if token provided
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// This will fail until we implement the handler
			// Call the handler (not implemented yet - this MUST fail)
			handler := getUsersMeHandler() // This function doesn't exist yet
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

// getUsersMeHandler returns the handler for GET /users/me
// This function doesn't exist yet and MUST be implemented in Phase 3.6
func getUsersMeHandler() http.Handler {
	// This will cause the test to fail - exactly what we want for TDD
	panic("getUsersMeHandler not implemented - implement in Phase 3.6 (T060)")
}