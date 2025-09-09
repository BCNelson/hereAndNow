package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/bcnelson/hereAndNow/internal/auth"
	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

func handleUserCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: user requires a subcommand")
		fmt.Println("Run 'hereandnow user --help' for usage")
		os.Exit(1)
	}

	if args[0] == "--help" || args[0] == "-h" {
		fmt.Printf(`User Management Commands

USAGE:
    hereandnow user <SUBCOMMAND> [OPTIONS]

SUBCOMMANDS:
    create              Create a new user
    list                List all users
    show <username>     Show user details
    update <username>   Update user information
    delete <username>   Delete a user
    password <username> Change user password

OPTIONS:
    --admin             Make user an admin (create only)
    --email <email>     Set user email
    --timezone <tz>     Set user timezone (default: UTC)
    --help, -h         Show this help

EXAMPLES:
    # Create an admin user interactively
    hereandnow user create --admin

    # Create a user with email
    hereandnow user create --email user@example.com

    # List all users
    hereandnow user list

    # Show user details
    hereandnow user show john

    # Change password
    hereandnow user password john

    # Update user timezone
    hereandnow user update john --timezone America/New_York
`)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "create":
		executeUserCreate(subArgs)
	case "list":
		executeUserList(subArgs)
	case "show":
		executeUserShow(subArgs)
	case "update":
		executeUserUpdate(subArgs)
	case "delete":
		executeUserDelete(subArgs)
	case "password":
		executeUserPassword(subArgs)
	default:
		fmt.Printf("Unknown user subcommand: %s\n", subcommand)
		fmt.Println("Run 'hereandnow user --help' for usage")
		os.Exit(1)
	}
}

func executeUserCreate(args []string) {
	admin := false
	email := ""
	timezone := "UTC"

	for i, arg := range args {
		switch arg {
		case "--admin":
			admin = true
		case "--email":
			if i+1 < len(args) {
				email = args[i+1]
			}
		case "--timezone":
			if i+1 < len(args) {
				timezone = args[i+1]
			}
		}
	}

	// Initialize database connection
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

	userRepo := storage.NewUserRepository(db)
	authService := auth.NewAuthService(userRepo)

	// Get user input
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	if username == "" {
		fmt.Fprintf(os.Stderr, "Error: Username cannot be empty\n")
		os.Exit(1)
	}

	if email == "" {
		fmt.Print("Email: ")
		email, _ = reader.ReadString('\n')
		email = strings.TrimSpace(email)
	}

	if email == "" {
		fmt.Fprintf(os.Stderr, "Error: Email cannot be empty\n")
		os.Exit(1)
	}

	// Get password securely
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		os.Exit(1)
	}
	password := string(passwordBytes)
	fmt.Println() // New line after password input

	if len(password) < 6 {
		fmt.Fprintf(os.Stderr, "Error: Password must be at least 6 characters\n")
		os.Exit(1)
	}

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password confirmation: %v\n", err)
		os.Exit(1)
	}
	confirm := string(confirmBytes)
	fmt.Println()

	if password != confirm {
		fmt.Fprintf(os.Stderr, "Error: Passwords do not match\n")
		os.Exit(1)
	}

	// Create user
	user, err := authService.CreateUser(username, email, password, admin, timezone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("User %s created successfully", user.Username))
}

func executeUserList(args []string) {
	// Initialize database connection
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

	userRepo := storage.NewUserRepository(db)

	users, err := userRepo.GetAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving users: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, users)
}

func executeUserShow(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: user show requires username\n")
		fmt.Println("Usage: hereandnow user show <username>")
		os.Exit(1)
	}

	username := args[0]

	// Initialize database connection
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

	userRepo := storage.NewUserRepository(db)

	user, err := userRepo.GetByUsername(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: User '%s' not found\n", username)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, *user)
}

func executeUserUpdate(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: user update requires username\n")
		fmt.Println("Usage: hereandnow user update <username> [OPTIONS]")
		os.Exit(1)
	}

	username := args[0]
	email := ""
	timezone := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--email":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		case "--timezone":
			if i+1 < len(args) {
				timezone = args[i+1]
				i++
			}
		}
	}

	if email == "" && timezone == "" {
		fmt.Fprintf(os.Stderr, "Error: At least one field must be updated\n")
		fmt.Println("Available options: --email, --timezone")
		os.Exit(1)
	}

	// Initialize database connection
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

	userRepo := storage.NewUserRepository(db)

	user, err := userRepo.GetByUsername(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: User '%s' not found\n", username)
		os.Exit(1)
	}

	// Update fields
	if email != "" {
		user.Email = email
	}
	if timezone != "" {
		user.Timezone = timezone
	}
	user.UpdatedAt = time.Now()

	if err := userRepo.Update(*user); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating user: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("User %s updated successfully", username))
}

func executeUserDelete(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: user delete requires username\n")
		fmt.Println("Usage: hereandnow user delete <username>")
		os.Exit(1)
	}

	username := args[0]

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete user '%s'? This action cannot be undone.\n", username)
	fmt.Print("Type 'yes' to confirm: ")
	
	reader := bufio.NewReader(os.Stdin)
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(strings.ToLower(confirmation))

	if confirmation != "yes" {
		fmt.Println("Deletion cancelled")
		return
	}

	// Initialize database connection
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

	userRepo := storage.NewUserRepository(db)

	user, err := userRepo.GetByUsername(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: User '%s' not found\n", username)
		os.Exit(1)
	}

	if err := userRepo.Delete(user.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting user: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("User %s deleted successfully", username))
}

func executeUserPassword(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: user password requires username\n")
		fmt.Println("Usage: hereandnow user password <username>")
		os.Exit(1)
	}

	username := args[0]

	// Initialize database connection
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

	userRepo := storage.NewUserRepository(db)

	user, err := userRepo.GetByUsername(username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: User '%s' not found\n", username)
		os.Exit(1)
	}

	// Get new password
	fmt.Print("New password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		os.Exit(1)
	}
	password := string(passwordBytes)
	fmt.Println()

	if len(password) < 6 {
		fmt.Fprintf(os.Stderr, "Error: Password must be at least 6 characters\n")
		os.Exit(1)
	}

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password confirmation: %v\n", err)
		os.Exit(1)
	}
	confirm := string(confirmBytes)
	fmt.Println()

	if password != confirm {
		fmt.Fprintf(os.Stderr, "Error: Passwords do not match\n")
		os.Exit(1)
	}

	// Hash new password
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		// Fallback to UUID for salt
		salt = []byte(uuid.New().String())
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	
	user.PasswordHash = string(hash)
	user.Salt = string(salt)
	user.UpdatedAt = time.Now()

	if err := userRepo.Update(*user); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating password: %v\n", err)
		os.Exit(1)
	}

	formatter := NewFormatter(globalConfig.Format)
	Output(formatter, fmt.Sprintf("Password updated successfully for user %s", username))
}