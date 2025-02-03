[![Go Reference](https://pkg.go.dev/badge/github.com/dwrtz/mcp-go.svg)](https://pkg.go.dev/github.com/dwrtz/mcp-go)

# Model Context Protocol SDK for Go

A Go implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io).

## Overview

This SDK provides a comprehensive implementation of MCP in Go, allowing developers to:

- Create MCP clients to connect to MCP servers
- Build MCP servers that provide context and capabilities
- Handle all core MCP features: resources, prompts, tools, and more
- Support for both stdio and SSE transport

## Examples

The SDK includes several example implementations:

### Standard stdio Examples
See the [example client](examples/client/main.go) that runs the example servers:
- [prompts](examples/server/prompts/main.go)
- [tools](examples/server/tools/main.go)
- [resources](examples/server/resources/main.go)

### SSE Transport Examples
See the SSE (Server-Sent Events) transport examples:
- [SSE server](examples/sse/server/main.go)
- [SSE client](examples/sse/client/main.go)

### Running the Examples

Run the stdio-based examples with different servers:
- `make run-client-prompts`
- `make run-client-tools`
- `make run-client-resources`

Run the SSE examples (in separate terminals):
1. Start the SSE server: `make run-sse-server`
2. Start the SSE client: `make run-sse-client`

The SSE examples accept command-line flags:
- Server: `--addr` to specify listen address (default ":8080")
- Client: `--server` to specify server address (default "localhost:8080")

## Development Status

This SDK is currently in development. While core functionality is implemented, some features are still in progress:

- [ ] `notifications/cancelled` for request cancellation
- [ ] `notifications/progress` for long-running operations
- [ ] `logging/setLevel` and `notifications/message` for logs
- [x] SSE transport
- [ ] Advanced examples

## License

[MIT License](LICENSE)
