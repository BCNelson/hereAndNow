package auth

import (
	"fmt"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type AuthService struct {
	userRepo     UserRepository
	sessionRepo  SessionRepository
	jwtService   JWTService
	config       AuthConfig
}

type UserRepository interface {
	Create(user models.User) error
	GetByID(userID string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Update(user models.User) error
	UpdatePassword(userID string, hashedPassword string) error
}

type SessionRepository interface {
	Create(session Session) error
	GetByToken(token string) (*Session, error)
	GetByUserID(userID string) ([]Session, error)
	Delete(token string) error
	DeleteExpired() error
	DeleteByUserID(userID string) error
}

type JWTService interface {
	GenerateToken(userID string, expiresAt time.Time) (string, error)
	ValidateToken(token string) (*TokenClaims, error)
	RefreshToken(token string) (string, error)
}

type AuthConfig struct {
	SessionDuration    time.Duration `json:"session_duration"`
	MaxSessions        int           `json:"max_sessions"`
	PasswordMinLength  int           `json:"password_min_length"`
	RequireEmailVerify bool          `json:"require_email_verify"`
	JWTSecret          string        `json:"jwt_secret"`
	Argon2Time         uint32        `json:"argon2_time"`
	Argon2Memory       uint32        `json:"argon2_memory"`
	Argon2Threads      uint8         `json:"argon2_threads"`
	Argon2KeyLen       uint32        `json:"argon2_key_len"`
}

type Session struct {
	Token     string    `db:"token" json:"token"`
	UserID    string    `db:"user_id" json:"user_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	UserAgent string    `db:"user_agent" json:"user_agent"`
	IPAddress string    `db:"ip_address" json:"ip_address"`
}

type TokenClaims struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      models.User `json:"user"`
}

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Timezone  string `json:"timezone"`
}

func NewAuthService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	jwtService JWTService,
	config AuthConfig,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		jwtService:  jwtService,
		config:      config,
	}
}

func (s *AuthService) Login(req LoginRequest, userAgent, ipAddress string) (*LoginResponse, error) {
	if err := s.validateLoginRequest(req); err != nil {
		return nil, fmt.Errorf("invalid login request: %w", err)
	}

	// Try to get user by email first, then by username
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		// If email lookup fails, try username lookup
		if userRepo, ok := s.userRepo.(interface{ GetByUsername(string) (*models.User, error) }); ok {
			user, err = userRepo.GetByUsername(req.Email)
			if err != nil {
				return nil, fmt.Errorf("invalid credentials")
			}
		} else {
			return nil, fmt.Errorf("invalid credentials")
		}
	}

	if !s.verifyPassword(req.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Note: EmailVerified field not available in current User model
	// TODO: Add EmailVerified field to User model if email verification is needed

	if err := s.cleanupOldSessions(user.ID); err != nil {
		return nil, fmt.Errorf("failed to cleanup old sessions: %w", err)
	}

	expiresAt := time.Now().Add(s.config.SessionDuration)
	token, err := s.jwtService.GenerateToken(user.ID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	session := Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Note: LastLoginAt field not available in current User model
	// Using LastSeenAt instead
	user.LastSeenAt = session.CreatedAt
	if err := s.userRepo.Update(*user); err != nil {
		return nil, fmt.Errorf("failed to update user login time: %w", err)
	}

	sanitizedUser := *user
	sanitizedUser.PasswordHash = ""

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      sanitizedUser,
	}, nil
}

func (s *AuthService) Register(req RegisterRequest) (*models.User, error) {
	if err := s.validateRegisterRequest(req); err != nil {
		return nil, fmt.Errorf("invalid registration request: %w", err)
	}

	existingUser, _ := s.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := models.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		DisplayName:  fmt.Sprintf("%s %s", req.FirstName, req.LastName),
		PasswordHash: hashedPassword,
		TimeZone:     req.Timezone,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastSeenAt:   time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	sanitizedUser := user
	sanitizedUser.PasswordHash = ""

	return &sanitizedUser, nil
}

func (s *AuthService) Logout(token string) error {
	_, err := s.sessionRepo.GetByToken(token)
	if err != nil {
		return fmt.Errorf("invalid session")
	}

	if err := s.sessionRepo.Delete(token); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (s *AuthService) LogoutAll(userID string) error {
	if err := s.sessionRepo.DeleteByUserID(userID); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

func (s *AuthService) ValidateToken(token string) (*models.User, error) {
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	session, err := s.sessionRepo.GetByToken(token)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		s.sessionRepo.Delete(token)
		return nil, fmt.Errorf("session expired")
	}

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	sanitizedUser := *user
	sanitizedUser.PasswordHash = ""

	return &sanitizedUser, nil
}

func (s *AuthService) RefreshToken(token string) (*LoginResponse, error) {
	session, err := s.sessionRepo.GetByToken(token)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		s.sessionRepo.Delete(token)
		return nil, fmt.Errorf("session expired")
	}

	user, err := s.userRepo.GetByID(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	newExpiresAt := time.Now().Add(s.config.SessionDuration)
	newToken, err := s.jwtService.GenerateToken(user.ID, newExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}

	s.sessionRepo.Delete(token)

	newSession := Session{
		Token:     newToken,
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: newExpiresAt,
		UserAgent: session.UserAgent,
		IPAddress: session.IPAddress,
	}

	if err := s.sessionRepo.Create(newSession); err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	sanitizedUser := *user
	sanitizedUser.PasswordHash = ""

	return &LoginResponse{
		Token:     newToken,
		ExpiresAt: newExpiresAt,
		User:      sanitizedUser,
	}, nil
}

func (s *AuthService) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if !s.verifyPassword(oldPassword, user.PasswordHash) {
		return fmt.Errorf("invalid current password")
	}

	if err := s.validatePassword(newPassword); err != nil {
		return fmt.Errorf("invalid new password: %w", err)
	}

	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(userID, hashedPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.sessionRepo.DeleteByUserID(userID); err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	return nil
}

func (s *AuthService) GetUserSessions(userID string) ([]Session, error) {
	sessions, err := s.sessionRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	activeSessions := []Session{}
	for _, session := range sessions {
		if time.Now().Before(session.ExpiresAt) {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions, nil
}

func (s *AuthService) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(time.Now().UnixNano() % 256)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.config.Argon2Time,
		s.config.Argon2Memory,
		s.config.Argon2Threads,
		s.config.Argon2KeyLen,
	)

	return fmt.Sprintf("%x:%x", salt, hash), nil
}

func (s *AuthService) verifyPassword(password, hashedPassword string) bool {
	parts := splitString(hashedPassword, ":")
	if len(parts) != 2 {
		return false
	}

	salt := hexDecode(parts[0])
	if salt == nil {
		return false
	}

	expectedHash := hexDecode(parts[1])
	if expectedHash == nil {
		return false
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.config.Argon2Time,
		s.config.Argon2Memory,
		s.config.Argon2Threads,
		s.config.Argon2KeyLen,
	)

	return constantTimeEqual(hash, expectedHash)
}

func (s *AuthService) validateLoginRequest(req LoginRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

func (s *AuthService) validateRegisterRequest(req RegisterRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.FirstName == "" {
		return fmt.Errorf("first name is required")
	}
	if err := s.validatePassword(req.Password); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) validatePassword(password string) error {
	if len(password) < s.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", s.config.PasswordMinLength)
	}
	return nil
}

func (s *AuthService) cleanupOldSessions(userID string) error {
	sessions, err := s.sessionRepo.GetByUserID(userID)
	if err != nil {
		return err
	}

	if len(sessions) >= s.config.MaxSessions {
		for i := 0; i < len(sessions)-s.config.MaxSessions+1; i++ {
			s.sessionRepo.Delete(sessions[i].Token)
		}
	}

	return s.sessionRepo.DeleteExpired()
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	
	parts := []string{}
	start := 0
	
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, s[start:])
	
	return parts
}

func hexDecode(s string) []byte {
	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		if i+1 >= len(s) {
			return nil
		}
		
		high := hexCharToByte(s[i])
		low := hexCharToByte(s[i+1])
		if high == 255 || low == 255 {
			return nil
		}
		
		result[i/2] = (high << 4) | low
	}
	return result
}

func hexCharToByte(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 255
	}
}

func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	
	result := byte(0)
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	
	return result == 0
}

var DefaultAuthConfig = AuthConfig{
	SessionDuration:    24 * time.Hour,
	MaxSessions:        5,
	PasswordMinLength:  8,
	RequireEmailVerify: false,
	Argon2Time:         1,
	Argon2Memory:       64 * 1024,
	Argon2Threads:      4,
	Argon2KeyLen:       32,
}