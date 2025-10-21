# Makefile for Spider-golang

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=spider
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME)_windows.exe

# Main application path
MAIN_PATH=cmd/spider

# Build for current platform
build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_WINDOWS) $(MAIN_PATH)

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) $(MAIN_PATH)

# Build for macOS
build-mac:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) $(MAIN_PATH)

# Build for all platforms
build-all: build-windows build-linux build-mac

# Install dependencies
deps:
	$(GOMOD) tidy

# Run tests
test:
	$(GOTEST) -v ./...

# Clean build files
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_WINDOWS)

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)
	./$(BINARY_NAME)

# Install the application
install:
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) $(MAIN_PATH)

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build for current platform"
	@echo "  build-windows - Build for Windows"
	@echo "  build-linux   - Build for Linux"
	@echo "  build-mac     - Build for macOS"
	@echo "  build-all     - Build for all platforms"
	@echo "  deps          - Install dependencies"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build files"
	@echo "  run           - Run the application"
	@echo "  install       - Install the application"
	@echo "  help          - Show this help message"

.PHONY: build build-windows build-linux build-mac build-all deps test clean run install help