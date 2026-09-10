# ADR 0001: Electron-style typed service bridge

- Status: accepted
- Date: 2026-09-09

## Context

The proof of concept needs to show a familiar Electron-style boundary between a TypeScript renderer and privileged backend behavior while running domain logic and SQLite persistence in a standalone Go sidecar.

Electron's recommended pattern is a context-isolated preload facade that exposes one narrow method per capability. Those methods use asynchronous `ipcRenderer.invoke` calls handled by `ipcMain.handle`; Electron explicitly warns against exposing the complete IPC primitive to renderer code. Values crossing Electron IPC must be compatible with its structured-clone serialization, and thrown main-process errors lose information during serialization unless the application defines its own error contract.

Tauri offers a close conceptual match: asynchronous commands accept and return JSON-serializable values, Rust is the privileged core boundary, and capabilities constrain which commands a webview may invoke. Tauri sidecars can package and run external executables, including the Go service required by this project.

The POC also needs remote web development. A desktop-only stdin/stdout protocol would require a second Go adapter in addition to HTTP and would make transport parity harder to demonstrate.

## Decision

Use four explicit layers:

1. `TaskCoreClient`, extracted to `packages/node/task-core-library`, is the typed task facade consumed by React features and other adapters.
2. `DesktopTransport` and `HttpTransport` implement the same internal transport interface.
3. A single capability-scoped Tauri command, `service_invoke`, is the desktop trust boundary. It accepts only a versioned request envelope, rejects unknown operation names and oversized input, and proxies to a fixed Go-service address. It never accepts a URL from the renderer.
4. The Go sidecar owns request validation, operation dispatch, domain behavior, configuration, logging, and SQLite persistence.

Desktop flow:

```text
React -> TaskCoreClient -> DesktopTransport -> Tauri service_invoke -> Go loopback HTTP -> SQLite
```

Remote-development flow:

```text
Browser -> TaskCoreClient -> HttpTransport -------------------------> Go HTTP -> SQLite
```

The Go service listens on `127.0.0.1` only. In desktop mode Rust generates a per-launch bearer token, passes it to the child without exposing it to the webview, waits for a machine-readable readiness message, and adds the token while proxying requests. Human-readable and structured service logs go to stderr so stdout can be reserved for lifecycle control messages.

The bridge protocol is JSON and uses one operation endpoint. The initial operations are explicitly allowlisted, for example `system.health`, `config.getPublic`, `tasks.list`, `tasks.get`, `tasks.create`, `tasks.update`, and `tasks.delete`. TypeScript and Go representations are derived from the canonical contract. Rust needs to understand only the envelope, protocol version, operation allowlist, payload limit, and response limit.

The primary interaction is asynchronous request/response. Small lifecycle notifications may use Tauri events. Tauri channels are reserved for future ordered streams; this CRUD POC does not introduce streaming merely to demonstrate it.

## Why Rust remains in the middle

The Rust layer is intentionally thin but useful. It:

- starts and stops the sidecar with the desktop application;
- keeps the service address and bearer token out of renderer code;
- constrains which operations the webview may request;
- enforces protocol and payload limits before privileged work;
- translates process startup, timeout, and crash failures into stable bridge errors;
- supplies correlation context for logs across the trust boundary.

Rust does not contain task rules, SQL, migrations, or a second copy of the Go domain API.

## Alternatives considered

### Webview calls the Go loopback server directly

Rejected for desktop mode. It exposes connection credentials to renderer code, introduces CORS and local-port concerns, and bypasses Tauri's privileged command boundary. Direct HTTP remains appropriate for the explicitly enabled remote-development mode.

### Rust and Go communicate over newline-delimited JSON on stdin/stdout

Deferred. It avoids a listening socket, but requires framing, concurrent request matching, backpressure, and a second Go transport alongside the HTTP server needed for remote development. It can be reconsidered if eliminating the loopback listener becomes a requirement.

### One Tauri command per domain operation

Rejected for the initial POC because it duplicates the complete Go contract in Rust. The generic command is not unrestricted: Rust deserializes the operation as a closed allowlist and always targets the internally managed service.

### Put domain behavior in Rust

Rejected. It would stop the POC from proving the TypeScript-to-standalone-Go-service model.

## Consequences

- React code has an Electron-like, testable API without importing Tauri or using `fetch` directly.
- Desktop and remote modes share application methods, schemas, and Go behavior.
- The Go service must provide authentication, request limits, timeouts, and a readiness protocol in addition to CRUD behavior.
- Builds must produce and bundle one Go sidecar binary per supported Tauri target.
- Contract compatibility and transport equivalence become explicit test responsibilities.
- Loopback HTTP adds a small local attack surface, mitigated by loopback-only binding, a per-launch secret, fixed proxy destinations, strict operation validation, and shutdown with the parent application.

## Sources

- [Electron: Inter-Process Communication](https://www.electronjs.org/docs/latest/tutorial/ipc)
- [Electron: Context Isolation](https://www.electronjs.org/docs/latest/tutorial/context-isolation)
- [Electron: Security](https://www.electronjs.org/docs/latest/tutorial/security)
- [Tauri: Inter-Process Communication](https://v2.tauri.app/concept/inter-process-communication/)
- [Tauri: Calling Rust from the Frontend](https://v2.tauri.app/develop/calling-rust/)
- [Tauri: Permissions](https://v2.tauri.app/security/permissions/)
- [Tauri: Embedding External Binaries](https://v2.tauri.app/develop/sidecar/)
