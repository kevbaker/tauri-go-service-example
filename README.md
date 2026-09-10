# Tauri Go Service Example

A small reference desktop application that demonstrates how to combine a React and TypeScript frontend, a Tauri desktop shell, embedded Go services, and SQLite persistence.

The communication layer is intentionally modeled after Electron's context-isolated preload pattern: frontend code calls a narrow typed application API without knowing whether the implementation is running through desktop IPC or over HTTP in remote development mode. The concrete bridge design is recorded in [ADR 0001](.agents/decisions/0001-electron-style-service-bridge.md), with executable acceptance criteria in the [service bridge POC specification](.agents/specs/service-bridge-poc.md).

> **Project status:** the primary service-bridge POC is implemented. The React/Vaadin UI calls the reusable `TaskCoreClient`; desktop mode uses a narrow Tauri command to manage and proxy to a bundled Go sidecar, while remote development uses an authenticated Vite-to-Go proxy. Go owns full task CRUD and SQLite persistence. Cross-platform packaged bundles and browser-driven end-to-end tests still require CI verification.

## What works today

| Capability | Status | Notes |
| --- | --- | --- |
| React, TypeScript, and Vite application | Implemented | Runs in a browser and in the generated Tauri shell |
| Responsive Vaadin task CRUD UI | Implemented | Uses the Go service, persists through SQLite, and refreshes backend changes manually or every five seconds |
| Shared task core library | Implemented and unit tested | Owns generated task DTOs, five CRUD methods, bridge requests, response validation, and normalized errors |
| Canonical task API contract | Implemented and contract tested | Draft 2020-12 JSON Schema generates TypeScript DTOs and validates Go wire representations |
| Go task domain and CRUD service | Implemented and unit tested | Includes validation, stable application errors, filtering, and pagination |
| SQLite repository and migrations | Implemented and tested | Applies embedded migrations and persists across service restarts |
| Authenticated Go HTTP bridge | Implemented and tested | Provides health, invoke, and graceful-shutdown endpoints |
| Central Go configuration | Implemented and tested | Supports defaults, YAML, `TGS_` environment values, CLI overrides, and a public allowlist |
| Structured Go logging and readiness output | Implemented and tested | Logs go to stderr; the machine-readable readiness record goes to stdout |
| MCP task server | Implemented and protocol tested | A separate Go stdio command exposes five task tools through the same Go domain service and SQLite repository |
| Tauri-to-Go sidecar lifecycle and IPC proxy | Implemented and unit tested | Rust allowlists task operations, owns the token/address, validates readiness, bounds response buffering, and waits for graceful shutdown before forcing termination |
| Go-backed browser transport | Implemented and unit tested | `dev:remote` starts Go and Vite; Vite authenticates the browser session and keeps the Go token out of browser JavaScript |
| Target-specific Go sidecar packaging | Configured | Local target build works; the full Linux/macOS/Windows CI matrix remains unverified |

## Goals

- Provide an easy-to-follow Tauri example with a React TypeScript frontend.
- Run the application backend as an embedded Go sidecar.
- Persist application data locally in SQLite.
- Expose one typed frontend API with interchangeable desktop IPC and HTTP transports.
- Keep configuration centralized while exposing only browser-safe values to the UI.
- Produce structured logs across the frontend, Tauri shell, and Go service.
- Support a remote development mode with a responsive, mobile-friendly UI.
- Demonstrate a complete but deliberately small CRUD workflow.

## Example domain: Tasks

The application stores a simple task list. A task has:

- an ID;
- a title and optional description;
- a status (`todo`, `in_progress`, or `done`);
- created and updated timestamps.

The primary flow is:

1. View tasks in a responsive list.
2. Create a task.
3. Open and edit a task.
4. Change its status.
5. Delete it after confirmation.

This domain is intentionally ordinary. It is large enough to exercise validation, errors, migrations, persistence, and all CRUD operations without distracting from the service and transport patterns.

## Technology choices

