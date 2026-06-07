package mcp_tools

import "sync"

var (
	toolRegistry = make(map[string]func() ToolImplementation)
	registryMu   sync.RWMutex
)

// RegisterTool registers a tool constructor by name (called from init() in tool files).
func RegisterTool(name string, constructor func() ToolImplementation) {
	registryMu.Lock()
	defer registryMu.Unlock()
	toolRegistry[name] = constructor
}

// RegisterMCPTool is a convenience wrapper for stateless handler-only tools.
func RegisterMCPTool(name string, handler ToolHandler) {
	RegisterTool(name, func() ToolImplementation {
		return NewSimpleTool(name, handler)
	})
}

// RegisteredTool pairs a name with a constructed ToolImplementation.
type RegisteredTool struct {
	Name string
	Impl ToolImplementation
}

// GetAllTools returns an instance of every registered tool.
func GetAllTools() []RegisteredTool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	tools := make([]RegisteredTool, 0, len(toolRegistry))
	for name, ctor := range toolRegistry {
		tools = append(tools, RegisteredTool{Name: name, Impl: ctor()})
	}
	return tools
}
