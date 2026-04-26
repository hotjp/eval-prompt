.PHONY: build test lint clean tidy help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Binary name
BINARY_NAME=ep
BINARY_PATH=./bin/$(BINARY_NAME)

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: lint test build

## build: Build the binary (CLI ep + server)
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p ./bin
	cd web && npm run build 2>/dev/null || true
	rm -rf internal/gateway/web/dist && cp -r web/dist internal/gateway/web/
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/ep/

## build-server: Build the server binary
build-server:
	@echo "Building server..."
	@mkdir -p ./bin
	$(GOBUILD) $(LDFLAGS) -o ./bin/server ./cmd/server/

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## lint: Run linter
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) > /dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

## tidy: Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf ./bin
	@rm -f coverage.out coverage.html

## install: Install the binary to /usr/local/bin/
install:
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p /usr/local/bin/
	cp $(BINARY_PATH) /usr/local/bin/
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

## release: Build all platform binaries for release
release:
	@echo "Building release binaries..."
	@mkdir -p ./dist
	@cd web && npm run build
	@rm -rf internal/gateway/web/dist && cp -r web/dist internal/gateway/web/
	@echo "  Building all platforms in parallel..."
	$(GOBUILD) -o ./dist/ep-darwin-arm64 ./cmd/ep/ &
	$(GOBUILD) -o ./dist/ep-darwin-amd64 ./cmd/ep/ &
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o ./dist/ep-linux-arm64 ./cmd/ep/ &
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o ./dist/ep-linux-amd64 ./cmd/ep/ &
	@wait
	@echo "Done! Binaries in ./dist/"

## help: Show this help message
help:
	@echo "Available targets:"
	@echo "  build          Build the CLI (ep) to ./bin/"
	@echo "  build-server   Build the server binary"
	@echo "  release        Build all platform binaries to ./dist/"
	@echo "  test           Run tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  lint           Run linter"
	@echo "  tidy           Tidy dependencies"
	@echo "  fmt            Format code"
	@echo "  clean          Clean build artifacts"
	@echo "  install        Install to /usr/local/bin/"
	@echo "  help           Show this help message"
