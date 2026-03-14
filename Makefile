# Makefile for Polar-Gosling

# Binary names
BINARY_NAME=gosling
RIFT_BINARY_NAME=rift

# Build directory
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Main package paths
MAIN_PATH=./cmd/gosling
RIFT_MAIN_PATH=./cmd/rift

# Detect OS
ifeq ($(OS),Windows_NT)
	BINARY_EXT=.exe
else
	BINARY_EXT=
endif

# Build targets
.PHONY: all build build-all build-rift build-rift-all clean test linux windows darwin rift-linux rift-windows rift-darwin

all: clean build build-rift

build:
	@echo "Building gosling for current OS..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT) $(MAIN_PATH)

build-rift:
	@echo "Building rift for current OS..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(RIFT_BINARY_NAME)$(BINARY_EXT) $(RIFT_MAIN_PATH)

build-all: linux windows darwin rift-linux rift-windows rift-darwin

linux:
	@echo "Building gosling for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)

windows:
	@echo "Building gosling for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

darwin:
	@echo "Building gosling for Darwin (macOS)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

rift-linux:
	@echo "Building rift for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(RIFT_BINARY_NAME)-linux-amd64 $(RIFT_MAIN_PATH)

rift-windows:
	@echo "Building rift for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(RIFT_BINARY_NAME)-windows-amd64.exe $(RIFT_MAIN_PATH)

rift-darwin:
	@echo "Building rift for Darwin (macOS)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(RIFT_BINARY_NAME)-darwin-amd64 $(RIFT_MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(RIFT_BINARY_NAME)-darwin-arm64 $(RIFT_MAIN_PATH)

clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	@$(GOTEST) -v ./...
