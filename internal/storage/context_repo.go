package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

// ContextRepository handles context data persistence and audit trail
type ContextRepository struct {
	db *DB
}

// NewContextRepository creates a new context repository
func NewContextRepository(db *DB) *ContextRepository {
	return &ContextRepository{db: db}
}

// ContextSearchOptions defines options for searching contexts
type ContextSearchOptions struct {
	UserID           string     // Filter by user ID
	After            *time.Time // Filter contexts after this time
	Before           *time.Time // Filter contexts before this time
	LocationID       *string    // Filter by current location
	SocialContext    *string    // Filter by social context
	MinEnergyLevel   *int       // Filter by minimum energy level
	MaxEnergyLevel   *int       // Filter by maximum energy level
	MinAvailableTime *int       // Filter by minimum available minutes
	MaxAvailableTime *int       // Filter by maximum available minutes
	WeatherCondition *string    // Filter by weather condition
	TrafficLevel     *string    // Filter by traffic level
	Limit            int        // Pagination limit
	Offset           int        // Pagination offset
	OrderBy          string     // Order by field (timestamp, energy_level, available_minutes)
	OrderDirection   string     // Order direction (ASC, DESC)
}

// Create creates a new context snapshot in the database
func (r *ContextRepository) Create(context *models.Context) error {
	if context.ID == "" {
		return fmt.Errorf("context ID cannot be empty")
	}

	// Validate the context before inserting
	if err := context.Validate(); err != nil {
		return fmt.Errorf("context validation failed: %w", err)
	}

	query := `
		INSERT INTO contexts (
			id, user_id, timestamp, current_latitude, current_longitude,
			current_location_id, available_minutes, social_context, energy_level,
			weather_condition, traffic_level, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		context.ID,
		context.UserID,
		context.Timestamp,
		context.CurrentLatitude,
		context.CurrentLongitude,
		context.CurrentLocationID,
		context.AvailableMinutes,
		context.SocialContext,
		context.EnergyLevel,
		context.WeatherCondition,
		context.TrafficLevel,
		context.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create context: %w", err)
	}

	return nil
}

// GetByID retrieves a context by its ID
func (r *ContextRepository) GetByID(id string) (*models.Context, error) {
	if id == "" {
		return nil, fmt.Errorf("context ID cannot be empty")
	}

	query := `
		SELECT id, user_id, timestamp, current_latitude, current_longitude,
		       current_location_id, available_minutes, social_context, energy_level,
		       weather_condition, traffic_level, metadata
		FROM contexts 
		WHERE id = ?`

	context := &models.Context{}

	err := r.db.QueryRow(query, id).Scan(
		&context.ID,
		&context.UserID,
		&context.Timestamp,
		&context.CurrentLatitude,
		&context.CurrentLongitude,
		&context.CurrentLocationID,
		&context.AvailableMinutes,
		&context.SocialContext,
		&context.EnergyLevel,
		&context.WeatherCondition,
		&context.TrafficLevel,
		&context.Metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("context not found")
		}
		return nil, fmt.Errorf("failed to get context by ID: %w", err)
	}

	return context, nil
}

// GetLatestByUser retrieves the most recent context for a user
func (r *ContextRepository) GetLatestByUser(userID string) (*models.Context, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	query := `
		SELECT id, user_id, timestamp, current_latitude, current_longitude,
		       current_location_id, available_minutes, social_context, energy_level,
		       weather_condition, traffic_level, metadata
		FROM contexts 
		WHERE user_id = ?
		ORDER BY timestamp DESC
		LIMIT 1`

	context := &models.Context{}

	err := r.db.QueryRow(query, userID).Scan(
		&context.ID,
		&context.UserID,
		&context.Timestamp,
		&context.CurrentLatitude,
		&context.CurrentLongitude,
		&context.CurrentLocationID,
		&context.AvailableMinutes,
		&context.SocialContext,
		&context.EnergyLevel,
		&context.WeatherCondition,
		&context.TrafficLevel,
		&context.Metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no context found for user")
		}
		return nil, fmt.Errorf("failed to get latest context: %w", err)
	}

	return context, nil
}

// Search searches contexts with various filters for audit trail analysis
func (r *ContextRepository) Search(options ContextSearchOptions) ([]*models.Context, error) {
	var conditions []string
	var args []interface{}

	// Base query
	baseQuery := `
		SELECT id, user_id, timestamp, current_latitude, current_longitude,
		       current_location_id, available_minutes, social_context, energy_level,
		       weather_condition, traffic_level, metadata
		FROM contexts
	`

	// Add user filter
	if options.UserID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, options.UserID)
	}

	// Add time range filters
	if options.After != nil {
		conditions = append(conditions, "timestamp > ?")
		args = append(args, *options.After)
	}
	if options.Before != nil {
		conditions = append(conditions, "timestamp < ?")
		args = append(args, *options.Before)
	}

	// Add location filter
	if options.LocationID != nil {
		conditions = append(conditions, "current_location_id = ?")
		args = append(args, *options.LocationID)
	}

	// Add social context filter
	if options.SocialContext != nil {
		conditions = append(conditions, "social_context = ?")
		args = append(args, *options.SocialContext)
	}

	// Add energy level filters
	if options.MinEnergyLevel != nil {
		conditions = append(conditions, "energy_level >= ?")
		args = append(args, *options.MinEnergyLevel)
	}
	if options.MaxEnergyLevel != nil {
		conditions = append(conditions, "energy_level <= ?")
		args = append(args, *options.MaxEnergyLevel)
	}

	// Add available time filters
	if options.MinAvailableTime != nil {
		conditions = append(conditions, "available_minutes >= ?")
		args = append(args, *options.MinAvailableTime)
	}
	if options.MaxAvailableTime != nil {
		conditions = append(conditions, "available_minutes <= ?")
		args = append(args, *options.MaxAvailableTime)
	}

	// Add weather condition filter
	if options.WeatherCondition != nil {
		conditions = append(conditions, "weather_condition = ?")
		args = append(args, *options.WeatherCondition)
	}

	// Add traffic level filter
	if options.TrafficLevel != nil {
		conditions = append(conditions, "traffic_level = ?")
		args = append(args, *options.TrafficLevel)
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderClause := "ORDER BY timestamp DESC" // Default ordering
	if options.OrderBy != "" {
		direction := "DESC"
		if options.OrderDirection == "ASC" {
			direction = "ASC"
		}
		
		// Validate order by field
		validOrderFields := map[string]bool{
			"timestamp": true, "energy_level": true, "available_minutes": true,
			"social_context": true, "weather_condition": true, "traffic_level": true,
		}
		if validOrderFields[options.OrderBy] {
			orderClause = fmt.Sprintf("ORDER BY %s %s", options.OrderBy, direction)
		}
	}

	// Build LIMIT clause
	limitClause := ""
	if options.Limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", options.Limit)
		if options.Offset > 0 {
			limitClause += fmt.Sprintf(" OFFSET %d", options.Offset)
		}
	}

	// Combine query parts
	query := baseQuery + whereClause + " " + orderClause + " " + limitClause

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search contexts: %w", err)
	}
	defer rows.Close()

	var contexts []*models.Context
	for rows.Next() {
		context := &models.Context{}

		err := rows.Scan(
			&context.ID,
			&context.UserID,
			&context.Timestamp,
			&context.CurrentLatitude,
			&context.CurrentLongitude,
			&context.CurrentLocationID,
			&context.AvailableMinutes,
			&context.SocialContext,
			&context.EnergyLevel,
			&context.WeatherCondition,
			&context.TrafficLevel,
			&context.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan context row: %w", err)
		}

		contexts = append(contexts, context)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating context rows: %w", err)
	}

	return contexts, nil
}

// GetHistoryByUser returns context history for a user within a time range
func (r *ContextRepository) GetHistoryByUser(userID string, after, before *time.Time, limit, offset int) ([]*models.Context, error) {
	options := ContextSearchOptions{
		UserID:         userID,
		After:          after,
		Before:         before,
		Limit:          limit,
		Offset:         offset,
		OrderBy:        "timestamp",
		OrderDirection: "DESC",
	}
	return r.Search(options)
}

// GetByTimeRange returns all contexts within a specific time range for analysis
func (r *ContextRepository) GetByTimeRange(userID string, start, end time.Time, limit, offset int) ([]*models.Context, error) {
	options := ContextSearchOptions{
		UserID:         userID,
		After:          &start,
		Before:         &end,
		Limit:          limit,
		Offset:         offset,
		OrderBy:        "timestamp",
		OrderDirection: "ASC",
	}
	return r.Search(options)
}

// GetByLocation returns contexts for a specific location
func (r *ContextRepository) GetByLocation(userID, locationID string, limit, offset int) ([]*models.Context, error) {
	options := ContextSearchOptions{
		UserID:         userID,
		LocationID:     &locationID,
		Limit:          limit,
		Offset:         offset,
		OrderBy:        "timestamp",
		OrderDirection: "DESC",
	}
	return r.Search(options)
}

// GetBySocialContext returns contexts for a specific social context
func (r *ContextRepository) GetBySocialContext(userID, socialContext string, limit, offset int) ([]*models.Context, error) {
	options := ContextSearchOptions{
		UserID:         userID,
		SocialContext:  &socialContext,
		Limit:          limit,
		Offset:         offset,
		OrderBy:        "timestamp",
		OrderDirection: "DESC",
	}
	return r.Search(options)
}

// Delete removes old context snapshots (for cleanup/privacy)
func (r *ContextRepository) Delete(contextID string) error {
	if contextID == "" {
		return fmt.Errorf("context ID cannot be empty")
	}

	// Check if context is referenced in filter audit records
	var auditCount int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM filter_audit WHERE context_id = ?
	`, contextID).Scan(&auditCount)
	
	if err != nil {
		return fmt.Errorf("failed to check context usage in audit records: %w", err)
	}

	if auditCount > 0 {
		return fmt.Errorf("cannot delete context: it is referenced by %d audit records", auditCount)
	}

	result, err := r.db.Exec(`DELETE FROM contexts WHERE id = ?`, contextID)
	if err != nil {
		return fmt.Errorf("failed to delete context: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("context not found")
	}

	return nil
}

