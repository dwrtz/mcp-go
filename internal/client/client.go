package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
	"github.com/sourcegraph/jsonrpc2"
)

// clientHandler receives all responses from the server.
type clientHandler struct {
	client *Client
	pingID string
	once   sync.Once
}

func (h *clientHandler) Handle(ctx context.Context, msg *types.Message) (*types.Message, error) {
	// If it's a response (has msg.ID) and a Result
	if msg.ID != nil && msg.Result != nil {
		// Convert to something readable
		var result map[string]interface{}
		if err := json.Unmarshal(*msg.Result, &result); err == nil {
			fmt.Fprintf(os.Stderr, "Client got response for ID=%v: %v\n", *msg.ID, result)
		} else {
			fmt.Fprintf(os.Stderr, "Client got response for ID=%v (unmarshal error: %v): %s\n",
				*msg.ID, err, string(*msg.Result))
		}

		// If this was our "ping" request, we can close the client
		if h.pingID == h.idToString(msg.ID) {
			h.once.Do(func() {
				fmt.Fprintf(os.Stderr, "Client: Received 'ping' response, closing now...\n")
				h.client.Close()
			})
		}
	}
	return nil, nil
}

func (h *clientHandler) idToString(idPtr *types.RequestID) string {
	if idPtr == nil {
		return ""
	}
	id := *idPtr
	if id.IsString {
		return id.Str
	}
	// For number IDs, convert to string
	return fmt.Sprintf("%d", id.Num)
}

// Client is a minimal struct that can send requests and handle responses.
type Client struct {
	transport transport.Transport
	mu        sync.Mutex
	closed    bool
}

// NewClient creates a new Client with the given transport.
func NewClient(t transport.Transport) *Client {
	return &Client{
		transport: t,
	}
}

// Start begins reading responses in a goroutine.
func (c *Client) Start(ctx context.Context, pingID string) error {
	// Set our handler
	h := &clientHandler{
		client: c,
		pingID: pingID,
	}
	c.transport.SetHandler(h)

	// Start the read loop in the background
	go func() {
		err := c.transport.Start(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "client transport stopped: %v\n", err)
		}
	}()

	return nil
}

// Ping sends a "ping" request to the server.
func (c *Client) Ping(ctx context.Context, pingID string) error {
	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      newID(pingID),
		Method:  "ping",
	}

	err := c.transport.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}
	return nil
}

// Close closes the client.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.transport.Close()
}

// Done returns a channel that is closed when the client transport is closed.
func (c *Client) Done() <-chan struct{} {
	return c.transport.Done()
}

// newID returns a pointer to a RequestID with a string value
func newID(str string) *types.RequestID {
	id := jsonrpc2.ID{
		Str:      str,
		IsString: true,
	}
	return &id
}
