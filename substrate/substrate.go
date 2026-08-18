// Package substrate is a minimal JSON-RPC-over-WebSocket client for
// Substrate nodes, covering the calls needed to check node health.
package substrate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/websocket"
)

// Client is a JSON-RPC connection to a Substrate node.
type Client struct {
	conn   *websocket.Conn
	nextID int
}

// Dial connects to the Substrate node at url (ws:// or wss://).
func Dial(ctx context.Context, url string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	conn.SetReadLimit(16 << 20)
	return &Client{conn: conn, nextID: 1}, nil
}

func (c *Client) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// Call performs a single JSON-RPC request and decodes its result.
func (c *Client) Call(ctx context.Context, method string, params []any, result any) error {
	id := c.nextID
	c.nextID++
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": id, "method": method, "params": params,
	})
	if err != nil {
		return err
	}
	if err := c.conn.Write(ctx, websocket.MessageText, payload); err != nil {
		return fmt.Errorf("%s: write: %w", method, err)
	}

	// Read until the response with our id arrives, skipping anything else
	// the node pushes (e.g. subscription notifications).
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("%s: read: %w", method, err)
		}
		var response struct {
			ID    *int `json:"id"`
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(data, &response); err != nil {
			return fmt.Errorf("%s: decode: %w", method, err)
		}
		if response.ID == nil || *response.ID != id {
			continue
		}
		if response.Error != nil {
			return fmt.Errorf("%s returned RPC error %d: %s", method, response.Error.Code, response.Error.Message)
		}
		if result == nil {
			return nil
		}
		return json.Unmarshal(response.Result, result)
	}
}

// Methods returns the set of RPC methods the node exposes, via rpc_methods.
func (c *Client) Methods(ctx context.Context) (map[string]bool, error) {
	var response struct {
		Methods []string `json:"methods"`
	}
	if err := c.Call(ctx, "rpc_methods", []any{}, &response); err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(response.Methods))
	for _, method := range response.Methods {
		set[method] = true
	}
	return set, nil
}
