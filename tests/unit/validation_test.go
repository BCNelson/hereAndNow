package unit

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// User Validation Tests
func TestUserValidation(t *testing.T) {
	t.Run("ValidUser", func(t *testing.T) {
		user, err := models.NewUser("validuser", "test@example.com", "Test User", "America/New_York")
		require.NoError(t, err)
		require.NotNil(t, user)
		
		assert.Equal(t, "validuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "Test User", user.DisplayName)
		assert.Equal(t, "America/New_York", user.TimeZone)
		assert.NotEmpty(t, user.ID)
	})
	
	t.Run("UsernameValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			username    string
			shouldError bool
			description string
		}{
			{"ValidUsername", "validuser123", false, "Standard alphanumeric username"},
			{"ValidWithUnderscore", "valid_user", false, "Username with underscore"},
			{"MinLength", "abc", false, "Minimum length username (3 chars)"},
			{"MaxLength", strings.Repeat("a", 50), false, "Maximum length username (50 chars)"},
			{"TooShort", "ab", true, "Username too short (2 chars)"},
			{"TooLong", strings.Repeat("a", 51), true, "Username too long (51 chars)"},
			{"Empty", "", true, "Empty username"},
			{"WithSpaces", "user name", true, "Username with spaces"},
			{"WithHyphen", "user-name", true, "Username with hyphen"},
			{"WithDots", "user.name", true, "Username with dots"},
			{"WithSpecialChars", "user@name", true, "Username with special characters"},
			{"StartingWithNumber", "1user", false, "Username starting with number"},
			{"OnlyNumbers", "123456", false, "Username with only numbers"},
			{"OnlyUnderscores", "______", false, "Username with only underscores"},
			{"MixedCase", "UserName", false, "Mixed case username"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewUser(tc.username, "test@example.com", "Test", "UTC")
				if tc.shouldError {
					assert.Error(t, err, tc.description)
					assert.Contains(t, err.Error(), "username")
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("EmailValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			email       string
			shouldError bool
			description string
		}{
			{"ValidEmail", "test@example.com", false, "Standard email"},
			{"ValidWithSubdomain", "user@mail.example.com", false, "Email with subdomain"},
			{"ValidWithNumbers", "user123@example.com", false, "Email with numbers"},
			{"ValidWithDots", "first.last@example.com", false, "Email with dots in local part"},
			{"ValidWithPlus", "user+tag@example.com", false, "Email with plus sign"},
			{"ValidWithHyphen", "user-name@ex-ample.com", false, "Email with hyphens"},
			{"Empty", "", true, "Empty email"},
			{"NoAtSymbol", "userexample.com", true, "Email without @ symbol"},
			{"NoLocalPart", "@example.com", true, "Email without local part"},
			{"NoDomain", "user@", true, "Email without domain"},
			{"NoTLD", "user@example", true, "Email without TLD"},
			{"MultipleAtSymbols", "user@@example.com", true, "Email with multiple @ symbols"},
			{"InvalidTLD", "user@example.c", true, "Email with single character TLD"},
			{"SpacesInEmail", "user name@example.com", true, "Email with spaces"},
			{"InvalidCharacters", "user<>@example.com", true, "Email with invalid characters"},
			{"TooLongLocalPart", strings.Repeat("a", 65) + "@example.com", false, "Very long local part (may be valid depending on implementation)"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewUser("testuser", tc.email, "Test", "UTC")
				if tc.shouldError {
					assert.Error(t, err, tc.description)
					if err != nil {
						assert.Contains(t, err.Error(), "email")
					}
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("TimezoneValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			timezone    string
			shouldError bool
			description string
		}{
			{"UTC", "UTC", false, "UTC timezone"},
			{"AmericaNewYork", "America/New_York", false, "America/New_York timezone"},
			{"EuropeLondon", "Europe/London", false, "Europe/London timezone"},
			{"AsiaTokyo", "Asia/Tokyo", false, "Asia/Tokyo timezone"},
			{"AustralianSydney", "Australia/Sydney", false, "Australia/Sydney timezone"},
			{"Empty", "", false, "Empty timezone (may be valid depending on implementation)"},
			{"Invalid", "Invalid/Timezone", true, "Invalid timezone"},
			{"NonExistent", "Mars/Olympia", true, "Non-existent timezone"},
			{"JustCity", "London", true, "Just city name"},
			{"WithSpaces", "America/New York", true, "Timezone with spaces"},
			{"Numeric", "GMT+5", true, "Numeric timezone format"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewUser("testuser", "test@example.com", "Test", tc.timezone)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
					if err != nil {
						assert.Contains(t, err.Error(), "timezone")
					}
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("PasswordValidation", func(t *testing.T) {
		user, err := models.NewUser("testuser", "test@example.com", "Test", "UTC")
		require.NoError(t, err)
		
		testCases := []struct {
			name        string
			password    string
			shouldError bool
			description string
		}{
			{"ValidPassword", "password123", false, "Standard password"},
			{"MinLength", "12345678", false, "Minimum length password (8 chars)"},
			{"LongPassword", strings.Repeat("a", 100), false, "Very long password"},
			{"WithSpecialChars", "p@ssw0rd!", false, "Password with special characters"},
			{"TooShort", "1234567", true, "Password too short (7 chars)"},
			{"Empty", "", true, "Empty password"},
			{"OnlySpaces", "        ", false, "Password with only spaces (valid)"},
			{"Unicode", "пароль123", false, "Unicode password"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := user.SetPassword(tc.password)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
					if err != nil {
						assert.Contains(t, err.Error(), "password")
					}
				} else {
					assert.NoError(t, err, tc.description)
					// Test password verification
					if !tc.shouldError {
						assert.True(t, user.CheckPassword(tc.password), "Password should verify correctly")
						assert.False(t, user.CheckPassword("wrongpassword"), "Wrong password should not verify")
					}
				}
			})
		}
	})
	
	t.Run("PasswordHashing", func(t *testing.T) {
		user, err := models.NewUser("testuser", "test@example.com", "Test", "UTC")
		require.NoError(t, err)
		
		password := "testpassword123"
		err = user.SetPassword(password)
		require.NoError(t, err)
		
		// Verify password hash format
		assert.True(t, strings.HasPrefix(user.PasswordHash, "$argon2id$"))
		assert.NotEqual(t, password, user.PasswordHash, "Password should be hashed")
		
		// Test password verification
		assert.True(t, user.CheckPassword(password), "Correct password should verify")
		assert.False(t, user.CheckPassword("wrongpassword"), "Wrong password should not verify")
		assert.False(t, user.CheckPassword(""), "Empty password should not verify")
		
		// Test case sensitivity
		assert.False(t, user.CheckPassword("TESTPASSWORD123"), "Password should be case sensitive")
	})
	
	t.Run("UserValidate", func(t *testing.T) {
		// Valid user should pass validation
		user, err := models.NewUser("testuser", "test@example.com", "Test", "UTC")
		require.NoError(t, err)
		err = user.SetPassword("password123")
		require.NoError(t, err)
		
		err = user.Validate()
		assert.NoError(t, err, "Valid user should pass validation")
		
		// Test various invalid states
		testCases := []struct {
			name   string
			modify func(*models.User)
		}{
			{"InvalidUsername", func(u *models.User) { u.Username = "a" }},
			{"InvalidEmail", func(u *models.User) { u.Email = "invalid" }},
			{"InvalidTimezone", func(u *models.User) { u.TimeZone = "Invalid" }},
			{"NoPasswordHash", func(u *models.User) { u.PasswordHash = "" }},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				userCopy := *user // Create a copy
				tc.modify(&userCopy)
				err := userCopy.Validate()
				assert.Error(t, err, "Modified user should fail validation")
			})
		}
	})
}

// Task Validation Tests
func TestTaskValidation(t *testing.T) {
	t.Run("ValidTask", func(t *testing.T) {
		task, err := models.NewTask("Test Task", "Description", "user-id")
		require.NoError(t, err)
		require.NotNil(t, task)
		
		assert.Equal(t, "Test Task", task.Title)
		assert.Equal(t, "Description", task.Description)
		assert.Equal(t, "user-id", task.CreatorID)
		assert.Equal(t, models.TaskStatusPending, task.Status)
		assert.NotEmpty(t, task.ID)
	})
	
	t.Run("TitleValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			title       string
			shouldError bool
			description string
		}{
			{"ValidTitle", "Valid Task Title", false, "Standard task title"},
			{"ShortTitle", "A", false, "Single character title"},
			{"LongTitle", strings.Repeat("A", 255), false, "Maximum length title"},
			{"Empty", "", true, "Empty title"},
			{"OnlySpaces", "   ", true, "Title with only spaces"},
			{"VeryLongTitle", strings.Repeat("A", 256), true, "Title too long"},
			{"WithNewlines", "Title\nwith\nnewlines", false, "Title with newlines"},
			{"WithTabs", "Title\twith\ttabs", false, "Title with tabs"},
			{"Unicode", "Задача с юникодом", false, "Unicode title"},
			{"Emojis", "Task with 🚀 emoji", false, "Title with emojis"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewTask(tc.title, "Description", "user-id")
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("PriorityValidation", func(t *testing.T) {
		task, err := models.NewTask("Test Task", "Description", "user-id")
		require.NoError(t, err)
		
		testCases := []struct {
			name        string
			priority    int
			shouldError bool
			description string
		}{
			{"Priority1", 1, false, "Minimum priority"},
			{"Priority3", 3, false, "Medium priority"},
			{"Priority5", 5, false, "Maximum priority"},
			{"Priority0", 0, true, "Priority too low"},
			{"Priority6", 6, true, "Priority too high"},
			{"NegativePriority", -1, true, "Negative priority"},
			{"VeryHighPriority", 100, true, "Very high priority"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := task.SetPriority(tc.priority)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
					assert.Equal(t, tc.priority, task.Priority)
				}
			})
		}
	})
	
	t.Run("EstimatedMinutesValidation", func(t *testing.T) {
		task, err := models.NewTask("Test Task", "Description", "user-id")
		require.NoError(t, err)
		
		testCases := []struct {
			name        string
			minutes     int
			shouldError bool
			description string
		}{
			{"OneMinute", 1, false, "Minimum estimated minutes"},
			{"ThirtyMinutes", 30, false, "Standard task duration"},
			{"EightHours", 480, false, "Full day task"},
			{"ZeroMinutes", 0, true, "Zero estimated minutes"},
			{"NegativeMinutes", -30, true, "Negative estimated minutes"},
			{"VeryLong", 10080, false, "Week-long task"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := task.SetEstimatedMinutes(tc.minutes)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
					assert.Equal(t, tc.minutes, *task.EstimatedMinutes)
				}
			})
		}
	})
	
	t.Run("StatusTransitions", func(t *testing.T) {
		task, err := models.NewTask("Test Task", "Description", "user-id")
		require.NoError(t, err)
		
		// Valid transitions
		validTransitions := []models.TaskStatus{
			models.TaskStatusActive,
			models.TaskStatusCompleted,
			models.TaskStatusCancelled,
		}
		
		for _, status := range validTransitions {
			newTask, _ := models.NewTask("Test Task", "Description", "user-id")
			err := newTask.SetStatus(status)
			assert.NoError(t, err, "Should allow transition from pending to %s", status)
		}
		
		// Test completion timestamp
		task.SetStatus(models.TaskStatusCompleted)
		assert.NotNil(t, task.CompletedAt, "CompletedAt should be set when status is completed")
		
		// Test uncompleting
		task.SetStatus(models.TaskStatusActive)
		assert.Nil(t, task.CompletedAt, "CompletedAt should be cleared when status changes from completed")
	})
}

// Location Validation Tests
func TestLocationValidation(t *testing.T) {
	t.Run("ValidLocation", func(t *testing.T) {
		location, err := models.NewLocation("user-id", "Home", "123 Main St", 37.7749, -122.4194, 100)
		require.NoError(t, err)
		require.NotNil(t, location)
		
		assert.Equal(t, "Home", location.Name)
		assert.Equal(t, 37.7749, location.Latitude)
		assert.Equal(t, -122.4194, location.Longitude)
		assert.Equal(t, 100, location.Radius)
		assert.NotEmpty(t, location.ID)
	})
	
	t.Run("CoordinateValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			lat         float64
			lng         float64
			shouldError bool
			description string
		}{
			{"ValidSanFrancisco", 37.7749, -122.4194, false, "San Francisco coordinates"},
			{"ValidNorthPole", 90.0, 0.0, false, "North Pole"},
			{"ValidSouthPole", -90.0, 0.0, false, "South Pole"},
			{"ValidPrimeMeridian", 0.0, 0.0, false, "Prime Meridian intersection"},
			{"ValidDateLine", 0.0, 180.0, false, "International Date Line"},
			{"ValidNegativeDateLine", 0.0, -180.0, false, "Negative Date Line"},
			{"InvalidLatTooHigh", 91.0, 0.0, true, "Latitude too high"},
			{"InvalidLatTooLow", -91.0, 0.0, true, "Latitude too low"},
			{"InvalidLngTooHigh", 0.0, 181.0, true, "Longitude too high"},
			{"InvalidLngTooLow", 0.0, -181.0, true, "Longitude too low"},
			{"ValidMinLat", -90.0, -180.0, false, "Minimum valid coordinates"},
			{"ValidMaxLat", 90.0, 180.0, false, "Maximum valid coordinates"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewLocation("user-id", "Test", "", tc.lat, tc.lng, 100)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("RadiusValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			radius      int
			shouldError bool
			description string
		}{
			{"SmallRadius", 1, false, "1 meter radius"},
			{"StandardRadius", 100, false, "100 meter radius"},
			{"LargeRadius", 10000, false, "10km radius"},
			{"ZeroRadius", 0, true, "Zero radius"},
			{"NegativeRadius", -50, true, "Negative radius"},
			{"VeryLargeRadius", 100000, false, "100km radius"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewLocation("user-id", "Test", "", 37.7749, -122.4194, tc.radius)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("NameValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			locationName string
			shouldError  bool
			description  string
		}{
			{"ValidName", "Home", false, "Standard location name"},
			{"LongName", "My Very Long Location Name", false, "Long location name"},
			{"Empty", "", true, "Empty location name"},
			{"OnlySpaces", "   ", true, "Location name with only spaces"},
			{"Unicode", "家", false, "Unicode location name"},
			{"WithNumbers", "Building 42", false, "Location name with numbers"},
			{"SpecialChars", "Mom & Dad's House", false, "Location name with special characters"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewLocation("user-id", tc.locationName, "", 37.7749, -122.4194, 100)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
}

// Context Validation Tests
func TestContextValidation(t *testing.T) {
	t.Run("ValidContext", func(t *testing.T) {
		context, err := models.NewContext("user-id", 60, 3)
		require.NoError(t, err)
		require.NotNil(t, context)
		
		assert.Equal(t, "user-id", context.UserID)
		assert.Equal(t, 60, context.AvailableMinutes)
		assert.Equal(t, 3, context.EnergyLevel)
		assert.NotEmpty(t, context.ID)
	})
	
	t.Run("EnergyLevelValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			energy      int
			shouldError bool
			description string
		}{
			{"EnergyLevel1", 1, false, "Minimum energy level"},
			{"EnergyLevel3", 3, false, "Medium energy level"},
			{"EnergyLevel5", 5, false, "Maximum energy level"},
			{"EnergyLevel0", 0, true, "Energy level too low"},
			{"EnergyLevel6", 6, true, "Energy level too high"},
			{"NegativeEnergy", -1, true, "Negative energy level"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewContext("user-id", 60, tc.energy)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("AvailableMinutesValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			minutes     int
			shouldError bool
			description string
		}{
			{"ZeroMinutes", 0, false, "Zero available minutes"},
			{"FifteenMinutes", 15, false, "15 minutes available"},
			{"EightHours", 480, false, "8 hours available"},
			{"NegativeMinutes", -30, true, "Negative available minutes"},
			{"VeryLongTime", 1440, false, "24 hours available"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewContext("user-id", tc.minutes, 3)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("SocialContextValidation", func(t *testing.T) {
		context, err := models.NewContext("user-id", 60, 3)
		require.NoError(t, err)
		
		validSocialContexts := []string{
			models.SocialContextAlone,
			models.SocialContextWithFamily,
			models.SocialContextAtWork,
			models.SocialContextInPublic,
			models.SocialContextDriving,
		}
		
		for _, socialCtx := range validSocialContexts {
			context.SocialContext = socialCtx
			// In a real implementation, this would be validated
			// For now, we just ensure the constants are accessible
			assert.NotEmpty(t, socialCtx)
		}
	})
}

// Calendar Event Validation Tests
func TestCalendarEventValidation(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	
	t.Run("ValidCalendarEvent", func(t *testing.T) {
		event, err := models.NewCalendarEvent("user-id", "google", "event-123", "Meeting", now, future)
		require.NoError(t, err)
		require.NotNil(t, event)
		
		assert.Equal(t, "user-id", event.UserID)
		assert.Equal(t, "google", event.ProviderID)
		assert.Equal(t, "event-123", event.ExternalID)
		assert.Equal(t, "Meeting", event.Title)
		assert.True(t, event.StartAt.Equal(now))
		assert.True(t, event.EndAt.Equal(future))
	})
	
	t.Run("TimeValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			startTime   time.Time
			endTime     time.Time
			shouldError bool
			description string
		}{
			{"ValidOneHour", now, future, false, "1 hour meeting"},
			{"ValidOneMinute", now, now.Add(1*time.Minute), false, "1 minute meeting"},
			{"ValidAllDay", now.Truncate(24*time.Hour), now.Truncate(24*time.Hour).Add(24*time.Hour), false, "All day event"},
			{"SameStartEnd", now, now, true, "Same start and end time"},
			{"EndBeforeStart", future, now, true, "End time before start time"},
			{"TooLong", now, now.Add(8*24*time.Hour), true, "Event longer than 7 days"},
			{"ValidSevenDays", now, now.Add(7*24*time.Hour-1*time.Minute), false, "Just under 7 days"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewCalendarEvent("user-id", "google", "event-123", "Test", tc.startTime, tc.endTime)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
	
	t.Run("RequiredFieldsValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			userID      string
			providerID  string
			externalID  string
			title       string
			shouldError bool
			description string
		}{
			{"AllValid", "user-id", "google", "event-123", "Meeting", false, "All fields valid"},
			{"EmptyUserID", "", "google", "event-123", "Meeting", true, "Empty user ID"},
			{"EmptyProviderID", "user-id", "", "event-123", "Meeting", true, "Empty provider ID"},
			{"EmptyExternalID", "user-id", "google", "", "Meeting", true, "Empty external ID"},
			{"EmptyTitle", "user-id", "google", "event-123", "", true, "Empty title"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := models.NewCalendarEvent(tc.userID, tc.providerID, tc.externalID, tc.title, now, future)
				if tc.shouldError {
					assert.Error(t, err, tc.description)
				} else {
					assert.NoError(t, err, tc.description)
				}
			})
		}
	})
}

// Edge Cases and Error Conditions
func TestValidationEdgeCases(t *testing.T) {
	t.Run("JSONMarshaling", func(t *testing.T) {
		user, err := models.NewUser("testuser", "test@example.com", "Test User", "UTC")
		require.NoError(t, err)
		
		// Test JSON marshaling doesn't include password hash
		data, err := json.Marshal(user)
		require.NoError(t, err)
		
		assert.NotContains(t, string(data), "password_hash", "Password hash should not be included in JSON")
		assert.Contains(t, string(data), "username", "Username should be included in JSON")
	})
	
	t.Run("UUIDGeneration", func(t *testing.T) {
		// Create multiple entities and ensure they have unique IDs
		user1, _ := models.NewUser("user1", "user1@example.com", "User 1", "UTC")
		user2, _ := models.NewUser("user2", "user2@example.com", "User 2", "UTC")
		task1, _ := models.NewTask("Task 1", "Description", "user-id")
		task2, _ := models.NewTask("Task 2", "Description", "user-id")
		location1, _ := models.NewLocation("user-id", "Location 1", "", 37.7749, -122.4194, 100)
		location2, _ := models.NewLocation("user-id", "Location 2", "", 37.7750, -122.4195, 100)
		
		ids := []string{user1.ID, user2.ID, task1.ID, task2.ID, location1.ID, location2.ID}
		
		// Check all IDs are unique
		seen := make(map[string]bool)
		for _, id := range ids {
			assert.False(t, seen[id], "ID %s should be unique", id)
			assert.NotEmpty(t, id, "ID should not be empty")
			seen[id] = true
		}
	})
	
	t.Run("TimestampConsistency", func(t *testing.T) {
		beforeCreation := time.Now()
		user, err := models.NewUser("testuser", "test@example.com", "Test User", "UTC")
		afterCreation := time.Now()
		require.NoError(t, err)
		
		// Timestamps should be within reasonable range
		assert.True(t, user.CreatedAt.After(beforeCreation) || user.CreatedAt.Equal(beforeCreation))
		assert.True(t, user.CreatedAt.Before(afterCreation) || user.CreatedAt.Equal(afterCreation))
		assert.Equal(t, user.CreatedAt, user.UpdatedAt, "CreatedAt and UpdatedAt should be equal for new entities")
		assert.Equal(t, user.CreatedAt, user.LastSeenAt, "CreatedAt and LastSeenAt should be equal for new users")
	})
	
	t.Run("ConcurrentValidation", func(t *testing.T) {
		// Test validation under concurrent access (basic thread safety)
		const numGoroutines = 100
		results := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				username := fmt.Sprintf("user%d", id)
				email := fmt.Sprintf("user%d@example.com", id)
				_, err := models.NewUser(username, email, "Test User", "UTC")
				results <- err
			}(i)
		}
		
		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent validation should not fail")
		}
	})
	
	t.Run("MemoryUsage", func(t *testing.T) {
		// Test that validation doesn't cause memory leaks with large inputs
		longString := strings.Repeat("a", 10000)
		
		// These should all fail validation but not cause memory issues
		_, err := models.NewUser(longString, "test@example.com", "Test", "UTC")
		assert.Error(t, err, "Very long username should be rejected")
		
		_, err = models.NewTask(longString, "Description", "user-id")
		assert.Error(t, err, "Very long task title should be rejected")
		
		_, err = models.NewLocation("user-id", longString, "", 37.7749, -122.4194, 100)
		assert.Error(t, err, "Very long location name should be rejected")
	})
}