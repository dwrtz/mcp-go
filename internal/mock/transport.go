package mock

import (
	"io"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/internal/transport/stdio"
	"github.com/dwrtz/mcp-go/pkg/logger"
)

// NewMockPipeTransports returns two separate transports
// (e.g. serverTransport, clientTransport) that communicate
// with each other via in-process pipes. Both sides implement
// transport.Transport using stdio pipes.
func NewMockPipeTransports(l logger.Logger) (transport.Transport, transport.Transport) {
	// Create the pipe pairs
	serverStdinR, serverStdinW := io.Pipe()
	serverStdoutR, serverStdoutW := io.Pipe()
	clientStdinR, clientStdinW := io.Pipe()
	clientStdoutR, clientStdoutW := io.Pipe()

	// Wire up:
	//   serverStdout -> clientStdin
	//   clientStdout -> serverStdin
	// so that anything the server writes is read by the client, and vice versa.
	go func() {
		defer serverStdinW.Close()
		io.Copy(serverStdinW, clientStdoutR)
	}()
	go func() {
		defer clientStdinW.Close()
		io.Copy(clientStdinW, serverStdoutR)
	}()

	// Build two transports using StdioTransport
	//   serverTransport: reads from serverStdinR / writes to serverStdoutW
	//   clientTransport: reads from clientStdinR / writes to clientStdoutW
	serverTransport := stdio.NewTransport(serverStdinR, serverStdoutW)
	serverTransport.SetLogger(l)
	clientTransport := stdio.NewTransport(clientStdinR, clientStdoutW)
	clientTransport.SetLogger(l)

	return serverTransport, clientTransport
}
