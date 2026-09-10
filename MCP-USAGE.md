# Using the Task App MCP Server

This guide explains how to build, configure, test, and safely use the task application's Model Context Protocol (MCP) server.

## Current status

The Go MCP server is implemented as `task-mcp` and communicates with an MCP host over standard input and output:

```text
MCP host -> task-mcp -> Go task service -> SQLite
```

It exposes the existing Go task domain and SQLite repository through five MCP tools. It does not route through React, Tauri, HTTP, or the TypeScript task-core library.

The React application now uses the Go service and SQLite through desktop or browser transports. The MCP command is a separate Go process: it does not automatically discover the database used by a running desktop application. To operate on the same data, configure the exact database path and stop the other task-service process first; the current single-owner design does not support running `task-mcp` and `task-service` concurrently against one database.

## Prerequisites

- Go 1.25 or newer.
- An MCP host that can launch local stdio servers, or the MCP Inspector for manual testing.
- An absolute path for the SQLite database you want the MCP process to own.

## Build the server

From `services/go/task-service`:

```bash
go build -o /absolute/path/to/bin/task-mcp ./cmd/task-mcp
```

Use an absolute executable path in host configuration. This avoids depending on the host application's working directory or `PATH`.

For a temporary development run without building:

```bash
go run ./cmd/task-mcp --database-path /absolute/path/to/data/tasks.db
```

`go run` is convenient for development but slower and less predictable when an MCP host starts the process repeatedly. Prefer the built binary for durable configuration.

## Configure an MCP host

MCP hosts use different settings screens and config-file locations, but a typical stdio server entry has this shape:

```json
{
  "mcpServers": {
    "task-app": {
      "type": "stdio",
      "command": "/absolute/path/to/bin/task-mcp",
      "args": [
        "--database-path",
        "/absolute/path/to/data/tasks.db",
        "--log-level",
        "info"
      ]
    }
  }
}
```

Replace every placeholder with an absolute path. Some hosts infer `"type": "stdio"` and do not require that field; follow the host's MCP configuration documentation when its shape differs.

After saving the configuration, restart or reload the MCP host. It should launch `task-mcp`, negotiate an MCP session, and discover the five tools below.

## Available tools

| Tool | Purpose | Input |
| --- | --- | --- |
| `tasks_list` | List persisted tasks | Optional `status`, `limit`, and `offset` |
| `tasks_get` | Read one task | Required `id` |
| `tasks_create` | Create a task | Required `title`; optional `description` and `status` |
| `tasks_update` | Change selected fields | Required `id`; at least one of `title`, `description`, `clearDescription`, or `status` |
| `tasks_delete` | Permanently delete a task | Required `id` |

Valid status values are `todo`, `in_progress`, and `done`. Lists default to 25 tasks and accept at most 100.

To remove a description, use `clearDescription: true`. Do not send `description` and `clearDescription: true` in the same update.

Tool results use structured content. Each call also carries a correlation ID in result metadata and structured stderr logs.

## Example requests to an AI host

- “List my open tasks.”
- “Create a task titled ‘Verify the Tauri bridge’ with status `todo`.”
- “Mark task `<id>` as done.”
- “Remove the description from task `<id>`.”
- “Delete task `<id>`.”

Always inspect the selected task and explicitly confirm before allowing `tasks_delete`. The tool is marked with MCP's destructive annotation, but annotations are hints to hosts—not an authorization mechanism or a guaranteed confirmation dialog.

## Test with MCP Inspector

The official MCP Inspector can launch a local stdio server without adding it permanently to another host.

First build a temporary binary:

```bash
cd services/go/task-service
go build -o /tmp/task-mcp ./cmd/task-mcp
```

Then open the Inspector's web interface:

```bash
npx @modelcontextprotocol/inspector --web -- \
  /tmp/task-mcp \
  --database-path /tmp/task-mcp-demo.db \
  --log-level debug
```

The `--` separator ensures the database and log-level flags are passed to `task-mcp`, rather than interpreted by the Inspector.

In the Inspector:

1. Connect to the server.
2. Open **Tools** and confirm that five `tasks_*` tools are listed.
3. Call `tasks_create` with `{"title":"Inspector test"}`.
4. Call `tasks_list` and copy the created task ID.
5. Call `tasks_update` with that ID and `{"status":"done"}`.
6. Call `tasks_get` to verify the update.
7. After confirming the target, call `tasks_delete` and verify it no longer appears.

The Inspector is a development tool. Do not expose its web interface beyond loopback or use a production database for experiments.

## Configuration

The MCP command accepts:

```text
--config <shared-yaml-path>
--database-path <sqlite-path>
--log-level <debug|info|warn|error>
```

It also reads:

```text
TGS_DATABASE_PATH
TGS_APP_LOG_LEVEL
```

Precedence is built-in defaults, YAML configuration, environment variables, then command-line overrides.

The stdio server does not use the HTTP service's bearer token, bind address, origin allowlist, or network port. The host controls access by controlling which local process it launches.

## Persistence and process ownership

- SQLite data persists after the MCP session ends.
- Relative database paths resolve from the MCP host's working directory; use an absolute path.
- Do not point `task-mcp` and `task-service` at the same database concurrently. The current design expects one application process to own that database.
- Back up important database files before testing destructive tools.
- Do not commit databases, WAL files, or host configuration containing sensitive paths.

## Logging and troubleshooting

MCP protocol messages are written only to stdout. Structured diagnostic and audit logs are written to stderr; sending ordinary logs to stdout would corrupt the stdio protocol.

If the server does not appear in the host:

1. Run the configured binary directly with `--database-path` and check stderr.
2. Confirm that the executable and database paths are absolute and accessible to the host.
3. Confirm the host is configured for stdio rather than Streamable HTTP.
4. Check that another task-service process is not using the same intended database.
5. Use MCP Inspector to distinguish a server problem from host-specific configuration.
6. Run `go test ./...` from `services/go/task-service`.

## Security boundary

Treat enabling this MCP server as granting the selected host permission to read and mutate the configured task database. Tool descriptions and annotations improve host behavior but do not replace process isolation, filesystem permissions, user confirmation, or host trust.

The server intentionally exposes no generic SQL, filesystem, shell, HTTP, or arbitrary bridge-operation tool.

## Implementation references

- [`cmd/task-mcp`](services/go/task-service/cmd/task-mcp/main.go)
- [MCP adapter](services/go/task-service/internal/mcpserver/server.go)
- [MCP architecture decision](.agents/decisions/0005-go-stdio-mcp-server.md)
- [Task service documentation](services/go/task-service/README.md)
- [Official Go SDK quick start](https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/quick_start.md)
- [Official MCP Inspector guide](https://github.com/modelcontextprotocol/docs/blob/main/docs/tools/inspector.mdx)
- [MCP stdio transport specification](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports#stdio)