// DeleteByUser removes all contexts for a user (for cleanup/privacy)
func (r *ContextRepository) DeleteByUser(userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	result, err := r.db.Exec(`DELETE FROM contexts WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete contexts for user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	return nil
}

// DeleteOlderThan removes contexts older than the specified time (for cleanup)
func (r *ContextRepository) DeleteOlderThan(before time.Time) error {
	result, err := r.db.Exec(`DELETE FROM contexts WHERE timestamp < ?`, before)
	if err != nil {
		return fmt.Errorf("failed to delete old contexts: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	return nil
}

// Count returns the total number of contexts matching the search options
func (r *ContextRepository) Count(options ContextSearchOptions) (int, error) {
	var conditions []string
	var args []interface{}

	// Build query conditions (similar to Search method but for count)
	if options.UserID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, options.UserID)
	}

	if options.After != nil {
		conditions = append(conditions, "timestamp > ?")
		args = append(args, *options.After)
	}
	if options.Before != nil {
		conditions = append(conditions, "timestamp < ?")
		args = append(args, *options.Before)
	}

	if options.LocationID != nil {
		conditions = append(conditions, "current_location_id = ?")
		args = append(args, *options.LocationID)
	}

	if options.SocialContext != nil {
		conditions = append(conditions, "social_context = ?")
		args = append(args, *options.SocialContext)
	}

	if options.MinEnergyLevel != nil {
		conditions = append(conditions, "energy_level >= ?")
		args = append(args, *options.MinEnergyLevel)
	}
	if options.MaxEnergyLevel != nil {
		conditions = append(conditions, "energy_level <= ?")
		args = append(args, *options.MaxEnergyLevel)
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := "SELECT COUNT(*) FROM contexts " + whereClause

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count contexts: %w", err)
	}

	return count, nil
}

// GetAggregatedStats returns aggregated statistics for contexts within a time range
type ContextStats struct {
	TotalSnapshots      int                        `json:"total_snapshots"`
	AverageEnergyLevel  float64                    `json:"average_energy_level"`
	AverageAvailableMin float64                    `json:"average_available_minutes"`
	LocationFrequency   map[string]int             `json:"location_frequency"`
	SocialContextFreq   map[string]int             `json:"social_context_frequency"`
	WeatherFrequency    map[string]int             `json:"weather_frequency"`
	TrafficFrequency    map[string]int             `json:"traffic_frequency"`
	EnergyDistribution  map[int]int                `json:"energy_distribution"`
	TimeRange           map[string]time.Time       `json:"time_range"`
}

// GetAggregatedStats returns aggregated statistics for a user's contexts
func (r *ContextRepository) GetAggregatedStats(userID string, after, before *time.Time) (*ContextStats, error) {
	contexts, err := r.GetHistoryByUser(userID, after, before, 0, 0) // Get all contexts
	if err != nil {
		return nil, fmt.Errorf("failed to get context history: %w", err)
	}

	if len(contexts) == 0 {
		return &ContextStats{
			TotalSnapshots:     0,
			LocationFrequency:  make(map[string]int),
			SocialContextFreq:  make(map[string]int),
			WeatherFrequency:   make(map[string]int),
			TrafficFrequency:   make(map[string]int),
			EnergyDistribution: make(map[int]int),
			TimeRange:          make(map[string]time.Time),
		}, nil
	}

	stats := &ContextStats{
		TotalSnapshots:     len(contexts),
		LocationFrequency:  make(map[string]int),
		SocialContextFreq:  make(map[string]int),
		WeatherFrequency:   make(map[string]int),
		TrafficFrequency:   make(map[string]int),
		EnergyDistribution: make(map[int]int),
		TimeRange:          make(map[string]time.Time),
	}

	var totalEnergy, totalAvailableMin int
	var earliest, latest time.Time

	for i, ctx := range contexts {
		// Track totals for averages
		totalEnergy += ctx.EnergyLevel
		totalAvailableMin += ctx.AvailableMinutes

		// Track time range
		if i == 0 {
			earliest = ctx.Timestamp
			latest = ctx.Timestamp
		} else {
			if ctx.Timestamp.Before(earliest) {
				earliest = ctx.Timestamp
			}
			if ctx.Timestamp.After(latest) {
				latest = ctx.Timestamp
			}
		}

		// Track location frequency
		if ctx.CurrentLocationID != nil {
			stats.LocationFrequency[*ctx.CurrentLocationID]++
		}

		// Track social context frequency
		stats.SocialContextFreq[ctx.SocialContext]++

		// Track weather frequency
		if ctx.WeatherCondition != nil {
			stats.WeatherFrequency[*ctx.WeatherCondition]++
		}

		// Track traffic frequency
		if ctx.TrafficLevel != nil {
			stats.TrafficFrequency[*ctx.TrafficLevel]++
		}

		// Track energy distribution
		stats.EnergyDistribution[ctx.EnergyLevel]++
	}

	// Calculate averages
	stats.AverageEnergyLevel = float64(totalEnergy) / float64(len(contexts))
	stats.AverageAvailableMin = float64(totalAvailableMin) / float64(len(contexts))

	// Set time range
	stats.TimeRange["earliest"] = earliest
	stats.TimeRange["latest"] = latest

	return stats, nil
}

// UpdateMetadata updates a context's metadata
func (r *ContextRepository) UpdateMetadata(contextID string, metadata map[string]interface{}) error {
	if contextID == "" {
		return fmt.Errorf("context ID cannot be empty")
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `UPDATE contexts SET metadata = ? WHERE id = ?`
	_, err = r.db.Exec(query, metadataJSON, contextID)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// Exists checks if a context exists by ID
func (r *ContextRepository) Exists(contextID string) (bool, error) {
	if contextID == "" {
		return false, fmt.Errorf("context ID cannot be empty")
	}

	var count int
	query := `SELECT COUNT(*) FROM contexts WHERE id = ?`
	
	err := r.db.QueryRow(query, contextID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check context existence: %w", err)
	}

	return count > 0, nil
}