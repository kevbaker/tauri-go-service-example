# Why This Tauri Architecture

## Executive pitch

This architecture keeps the development model that makes Electron productive—a web UI calling a narrow, typed application API—while replacing Electron-specific runtime coupling with explicit, reusable boundaries.

React remains responsible for presentation. A transport-neutral TypeScript core library defines the application-facing contract. Tauri provides the desktop shell and privileged security boundary. Go owns domain behavior, integrations, and SQLite persistence. The same core contract can be reached through desktop IPC, remote-development HTTP, tests, and future Model Context Protocol (MCP) adapters.

The result is not merely “an Electron application rewritten in Rust.” It is a portable application architecture in which the UI, desktop shell, service runtime, and AI-facing integrations can evolve independently.

For this project, the central proposition is:

> Preserve the Electron pattern developers already understand, but make the application contract—not Electron, Tauri, HTTP, or MCP—the center of the system.

## The architecture at a glance

```text
                            React features
                                  |
                                  v
                    Transport-neutral core client
                   domain methods + contracts + errors
                                  |
              +-------------------+-------------------+
              |                   |                   |
              v                   v                   v
       Desktop transport    HTTP transport       Test transport
              |                   |                   |
              v                   |                   v
      Tauri/Rust command          |             deterministic fake
              |                   |
              +---------+---------+
                        |
                        v
                  Go application service
              domain rules + integrations + logs
                        |
                 +------+------+
                 |             |
                 v             v
              SQLite      MCP client services
                            external tools,
                         resources, and prompts
```

An MCP server adapter can face the other direction:

```text
AI host -> stdio MCP adapter -> Go task service -> SQLite/integrations
```

MCP stays at an edge in both cases. It does not become the internal domain model, UI API, persistence layer, or desktop IPC protocol.

## Keep the best Electron pattern

