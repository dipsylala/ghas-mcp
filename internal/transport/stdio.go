// Package transport provides stdio-based JSON-RPC 2.0 transport for the MCP server.
package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/dipsylala/ghas-mcp/internal/types"
)

// StdioTransport handles JSON-RPC over stdin/stdout.
type StdioTransport struct {
	handler RequestHandler
	reader  *bufio.Reader
	writer  io.Writer
	mu      sync.Mutex
}

// NewStdioTransport creates a StdioTransport that dispatches to handler.
func NewStdioTransport(handler RequestHandler) *StdioTransport {
	return &StdioTransport{
		handler: handler,
		reader:  bufio.NewReader(os.Stdin),
		writer:  bufio.NewWriter(os.Stdout),
	}
}

// Start reads newline-delimited JSON from stdin until EOF or error.
func (t *StdioTransport) Start() error {
	for {
		line, err := t.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("stdio: client disconnected (EOF)")
				return nil
			}
			return fmt.Errorf("stdio read error: %w", err)
		}

		if len(line) <= 1 {
			continue
		}

		log.Printf("stdio: recv: %s", truncate(string(line), 120))

		var req types.JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("stdio: parse error: %v", err)
			_ = t.sendError(nil, -32700, "Parse error")
			continue
		}

		// MCP clients are sequential: process synchronously so the response is
		// guaranteed to be sent before we read the next message.
		t.dispatch(&req)
	}
}

func (t *StdioTransport) dispatch(req *types.JSONRPCRequest) {
	resp := t.handler.HandleRequest(req)
	if resp != nil {
		if err := t.send(resp); err != nil {
			log.Printf("stdio: send error: %v", err)
		}
	}
}

func (t *StdioTransport) send(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	_, err = fmt.Fprintf(t.writer, "%s\n", data)
	if f, ok := t.writer.(*bufio.Writer); ok {
		_ = f.Flush()
	}
	return err
}

func (t *StdioTransport) sendError(id interface{}, code int, message string) error {
	return t.send(&types.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &types.RPCError{Code: code, Message: message},
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
