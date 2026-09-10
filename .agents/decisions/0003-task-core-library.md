# ADR 0003: Shared task core client library

- Status: accepted
- Date: 2026-09-10

## Context

The React task feature initially owned its task types and `AppClient` contract. Those definitions are also needed by desktop and HTTP transports, contract tests, and a possible future MCP server. Keeping them inside the React application would either couple non-UI consumers to the app or cause each adapter to recreate the task API.

Go remains the source of task domain rules and owns SQLite persistence. The shared TypeScript code needs to describe and safely call that backend, not reproduce its domain service.

## Decision

Create `packages/node/task-core-library` as a framework- and transport-neutral workspace package.

The package owns:

- task data-transfer types, inputs, filters, statuses, and operation names;
- the `TaskCoreClient` and its `list`, `get`, `create`, `update`, and `delete` methods;
- versioned request-envelope construction and request IDs;
- runtime validation of response envelopes and task data;
- normalized, correlation-aware `TaskClientError` values.

The package accepts a narrow `TaskBackendTransport`. Applications provide Tauri IPC, HTTP, or test implementations. It contains no React, Vaadin, Tauri, HTTP, authentication, Go-process, SQLite, or MCP dependencies.

The React app imports its task contracts from this package. Its temporary in-memory client implements the same `TaskCoreClient` for UI development; production adapters will use `createTaskCoreClient(transport)`.

A future MCP server may depend on this package and map MCP tools to the same task client methods. MCP will be an adapter at the edge and will not own task validation or persistence.

## Consequences

- UI, desktop, HTTP, tests, and a future MCP adapter can share one TypeScript task API.
- Transport implementations deal with one generic invocation primitive while consumers receive domain-named methods.
- Runtime responses are checked before untrusted backend data reaches consumers.
- The manually maintained TypeScript and Go contract still needs a canonical schema or generated compatibility tests before the POC is complete.
- This package is private and source-consumed inside the monorepo; publishing and a compiled distribution format are deferred until there is a real external consumer.
