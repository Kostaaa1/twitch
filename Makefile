BIN_DIR := ./bin
# DOWNLOADER_BIN := $(BIN_DIR)/downloader
# CHAT_BIN := $(BIN_DIR)/twitchat

.PHONY: all clean build-downloader build-chat build run

all: clean build

build:
	@echo "Building Twitch CLI..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/twitch ./cmd
	@echo "Twitch CLI built successfully."

clean:
	@echo "Cleaning up old binaries..."
	@rm -rf $(BIN_DIR)
	@echo "Cleanup complete."