# Here and Now Task Management System Makefile
.PHONY: build test lint clean dev install help
.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME=hereandnow
BUILD_DIR=bin
GO_FILES=$(shell find . -name "*.go" -type f)

# Build the CLI binary
build: ## Build the CLI binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/hereandnow

# Run all tests
test: ## Run all tests
	@echo "Running tests..."
	go test ./... -v

# Run contract tests specifically
test-contract: ## Run contract tests (API schema validation)
	@echo "Running contract tests..."
	go test ./tests/contract/... -v

# Run integration tests
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test ./tests/integration/... -v

# Run unit tests
test-unit: ## Run unit tests
	@echo "Running unit tests..."
	go test ./tests/unit/... -v

# Run benchmarks
benchmark: ## Run performance benchmarks
	@echo "Running benchmarks..."
	go test ./... -bench=. -benchmem

# Lint the code
lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	go clean

# Development server with hot reload
dev: ## Start development server
	@echo "Starting development server..."
	go run ./cmd/hereandnow serve --port=8080

# Database operations
migrate-up: ## Apply database migrations
	@echo "Applying database migrations..."
	go run ./cmd/hereandnow migrate up

migrate-down: ## Rollback database migrations
	@echo "Rolling back database migrations..."
	go run ./cmd/hereandnow migrate down

reset-db: ## Reset database (development only)
	@echo "Resetting database..."
	rm -f hereandnow.db
	$(MAKE) migrate-up

# Installation
install: build ## Install binary to system PATH
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Cross-platform builds
release: ## Build for multiple platforms
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/hereandnow
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/hereandnow
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/hereandnow

# Docker development
docker-dev: ## Start development environment in Docker
	@echo "Starting Docker development environment..."
	docker compose up --build

# Help target
help: ## Show this help message
	@echo "Here and Now Task Management System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m\033[0m"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)