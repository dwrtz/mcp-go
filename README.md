[![Go Reference](https://pkg.go.dev/badge/github.com/dwrtz/mcp-go.svg)](https://pkg.go.dev/github.com/dwrtz/mcp-go)

# Model Context Protocol SDK for Go

A Go implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io).

## Overview

This SDK provides a comprehensive implementation of MCP in Go, allowing developers to:

- Create MCP clients to connect to MCP servers
- Build MCP servers that provide context and capabilities
- Handle all core MCP features: resources, prompts, tools, and more
- Support for stdio transport

## Examples

See the [example client](examples/client/main.go) that runs the example servers:
- [prompts](examples/server/prompts/main.go)
- [tools](examples/server/tools/main.go)
- [resources](examples/server/resources/main.go)

### Running the examples

Run the client with the different servers:
- `make run-client-prompts`
- `make run-client-tools`
- `make run-client-resources`


## Development Status

This SDK is currently in development. While core functionality is implemented, some features are still in progress:

- [ ] `notifications/cancelled` for request cancellation
- [ ] `notifications/progress` for long-running operations
- [ ] `logging/setLevel` and `notifications/message` for logs
- [ ] SSE transport
- [ ] Advanced examples


## License

[MIT License](LICENSE)
