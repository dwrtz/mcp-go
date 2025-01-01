package client

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/dwrtz/mcp-go/internal/transport"
	"github.com/dwrtz/mcp-go/pkg/types"
)

// clientHandler receives all responses from the server.
type clientHandler struct {
	client *Client
	pingID string // Note: This will now be "1" to match our numeric ID
	logger interface {
		Logf(format string, args ...interface{})
	}
	once sync.Once
}

func (h *clientHandler) Handle(ctx context.Context, msg *types.Message) (*types.Message, error) {
	if h.logger != nil {
		h.logger.Logf("Client received message: %+v", msg)
	}

	// Check if it's a response to our ping
	if msg.ID != nil && msg.Result != nil {
		msgID := h.idToString(msg.ID)
		if h.logger != nil {
			h.logger.Logf("Client comparing IDs - received: %v, expected: %v", msgID, h.pingID)
		}

		if msgID == h.pingID {
			if h.logger != nil {
				h.logger.Logf("Client received matching ping response, closing")
			}
			h.once.Do(func() {
				h.client.Close()
			})
		}
	}
	return nil, nil
}

func (h *clientHandler) idToString(id *types.RequestID) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf("%v", *id)
}

// Client is a minimal struct that can send requests and handle responses.
type Client struct {
	transport transport.Transport
	ready     chan struct{}
	logger    interface {
		Logf(format string, args ...interface{})
	}
	mu     sync.Mutex
	closed bool
}

// NewClient creates a new Client with the given transport and optional logger
func NewClient(t transport.Transport, logger interface{ Logf(string, ...interface{}) }) *Client {
	return &Client{
		transport: t,
		ready:     make(chan struct{}),
		logger:    logger,
	}
}

func (c *Client) Start(ctx context.Context, pingID string) error {
	h := &clientHandler{
		client: c,
		pingID: pingID,
		logger: c.logger,
	}
	c.transport.SetHandler(h)

	go func() {
		err := c.transport.Start(ctx)
		if err != nil && c.logger != nil {
			c.logger.Logf("Client transport stopped: %v", err)
		}
	}()

	close(c.ready)
	return nil
}

// Ping sends a "ping" request to the server.
func (c *Client) Ping(ctx context.Context, pingID string) error {
	// Ensure transport is ready
	select {
	case <-c.ready:
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for transport: %w", ctx.Err())
	}

	// Create a numeric ID that matches what the client expects
	num, _ := strconv.ParseUint(pingID, 10, 64)
	id := types.RequestID{
		Num:      num,
		IsString: false,
	}

	msg := &types.Message{
		JSONRPC: types.JSONRPCVersion,
		ID:      &id,
		Method:  "ping",
	}

	if c.logger != nil {
		c.logger.Logf("Client sending ping with ID %v", id)
	}

	err := c.transport.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	return c.transport.Close()
}

func (c *Client) Done() <-chan struct{} {
	return c.transport.Done()
}
