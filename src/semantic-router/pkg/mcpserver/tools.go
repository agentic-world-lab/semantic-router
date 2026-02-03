package mcpserver

import (
	"encoding/json"
	"fmt"

	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/config"
	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/observability/logging"
	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/observability/metrics"
	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/services"
)

// getToolDefinitions returns all available MCP tools
func getToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "classify_intent",
			Description: "Classify the intent/category of a user query using the semantic router's classification service",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The user query to classify",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "detect_pii",
			Description: "Detect personally identifiable information (PII) in a query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The text to analyze for PII",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "detect_security",
			Description: "Detect security threats and jailbreak attempts in a query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The query to analyze for security threats",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "route_query",
			Description: "Get model routing decision for a query based on routing signals",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The query to route",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_models",
			Description: "List all available models configured in the semantic router",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "check_semantic_cache",
			Description: "Check if a query exists in the semantic cache",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The query to check in the cache",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_metrics",
			Description: "Get routing and classification metrics from the semantic router",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// handleToolCall routes tool calls to the appropriate handler
func handleToolCall(name string, arguments map[string]interface{}) (*CallToolResponse, error) {
	logging.Infof("MCP tool call: %s", name)

	switch name {
	case "classify_intent":
		return handleClassifyIntent(arguments)
	case "detect_pii":
		return handleDetectPII(arguments)
	case "detect_security":
		return handleDetectSecurity(arguments)
	case "route_query":
		return handleRouteQuery(arguments)
	case "get_models":
		return handleGetModels(arguments)
	case "check_semantic_cache":
		return handleCheckSemanticCache(arguments)
	case "get_metrics":
		return handleGetMetrics(arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// handleClassifyIntent implements the classify_intent tool
func handleClassifyIntent(arguments map[string]interface{}) (*CallToolResponse, error) {
	query, ok := arguments["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter must be a string")
	}

	// Get the global classification service
	classificationSvc := services.GetGlobalClassificationService()
	if classificationSvc == nil {
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: `{"error": "Classification service not available", "status": "service_unavailable"}`,
				},
			},
			IsError: true,
		}, nil
	}

	// Classify the intent
	result, err := classificationSvc.ClassifyIntent(services.IntentRequest{Text: query})
	if err != nil {
		logging.Errorf("Classification error: %v", err)
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: fmt.Sprintf(`{"error": "%s", "status": "classification_failed"}`, err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(map[string]interface{}{
		"category":            result.Classification.Category,
		"confidence":          result.Classification.Confidence,
		"processing_time_ms":  result.Classification.ProcessingTimeMs,
		"recommended_model":   result.RecommendedModel,
		"routing_decision":    result.RoutingDecision,
		"status":              "success",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &CallToolResponse{
		Content: []TextContent{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// handleDetectPII implements the detect_pii tool
func handleDetectPII(arguments map[string]interface{}) (*CallToolResponse, error) {
	query, ok := arguments["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter must be a string")
	}

	// Get the global classification service
	classificationSvc := services.GetGlobalClassificationService()
	if classificationSvc == nil {
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: `{"error": "Classification service not available", "status": "service_unavailable"}`,
				},
			},
			IsError: true,
		}, nil
	}

	// Detect PII
	result, err := classificationSvc.DetectPII(services.PIIRequest{Text: query})
	if err != nil {
		logging.Errorf("PII detection error: %v", err)
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: fmt.Sprintf(`{"error": "%s", "status": "pii_detection_failed"}`, err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(map[string]interface{}{
		"has_pii":                result.HasPII,
		"entities":               result.Entities,
		"masked_text":            result.MaskedText,
		"security_recommendation": result.SecurityRecommendation,
		"processing_time_ms":     result.ProcessingTimeMs,
		"status":                 "success",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &CallToolResponse{
		Content: []TextContent{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// handleDetectSecurity implements the detect_security tool
func handleDetectSecurity(arguments map[string]interface{}) (*CallToolResponse, error) {
	query, ok := arguments["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter must be a string")
	}

	// Get the global classification service
	classificationSvc := services.GetGlobalClassificationService()
	if classificationSvc == nil {
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: `{"error": "Classification service not available", "status": "service_unavailable"}`,
				},
			},
			IsError: true,
		}, nil
	}

	// Detect security threats
	result, err := classificationSvc.CheckSecurity(services.SecurityRequest{Text: query})
	if err != nil {
		logging.Errorf("Security detection error: %v", err)
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: fmt.Sprintf(`{"error": "%s", "status": "security_detection_failed"}`, err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(map[string]interface{}{
		"is_jailbreak":       result.IsJailbreak,
		"risk_score":         result.RiskScore,
		"detection_types":    result.DetectionTypes,
		"confidence":         result.Confidence,
		"recommendation":     result.Recommendation,
		"reasoning":          result.Reasoning,
		"patterns_detected":  result.PatternsDetected,
		"processing_time_ms": result.ProcessingTimeMs,
		"status":             "success",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &CallToolResponse{
		Content: []TextContent{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// handleRouteQuery implements the route_query tool
func handleRouteQuery(arguments map[string]interface{}) (*CallToolResponse, error) {
	query, ok := arguments["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter must be a string")
	}

	// Get the global classification service to access router
	classificationSvc := services.GetGlobalClassificationService()
	if classificationSvc == nil {
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: `{"error": "Classification service not available", "status": "service_unavailable"}`,
				},
			},
			IsError: true,
		}, nil
	}

	// For routing, we'll use the classification result as a proxy
	// The actual routing logic is in the extproc router, but we can provide intent-based routing signals
	result, err := classificationSvc.ClassifyIntent(services.IntentRequest{Text: query})
	if err != nil {
		logging.Errorf("Routing error: %v", err)
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: fmt.Sprintf(`{"error": "%s", "status": "routing_failed"}`, err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	// Get config for model information
	cfg := config.Get()
	selectedModel := cfg.DefaultModel
	if selectedModel == "" {
		selectedModel = "default"
	}
	
	// Use recommended model from classification if available
	if result.RecommendedModel != "" {
		selectedModel = result.RecommendedModel
	}

	// Convert result to JSON with routing information
	resultJSON, err := json.Marshal(map[string]interface{}{
		"selected_model":      selectedModel,
		"category":            result.Classification.Category,
		"confidence":          result.Classification.Confidence,
		"routing_decision":    result.RoutingDecision,
		"processing_time_ms":  result.Classification.ProcessingTimeMs,
		"status":              "success",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &CallToolResponse{
		Content: []TextContent{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// handleGetModels implements the get_models tool
func handleGetModels(arguments map[string]interface{}) (*CallToolResponse, error) {
	// Get configuration
	cfg := config.Get()
	if cfg == nil {
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: `{"error": "Configuration not available", "status": "config_unavailable"}`,
				},
			},
			IsError: true,
		}, nil
	}

	// Build models list from ModelConfig
	models := make([]map[string]interface{}, 0, len(cfg.ModelConfig))
	for modelName, modelParams := range cfg.ModelConfig {
		modelInfo := map[string]interface{}{
			"name": modelName,
		}
		// Add preferred endpoints if available
		if len(modelParams.PreferredEndpoints) > 0 {
			modelInfo["endpoints"] = modelParams.PreferredEndpoints
		}
		// Add reasoning family if available
		if modelParams.ReasoningFamily != "" {
			modelInfo["reasoning_family"] = modelParams.ReasoningFamily
		}
		models = append(models, modelInfo)
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(map[string]interface{}{
		"models":        models,
		"default_model": cfg.DefaultModel,
		"count":         len(models),
		"status":        "success",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &CallToolResponse{
		Content: []TextContent{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// handleCheckSemanticCache implements the check_semantic_cache tool
func handleCheckSemanticCache(arguments map[string]interface{}) (*CallToolResponse, error) {
	_, ok := arguments["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter must be a string")
	}

	// Get configuration to check if cache is enabled
	cfg := config.Get()
	if cfg == nil || !cfg.SemanticCache.Enabled {
		return &CallToolResponse{
			Content: []TextContent{
				{
					Type: "text",
					Text: `{"cache_enabled": false, "status": "cache_disabled"}`,
				},
			},
		}, nil
	}

	// Note: Actual cache lookup would require access to the cache implementation
	// For now, we return cache configuration information
	resultJSON, err := json.Marshal(map[string]interface{}{
		"cache_enabled":        true,
		"backend_type":         cfg.SemanticCache.BackendType,
		"embedding_model":      cfg.SemanticCache.EmbeddingModel,
		"similarity_threshold": cfg.SemanticCache.SimilarityThreshold,
		"note":                 "Cache lookup not implemented in MCP server (requires cache instance access)",
		"status":               "success",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &CallToolResponse{
		Content: []TextContent{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// handleGetMetrics implements the get_metrics tool
func handleGetMetrics(arguments map[string]interface{}) (*CallToolResponse, error) {
	// Collect metrics from the metrics package
	metricsData := map[string]interface{}{
		"status": "success",
	}

	// Get windowed metrics if available
	windowedMgr := metrics.GetWindowedMetricsManager()
	if windowedMgr != nil {
		metricsData["windowed_metrics_enabled"] = true
	} else {
		metricsData["windowed_metrics_enabled"] = false
	}

	// Add configuration info
	cfg := config.Get()
	if cfg != nil && cfg.Observability.Metrics.Enabled != nil {
		metricsData["metrics_enabled"] = *cfg.Observability.Metrics.Enabled
	}

	// Convert result to JSON
	resultJSON, err := json.Marshal(metricsData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &CallToolResponse{
		Content: []TextContent{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}
