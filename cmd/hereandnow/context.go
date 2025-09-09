package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/hereandnow"
	"github.com/bcnelson/hereAndNow/pkg/models"
)

func handleContextCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: context requires a subcommand")
		fmt.Println("Run 'hereandnow context --help' for usage")
		os.Exit(1)
	}

	if args[0] == "--help" || args[0] == "-h" {
		fmt.Printf(`Context Management Commands

USAGE:
    hereandnow context <SUBCOMMAND> [OPTIONS]

SUBCOMMANDS:
    show                Show current context
    update              Update current context
    suggestions         Get context-based suggestions
    estimate <location> Estimate time to location

DESCRIPTION:
    Context represents your current situation including location, available time,
    energy level, and social context. This information is used to filter tasks
    and show only what can be completed right now.

UPDATE OPTIONS:
    --lat <latitude>        GPS latitude coordinate
    --lng <longitude>       GPS longitude coordinate
    --location <name>       Set location by name (must exist)
    --available-minutes <n> Available time in minutes
    --energy <1-5>          Energy level (1=exhausted, 5=maximum)
    --social <context>      Social context (alone|family|work|friends)
    --help, -h              Show this help

EXAMPLES:
    # Show current context
    hereandnow context show

    # Update GPS location
    hereandnow context update --lat 37.7749 --lng -122.4194

    # Update location by name
    hereandnow context update --location "Office"

    # Update available time and energy
    hereandnow context update --available-minutes 45 --energy 3

    # Update social context
    hereandnow context update --social family

    # Get context-based suggestions
    hereandnow context suggestions

    # Estimate travel time to a location
    hereandnow context estimate "Grocery Store"

SOCIAL CONTEXT VALUES:
    alone    - Working alone, full focus available
    family   - With family, limited work time
    work     - At work, professional tasks preferred
    friends  - Social time, avoid work tasks
`)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "show":
		executeContextShow(subArgs)
	case "update":
		executeContextUpdate(subArgs)
	case "suggestions":
		executeContextSuggestions(subArgs)
	case "estimate":
		executeContextEstimate(subArgs)
	default:
		fmt.Printf("Unknown context subcommand: %s\n", subcommand)
		fmt.Println("Run 'hereandnow context --help' for usage")
		os.Exit(1)
	}
}

func executeContextShow(args []string) {
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	contextService, err := initContextService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing context service: %v\n", err)
		os.Exit(1)
	}

	context, err := contextService.GetCurrentContext(userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: No current context found\n")
		fmt.Println("Use 'hereandnow context update' to set your initial context")
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, *context)
}

func executeContextUpdate(args []string) {
	var lat, lng *float64
	locationName := ""
	availableMinutes := 0
	energyLevel := 0
	socialContext := ""

	for i, arg := range args {
		switch arg {
		case "--lat":
			if i+1 < len(args) {
				if l, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					lat = &l
				}
			}
		case "--lng":
			if i+1 < len(args) {
				if l, err := strconv.ParseFloat(args[i+1], 64); err == nil {
					lng = &l
				}
			}
		case "--location":
			if i+1 < len(args) {
				locationName = args[i+1]
			}
		case "--available-minutes":
			if i+1 < len(args) {
				if m, err := strconv.Atoi(args[i+1]); err == nil {
					availableMinutes = m
				}
			}
		case "--energy":
			if i+1 < len(args) {
				if e, err := strconv.Atoi(args[i+1]); err == nil && e >= 1 && e <= 5 {
					energyLevel = e
				}
			}
		case "--social":
			if i+1 < len(args) {
				social := args[i+1]
				if social == "alone" || social == "family" || social == "work" || social == "friends" {
					socialContext = social
				}
			}
		}
	}

	// Validate GPS coordinates if provided
	if lat != nil {
		if *lat < -90 || *lat > 90 {
			fmt.Fprintf(os.Stderr, "Error: Latitude must be between -90 and 90\n")
			os.Exit(1)
		}
	}

	if lng != nil {
		if *lng < -180 || *lng > 180 {
			fmt.Fprintf(os.Stderr, "Error: Longitude must be between -180 and 180\n")
			os.Exit(1)
		}
	}

	// If both GPS and location name provided, prefer GPS
	if lat != nil && lng != nil && locationName != "" {
		fmt.Println("Note: Both GPS coordinates and location name provided. Using GPS coordinates.")
		locationName = ""
	}

	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	contextService, err := initContextService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing context service: %v\n", err)
		os.Exit(1)
	}

	// Handle location name resolution
	var locationID *string
	if locationName != "" {
		location, err := findLocationByNameForUser(locationName, userID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Location '%s' not found\n", locationName)
			os.Exit(1)
		}
		locationID = &location.ID
		// Set GPS coordinates from the named location
		lat = &location.Latitude
		lng = &location.Longitude
	}

	// Build update request
	req := hereandnow.UpdateContextRequest{
		Latitude:         lat,
		Longitude:        lng,
		LocationID:       locationID,
		AvailableMinutes: availableMinutes,
		SocialContext:    socialContext,
		EnergyLevel:      energyLevel,
	}

	context, err := contextService.UpdateUserContext(userID, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating context: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, "Context updated successfully")

	if globalConfig.Verbose {
		Output(formatter, *context)
	}
}

func executeContextSuggestions(args []string) {
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	contextService, err := initContextService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing context service: %v\n", err)
		os.Exit(1)
	}

	suggestions, err := contextService.GetContextSuggestions(userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting suggestions: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, *suggestions)
}

func executeContextEstimate(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: context estimate requires location name\n")
		fmt.Println("Usage: hereandnow context estimate <location>")
		os.Exit(1)
	}

	locationName := args[0]
	userID := getCurrentUserID()
	if userID == "" {
		fmt.Fprintf(os.Stderr, "Error: No current user\n")
		os.Exit(1)
	}

	// Find the location
	location, err := findLocationByNameForUser(locationName, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Location '%s' not found\n", locationName)
		os.Exit(1)
	}

	contextService, err := initContextService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing context service: %v\n", err)
		os.Exit(1)
	}

	estimate, err := contextService.EstimateTimeToLocation(userID, location.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error estimating travel time: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, *estimate)
}

// Helper function to initialize context service
func initContextService() (*hereandnow.ContextService, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		return nil, err
	}

	contextRepo := storage.NewContextRepository(db)
	locationRepo := storage.NewLocationRepository(db)
	// Calendar repository would be needed for full functionality
	// For now, we'll pass nil for optional services

	return hereandnow.NewContextService(contextRepo, locationRepo, nil, nil, nil), nil
}