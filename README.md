# Tauri Go Service Example

A small reference desktop application that demonstrates how to combine a React and TypeScript frontend, a Tauri desktop shell, embedded Go services, and SQLite persistence.

The communication layer is intentionally modeled after Electron's context-isolated preload pattern: frontend code calls a narrow typed application API without knowing whether the implementation is running through desktop IPC or over HTTP in remote development mode. The concrete bridge design is recorded in [ADR 0001](.agents/decisions/0001-electron-style-service-bridge.md), with executable acceptance criteria in the [service bridge POC specification](.agents/specs/service-bridge-poc.md).

> **Project status:** this is an executable scaffold, not yet the completed service-bridge POC. The reusable `TaskCoreClient` package and React/Vaadin task UI are implemented, but the UI currently uses an in-memory adapter and resets on reload. The standalone Go service implements authenticated HTTP task CRUD and SQLite persistence. The Rust sidecar lifecycle, desktop IPC adapter, browser HTTP adapter, and frontend-to-Go integration are still pending.

## What works today

| Capability | Status | Notes |
| --- | --- | --- |
| React, TypeScript, and Vite application | Implemented | Runs in a browser and in the generated Tauri shell |
| Responsive Vaadin task CRUD UI | Implemented | Uses an in-memory client; data resets when the page reloads |
| Shared task core library | Implemented and unit tested | Owns task DTOs, five CRUD methods, bridge requests, response validation, and normalized errors; production transports and generated contract types are pending |
| Go task domain and CRUD service | Implemented and unit tested | Includes validation, stable application errors, filtering, and pagination |
| SQLite repository and migrations | Implemented and tested | Applies embedded migrations and persists across service restarts |
| Authenticated Go HTTP bridge | Implemented and tested | Provides health, invoke, and graceful-shutdown endpoints |
| Central Go configuration | Implemented and tested | Supports defaults, YAML, `TGS_` environment values, CLI overrides, and a public allowlist |
| Structured Go logging and readiness output | Implemented and tested | Logs go to stderr; the machine-readable readiness record goes to stdout |
| Tauri-to-Go sidecar lifecycle and IPC proxy | Planned | Rust still contains the generated example command |
| Go-backed browser transport | Planned | `dev:remote` currently serves only the in-memory UI |
| Packaged Go sidecar and end-to-end parity tests | Planned | Required before the POC is complete |

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
| API contract | Shared JSON Schema or OpenAPI-generated TypeScript and Go types | Keeps requests, responses, and validation aligned |
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

Start the current in-memory UI from the repository root:

```bash
npm run dev --workspace task-app
```

Vite serves the application at `http://localhost:1420`. This exercises the responsive task UI and typed client boundary, but it does not start or call the Go service. Task data is intentionally reset on reload.

To opt into a network-accessible Vite server for testing from another device:

```bash
npm run dev:remote
```

This binds Vite to all local interfaces. Use it only on a trusted network. The Go service still needs to be started separately, and the UI remains on its in-memory client until `HttpTransport` is implemented.

### Tauri shell

Run the generated desktop shell with:

```bash
npm run tauri --workspace task-app -- dev
```

The shell displays the same in-memory UI. It does not yet build, launch, authenticate, or stop the Go sidecar; the Rust core still exposes only the generated example command.

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

This listens on `127.0.0.1:8787` by default. Binding beyond loopback requires an explicit host override and a matching origin allowlist. Remote mode is development-only and is not currently connected to the browser UI.

## Validate the scaffold

Run the repository test suites and type checking from the root:

```bash
npm test
npm run typecheck
npm run build --workspace task-app
```

The root test command runs the frontend tests, Go tests, and Rust tests. Additional Go checks can be run from `services/go/task-service`:

```bash
go vet ./...
go test -race ./...
```

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

The eventual desktop flow is launched with:

```bash
npm run tauri --workspace task-app -- dev
```

Today this launches the generated Tauri shell and in-memory UI only. Rust does not yet manage the Go process or proxy service calls.

### Remote web mode

Remote mode starts the Go HTTP server and Vite development server so the UI can be opened from another device on the local network.

The intended command is:

```bash
npm run dev:remote
```

At the current stage, this command still starts only the React UI; it has not yet been wired to launch the implemented Go HTTP service. Tauri commands are not available in an ordinary browser; browser service calls will use the planned `HttpTransport`.

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

Development may use `./data/tasks.db`, which remains outside version control. Resolving the platform-appropriate application data directory is part of the future packaged-sidecar integration.

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
- **Contract tests:** generated TypeScript and Go models plus equivalent IPC/HTTP behavior.
- **React tests:** task flows, validation messages, loading states, and responsive variants.
- **End-to-end tests:** create, edit, complete, reload, and delete a task in desktop and remote modes.

The minimum acceptance flow is: create a task, restart the application, confirm it persisted, edit it, mark it done, and delete it.

## Initial implementation milestones

1. Scaffold the React/Vite frontend, Tauri shell, and Go service. (Complete)
2. Define configuration and task API schemas, then generate shared types. (Configuration schema complete; API schema and generated bindings remain)
3. Implement SQLite migrations and task CRUD in Go. (Complete)
4. Add the production desktop IPC adapter and connect it to the typed frontend client.
5. Replace the in-memory Vaadin task UI adapter with the real typed desktop and HTTP transports.
6. Add structured logging and correlation IDs.
7. Add secure remote development mode and responsive validation.
8. Package the Go sidecar with the desktop application and add end-to-end tests.

## Non-goals

The first version will not include user accounts, cloud synchronization, collaborative editing, plugin support, or a generalized microservice framework. The goal is a clear, working example of one desktop application and one embedded service with well-defined boundaries.

## License

UNLICENSED. This repository is currently private unless the license is changed explicitly.
