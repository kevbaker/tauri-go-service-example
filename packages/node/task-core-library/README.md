# Task Core Library

Transport-neutral TypeScript contracts and CRUD client for the task proof of concept. Its data-transfer types are generated from the repository's canonical [task JSON Schema](../../../contracts/task-api.schema.json).

The package owns:

- task domain types and status values;
- the `tasks.list`, `tasks.get`, `tasks.create`, `tasks.update`, and `tasks.delete` operations;
- construction of protocol-versioned, correlation-aware bridge requests;
- validation of backend response envelopes and task data;
- normalized `TaskClientError` values.

It does not own React, Tauri, HTTP, SQLite, authentication, or process lifecycle. An application supplies a `TaskBackendTransport` and chooses whether that transport invokes Tauri, HTTP, an in-memory preview, or a test fake.

```ts
import { createTaskCoreClient } from "@tauri-go-service-example/task-core-library";

const client = createTaskCoreClient(transport);
const tasks = await client.tasks.list({ status: "todo" });
const created = await client.tasks.create({ title: "Prove the bridge" });
await client.tasks.update(created.id, { status: "done" });
await client.tasks.delete(created.id);
```

This boundary is also suitable for a future MCP server: MCP tools can translate tool arguments into these task client calls while transport and persistence remain outside the MCP adapter.

After changing the schema, regenerate and verify the committed TypeScript bindings from the repository root:

```bash
npm run contract:generate
npm run contract:check
```

Do not edit files under `src/generated` by hand. Runtime response parsing remains explicit because TypeScript types alone do not validate untrusted process-boundary data.
