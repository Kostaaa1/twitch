BIN_DIR := ./bin
DOWNLOADER_BIN := $(BIN_DIR)/twitchdl
CHAT_BIN := $(BIN_DIR)/twitchat

.PHONY: all clean build-downloader build-chat build run

all: clean build

build: build-downloader build-chat

build-downloader:
	@echo "Building twitch downloader..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(DOWNLOADER_BIN) ./cli/downloader
	@echo "Twitch downloader built successfully."

build-chat:
	@echo "Building twitch chat..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(CHAT_BIN) ./cli/chat
	@echo "Twitch chat built successfully."

clean:
	@echo "Cleaning up old binaries..."
	@rm -rf $(BIN_DIR)
	@echo "Cleanup complete."
