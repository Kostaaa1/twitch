# Variables
BUILD_DIR := build
GO := go
VERSION := $(shell git describe --tags --always)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)"

# Targets
.PHONY: all clean build run

all: clean build

build:
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) -ldflags $(LDFLAGS) .

run: build
	@echo "Running the application..."
	./$(BUILD_DIR)/$(APP_NAME)

clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	$(GO) test ./...

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

lint:
	@echo "Running golint..."
	golangci-lint run

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

