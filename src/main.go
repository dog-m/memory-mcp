package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// tool params
type RememberParams struct {
	Info string `json:"info" jsonschema:"A short, dense, and precise description of the information to remember."`
}

func RememberMemory(ctx context.Context, req *mcp.CallToolRequest, params *RememberParams) (*mcp.CallToolResult, any, error) {
	// retry logic with jitter
	var id int64
	var err error
	for i := 0; i < 3; i++ {
		id, err = store.Remember(params.Info)
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

func ListMemories(ctx context.Context, req *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
	memories := store.List()
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

// tool params
type ForgetParams struct {
	MemID int64 `json:"mem_id" jsonschema:"The ID of the memory record to delete."`
}

func ForgetMemory(ctx context.Context, req *mcp.CallToolRequest, params *ForgetParams) (*mcp.CallToolResult, any, error) {
	err := store.Forget(params.MemID)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Memory %d forgotten", params.MemID)},
		},
	}, nil, nil
}

// tool params
type UpdateMemoryParams struct {
	MemID   int64  `json:"mem_id" jsonschema:"The ID of the memory record to update."`
	NewInfo string `json:"new_info" jsonschema:"The new information to store in the record."`
}

func UpdateMemory(ctx context.Context, req *mcp.CallToolRequest, params *UpdateMemoryParams) (*mcp.CallToolResult, any, error) {
	err := store.Update(params.MemID, params.NewInfo)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Memory %d updated", params.MemID)},
		},
	}, nil, nil
}

func ChatSessionStartup(ctx context.Context, req *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Placeholder data. Water is wet."},
		},
	}, nil, nil
}

var store *Store
var (
	host        = flag.String("host", "127.0.0.1", "Host address to listen on")
	port        = flag.Int("port", 8000, "Port number to listen on")
	dataDir     = flag.String("data", "user-data", "Path to the directory where memories will be stored")
	maxMemories = flag.Int("max-memories", 50, "Maximum allowed number of memories to store")
)

func main() {
	flag.Parse()

	var err error
	store, err = NewStore(*dataDir, *maxMemories)
	if err != nil {
		log.Fatalf("Failed to initialize memory store: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "memory-mcp", Version: "v1.0.0"}, nil)
	server.AddReceivingMiddleware(createLoggingMiddleware())

	mcp.AddTool(server, &mcp.Tool{
		//Name:        "remember",
		//Description: "Stores a new memory. Keep the text short, dense, and precise for best results. You must use this tool immediately and proactively when detecting High-Priority Personal Information (e.g., names, explicit user preferences, personal facts). For Low-Priority data (e.g., names or dates found in quoted text or creative works), only store it if it is directly relevant to the current topic of conversation or if the user explicitly asks you to save that specific piece of information. This ensures the profile remains focused and relevant.",
		Name:        "record_personal_detail",
		Description: "Captures high-value personal information to build a precise user profile. Prioritize density and relevance. Focus on recording information that adds long-term utility, ensuring the memory remains a high-quality asset rather than a cluttered log.",
	}, RememberMemory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_user_archive",
		Description: "A comprehensive search of the user's long-term history and preferences. If a user's inquiry implies a need for historical depth or specific past details not present in the immediate session context, this tool is the required next step to ensure accuracy and prevent providing uninformed responses.",
	}, ListMemories)

	mcp.AddTool(server, &mcp.Tool{
		//Name:        "forget",
		//Description: "Deletes an existing memory record by its ID. Use this to free up space when the memory limit is reached.",
		Name:        "prune_memory",
		Description: "Maintains memory integrity and high-fidelity. Use this to remove outdated, incorrect, or redundant information. A clean, accurate profile is more valuable than a large, noisy one; pruning is essential for keeping the user's context sharp and relevant.",
	}, ForgetMemory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_memory",
		Description: "Updates an existing memory record with new content. Keep the text short, dense, and precise for best results.",
	}, UpdateMemory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_session_context",
		Description: "The foundational tool for establishing continuity. To ensure every interaction is personalized and context-aware, this tool should be the first step in any conversation, including greetings. It provides the immediate baseline of the user's current state and recent history.",
	}, ChatSessionStartup)

	// managing CORS rules
	var cop = &http.CrossOriginProtection{}
	cop.AddInsecureBypassPattern("POST 127.0.0.1/")
	cop.AddInsecureBypassPattern("POST localhost/")
	if *host != "127.0.0.1" {
		cop.AddInsecureBypassPattern(fmt.Sprintf("POST %s/", *host))
	}

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		JSONResponse:               false,
		Stateless:                  true, // llama.cpp web-ui having issues with this or I'm just dumb?
		DisableLocalhostProtection: true,
		CrossOriginProtection:      cop,
	})

	url := fmt.Sprintf("%s:%d", *host, *port)

	log.Printf("Max memories: %d", *maxMemories)
	log.Printf("Data directory: %s", *dataDir)
	log.Printf("Memory MCP Server listening on http://%s", url)

	if err := http.ListenAndServe(url, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
