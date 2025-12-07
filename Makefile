# ServerEye Bot Makefile

.PHONY: build test clean docker-build docker-up docker-down lint

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name
BOT_BINARY=servereye-bot

# Build directory
BUILD_DIR=build

# Default target
all: build

# Версия и build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -X github.com/servereye/servereyebot/internal/version.Version=$(VERSION) \
	-X github.com/servereye/servereyebot/internal/version.BuildDate=$(BUILD_DATE) \
	-X github.com/servereye/servereyebot/internal/version.GitCommit=$(GIT_COMMIT)

# Build bot
build:
	@echo "Building bot $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BOT_BINARY) ./cmd/bot
	@echo "✅ Bot built: $(BUILD_DIR)/$(BOT_BINARY)"

# Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -short ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -short -coverprofile=coverage.txt -covermode=atomic ./...
	@echo ""
	@echo "📊 Coverage Report:"
	go tool cover -func=coverage.txt | grep -E "internal/bot|^total:"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -f deployments/Dockerfile.bot -t servereye/bot:latest .

# Start services with Docker Compose
docker-up:
	@echo "Starting services..."
	cd deployments && docker-compose up -d

# Stop services
docker-down:
	@echo "Stopping services..."
	cd deployments && docker-compose down

# Development target
dev-bot:
	@echo "Running bot in development mode..."
	$(GOCMD) run ./cmd/bot --log-level=debug

# Security scan (requires gosec)
security:
	@echo "Running security scan..."
	gosec ./...

# Check for vulnerabilities (requires govulncheck)
vuln-check:
	@echo "Checking for vulnerabilities..."
	govulncheck ./...

# Release build with optimizations
RELEASE_LDFLAGS = -w -s \
	-X github.com/servereye/servereyebot/internal/version.Version=$(VERSION) \
	-X github.com/servereye/servereyebot/internal/version.BuildDate=$(BUILD_DATE) \
	-X github.com/servereye/servereyebot/internal/version.GitCommit=$(GIT_COMMIT)

release: clean
	@echo "Building release binary for version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="$(RELEASE_LDFLAGS)" -o $(BUILD_DIR)/$(BOT_BINARY)-linux-amd64 ./cmd/bot
	@echo "✅ Release build complete!"
	@$(BUILD_DIR)/$(BOT_BINARY)-linux-amd64 --version

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build         - Build bot"
	@echo "  release       - Build optimized release binary"
	@echo ""
	@echo "Tests:"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  security      - Run security scan"
	@echo "  vuln-check    - Check for vulnerabilities"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-up     - Start services with Docker Compose"
	@echo "  docker-down   - Stop services"
	@echo ""
	@echo "Development:"
	@echo "  dev-bot       - Run bot in development mode"
	@echo ""
	@echo "Utilities:"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  version       - Show current version"
	@echo "  help          - Show this help"
