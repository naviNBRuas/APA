# Makefile for APA Agent

BINARY_NAME=agentd
BUILD_DIR=bin
CMD_PATH=./cmd/agentd/main.go

.PHONY: all build clean test build-linux build-windows build-darwin

all: build

build:
	@echo "Building for current OS..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

build-linux:
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@echo "Building for Linux (arm64)..."
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)

build-windows:
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)

build-darwin:
	@echo "Building for macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
