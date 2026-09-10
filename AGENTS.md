# Project Agent Guide

This file defines the repository-wide working agreement for people and coding agents. Read it before changing the project.

## Project intent

Build the smallest clear reference application that demonstrates:

- a React and TypeScript UI inside a Tauri desktop shell;
- a typed application client inspired by Electron preload/backend patterns;
- an embedded Go task service;
- SQLite persistence owned by the Go service;
- equivalent desktop IPC and remote-development HTTP transports;
- centralized, schema-validated configuration;
- structured, correlation-aware logging.

The current repository is a scaffold. Do not describe planned behavior as implemented or verified.

## Repository map

- `apps/task-app`: React, TypeScript, Vite, Vaadin Web Components, and the standard `src-tauri` Rust project.
- `services/go/task-service`: task domain logic, API adapters, and SQLite persistence.
- `packages/node/task-core-library`: reusable task contracts, CRUD client, bridge parsing, and normalized client errors.
- `packages/go`: reusable Go packages that are not owned by one service.
- `config`: shared non-secret configuration and its schema.
- `.agents/patterns`: accepted implementation patterns and conventions.
- `.agents/specs`: feature and system specifications.
- `.agents/decisions`: short architecture decision records.

Language and framework-specific guidance lives under `.agents/patterns`. Read the relevant Go, TypeScript/React, and Tauri pattern before implementing in that area.

More specific `AGENTS.md` files may be added inside subprojects. The nearest file governs when instructions differ.

## Working rules

1. Keep domain behavior in Go and presentation behavior in React.
2. React features call the typed application client. They do not call Tauri APIs or `fetch` directly.
3. Keep IPC and HTTP behavior contract-equivalent and test both adapters.
4. Treat the API/config schema as canonical; generate or validate TypeScript and Go representations from it.
5. Expose frontend configuration through an explicit browser-safe allowlist. Never send service secrets to the UI.
6. Keep SQLite access behind the Go repository layer and apply ordered migrations before serving requests.
7. Propagate correlation IDs across frontend, Tauri, Go, and persistence-related logs.
8. Do not bind remote development servers beyond loopback without explicit opt-in and an access-control mechanism.
9. Prefer small, domain-named packages over generalized framework code.
10. Update the relevant spec, pattern, or decision record when a change alters an agreed boundary.

## Change workflow

Before editing:

- read the root README and relevant files under `.agents`;
- inspect `git status` and preserve unrelated changes;
- identify which application, service, package, or contract owns the change.

Before handing off:

- run the smallest relevant tests and builds;
- run formatting and static checks for each changed language;
- run `git diff --check`;
- report separately what changed, what was validated, and what remains unverified.

Do not commit generated binaries, SQLite databases, secrets, local environment files, or machine-specific application data.
