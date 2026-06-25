# GoLogify Makefile
# Build automation for development and releases

BINARY_NAME := gologify
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -X github.com/StardustEnigma/gologify/cmd.Version=$(VERSION)"

# Go settings
GOBIN := $(shell go env GOPATH)/bin

.PHONY: all build run test lint clean install fmt vet help

## all: Build the binary (default target)
all: build

## build: Compile the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME).exe .

## run: Build and run with arguments (use ARGS="analyze app.log")
run: build
	./$(BINARY_NAME).exe $(ARGS)

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v ./...

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race -v ./...

## test-cover: Run tests with coverage report
test-cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## bench: Run benchmarks
bench:
	go test -bench=. -benchmem ./...

## lint: Run go vet and staticcheck
lint: vet
	@echo "Running linters..."
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -f coverage.out coverage.html

## build-all: Cross-compile for all platforms
build-all:
	@echo "Building for all platforms..."
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' 2>/dev/null || sed -n 's/^## //p' $(MAKEFILE_LIST)
