# Task App

This directory is the root of the React, TypeScript, Vite, Vaadin Web Components, and Tauri application.

It follows Tauri's [standard project structure](https://v2.tauri.app/start/project-structure/): the JavaScript application lives at this level and the Rust application core lives under `src-tauri`.

```text
task-app/
├── package.json
├── index.html
├── src/                      # React and TypeScript source
└── src-tauri/
    ├── Cargo.toml
    ├── Cargo.lock
    ├── build.rs
    ├── tauri.conf.json
    ├── capabilities/
    ├── icons/
    └── src/
        ├── main.rs
        └── lib.rs
```

The baseline was generated on 2026-09-09 with the official `create-tauri-app` 4.7.4 React/TypeScript template and Tauri 2 option. Dependency manifests retain the template's current compatible version ranges; the root lockfile records the versions actually installed for reproducible builds.

The task UI performs full CRUD through the production `TaskCoreClient`. Task types and request construction live in [`packages/node/task-core-library`](../../packages/node/task-core-library/README.md). The application composition root selects a narrow Tauri transport in the desktop webview and an HTTP transport in an ordinary browser. Task components import neither Tauri APIs nor HTTP helpers.

React features must use the typed application client from `packages/node` rather than importing Tauri APIs or calling backend HTTP endpoints directly. The Rust core remains a thin lifecycle, validation, and proxy layer around the Go task service.

See the repository [Tauri patterns](../../.agents/patterns/tauri.md) and [service bridge specification](../../.agents/specs/service-bridge-poc.md) before implementing the app.

## Browser development

Run the complete remote-development stack from the repository root:

```bash
npm run dev:remote
```

This is an explicit opt-in that starts the loopback Go service and binds Vite to all local interfaces. The command prints a browser-access token; append it as `?access_token=...` to a printed Vite Network URL on first use. Vite exchanges it for an HTTP-only, same-site session cookie and then adds a separate generated service token while proxying requests to Go. Neither credential is compiled into or returned to frontend JavaScript. Do not expose Vite to an untrusted or public network.

The task list refreshes from Go every five seconds and also provides a Refresh button. Polling requests never overlap, and stale list responses cannot overwrite a task mutation that completed while a refresh was running.

`npm run dev` inside this directory starts Vite only and is useful for frontend failure-state work; task calls require the complete stack above.

To run the desktop application, use:

```bash
npm run tauri --workspace task-app -- dev
```

That command builds a target-triple-named Go binary, bundles it as a Tauri sidecar, and persists tasks in the platform application-data directory.
