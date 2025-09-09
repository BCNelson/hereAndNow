# Quick Start Guide: Here and Now Task Management

**Version**: 0.1.0  
**Date**: 2025-09-08

## Prerequisites

- Go 1.21 or higher installed
- Git for cloning the repository
- SQLite3 (usually pre-installed on most systems)
- 50MB free disk space
- 512MB RAM minimum (1GB recommended)

## Installation

### Option 1: Install from Binary (Recommended)

```bash
# Download the latest release for your platform
curl -L https://github.com/bcnelson/hereAndNow/releases/latest/download/hereandnow-$(uname -s)-$(uname -m).tar.gz -o hereandnow.tar.gz

# Extract the binary
tar -xzf hereandnow.tar.gz

# Move to PATH
sudo mv hereandnow /usr/local/bin/

# Verify installation
hereandnow --version
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/bcnelson/hereAndNow.git
cd hereAndNow

# Build the binary
make build

# Install to PATH
sudo make install

# Verify installation
hereandnow --version
```

## First Run Setup

### 1. Initialize the Database

```bash
# Create initial configuration and database
hereandnow init

# This creates:
# - ~/.hereandnow/config.yaml (configuration file)
# - ~/.hereandnow/data.db (SQLite database)
# - ~/.hereandnow/logs/ (log directory)
```

### 2. Create Your First User

```bash
# Create an admin user
hereandnow user create --admin
# Enter username: admin
# Enter email: admin@example.com
# Enter password: ********

# Create a regular user
hereandnow user create
# Enter username: john
# Enter email: john@example.com
# Enter password: ********
```

### 3. Start the Server

```bash
# Start the API server (default port 8080)
hereandnow serve

# Or specify a custom port
hereandnow serve --port 3000

# Run in background
hereandnow serve --daemon
```

## Basic Usage

### CLI Quick Commands

```bash
# Add a task quickly
hereandnow task add "Buy milk when at grocery store"

# List current tasks (filtered by context)
hereandnow task list

# List ALL tasks (override filtering)
hereandnow task list --all

# Complete a task
hereandnow task complete <task-id>

# Add a location
hereandnow location add --name "Home" --lat 37.7749 --lng -122.4194 --radius 100

# Update your current context
hereandnow context update --lat 37.7749 --lng -122.4194
```

### Web Interface

Open your browser to `http://localhost:8080` after starting the server.

Default login:
- Username: `admin`
- Password: (the password you set during setup)

## Testing Your Installation

### Verification Script

Run the built-in verification to ensure everything works:

```bash
# Run system check
hereandnow doctor

# Expected output:
# ✓ Database connection: OK
# ✓ Configuration file: OK
# ✓ Write permissions: OK
# ✓ API server: OK (port 8080)
# ✓ Location services: OK
# ✓ Calendar sync: Not configured
```

### Manual Test Workflow

1. **Create a test task with location requirement:**
```bash
# First, add a location
hereandnow location add --name "Office" --lat 37.7858 --lng -122.4065 --radius 200

# Create a task for that location
hereandnow task add "Review quarterly reports" --location "Office" --estimate 60
```

2. **Simulate being at different locations:**
```bash
# Update context to home location
hereandnow context update --lat 37.7749 --lng -122.4194
hereandnow task list
# Should NOT show "Review quarterly reports"

# Update context to office location
hereandnow context update --lat 37.7858 --lng -122.4065
hereandnow task list
# Should show "Review quarterly reports"
```

3. **Test time-based filtering:**
```bash
# Add a task with time estimate
hereandnow task add "Quick email check" --estimate 5

# Simulate having only 10 minutes available
hereandnow context update --available-minutes 10
hereandnow task list
# Should show "Quick email check"

# Simulate having only 3 minutes
hereandnow context update --available-minutes 3
hereandnow task list
# Should NOT show "Quick email check"
```

4. **Test task dependencies:**
```bash
# Create dependent tasks
hereandnow task add "Write report draft" --id draft-task
hereandnow task add "Review report" --depends-on draft-task
hereandnow task list
# Should show only "Write report draft"

# Complete the first task
hereandnow task complete draft-task
hereandnow task list
# Should now show "Review report"
```

5. **Test shared lists:**
```bash
# Create a shared list
hereandnow list create "Family Chores" --shared

# Add tasks to the list
hereandnow task add "Take out trash" --list "Family Chores"
hereandnow task add "Walk the dog" --list "Family Chores"

# Share with another user
hereandnow list share "Family Chores" --user john --role editor
```

## Configuration

### Basic Configuration

Edit `~/.hereandnow/config.yaml`:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  
database:
  path: ~/.hereandnow/data.db
  
logging:
  level: info
  path: ~/.hereandnow/logs
  
features:
  natural_language: true
  calendar_sync: false
  weather_integration: false
```

### Calendar Integration (Optional)

```bash
# Setup Google Calendar sync
hereandnow calendar add google
# Follow OAuth flow...

# Setup CalDAV (Nextcloud, etc.)
hereandnow calendar add caldav \
  --url https://your-server.com/remote.php/dav \
  --username your-user \
  --password your-password

# Sync calendars
hereandnow calendar sync
```

## API Integration Example

### Using curl

```bash
# Get auth token
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"yourpassword"}' \
  | jq -r .token)

# Get filtered tasks
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/tasks

# Create a task
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test task","priority":3}'
```

### Using the Go Library

```go
package main

import (
    "fmt"
    "github.com/bcnelson/hereAndNow/pkg/hereandnow"
)

func main() {
    // Initialize client
    client, err := hereandnow.NewClient(hereandnow.Config{
        DatabasePath: "~/.hereandnow/data.db",
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // Get current context tasks
    tasks, err := client.GetFilteredTasks(hereandnow.Context{
        Latitude:  37.7749,
        Longitude: -122.4194,
        AvailableMinutes: 30,
    })
    
    for _, task := range tasks {
        fmt.Printf("- %s (Priority: %d)\n", task.Title, task.Priority)
    }
}
```

## Troubleshooting

### Common Issues

**Server won't start:**
```bash
# Check if port is in use
lsof -i :8080

# Check logs
tail -f ~/.hereandnow/logs/server.log
```

**Database locked:**
```bash
# Stop all hereandnow processes
pkill hereandnow

# Check database integrity
sqlite3 ~/.hereandnow/data.db "PRAGMA integrity_check;"
```

**Tasks not showing:**
```bash
# Check filtering audit
hereandnow task audit <task-id>

# View current context
hereandnow context show
```

### Reset Everything

```bash
# Backup your data first!
cp -r ~/.hereandnow ~/.hereandnow.backup

# Reset to fresh install
hereandnow reset --confirm
```

## Next Steps

1. **Configure locations** for your common places (home, office, gym, etc.)
2. **Import existing tasks** from other systems using the import command
3. **Setup calendar sync** to automatically block time for meetings
4. **Install mobile app** for location-based updates (when available)
5. **Explore natural language input** for quick task creation

## Support

- Documentation: https://docs.hereandnow.local
- Issues: https://github.com/bcnelson/hereAndNow/issues
- Community: https://community.hereandnow.local

---
*For advanced configuration and API documentation, see the full documentation.*