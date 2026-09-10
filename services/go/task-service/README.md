# Task Service

The task service owns task validation, CRUD operations, the versioned HTTP bridge, the stdio MCP adapter, configuration loading, structured logging, ordered migrations, and SQLite persistence.

Its Go domain and bridge structs remain idiomatic handwritten types. Contract tests validate the JSON they produce against the canonical [Draft 2020-12 task schema](../../../contracts/task-api.schema.json), including nullable task descriptions and RFC 3339 timestamps.

## Run

Desktop-sidecar mode binds an operating-system-assigned loopback port and requires a per-launch token:

```bash
TGS_SERVICE_TOKEN="development-only-secret" \
  go run ./cmd/task-service --database-path ./data/tasks.db
```

Remote-development mode must be selected explicitly and uses a separate token:

```bash
TGS_REMOTE_ACCESS_TOKEN="development-only-secret" \
  go run ./cmd/task-service --config ../../../config/development.yaml
```

The process emits one JSON readiness record on stdout. Logs are written to stderr. The authenticated endpoints are:

- `POST /v1/invoke` for the versioned application envelope;
- `GET /health` for process readiness;
- `POST /v1/shutdown` for bounded graceful shutdown.

Request bodies are limited to 64 KiB and encoded responses to 512 KiB. All endpoints require a bearer token. Browser requests additionally pass the configured origin allowlist.

## Run as an MCP server

See the repository-level [MCP usage guide](../../../MCP-USAGE.md) for host configuration, tool inputs, Inspector usage, and troubleshooting.

The separate `task-mcp` command exposes the same Go task service through MCP over stdio:

```bash
go run ./cmd/task-mcp --database-path ./data/tasks.db
```

It provides five tools:

- `tasks_list` and `tasks_get` are read-only;
- `tasks_create` is additive;
- `tasks_update` modifies an existing task;
- `tasks_delete` is destructive and should be called only after explicit user confirmation.

Tool inputs and structured outputs have generated JSON Schemas. MCP-specific argument handling stays in `internal/mcpserver`; task validation and persistence still run through `internal/task` and `internal/sqlite`. Each tool result carries a correlation ID, and structured audit logs go only to stderr.

The command supports the shared `--config` YAML file, `TGS_DATABASE_PATH`, and `TGS_APP_LOG_LEVEL`; explicit `--database-path` and `--log-level` flags take precedence. Stdio needs no bearer token because the MCP host launches and controls the process. Do not point `task-mcp` and `task-service` at the same database concurrently; each executable is intended to be the sole application owner of its SQLite connection.

## Validate

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
```

The Tauri lifecycle integration and remote-development launcher remain outside this module and are not yet implemented.
