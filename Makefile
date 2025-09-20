# Task Breaker Makefile
# Provides convenient commands for development and testing

# Variables
BINARY_NAME=task-breaker
BUILD_DIR=build
VERSION?=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Default target
.PHONY: all
all: test build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/chat.go

# Run tests
.PHONY: test
test:
	@echo "Running unit tests..."
	go test -v -race -short ./ai ./backends/mock ./chat -run "^Test[^I]"

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test -v -race -timeout=30m -run "TestIntegration" -short ./...

# Run all tests
.PHONY: test-all
test-all: test test-integration

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./ai ./backends/mock ./chat
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./...

# Lint code (optional - requires golangci-lint installation)
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not found, skipping import formatting. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet ./...

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

# Update dependencies
.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install the binary
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/chat.go

# Run the CLI
.PHONY: run
run: build
	@echo "Starting $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Development workflow - format, vet, test (core tools only)
.PHONY: check
check: fmt vet test

# Development workflow with optional linting - format, vet, lint, test
.PHONY: check-all
check-all: fmt vet lint test

# Security scan (optional - requires gosec installation)
.PHONY: security
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/chat.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/chat.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/chat.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/chat.go


# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Run tests and build binary"
	@echo "  build        - Build the binary"
	@echo "  test         - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-all     - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  bench        - Run benchmarks"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  deps         - Download dependencies"
	@echo "  deps-update  - Update dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install binary"
	@echo "  run          - Build and run CLI"
	@echo "  check        - Run format, vet, and test (core tools only)"
	@echo "  check-all    - Run format, vet, lint, and test (requires linter)"
	@echo "  security     - Run security scan (optional tool)"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  help         - Show this help message"

