package mcpserver

// MCP Protocol Version 2024-11-05
const ProtocolVersion = "2024-11-05"

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
const (
	ParseError     = -32700 // Parse error
	InvalidRequest = -32600 // Invalid request
	MethodNotFound = -32601 // Method not found
	InvalidParams  = -32602 // Invalid params
	InternalError  = -32603 // Internal error
)

// InitializeRequest represents an MCP initialize request
type InitializeRequest struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	ClientInfo      map[string]interface{} `json:"clientInfo,omitempty"`
	Capabilities    map[string]interface{} `json:"capabilities,omitempty"`
}

// InitializeResponse represents an MCP initialize response
type InitializeResponse struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

// ServerInfo contains information about the MCP server
type ServerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// ToolListRequest represents a request to list available tools
type ToolListRequest struct {
	// No parameters for list tools
}

// ToolListResponse represents a response containing available tools
type ToolListResponse struct {
	Tools []Tool `json:"tools"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// CallToolRequest represents a request to call a tool
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResponse represents a response from calling a tool
type CallToolResponse struct {
	Content []TextContent          `json:"content"`
	IsError bool                   `json:"isError,omitempty"`
	Meta    map[string]interface{} `json:"_meta,omitempty"`
}

// TextContent represents text content in an MCP response
type TextContent struct {
	Type string `json:"type"` // Always "text" for text content
	Text string `json:"text"`
}

// PingRequest represents a ping request
type PingRequest struct {
	// No parameters for ping
}

// PingResponse represents a ping response
type PingResponse struct {
	// Empty response for ping
}
