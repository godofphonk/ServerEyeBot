.PHONY: build run test lint clean docker-build docker-run docker-stop install-deps help

# Default target
all: build

# Build the application
build:
	@echo "Building ServerEyeBot..."
	go build -o bin/servereye-bot ./cmd/bot

# Run the application
run: build
	@echo "Running ServerEyeBot..."
	./bin/servereye-bot

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t servereye-bot:latest .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker-compose up -d

# Docker stop
docker-stop:
	@echo "Stopping Docker container..."
	docker-compose down

# Docker logs
docker-logs:
	@echo "Showing Docker logs..."
	docker-compose logs -f

# Development setup
dev-setup: install-deps
	@echo "Setting up development environment..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Generate mocks (if using mockery)
mocks:
	@echo "Generating mocks..."
	mockery --all --output=mocks

# Security scan
security:
	@echo "Running security scan..."
	gosec ./...

# Dependency check
deps-check:
	@echo "Checking dependencies..."
	go list -u -m all

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Help
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Build and run the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  clean          - Clean build artifacts"
	@echo "  install-deps   - Install dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-stop    - Stop Docker container"
	@echo "  docker-logs    - Show Docker logs"
	@echo "  dev-setup      - Set up development environment"
	@echo "  mocks          - Generate mocks"
	@echo "  security       - Run security scan"
	@echo "  deps-check     - Check dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  help           - Show this help message"