| Area | Choice | Why |
| --- | --- | --- |
| Desktop shell | Tauri | Small native shell with explicit command and sidecar boundaries |
| Frontend | React + TypeScript + Vite | Fast development and a strongly typed UI |
| Design system | [Vaadin Web Components](https://vaadin.com/docs/latest/components) with official React wrappers | Open-source controls, accessible form behavior, Lumo design tokens, and webview/browser portability |
| Backend | Go | Simple deployment, concurrency, and a single sidecar binary |
| Persistence | SQLite | Local, durable storage with no external database dependency |
| API contract | JSON Schema Draft 2020-12 | Generates TypeScript DTOs and validates the idiomatic Go wire representation without losing missing-versus-null semantics |
| Logging | Structured JSON with request/correlation IDs | Makes events traceable across process boundaries |

## Prerequisites

- Node.js with npm. The committed `package-lock.json` is the dependency source of truth.
- Go 1.25 or newer for the task service.
- Rust and the [Tauri 2 platform prerequisites](https://v2.tauri.app/start/prerequisites/) to run the desktop shell.

Install the JavaScript dependencies from the repository root:

```bash
npm install
```

## Run the current implementation

### Browser UI

For the complete browser-to-Go development stack, run from the repository root:

```bash
npm run dev:remote
```

This generates separate short-lived service and browser-access tokens, starts Go on loopback using `config/development.yaml`, and starts Vite on all local interfaces. Append the printed `?access_token=...` value to a Vite Network URL the first time it is opened. Vite exchanges that capability URL for an HTTP-only, same-site session cookie before serving the application. It then proxies `/task-service/*` to Go and adds the separate Go token server-side. Task CRUD persists to `services/go/task-service/data/tasks.db`. Stop both processes with Ctrl+C, and do not expose this development server to an untrusted or public network.

Running `npm run dev --workspace task-app` alone starts only Vite and will display a service-unavailable error because no authenticated Go proxy is present.

### Tauri shell

Run the desktop application with:

```bash
npm run tauri --workspace task-app -- dev
```

The Tauri pre-development step builds the current platform's Go sidecar. On the first task request, Rust starts it with a per-launch token and an application-data SQLite path, validates its readiness message, and proxies the typed request. Closing Tauri requests graceful shutdown and terminates the child if needed.

### Standalone Go service

From `services/go/task-service`, start the service in desktop-compatible loopback mode:

```bash
TGS_SERVICE_TOKEN="development-only-secret" \
  go run ./cmd/task-service --database-path ./data/tasks.db
```

The service selects an available loopback port, writes one readiness JSON record to stdout, and writes logs to stderr. Stop it with an authenticated `POST /v1/shutdown` request or an interrupt signal.

For explicit remote-development mode, use the shared development configuration:

```bash
TGS_REMOTE_ACCESS_TOKEN="development-only-secret" \
  go run ./cmd/task-service --config ../../../config/development.yaml
```

This listens on `127.0.0.1:8787` by default. Binding beyond loopback requires an explicit host override and a matching origin allowlist. The root `npm run dev:remote` command handles the access token and connects this service to the browser through Vite's development proxy.

### MCP task server

See [Using the Task App MCP Server](MCP-USAGE.md) for host configuration, tool inputs, Inspector usage, persistence behavior, and troubleshooting.

Run the stdio MCP server from `services/go/task-service`:

```bash
go run ./cmd/task-mcp --database-path ./data/tasks.db
```

The MCP host owns the process and communicates over stdin/stdout. All logs go to stderr so they cannot corrupt MCP protocol messages. The server exposes `tasks_list`, `tasks_get`, `tasks_create`, `tasks_update`, and `tasks_delete`; delete is marked destructive and its description requires explicit user confirmation.

For durable host configuration, build the executable and point the host at the resulting absolute path:

```bash
go build -o /absolute/path/to/task-mcp ./cmd/task-mcp
```

The command reads the shared YAML configuration with `--config`, honors `TGS_DATABASE_PATH` and `TGS_APP_LOG_LEVEL`, and accepts `--database-path` and `--log-level` as highest-precedence overrides. It opens SQLite directly, so use it as an alternative task-service process for that database rather than running it alongside another writer.

## Validate the scaffold

Run the repository test suites and type checking from the root:

```bash
npm test
npm run typecheck
npm run build --workspace task-app
```

`npm run typecheck` also checks that the committed TypeScript task bindings match `contracts/task-api.schema.json`. After intentionally changing that schema, run `npm run contract:generate`.

The root test command runs the frontend tests, Go tests, and Rust tests. Additional Go checks can be run from `services/go/task-service`:

```bash
go vet ./...
go test -race ./...
```

## Desktop builds and releases

Merges to `main` run the desktop workflow and produce unsigned Linux, macOS, and Windows bundles as GitHub Actions artifacts. The root `package.json` is the canonical desktop version: when its `version` changes in a merge to `main`, the same workflow also creates the matching `v<version>` GitHub Release and uploads the bundles. A manually dispatched build does not create a release.

## Target architecture

```text
React feature code
       |
       v
Shared task core client
       |
       +--------------------+
       |                    |
       v                    v
Tauri IPC transport     HTTP transport
(desktop mode)          (remote dev mode)
       |                    |
       v                    |
Thin Rust bridge            |
       |                    |
       +----------+---------+
                  |
                  v
       Authenticated Go RPC
          config / logging
                  |
                  v
               SQLite
```

The important boundary is the transport-neutral client in [`packages/node/task-core-library`](packages/node/task-core-library/README.md). Components and future adapters call methods such as:

```ts
const tasks = await taskCoreClient.tasks.list();
await taskCoreClient.tasks.create({ title: "Try the Go sidecar" });
await taskCoreClient.tasks.update(taskId, { status: "done" });
await taskCoreClient.tasks.delete(taskId);
```

Feature code does not call Tauri primitives or `fetch` directly. Once the bridge is complete, the application will select a transport at startup:

- **Desktop mode:** the client invokes a capability-scoped Tauri command. Thin Rust code validates the envelope and operation, manages the sidecar lifecycle, and proxies to the authenticated loopback-only Go endpoint.
- **Remote development mode:** the same client uses the Go service's HTTP API.

Requests and responses will use the same versioned JSON envelope in both modes. Transport-specific details stay in adapters, making the TypeScript facade analogous to an Electron preload API while Tauri capabilities, Rust validation, and Go validation enforce the actual security boundaries.

## Repository layout

```text
.
├── .agents/                  # Patterns, specifications, and decisions
├── apps/
│   └── task-app/             # React/Vite root with Rust in src-tauri
├── services/
│   └── go/
│       └── task-service/     # Go service, API adapters, and SQLite access
├── packages/
│   ├── node/                 # Shared TypeScript packages
│   │   └── task-core-library/ # Transport-neutral task CRUD client
│   └── go/                   # Shared Go packages
├── config/
│   ├── schema.json           # Canonical configuration schema
│   ├── default.yaml          # Shared non-secret defaults
│   └── development.yaml      # Development overrides
├── contracts/
│   └── task-api.schema.json  # Canonical task request and response contract
├── scripts/
│   └── generate-task-contract.mjs
├── AGENTS.md                 # Repository-wide working agreement
└── README.md
```

Package-specific READMEs provide more detail for the [task app](apps/task-app/README.md), [task core library](packages/node/task-core-library/README.md), [Go task service](services/go/task-service/README.md), and [configuration](config/README.md). The accepted conventions, specifications, and architecture records live under [`.agents`](.agents/README.md).

For a concise case for this design—including Electron migration and MCP extensibility—see [Why This Tauri Architecture](ARCHITECTURE-EXPLAINED.md).

## Communication model

The Go service owns domain logic and persistence. The frontend owns presentation and user interaction. Tauri owns desktop lifecycle and the narrow bridge between them.

The initial task contract should include these allowlisted operations:

| Operation | Application method | Bridge operation |
| --- | --- | --- |
| List tasks | `tasks.list()` | `tasks.list` |
| Read task | `tasks.get(id)` | `tasks.get` |
| Create task | `tasks.create(input)` | `tasks.create` |
| Update task | `tasks.update(id, input)` | `tasks.update` |
| Delete task | `tasks.delete(id)` | `tasks.delete` |

The completed transports will submit operations through the Go service's versioned RPC-over-HTTP endpoint. Each call will carry a correlation ID. Errors cross the Go HTTP boundary today as a stable application error shape; the desktop and browser clients must preserve that shape rather than exposing raw Go, SQLite, HTTP, or Tauri errors.

```ts
type TaskClientError = {
  code:
    | "VALIDATION"
    | "NOT_FOUND"
    | "CONFLICT"
    | "INTERNAL"
    | "SERVICE_UNAVAILABLE"
    | "PROTOCOL_ERROR";
  message: string;
  requestId?: string;
  fieldErrors?: Record<string, string>;
};
```

## Configuration

Configuration has one schema and a predictable precedence order:

1. built-in defaults;
2. the selected configuration file;
3. environment variables;
4. explicit command-line overrides.

The Go service loads and validates the complete configuration. The frontend receives a read-only, browser-safe subset at startup through the application client. Secrets, filesystem credentials, and other service-only values must never be serialized to the frontend.

Example configuration:

```yaml
app:
  environment: development
  logLevel: debug

database:
  path: ./data/tasks.db

server:
  enabled: false
  host: 127.0.0.1
  port: 8787

ui:
  appName: Tauri Go Tasks
  pageSize: 25
```

Environment variables use a common prefix and nested keys, for example `TGS_APP_LOG_LEVEL` and `TGS_SERVER_PORT`.

## Target cross-layer logging

The Go service currently emits structured events. The completed application will use consistent fields across every layer:

```json
{
  "time": "2026-01-01T12:00:00Z",
  "level": "info",
  "component": "go-service",
  "event": "task.created",
  "correlationId": "01J...",
  "taskId": "01J..."
}
```

- The Go service writes structured logs and is the source of truth for domain and persistence events.
- Tauri records process lifecycle, bridge, and sidecar failures.
- The frontend records UI diagnostics and forwards relevant application events without logging task descriptions or other sensitive content by default.
- Desktop development uses readable console output; automated and packaged environments use JSON.

## Development modes

### Desktop mode

Desktop mode starts Vite, launches the Tauri application, and lets Rust manage the Go sidecar. Go binds to an operating-system-assigned loopback port and requires a per-launch bearer token known only to Rust. Rust waits for a structured readiness message before accepting calls. Closing the desktop application also stops its child service.

Launch the desktop flow with:

```bash
npm run tauri --workspace task-app -- dev
```

The Tauri build hook creates the correctly named Go sidecar for the current Rust target. Rust keeps the listener address, access token, and database path outside the webview.

### Remote web mode

Remote mode starts the Go HTTP server and Vite development server so the UI can be opened from another device on the local network.

Run both services with:

```bash
npm run dev:remote
```

This command generates an ephemeral development token, starts Go on loopback, waits for readiness, and then exposes Vite to the local network. The browser uses `HttpTaskTransport`; Vite injects authentication while proxying to Go.

Remote mode must:

- require an explicit opt-in rather than exposing a server by default;
- print the local and LAN URLs at startup;
- use configurable bind addresses and ports;
- restrict allowed origins;
- provide development-only authentication or a short-lived access token before binding beyond loopback;
- keep the SQLite database and Go service on the host machine;
- use the HTTP transport without changing feature code.

The UI is designed mobile-first. Forms collapse to a single column, task actions remain touch-friendly, dialogs fit narrow screens, and the task table becomes a card list at small breakpoints.

## Data and migrations

SQLite data belongs to the Go service. The frontend never opens or modifies the database directly.

The current Go service:

- enable foreign keys;
- use embedded, ordered migrations;
- apply migrations during startup before accepting requests;
- store timestamps in UTC;
- use parameterized queries;
- persist task records at the configured database path.

Remote development uses `services/go/task-service/data/tasks.db`, which remains outside version control. Desktop mode resolves the platform-appropriate Tauri application data directory and stores `tasks.db` there.

## Security boundaries

- Only explicit application operations are available over Tauri IPC.
- The Go service validates all input regardless of transport.
- SQL implementation details and filesystem paths are not exposed to the UI.
- Browser-safe configuration is allowlisted, not created by removing known secrets.
- Remote access is disabled in packaged builds unless a future use case deliberately enables and secures it.
- Logs avoid task content, tokens, and configuration secrets by default.

## Testing strategy

- **Go unit tests:** domain validation, service behavior, configuration, and error mapping.
- **Repository tests:** SQLite queries and migrations against temporary databases.
- **Contract tests:** generated TypeScript DTO drift, runtime enum parity, Go wire-schema compatibility, and equivalent IPC/HTTP behavior.
- **React tests:** task flows, validation messages, loading states, and responsive variants.
- **End-to-end tests:** create, edit, complete, reload, and delete a task in desktop and remote modes.

The minimum acceptance flow is: create a task, restart the application, confirm it persisted, edit it, mark it done, and delete it.

## Initial implementation milestones

1. Scaffold the React/Vite frontend, Tauri shell, and Go service. (Complete)
2. Define configuration and task API schemas, then generate or validate language bindings. (Complete for configuration and the task API)
3. Implement SQLite migrations and task CRUD in Go. (Complete)
4. Add the production desktop IPC adapter and connect it to the typed frontend client. (Complete)
5. Replace the in-memory Vaadin task UI adapter with the real typed desktop and HTTP transports. (Complete)
6. Add structured logging and correlation IDs.
7. Add secure remote development mode and responsive validation. (Complete for the trusted-network POC)
8. Package the Go sidecar with the desktop application and add end-to-end tests. (Packaging configured; cross-platform UI automation remains)

## Non-goals

The first version will not include user accounts, cloud synchronization, collaborative editing, plugin support, or a generalized microservice framework. The goal is a clear, working example of one desktop application and one embedded service with well-defined boundaries.

## License

UNLICENSED. This repository is currently private unless the license is changed explicitly.
