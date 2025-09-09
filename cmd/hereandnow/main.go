package main

import (
	"fmt"
	"os"
	"strings"
)

const Version = "0.1.0"

type GlobalConfig struct {
	Format     string // json, table, human
	ConfigPath string
	Verbose    bool
	NoColor    bool
}

var globalConfig GlobalConfig

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	// Parse global flags first
	args, err := parseGlobalFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(args) == 0 {
		showHelp()
		return
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "help", "--help", "-h":
		showHelp()
	case "version", "--version", "-v":
		showVersion()
	case "init":
		handleInit(commandArgs)
	case "user":
		handleUserCommand(commandArgs)
	case "task":
		handleTaskCommand(commandArgs)
	case "location":
		handleLocationCommand(commandArgs)
	case "context":
		handleContextCommand(commandArgs)
	case "serve":
		handleServeCommand(commandArgs)
	case "migrate":
		handleMigrateCommand(commandArgs)
	case "doctor":
		handleDoctorCommand(commandArgs)
	case "calendar":
		handleCalendarCommand(commandArgs)
	case "list":
		handleListCommand(commandArgs)
	case "reset":
		handleResetCommand(commandArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintf(os.Stderr, "Run 'hereandnow help' for usage information.\n")
		os.Exit(1)
	}
}

func parseGlobalFlags(args []string) ([]string, error) {
	remainingArgs := []string{}
	globalConfig.Format = "human" // default

	for i := 0; i < len(args); i++ {
		arg := args[i]
		
		if arg == "--format" && i+1 < len(args) {
			format := args[i+1]
			if format != "json" && format != "table" && format != "human" {
				return nil, fmt.Errorf("invalid format: %s (must be json, table, or human)", format)
			}
			globalConfig.Format = format
			i++ // skip the next argument as it's the format value
		} else if strings.HasPrefix(arg, "--format=") {
			format := strings.TrimPrefix(arg, "--format=")
			if format != "json" && format != "table" && format != "human" {
				return nil, fmt.Errorf("invalid format: %s (must be json, table, or human)", format)
			}
			globalConfig.Format = format
		} else if arg == "--config" && i+1 < len(args) {
			globalConfig.ConfigPath = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--config=") {
			globalConfig.ConfigPath = strings.TrimPrefix(arg, "--config=")
		} else if arg == "--verbose" || arg == "-v" {
			globalConfig.Verbose = true
		} else if arg == "--no-color" {
			globalConfig.NoColor = true
		} else if strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("unknown global flag: %s", arg)
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}

	return remainingArgs, nil
}

func showHelp() {
	fmt.Printf(`Here and Now - Context-Aware Task Management

USAGE:
    hereandnow [GLOBAL OPTIONS] <COMMAND> [OPTIONS]

VERSION:
    %s

GLOBAL OPTIONS:
    --format <format>    Output format: json, table, human (default: human)
    --config <path>      Config file path (default: ~/.hereandnow/config.yaml)
    --verbose, -v        Enable verbose output
    --no-color          Disable colored output
    --help, -h          Show help
    --version           Show version

COMMANDS:
    init                 Initialize database and configuration
    serve                Start the API server
    migrate              Run database migrations
    doctor               Check system health and configuration

    user                 User management commands
    task                 Task management commands
    location             Location management commands  
    context              Context management commands
    list                 Task list management commands
    calendar             Calendar integration commands

    reset                Reset all data (destructive)

EXAMPLES:
    # Initialize the system
    hereandnow init

    # Start the server
    hereandnow serve

    # Add a task with location
    hereandnow task add "Buy milk" --location "Grocery Store"

    # List current tasks (context filtered)
    hereandnow task list

    # Update your location
    hereandnow context update --lat 37.7749 --lng -122.4194

    # Add a location
    hereandnow location add --name "Home" --lat 37.7749 --lng -122.4194 --radius 100

    # Check system status
    hereandnow doctor

Use 'hereandnow <command> --help' for more information about a specific command.
`, Version)
}

func showVersion() {
	fmt.Printf("hereandnow version %s\n", Version)
}

func handleInit(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Printf(`Initialize Here and Now

USAGE:
    hereandnow init [OPTIONS]

DESCRIPTION:
    Creates the initial configuration file and database.
    This should be run once after installation.

OPTIONS:
    --force              Force initialization even if config exists
    --db-path <path>     Custom database path
    --help, -h          Show this help

EXAMPLES:
    hereandnow init
    hereandnow init --force
    hereandnow init --db-path ./custom.db
`)
		return
	}

	executeInit(args)
}

func handleDoctorCommand(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Printf(`System Health Check

USAGE:
    hereandnow doctor [OPTIONS]

DESCRIPTION:
    Checks system health, database connectivity, and configuration.
    Provides detailed diagnostics for troubleshooting.

OPTIONS:
    --fix               Attempt to fix common issues
    --help, -h         Show this help

EXAMPLES:
    hereandnow doctor
    hereandnow doctor --fix
`)
		return
	}

	executeDoctor(args)
}

func handleMigrateCommand(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Printf(`Database Migration Management

USAGE:
    hereandnow migrate <SUBCOMMAND> [OPTIONS]

SUBCOMMANDS:
    up                  Apply pending migrations
    down <n>           Rollback n migrations
    status             Show migration status
    force <version>    Force database to specific version

OPTIONS:
    --help, -h         Show this help

EXAMPLES:
    hereandnow migrate up
    hereandnow migrate down 1
    hereandnow migrate status
`)
		return
	}

	executeMigrate(args)
}

func handleCalendarCommand(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Printf(`Calendar Integration Commands

USAGE:
    hereandnow calendar <SUBCOMMAND> [OPTIONS]

SUBCOMMANDS:
    add <provider>     Add calendar integration (google, caldav)
    sync              Sync all calendars
    list              List configured calendars
    remove <name>     Remove calendar integration

OPTIONS:
    --help, -h         Show this help

EXAMPLES:
    hereandnow calendar add google
    hereandnow calendar add caldav --url https://server.com/dav
    hereandnow calendar sync
    hereandnow calendar list
`)
		return
	}

	executeCalendar(args)
}

func handleListCommand(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Printf(`Task List Management Commands

USAGE:
    hereandnow list <SUBCOMMAND> [OPTIONS]

SUBCOMMANDS:
    create <name>      Create a new task list
    list              Show all task lists
    share <name>      Share a task list with users
    members <name>    Show list members
    delete <name>     Delete a task list

OPTIONS:
    --shared           Create as shared list
    --help, -h         Show this help

EXAMPLES:
    hereandnow list create "Family Chores"
    hereandnow list create "Work Projects" --shared
    hereandnow list share "Family Chores" --user john --role editor
    hereandnow list list
`)
		return
	}

	executeList(args)
}

func handleResetCommand(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Printf(`Reset All Data (Destructive)

USAGE:
    hereandnow reset [OPTIONS]

DESCRIPTION:
    WARNING: This will delete all data including tasks, users, and configuration.
    This action cannot be undone.

OPTIONS:
    --confirm          Confirm the reset operation
    --backup           Create backup before reset
    --help, -h         Show this help

EXAMPLES:
    hereandnow reset --confirm
    hereandnow reset --backup --confirm
`)
		return
	}

	executeReset(args)
}