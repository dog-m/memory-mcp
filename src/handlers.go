package main

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// tool params
type RememberParams struct {
	Info string `json:"info" jsonschema:"A short, dense, and precise description of the information to remember."`
}

const WRITE_ATTEMPTS = 3

func GetRememberMemoryHandler(storage *MemoryStorage) mcp.ToolHandlerFor[*RememberParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *RememberParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		var result string
		if len(storage.memories) >= storage.maxMemories {
			result = fmt.Sprintf("Memory limit reached: %d/%d", storage.maxMemories, storage.maxMemories)
		} else {
			// retry logic with jitter
			var id RecordID
			var err error
			for i := range WRITE_ATTEMPTS {
				if id, err = storage.NewRecord(params.Info); err == nil {
					break
				}

				time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			}

			if err == nil {
				result = fmt.Sprintf("Memory recorded with ID: '%s'", id)
			} else {
				return nil, nil, fmt.Errorf("Failed to remember after retries: %w", err)
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	}
}

// tool params
type EmptyParams struct {
}

func GetListMemoriesHandler(storage *MemoryStorage) mcp.ToolHandlerFor[*EmptyParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		_ *EmptyParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		memories := storage.GetAllRecords()

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
	MemID RecordID `json:"mem_id" jsonschema:"The ID of the memory record to delete."`
}

func GetForgetMemoryHandler(storage *MemoryStorage) mcp.ToolHandlerFor[*ForgetParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *ForgetParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		err := storage.DeleteRecord(params.MemID)
		if err != nil {
			return nil, nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Memory '%s' forgotten", params.MemID)},
			},
		}, nil, nil
	}
}

// tool params
type UpdateMemoryParams struct {
	MemID   RecordID `json:"mem_id" jsonschema:"The ID of the memory record to update."`
	NewInfo string   `json:"new_info" jsonschema:"The new information to store in the record."`
}

func GetUpdateMemoryHandler(storage *MemoryStorage) mcp.ToolHandlerFor[*UpdateMemoryParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *UpdateMemoryParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		err := storage.UpdateRecord(params.MemID, params.NewInfo)
		if err != nil {
			return nil, nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Memory '%s' updated", params.MemID)},
			},
		}, nil, nil
	}
}

type ChatSessionStartupResponse struct {
	CurrentTime       string `json:"current_time"`
	MemoryUtilization string `json:"memory_utilization"`
	EditSummary       struct {
		Count   int             `json:"count"`
		Details []*MemoryRecord `json:"details"`
	} `json:"edit_summary"`
}

func GetChatSessionStartupHandler(storage *MemoryStorage, maxRecentEdits int) mcp.ToolHandlerFor[*EmptyParams, *ChatSessionStartupResponse] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		_ *EmptyParams,
	) (
		*mcp.CallToolResult,
		*ChatSessionStartupResponse,
		error,
	) {
		// order
		mem_count := len(storage.memories)
		memories_sorted := make([]*MemoryRecord, 0, mem_count)
		for _, record := range storage.memories {
			memories_sorted = append(memories_sorted, &record)
		}
		slices.SortStableFunc(memories_sorted, func(a, b *MemoryRecord) int {
			return strings.Compare(b.LastUpdate, a.LastUpdate)
		})

		// truncate
		if mem_count > maxRecentEdits {
			memories_sorted = memories_sorted[:maxRecentEdits]
		}

		// assemble response
		result := &ChatSessionStartupResponse{}
		result.CurrentTime = fmt.Sprintf("%s (RFC1123)", time.Now().Format(time.RFC1123))
		result.MemoryUtilization = fmt.Sprintf(
			"%d/%d records used (%0.1f %%)",
			mem_count,
			storage.maxMemories,
			100.0*float32(mem_count)/float32(storage.maxMemories),
		)
		result.EditSummary.Count = len(memories_sorted)
		result.EditSummary.Details = memories_sorted

		return nil, result, nil
	}
}
