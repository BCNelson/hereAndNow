package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

// LocationRepository handles location data persistence with spatial queries
type LocationRepository struct {
	db *DB
}

// NewLocationRepository creates a new location repository
func NewLocationRepository(db *DB) *LocationRepository {
	return &LocationRepository{db: db}
}

// LocationSearchOptions defines options for searching locations
type LocationSearchOptions struct {
	UserID           string   // Filter by user ID
	Category         *string  // Filter by category
	NearLatitude     *float64 // Latitude for proximity search
	NearLongitude    *float64 // Longitude for proximity search
	WithinMeters     *float64 // Maximum distance in meters for proximity search
	Query            string   // Full-text search query for name/address
	Limit            int      // Pagination limit
	Offset           int      // Pagination offset
	OrderBy          string   // Order by field (name, created_at, distance)
	OrderDirection   string   // Order direction (ASC, DESC)
}

// Create creates a new location in the database
func (r *LocationRepository) Create(location *models.Location) error {
	if location.ID == "" {
		return fmt.Errorf("location ID cannot be empty")
	}

	// Validate the location before inserting
	if err := location.Validate(); err != nil {
		return fmt.Errorf("location validation failed: %w", err)
	}

	query := `
		INSERT INTO locations (
			id, user_id, name, address, latitude, longitude, 
			radius, category, place_id, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		location.ID,
		location.UserID,
		location.Name,
		location.Address,
		location.Latitude,
		location.Longitude,
		location.Radius,
		location.Category,
		location.PlaceID,
		location.Metadata,
		location.CreatedAt,
		location.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create location: %w", err)
	}

	return nil
}

// GetByID retrieves a location by its ID
func (r *LocationRepository) GetByID(id string) (*models.Location, error) {
	if id == "" {
		return nil, fmt.Errorf("location ID cannot be empty")
	}

	query := `
		SELECT id, user_id, name, address, latitude, longitude, 
		       radius, category, place_id, metadata, created_at, updated_at
		FROM locations 
		WHERE id = ?`

	location := &models.Location{}

	err := r.db.QueryRow(query, id).Scan(
		&location.ID,
		&location.UserID,
		&location.Name,
		&location.Address,
		&location.Latitude,
		&location.Longitude,
		&location.Radius,
		&location.Category,
		&location.PlaceID,
		&location.Metadata,
		&location.CreatedAt,
		&location.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("location not found")
		}
		return nil, fmt.Errorf("failed to get location by ID: %w", err)
	}

	return location, nil
}

// Update updates an existing location
func (r *LocationRepository) Update(location *models.Location) error {
	if location.ID == "" {
		return fmt.Errorf("location ID cannot be empty")
	}

	// Validate the location before updating
	if err := location.Validate(); err != nil {
		return fmt.Errorf("location validation failed: %w", err)
	}

	// Update the timestamp
	location.UpdatedAt = time.Now()

	query := `
		UPDATE locations 
		SET name = ?, address = ?, latitude = ?, longitude = ?, 
		    radius = ?, category = ?, place_id = ?, metadata = ?, updated_at = ?
		WHERE id = ?`

	result, err := r.db.Exec(query,
		location.Name,
		location.Address,
		location.Latitude,
		location.Longitude,
		location.Radius,
		location.Category,
		location.PlaceID,
		location.Metadata,
		location.UpdatedAt,
		location.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update location: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("location not found")
	}

	return nil
}

// Delete deletes a location from the database
func (r *LocationRepository) Delete(locationID string) error {
	if locationID == "" {
		return fmt.Errorf("location ID cannot be empty")
	}

	// Check if location is used in any tasks
	var taskCount int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM task_locations WHERE location_id = ?
	`, locationID).Scan(&taskCount)
	
	if err != nil {
		return fmt.Errorf("failed to check location usage: %w", err)
	}

	if taskCount > 0 {
		return fmt.Errorf("cannot delete location: it is referenced by %d tasks", taskCount)
	}

	// Check if location is used in any contexts
	var contextCount int
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM contexts WHERE current_location_id = ?
	`, locationID).Scan(&contextCount)
	
	if err != nil {
		return fmt.Errorf("failed to check location context usage: %w", err)
	}

	if contextCount > 0 {
		return fmt.Errorf("cannot delete location: it is referenced by %d context records", contextCount)
	}

	// Delete the location
	result, err := r.db.Exec(`DELETE FROM locations WHERE id = ?`, locationID)
	if err != nil {
		return fmt.Errorf("failed to delete location: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("location not found")
	}

	return nil
}

// Search searches locations with various filters including spatial queries
func (r *LocationRepository) Search(options LocationSearchOptions) ([]*models.Location, error) {
	var conditions []string
	var args []interface{}
	var selectClause string
	var orderClause string

	// Base select clause
	selectClause = `
		SELECT l.id, l.user_id, l.name, l.address, l.latitude, l.longitude, 
		       l.radius, l.category, l.place_id, l.metadata, l.created_at, l.updated_at
	`

	// Add distance calculation if proximity search is requested
	if options.NearLatitude != nil && options.NearLongitude != nil {
		selectClause += fmt.Sprintf(`, %f AS distance`, 0.0) // Placeholder, calculated below
	}

	var fromClause string
	if options.Query != "" {
		// Use full-text search
		fromClause = `
			FROM locations l
			JOIN locations_fts fts ON l.rowid = fts.rowid
		`
		conditions = append(conditions, "locations_fts MATCH ?")
		args = append(args, options.Query)
	} else {
		fromClause = "FROM locations l"
	}

	// Add user filter
	if options.UserID != "" {
		conditions = append(conditions, "l.user_id = ?")
		args = append(args, options.UserID)
	}

	// Add category filter
	if options.Category != nil {
		conditions = append(conditions, "l.category = ?")
		args = append(args, *options.Category)
	}

	// Add proximity filter using Haversine formula
	if options.NearLatitude != nil && options.NearLongitude != nil && options.WithinMeters != nil {
		// Use Haversine formula in SQL
		haversineSQL := fmt.Sprintf(`
			(6371000 * acos(
				cos(radians(%f)) * cos(radians(l.latitude)) * 
				cos(radians(l.longitude) - radians(%f)) + 
				sin(radians(%f)) * sin(radians(l.latitude))
			)) <= %f`,
			*options.NearLatitude, *options.NearLongitude,
			*options.NearLatitude, *options.WithinMeters)
		
		conditions = append(conditions, haversineSQL)
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	if options.NearLatitude != nil && options.NearLongitude != nil && options.OrderBy == "distance" {
		// Order by calculated distance
		orderClause = fmt.Sprintf(`
			ORDER BY (6371000 * acos(
				cos(radians(%f)) * cos(radians(l.latitude)) * 
				cos(radians(l.longitude) - radians(%f)) + 
				sin(radians(%f)) * sin(radians(l.latitude))
			))`,
			*options.NearLatitude, *options.NearLongitude, *options.NearLatitude)
		
		if options.OrderDirection == "DESC" {
			orderClause += " DESC"
		} else {
			orderClause += " ASC"
		}
	} else if options.OrderBy != "" {
		direction := "ASC"
		if options.OrderDirection == "DESC" {
			direction = "DESC"
		}
		
		// Validate order by field
		validOrderFields := map[string]bool{
			"name": true, "created_at": true, "updated_at": true,
			"category": true, "address": true,
		}
		if validOrderFields[options.OrderBy] {
			orderClause = fmt.Sprintf("ORDER BY l.%s %s", options.OrderBy, direction)
		} else {
			orderClause = "ORDER BY l.name ASC"
		}
	} else {
		orderClause = "ORDER BY l.name ASC" // Default ordering
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
	query := selectClause + fromClause + " " + whereClause + " " + orderClause + " " + limitClause

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search locations: %w", err)
	}
	defer rows.Close()

	var locations []*models.Location
	for rows.Next() {
		location := &models.Location{}
		var distance *float64

		// Prepare scan arguments
		scanArgs := []interface{}{
			&location.ID,
			&location.UserID,
			&location.Name,
			&location.Address,
			&location.Latitude,
			&location.Longitude,
			&location.Radius,
			&location.Category,
			&location.PlaceID,
			&location.Metadata,
			&location.CreatedAt,
			&location.UpdatedAt,
		}

		// Add distance to scan if proximity search was used
		if options.NearLatitude != nil && options.NearLongitude != nil {
			scanArgs = append(scanArgs, &distance)
		}

		err := rows.Scan(scanArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan location row: %w", err)
		}

		// Calculate actual distance using Go function for accuracy
		if options.NearLatitude != nil && options.NearLongitude != nil {
			actualDistance := location.DistanceFrom(*options.NearLatitude, *options.NearLongitude)
			distance = &actualDistance
		}

		locations = append(locations, location)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating location rows: %w", err)
	}

	return locations, nil
}

// GetByUser returns all locations for a user
func (r *LocationRepository) GetByUser(userID string, limit, offset int) ([]*models.Location, error) {
	options := LocationSearchOptions{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
		OrderBy: "name",
		OrderDirection: "ASC",
	}
	return r.Search(options)
}

// GetByCategory returns all locations in a specific category for a user
func (r *LocationRepository) GetByCategory(userID, category string, limit, offset int) ([]*models.Location, error) {
	options := LocationSearchOptions{
		UserID:   userID,
		Category: &category,
		Limit:    limit,
		Offset:   offset,
		OrderBy:  "name",
		OrderDirection: "ASC",
	}
	return r.Search(options)
}

// GetNearby returns locations near the given coordinates within a specified radius
func (r *LocationRepository) GetNearby(userID string, latitude, longitude, radiusMeters float64, limit, offset int) ([]*models.Location, error) {
	options := LocationSearchOptions{
		UserID:        userID,
		NearLatitude:  &latitude,
		NearLongitude: &longitude,
		WithinMeters:  &radiusMeters,
		Limit:         limit,
		Offset:        offset,
		OrderBy:       "distance",
		OrderDirection: "ASC",
	}
	return r.Search(options)
}

// FindAtCoordinates finds locations that contain the given coordinates within their radius
func (r *LocationRepository) FindAtCoordinates(userID string, latitude, longitude float64) ([]*models.Location, error) {
	// Get all user locations and filter by those containing the coordinates
	query := `
		SELECT id, user_id, name, address, latitude, longitude, 
		       radius, category, place_id, metadata, created_at, updated_at,
		       (6371000 * acos(
				cos(radians(?)) * cos(radians(latitude)) * 
				cos(radians(longitude) - radians(?)) + 
				sin(radians(?)) * sin(radians(latitude))
			)) as distance
		FROM locations 
		WHERE user_id = ? 
		AND (6371000 * acos(
			cos(radians(?)) * cos(radians(latitude)) * 
			cos(radians(longitude) - radians(?)) + 
			sin(radians(?)) * sin(radians(latitude))
		)) <= radius
		ORDER BY distance ASC`

	rows, err := r.db.Query(query, latitude, longitude, latitude, userID, latitude, longitude, latitude)
	if err != nil {
		return nil, fmt.Errorf("failed to find locations at coordinates: %w", err)
	}
	defer rows.Close()

	var locations []*models.Location
	for rows.Next() {
		location := &models.Location{}
		var distance float64

		err := rows.Scan(
			&location.ID,
			&location.UserID,
			&location.Name,
			&location.Address,
			&location.Latitude,
			&location.Longitude,
			&location.Radius,
			&location.Category,
			&location.PlaceID,
			&location.Metadata,
			&location.CreatedAt,
			&location.UpdatedAt,
			&distance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan location row: %w", err)
		}

		locations = append(locations, location)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating location rows: %w", err)
	}

	return locations, nil
}

// FullTextSearch performs a full-text search on location names and addresses
func (r *LocationRepository) FullTextSearch(userID, query string, limit, offset int) ([]*models.Location, error) {
	options := LocationSearchOptions{
		UserID: userID,
		Query:  query,
		Limit:  limit,
		Offset: offset,
		OrderBy: "name",
		OrderDirection: "ASC",
	}
	return r.Search(options)
}

// GetCategories returns all unique categories for a user's locations
func (r *LocationRepository) GetCategories(userID string) ([]string, error) {
	query := `
		SELECT DISTINCT category 
		FROM locations 
		WHERE user_id = ? 
		ORDER BY category ASC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get location categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
}

