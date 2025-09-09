package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Context struct {
	ID                string          `db:"id" json:"id"`
	UserID            string          `db:"user_id" json:"user_id"`
	Timestamp         time.Time       `db:"timestamp" json:"timestamp"`
	CurrentLatitude   *float64        `db:"current_latitude" json:"current_latitude"`
	CurrentLongitude  *float64        `db:"current_longitude" json:"current_longitude"`
	CurrentLocationID *string         `db:"current_location_id" json:"current_location_id"`
	AvailableMinutes  int             `db:"available_minutes" json:"available_minutes"`
	SocialContext     string          `db:"social_context" json:"social_context"`
	EnergyLevel       int             `db:"energy_level" json:"energy_level"`
	WeatherCondition  *string         `db:"weather_condition" json:"weather_condition"`
	TrafficLevel      *string         `db:"traffic_level" json:"traffic_level"`
	Metadata          json.RawMessage `db:"metadata" json:"metadata"`
}

const (
	SocialContextAlone      = "alone"
	SocialContextWithFamily = "with_family"
	SocialContextAtWork     = "at_work"
	SocialContextInPublic   = "in_public"
	SocialContextDriving    = "driving"
)

const (
	WeatherSunny   = "sunny"
	WeatherCloudy  = "cloudy"
	WeatherRainy   = "rainy"
	WeatherSnowy   = "snowy"
	WeatherStormy  = "stormy"
	WeatherFoggy   = "foggy"
)

const (
	TrafficLow      = "low"
	TrafficModerate = "moderate"
	TrafficHeavy    = "heavy"
)

func NewContext(userID string, availableMinutes, energyLevel int) (*Context, error) {
	if err := validateEnergyLevel(energyLevel); err != nil {
		return nil, err
	}

	if availableMinutes < 0 {
		return nil, fmt.Errorf("available minutes cannot be negative")
	}

	return &Context{
		ID:               uuid.New().String(),
		UserID:           userID,
		Timestamp:        time.Now(),
		AvailableMinutes: availableMinutes,
		SocialContext:    SocialContextAlone,
		EnergyLevel:      energyLevel,
		Metadata:         json.RawMessage(`{}`),
	}, nil
}

func (c *Context) SetCurrentPosition(latitude, longitude float64) error {
	if err := validateCoordinates(latitude, longitude); err != nil {
		return err
	}
	c.CurrentLatitude = &latitude
	c.CurrentLongitude = &longitude
	return nil
}

func (c *Context) ClearCurrentPosition() {
	c.CurrentLatitude = nil
	c.CurrentLongitude = nil
}

func (c *Context) SetCurrentLocation(locationID string) {
	c.CurrentLocationID = &locationID
}

func (c *Context) ClearCurrentLocation() {
	c.CurrentLocationID = nil
}

func (c *Context) SetAvailableMinutes(minutes int) error {
	if minutes < 0 {
		return fmt.Errorf("available minutes cannot be negative")
	}
	c.AvailableMinutes = minutes
	return nil
}

func (c *Context) SetSocialContext(socialContext string) error {
	if !isValidSocialContext(socialContext) {
		return fmt.Errorf("invalid social context: %s", socialContext)
	}
	c.SocialContext = socialContext
	return nil
}

func (c *Context) SetEnergyLevel(energyLevel int) error {
	if err := validateEnergyLevel(energyLevel); err != nil {
		return err
	}
	c.EnergyLevel = energyLevel
	return nil
}

func (c *Context) SetWeatherCondition(condition string) error {
	if !isValidWeatherCondition(condition) {
		return fmt.Errorf("invalid weather condition: %s", condition)
	}
	c.WeatherCondition = &condition
	return nil
}

func (c *Context) ClearWeatherCondition() {
	c.WeatherCondition = nil
}

func (c *Context) SetTrafficLevel(level string) error {
	if !isValidTrafficLevel(level) {
		return fmt.Errorf("invalid traffic level: %s", level)
	}
	c.TrafficLevel = &level
	return nil
}

func (c *Context) ClearTrafficLevel() {
	c.TrafficLevel = nil
}

func (c *Context) HasCurrentPosition() bool {
	return c.CurrentLatitude != nil && c.CurrentLongitude != nil
}

func (c *Context) HasCurrentLocation() bool {
	return c.CurrentLocationID != nil
}

func (c *Context) IsAtLocation(location *Location) bool {
	if !c.HasCurrentPosition() || location == nil {
		return false
	}
	return location.IsWithinRadius(*c.CurrentLatitude, *c.CurrentLongitude)
}

func (c *Context) HasEnoughTime(estimatedMinutes int) bool {
	return c.AvailableMinutes >= estimatedMinutes
}

func (c *Context) HasEnoughEnergy(requiredEnergyLevel int) bool {
	return c.EnergyLevel >= requiredEnergyLevel
}

func (c *Context) IsOwnedBy(userID string) bool {
	return c.UserID == userID
}

func (c *Context) Validate() error {
	if c.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if c.AvailableMinutes < 0 {
		return fmt.Errorf("available minutes cannot be negative")
	}

	if err := validateEnergyLevel(c.EnergyLevel); err != nil {
		return err
	}

	if !isValidSocialContext(c.SocialContext) {
		return fmt.Errorf("invalid social context: %s", c.SocialContext)
	}

	if c.CurrentLatitude != nil && c.CurrentLongitude != nil {
		if err := validateCoordinates(*c.CurrentLatitude, *c.CurrentLongitude); err != nil {
			return err
		}
	}

	if c.WeatherCondition != nil && !isValidWeatherCondition(*c.WeatherCondition) {
		return fmt.Errorf("invalid weather condition: %s", *c.WeatherCondition)
	}

	if c.TrafficLevel != nil && !isValidTrafficLevel(*c.TrafficLevel) {
		return fmt.Errorf("invalid traffic level: %s", *c.TrafficLevel)
	}

	return nil
}

func validateEnergyLevel(energyLevel int) error {
	if energyLevel < 1 || energyLevel > 5 {
		return fmt.Errorf("energy level must be between 1 and 5")
	}
	return nil
}

func isValidSocialContext(context string) bool {
	validContexts := []string{
		SocialContextAlone,
		SocialContextWithFamily,
		SocialContextAtWork,
		SocialContextInPublic,
		SocialContextDriving,
	}

	for _, valid := range validContexts {
		if context == valid {
			return true
		}
	}
	return false
}

func isValidWeatherCondition(condition string) bool {
	validConditions := []string{
		WeatherSunny,
		WeatherCloudy,
		WeatherRainy,
		WeatherSnowy,
		WeatherStormy,
		WeatherFoggy,
	}

	for _, valid := range validConditions {
		if condition == valid {
			return true
		}
	}
	return false
}

func isValidTrafficLevel(level string) bool {
	validLevels := []string{
		TrafficLow,
		TrafficModerate,
		TrafficHeavy,
	}

	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}