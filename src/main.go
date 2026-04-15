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

const PROMPTS_DEFAULT = "<builtin>"

var (
	host            = flag.String("host", "127.0.0.1", "Host address to listen on")
	port            = flag.Uint("port", 8000, "Port number to listen on")
	dataDir         = flag.String("data", "user-data", "Path to the directory where memories will be stored")
	maxMemories     = flag.Uint("max-memories", 50, "Maximum allowed number of memories to store")
	toolDefinitions = flag.String("prompts", PROMPTS_DEFAULT, "Path to a file containing tool names and matching descriptions")
	maxRecentEdits  = flag.Uint("max-recent-edits", 5, "Number of recent edits shown by the session info tool")
)

func main() {
	flag.Parse()

	if *port >= math.MaxUint16 {
		log.Fatalf("Invalid port: %d", *port)
	}
	if *maxMemories >= math.MaxUint16 {
		log.Fatalf("Invalid record limit: %d", *maxMemories)
	}
	log.Printf("Max memories: %d", *maxMemories)

	var err error
	var dataDirResolved string
	if dataDirResolved, err = filepath.Abs(*dataDir); err != nil {
		log.Fatalf("Unable to resolve data directory: %v", err)
	}
	log.Printf("User data directory: %s", dataDirResolved)

	var storage *MemoryStorage
	storage, err = StorageInit(dataDirResolved, int(*maxMemories))
	if err != nil {
		log.Fatalf("Failed to initialize memory store: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "memory-mcp", Version: "v1.0.0"}, nil)
	server.AddReceivingMiddleware(createLoggingMiddleware())

	var tools *Tools
	log.Printf("Tool definition source: %s", *toolDefinitions)
	if *toolDefinitions != PROMPTS_DEFAULT {
		tools, err = LoadToolsFrom(*toolDefinitions)
		if err != nil {
			log.Fatalf("Failed to load tool definitions: %v", err)
		}
	} else {
		tools = GetToolsDefault()
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.Startup.Name,
		Description: tools.Startup.Description,
	}, GetChatSessionStartupHandler(storage, int(*maxRecentEdits)))

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.Add.Name,
		Description: tools.Add.Description,
	}, GetAddMemoryHandler(storage, tools))

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.List.Name,
		Description: tools.List.Description,
	}, GetListMemoriesHandler(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.Remove.Name,
		Description: tools.Remove.Description,
	}, GetRemoveMemoryHandler(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        tools.Update.Name,
		Description: tools.Update.Description,
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

	log.Printf("Memory MCP Server listening on http://%s", url)
	if err := http.ListenAndServe(url, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
