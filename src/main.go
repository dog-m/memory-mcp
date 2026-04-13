package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	host            = flag.String("host", "127.0.0.1", "Host address to listen on")
	port            = flag.Int("port", 8000, "Port number to listen on")
	dataDir         = flag.String("data", "user-data", "Path to the directory where memories will be stored")
	maxMemories     = flag.Int("max-memories", 50, "Maximum allowed number of memories to store")
	toolDefinitions = flag.String("prompts", "", "Path to a file containing tool names and matching descriptions")
)

func main() {
	flag.Parse()

	if *maxMemories <= 0 || *maxMemories >= math.MaxUint16 {
		log.Fatalf("Invalid record limit: %d", *maxMemories)
	}

	var err error
	var dataDirResolved string
	if dataDirResolved, err = filepath.Abs(*dataDir); err != nil {
		log.Fatalf("Unable to resolve data directory: %v", err)
	}

	var storage *MemoryStorage
	storage, err = StorageInit(dataDirResolved, *maxMemories)
	if err != nil {
		log.Fatalf("Failed to initialize memory store: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "memory-mcp", Version: "v1.0.0"}, nil)
	server.AddReceivingMiddleware(createLoggingMiddleware())

	var tools *Tools
	if *toolDefinitions != "" {
		tools, err = LoadToolsFrom(*toolDefinitions)
		if err != nil {
			log.Fatalf("Failed to load tool definitions: %v", err)
		}
	} else {
		tools = GetToolsDefault()
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.ChatSessionStartup.Name,
		Description: tools.ChatSessionStartup.Description,
	}, GetChatSessionStartupHandler())

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.Remember.Name,
		Description: tools.Remember.Description,
	}, GetRememberMemoryHandler(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.ListMemories.Name,
		Description: tools.ListMemories.Description,
	}, GetListMemoriesHandler(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.Forget.Name,
		Description: tools.Forget.Description,
	}, GetForgetMemoryHandler(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.UpdateMemory.Name,
		Description: tools.UpdateMemory.Description,
	}, GetUpdateMemoryHandler(storage))

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
	log.Printf("Data directory: %s", dataDirResolved)
	log.Printf("Memory MCP Server listening on http://%s", url)

	if err := http.ListenAndServe(url, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
