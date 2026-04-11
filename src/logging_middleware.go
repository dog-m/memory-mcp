package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func createLoggingMiddleware() mcp.Middleware {
	// source of inspiration - the official MCP Go SDK examples
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			sessionID := req.GetSession().ID()

			log.Printf("[~] Method: %s | Session: %s", method, sessionID)

			// call the actual handler
			result, err := next(ctx, method, req)

			if err != nil {
				log.Printf("[x] Method: %s | Session: %s | ERROR: %v", method, sessionID, err)
			} else {
				log.Printf("[i] Method: %s | Session: %s | OK", method, sessionID)
			}

			// propagating both returned value and error
			return result, err
		}
	}
}
