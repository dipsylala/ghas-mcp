// Package server contains the MCP server implementation.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	tools "github.com/dipsylala/ghas-mcp/internal/tool_registry"
	"github.com/dipsylala/ghas-mcp/internal/transport"
	"github.com/dipsylala/ghas-mcp/internal/types"
)

const mcpProtocolVersion = "2024-11-05"

var serverInstructions string

// SetInstructions stores the instructions string read from instructions.json.
func SetInstructions(data []byte) {
	var v struct {
		Instructions string `json:"instructions"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		log.Printf("warning: failed to parse instructions.json: %v", err)
		return
	}
	serverInstructions = v.Instructions
}

// MCPServer is the core MCP server. It implements transport.RequestHandler.
type MCPServer struct {
	initialized bool
	version     string
	toolManager *tools.ToolManager
}

// NewMCPServer creates and initialises an MCPServer.
func NewMCPServer(version string) (*MCPServer, error) {
	tm, err := tools.NewToolManager()
	if err != nil {
		return nil, fmt.Errorf("tool manager: %w", err)
	}
	return &MCPServer{version: version, toolManager: tm}, nil
}

// ServeStdio starts the stdio JSON-RPC loop; blocks until EOF or error.
func (s *MCPServer) ServeStdio() error {
	t := transport.NewStdioTransport(s)
	return t.Start()
}

// HandleRequest dispatches a JSON-RPC request to the correct method handler.
func (s *MCPServer) HandleRequest(req *types.JSONRPCRequest) *types.JSONRPCResponse {
	resp := &types.JSONRPCResponse{JSONRPC: "2.0"}
	if req.ID != nil {
		resp.ID = req.ID
	}

	switch req.Method {
	case "initialize":
		s.handleInitialize(req, resp)
	case "initialized":
		// Notification — no response needed.
		return nil
	case "tools/list":
		s.handleToolsList(req, resp)
	case "tools/call":
		s.handleToolsCall(req, resp)
	case "ping":
		resp.Result = map[string]interface{}{}
	default:
		resp.Error = &types.RPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
	return resp
}

// handleInitialize processes the MCP initialize handshake.
func (s *MCPServer) handleInitialize(_ *types.JSONRPCRequest, resp *types.JSONRPCResponse) {
	s.initialized = true
	resp.Result = InitializeResult{
		ProtocolVersion: mcpProtocolVersion,
		ServerInfo:      Implementation{Name: "ghas-mcp", Version: s.version},
		Capabilities:    ServerCapabilities{Tools: &ToolsCapability{}},
		Instructions:    serverInstructions,
	}
}

// handleToolsList returns all available tool definitions.
func (s *MCPServer) handleToolsList(_ *types.JSONRPCRequest, resp *types.JSONRPCResponse) {
	resp.Result = types.ListToolsResult{Tools: s.toolManager.GetAllMCPTools()}
}

// handleToolsCall executes a named tool.
func (s *MCPServer) handleToolsCall(req *types.JSONRPCRequest, resp *types.JSONRPCResponse) {
	var params types.CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		resp.Error = &types.RPCError{Code: -32600, Message: "invalid params: " + err.Error()}
		return
	}

	handler, ok := s.toolManager.GetToolHandler(params.Name)
	if !ok {
		resp.Error = &types.RPCError{Code: -32601, Message: fmt.Sprintf("unknown tool: %s", params.Name)}
		return
	}

	log.Printf("tool call: %s", params.Name)
	result, err := handler(context.Background(), params.Arguments)
	if err != nil {
		log.Printf("tool %s error: %v", params.Name, err)
		resp.Result = types.CallToolResult{
			IsError: true,
			Content: []types.Content{{Type: "text", Text: err.Error()}},
		}
		return
	}

	text, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		text = []byte(fmt.Sprintf("%v", result))
	}
	resp.Result = types.CallToolResult{
		Content: []types.Content{{Type: "text", Text: string(text)}},
	}
}
