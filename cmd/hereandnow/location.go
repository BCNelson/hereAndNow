package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
)

func handleLocationCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: location requires a subcommand")
		fmt.Println("Run 'hereandnow location --help' for usage")
		os.Exit(1)
	}

	if args[0] == "--help" || args[0] == "-h" {
		fmt.Printf(`Location Management Commands

USAGE:
    hereandnow location <SUBCOMMAND> [OPTIONS]

SUBCOMMANDS:
    add                 Create a new location
    list                List all locations
    show <name>         Show location details
    update <name>       Update location information
    delete <name>       Delete a location
    nearby              Find locations near current position

OPTIONS:
    --name <name>       Location name (required for add)
    --lat <latitude>    Latitude coordinate (required for add)
    --lng <longitude>   Longitude coordinate (required for add)
    --radius <meters>   Location radius in meters (default: 100)
    --help, -h          Show this help

EXAMPLES:
    # Add a location
    hereandnow location add --name "Home" --lat 37.7749 --lng -122.4194 --radius 100

    # Add work location
    hereandnow location add --name "Office" --lat 37.7858 --lng -122.4065 --radius 200

    # List all locations
    hereandnow location list

    # Show specific location
    hereandnow location show "Home"

    # Update location radius
    hereandnow location update "Office" --radius 150

    # Find nearby locations (requires current context with GPS)
    hereandnow location nearby
`)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "add":
		executeLocationAdd(subArgs)
	case "list":
		executeLocationList(subArgs)
	case "show":
		executeLocationShow(subArgs)
	case "update":
		executeLocationUpdate(subArgs)
	case "delete":
		executeLocationDelete(subArgs)
	case "nearby":
		executeLocationNearby(subArgs)
	default:
		fmt.Printf("Unknown location subcommand: %s\n", subcommand)
		fmt.Println("Run 'hereandnow location --help' for usage")
		os.Exit(1)
	}
}

func executeLocationAdd(args []string) {
	name := ""
	lat := 0.0
	lng := 0.0
	radius := 100

	for i, arg := range args {
		switch arg {
		case "--name":
			if i+1 < len(args) {
				name = args[i+1]
			}
		case "--lat":
			if i+1 < len(args) {
				if l, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					lat = l
				}
			}
		case "--lng":
			if i+1 < len(args) {
				if l, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					lng = l
				}
			}
		case "--radius":
			if i+1 < len(args) {
				if r, err := strconv.Atoi(args[i+1]); err == nil {
					radius = r
				}
			}
		}
	}

	// Validate required fields
	if name == "" {
		fmt.Fprintf(os.Stderr, "Error: --name is required\n")
		os.Exit(1)
	}

	if lat == 0.0 || lng == 0.0 {
		fmt.Fprintf(os.Stderr, "Error: --lat and --lng are required\n")
		os.Exit(1)
	}

	// Validate coordinates
	if lat < -90 || lat > 90 {
		fmt.Fprintf(os.Stderr, "Error: Latitude must be between -90 and 90\n")
		os.Exit(1)
	}

	if lng < -180 || lng > 180 {
		fmt.Fprintf(os.Stderr, "Error: Longitude must be between -180 and 180\n")
		os.Exit(1)
	}

	if radius < 1 || radius > 10000 {
		fmt.Fprintf(os.Stderr, "Error: Radius must be between 1 and 10000 meters\n")
		os.Exit(1)
	}

	// Get current user
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	// Initialize database
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	locationRepo := storage.NewLocationRepository(db)

	// Check if location with this name already exists for user
	existingLocations, err := locationRepo.GetByUserID(userID)
	if err == nil {
		for _, loc := range existingLocations {
			if loc.Name == name {
				fmt.Fprintf(os.Stderr, "Error: Location with name '%s' already exists\n", name)
				os.Exit(1)
			}
		}
	}

	// Create location
	location := models.Location{
		ID:        uuid.New().String(),
		Name:      name,
		Latitude:  lat,
		Longitude: lng,
		Radius:    radius,
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := locationRepo.Create(location); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating location: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("Location '%s' created successfully", name))
}

func executeLocationList(args []string) {
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	locationRepo := storage.NewLocationRepository(db)

	locations, err := locationRepo.GetByUserID(userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving locations: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, locations)
}

func executeLocationShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: location show requires name\n")
		fmt.Println("Usage: hereandnow location show <name>")
		os.Exit(1)
	}

	name := args[0]
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	location, err := findLocationByNameForUser(name, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Location '%s' not found\n", name)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, *location)
}

func executeLocationUpdate(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: location update requires name\n")
		fmt.Println("Usage: hereandnow location update <name> [OPTIONS]")
		os.Exit(1)
	}

	name := args[0]
	var lat, lng *float64
	var radius *int

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--lat":
			if i+1 < len(args) {
				if l, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					lat = &l
					i++
				}
			}
		case "--lng":
			if i+1 < len(args) {
				if l, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					lng = &l
					i++
				}
			}
		case "--radius":
			if i+1 < len(args) {
				if r, err := strconv.Atoi(args[i+1]); err == nil {
					radius = &r
					i++
				}
			}
		}
	}

	if lat == nil && lng == nil && radius == nil {
		fmt.Fprintf(os.Stderr, "Error: At least one field must be updated\n")
		fmt.Println("Available options: --lat, --lng, --radius")
		os.Exit(1)
	}

	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	location, err := findLocationByNameForUser(name, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Location '%s' not found\n", name)
		os.Exit(1)
	}

	// Update fields
	if lat != nil {
		if *lat < -90 || *lat > 90 {
			fmt.Fprintf(os.Stderr, "Error: Latitude must be between -90 and 90\n")
			os.Exit(1)
		}
		location.Latitude = *lat
	}

	if lng != nil {
		if *lng < -180 || *lng > 180 {
			fmt.Fprintf(os.Stderr, "Error: Longitude must be between -180 and 180\n")
			os.Exit(1)
		}
		location.Longitude = *lng
	}

	if radius != nil {
		if *radius < 1 || *radius > 10000 {
			fmt.Fprintf(os.Stderr, "Error: Radius must be between 1 and 10000 meters\n")
			os.Exit(1)
		}
		location.Radius = *radius
	}

	location.UpdatedAt = time.Now()

	// Save updated location
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	locationRepo := storage.NewLocationRepository(db)

	if err := locationRepo.Update(*location); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating location: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("Location '%s' updated successfully", name))
}

func executeLocationDelete(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: location delete requires name\n")
		fmt.Println("Usage: hereandnow location delete <name>")
		os.Exit(1)
	}

	name := args[0]
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	location, err := findLocationByNameForUser(name, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Location '%s' not found\n", name)
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	locationRepo := storage.NewLocationRepository(db)

	if err := locationRepo.Delete(location.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting location: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("Location '%s' deleted successfully", name))
}

func executeLocationNearby(args []string) {
	radius := 1000 // Default 1km radius

	for i, arg := range args {
		switch arg {
		case "--radius":
			if i+1 < len(args) {
				if r, err := strconv.Atoi(args[i+1]); err == nil {
					radius = r
				}
			}
		}
	}

	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	// Get current context to find user's location
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	contextRepo := storage.NewContextRepository(db)
	locationRepo := storage.NewLocationRepository(db)

	context, err := contextRepo.GetLatestByUserID(userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: No current context found. Update your location first with 'hereandnow context update'\n")
		os.Exit(1)
	}

	if context.CurrentLatitude == nil || context.CurrentLongitude == nil {
		fmt.Fprintf(os.Stderr, "Error: Current location unknown. Update your context with GPS coordinates\n")
		os.Exit(1)
	}

	nearbyLocations, err := locationRepo.FindNearby(*context.CurrentLatitude, *context.CurrentLongitude, radius)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding nearby locations: %v\n", err)
		os.Exit(1)
	}

	if len(nearbyLocations) == 0 {
		formatter := NewFormatter(globalConfig.Format)
		Output(formatter, fmt.Sprintf("No locations found within %d meters", radius))
		return
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, nearbyLocations)
}

// Helper function to find location by name for a specific user
func findLocationByNameForUser(name, userID string) (*models.Location, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	locationRepo := storage.NewLocationRepository(db)
	locations, err := locationRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	for _, loc := range locations {
		if loc.Name == name {
			return &loc, nil
		}
	}

	return nil, fmt.Errorf("location not found: %s", name)
}