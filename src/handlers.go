package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// tool params
type AddMemoryParams struct {
	Info string `json:"info" jsonschema:"A short, dense, and precise description of the information to remember."`
}

const WRITE_ATTEMPTS = 3
const ENTRY_TEXT_MAX_LENGTH = 750

func checkEntryTextSize(text string) error {
	if size := utf8.RuneCountInString(text); size > ENTRY_TEXT_MAX_LENGTH {
		return fmt.Errorf(
			"Memory entry too long (%d > %d). Keep things concise. Use multiple entries only if absolutely necessary.",
			size,
			ENTRY_TEXT_MAX_LENGTH,
		)
	}
	return nil
}

func GetAddMemoryHandler(storage *MemoryStorage, prompts *ToolDefinitions) mcp.ToolHandlerFor[*AddMemoryParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *AddMemoryParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		if err := checkEntryTextSize(params.Info); err != nil {
			return nil, nil, err
		}

		if len(storage.memories) >= storage.maxMemories {
			return nil, nil, fmt.Errorf(
				"Memory limit reached: %d/%d (full).\nTry using `%s` to remove stale memories or `%s` to update outdated ones.",
				storage.maxMemories,
				storage.maxMemories,
				prompts.Remove.Name,
				prompts.Update.Name,
			)
		} else {
			// retry logic with random jitter
			var err error
			var rec *MemoryRecord
			for range WRITE_ATTEMPTS {
				if rec, err = storage.AddRecord(params.Info); err == nil {
					break
				}

				time.Sleep(time.Duration(rand.Int63n(100)+1) * time.Millisecond)
			}
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to add a new memory entry: %v", err)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Memory recorded successfully (ID: '%s').", rec.ID)},
				},
			}, nil, nil
		}

	}
}

// tool params
type ListMemoriesParams struct {
	Include *string `json:"include,omitempty" jsonschema:"A pattern-based filter to find specific memories. Supports wildcards (*) and alternatives in curly braces {option1,option2}. Use this to narrow down results by structure or keywords. Example: '{I,he,she} * home at ??:??'"`
	Exclude *string `json:"exclude,omitempty" jsonschema:"Use this to filter out noise. If you are looking for food but want to avoid mentions of 'delivery', set this to 'delivery'."`
}

func GetListMemoriesHandler(storage *MemoryStorage) mcp.ToolHandlerFor[*ListMemoriesParams, any] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		params *ListMemoriesParams,
	) (
		*mcp.CallToolResult,
		any,
		error,
	) {
		// build filters
		filters, err := GetSearchMatchers(params.Include, params.Exclude)
		if err != nil {
			return nil, nil, err
		}

		// retrieve
		memories := storage.GetAllRecords(func(rec *MemoryRecord) bool {
			return filters.allowFilter(rec.Text) && !filters.denyFilter(rec.Text)
		})

		// keeping things organized
		slices.SortStableFunc(memories, func(a, b *MemoryRecord) int {
			return strings.Compare(a.LastUpdate, b.LastUpdate)
		})

		// print things neatly (unsure if this helps or not)
		if jsonData, err := json.MarshalIndent(memories, "", "\t"); err != nil {
			return nil, nil, fmt.Errorf("Failed to serialize memories: %v", err)
		} else {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(jsonData)},
				},
			}, nil, nil
		}
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
		if err := storage.DeleteRecord(params.MemID); err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Memory '%s' forgotten.", params.MemID)},
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
		if err := checkEntryTextSize(params.NewInfo); err != nil {
			return nil, nil, err
		}

		if err := storage.UpdateRecord(params.MemID, params.NewInfo); err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Memory '%s' updated successfully.", params.MemID)},
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

// tool params
type EmptyParams struct {
}

func GetChatSessionStartupHandler(storage *MemoryStorage, maxRecentEdits int, maxSummaryLen int) mcp.ToolHandlerFor[*EmptyParams, *ChatSessionStartupResponse] {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		_ *EmptyParams,
	) (
		*mcp.CallToolResult,
		*ChatSessionStartupResponse,
		error,
	) {
		mem_count := len(storage.memories)

		// retrieve and order
		memories_sorted := make([]*MemoryRecord, 0, mem_count)
		for _, rec := range storage.memories {
			memories_sorted = append(memories_sorted, rec)
		}
		slices.SortStableFunc(memories_sorted, func(a, b *MemoryRecord) int {
			return strings.Compare(a.LastUpdate, b.LastUpdate)
		})

		// trim and compact long entries without making changes to the stored values
		if mem_count > maxRecentEdits {
			memories_sorted = memories_sorted[:maxRecentEdits]
		}
		for i, rec := range memories_sorted {
			if utf8.RuneCountInString(rec.Text) > maxSummaryLen {
				rec = rec.Clone()
				rec.Text = rec.Text[:maxSummaryLen] + "..."
				memories_sorted[i] = rec
			}
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

		// TODO: unsure if custom serialization/formatting would make things better or not
		return nil, result, nil
	}
}
