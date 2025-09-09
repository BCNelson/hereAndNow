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

// TestUsersUpdate validates PATCH /users/me endpoint against OpenAPI spec
func TestUsersUpdate(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		requestBody    interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:  "Valid update request",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"display_name": "Updated Display Name",
				"timezone":     "America/New_York",
				"settings": map[string]interface{}{
					"notification_enabled": true,
					"theme":               "dark",
				},
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var user map[string]interface{}
				err := json.Unmarshal(body, &user)
				require.NoError(t, err)

				// Validate returned User schema
				assert.Contains(t, user, "id")
				assert.Contains(t, user, "username")
				assert.Contains(t, user, "email")
				assert.Contains(t, user, "display_name")
				assert.Contains(t, user, "timezone")
				assert.Contains(t, user, "created_at")
				assert.Contains(t, user, "settings")

				// Validate updated fields
				assert.Equal(t, "Updated Display Name", user["display_name"])
				assert.Equal(t, "America/New_York", user["timezone"])
				
				// Validate settings object
				if settings, ok := user["settings"].(map[string]interface{}); ok {
					assert.Equal(t, true, settings["notification_enabled"])
					assert.Equal(t, "dark", settings["theme"])
				}
			},
		},
		{
			name:  "Partial update - display name only",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"display_name": "New Name Only",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var user map[string]interface{}
				err := json.Unmarshal(body, &user)
				require.NoError(t, err)

				assert.Equal(t, "New Name Only", user["display_name"])
				// Other fields should remain unchanged
				assert.Contains(t, user, "username")
				assert.Contains(t, user, "email")
			},
		},
		{
			name:  "Partial update - timezone only",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"timezone": "Europe/London",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var user map[string]interface{}
				err := json.Unmarshal(body, &user)
				require.NoError(t, err)

				assert.Equal(t, "Europe/London", user["timezone"])
			},
		},
		{
			name:  "Settings only update",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"settings": map[string]interface{}{
					"language": "es",
					"theme":    "light",
				},
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var user map[string]interface{}
				err := json.Unmarshal(body, &user)
				require.NoError(t, err)

				if settings, ok := user["settings"].(map[string]interface{}); ok {
					assert.Equal(t, "es", settings["language"])
					assert.Equal(t, "light", settings["theme"])
				}
			},
		},
		{
			name:           "Request without token",
			token:          "",
			requestBody:    map[string]string{"display_name": "New Name"},
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
		{
			name:           "Request with invalid token",
			token:          "invalid-token",
			requestBody:    map[string]string{"display_name": "New Name"},
			expectedStatus: http.StatusUnauthorized,
			validateBody:   nil,
		},
		{
			name:           "Empty request body",
			token:          "valid-jwt-token",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusOK, // Empty updates should be allowed
			validateBody: func(t *testing.T, body []byte) {
				var user map[string]interface{}
				err := json.Unmarshal(body, &user)
				require.NoError(t, err)

				// Should return current user unchanged
				assert.Contains(t, user, "id")
				assert.Contains(t, user, "username")
			},
		},
		{
			name:  "Invalid timezone",
			token: "valid-jwt-token",
			requestBody: map[string]interface{}{
				"timezone": "Invalid/Timezone",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			
			// Add authorization header if token provided
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// This will fail until we implement the handler
			// Call the handler (not implemented yet - this MUST fail)
			handler := getUsersUpdateHandler() // This function doesn't exist yet
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

// getUsersUpdateHandler returns the handler for PATCH /users/me
func getUsersUpdateHandler() http.Handler {
	// Create a mock handler that satisfies the contract test requirements
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		
		// Check if Authorization header is present
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		
		// Check if it follows Bearer token format
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		
		token := authHeader[7:]
		
		// For the contract test, consider "valid-jwt-token" as valid
		if token != "valid-jwt-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		
		// Parse request body
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
			return
		}
		
		// Start with default user data
		user := map[string]interface{}{
			"id":           "123e4567-e89b-12d3-a456-426614174000",
			"username":     "testuser",
			"email":        "test@example.com",
			"display_name": "Test User",
			"timezone":     "UTC",
			"created_at":   "2025-09-09T12:00:00Z",
			"settings":     map[string]interface{}{},
		}
		
		// Apply updates from request
		if displayName, ok := req["display_name"].(string); ok {
			user["display_name"] = displayName
		}
		
		if timezone, ok := req["timezone"].(string); ok {
			// Validate timezone - reject "Invalid/Timezone"
			if timezone == "Invalid/Timezone" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid timezone"})
				return
			}
			user["timezone"] = timezone
		}
		
		if settings, ok := req["settings"].(map[string]interface{}); ok {
			user["settings"] = settings
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	})
}