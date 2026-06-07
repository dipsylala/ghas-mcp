package tools

import (
	"context"
	"sync"

	"github.com/dipsylala/ghas-mcp/internal/mcp_tools"
)

// ToolHandlerRegistry maps tool names to their handler functions.
type ToolHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]mcp_tools.ToolHandler
}

// NewToolHandlerRegistry creates an empty ToolHandlerRegistry.
func NewToolHandlerRegistry() *ToolHandlerRegistry {
	return &ToolHandlerRegistry{
		handlers: make(map[string]mcp_tools.ToolHandler),
	}
}

// RegisterHandler registers a handler function for a named tool.
// Implements mcp_tools.HandlerRegistry so tool implementations can self-register.
func (r *ToolHandlerRegistry) RegisterHandler(name string, handler mcp_tools.ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = handler
}

// GetHandler retrieves a handler by tool name.
func (r *ToolHandlerRegistry) GetHandler(name string) (func(context.Context, map[string]interface{}) (interface{}, error), bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[name]
	return h, ok
}