// Count returns the total number of locations matching the search options
func (r *LocationRepository) Count(options LocationSearchOptions) (int, error) {
	var conditions []string
	var args []interface{}

	// Build query conditions (similar to Search method but for count)
	var fromClause string
	if options.Query != "" {
		fromClause = `
			FROM locations l
			JOIN locations_fts fts ON l.rowid = fts.rowid
		`
		conditions = append(conditions, "locations_fts MATCH ?")
		args = append(args, options.Query)
	} else {
		fromClause = "FROM locations l"
	}

	if options.UserID != "" {
		conditions = append(conditions, "l.user_id = ?")
		args = append(args, options.UserID)
	}

	if options.Category != nil {
		conditions = append(conditions, "l.category = ?")
		args = append(args, *options.Category)
	}

	if options.NearLatitude != nil && options.NearLongitude != nil && options.WithinMeters != nil {
		haversineSQL := fmt.Sprintf(`
			(6371000 * acos(
				cos(radians(%f)) * cos(radians(l.latitude)) * 
				cos(radians(l.longitude) - radians(%f)) + 
				sin(radians(%f)) * sin(radians(l.latitude))
			)) <= %f`,
			*options.NearLatitude, *options.NearLongitude,
			*options.NearLatitude, *options.WithinMeters)
		
		conditions = append(conditions, haversineSQL)
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := "SELECT COUNT(*) " + fromClause + " " + whereClause

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count locations: %w", err)
	}

	return count, nil
}

// UpdateMetadata updates a location's metadata
func (r *LocationRepository) UpdateMetadata(locationID string, metadata map[string]interface{}) error {
	if locationID == "" {
		return fmt.Errorf("location ID cannot be empty")
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `UPDATE locations SET metadata = ?, updated_at = ? WHERE id = ?`
	_, err = r.db.Exec(query, metadataJSON, time.Now(), locationID)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// Exists checks if a location exists by ID
func (r *LocationRepository) Exists(locationID string) (bool, error) {
	if locationID == "" {
		return false, fmt.Errorf("location ID cannot be empty")
	}

	var count int
	query := `SELECT COUNT(*) FROM locations WHERE id = ?`
	
	err := r.db.QueryRow(query, locationID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check location existence: %w", err)
	}

	return count > 0, nil
}

// haversineDistance calculates the distance between two geographic points
// This is a helper function that matches the one in the Location model
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}