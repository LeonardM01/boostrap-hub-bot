# Bootstrap Hub Bot - Makefile
# A Discord bot for solo founders

.PHONY: build run register remove-commands invite clean test help

# Binary name
BINARY_NAME=bootstrap-hub-bot
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Default target
all: build

## Build the bot binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/bot
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## Run the bot (starts and listens for commands)
run: build
	@echo "Starting Bootstrap Hub Bot..."
	./$(BUILD_DIR)/$(BINARY_NAME)

## Register all slash commands with Discord
## This should be run whenever you add or modify commands
register: build
	@echo "Registering slash commands..."
	./$(BUILD_DIR)/$(BINARY_NAME) -register

## Remove all registered slash commands from Discord
remove-commands: build
	@echo "Removing all slash commands..."
	./$(BUILD_DIR)/$(BINARY_NAME) -remove-commands

## Display the bot invite URL
invite: build
	@./$(BUILD_DIR)/$(BINARY_NAME) -invite

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

## Display help
help:
	@echo ""
	@echo "Bootstrap Hub Bot - Makefile Commands"
	@echo "======================================"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build            Build the bot binary"
	@echo "  run              Build and run the bot"
	@echo "  register         Register all slash commands with Discord"
	@echo "  remove-commands  Remove all slash commands from Discord"
	@echo "  invite           Display the bot invite URL"
	@echo "  deps             Download and tidy dependencies"
	@echo "  test             Run tests"
	@echo "  clean            Clean build artifacts"
	@echo "  help             Display this help message"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. Copy .env.example to .env and fill in your credentials"
	@echo "  2. Run 'make register' to register commands"
	@echo "  3. Run 'make invite' to get the invite URL"
	@echo "  4. Run 'make run' to start the bot"
	@echo ""
