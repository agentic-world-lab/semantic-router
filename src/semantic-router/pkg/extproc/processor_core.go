package extproc

import (
	"context"
	"errors"
	"io"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	ext_proc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/headers"
	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/observability/logging"
	"github.com/vllm-project/semantic-router/src/semantic-router/pkg/observability/metrics"
)

// Process implements the ext_proc calls
func (r *OpenAIRouter) Process(stream ext_proc.ExternalProcessor_ProcessServer) error {
	logging.Infof("Processing at stage [init]")

	// Initialize request context
	ctx := &RequestContext{
		Headers: make(map[string]string),
	}

	for {
		req, err := stream.Recv()
		if err != nil {
			// Mark streaming as aborted if it was a streaming response
			// This prevents caching incomplete responses
			if ctx.IsStreamingResponse && !ctx.StreamingComplete {
				ctx.StreamingAborted = true
				logging.Infof("Streaming response aborted before completion, will not cache")
			}

			// Handle EOF - this indicates the client has closed the stream gracefully
			if errors.Is(err, io.EOF) {
				logging.Infof("Stream ended gracefully")
				return nil
			}

			// Handle gRPC status-based cancellations/timeouts
			if s, ok := status.FromError(err); ok {
				switch s.Code() {
				case codes.Canceled:
					return nil
				case codes.DeadlineExceeded:
					logging.Infof("Stream deadline exceeded")
					metrics.RecordRequestError(ctx.RequestModel, "timeout")
					return nil
				}
			}

			// Handle context cancellation from the server-side context
			if errors.Is(err, context.Canceled) {
				logging.Infof("Stream canceled gracefully")
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) {
				logging.Infof("Stream deadline exceeded")
				metrics.RecordRequestError(ctx.RequestModel, "timeout")
				return nil
			}

			logging.Errorf("Error receiving request: %v", err)
			return err
		}

		switch v := req.Request.(type) {
		case *ext_proc.ProcessingRequest_RequestHeaders:
			response, err := r.handleRequestHeaders(v, ctx)
			if err != nil {
				logging.Errorf("handleRequestHeaders failed: %v", err)
				return err
			}
			// Defer response until request body is processed (Wait for Body Pattern)
			// This allows us to update the routing headers based on the body content (e.g. model selection)
			// before AgentGateway/Envoy makes a routing decision.
			ctx.DeferredHeaderResponse = response
			logging.Infof("[REQ_HEADERS] Deferring response until request body is processed to support content-based routing")

		case *ext_proc.ProcessingRequest_RequestBody:
			response, err := r.handleRequestBody(v, ctx)
			if err != nil {
				logging.Errorf("handleRequestBody failed: %v", err)
				return err
			}

			// [WAIT-FOR-BODY PATTERN]
			// If we deferred the headers response, we must ALSO defer/swallow intermediate body chunk responses.
			// Sending a ProcessingResponse_RequestBody while the AgentGateway is waiting for a
			// ProcessingResponse_RequestHeaders violates the protocol flow and causes 404s/routing errors.
			//
			// If EndOfStream is false, we just continue (swallowing the intermediate CONTINUE response).
			// If EndOfStream is true, we proceed to send the header response first (below).
			if ctx.DeferredHeaderResponse != nil && !v.RequestBody.EndOfStream {
				logging.Infof("[PROCESSOR_CORE] Deferring intermediate body chunk response (Wait for Body for routing decision)")
				continue
			}



			// If this is the end of the stream, we can now make the routing decision final
			// and send the deferred headers response before the body response.
			if v.RequestBody.EndOfStream && ctx.DeferredHeaderResponse != nil && !ctx.HeaderResponseSent {
				logging.Infof("[PROCESSOR_CORE] RequestBody EndOfStream: Updating and sending deferred headers")

				if ctx.VSRSelectedModel != "" {
					if respHeaders, ok := ctx.DeferredHeaderResponse.Response.(*ext_proc.ProcessingResponse_RequestHeaders); ok {
						if respHeaders.RequestHeaders.Response == nil {
							respHeaders.RequestHeaders.Response = &ext_proc.CommonResponse{}
						}
						
						if respHeaders.RequestHeaders.Response.HeaderMutation == nil {
							respHeaders.RequestHeaders.Response.HeaderMutation = &ext_proc.HeaderMutation{}
						}

						// Replace the VSRSelectedModel header
						var newHeaders []*core.HeaderValueOption
						for _, h := range respHeaders.RequestHeaders.Response.HeaderMutation.SetHeaders {
							if h.Header != nil && h.Header.Key != headers.VSRSelectedModel {
								newHeaders = append(newHeaders, h)
							}
						}
						newHeaders = append(newHeaders, &core.HeaderValueOption{
							Header: &core.HeaderValue{
								Key:      headers.VSRSelectedModel,
								RawValue: []byte(ctx.VSRSelectedModel),
							},
						})
						respHeaders.RequestHeaders.Response.HeaderMutation.SetHeaders = newHeaders
						logging.Infof("[PROCESSOR_CORE] Updated deferred header %s to %s", headers.VSRSelectedModel, ctx.VSRSelectedModel)
					}
				}

				if err := sendResponse(stream, ctx.DeferredHeaderResponse, "deferred request header"); err != nil {
					logging.Errorf("sendResponse for deferred headers failed: %v", err)
					return err
				}
				ctx.HeaderResponseSent = true
				ctx.DeferredHeaderResponse = nil
			}
			logging.Infof("Modified request body: %+v", response)
			logging.Infof("[PROCESSOR_CORE] ========== About to send request body response ==========")
			logging.Infof("[PROCESSOR_CORE] Response type: %T", response.Response)
			if rb, ok := response.Response.(*ext_proc.ProcessingResponse_RequestBody); ok && rb != nil && rb.RequestBody != nil && rb.RequestBody.Response != nil {
				if hm := rb.RequestBody.Response.HeaderMutation; hm != nil && len(hm.SetHeaders) > 0 {
					logging.Infof("[PROCESSOR_CORE] Headers to be set: %d", len(hm.SetHeaders))
					for _, h := range hm.SetHeaders {
						if h.Header != nil {
							logging.Infof("[PROCESSOR_CORE]   Header: %s = %s", h.Header.Key, h.Header.Value)
						}
					}
				}
			}
			if err := sendResponse(stream, response, "request body"); err != nil {
				logging.Errorf("sendResponse for body failed: %v", err)
				return err
			}
			logging.Infof("[PROCESSOR_CORE] Response sent successfully")

		case *ext_proc.ProcessingRequest_ResponseHeaders:
			response, err := r.handleResponseHeaders(v, ctx)
			if err != nil {
				return err
			}
			if err := sendResponse(stream, response, "response header"); err != nil {
				return err
			}

		case *ext_proc.ProcessingRequest_ResponseBody:
			response, err := r.handleResponseBody(v, ctx)
			if err != nil {
				return err
			}
			if err := sendResponse(stream, response, "response body"); err != nil {
				return err
			}

		default:
			logging.Warnf("Unknown request type: %v", v)

			// For unknown message types, create a body response with CONTINUE status
			response := &ext_proc.ProcessingResponse{
				Response: &ext_proc.ProcessingResponse_RequestBody{
					RequestBody: &ext_proc.BodyResponse{
						Response: &ext_proc.CommonResponse{
							Status: ext_proc.CommonResponse_CONTINUE,
						},
					},
				},
			}

			if err := sendResponse(stream, response, "unknown"); err != nil {
				return err
			}
		}
	}
}
