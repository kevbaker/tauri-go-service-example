# ADR 0005: Go stdio MCP server

- Status: accepted
- Date: 2026-09-10

## Context

AI hosts need a narrow way to operate the task application through Model Context Protocol (MCP). The existing TypeScript core client and Go HTTP bridge could support a separate Node MCP process, but that path first requires a production HTTP transport, service discovery, token exchange, and two managed child processes. Those pieces are not implemented yet.

The Go module already contains the canonical task application service, SQLite repository, configuration, and safe error classification. MCP is another inbound adapter and must not become a second implementation of task behavior.

## Decision

Add `cmd/task-mcp` to the existing Go task-service module. It runs MCP over stdio and composes:

```text
MCP host -> task-mcp -> MCP tool adapter -> task.Service -> SQLite repository
```

The adapter exposes the allowlisted tools `tasks_list`, `tasks_get`, `tasks_create`, `tasks_update`, and `tasks_delete`. It uses typed MCP inputs and structured outputs, maps errors to stable safe messages, adds correlation IDs, writes audit logs to stderr, and describes delete as requiring explicit user confirmation.

The MCP process reads only the shared configuration values it needs: database path and log level. It does not start a network listener or require a bearer token because the MCP host owns the stdio child process. It is an alternative composition root for the task application and is not intended to write the same SQLite database concurrently with `task-service`.

## Consequences

- Task rules and persistence are reused directly with no TypeScript or MCP-specific domain implementation.
- The initial MCP server is one process with no loopback discovery or token handoff.
- MCP tool schemas are model-facing contracts. They intentionally differ where helpful; for example, updates use `clearDescription` to represent an explicit nullable update clearly.
- The TypeScript core library remains the UI and transport-neutral application client. A future remote MCP gateway may use it when the HTTP transport and service lifecycle exist.
- Stdio protocol traffic stays on stdout and all diagnostics stay on stderr.
