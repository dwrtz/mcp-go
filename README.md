# Model Context Protocol SDK for Go

A Go implementation of the [Model Context Protocol (MCP)](https://modelcontextprotocol.io), enabling seamless integration between LLMs and external data sources/tools.

## Overview

This SDK provides a comprehensive implementation of MCP in Go, allowing developers to:

- Create MCP clients to connect to MCP servers
- Build MCP servers that provide context and capabilities
- Handle all core MCP features: resources, prompts, tools, and more
- Support for stdio transport


## Development Status

This SDK is currently in development. While core functionality is implemented, some features are still in progress:

- [ ] `notifications/cancelled` for request cancellation
- [ ] `notifications/progress` for lon-running ops
- [ ] `logging/setLevel` and `notifications/message` for logs
- [ ] examples
- [ ] documentation
- [ ] versioned releases


## License

[MIT License](LICENSE)
