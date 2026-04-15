# Memory MCP

A simple and portable, single-user MCP server for managing long-term memories.

## Features

- **Session Startup**: Get current time, memory utilization, and a summary of recent edits.
- **Add Memory**: Create new memory records with automatic timestamps.
- **List Memories**: Retrieve all stored memories in JSON format.
- **Remove Memory**: Delete specific memory records by ID.
- **Update Memory**: Update the content of an existing memory record.
- **Configurable Limit**: Set a maximum number of memories to prevent storage bloat.

## Installation

Ensure you have Go installed on your system.

1. Clone the repository.
2. Build the executable:

```bash
go build -o build/memory-mcp.exe ./src
```

## Usage

Run the server via CLI:

```bash
./build/memory-mcp.exe -data ./user-data -max-memories 25 -host 127.0.0.1 -port 1234 -prompts "./tools-default.json"
```

### Arguments

- `-host`: Host address to listen on (default: `127.0.0.1`).
- `-port`: Port number to listen on (default: `8000`).
- `-data`: Path to the directory where memories will be stored (default: `user-data`).
- `-max-memories`: Maximum allowed number of memories to store (default: `50`, should be between `0` and `65535`).
- `-max-recent-edits`: Number of recent edits shown by the session info tool (default: `5`).
- `-prompts`: Path to a file containing tool names and matching descriptions (default: `<builtin>` - uses builtin ones).

## Implementation Details

- **Protocol**: Uses MCP over HTTP-streaming transport.
- **Persistence**: Memories are stored as individual JSON files (e.g., `mem-<ID>.json`) within the data directory.
- **Concurrency**: Uses `sync.RWMutex` to ensure thread safety when multiple agents access the server.
- **Time Format**: Timestamps use the RFC1123 format for session startup and `YYYY-MM-DD hh:mm:ss ZZZ` (with timezone) for records.
