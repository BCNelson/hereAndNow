package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

// UserRepository handles user data persistence
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user in the database
func (r *UserRepository) Create(user *models.User) error {
	if user.ID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Validate the user before inserting
	if err := user.Validate(); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	query := `
		INSERT INTO users (
			id, username, email, password_hash, display_name, 
			timezone, created_at, updated_at, last_seen_at, settings
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		user.ID,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.TimeZone,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastSeenAt,
		user.Settings,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID
func (r *UserRepository) GetByID(id string) (*models.User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	query := `
		SELECT id, username, email, password_hash, display_name, 
		       timezone, created_at, updated_at, last_seen_at, settings
		FROM users 
		WHERE id = ?`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.TimeZone,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastSeenAt,
		&user.Settings,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetByUsername retrieves a user by their username
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	query := `
		SELECT id, username, email, password_hash, display_name, 
		       timezone, created_at, updated_at, last_seen_at, settings
		FROM users 
		WHERE username = ?`

	user := &models.User{}
	err := r.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.TimeZone,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastSeenAt,
		&user.Settings,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by their email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	query := `
		SELECT id, username, email, password_hash, display_name, 
		       timezone, created_at, updated_at, last_seen_at, settings
		FROM users 
		WHERE email = ?`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.TimeZone,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastSeenAt,
		&user.Settings,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// Update updates an existing user
func (r *UserRepository) Update(user *models.User) error {
	if user.ID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Validate the user before updating
	if err := user.Validate(); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	// Update the timestamp
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users 
		SET username = ?, email = ?, password_hash = ?, display_name = ?, 
		    timezone = ?, updated_at = ?, last_seen_at = ?, settings = ?
		WHERE id = ?`

	result, err := r.db.Exec(query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.TimeZone,
		user.UpdatedAt,
		user.LastSeenAt,
		user.Settings,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(userID string, newPassword string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Get the user to update their password hash
	user, err := r.GetByID(userID)
	if err != nil {
		return err
	}

	// Set the new password (this will hash it)
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	// Update only the password hash and updated_at fields
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`

	_, err = r.db.Exec(query, user.PasswordHash, user.UpdatedAt, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// UpdateLastSeen updates a user's last seen timestamp
func (r *UserRepository) UpdateLastSeen(userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	query := `UPDATE users SET last_seen_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}

	return nil
}

// UpdateSettings updates a user's settings
func (r *UserRepository) UpdateSettings(userID string, settings map[string]interface{}) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `UPDATE users SET settings = ?, updated_at = ? WHERE id = ?`
	_, err = r.db.Exec(query, settingsJSON, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}

// Delete soft deletes a user (for compliance, we might want to keep user data)
func (r *UserRepository) Delete(userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Note: This is a hard delete. In production, you might want to implement
	// soft delete by adding a deleted_at column and updating it instead
	query := `DELETE FROM users WHERE id = ?`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// List retrieves users with pagination
func (r *UserRepository) List(limit, offset int) ([]*models.User, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, username, email, password_hash, display_name, 
		       timezone, created_at, updated_at, last_seen_at, settings
		FROM users 
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.DisplayName,
			&user.TimeZone,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastSeenAt,
			&user.Settings,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// Count returns the total number of users
func (r *UserRepository) Count() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`
	
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// Exists checks if a user exists by ID
func (r *UserRepository) Exists(userID string) (bool, error) {
	if userID == "" {
		return false, fmt.Errorf("user ID cannot be empty")
	}

	var count int
	query := `SELECT COUNT(*) FROM users WHERE id = ?`
	
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByUsername checks if a user exists by username
func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	if username == "" {
		return false, fmt.Errorf("username cannot be empty")
	}

	var count int
	query := `SELECT COUNT(*) FROM users WHERE username = ?`
	
	err := r.db.QueryRow(query, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByEmail checks if a user exists by email
func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	if email == "" {
		return false, fmt.Errorf("email cannot be empty")
	}

	var count int
	query := `SELECT COUNT(*) FROM users WHERE email = ?`
	
	err := r.db.QueryRow(query, email).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return count > 0, nil
}

// AuthenticateUser validates user credentials and returns the user if valid
func (r *UserRepository) AuthenticateUser(username, password string) (*models.User, error) {
	user, err := r.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	if !user.CheckPassword(password) {
		return nil, fmt.Errorf("authentication failed: invalid credentials")
	}

	// Update last seen timestamp
	if err := r.UpdateLastSeen(user.ID); err != nil {
		// Log this error but don't fail the authentication
		// In production, you'd use a proper logger here
		fmt.Printf("Warning: failed to update last seen for user %s: %v\n", user.ID, err)
	}

	return user, nil
}