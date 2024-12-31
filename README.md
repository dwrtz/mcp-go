# MCP-Go

Work-in-progress implementation of a Model Context Protocol (MCP) server and client in Go that communicate via JSON-RPC 2.0.

---

## Toy Example

1. A **server** (`mcpd`) reads from `stdin` and writes to `stdout`.
2. A **client** (`mcpc`) also expects to read from `stdin` and write to `stdout`.

Normally, running both in the same shell collides. Instead, we use **named pipes** to connect them in full-duplex mode (two FIFOs: one for each direction). The Makefile target `run-both` demonstrates how to:
- Build both binaries
- Create two FIFOs
- Launch the server **in the background**, reading/writing from those FIFOs
- Launch the client **in the foreground**, also connected to the same FIFOs
- Kill the server once the client is done

This way, you can see a fully automated round-trip test of the client and server.

---

## Usage

1. **Clone and navigate:**
   ```bash
   git clone https://github.com/dwrtz/mcp-go.git
   cd mcp-go
   ```

2. **Run `make run-both`**  
   This will:
   - Build `mcpd` (server) and `mcpc` (client) into `./bin/`
   - Create `/tmp/pipe-in` and `/tmp/pipe-out` FIFOs
   - Start `mcpd` in the background
   - Start `mcpc` in the foreground to send a `ping` request
   - Print output from the client and eventually kill the server
   - Clean up the FIFOs

You should see log messages in your terminal showing the “ping → pong” sequence.

---
