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

// TestAuthLoginContract validates POST /auth/login endpoint against OpenAPI spec
func TestAuthLoginContract(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name: "Valid login request",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "testpassword",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				require.NoError(t, err)

				// Validate AuthResponse schema
				assert.Contains(t, response, "token")
				assert.Contains(t, response, "user")
				assert.Contains(t, response, "expires_at")

				// Validate token is string
				assert.IsType(t, "", response["token"])
				assert.NotEmpty(t, response["token"])

				// Validate user object
				user, ok := response["user"].(map[string]interface{})
				require.True(t, ok, "user should be an object")
				
				assert.Contains(t, user, "id")
				assert.Contains(t, user, "username")
				assert.Contains(t, user, "email")
				assert.Contains(t, user, "display_name")
				assert.Contains(t, user, "timezone")
				assert.Contains(t, user, "created_at")

				// Validate expires_at is RFC3339 datetime
				assert.IsType(t, "", response["expires_at"])
				assert.NotEmpty(t, response["expires_at"])
			},
		},
		{
			name: "Invalid credentials",
			requestBody: map[string]string{
				"username": "invalid",
				"password": "wrong",
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body []byte) {
				// Should return error response
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				if err == nil {
					// If JSON response, should contain error
					assert.Contains(t, response, "error")
				}
			},
		},
		{
			name: "Missing username",
			requestBody: map[string]string{
				"password": "testpassword",
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
			name: "Missing password",
			requestBody: map[string]string{
				"username": "testuser",
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
			name:           "Empty request body",
			requestBody:    map[string]string{},
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
			// Prepare request body
			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// This will fail until we implement the handler
			// Call the handler (not implemented yet - this MUST fail)
			handler := getAuthLoginHandler() // This function doesn't exist yet
			handler.ServeHTTP(rr, req)

			// Validate response
			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			if tt.validateBody != nil {
				tt.validateBody(t, rr.Body.Bytes())
			}
		})
	}
}

// getAuthLoginHandler returns the handler for POST /auth/login
// This function doesn't exist yet and MUST be implemented in Phase 3.6
func getAuthLoginHandler() http.Handler {
	// This will cause the test to fail - exactly what we want for TDD
	panic("getAuthLoginHandler not implemented - implement in Phase 3.6 (T058)")
}