package unit

import (
	"crypto/rand"
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bcnelson/hereAndNow/internal/auth"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
)

// UUID Generation Tests
func TestUUIDGeneration(t *testing.T) {
	t.Run("UUIDUniqueness", func(t *testing.T) {
		const numUUIDs = 10000
		uuids := make(map[string]bool, numUUIDs)
		
		for i := 0; i < numUUIDs; i++ {
			id := uuid.New().String()
			
			// Check format
			assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), id)
			
			// Check uniqueness
			assert.False(t, uuids[id], "UUID %s should be unique", id)
			uuids[id] = true
		}
		
		assert.Equal(t, numUUIDs, len(uuids), "All UUIDs should be unique")
	})
	
	t.Run("UUIDVersion", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			id := uuid.New()
			
			// Check it's UUID v4 (random)
			assert.Equal(t, byte(4), id.Version(), "Should generate UUID v4")
			assert.Equal(t, uuid.RFC4122, id.Variant(), "Should use RFC4122 variant")
		}
	})
	
	t.Run("UUIDStringRepresentation", func(t *testing.T) {
		id := uuid.New()
		str := id.String()
		
		// Check length
		assert.Equal(t, 36, len(str), "UUID string should be 36 characters")
		
		// Check format (8-4-4-4-12)
		parts := strings.Split(str, "-")
		assert.Len(t, parts, 5, "UUID should have 5 parts separated by hyphens")
		assert.Len(t, parts[0], 8, "First part should be 8 characters")
		assert.Len(t, parts[1], 4, "Second part should be 4 characters")
		assert.Len(t, parts[2], 4, "Third part should be 4 characters")
		assert.Len(t, parts[3], 4, "Fourth part should be 4 characters")
		assert.Len(t, parts[4], 12, "Fifth part should be 12 characters")
		
		// All should be lowercase hex
		for _, part := range parts {
			assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]+$`), part)
		}
	})
	
	t.Run("UUIDFromString", func(t *testing.T) {
		original := uuid.New()
		str := original.String()
		
		parsed, err := uuid.Parse(str)
		require.NoError(t, err)
		assert.Equal(t, original, parsed)
		
		// Test invalid UUIDs
		invalidUUIDs := []string{
			"",
			"not-a-uuid",
			"12345678-1234-1234-1234-12345678901", // too short
			"12345678-1234-1234-1234-1234567890123", // too long
			"xxxxxxxx-1234-1234-1234-123456789012", // invalid hex
			"12345678:1234:1234:1234:123456789012", // wrong separators
		}
		
		for _, invalid := range invalidUUIDs {
			_, err := uuid.Parse(invalid)
			assert.Error(t, err, "Should fail to parse invalid UUID: %s", invalid)
		}
	})
}

// Password Hashing Tests
func TestPasswordHashing(t *testing.T) {
	t.Run("Argon2HashingBasics", func(t *testing.T) {
		password := "testpassword123"
		salt := []byte("testsalt12345678") // 16 bytes
		
		// Test with different parameters
		testCases := []struct {
			name     string
			time     uint32
			memory   uint32
			threads  uint8
			keyLen   uint32
		}{
			{"LowSecurity", 1, 32 * 1024, 1, 32},
			{"MediumSecurity", 2, 64 * 1024, 2, 32},
			{"HighSecurity", 3, 128 * 1024, 4, 32},
			{"DefaultSecurity", 1, 64 * 1024, 4, 32}, // Used in our implementation
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				hash := argon2.IDKey([]byte(password), salt, tc.time, tc.memory, tc.threads, tc.keyLen)
				
				assert.Equal(t, int(tc.keyLen), len(hash), "Hash should have correct length")
				assert.NotEqual(t, password, string(hash), "Hash should not equal original password")
				
				// Same inputs should produce same hash
				hash2 := argon2.IDKey([]byte(password), salt, tc.time, tc.memory, tc.threads, tc.keyLen)
				assert.Equal(t, hash, hash2, "Same inputs should produce same hash")
				
				// Different password should produce different hash
				hash3 := argon2.IDKey([]byte("differentpassword"), salt, tc.time, tc.memory, tc.threads, tc.keyLen)
				assert.NotEqual(t, hash, hash3, "Different passwords should produce different hashes")
			})
		}
	})
	
	t.Run("SaltImportance", func(t *testing.T) {
		password := "samepassword"
		salt1 := []byte("salt1234567890ab")
		salt2 := []byte("salt1234567890cd") // Only last 2 chars different
		
		hash1 := argon2.IDKey([]byte(password), salt1, 1, 64*1024, 4, 32)
		hash2 := argon2.IDKey([]byte(password), salt2, 1, 64*1024, 4, 32)
		
		assert.NotEqual(t, hash1, hash2, "Different salts should produce different hashes even with same password")
	})
	
	t.Run("UserPasswordHashing", func(t *testing.T) {
		user, err := models.NewUser("testuser", "test@example.com", "Test User", "UTC")
		require.NoError(t, err)
		
		password := "mypassword123"
		err = user.SetPassword(password)
		require.NoError(t, err)
		
		// Verify hash format
		assert.True(t, strings.HasPrefix(user.PasswordHash, "$argon2id$v=19$"))
		assert.Contains(t, user.PasswordHash, "$m=65536,t=1,p=4$")
		
		// Verify password checking
		assert.True(t, user.CheckPassword(password), "Correct password should verify")
		assert.False(t, user.CheckPassword("wrongpassword"), "Wrong password should not verify")
		assert.False(t, user.CheckPassword(""), "Empty password should not verify")
		
		// Test case sensitivity
		assert.False(t, user.CheckPassword("MYPASSWORD123"), "Password should be case sensitive")
		
		// Test with special characters
		specialPassword := "p@ssw0rd!#$%"
		err = user.SetPassword(specialPassword)
		require.NoError(t, err)
		assert.True(t, user.CheckPassword(specialPassword))
	})
	
	t.Run("PasswordHashConsistency", func(t *testing.T) {
		// Multiple users with same password should have different hashes
		password := "commonpassword"
		users := make([]*models.User, 10)
		
		for i := 0; i < 10; i++ {
			user, err := models.NewUser(fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@example.com", i), "User", "UTC")
			require.NoError(t, err)
			
			err = user.SetPassword(password)
			require.NoError(t, err)
			users[i] = user
		}
		
		// All should be able to verify the password
		for i, user := range users {
			assert.True(t, user.CheckPassword(password), "User %d should verify password", i)
		}
		
		// But all should have different hashes (due to different salts)
		hashes := make(map[string]bool)
		for i, user := range users {
			assert.False(t, hashes[user.PasswordHash], "User %d should have unique hash", i)
			hashes[user.PasswordHash] = true
		}
		assert.Equal(t, 10, len(hashes), "All hashes should be unique")
	})
	
	t.Run("PasswordHashTiming", func(t *testing.T) {
		user, err := models.NewUser("testuser", "test@example.com", "Test User", "UTC")
		require.NoError(t, err)
		
		password := "timingtest"
		
		// Measure hashing time
		start := time.Now()
		err = user.SetPassword(password)
		hashingTime := time.Since(start)
		require.NoError(t, err)
		
		// Measure verification time
		start = time.Now()
		result := user.CheckPassword(password)
		verificationTime := time.Since(start)
		
		assert.True(t, result)
		
		t.Logf("Password hashing time: %v", hashingTime)
		t.Logf("Password verification time: %v", verificationTime)
		
		// Should be reasonable but not too fast (security vs performance)
		assert.Less(t, hashingTime, 1*time.Second, "Hashing should complete within 1 second")
		assert.Greater(t, hashingTime, 1*time.Millisecond, "Hashing should take some time for security")
	})
}

// Distance Calculation Tests
func TestDistanceCalculation(t *testing.T) {
	t.Run("HaversineDistanceBasics", func(t *testing.T) {
		// Test known distances
		testCases := []struct {
			name           string
			lat1, lng1     float64
			lat2, lng2     float64
			expectedKM     float64
			tolerance      float64
		}{
			{
				name:       "SamePoint",
				lat1:       37.7749, lng1: -122.4194,
				lat2:       37.7749, lng2: -122.4194,
				expectedKM: 0.0,
				tolerance:  0.001,
			},
			{
				name:       "NewYorkToLondon",
				lat1:       40.7128, lng1: -74.0060,  // New York
				lat2:       51.5074, lng2: -0.1278,   // London
				expectedKM: 5585,                     // Approximate distance
				tolerance:  100,                      // 100km tolerance
			},
			{
				name:       "SanFranciscoToLosAngeles",
				lat1:       37.7749, lng1: -122.4194, // San Francisco
				lat2:       34.0522, lng2: -118.2437, // Los Angeles
				expectedKM: 559,                      // Approximate distance
				tolerance:  50,                       // 50km tolerance
			},
			{
				name:       "TokyoToSydney",
				lat1:       35.6762, lng1: 139.6503, // Tokyo
				lat2:       -33.8688, lng2: 151.2093, // Sydney
				expectedKM: 7823,                     // Approximate distance
				tolerance:  100,                      // 100km tolerance
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test location model method
				location, err := models.NewLocation("user-id", "Test Location", "", tc.lat1, tc.lng1, 100)
				require.NoError(t, err)
				
				distanceMeters := location.DistanceFrom(tc.lat2, tc.lng2)
				distanceKM := distanceMeters / 1000
				
				assert.InDelta(t, tc.expectedKM, distanceKM, tc.tolerance, 
					"Distance from %s should be approximately %.0f km, got %.2f km", 
					tc.name, tc.expectedKM, distanceKM)
			})
		}
	})
	
	t.Run("DistanceEdgeCases", func(t *testing.T) {
		location, err := models.NewLocation("user-id", "Test Location", "", 0, 0, 100)
		require.NoError(t, err)
		
		testCases := []struct {
			name        string
			lat, lng    float64
			description string
		}{
			{"Equator180", 0, 180, "Point on equator at 180° longitude"},
			{"Equator-180", 0, -180, "Point on equator at -180° longitude"},
			{"NorthPole", 90, 0, "North Pole"},
			{"SouthPole", -90, 0, "South Pole"},
			{"PrimeMeridian", 0, 0, "Prime Meridian intersection"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				distance := location.DistanceFrom(tc.lat, tc.lng)
				
				// Distance should be valid (not NaN or infinite)
				assert.False(t, math.IsNaN(distance), "Distance should not be NaN for %s", tc.description)
				assert.False(t, math.IsInf(distance, 0), "Distance should not be infinite for %s", tc.description)
				assert.GreaterOrEqual(t, distance, 0.0, "Distance should be non-negative for %s", tc.description)
				
				// Distance should be reasonable (not more than half circumference of Earth)
				maxDistance := 20015.086 * 1000 // Half circumference in meters
				assert.LessOrEqual(t, distance, maxDistance, "Distance should not exceed half Earth circumference for %s", tc.description)
			})
		}
	})
	
	t.Run("DistanceSymmetry", func(t *testing.T) {
		// Distance from A to B should equal distance from B to A
		loc1, err := models.NewLocation("user-id", "Location 1", "", 37.7749, -122.4194, 100)
		require.NoError(t, err)
		
		loc2, err := models.NewLocation("user-id", "Location 2", "", 40.7128, -74.0060, 100)
		require.NoError(t, err)
		
		distance1to2 := loc1.DistanceFrom(40.7128, -74.0060)
		distance2to1 := loc2.DistanceFrom(37.7749, -122.4194)
		
		assert.InDelta(t, distance1to2, distance2to1, 0.001, 
			"Distance should be symmetric: A to B should equal B to A")
	})
	
	t.Run("SmallDistanceAccuracy", func(t *testing.T) {
		// Test very small distances (within a city)
		baseLat, baseLng := 37.7749, -122.4194
		
		// Move ~100 meters north (approximately 0.001 degrees latitude)
		targetLat := baseLat + 0.001
		
		location, err := models.NewLocation("user-id", "Base Location", "", baseLat, baseLng, 100)
		require.NoError(t, err)
		
		distance := location.DistanceFrom(targetLat, baseLng)
		
		// Should be approximately 111 meters (1 degree latitude ≈ 111 km)
		assert.InDelta(t, 111, distance, 20, "Small distance calculation should be accurate")
	})
}

// JWT Token Tests
func TestJWTTokenGeneration(t *testing.T) {
	t.Run("JWTTokenBasics", func(t *testing.T) {
		jwtService := auth.NewJWTService("test-secret-key-32-chars-long!!")
		
		userID := "test-user-id"
		expiresAt := time.Now().Add(1 * time.Hour)
		
		token, err := jwtService.GenerateToken(userID, expiresAt)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		
		// JWT should have 3 parts separated by dots
		parts := strings.Split(token, ".")
		assert.Len(t, parts, 3, "JWT should have 3 parts: header.payload.signature")
		
		// Each part should be non-empty
		for i, part := range parts {
			assert.NotEmpty(t, part, "JWT part %d should not be empty", i)
		}
	})
	
	t.Run("JWTTokenValidation", func(t *testing.T) {
		jwtService := auth.NewJWTService("test-secret-key-32-chars-long!!")
		
		userID := "test-user-id"
		expiresAt := time.Now().Add(1 * time.Hour)
		
		token, err := jwtService.GenerateToken(userID, expiresAt)
		require.NoError(t, err)
		
		// Validate the token
		claims, err := jwtService.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		
		// Test with invalid token
		_, err = jwtService.ValidateToken("invalid.token.here")
		assert.Error(t, err, "Invalid token should fail validation")
		
		// Test with empty token
		_, err = jwtService.ValidateToken("")
		assert.Error(t, err, "Empty token should fail validation")
	})
	
	t.Run("JWTTokenExpiration", func(t *testing.T) {
		jwtService := auth.NewJWTService("test-secret-key-32-chars-long!!")
		
		userID := "test-user-id"
		
		// Create expired token
		expiredTime := time.Now().Add(-1 * time.Hour)
		expiredToken, err := jwtService.GenerateToken(userID, expiredTime)
		require.NoError(t, err)
		
		// Should fail validation due to expiration
		_, err = jwtService.ValidateToken(expiredToken)
		assert.Error(t, err, "Expired token should fail validation")
		
		// Create valid token
		validTime := time.Now().Add(1 * time.Hour)
		validToken, err := jwtService.GenerateToken(userID, validTime)
		require.NoError(t, err)
		
		// Should pass validation
		claims, err := jwtService.ValidateToken(validToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})
	
	t.Run("JWTSecretKeyImportance", func(t *testing.T) {
		service1 := auth.NewJWTService("secret-key-1-32-characters-long")
		service2 := auth.NewJWTService("secret-key-2-32-characters-long")
		
		userID := "test-user-id"
		expiresAt := time.Now().Add(1 * time.Hour)
		
		token1, err := service1.GenerateToken(userID, expiresAt)
		require.NoError(t, err)
		
		// Token generated with one key should not validate with another key
		_, err = service2.ValidateToken(token1)
		assert.Error(t, err, "Token from different key should not validate")
		
		// But should validate with the correct key
		_, err = service1.ValidateToken(token1)
		assert.NoError(t, err, "Token should validate with correct key")
	})
}

// Random Number Generation Tests
func TestRandomGeneration(t *testing.T) {
	t.Run("CryptoRandomBytes", func(t *testing.T) {
		// Test generating random bytes
		sizes := []int{8, 16, 32, 64, 128}
		
		for _, size := range sizes {
			bytes1 := make([]byte, size)
			bytes2 := make([]byte, size)
			
			_, err := rand.Read(bytes1)
			require.NoError(t, err)
			
			_, err = rand.Read(bytes2)
			require.NoError(t, err)
			
			// Should be different
			assert.NotEqual(t, bytes1, bytes2, "Random bytes should be different")
			
			// Should have correct length
			assert.Len(t, bytes1, size)
			assert.Len(t, bytes2, size)
		}
	})
	
	t.Run("RandomDistribution", func(t *testing.T) {
		// Test that random bytes have roughly uniform distribution
		const numSamples = 10000
		const numBytes = 1000
		
		byteCounts := make([]int, 256)
		
		for i := 0; i < numSamples; i++ {
			randomBytes := make([]byte, numBytes)
			_, err := rand.Read(randomBytes)
			require.NoError(t, err)
			
			for _, b := range randomBytes {
				byteCounts[b]++
			}
		}
		
		// Check that distribution is roughly uniform
		expectedCount := numSamples * numBytes / 256
		tolerance := expectedCount / 4 // 25% tolerance
		
		for i, count := range byteCounts {
			assert.InDelta(t, expectedCount, count, float64(tolerance),
				"Byte value %d should appear roughly %d times, got %d", i, expectedCount, count)
		}
	})
}

// Time and Date Utilities Tests
func TestTimeUtilities(t *testing.T) {
	t.Run("TimestampGeneration", func(t *testing.T) {
		before := time.Now()
		
		// Create a new model that generates timestamps
		user, err := models.NewUser("testuser", "test@example.com", "Test User", "UTC")
		require.NoError(t, err)
		
		after := time.Now()
		
		// CreatedAt should be between before and after
		assert.True(t, user.CreatedAt.After(before) || user.CreatedAt.Equal(before))
		assert.True(t, user.CreatedAt.Before(after) || user.CreatedAt.Equal(after))
		
		// UpdatedAt should equal CreatedAt for new entities
		assert.Equal(t, user.CreatedAt, user.UpdatedAt)
	})
	
	t.Run("TimezonHandling", func(t *testing.T) {
		// Test that times are consistently in UTC
		user, err := models.NewUser("testuser", "test@example.com", "Test User", "America/New_York")
		require.NoError(t, err)
		
		// Timestamps should be in UTC regardless of user timezone
		assert.Equal(t, time.UTC, user.CreatedAt.Location())
		assert.Equal(t, time.UTC, user.UpdatedAt.Location())
	})
	
	t.Run("DurationCalculations", func(t *testing.T) {
		// Test calendar event duration calculations
		start := time.Now()
		end := start.Add(2 * time.Hour)
		
		event, err := models.NewCalendarEvent("user-id", "provider", "external-id", "Test Event", start, end)
		require.NoError(t, err)
		
		duration := event.Duration()
		assert.Equal(t, 2*time.Hour, duration)
		
		minutes := event.DurationMinutes()
		assert.Equal(t, 120, minutes)
	})
}

// Error Handling and Edge Cases
func TestUtilityEdgeCases(t *testing.T) {
	t.Run("NilPointerSafety", func(t *testing.T) {
		// Test that utility functions handle nil inputs gracefully
		var user *models.User
		
		// This should not panic (though it will return false)
		assert.False(t, user.CheckPassword("anypassword"))
	})
	
	t.Run("EmptyStringSafety", func(t *testing.T) {
		// Test UUID parsing with empty string
		_, err := uuid.Parse("")
		assert.Error(t, err)
		
		// Test password hashing with empty string
		user, err := models.NewUser("testuser", "test@example.com", "Test", "UTC")
		require.NoError(t, err)
		
		err = user.SetPassword("")
		assert.Error(t, err, "Empty password should be rejected")
	})
	
	t.Run("ExtremeValues", func(t *testing.T) {
		// Test distance calculation with extreme coordinates
		location, err := models.NewLocation("user-id", "Test", "", 0, 0, 100)
		require.NoError(t, err)
		
		// Test with extreme but valid coordinates
		distance := location.DistanceFrom(90, 180)
		assert.False(t, math.IsNaN(distance))
		assert.False(t, math.IsInf(distance, 0))
		assert.GreaterOrEqual(t, distance, 0.0)
	})
	
	t.Run("ConcurrencySafety", func(t *testing.T) {
		// Test UUID generation under concurrent access
		const numGoroutines = 100
		const uuidsPerGoroutine = 100
		
		uuids := make(chan string, numGoroutines*uuidsPerGoroutine)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < uuidsPerGoroutine; j++ {
					uuids <- uuid.New().String()
				}
			}()
		}
		
		// Collect all UUIDs
		uniqueUUIDs := make(map[string]bool)
		for i := 0; i < numGoroutines*uuidsPerGoroutine; i++ {
			id := <-uuids
			assert.False(t, uniqueUUIDs[id], "UUID should be unique even under concurrent generation")
			uniqueUUIDs[id] = true
		}
		
		assert.Equal(t, numGoroutines*uuidsPerGoroutine, len(uniqueUUIDs))
	})
}

// Performance Tests
func TestUtilityPerformance(t *testing.T) {
	t.Run("UUIDGenerationPerformance", func(t *testing.T) {
		const numUUIDs = 10000
		
		start := time.Now()
		for i := 0; i < numUUIDs; i++ {
			_ = uuid.New()
		}
		duration := time.Since(start)
		
		t.Logf("Generated %d UUIDs in %v (%.2f UUIDs/ms)", numUUIDs, duration, float64(numUUIDs)/float64(duration.Milliseconds()))
		
		// Should be very fast
		assert.Less(t, duration, 100*time.Millisecond, "UUID generation should be fast")
	})
	
	t.Run("DistanceCalculationPerformance", func(t *testing.T) {
		location, err := models.NewLocation("user-id", "Test", "", 37.7749, -122.4194, 100)
		require.NoError(t, err)
		
		const numCalculations = 10000
		
		start := time.Now()
		for i := 0; i < numCalculations; i++ {
			lat := 37.0 + float64(i)*0.0001
			lng := -122.0 + float64(i)*0.0001
			_ = location.DistanceFrom(lat, lng)
		}
		duration := time.Since(start)
		
		t.Logf("Calculated %d distances in %v (%.2f calcs/ms)", numCalculations, duration, float64(numCalculations)/float64(duration.Milliseconds()))
		
		// Should be reasonable performance
		assert.Less(t, duration, 50*time.Millisecond, "Distance calculation should be fast")
	})
}