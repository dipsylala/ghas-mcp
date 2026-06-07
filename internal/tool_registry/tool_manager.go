package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/dipsylala/ghas-mcp/internal/mcp_tools"

	// Import tool packages to trigger their init() auto-registration.
	_ "github.com/dipsylala/ghas-mcp/internal/mcp_tools"
	"github.com/dipsylala/ghas-mcp/internal/types"
)

// ToolManager wires together tool definitions, handler functions, and implementations.
type ToolManager struct {
	definitions *ToolRegistry
	handlers    *ToolHandlerRegistry
}

// NewToolManager loads definitions and initialises all registered tools.
func NewToolManager() (*ToolManager, error) {
	defs, err := LoadToolDefinitions()
	if err != nil {
		return nil, fmt.Errorf("load tool definitions: %w", err)
	}

	handlers := NewToolHandlerRegistry()
	tm := &ToolManager{definitions: defs, handlers: handlers}
	tm.loadAllTools()
	return tm, nil
}

// loadAllTools initialises every auto-registered tool and wires up its handlers.
func (tm *ToolManager) loadAllTools() {
	for _, reg := range mcp_tools.GetAllTools() {
		if err := reg.Impl.Initialize(); err != nil {
			log.Printf("tool %s: initialize error: %v", reg.Name, err)
			continue
		}
		if err := reg.Impl.RegisterHandlers(tm.handlers); err != nil {
			log.Printf("tool %s: register handlers error: %v", reg.Name, err)
			continue
		}
		log.Printf("tool loaded: %s", reg.Name)
	}
}

// GetAllMCPTools returns all tool definitions in MCP wire format.
func (tm *ToolManager) GetAllMCPTools() []types.Tool {
	return tm.definitions.GetAllMCPTools()
}

// GetToolHandler looks up a handler function by tool name.
func (tm *ToolManager) GetToolHandler(name string) (func(context.Context, map[string]interface{}) (interface{}, error), bool) {
	return tm.handlers.GetHandler(name)
}
