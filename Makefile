.PHONY: help build run test clean install api cli

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: build-api build-cli ## Build all binaries

build-api: ## Build the API server
	@echo "Building API server..."
	@go build -o bin/featury-api ./cmd/api

build-cli: ## Build the CLI tool
	@echo "Building CLI tool..."
	@go build -o bin/featury ./cmd/cli

# Run targets
run-api: ## Run the API server
	@echo "Starting API server..."
	@go run ./cmd/api

run-cli: ## Run the CLI tool
	@go run ./cmd/cli

# Development targets
dev: ## Run API server with hot reload (requires air)
	@if command -v air > /dev/null; then \
		air -c .air.toml; \
	else \
		echo "Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Or run: make run-api"; \
	fi

# Testing and quality
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter (requires golangci-lint)
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install from: https://golangci-lint.run/"; \
	fi

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# Installation
install: build ## Install binaries to $GOPATH/bin
	@echo "Installing binaries..."
	@cp bin/featury-api $(GOPATH)/bin/
	@cp bin/featury $(GOPATH)/bin/

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Docker targets (optional)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t featury:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 featury:latest