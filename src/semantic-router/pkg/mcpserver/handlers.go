package mcpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/observability/logging"
)

// handleMCP is the main JSON-RPC 2.0 endpoint handler
func handleMCP(w http.ResponseWriter, r *http.Request) {
	// Parse JSON-RPC request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, nil, ParseError, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCError(w, nil, ParseError, "Invalid JSON")
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		writeJSONRPCError(w, req.ID, InvalidRequest, "Invalid JSON-RPC version")
		return
	}

	// Route based on method
	var result interface{}
	switch req.Method {
	case "initialize":
		result, err = handleInitialize(req.Params)
	case "ping":
		result, err = handlePing(req.Params)
	case "tools/list":
		result, err = handleListTools(req.Params)
	case "tools/call":
		result, err = handleCallTool(req.Params)
	default:
		writeJSONRPCError(w, req.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
		return
	}

	if err != nil {
		logging.Errorf("Error handling method %s: %v", req.Method, err)
		writeJSONRPCError(w, req.ID, InternalError, err.Error())
		return
	}

	// Write success response
	writeJSONRPCResponse(w, req.ID, result)
}

// handleInitialize handles the MCP initialize request
func handleInitialize(params interface{}) (interface{}, error) {
	logging.Infof("MCP initialize request")

	// Return initialize response
	return InitializeResponse{
		ProtocolVersion: ProtocolVersion,
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		ServerInfo: ServerInfo{
			Name:        "semantic-router-mcp",
			Version:     "1.0.0",
			Description: "MCP server for vLLM Semantic Router - intent classification, PII detection, and intelligent routing",
		},
	}, nil
}

// handlePing handles ping requests
func handlePing(params interface{}) (interface{}, error) {
	logging.Debugf("MCP ping request")
	return PingResponse{}, nil
}

// handleListTools handles tools/list requests
func handleListTools(params interface{}) (interface{}, error) {
	logging.Infof("MCP tools/list request")

	tools := getToolDefinitions()
	return ToolListResponse{
		Tools: tools,
	}, nil
}

// handleCallTool handles tools/call requests
func handleCallTool(params interface{}) (interface{}, error) {
	// Parse params as CallToolRequest
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	var callReq CallToolRequest
	if err := json.Unmarshal(paramsJSON, &callReq); err != nil {
		return nil, fmt.Errorf("invalid tool call parameters: %w", err)
	}

	// Validate tool name
	if callReq.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	// Call the tool
	result, err := handleToolCall(callReq.Name, callReq.Arguments)
	if err != nil {
		logging.Errorf("Tool call error for %s: %v", callReq.Name, err)
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: fmt.Sprintf(`{"error": "%s", "status": "tool_call_failed"}`, err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	return result, nil
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "healthy", "service": "mcp-server"}`))
}

// writeJSONRPCResponse writes a JSON-RPC success response
func writeJSONRPCResponse(w http.ResponseWriter, id interface{}, result interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Errorf("Failed to encode JSON-RPC response: %v", err)
	}
}

// writeJSONRPCError writes a JSON-RPC error response
func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	// Per JSON-RPC 2.0 spec, always return HTTP 200 even for errors
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Errorf("Failed to encode JSON-RPC error response: %v", err)
	}
}
