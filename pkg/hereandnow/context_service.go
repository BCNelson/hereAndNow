package hereandnow

import (
	"fmt"
	"math"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
)

type ContextService struct {
	contextRepo     ContextRepository
	locationRepo    LocationRepository
	calendarRepo    CalendarEventRepository
	weatherService  WeatherService
	trafficService  TrafficService
}

type LocationRepository interface {
	GetByID(locationID string) (*models.Location, error)
	GetByUserID(userID string) ([]models.Location, error)
	FindNearby(latitude, longitude float64, radiusMeters int) ([]models.Location, error)
}

type CalendarEventRepository interface {
	GetEventsByUserIDAndTimeRange(userID string, start, end time.Time) ([]models.CalendarEvent, error)
	GetNextEvent(userID string, after time.Time) (*models.CalendarEvent, error)
}

type WeatherService interface {
	GetCurrentWeather(latitude, longitude float64) (*WeatherInfo, error)
}

type TrafficService interface {
	GetTrafficLevel(latitude, longitude float64) (*TrafficInfo, error)
}

type WeatherInfo struct {
	Condition   string  `json:"condition"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

type TrafficInfo struct {
	Level       string `json:"level"`
	Congestion  int    `json:"congestion"`
	Description string `json:"description"`
}

func NewContextService(
	contextRepo ContextRepository,
	locationRepo LocationRepository,
	calendarRepo CalendarEventRepository,
	weatherService WeatherService,
	trafficService TrafficService,
) *ContextService {
	return &ContextService{
		contextRepo:    contextRepo,
		locationRepo:   locationRepo,
		calendarRepo:   calendarRepo,
		weatherService: weatherService,
		trafficService: trafficService,
	}
}

func (s *ContextService) UpdateUserContext(userID string, req UpdateContextRequest) (*models.Context, error) {
	context := models.Context{
		ID:                uuid.New().String(),
		UserID:            userID,
		Timestamp:         time.Now(),
		CurrentLatitude:   req.Latitude,
		CurrentLongitude:  req.Longitude,
		CurrentLocationID: req.LocationID,
		AvailableMinutes:  req.AvailableMinutes,
		SocialContext:     req.SocialContext,
		EnergyLevel:       req.EnergyLevel,
		WeatherCondition:  req.WeatherCondition,
		TrafficLevel:      req.TrafficLevel,
		Metadata:          req.Metadata,
	}

	if req.Latitude != nil && req.Longitude != nil {
		if err := s.enrichContextWithLocation(&context); err != nil {
			return nil, fmt.Errorf("failed to enrich context with location: %w", err)
		}

		if err := s.enrichContextWithWeather(&context); err != nil {
			return nil, fmt.Errorf("failed to enrich context with weather: %w", err)
		}

		if err := s.enrichContextWithTraffic(&context); err != nil {
			return nil, fmt.Errorf("failed to enrich context with traffic: %w", err)
		}
	}

	if context.AvailableMinutes == 0 {
		availableMinutes, err := s.calculateAvailableMinutes(userID, context.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate available minutes: %w", err)
		}
		context.AvailableMinutes = availableMinutes
	}

	if err := s.contextRepo.Create(context); err != nil {
		return nil, fmt.Errorf("failed to save context: %w", err)
	}

	return &context, nil
}

func (s *ContextService) GetCurrentContext(userID string) (*models.Context, error) {
	context, err := s.contextRepo.GetLatestByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current context: %w", err)
	}

	if time.Since(context.Timestamp) > 15*time.Minute {
		context, err = s.refreshContext(userID, *context)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh context: %w", err)
		}
	}

	return context, nil
}

func (s *ContextService) CreateContextFromLocation(userID string, latitude, longitude float64) (*models.Context, error) {
	availableMinutes, err := s.calculateAvailableMinutes(userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to calculate available minutes: %w", err)
	}

	context := models.Context{
		ID:               uuid.New().String(),
		UserID:           userID,
		Timestamp:        time.Now(),
		CurrentLatitude:  &latitude,
		CurrentLongitude: &longitude,
		AvailableMinutes: availableMinutes,
		SocialContext:    models.SocialContextAlone,
		EnergyLevel:      3,
	}

	if err := s.enrichContextWithLocation(&context); err != nil {
		return nil, fmt.Errorf("failed to enrich context: %w", err)
	}

	if err := s.enrichContextWithWeather(&context); err != nil {
		return nil, fmt.Errorf("failed to enrich weather: %w", err)
	}

	if err := s.enrichContextWithTraffic(&context); err != nil {
		return nil, fmt.Errorf("failed to enrich traffic: %w", err)
	}

	if err := s.contextRepo.Create(context); err != nil {
		return nil, fmt.Errorf("failed to save context: %w", err)
	}

	return &context, nil
}

func (s *ContextService) enrichContextWithLocation(context *models.Context) error {
	if context.CurrentLatitude == nil || context.CurrentLongitude == nil {
		return nil
	}

	nearbyLocations, err := s.locationRepo.FindNearby(
		*context.CurrentLatitude,
		*context.CurrentLongitude,
		500,
	)
	if err != nil {
		return err
	}

	if len(nearbyLocations) > 0 {
		closest := s.findClosestLocation(*context.CurrentLatitude, *context.CurrentLongitude, nearbyLocations)
		if closest != nil {
			context.CurrentLocationID = &closest.ID
		}
	}

	return nil
}

func (s *ContextService) enrichContextWithWeather(context *models.Context) error {
	if context.CurrentLatitude == nil || context.CurrentLongitude == nil {
		return nil
	}

	if s.weatherService == nil {
		return nil
	}

	weather, err := s.weatherService.GetCurrentWeather(*context.CurrentLatitude, *context.CurrentLongitude)
	if err != nil {
		return nil
	}

	context.WeatherCondition = &weather.Condition
	return nil
}

func (s *ContextService) enrichContextWithTraffic(context *models.Context) error {
	if context.CurrentLatitude == nil || context.CurrentLongitude == nil {
		return nil
	}

	if s.trafficService == nil {
		return nil
	}

	traffic, err := s.trafficService.GetTrafficLevel(*context.CurrentLatitude, *context.CurrentLongitude)
	if err != nil {
		return nil
	}

	context.TrafficLevel = &traffic.Level
	return nil
}

func (s *ContextService) calculateAvailableMinutes(userID string, timestamp time.Time) (int, error) {
	endOfDay := time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 23, 59, 59, 0, timestamp.Location())
	
	events, err := s.calendarRepo.GetEventsByUserIDAndTimeRange(userID, timestamp, endOfDay)
	if err != nil {
		return 120, nil
	}

	if len(events) == 0 {
		remaining := int(endOfDay.Sub(timestamp).Minutes())
		if remaining > 480 {
			return 480, nil
		}
		return remaining, nil
	}

	nextEvent, err := s.calendarRepo.GetNextEvent(userID, timestamp)
	if err != nil || nextEvent == nil {
		remaining := int(endOfDay.Sub(timestamp).Minutes())
		if remaining > 240 {
			return 240, nil
		}
		return remaining, nil
	}

	availableMinutes := int(nextEvent.StartAt.Sub(timestamp).Minutes())
	if availableMinutes < 0 {
		return 0, nil
	}
	if availableMinutes > 240 {
		return 240, nil
	}

	return availableMinutes, nil
}

func (s *ContextService) refreshContext(userID string, oldContext models.Context) (*models.Context, error) {
	newContext := models.Context{
		ID:                uuid.New().String(),
		UserID:            userID,
		Timestamp:         time.Now(),
		CurrentLatitude:   oldContext.CurrentLatitude,
		CurrentLongitude:  oldContext.CurrentLongitude,
		CurrentLocationID: oldContext.CurrentLocationID,
		SocialContext:     oldContext.SocialContext,
		EnergyLevel:       oldContext.EnergyLevel,
	}

	availableMinutes, err := s.calculateAvailableMinutes(userID, newContext.Timestamp)
	if err != nil {
		return nil, err
	}
	newContext.AvailableMinutes = availableMinutes

	if newContext.CurrentLatitude != nil && newContext.CurrentLongitude != nil {
		s.enrichContextWithWeather(&newContext)
		s.enrichContextWithTraffic(&newContext)
	}

	if err := s.contextRepo.Create(newContext); err != nil {
		return nil, err
	}

	return &newContext, nil
}

func (s *ContextService) findClosestLocation(latitude, longitude float64, locations []models.Location) *models.Location {
	if len(locations) == 0 {
		return nil
	}

	closest := &locations[0]
	minDistance := s.calculateDistance(latitude, longitude, closest.Latitude, closest.Longitude)

	for i := 1; i < len(locations); i++ {
		distance := s.calculateDistance(latitude, longitude, locations[i].Latitude, locations[i].Longitude)
		if distance < minDistance {
			minDistance = distance
			closest = &locations[i]
		}
	}

	if minDistance <= float64(closest.Radius) {
		return closest
	}

	return nil
}

func (s *ContextService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
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

	return models.EarthRadiusMeters * c
}

func (s *ContextService) GetContextSuggestions(userID string) (*ContextSuggestions, error) {
	currentContext, err := s.GetCurrentContext(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current context: %w", err)
	}

	suggestions := &ContextSuggestions{}

	if currentContext.EnergyLevel <= 2 {
		suggestions.EnergyAdvice = "Consider taking a break or doing lighter tasks"
	}

	if currentContext.AvailableMinutes < 15 {
		suggestions.TimeAdvice = "Focus on quick tasks that can be completed in under 15 minutes"
	} else if currentContext.AvailableMinutes > 120 {
		suggestions.TimeAdvice = "You have plenty of time for longer, more complex tasks"
	}

	if currentContext.CurrentLatitude != nil && currentContext.CurrentLongitude != nil {
		nearbyLocations, err := s.locationRepo.FindNearby(
			*currentContext.CurrentLatitude,
			*currentContext.CurrentLongitude,
			1000,
		)
		if err == nil && len(nearbyLocations) > 0 {
			locationNames := make([]string, len(nearbyLocations))
			for i, loc := range nearbyLocations {
				locationNames[i] = loc.Name
			}
			suggestions.NearbyLocations = locationNames
		}
	}

	return suggestions, nil
}

func (s *ContextService) EstimateTimeToLocation(userID string, locationID string) (*TimeEstimate, error) {
	currentContext, err := s.GetCurrentContext(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current context: %w", err)
	}

	if currentContext.CurrentLatitude == nil || currentContext.CurrentLongitude == nil {
		return nil, fmt.Errorf("current location unknown")
	}

	location, err := s.locationRepo.GetByID(locationID)
	if err != nil {
		return nil, fmt.Errorf("location not found: %w", err)
	}

	distance := s.calculateDistance(
		*currentContext.CurrentLatitude,
		*currentContext.CurrentLongitude,
		location.Latitude,
		location.Longitude,
	)

	estimate := &TimeEstimate{
		DistanceMeters: int(distance),
		WalkingMinutes: int(distance / 83.33), // ~5 km/h walking speed
		DrivingMinutes: int(distance / 500),   // ~30 km/h average city driving
		Location:       *location,
	}

	if estimate.WalkingMinutes < 1 {
		estimate.WalkingMinutes = 1
	}
	if estimate.DrivingMinutes < 1 {
		estimate.DrivingMinutes = 1
	}

	return estimate, nil
}

type UpdateContextRequest struct {
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	LocationID       *string  `json:"location_id"`
	AvailableMinutes int      `json:"available_minutes"`
	SocialContext    string   `json:"social_context"`
	EnergyLevel      int      `json:"energy_level"`
	WeatherCondition *string  `json:"weather_condition"`
	TrafficLevel     *string  `json:"traffic_level"`
	Metadata         []byte   `json:"metadata"`
}

type ContextSuggestions struct {
	TimeAdvice       string   `json:"time_advice"`
	EnergyAdvice     string   `json:"energy_advice"`
	NearbyLocations  []string `json:"nearby_locations"`
	RecommendedTasks []string `json:"recommended_tasks"`
}

type TimeEstimate struct {
	DistanceMeters int             `json:"distance_meters"`
	WalkingMinutes int             `json:"walking_minutes"`
	DrivingMinutes int             `json:"driving_minutes"`
	Location       models.Location `json:"location"`
}