package models

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type Location struct {
	ID        string          `db:"id" json:"id"`
	UserID    string          `db:"user_id" json:"user_id"`
	Name      string          `db:"name" json:"name"`
	Address   string          `db:"address" json:"address"`
	Latitude  float64         `db:"latitude" json:"latitude"`
	Longitude float64         `db:"longitude" json:"longitude"`
	Radius    int             `db:"radius" json:"radius"`
	Category  string          `db:"category" json:"category"`
	PlaceID   *string         `db:"place_id" json:"place_id"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

const (
	EarthRadiusMeters = 6371000.0
)

func NewLocation(userID, name, address string, latitude, longitude float64, radius int) (*Location, error) {
	if err := validateLocationName(name); err != nil {
		return nil, err
	}

	if err := validateCoordinates(latitude, longitude); err != nil {
		return nil, err
	}

	if err := validateRadius(radius); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Location{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		Address:   address,
		Latitude:  latitude,
		Longitude: longitude,
		Radius:    radius,
		Category:  "general",
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  json.RawMessage(`{}`),
	}, nil
}

func (l *Location) SetName(name string) error {
	if err := validateLocationName(name); err != nil {
		return err
	}
	l.Name = name
	l.UpdatedAt = time.Now()
	return nil
}

func (l *Location) SetAddress(address string) {
	l.Address = address
	l.UpdatedAt = time.Now()
}

func (l *Location) SetCoordinates(latitude, longitude float64) error {
	if err := validateCoordinates(latitude, longitude); err != nil {
		return err
	}
	l.Latitude = latitude
	l.Longitude = longitude
	l.UpdatedAt = time.Now()
	return nil
}

func (l *Location) SetRadius(radius int) error {
	if err := validateRadius(radius); err != nil {
		return err
	}
	l.Radius = radius
	l.UpdatedAt = time.Now()
	return nil
}

func (l *Location) SetCategory(category string) {
	l.Category = category
	l.UpdatedAt = time.Now()
}

func (l *Location) SetPlaceID(placeID string) {
	l.PlaceID = &placeID
	l.UpdatedAt = time.Now()
}

func (l *Location) ClearPlaceID() {
	l.PlaceID = nil
	l.UpdatedAt = time.Now()
}

func (l *Location) DistanceFrom(latitude, longitude float64) float64 {
	return haversineDistance(l.Latitude, l.Longitude, latitude, longitude)
}

func (l *Location) IsWithinRadius(latitude, longitude float64) bool {
	distance := l.DistanceFrom(latitude, longitude)
	return distance <= float64(l.Radius)
}

func (l *Location) IsOwnedBy(userID string) bool {
	return l.UserID == userID
}

func (l *Location) Validate() error {
	if err := validateLocationName(l.Name); err != nil {
		return err
	}

	if l.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if err := validateCoordinates(l.Latitude, l.Longitude); err != nil {
		return err
	}

	if err := validateRadius(l.Radius); err != nil {
		return err
	}

	return nil
}

func validateLocationName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(name) > 200 {
		return fmt.Errorf("name must not exceed 200 characters")
	}
	return nil
}

func validateCoordinates(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90 degrees")
	}
	if longitude < -180 || longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180 degrees")
	}
	return nil
}

func validateRadius(radius int) error {
	if radius <= 0 {
		return fmt.Errorf("radius must be positive")
	}
	if radius > 10000 {
		return fmt.Errorf("radius cannot exceed 10,000 meters")
	}
	return nil
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
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

	return EarthRadiusMeters * c
}