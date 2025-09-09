package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type User struct {
	ID           string          `db:"id" json:"id"`
	Username     string          `db:"username" json:"username"`
	Email        string          `db:"email" json:"email"`
	PasswordHash string          `db:"password_hash" json:"-"`
	DisplayName  string          `db:"display_name" json:"display_name"`
	TimeZone     string          `db:"timezone" json:"timezone"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
	LastSeenAt   time.Time       `db:"last_seen_at" json:"last_seen_at"`
	Settings     json.RawMessage `db:"settings" json:"settings"`
}

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func NewUser(username, email, displayName, timezone string) (*User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validateTimezone(timezone); err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:          uuid.New().String(),
		Username:    strings.ToLower(username),
		Email:       strings.ToLower(email),
		DisplayName: displayName,
		TimeZone:    timezone,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastSeenAt:  now,
		Settings:    json.RawMessage(`{}`),
	}, nil
}

func (u *User) SetPassword(password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}

	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i)
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	u.PasswordHash = fmt.Sprintf("$argon2id$v=19$m=65536,t=1,p=4$%x$%x", salt, hash)
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) CheckPassword(password string) bool {
	parts := strings.Split(u.PasswordHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	var salt []byte
	var storedHash []byte
	fmt.Sscanf(parts[4], "%x", &salt)
	fmt.Sscanf(parts[5], "%x", &storedHash)

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	
	if len(hash) != len(storedHash) {
		return false
	}
	
	for i := range hash {
		if hash[i] != storedHash[i] {
			return false
		}
	}
	
	return true
}

func (u *User) Validate() error {
	if err := validateUsername(u.Username); err != nil {
		return err
	}
	if err := validateEmail(u.Email); err != nil {
		return err
	}
	if err := validateTimezone(u.TimeZone); err != nil {
		return err
	}
	if u.PasswordHash == "" {
		return fmt.Errorf("password hash is required")
	}
	return nil
}

func validateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username must be 3-50 characters, alphanumeric + underscore only")
	}
	return nil
}

func validateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

func validateTimezone(timezone string) error {
	_, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid IANA timezone: %s", timezone)
	}
	return nil
}