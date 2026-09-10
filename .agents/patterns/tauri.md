# Tauri Application Patterns

These patterns apply to the Tauri project rooted at `apps/task-app` and its Rust core under `apps/task-app/src-tauri`.

## Standard project structure

Follow Tauri's standard split: the JavaScript project is the application root and the Rust project lives in `src-tauri`.

```text
apps/task-app/
├── package.json
├── index.html
├── src/
│   └── main.tsx
└── src-tauri/
    ├── Cargo.toml
    ├── Cargo.lock
    ├── build.rs
    ├── tauri.conf.json
    ├── capabilities/
    │   └── default.json
    ├── icons/
    └── src/
        ├── main.rs
        └── lib.rs
```

- Treat `src-tauri` as a normal Cargo project with Tauri-specific configuration.
- Keep desktop `main.rs` as the generated thin entrypoint that calls the library `run` function. Put application setup and commands in `lib.rs` or modules called by it so the structure remains compatible with mobile entrypoints.
- Store capabilities in `src-tauri/capabilities` and generated icon assets in `src-tauri/icons`.
- Keep the Go source under `services/go/task-service`; only target-specific built sidecars are copied into the Tauri binary staging directory.

## Rust core responsibility

Rust is a privileged adapter, not the domain backend. It owns:

- Tauri application setup and shutdown;
- Go sidecar start, readiness, managed state, and termination;
- validation of the bridge envelope and operation allowlist;
- fixed-destination authenticated proxying to Go;
- translation of process and transport failures into stable bridge errors;
- small desktop-native capabilities deliberately added to the contract.

Rust does not own task validation, SQLite, migrations, or duplicate Go services.

## Commands and state

- Prefer asynchronous Tauri commands for process, HTTP, or filesystem work. Do not block the UI thread.
- Commands accept and return explicit Serde DTOs. Keep JSON field naming aligned with the canonical camel-case contract.
- Return a serializable application error type; never return formatted internal errors as the API contract.
- Register only deliberate commands and constrain their availability with capabilities.
- Keep `service_invoke` generic only inside the closed bridge envelope: deserialize `operation` as an enum, reject unknown versions, and never accept a destination URL from the webview.
- Store the sidecar handle, fixed address, secret, and readiness state in managed Rust state with explicit synchronization. Do not use mutable statics.
- Apply request and response size limits and per-call timeouts on both sides of the proxy.

## Capabilities and webview security

- Grant the main webview only the commands and plugin permissions it uses.
- Scope filesystem, shell, dialog, opener, and HTTP permissions narrowly; do not add broad defaults for convenience.
- Do not grant shell sidecar execution directly to frontend JavaScript. Rust owns sidecar launch so it can control arguments, secrets, readiness, and shutdown.
- Load packaged local frontend assets in production. Do not grant remote origins access to Tauri commands for the remote-development web UI.
- Keep normal Vite development bound to loopback. Use the explicit `dev:remote` script for LAN viewing and never set `server.allowedHosts` or CORS to unrestricted values.
- Set a restrictive Content Security Policy and do not weaken it to solve development-only tooling issues.
- Treat the TypeScript client as ergonomics, not enforcement; validate every privileged request again in Rust and Go.

## Sidecar lifecycle

- Declare the Go executable with `bundle.externalBin` and produce the required target-triple-suffixed binary for each supported platform.
- Pass only validated arguments and child-specific environment values.
- Keep the per-launch bearer token out of command arguments, logs, frontend configuration, and error messages.
- Wait for the versioned readiness record before marking the bridge ready. Bound startup time and capture early process exit.
- Consume stdout and stderr continuously to avoid child-process pipe backpressure. Reserve stdout for control records and route stderr into structured application logging.
- On application exit, attempt bounded graceful shutdown, then terminate and reap the child.
- Do not add automatic restart until its effect on mutations and SQLite ownership is explicitly designed.

## Events and channels

- Use commands for typed request/response operations.
- Use a webview-scoped event for infrequent lifecycle state changes only when polling `system.health()` is insufficient.
- Wrap event subscription in a typed TypeScript method that removes internal Tauri event objects before invoking application callbacks.
- Use channels for ordered or higher-throughput streams. Do not simulate request/response with paired global events.
- Always return or invoke an unsubscribe function when a React subscriber unmounts.

## Build and test

- Build the Go sidecar for the exact Rust target before `tauri build` and fail clearly when the binary is absent.
- Keep development and packaged sidecar discovery paths explicit; do not depend on the developer shell's working directory.
- Unit-test envelope validation and error mapping as ordinary Rust code.
- Test commands with a controlled fake Go endpoint and test lifecycle behavior with a disposable child fixture.
- Verify the packaged application, not only `tauri dev`: startup, readiness, CRUD, restart persistence, clean exit, and absence of an orphan process.
- Verify each supported target independently, including code signing and application-data paths.

## Sources

- [Tauri: Project Structure](https://v2.tauri.app/start/project-structure/)
- [Tauri: Inter-Process Communication](https://v2.tauri.app/concept/inter-process-communication/)
- [Tauri: Calling Rust from the Frontend](https://v2.tauri.app/develop/calling-rust/)
- [Tauri: Calling the Frontend from Rust](https://v2.tauri.app/develop/calling-frontend/)
- [Tauri: Permissions](https://v2.tauri.app/security/permissions/)
- [Tauri: Embedding External Binaries](https://v2.tauri.app/develop/sidecar/)
