package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bcnelson/hereAndNow/internal/auth"
)

type SessionRepository struct {
	db *DB
}

func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(session auth.Session) error {
	if session.Token == "" {
		return fmt.Errorf("session token cannot be empty")
	}
	if session.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	query := `
		INSERT INTO sessions (token, user_id, created_at, expires_at, user_agent, ip_address)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		session.Token,
		session.UserID,
		session.CreatedAt,
		session.ExpiresAt,
		session.UserAgent,
		session.IPAddress,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *SessionRepository) GetByToken(token string) (*auth.Session, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	query := `
		SELECT token, user_id, created_at, expires_at, user_agent, ip_address
		FROM sessions
		WHERE token = ?`

	session := &auth.Session{}
	err := r.db.QueryRow(query, token).Scan(
		&session.Token,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.UserAgent,
		&session.IPAddress,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}

	return session, nil
}

func (r *SessionRepository) GetByUserID(userID string) ([]auth.Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	query := `
		SELECT token, user_id, created_at, expires_at, user_agent, ip_address
		FROM sessions
		WHERE user_id = ?
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user ID: %w", err)
	}
	defer rows.Close()

	var sessions []auth.Session
	for rows.Next() {
		session := auth.Session{}
		err := rows.Scan(
			&session.Token,
			&session.UserID,
			&session.CreatedAt,
			&session.ExpiresAt,
			&session.UserAgent,
			&session.IPAddress,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}

func (r *SessionRepository) Delete(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	query := `DELETE FROM sessions WHERE token = ?`

	result, err := r.db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

func (r *SessionRepository) DeleteExpired() error {
	query := `DELETE FROM sessions WHERE expires_at < ?`

	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}

func (r *SessionRepository) DeleteByUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	query := `DELETE FROM sessions WHERE user_id = ?`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete sessions by user ID: %w", err)
	}

	return nil
}

func (r *SessionRepository) Count() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM sessions WHERE expires_at > ?`
	
	err := r.db.QueryRow(query, time.Now()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}

	return count, nil
}

func (r *SessionRepository) Cleanup() error {
	return r.DeleteExpired()
}