package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bcnelson/hereAndNow/internal/api"
	"github.com/bcnelson/hereAndNow/internal/auth"
	"github.com/bcnelson/hereAndNow/internal/storage"
	"github.com/bcnelson/hereAndNow/pkg/filters"
	"github.com/bcnelson/hereAndNow/pkg/hereandnow"
	"github.com/gin-gonic/gin"
)

func handleServeCommand(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Printf(`Start the Here and Now API Server

USAGE:
    hereandnow serve [OPTIONS]

DESCRIPTION:
    Starts the HTTP API server that provides REST endpoints for task management.
    The server handles user authentication, task filtering, and real-time updates.

OPTIONS:
    --port <port>       Server port (default: from config, usually 8080)
    --host <host>       Server host (default: from config, usually 127.0.0.1)
    --daemon, -d        Run as daemon (background process)
    --dev               Development mode (verbose logging, auto-reload)
    --help, -h         Show this help

EXAMPLES:
    hereandnow serve
    hereandnow serve --port 3000
    hereandnow serve --host 0.0.0.0 --port 8080
    hereandnow serve --daemon

ENDPOINTS:
    GET  /health                    Health check
    POST /api/v1/auth/login         User authentication
    POST /api/v1/auth/logout        User logout
    GET  /api/v1/tasks              List filtered tasks
    POST /api/v1/tasks              Create task
    GET  /api/v1/users/me           Get current user
    GET  /api/v1/context            Get current context
    POST /api/v1/context            Update context
`)
		return
	}

	executeServe(args)
}

func executeServe(args []string) {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Parse command line arguments
	port := config.Server.Port
	host := config.Server.Host
	daemon := false
	devMode := false

	for i, arg := range args {
		switch arg {
		case "--port":
			if i+1 < len(args) {
				if p, err := strconv.Atoi(args[i+1]); err == nil {
					port = p
				}
			}
		case "--host":
			if i+1 < len(args) {
				host = args[i+1]
			}
		case "--daemon", "-d":
			daemon = true
		case "--dev":
			devMode = true
		}
	}

	if daemon {
		fmt.Printf("Starting server in daemon mode on %s:%d\n", host, port)
		// In a real implementation, this would fork the process
		// For now, we'll just continue normally
	}

	// Set Gin mode
	if devMode {
		gin.SetMode(gin.DebugMode)
		fmt.Println("Running in development mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := InitDatabase(config.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize repositories
	userRepo := storage.NewUserRepository(db)
	taskRepo := storage.NewTaskRepository(db)
	locationRepo := storage.NewLocationRepository(db)
	contextRepo := storage.NewContextRepository(db)
	dependencyRepo := storage.NewTaskDependencyRepository(db)
	taskLocationRepo := storage.NewTaskLocationRepository(db)

	// Initialize services
	authService := auth.NewAuthService(userRepo)
	filterEngine := filters.NewFilterEngine()
	taskService := hereandnow.NewTaskService(taskRepo, contextRepo, dependencyRepo, taskLocationRepo, *filterEngine)
	contextService := hereandnow.NewContextService(contextRepo, locationRepo, nil, nil, nil)

	// Initialize handlers
	authHandler := api.NewAuthHandler(authService)
	taskHandler := api.NewTaskHandler(taskService, authService)
	userHandler := api.NewUserHandler(userRepo, authService)

	// Setup router
	router := setupRouter(authHandler, taskHandler, userHandler, authService)

	// Server configuration
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		fmt.Printf("🚀 Server starting on %s:%d\n", host, port)
		if devMode {
			fmt.Printf("📖 API Documentation: http://%s:%d/docs\n", host, port)
			fmt.Printf("🏥 Health Check: http://%s:%d/health\n", host, port)
		}
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server failed to start: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Server shutting down...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Server shutdown complete")
}

func setupRouter(authHandler *api.AuthHandler, taskHandler *api.TaskHandler, userHandler *api.UserHandler, authService *auth.AuthService) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "hereandnow-api",
			"version":   Version,
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes (require authentication)
		protected := v1.Group("/")
		protected.Use(authMiddleware(authService))
		{
			// User routes
			users := protected.Group("/users")
			{
				users.GET("/me", userHandler.GetCurrentUser)
				users.PATCH("/me", userHandler.UpdateCurrentUser)
			}

			// Task routes
			tasks := protected.Group("/tasks")
			{
				tasks.GET("", taskHandler.GetTasks)
				tasks.POST("", taskHandler.CreateTask)
				tasks.GET("/:taskId", taskHandler.GetTask)
				tasks.PATCH("/:taskId", taskHandler.UpdateTask)
				tasks.DELETE("/:taskId", taskHandler.DeleteTask)
				tasks.POST("/:taskId/assign", taskHandler.AssignTask)
				tasks.POST("/:taskId/complete", taskHandler.CompleteTask)
				tasks.GET("/:taskId/audit", taskHandler.GetTaskAudit)
			}

			// Context routes (placeholder)
			context := protected.Group("/context")
			{
				context.GET("", func(c *gin.Context) {
					c.JSON(http.StatusNotImplemented, gin.H{
						"error": "Context endpoints not yet implemented",
					})
				})
				context.POST("", func(c *gin.Context) {
					c.JSON(http.StatusNotImplemented, gin.H{
						"error": "Context endpoints not yet implemented",
					})
				})
			}

			// Location routes (placeholder)
			locations := protected.Group("/locations")
			{
				locations.GET("", func(c *gin.Context) {
					c.JSON(http.StatusNotImplemented, gin.H{
						"error": "Location endpoints not yet implemented",
					})
				})
				locations.POST("", func(c *gin.Context) {
					c.JSON(http.StatusNotImplemented, gin.H{
						"error": "Location endpoints not yet implemented",
					})
				})
			}
		}
	}

	// Static documentation (if exists)
	router.Static("/docs", "./docs")
	
	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Endpoint not found",
			"path":  c.Request.URL.Path,
		})
	})

	return router
}

// CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Authentication middleware
func authMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Store user ID in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}