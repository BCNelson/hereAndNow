package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

type Formatter interface {
	FormatTasks(tasks []models.Task) string
	FormatTask(task models.Task) string
	FormatUsers(users []models.User) string
	FormatUser(user models.User) string
	FormatLocations(locations []models.Location) string
	FormatLocation(location models.Location) string
	FormatContext(context models.Context) string
	FormatAnalytics(analytics map[string]interface{}) string
	FormatError(err error) string
	FormatSuccess(message string) string
	FormatWarning(message string) string
	FormatInfo(message string) string
}

func NewFormatter(format string) Formatter {
	switch format {
	case "json":
		return &JSONFormatter{}
	case "table":
		return &TableFormatter{}
	case "human":
		return &HumanFormatter{}
	default:
		return &HumanFormatter{}
	}
}

// JSON Formatter
type JSONFormatter struct{}

func (f *JSONFormatter) FormatTasks(tasks []models.Task) string {
	data, _ := json.MarshalIndent(tasks, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatTask(task models.Task) string {
	data, _ := json.MarshalIndent(task, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatUsers(users []models.User) string {
	data, _ := json.MarshalIndent(users, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatUser(user models.User) string {
	data, _ := json.MarshalIndent(user, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatLocations(locations []models.Location) string {
	data, _ := json.MarshalIndent(locations, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatLocation(location models.Location) string {
	data, _ := json.MarshalIndent(location, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatContext(context models.Context) string {
	data, _ := json.MarshalIndent(context, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatAnalytics(analytics map[string]interface{}) string {
	data, _ := json.MarshalIndent(analytics, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatError(err error) string {
	result := map[string]interface{}{
		"error": err.Error(),
		"type":  "error",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatSuccess(message string) string {
	result := map[string]interface{}{
		"message": message,
		"type":    "success",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatWarning(message string) string {
	result := map[string]interface{}{
		"message": message,
		"type":    "warning",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}

func (f *JSONFormatter) FormatInfo(message string) string {
	result := map[string]interface{}{
		"message": message,
		"type":    "info",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}

// Table Formatter
type TableFormatter struct{}

func (f *TableFormatter) FormatTasks(tasks []models.Task) string {
	if len(tasks) == 0 {
		return "No tasks found.\n"
	}

	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "ID\tTitle\tStatus\tPriority\tEstimate\tDue\tLocation\n")
	fmt.Fprintf(w, "--\t-----\t------\t--------\t--------\t---\t--------\n")

	for _, task := range tasks {
		id := truncateString(task.ID, 8)
		title := truncateString(task.Title, 30)
		status := string(task.Status)
		priority := strconv.Itoa(task.Priority)
		estimate := "N/A"
		if task.EstimatedMinutes != nil {
			estimate = fmt.Sprintf("%dm", *task.EstimatedMinutes)
		}
		due := "N/A"
		if task.DueAt != nil {
			due = task.DueAt.Format("2006-01-02")
		}
		location := "Any"

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			id, title, status, priority, estimate, due, location)
	}

	w.Flush()
	return sb.String()
}

func (f *TableFormatter) FormatTask(task models.Task) string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "Field\tValue\n")
	fmt.Fprintf(w, "-----\t-----\n")
	fmt.Fprintf(w, "ID\t%s\n", task.ID)
	fmt.Fprintf(w, "Title\t%s\n", task.Title)
	fmt.Fprintf(w, "Description\t%s\n", task.Description)
	fmt.Fprintf(w, "Status\t%s\n", task.Status)
	fmt.Fprintf(w, "Priority\t%d\n", task.Priority)
	
	if task.EstimatedMinutes != nil {
		fmt.Fprintf(w, "Estimate\t%d minutes\n", *task.EstimatedMinutes)
	}
	
	if task.DueAt != nil {
		fmt.Fprintf(w, "Due\t%s\n", task.DueAt.Format("2006-01-02 15:04"))
	}
	
	fmt.Fprintf(w, "Created\t%s\n", task.CreatedAt.Format("2006-01-02 15:04"))

	w.Flush()
	return sb.String()
}

func (f *TableFormatter) FormatUsers(users []models.User) string {
	if len(users) == 0 {
		return "No users found.\n"
	}

	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "ID\tUsername\tEmail\tAdmin\tTimezone\tCreated\n")
	fmt.Fprintf(w, "--\t--------\t-----\t-----\t--------\t-------\n")

	for _, user := range users {
		id := truncateString(user.ID, 8)
		admin := "No"
		if user.IsAdmin {
			admin = "Yes"
		}
		created := user.CreatedAt.Format("2006-01-02")

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			id, user.Username, user.Email, admin, user.Timezone, created)
	}

	w.Flush()
	return sb.String()
}

func (f *TableFormatter) FormatUser(user models.User) string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "Field\tValue\n")
	fmt.Fprintf(w, "-----\t-----\n")
	fmt.Fprintf(w, "ID\t%s\n", user.ID)
	fmt.Fprintf(w, "Username\t%s\n", user.Username)
	fmt.Fprintf(w, "Email\t%s\n", user.Email)
	fmt.Fprintf(w, "Admin\t%t\n", user.IsAdmin)
	fmt.Fprintf(w, "Timezone\t%s\n", user.Timezone)
	fmt.Fprintf(w, "Created\t%s\n", user.CreatedAt.Format("2006-01-02 15:04"))

	w.Flush()
	return sb.String()
}

func (f *TableFormatter) FormatLocations(locations []models.Location) string {
	if len(locations) == 0 {
		return "No locations found.\n"
	}

	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "ID\tName\tLatitude\tLongitude\tRadius\tCreated\n")
	fmt.Fprintf(w, "--\t----\t--------\t---------\t------\t-------\n")

	for _, location := range locations {
		id := truncateString(location.ID, 8)
		name := truncateString(location.Name, 20)
		created := location.CreatedAt.Format("2006-01-02")

		fmt.Fprintf(w, "%s\t%s\t%.6f\t%.6f\t%dm\t%s\n",
			id, name, location.Latitude, location.Longitude, location.Radius, created)
	}

	w.Flush()
	return sb.String()
}

func (f *TableFormatter) FormatLocation(location models.Location) string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "Field\tValue\n")
	fmt.Fprintf(w, "-----\t-----\n")
	fmt.Fprintf(w, "ID\t%s\n", location.ID)
	fmt.Fprintf(w, "Name\t%s\n", location.Name)
	fmt.Fprintf(w, "Latitude\t%.6f\n", location.Latitude)
	fmt.Fprintf(w, "Longitude\t%.6f\n", location.Longitude)
	fmt.Fprintf(w, "Radius\t%d meters\n", location.Radius)
	fmt.Fprintf(w, "Created\t%s\n", location.CreatedAt.Format("2006-01-02 15:04"))

	w.Flush()
	return sb.String()
}

func (f *TableFormatter) FormatContext(context models.Context) string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "Field\tValue\n")
	fmt.Fprintf(w, "-----\t-----\n")
	fmt.Fprintf(w, "Timestamp\t%s\n", context.Timestamp.Format("2006-01-02 15:04:05"))
	
	if context.CurrentLatitude != nil && context.CurrentLongitude != nil {
		fmt.Fprintf(w, "Location\t%.6f, %.6f\n", *context.CurrentLatitude, *context.CurrentLongitude)
	}
	
	fmt.Fprintf(w, "Available Minutes\t%d\n", context.AvailableMinutes)
	fmt.Fprintf(w, "Social Context\t%s\n", context.SocialContext)
	fmt.Fprintf(w, "Energy Level\t%d/5\n", context.EnergyLevel)
	
	if context.WeatherCondition != nil {
		fmt.Fprintf(w, "Weather\t%s\n", *context.WeatherCondition)
	}
	
	if context.TrafficLevel != nil {
		fmt.Fprintf(w, "Traffic\t%s\n", *context.TrafficLevel)
	}

	w.Flush()
	return sb.String()
}

func (f *TableFormatter) FormatAnalytics(analytics map[string]interface{}) string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "Metric\tValue\n")
	fmt.Fprintf(w, "------\t-----\n")

	// Sort keys for consistent output
	keys := make([]string, 0, len(analytics))
	for k := range analytics {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Fprintf(w, "%s\t%v\n", key, analytics[key])
	}

	w.Flush()
	return sb.String()
}

func (f *TableFormatter) FormatError(err error) string {
	return fmt.Sprintf("ERROR: %s\n", err.Error())
}

func (f *TableFormatter) FormatSuccess(message string) string {
	return fmt.Sprintf("SUCCESS: %s\n", message)
}

func (f *TableFormatter) FormatWarning(message string) string {
	return fmt.Sprintf("WARNING: %s\n", message)
}

func (f *TableFormatter) FormatInfo(message string) string {
	return fmt.Sprintf("INFO: %s\n", message)
}

// Human-Readable Formatter
type HumanFormatter struct{}

func (f *HumanFormatter) FormatTasks(tasks []models.Task) string {
	if len(tasks) == 0 {
		return f.colorize(ColorDim, "No tasks found.\n")
	}

	var sb strings.Builder
	sb.WriteString(f.colorize(ColorBold, fmt.Sprintf("Found %d task(s):\n\n", len(tasks))))

	for i, task := range tasks {
		sb.WriteString(f.formatTaskSummary(task, i+1))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (f *HumanFormatter) FormatTask(task models.Task) string {
	var sb strings.Builder

	// Title and ID
	sb.WriteString(f.colorize(ColorBold, fmt.Sprintf("Task: %s\n", task.Title)))
	sb.WriteString(f.colorize(ColorDim, fmt.Sprintf("ID: %s\n", task.ID)))

	// Description
	if task.Description != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", task.Description))
	}

	// Status and Priority
	statusColor := ColorYellow
	switch task.Status {
	case models.TaskStatusCompleted:
		statusColor = ColorGreen
	case models.TaskStatusInProgress:
		statusColor = ColorBlue
	case models.TaskStatusBlocked:
		statusColor = ColorRed
	}
	
	sb.WriteString(fmt.Sprintf("\nStatus: %s\n", f.colorize(statusColor, string(task.Status))))
	sb.WriteString(fmt.Sprintf("Priority: %s\n", f.priorityIndicator(task.Priority)))

	// Time information
	if task.EstimatedMinutes != nil {
		sb.WriteString(fmt.Sprintf("Estimated time: %d minutes\n", *task.EstimatedMinutes))
	}
	
	if task.DueAt != nil {
		dueStr := task.DueAt.Format("Monday, January 2, 2006 at 3:04 PM")
		if task.DueAt.Before(time.Now()) {
			dueStr = f.colorize(ColorRed, dueStr+" (OVERDUE)")
		}
		sb.WriteString(fmt.Sprintf("Due: %s\n", dueStr))
	}

	if task.CompletedAt != nil {
		sb.WriteString(fmt.Sprintf("Completed: %s\n", task.CompletedAt.Format("Monday, January 2, 2006 at 3:04 PM")))
	}

	sb.WriteString(fmt.Sprintf("\nCreated: %s\n", task.CreatedAt.Format("Monday, January 2, 2006 at 3:04 PM")))
	sb.WriteString(fmt.Sprintf("Updated: %s\n", task.UpdatedAt.Format("Monday, January 2, 2006 at 3:04 PM")))

	return sb.String()
}

func (f *HumanFormatter) FormatUsers(users []models.User) string {
	if len(users) == 0 {
		return f.colorize(ColorDim, "No users found.\n")
	}

	var sb strings.Builder
	sb.WriteString(f.colorize(ColorBold, fmt.Sprintf("Found %d user(s):\n\n", len(users))))

	for i, user := range users {
		sb.WriteString(fmt.Sprintf("%d. %s", i+1, f.colorize(ColorBold, user.Username)))
		if user.IsAdmin {
			sb.WriteString(f.colorize(ColorYellow, " (Admin)"))
		}
		sb.WriteString(fmt.Sprintf("\n   Email: %s\n", user.Email))
		sb.WriteString(fmt.Sprintf("   Timezone: %s\n", user.Timezone))
		sb.WriteString(fmt.Sprintf("   Created: %s\n\n", user.CreatedAt.Format("2006-01-02")))
	}

	return sb.String()
}

func (f *HumanFormatter) FormatUser(user models.User) string {
	var sb strings.Builder

	sb.WriteString(f.colorize(ColorBold, fmt.Sprintf("User: %s", user.Username)))
	if user.IsAdmin {
		sb.WriteString(f.colorize(ColorYellow, " (Administrator)"))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("Email: %s\n", user.Email))
	sb.WriteString(fmt.Sprintf("Timezone: %s\n", user.Timezone))
	sb.WriteString(fmt.Sprintf("Created: %s\n", user.CreatedAt.Format("Monday, January 2, 2006")))

	return sb.String()
}

func (f *HumanFormatter) FormatLocations(locations []models.Location) string {
	if len(locations) == 0 {
		return f.colorize(ColorDim, "No locations found.\n")
	}

	var sb strings.Builder
	sb.WriteString(f.colorize(ColorBold, fmt.Sprintf("Found %d location(s):\n\n", len(locations))))

	for i, location := range locations {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, f.colorize(ColorBold, location.Name)))
		sb.WriteString(fmt.Sprintf("   Coordinates: %.6f, %.6f\n", location.Latitude, location.Longitude))
		sb.WriteString(fmt.Sprintf("   Radius: %d meters\n", location.Radius))
		sb.WriteString(fmt.Sprintf("   Created: %s\n\n", location.CreatedAt.Format("2006-01-02")))
	}

	return sb.String()
}

func (f *HumanFormatter) FormatLocation(location models.Location) string {
	var sb strings.Builder

	sb.WriteString(f.colorize(ColorBold, fmt.Sprintf("Location: %s\n", location.Name)))
	sb.WriteString(fmt.Sprintf("Coordinates: %.6f, %.6f\n", location.Latitude, location.Longitude))
	sb.WriteString(fmt.Sprintf("Radius: %d meters\n", location.Radius))
	sb.WriteString(fmt.Sprintf("Created: %s\n", location.CreatedAt.Format("Monday, January 2, 2006")))

	return sb.String()
}

func (f *HumanFormatter) FormatContext(context models.Context) string {
	var sb strings.Builder

	sb.WriteString(f.colorize(ColorBold, "Current Context\n"))
	sb.WriteString(fmt.Sprintf("Updated: %s\n\n", context.Timestamp.Format("Monday, January 2, 2006 at 3:04 PM")))

	if context.CurrentLatitude != nil && context.CurrentLongitude != nil {
		sb.WriteString(fmt.Sprintf("📍 Location: %.6f, %.6f\n", *context.CurrentLatitude, *context.CurrentLongitude))
	} else {
		sb.WriteString("📍 Location: Unknown\n")
	}

	sb.WriteString(fmt.Sprintf("⏱️  Available time: %d minutes\n", context.AvailableMinutes))
	sb.WriteString(fmt.Sprintf("👥 Social context: %s\n", context.SocialContext))
	sb.WriteString(fmt.Sprintf("⚡ Energy level: %s\n", f.energyIndicator(context.EnergyLevel)))

	if context.WeatherCondition != nil {
		sb.WriteString(fmt.Sprintf("🌤️  Weather: %s\n", *context.WeatherCondition))
	}

	if context.TrafficLevel != nil {
		sb.WriteString(fmt.Sprintf("🚗 Traffic: %s\n", *context.TrafficLevel))
	}

	return sb.String()
}

func (f *HumanFormatter) FormatAnalytics(analytics map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString(f.colorize(ColorBold, "Analytics Summary\n\n"))

	// Sort keys for consistent output
	keys := make([]string, 0, len(analytics))
	for k := range analytics {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		sb.WriteString(fmt.Sprintf("📊 %s: %v\n", strings.Title(strings.ReplaceAll(key, "_", " ")), analytics[key]))
	}

	return sb.String()
}

func (f *HumanFormatter) FormatError(err error) string {
	return f.colorize(ColorRed, fmt.Sprintf("❌ Error: %s\n", err.Error()))
}

func (f *HumanFormatter) FormatSuccess(message string) string {
	return f.colorize(ColorGreen, fmt.Sprintf("✅ %s\n", message))
}

func (f *HumanFormatter) FormatWarning(message string) string {
	return f.colorize(ColorYellow, fmt.Sprintf("⚠️  %s\n", message))
}

func (f *HumanFormatter) FormatInfo(message string) string {
	return f.colorize(ColorBlue, fmt.Sprintf("ℹ️  %s\n", message))
}

// Helper methods for HumanFormatter

func (f *HumanFormatter) colorize(color, text string) string {
	if globalConfig.NoColor {
		return text
	}
	return color + text + ColorReset
}

func (f *HumanFormatter) formatTaskSummary(task models.Task, index int) string {
	var sb strings.Builder

	// Task number and title
	sb.WriteString(fmt.Sprintf("%d. %s", index, f.colorize(ColorBold, task.Title)))

	// Status indicator
	switch task.Status {
	case models.TaskStatusCompleted:
		sb.WriteString(f.colorize(ColorGreen, " ✅"))
	case models.TaskStatusInProgress:
		sb.WriteString(f.colorize(ColorBlue, " 🔄"))
	case models.TaskStatusBlocked:
		sb.WriteString(f.colorize(ColorRed, " 🚫"))
	default:
		sb.WriteString(f.colorize(ColorYellow, " ⏳"))
	}

	// Priority
	sb.WriteString(fmt.Sprintf(" %s", f.priorityIndicator(task.Priority)))

	// Time estimate
	if task.EstimatedMinutes != nil {
		sb.WriteString(f.colorize(ColorCyan, fmt.Sprintf(" (%dm)", *task.EstimatedMinutes)))
	}

	// Due date
	if task.DueAt != nil {
		if task.DueAt.Before(time.Now()) {
			sb.WriteString(f.colorize(ColorRed, " (OVERDUE)"))
		} else {
			sb.WriteString(f.colorize(ColorDim, fmt.Sprintf(" (due %s)", task.DueAt.Format("Jan 2"))))
		}
	}

	// Description preview
	if task.Description != "" {
		desc := truncateString(task.Description, 60)
		sb.WriteString(f.colorize(ColorDim, fmt.Sprintf("\n   %s", desc)))
	}

	return sb.String()
}

func (f *HumanFormatter) priorityIndicator(priority int) string {
	switch {
	case priority >= 8:
		return f.colorize(ColorRed, "🔥 Critical")
	case priority >= 6:
		return f.colorize(ColorYellow, "⚡ High")
	case priority >= 4:
		return f.colorize(ColorBlue, "📋 Medium")
	default:
		return f.colorize(ColorDim, "📝 Low")
	}
}

func (f *HumanFormatter) energyIndicator(energy int) string {
	switch energy {
	case 5:
		return f.colorize(ColorGreen, "🟢🟢🟢🟢🟢 Maximum")
	case 4:
		return f.colorize(ColorGreen, "🟢🟢🟢🟢⚪ High")
	case 3:
		return f.colorize(ColorYellow, "🟢🟢🟢⚪⚪ Medium")
	case 2:
		return f.colorize(ColorYellow, "🟢🟢⚪⚪⚪ Low")
	case 1:
		return f.colorize(ColorRed, "🟢⚪⚪⚪⚪ Very Low")
	default:
		return f.colorize(ColorRed, "⚪⚪⚪⚪⚪ Exhausted")
	}
}

// Utility functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func Output(formatter Formatter, data interface{}) {
	var output string

	switch v := data.(type) {
	case []models.Task:
		output = formatter.FormatTasks(v)
	case models.Task:
		output = formatter.FormatTask(v)
	case []models.User:
		output = formatter.FormatUsers(v)
	case models.User:
		output = formatter.FormatUser(v)
	case []models.Location:
		output = formatter.FormatLocations(v)
	case models.Location:
		output = formatter.FormatLocation(v)
	case models.Context:
		output = formatter.FormatContext(v)
	case map[string]interface{}:
		output = formatter.FormatAnalytics(v)
	case error:
		output = formatter.FormatError(v)
		fmt.Fprint(os.Stderr, output)
		return
	case string:
		// Determine message type based on content or use info as default
		if strings.Contains(strings.ToLower(v), "error") {
			output = formatter.FormatError(fmt.Errorf(v))
		} else if strings.Contains(strings.ToLower(v), "success") {
			output = formatter.FormatSuccess(v)
		} else if strings.Contains(strings.ToLower(v), "warning") {
			output = formatter.FormatWarning(v)
		} else {
			output = formatter.FormatInfo(v)
		}
	default:
		// Fallback to JSON for unknown types
		if data, err := json.MarshalIndent(v, "", "  "); err == nil {
			output = string(data) + "\n"
		} else {
			output = formatter.FormatError(fmt.Errorf("unable to format data: %v", v))
		}
	}

	fmt.Print(output)
}