SHELL := /bin/bash

# Where we'll put all built binaries
BIN_DIR := bin

.PHONY: all build clean run-client-resources run-client-prompts run-client-tools run-sse

## Default target: build everything
all: build

## Build the client and all server binaries
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/mcp-client ./examples/client
	go build -o $(BIN_DIR)/mcp-server-resources ./examples/server/resources
	go build -o $(BIN_DIR)/mcp-server-prompts ./examples/server/prompts
	go build -o $(BIN_DIR)/mcp-server-tools ./examples/server/tools
	go build -o $(BIN_DIR)/mcp-sse-server ./examples/sse/server
	go build -o $(BIN_DIR)/mcp-sse-client ./examples/sse/client

## Run the client with the resources-only server
run-client-resources: build
	@echo "=== Running client + resources server ==="
	$(BIN_DIR)/mcp-client --server-binary=$(BIN_DIR)/mcp-server-resources

## Run the client with the prompts-only server
run-client-prompts: build
	@echo "=== Running client + prompts server ==="
	$(BIN_DIR)/mcp-client --server-binary=$(BIN_DIR)/mcp-server-prompts

## Run the client with the tools-only server
run-client-tools: build
	@echo "=== Running client + tools server ==="
	$(BIN_DIR)/mcp-client --server-binary=$(BIN_DIR)/mcp-server-tools

## Run the SSE server (in one terminal) and client (in another)
run-sse-server: build
	@echo "=== Running SSE server on port 8080 ==="
	$(BIN_DIR)/mcp-sse-server

run-sse-client: build
	@echo "=== Running SSE client connecting to localhost:8080 ==="
	$(BIN_DIR)/mcp-sse-client

## Clean up built artifacts
clean:
	rm -rf $(BIN_DIR)
