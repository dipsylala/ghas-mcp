// Package mcp_tools provides the tool registration infrastructure and shared utilities.
package mcp_tools

import (
	"context"
	"log"
)

// ToolHandler is a function that handles a tool call.
type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// HandlerRegistry stores named tool handler functions.
type HandlerRegistry interface {
	RegisterHandler(name string, handler ToolHandler)
}

// ToolImplementation is implemented by every MCP tool.
type ToolImplementation interface {
	Initialize() error
	RegisterHandlers(registry HandlerRegistry) error
	Shutdown() error
}

// SimpleTool is a stateless ToolImplementation backed by a single handler function.
type SimpleTool struct {
	name    string
	handler ToolHandler
}

// NewSimpleTool wraps a handler function as a ToolImplementation.
func NewSimpleTool(name string, handler ToolHandler) ToolImplementation {
	return &SimpleTool{name: name, handler: handler}
}

func (t *SimpleTool) Initialize() error { return nil }

func (t *SimpleTool) RegisterHandlers(registry HandlerRegistry) error {
	registry.RegisterHandler(t.name, t.handler)
	return nil
}

func (t *SimpleTool) Shutdown() error {
	log.Printf("tool %s shutdown", t.name)
	return nil
}
