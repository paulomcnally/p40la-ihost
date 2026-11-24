package server

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/audit"
)

// auditMiddleware registra solo tools/call, sin argumentos ni resultados.
func (s *Server) auditMiddleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			start := time.Now()
			result, err := next(ctx, method, req)

			entry := audit.Entry{
				DurationMS: time.Since(start).Milliseconds(),
				Result:     "ok",
			}
			if call, ok := req.(*mcp.CallToolRequest); ok {
				entry.Tool = call.Params.Name
			}
			if err != nil {
				entry.Result = "error"
			} else if toolResult, ok := result.(*mcp.CallToolResult); ok && toolResult.IsError {
				entry.Result = "tool_error"
			}
			if subject, ok := ctx.Value(subjectKey).(string); ok {
				entry.Subject = subject
			}
			if requestID, ok := ctx.Value(requestIDKey).(string); ok {
				entry.RequestID = requestID
			}
			if logErr := s.audit.Log(entry); logErr != nil {
				s.logger.Error("auditoría", "error", logErr)
			}
			return result, err
		}
	}
}
