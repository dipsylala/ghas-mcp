// Package server implements an MCP server over stdio JSON-RPC 2.0.
package server

// MCP protocol types used during session initialisation.

// InitializeParams is sent by the client in the initialize request.
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

// ClientCapabilities lists optional features the client supports.
type ClientCapabilities struct{}

// Implementation describes a software component (name + version).
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult is the server's response to an initialize request.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

// ServerCapabilities lists the optional MCP features this server supports.
type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability advertises that this server exposes tools.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}