Electron’s recommended security pattern uses context isolation and a preload facade that exposes one deliberately scoped method per IPC capability. Electron specifically warns against passing the whole `ipcRenderer` API into web content. Its main and renderer processes then communicate asynchronously across IPC. See Electron’s documentation for [context isolation](https://www.electronjs.org/docs/latest/tutorial/context-isolation) and its [process model](https://www.electronjs.org/docs/latest/tutorial/process-model).

Our Tauri architecture deliberately mirrors that shape:

| Electron pattern | This architecture | What remains familiar |
| --- | --- | --- |
| Renderer feature code | React feature code | Components call application methods, not privileged APIs |
| Preload `contextBridge` facade | Transport-neutral core client | Narrow, typed, domain-named methods |
| `ipcRenderer.invoke()` | Tauri desktop transport | Promise-based request/response |
| `ipcMain.handle()` | Capability-scoped Rust command | Privileged validation and dispatch boundary |
| Node main-process modules | Go application service | Domain behavior and service integrations |
| Electron IPC payloads | Versioned JSON envelopes | Serializable DTOs and explicit errors |

That similarity matters for migration. Existing React components, TypeScript contracts, validation behavior, and frontend tests can move first. Electron preload methods can be reimplemented behind the core library one capability at a time. Node-specific main-process logic can then move behind the same interface into Go or narrowly scoped Rust commands.

### The implemented interface

React receives only the domain-shaped client:

```ts
client.tasks.list();
client.tasks.get(taskId);
client.tasks.create({ title: "Persist this task" });
client.tasks.update(taskId, { status: "done" });
client.tasks.delete(taskId);
```

Those methods build the versioned JSON envelope. The desktop adapter maps every envelope to one private Tauri invocation:

```ts
invoke("service_invoke", { request });
```

This is deliberately analogous to an Electron preload implementation calling `ipcRenderer.invoke("service:invoke", request)`. Neither React nor the task-core package receives the general Tauri `invoke` function. The adapter is the only frontend module that imports it.

On the privileged side, Rust implements the role normally held by `ipcMain.handle`. It:

1. accepts only protocol version 1 and the five `tasks.*` operations;
2. rejects oversized requests;
3. lazily starts the bundled target-specific Go binary;
4. gives that child a random per-launch token and an application-data SQLite path;
5. accepts readiness only from a loopback address using the expected protocol;
6. forwards the envelope to the fixed `/v1/invoke` endpoint with a timeout;
7. checks response size, JSON shape, protocol version, and request ID;
8. requests graceful shutdown and terminates the child when Tauri exits.

The browser adapter sends the identical envelope to `/task-service/v1/invoke`. In remote development, Vite first authenticates the browser through a generated capability URL and HTTP-only same-site cookie. It then proxies that relative route to the loopback Go server and adds a separate short-lived service token on the server side. The browser and an attached phone never receive the service token or the Go listener address.

The task screen pulls the authoritative list every five seconds and on an explicit Refresh action. Polls do not overlap, and a response that started before a local mutation cannot overwrite that mutation's UI result. This deliberately small synchronization mechanism can later be replaced by an event-driven invalidation channel without changing the task CRUD contract.

```text
Desktop
React -> TaskCoreClient -> DesktopTaskTransport -> service_invoke
      -> Rust-owned Go sidecar -> Go dispatcher -> SQLite

Remote development
React -> TaskCoreClient -> HttpTaskTransport -> Vite development proxy
      -> loopback Go service -> Go dispatcher -> SQLite
```

Both paths therefore reuse the same TypeScript request construction, Go validation, domain service, migrations, and repository. Only envelope delivery changes.

This supports a gradual migration instead of a flag-day rewrite:

1. Extract the renderer-facing API into the core library.
2. Put existing Electron IPC behind a temporary transport adapter.
3. Move domain and persistence behavior into the Go service.
4. Add the Tauri transport and Rust proxy.
5. Run shared contract fixtures against old and new transports.
6. Retire Electron-specific adapters after parity is demonstrated.

The core library makes progress measurable. Migration is complete when both implementations satisfy the same contract and user-visible scenarios—not when files have merely been moved.

## Why Tauri is a better fit here

These are advantages for this particular service-oriented architecture, not claims that Tauri is universally better than Electron.

### 1. The webview is not the backend runtime

Electron packages Chromium and Node.js with the application. That gives Electron a stable rendering target and a very mature JavaScript ecosystem, but it also makes those runtimes part of every shipped desktop application. Electron documents this bundling model directly in [Why Electron](https://www.electronjs.org/docs/latest/why-electron).

Tauri uses the operating system webview for presentation and a Rust application core for privileged behavior. In our design, the substantial backend is a separately compiled Go sidecar. That separation can reduce packaged runtime duplication and idle overhead, particularly for small applications, although installer size, startup time, and memory must be measured on every supported platform rather than assumed.

### 2. A smaller privileged surface

The React webview receives no Node.js runtime and no generic native bridge. It can request only named application operations. Tauri capabilities restrict command access, Rust validates the closed operation envelope, and Go validates the request again before executing domain behavior.

Tauri’s security model explicitly separates webview code from privileged core code and uses IPC capabilities to constrain access. See the official [Tauri security model](https://v2.tauri.app/security/).

This does not make the application automatically secure. The Rust core and Go service remain privileged software, and every boundary still requires input validation, authentication, size limits, timeouts, safe logging, and least-privilege configuration.

### 3. Go remains a real service, not desktop-only glue

Task rules, persistence, configuration, logging, and integrations live in Go rather than in the Tauri shell. The Go service can run:

- as a Tauri-managed loopback sidecar;
- as a standalone process during development and testing;
- behind an authenticated HTTP transport for remote development;
- under an MCP adapter without importing React or Tauri concepts.

Rust stays deliberately thin: lifecycle, capability enforcement, authenticated proxying, and desktop-native operations. We gain Rust’s strong boundary tooling without creating a second domain implementation.

### 4. Transport equivalence becomes testable

The core library owns domain-shaped methods, request construction, response validation, and normalized errors. A transport owns only delivery.

```ts
const client = createTaskCoreClient(transport);

const task = await client.tasks.create({ title: "Prove the boundary" });
await client.tasks.update(task.id, { status: "done" });
```

The same client can run over Tauri IPC, HTTP, or a deterministic test adapter. Contract fixtures can verify that every transport produces equivalent successes and failures. React tests inject a fake client instead of mocking desktop internals.

### 5. Browser and desktop experiences share feature code

Remote development is an adapter choice, not a fork of the application. Components never ask whether they are running in Tauri and never call `fetch` directly. The composition root selects the desktop or HTTP transport, while task features remain unchanged.

This creates a useful path for browser-based development, mobile viewport testing, support tooling, and future hosted surfaces without weakening the packaged desktop trust boundary.

### 6. Native capabilities stay intentional

The desktop shell can still add operating-system integrations, but each one becomes a reviewed capability rather than an ambient Node.js privilege. The application can expose a narrow “open file,” “show notification,” or “manage sidecar” operation while keeping raw shell, filesystem, and network primitives away from feature code.

### 7. Failure and lifecycle semantics are explicit

The sidecar protocol defines readiness, authentication, timeouts, shutdown, response limits, and typed failure states. This is more work than calling a local JavaScript function, but it creates operational clarity:

- the UI knows whether the service is starting, ready, unavailable, or stopped;
- one request ID can be followed through TypeScript, Rust, and Go logs;
- application exit owns sidecar shutdown;
- SQLite has one accountable process owner;
- raw transport and database errors do not leak into the UI.

Those properties become increasingly valuable when the backend begins managing AI services, external tools, and long-running operations.

## The core library is the portability layer

The current `@tauri-go-service-example/task-core-library` package already demonstrates the intended boundary. It owns:

- domain-facing TypeScript contracts;
- operation names and protocol version;
- correlation-aware request construction;
- untrusted response validation;
- stable client errors;
- a transport interface supplied by the host application.

It deliberately does not own React, Tauri, HTTP, authentication, SQLite, process lifecycle, or MCP sessions.

This is the architectural center of gravity. UI components, automation, command-line tools, and MCP adapters can share the same application semantics without sharing their delivery mechanisms.

The package should remain small. New abstractions belong there only when at least two real consumers need the same behavior. Service-specific policy remains in Go, and presentation-specific behavior remains in React.

## How MCP fits

MCP defines a host-client-server architecture. A host manages one or more MCP clients; each client maintains an isolated connection to a server, while servers expose focused tools, resources, and prompts. The protocol uses capability negotiation so participants explicitly advertise what they support. See the official [MCP architecture specification](https://modelcontextprotocol.io/specification/2025-06-18/architecture).

That model complements this architecture when MCP is treated as an adapter.

### Consuming MCP servers

The Go application service can host MCP client services for focused external capabilities. For example, one client might connect to a document server and another to a task-automation server. The Go layer owns connection lifecycle, credentials, capability negotiation, timeouts, consent policy, and audit logging.

Those MCP clients should expose normalized application ports to domain code. Raw MCP tool names, JSON-RPC messages, and vendor-specific result shapes should not flow into React components or become the core library’s public contract.

```text
React -> core client -> Go use case -> MCP client adapter -> external MCP server
```

This lets the application change MCP SDKs, transports, or servers without rewriting the UI.

### Exposing application capabilities through MCP

If an AI host needs to operate the task service, a separate MCP server command translates model-facing tool arguments into calls on the same Go application service used by the HTTP bridge:

```text
MCP tool `tasks_create`
        -> validate MCP input
        -> Go domain service
        -> SQLite repository
```

The current `task-mcp` executable uses this direct Go composition because it is a local stdio process and the Go service is the canonical owner of domain rules. A future remote MCP gateway may instead use the TypeScript core client and an authenticated backend transport. Both shapes keep MCP at the edge; neither duplicates task behavior in the adapter.

MCP tools define JSON Schema inputs and may define structured output schemas, which aligns well with our schema-validated application contract. See the MCP specification for [tools and schemas](https://modelcontextprotocol.io/specification/2025-11-25/server/tools).

The MCP adapter still has responsibilities of its own:

- expose only intentionally approved operations;
- validate tool input and service output;
- preserve correlation IDs and audit events;
- require confirmation for consequential actions;
- enforce authorization independently of descriptive tool metadata;
- translate domain errors into safe MCP results;
- avoid granting a model generic IPC, HTTP, SQL, filesystem, or shell access.

The application contract and MCP contract may share generated data types, but they are not automatically the same public API. MCP descriptions and schemas are optimized for discovery and model use; the application client is optimized for deterministic product behavior.

## Honest tradeoffs

Choosing Tauri shifts rather than eliminates complexity.

| Consideration | Tauri/Go consequence |
| --- | --- |
| Rendering consistency | System webviews differ by operating system; browser and packaged-webview testing are mandatory |
| Team skills | Rust, Go, cross-compilation, and platform packaging add toolchains beyond TypeScript |
| Ecosystem | Electron’s Node/npm desktop ecosystem is more mature; Tauri plugins and custom Rust may require more integration work |
| Sidecar packaging | A Go binary must be built, named, bundled, signed, and verified for every supported target |
| IPC complexity | Serialization, readiness, authentication, cancellation, and shutdown must be explicitly designed |
| Security | Smaller capabilities help, but unsafe commands, broad permissions, or weak service authentication can erase the advantage |
| Updates | Electron controls its bundled Chromium version; Tauri depends partly on operating-system webview availability and update policy |

Electron remains a strong choice when uniform Chromium behavior, the Node ecosystem, or a JavaScript-only team outweighs runtime footprint and service separation. Our choice favors a narrower desktop shell, a reusable Go backend, explicit contracts, and multiple delivery surfaces.

## What success looks like

The architecture earns its keep when we can demonstrate all of the following:

- The same CRUD scenario passes through desktop IPC and remote HTTP.
- React feature code imports neither Tauri APIs nor backend HTTP helpers.
- An Electron-compatible adapter and Tauri adapter can pass the same contract fixtures during migration.
- A packaged application starts the correct Go binary, waits for readiness, persists data, and leaves no orphan process.
- One correlation ID appears coherently in TypeScript, Rust, Go, and MCP-related audit events.
- Browser-safe configuration never contains service tokens, database paths, or MCP credentials.
- An MCP client integration can be replaced without changing React features or Go domain rules.
- An optional MCP server exposes approved application capabilities without bypassing the core contract.

## Current implementation boundary

Today, the transport-neutral task core library, responsive Go-backed UI, desktop and browser transports, Tauri sidecar lifecycle and proxy, authenticated Go HTTP bridge, structured Go logging, configuration loading, SQLite repository, migrations, target-aware sidecar build, and local Go stdio MCP adapter are implemented.

TypeScript adapter tests, Rust boundary tests, Go HTTP integration tests, and SQLite restart tests cover the major seams. Automated packaged-application UI tests on every desktop target, full cross-transport fixture parity, frontend log forwarding, sidecar crash notification, and remote or TypeScript-hosted MCP adapters remain future work. Cross-platform bundling is configured but must still be demonstrated by the CI matrix before it is considered verified.

## Related project documents

- [Root project README](README.md)
- [MCP usage guide](MCP-USAGE.md)
- [ADR 0001: Electron-style typed service bridge](.agents/decisions/0001-electron-style-service-bridge.md)
- [Service bridge POC specification](.agents/specs/service-bridge-poc.md)
- [Tauri application patterns](.agents/patterns/tauri.md)
- [TypeScript and React patterns](.agents/patterns/typescript-react.md)
- [Task core library](packages/node/task-core-library/README.md)
