# TypeScript and React Patterns

These patterns apply to the frontend in `apps/task-app` and reusable TypeScript packages under `packages/node`.

## Project boundaries

Use a feature-oriented application layout:

```text
src/
├── app/                     # providers, startup, routing, global shell
├── bridge/                  # transport construction and selection
├── features/tasks/          # task UI and hooks
├── components/              # genuinely shared presentation components
├── config/                  # browser-safe runtime config access
└── main.tsx
```

- React task features depend on `TaskCoreClient` from `packages/node/task-core-library`, never directly on Tauri `invoke` or backend `fetch`.
- Keep transport implementations behind the bridge package.
- Keep shared contract types and domain-named client functions in a dedicated workspace package. When contract generation is added, do not edit generated output by hand.
- Keep shared core packages free of React, design-system, Tauri, HTTP, and persistence dependencies.
- Do not create a shared component or utility until more than one concrete caller needs it.
- Avoid barrel files that hide ownership or introduce import cycles.

## TypeScript configuration

Enable strict checking and keep it enabled:

```json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": true,
    "noImplicitOverride": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true,
    "useUnknownInCatchVariables": true
  }
}
```

- Use `unknown` for untrusted input and caught failures; narrow it deliberately.
- Do not use `any`, non-null assertions, or unchecked type casts to silence boundary errors.
- Distinguish an absent optional field from an explicit `null` in contracts.
- Use string literal unions or generated enums and exhaustive `switch` checks for operations, states, and error codes.
- Prefer `type` for data shapes and unions; use `interface` where declaration merging or an explicitly implementable object contract is useful.
- TypeScript types do not validate runtime JSON. Parse bridge responses with the generated/runtime schema before using them.

## Typed application client

- Put task DTOs, operation names, runtime response parsing, normalized errors, and CRUD methods in `@tauri-go-service-example/task-core-library`.
- Expose domain methods such as `tasks.create(input)`, not a general `send(channel, payload)` API to feature code.
- Keep the generic wire operation inside transport implementations.
- Return promises for all process-boundary operations; never emulate synchronous IPC.
- Normalize both Tauri and HTTP failures into the same `AppError` discriminated union.
- Generate `requestId` once in the client and preserve it through retries, logs, and error presentation. The first POC should not retry mutations automatically.
- Make transport selection explicit at application composition time. Do not scatter runtime checks throughout features.

## React

- Keep components pure and keep hooks at the top level.
- Put remote/service state in feature hooks; keep transient visual state local to the component that owns it.
- Represent loading, empty, success, validation-error, unavailable, and unexpected-error states deliberately.
- Avoid mirroring derived values into state or using effects for computations that belong in render/event handlers.
- Clean up subscriptions and async work. Ignore or cancel stale responses after unmount or query changes.
- Place an error boundary around the application shell, but handle expected `AppError` values within the relevant feature.
- Use semantic HTML for page structure and Vaadin Web Components through `@vaadin/react-components` for interactive controls. Preserve keyboard access, visible focus, labels, and touch-sized targets.
- Import the shared Lumo theme once at the application entrypoint. Prefer Lumo design tokens over copied color, spacing, radius, and typography values.
- Verify layouts at narrow mobile, normal desktop, zoomed text, empty data, long content, and error states.

## Configuration and security

- Read runtime values only through `config.getPublic()` and a typed configuration provider.
- Treat every value available to frontend JavaScript as public. Never embed service tokens, database paths, or secrets in Vite variables.
- Do not render backend-provided HTML. Treat URLs and future native-open operations as untrusted input requiring scheme validation.
- Keep the desktop Content Security Policy compatible with the smallest required set of sources.

## Logging and errors

- Log stable event names and metadata, not whole objects or user-entered task content.
- Include `requestId`, `operation`, UI surface, and normalized `errorCode` when available.
- Present actionable, user-safe messages; retain technical diagnostics in structured logs.
- Do not expose Rust, Go, HTTP, or SQLite implementation details in component-level error handling.

## Tests and checks

- Unit-test contract parsing, transport error normalization, and configuration selection.
- Component-test behavior through an injected fake `TaskCoreClient`; do not mock Tauri imports in every feature test.
- Run the same contract fixtures against desktop and HTTP transports.
- Use end-to-end tests only for the cross-process flows that unit/component tests cannot prove.
- Run formatting, linting, type checking, unit tests, production build, and the relevant end-to-end smoke test before handoff.

## Sources

- [TypeScript: `strict`](https://www.typescriptlang.org/tsconfig/strict.html)
- [TypeScript: `noUncheckedIndexedAccess`](https://www.typescriptlang.org/tsconfig/noUncheckedIndexedAccess.html)
- [TypeScript: `exactOptionalPropertyTypes`](https://www.typescriptlang.org/tsconfig/exactOptionalPropertyTypes.html)
- [React: Rules of React](https://react.dev/reference/rules)
- [React: You Might Not Need an Effect](https://react.dev/learn/you-might-not-need-an-effect)
- [React: Rules of Hooks](https://react.dev/reference/rules/rules-of-hooks)
- [Vite: Env Variables and Modes](https://vite.dev/guide/env-and-mode)
