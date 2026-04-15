package main

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// tool params
type AddMemoryParams struct {
	Info string `json:"info" jsonschema:"A short, dense, and precise description of the information to remember."`
}

const WRITE_ATTEMPTS = 3

func GetAddMemoryHandler(storage *MemoryStorage, prompts *Tools) mcp.ToolHandlerFor[*AddMemoryParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *AddMemoryParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		if len(storage.memories) >= storage.maxMemories {
			return nil, nil, fmt.Errorf(
				"Memory limit reached: %d/%d (100%% full).\nTry using `%s` to remove stale memories or `%s` to update outdated ones.",
				storage.maxMemories,
				storage.maxMemories,
				prompts.Remove.Name,
				prompts.Update.Name,
			)
		} else {
			// retry logic with jitter
			var err error
			var rec *MemoryRecord
			for i := range WRITE_ATTEMPTS {
				if rec, err = storage.AddRecord(params.Info); err == nil {
					break
				}

				time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
			}

			if err != nil {
				return nil, nil, fmt.Errorf("Failed to remember after retries: %w", err)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Memory recorded with ID: '%s'", rec.ID)},
				},
			}, nil, nil
		}

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

		// keeping things organized
		sort.Slice(memories, func(i, j int) bool {
			return memories[i].LastUpdate < memories[j].LastUpdate
		})

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
type RemoveMemoryParams struct {
	MemID RecordID `json:"mem_id" jsonschema:"The ID of the memory record to delete."`
}

func GetRemoveMemoryHandler(storage *MemoryStorage) mcp.ToolHandlerFor[*RemoveMemoryParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *RemoveMemoryParams,
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
			memories_sorted = append(memories_sorted, record)
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
