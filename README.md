# Memory MCP (WIP)

A simple and portable, single-user MCP server for managing long-term memories.

## Features

- **Remember**: Create new memory records with automatic timestamps.
- **List Memories**: Retrieve all stored memories in JSON format.
- **Forget**: Delete specific memory records by ID.
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
./build/memory-mcp.exe -data ./user-data -max-memories 25 -host 127.0.0.1 -port 1234
```

### Arguments

- `-host`: Host address to listen on (default: `127.0.0.1`).
- `-port`: Port number to listen on (default: `8000`).
- `-data`: Path to the directory where memories will be stored (default: `user-data`).
- `-max-memories`: Maximum allowed number of memories to store (default: `50`).

## Implementation Details

- **Concurrency**: Uses `sync.RWMutex` to ensure thread safety when multiple agents access the server.
- **Persistence**: Memories are stored in `memories.json` within the data directory.
- **Protocol**: Uses MCP over HTTP-streaming transport.
