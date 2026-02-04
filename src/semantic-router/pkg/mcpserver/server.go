package mcpserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/config"
	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/observability/logging"
)

// Init initializes and starts the MCP server
func Init(configPath string, port int) error {
	// Get the global configuration
	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("configuration not initialized")
	}

	logging.Infof("Initializing MCP server on port %d", port)

	// Create HTTP server with routes
	mux := setupRoutes()
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logging.Infof("MCP server listening on port %d", port)
	return server.ListenAndServe()
}

// setupRoutes configures all MCP server routes
func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", handleHealth)

	// Main MCP JSON-RPC endpoint
	mux.HandleFunc("POST /mcp", handleMCP)

	// Convenience endpoints for specific MCP methods
	mux.HandleFunc("POST /mcp/initialize", handleMCP)
	mux.HandleFunc("POST /mcp/tools/list", handleMCP)
	mux.HandleFunc("POST /mcp/tools/call", handleMCP)

	return mux
}
