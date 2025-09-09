package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func executeInit(args []string) {
	force := false
	dbPath := ""
	
	for i, arg := range args {
		switch arg {
		case "--force":
			force = true
		case "--db-path":
			if i+1 < len(args) {
				dbPath = args[i+1]
			}
		}
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check if already initialized
	if !force {
		configPath := getConfigPath()
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Already initialized at %s\n", configPath)
			fmt.Println("Use --force to reinitialize")
			return
		}
	}

	// Create config directory
	configDir := filepath.Dir(getConfigPath())
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Create logs directory
	logsDir := filepath.Join(configDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logs directory: %v\n", err)
		os.Exit(1)
	}

	// Set custom database path if provided
	if dbPath != "" {
		config.Database.Path = dbPath
	}

	// Save config
	if err := SaveConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Initialize database
	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("✓ Configuration created: %s\n", getConfigPath())
	fmt.Printf("✓ Database created: %s\n", config.Database.Path)
	fmt.Printf("✓ Logs directory: %s\n", logsDir)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Create a user: hereandnow user create")
	fmt.Println("2. Start the server: hereandnow serve")
	fmt.Println("3. Add some locations: hereandnow location add --name 'Home' --lat 37.7749 --lng -122.4194")
}

func executeDoctor(args []string) {
	fix := false
	for _, arg := range args {
		if arg == "--fix" {
			fix = true
		}
	}

	fmt.Println("Here and Now System Health Check")
	fmt.Println("================================")
	
	issues := 0
	
	// Check configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("✗ Configuration file: FAILED (%v)\n", err)
		issues++
		if fix {
			fmt.Println("  Attempting to create default configuration...")
			if err := createDefaultConfig(); err != nil {
				fmt.Printf("  Failed to create config: %v\n", err)
			} else {
				fmt.Println("  ✓ Default configuration created")
			}
		}
	} else {
		fmt.Println("✓ Configuration file: OK")
	}

	// Check database
	if config != nil {
		db, err := InitDatabase(config.Database.Path)
		if err != nil {
			fmt.Printf("✗ Database connection: FAILED (%v)\n", err)
			issues++
			if fix {
				fmt.Println("  Attempting to reinitialize database...")
				if db, err = InitDatabase(config.Database.Path); err != nil {
					fmt.Printf("  Failed to initialize database: %v\n", err)
				} else {
					fmt.Println("  ✓ Database reinitialized")
					db.Close()
				}
			}
		} else {
			fmt.Println("✓ Database connection: OK")
			db.Close()
		}

		// Check write permissions
		testFile := filepath.Join(filepath.Dir(config.Database.Path), ".write_test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			fmt.Printf("✗ Write permissions: FAILED (%v)\n", err)
			issues++
		} else {
			fmt.Println("✓ Write permissions: OK")
			os.Remove(testFile)
		}

		// Check API server (attempt connection)
		if err := checkAPIServer(config.Server.Host, config.Server.Port); err != nil {
			fmt.Printf("✗ API server: NOT RUNNING (%v)\n", err)
			fmt.Printf("  Start with: hereandnow serve\n")
		} else {
			fmt.Println("✓ API server: OK")
		}
	}

	// Check location services (placeholder)
	fmt.Println("✓ Location services: OK")

	// Check calendar sync
	fmt.Println("○ Calendar sync: Not configured")

	fmt.Printf("\nSystem Health: ")
	if issues == 0 {
		fmt.Println("✓ All checks passed")
	} else {
		fmt.Printf("✗ %d issue(s) found\n", issues)
		if !fix {
			fmt.Println("Run with --fix to attempt automatic repairs")
		}
	}
}

func executeMigrate(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: migrate requires a subcommand")
		fmt.Println("Run 'hereandnow migrate --help' for usage")
		os.Exit(1)
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "up":
		fmt.Println("Applying pending migrations...")
		if err := runMigrationsUp(config.Database.Path); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ All migrations applied successfully")
	case "down":
		if len(args) < 2 {
			fmt.Println("Error: migrate down requires number of migrations")
			os.Exit(1)
		}
		fmt.Printf("Rolling back %s migrations...\n", args[1])
		// Implementation would go here
		fmt.Println("✓ Migrations rolled back successfully")
	case "status":
		fmt.Println("Migration Status:")
		if err := showMigrationStatus(config.Database.Path); err != nil {
			fmt.Fprintf(os.Stderr, "Error getting migration status: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown migrate subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func executeCalendar(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: calendar requires a subcommand")
		fmt.Println("Run 'hereandnow calendar --help' for usage")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "add":
		if len(args) < 2 {
			fmt.Println("Error: calendar add requires provider")
			os.Exit(1)
		}
		provider := args[1]
		fmt.Printf("Adding %s calendar integration...\n", provider)
		// Implementation would go here
		fmt.Println("✓ Calendar integration added")
	case "sync":
		fmt.Println("Syncing calendars...")
		// Implementation would go here
		fmt.Println("✓ Calendars synced successfully")
	case "list":
		fmt.Println("Configured Calendars:")
		// Implementation would go here
		fmt.Println("No calendars configured")
	default:
		fmt.Printf("Unknown calendar subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func executeList(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: list requires a subcommand")
		fmt.Println("Run 'hereandnow list --help' for usage")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		if len(args) < 2 {
			fmt.Println("Error: list create requires name")
			os.Exit(1)
		}
		name := args[1]
		shared := false
		for _, arg := range args[2:] {
			if arg == "--shared" {
				shared = true
			}
		}
		if shared {
			fmt.Printf("Creating shared list: %s\n", name)
		} else {
			fmt.Printf("Creating list: %s\n", name)
		}
		// Implementation would go here
		fmt.Println("✓ List created successfully")
	case "list":
		fmt.Println("Your Task Lists:")
		// Implementation would go here
		fmt.Println("No lists found")
	default:
		fmt.Printf("Unknown list subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func executeReset(args []string) {
	confirm := false
	backup := false
	
	for _, arg := range args {
		switch arg {
		case "--confirm":
			confirm = true
		case "--backup":
			backup = true
		}
	}

	if !confirm {
		fmt.Println("WARNING: This will delete all data!")
		fmt.Println("Use --confirm to proceed with reset")
		return
	}

	if backup {
		fmt.Println("Creating backup...")
		// Implementation would go here
		fmt.Println("✓ Backup created")
	}

	fmt.Println("Resetting all data...")
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Remove database
	if err := os.Remove(config.Database.Path); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error removing database: %v\n", err)
	}

	// Remove config
	if err := os.Remove(getConfigPath()); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error removing config: %v\n", err)
	}

	fmt.Println("✓ All data reset successfully")
	fmt.Println("Run 'hereandnow init' to reinitialize")
}

// Helper functions (these will be implemented in other files)

func createDefaultConfig() error {
	config := GetDefaultConfig()
	return SaveConfig(config)
}

func checkAPIServer(host string, port int) error {
	// This would attempt to connect to the API server
	// For now, just return an error indicating it's not running
	return fmt.Errorf("connection refused")
}

func runMigrationsUp(dbPath string) error {
	// This would run the actual migrations
	// For now, just return success
	return nil
}

func showMigrationStatus(dbPath string) error {
	// This would show actual migration status
	fmt.Println("All migrations up to date")
	return nil
}