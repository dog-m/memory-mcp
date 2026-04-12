package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// tool params
type RememberParams struct {
	Info string `json:"info" jsonschema:"A short, dense, and precise description of the information to remember."`
}

const WRITE_ATTEMPTS = 3

func GetRememberMemoryHandler(memoryStorage *MemoryStorage) mcp.ToolHandlerFor[*RememberParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *RememberParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		// retry logic with jitter
		var id int64
		var err error
		for i := range WRITE_ATTEMPTS {
			id, err = memoryStorage.NewRecord(params.Info)
			if err == nil {
				break
			}
			if err.Error() == "memory limit reached" {
				return nil, nil, err
			}

			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
		}

		if err != nil {
			return nil, nil, fmt.Errorf("failed to remember after retries: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Memory recorded with ID: %d", id)},
			},
		}, nil, nil
	}
}

// tool params
type EmptyParams struct {
}

func GetListMemoriesHandler(memoryStorage *MemoryStorage) mcp.ToolHandlerFor[*EmptyParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		_ *EmptyParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		memories := memoryStorage.GetAllRecords()
		jsonData, err := json.MarshalIndent(memories, "", "  ")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal memories: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(jsonData)},
			},
		}, nil, nil
	}
}

// tool params
type ForgetParams struct {
	MemID int64 `json:"mem_id" jsonschema:"The ID of the memory record to delete."`
}

func GetForgetMemoryHandler(memoryStorage *MemoryStorage) mcp.ToolHandlerFor[*ForgetParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *ForgetParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		err := memoryStorage.DeleteRecord(params.MemID)
		if err != nil {
			return nil, nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Memory %d forgotten", params.MemID)},
			},
		}, nil, nil
	}
}

// tool params
type UpdateMemoryParams struct {
	MemID   int64  `json:"mem_id" jsonschema:"The ID of the memory record to update."`
	NewInfo string `json:"new_info" jsonschema:"The new information to store in the record."`
}

func GetUpdateMemoryHandler(memoryStorage *MemoryStorage) mcp.ToolHandlerFor[*UpdateMemoryParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *UpdateMemoryParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		err := memoryStorage.UpdateRecord(params.MemID, params.NewInfo)
		if err != nil {
			return nil, nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Memory '%d' updated", params.MemID)},
			},
		}, nil, nil
	}
}

func GetChatSessionStartupHandler() mcp.ToolHandlerFor[*EmptyParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		_ *EmptyParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				// FIXME: not implemented!
				&mcp.TextContent{Text: "Placeholder data. Water is wet."},
			},
		}, nil, nil
	}
}
