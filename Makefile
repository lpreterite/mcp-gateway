.PHONY: build test lint clean run install-deps tidy fmt vet check help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=mcp-gateway
BINARY_DIR=dist
VERSION?=1.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/gateway

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Lint the code
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Format the code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out coverage.html
	@rm -f gateway-test mcp-gateway-test

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_DIR)/$(BINARY_NAME)

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Cross-compile for multiple platforms
build-all: build-darwin-arm64 build-darwin-amd64 build-linux-amd64 build-windows-amd64

build-darwin-arm64:
	@echo "Building for darwin/arm64..."
	@mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/gateway

build-darwin-amd64:
	@echo "Building for darwin/amd64..."
	@mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/gateway

build-linux-amd64:
	@echo "Building for linux/amd64..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/gateway

build-windows-amd64:
	@echo "Building for windows/amd64..."
	@mkdir -p $(BINARY_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/gateway

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Cross-compile for all platforms"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Lint the code"
	@echo "  fmt            - Format the code"
	@echo "  vet            - Run go vet"
	@echo "  tidy           - Tidy go.mod"
	@echo "  check          - Run all checks (fmt, vet, lint, test)"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Build and run the application"
	@echo "  install-deps   - Install dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  help           - Show this help message"
