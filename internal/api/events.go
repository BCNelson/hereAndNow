package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bcnelson/hereAndNow/pkg/models"
	"github.com/gin-gonic/gin"
)

type EventsHandler struct {
	eventService EventService
}

type EventService interface {
	Subscribe(userID string) (<-chan Event, func(), error)
	PublishTaskEvent(event TaskEvent) error
	PublishContextEvent(event ContextEvent) error
	PublishSystemEvent(event SystemEvent) error
	GetActiveSubscribers() int
}

type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	UserID    string      `json:"user_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type TaskEvent struct {
	TaskID    string      `json:"task_id"`
	Action    string      `json:"action"` // created, updated, completed, deleted, assigned
	Task      models.Task `json:"task"`
	Changes   []string    `json:"changes,omitempty"`
	UserID    string      `json:"user_id"`
}

type ContextEvent struct {
	ContextID string         `json:"context_id"`
	Action    string         `json:"action"` // updated, location_changed
	Context   models.Context `json:"context"`
	UserID    string         `json:"user_id"`
}

type SystemEvent struct {
	Message  string                 `json:"message"`
	Level    string                 `json:"level"` // info, warning, error
	Category string                 `json:"category"`
	Data     map[string]interface{} `json:"data,omitempty"`
	UserID   string                 `json:"user_id"`
}

func NewEventsHandler(eventService EventService) *EventsHandler {
	return &EventsHandler{
		eventService: eventService,
	}
}

// GetEvents handles GET /events (SSE) - Server-Sent Events for real-time updates
func (h *EventsHandler) GetEvents(c *gin.Context) {
	userID, err := GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authentication required",
		})
		return
	}

	// Parse query parameters
	keepAlive := 30 // Default keep-alive interval in seconds
	if keepAliveStr := c.Query("keep_alive"); keepAliveStr != "" {
		if ka, err := strconv.Atoi(keepAliveStr); err == nil && ka > 0 && ka <= 300 {
			keepAlive = ka
		}
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Subscribe to events for this user
	eventChan, unsubscribe, err := h.eventService.Subscribe(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to subscribe to events",
		})
		return
	}

	// Ensure cleanup on connection close
	defer unsubscribe()

	// Send initial connection event
	h.sendSSEEvent(c, "connected", map[string]interface{}{
		"user_id":    userID,
		"timestamp":  time.Now(),
		"keep_alive": keepAlive,
		"message":    "Successfully connected to event stream",
	})

	// Context for handling client disconnection
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Keep-alive ticker
	keepAliveTicker := time.NewTicker(time.Duration(keepAlive) * time.Second)
	defer keepAliveTicker.Stop()

	// Event loop
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return

		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, send close event and exit
				h.sendSSEEvent(c, "disconnected", map[string]interface{}{
					"message": "Event stream closed",
					"reason":  "service_shutdown",
				})
				return
			}

			// Send the event to client
			h.sendSSEEvent(c, event.Type, event)

		case <-keepAliveTicker.C:
			// Send keep-alive ping
			h.sendSSEEvent(c, "ping", map[string]interface{}{
				"timestamp": time.Now(),
				"active_subscribers": h.eventService.GetActiveSubscribers(),
			})
		}

		// Flush the response to ensure real-time delivery
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}
	}
}

// sendSSEEvent formats and sends a Server-Sent Event
func (h *EventsHandler) sendSSEEvent(c *gin.Context, eventType string, data interface{}) {
	// Generate event ID
	eventID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Marshal data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		// Send error event if JSON marshaling fails
		c.Writer.Write([]byte(fmt.Sprintf("event: error\n")))
		c.Writer.Write([]byte(fmt.Sprintf("id: %s\n", eventID)))
		c.Writer.Write([]byte(fmt.Sprintf("data: {\"error\":\"Failed to marshal event data\"}\n\n")))
		return
	}

	// Write SSE formatted event
	c.Writer.Write([]byte(fmt.Sprintf("event: %s\n", eventType)))
	c.Writer.Write([]byte(fmt.Sprintf("id: %s\n", eventID)))
	c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", jsonData)))
}