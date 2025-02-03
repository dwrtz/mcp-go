package sse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/logger"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// SSETransport implements Transport using Server-Sent Events
type SSETransport struct {
	router *transport.MessageRouter
	done   chan struct{}

	httpServer *http.Server

	// We hold our net.Listener if we're in server mode
	listener  net.Listener
	client    chan []byte
	mu        sync.Mutex
	connected bool

	endpoint      string
	connectionErr error // non-nil if client SSE connection fails

	logger logger.Logger
	// Actual address we ended up listening on (for ephemeral port usage)
	boundAddr string
}

// NewSSEServer creates a new SSE transport in server mode.
// If addr == ":0", we will bind an ephemeral port automatically.
func NewSSEServer(addr string) *SSETransport {
	router := transport.NewMessageRouter()
	doneCh := make(chan struct{})
	clientCh := make(chan []byte, 32) // small buffer

	return &SSETransport{
		router: router,
		done:   doneCh,
		client: clientCh,
		// We'll set up httpServer + net.Listener in Start()
		httpServer: &http.Server{},
		boundAddr:  addr, // store the desired address (may be ":0")
	}
}

// NewSSEClient creates a new SSE transport in client mode
func NewSSEClient(serverAddr string) *SSETransport {
	return &SSETransport{
		router:   transport.NewMessageRouter(),
		done:     make(chan struct{}),
		endpoint: fmt.Sprintf("http://%s/send", serverAddr),
	}
}

// Start begins processing messages. In server mode, we create a net.Listener
// and call httpServer.Serve() manually so we can retrieve the ephemeral port.
func (t *SSETransport) Start(ctx context.Context) error {
	if t.httpServer != nil {
		// SERVER MODE
		mux := http.NewServeMux()
		mux.HandleFunc("/events", t.handleSSE)
		mux.HandleFunc("/send", t.handleSend)
		t.httpServer.Handler = mux

		// 1) Create a listener (this picks an ephemeral port if boundAddr == ":0")
		ln, err := net.Listen("tcp", t.boundAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", t.boundAddr, err)
		}
		t.listener = ln
		t.boundAddr = ln.Addr().String() // store the actual address/port

		// 2) Start serving
		go func() {
			if err := t.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
				t.Logf("HTTP server error: %v", err)
			}
		}()
		return nil
	}

	// CLIENT MODE...
	go t.connectSSE(ctx)
	return nil
}

// BoundAddr returns the actual address the SSE server is listening on.
func (t *SSETransport) BoundAddr() string {
	return t.boundAddr
}

// connectSSE tries a single SSE connection to /events in client mode.
// We intentionally do NOT shut down the entire transport if it fails.
func (t *SSETransport) connectSSE(ctx context.Context) {
	serverURL := strings.Replace(t.endpoint, "/send", "/events", 1)

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
	if err != nil {
		t.Logf("Failed to create SSE request: %v", err)
		t.setConnectionErr(err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("Failed to connect to SSE: %v", err)
		t.setConnectionErr(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf("failed to connect to SSE: status code %d", resp.StatusCode)
		t.Logf(errMsg.Error())
		t.setConnectionErr(errMsg)
		return
	}

	// If we reach here, SSE connected successfully. Process the stream.
	t.processSSE(resp.Body)
}

// processSSE reads lines from SSE response body, parsing JSON messages.
func (t *SSETransport) processSSE(r io.Reader) {
	scanner := bufio.NewScanner(r)
	var buffer bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			buffer.WriteString(data)
			continue
		}

		// blank line indicates end of SSE event
		if line == "" && buffer.Len() > 0 {
			var msg types.Message
			if err := json.Unmarshal(buffer.Bytes(), &msg); err != nil {
				t.Logf("Failed to unmarshal SSE message: %v", err)
			} else {
				t.router.Handle(context.Background(), &msg) // pass a BG context
			}
			buffer.Reset()
		}
	}
	if err := scanner.Err(); err != nil {
		t.Logf("SSE scanner error: %v", err)
	}
}

// setConnectionErr safely sets a client-side connection error
func (t *SSETransport) setConnectionErr(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connectionErr = err
}

func (t *SSETransport) getConnectionErr() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connectionErr
}

// Send sends a message through the transport
func (t *SSETransport) Send(ctx context.Context, msg *types.Message) error {
	if t.httpServer == nil {
		// CLIENT mode
		if cErr := t.getConnectionErr(); cErr != nil {
			return cErr
		}
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", t.endpoint, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return nil
	}

	// SERVER mode
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return fmt.Errorf("no client connected")
	}
	select {
	case t.client <- data:
		return nil
	default:
		return fmt.Errorf("client message buffer full")
	}
}

// GetRouter returns the message router
func (t *SSETransport) GetRouter() *transport.MessageRouter {
	return t.router
}

// Close gracefully shuts down the server
func (t *SSETransport) Close() error {
	select {
	case <-t.done:
		return nil
	default:
		close(t.done)
	}
	if t.httpServer != nil {
		_ = t.httpServer.Close()
		if t.listener != nil {
			_ = t.listener.Close()
		}
	}
	return nil
}

// Done returns a channel that is closed when the transport is closed
func (t *SSETransport) Done() <-chan struct{} {
	return t.done
}

// Logf logs a formatted message
func (t *SSETransport) Logf(format string, args ...interface{}) {
	if t.logger != nil {
		t.logger.Logf(format, args...)
	}
}

// SetLogger sets the logger
func (t *SSETransport) SetLogger(l logger.Logger) {
	t.logger = l
	t.router.SetLogger(l)
}

// handleSSE is the handler for /events. Only one client at a time is allowed.
func (t *SSETransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	t.mu.Lock()
	if t.connected {
		t.mu.Unlock()
		http.Error(w, "Client already connected", http.StatusConflict)
		return
	}
	t.connected = true
	t.mu.Unlock()

	t.Logf("Client connected")

	defer func() {
		t.mu.Lock()
		t.connected = false
		t.mu.Unlock()
		t.Logf("Client disconnected")
	}()

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Stream SSE messages from t.client channel
	for {
		select {
		case <-t.done:
			return
		case <-r.Context().Done():
			// The client disconnected
			return
		case data := <-t.client:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleSend is the handler for /send. It receives an HTTP POST JSON message from the client
// and routes it to the server's message router.
func (t *SSETransport) handleSend(w http.ResponseWriter, r *http.Request) {
	var msg types.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid message: %v", err), http.StatusBadRequest)
		return
	}

	t.router.Handle(r.Context(), &msg)
	w.WriteHeader(http.StatusOK)
}
