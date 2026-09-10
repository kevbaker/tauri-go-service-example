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

This is an executable framework scaffold, not the completed service-bridge POC. The generated Rust greeting remains temporary until the bridge implementation replaces it.

The task UI currently implements browser-safe CRUD through an in-memory `TaskCoreClient`. Task types and the production CRUD client live in [`packages/node/task-core-library`](../../packages/node/task-core-library/README.md). The preview intentionally resets on reload and is visibly labeled as in-memory. Task components do not import Tauri APIs or call HTTP directly, so the composition root can replace the preview with `createTaskCoreClient` plus the future desktop or browser transport.

React features must use the typed application client from `packages/node` rather than importing Tauri APIs or calling backend HTTP endpoints directly. The Rust core remains a thin lifecycle, validation, and proxy layer around the Go task service.

See the repository [Tauri patterns](../../.agents/patterns/tauri.md) and [service bridge specification](../../.agents/specs/service-bridge-poc.md) before implementing the app.

## Browser development

Run the frontend without the desktop shell:

```bash
npm run dev
```

To make the development UI reachable from another device on the same network, run this from the repository root:

```bash
npm run dev:remote
```

This is an explicit opt-in that binds Vite to all local interfaces. Do not expose it to an untrusted or public network. The current Vaadin CRUD UI renders normally in a browser. Once service calls are implemented, browser mode will use the specified HTTP transport because Tauri commands are unavailable outside the desktop webview.

To inspect the production frontend bundle instead, build it and run:

```bash
npm run build --workspace task-app
npm run preview:remote
```
