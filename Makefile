BIN_DIR := bin

.PHONY: all build run-both server client pipe clean

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/mcpd ./cmd/mcpd
	go build -o $(BIN_DIR)/mcpc ./cmd/mcpc
	@echo "Binaries built in '$(BIN_DIR)'!"

# Automated test: server <-> client using named pipes in one go
run-both: build
	@rm -f /tmp/pipe-in /tmp/pipe-out
	mkfifo /tmp/pipe-in
	mkfifo /tmp/pipe-out
	@echo "[Makefile] Starting server in background..."
	$(BIN_DIR)/mcpd < /tmp/pipe-in > /tmp/pipe-out & \
	SERVER_PID=$$!; \
	echo "[Makefile] Server PID: $$SERVER_PID"; \
	sleep 1; \
	echo "[Makefile] Running client now..."; \
	$(BIN_DIR)/mcpc < /tmp/pipe-out > /tmp/pipe-in; \
	echo "[Makefile] Client done."; \
	echo "[Makefile] Stopping server..."; \
	kill $$SERVER_PID || true; \
	rm -f /tmp/pipe-in /tmp/pipe-out

# (Optional) Just start the server (blocking) for debugging
server: build
	@echo "Starting server..."
	$(BIN_DIR)/mcpd

# (Optional) Just start the client (blocking) for debugging
client: build
	@echo "Starting client..."
	$(BIN_DIR)/mcpc

# (Optional) If you wanted to manually run them in 2 terminals with named pipes:
pipe: build
	@rm -f /tmp/pipe-in /tmp/pipe-out
	@mkfifo /tmp/pipe-in
	@mkfifo /tmp/pipe-out
	@echo "Named pipes created: /tmp/pipe-in, /tmp/pipe-out."
	@echo "In one terminal:   $(BIN_DIR)/mcpd < /tmp/pipe-in > /tmp/pipe-out"
	@echo "In another term.:  $(BIN_DIR)/mcpc < /tmp/pipe-out > /tmp/pipe-in"

clean:
	@rm -rf $(BIN_DIR)
	@rm -f /tmp/pipe-in /tmp/pipe-out
	@echo "Cleaned build artifacts."
