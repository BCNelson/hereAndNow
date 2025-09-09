package filters

import (
	"fmt"
	"math"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

type LocationFilter struct {
	config        FilterConfig
	locationRepo  LocationRepository
	taskLocations TaskLocationRepository
}

type LocationRepository interface {
	GetByID(locationID string) (*models.Location, error)
	GetByUserID(userID string) ([]models.Location, error)
}

type TaskLocationRepository interface {
	GetLocationsByTaskID(taskID string) ([]models.Location, error)
}

func NewLocationFilter(config FilterConfig, locationRepo LocationRepository, taskLocRepo TaskLocationRepository) *LocationFilter {
	return &LocationFilter{
		config:        config,
		locationRepo:  locationRepo,
		taskLocations: taskLocRepo,
	}
}

func (f *LocationFilter) Name() string {
	return "location"
}

func (f *LocationFilter) Priority() int {
	return 100
}

func (f *LocationFilter) Apply(ctx models.Context, task models.Task) (visible bool, reason string) {
	if !f.config.EnableLocationFilter {
		return true, "location filtering disabled"
	}

	if ctx.CurrentLatitude == nil || ctx.CurrentLongitude == nil {
		return true, "current location unknown - showing all tasks"
	}

	taskLocations, err := f.taskLocations.GetLocationsByTaskID(task.ID)
	if err != nil {
		return false, fmt.Sprintf("error fetching task locations: %v", err)
	}

	if len(taskLocations) == 0 {
		return true, "task has no location requirements"
	}

	currentLat := *ctx.CurrentLatitude
	currentLon := *ctx.CurrentLongitude

	for _, location := range taskLocations {
		distance := f.calculateDistance(currentLat, currentLon, location.Latitude, location.Longitude)
		maxDistance := float64(location.Radius)
		
		if maxDistance == 0 {
			maxDistance = f.config.MaxDistanceMeters
		}

		if distance <= maxDistance {
			return true, fmt.Sprintf("within %dm of %s (%.0fm away)", int(maxDistance), location.Name, distance)
		}
	}

	nearestLocation := f.findNearestLocation(currentLat, currentLon, taskLocations)
	if nearestLocation != nil {
		distance := f.calculateDistance(currentLat, currentLon, nearestLocation.Latitude, nearestLocation.Longitude)
		return false, fmt.Sprintf("too far from %s (%.0fm away, need to be within %dm)", 
			nearestLocation.Name, distance, nearestLocation.Radius)
	}

	return false, "not within range of any required locations"
}

func (f *LocationFilter) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
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

func (f *LocationFilter) findNearestLocation(currentLat, currentLon float64, locations []models.Location) *models.Location {
	if len(locations) == 0 {
		return nil
	}

	nearest := &locations[0]
	minDistance := f.calculateDistance(currentLat, currentLon, nearest.Latitude, nearest.Longitude)

	for i := 1; i < len(locations); i++ {
		distance := f.calculateDistance(currentLat, currentLon, locations[i].Latitude, locations[i].Longitude)
		if distance < minDistance {
			minDistance = distance
			nearest = &locations[i]
		}
	}

	return nearest
}