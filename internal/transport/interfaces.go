package transport

import "github.com/dipsylala/ghas-mcp/internal/types"

// RequestHandler processes a JSON-RPC request and returns a response.
type RequestHandler interface {
	HandleRequest(req *types.JSONRPCRequest) *types.JSONRPCResponse
}
