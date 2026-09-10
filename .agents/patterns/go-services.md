# Go Service Patterns

These patterns apply to `services/go/task-service` and to shared Go code under `packages/go`. They favor standard-library facilities and a small service over framework-driven architecture.

## Module and package layout

The task service is a self-contained Go module and executable:

```text
services/go/task-service/
├── cmd/task-service/main.go
├── internal/
│   ├── app/                 # composition and lifecycle
│   ├── bridge/              # HTTP envelope and operation dispatch
│   ├── config/              # load and validate service configuration
│   ├── task/                # domain types and behavior
│   └── sqlite/              # migrations and task repository
├── migrations/              # embedded ordered SQL migrations
├── go.mod
└── go.sum
```

- Keep `main` small: parse startup inputs, build dependencies, run, and map shutdown to an exit status.
- Keep service-private packages under `internal`; Go enforces that boundary.
- Use packages named for their responsibility or domain, not generic buckets such as `utils`, `common`, or `helpers`.
- Add code to `packages/go` only after a concrete second consumer exists.
- Avoid a package per architectural noun when a few cohesive files in one package are clearer.

## Dependencies and interfaces

- Construct dependencies explicitly in the composition root. Avoid mutable package globals and service locators.
- Accept interfaces where behavior must vary; return concrete types by default.
- Define small interfaces next to the consumer rather than publishing broad repository interfaces from the implementation package.
- Keep domain and application behavior independent of HTTP, SQLite, Tauri, and process lifecycle types.
- Prefer the standard library until another dependency removes meaningful complexity.
- Commit `go.mod` and `go.sum`; use `go mod tidy` rather than editing dependency checksums manually.

## Context, cancellation, and lifecycle

- Pass `context.Context` as the first parameter of operations that can block or cross a process/database boundary.
- Never store a context in a struct. Derive request-scoped contexts and propagate cancellation to SQLite and downstream calls.
- Set explicit startup, readiness, request, graceful-shutdown, and forced-shutdown timeouts.
- Handle termination once at the application boundary. Stop accepting work, finish bounded in-flight work, close the database, then exit.
- Reserve stdout for the documented sidecar readiness/control protocol and write service logs to stderr.

## Errors and validation

- Validate untrusted input at the bridge boundary and enforce domain invariants again in domain constructors or operations.
- Return errors; do not panic for malformed input, database failures, or expected environmental conditions.
- Wrap errors with context using `%w` and inspect causes with `errors.Is` or `errors.As`.
- Map internal errors to the stable bridge error codes in one boundary package. Do not expose SQL text, filesystem paths, stack traces, or bearer tokens.
- Do not use error-message text as program logic.

## HTTP bridge

- Expose only the versioned bridge, health, and bounded shutdown endpoints required by the POC.
- Configure `http.Server` read-header, read, write, and idle timeouts; do not use an unconfigured package-level server.
- Limit request bodies before decoding, reject unknown JSON fields, require one JSON value, and cap response size.
- Require the desktop per-launch token or remote-development token using constant-time comparison.
- Bind desktop mode to `127.0.0.1:0`; never infer permission to bind externally from a development environment name.
- Return one response envelope for expected application failures. Reserve non-success HTTP status codes for failures where an envelope cannot be processed, such as authentication, media type, or body-size rejection.
- Treat operation names as a closed set and make dispatch exhaustive.

## SQLite

- Open and configure SQLite in one package; inject repository behavior into application services.
- Enable foreign-key enforcement for every connection.
- Apply embedded, ordered migrations before emitting readiness.
- Use parameterized statements and explicit column lists. Never build SQL from request strings.
- Use transactions for changes that must succeed atomically and always check commit/rollback errors.
- Store timestamps as RFC 3339 UTC values and convert at the repository boundary.
- Map uniqueness and missing-row outcomes to domain errors without exposing driver-specific errors.
- Decide journal mode, busy timeout, connection limits, and backup behavior explicitly before treating persistence as production-ready.
- Use a new temporary database per repository test; do not share the developer database with tests.

## Logging

- Use `log/slog` unless a demonstrated requirement needs another logger.
- Configure the logger once and inject it. Use JSON on the sidecar stderr stream and readable text only for an explicitly selected local mode.
- Include stable keys such as `component`, `event`, `requestId`, `operation`, `durationMs`, and `errorCode`.
- Log at process and operation boundaries. Avoid duplicating the same error at every call frame.
- Never log authorization values, complete configuration, raw request bodies, task descriptions, or other user content by default.

## Concurrency

- Start goroutines only with a named owner, cancellation path, and completion strategy.
- Protect shared state explicitly and run concurrency-sensitive tests with the race detector.
- Do not add worker pools, channels, or background queues until the POC has a concrete concurrent workload.
- Keep SQLite concurrency assumptions explicit; do not treat a successful single-user test as proof of multi-writer behavior.

## Tests and checks

- Keep tests beside the package they exercise and use table-driven cases when several inputs share behavior.
- Prefer small fakes over general mocking frameworks.
- Test public behavior and error classification rather than private implementation calls.
- Add integration tests for migrations, restart persistence, invalid tokens, malformed envelopes, cancellation, and shutdown.
- Run at minimum:

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
```

Run `govulncheck ./...` in CI once the module and dependency policy are established.

## Sources

- [Go: Organizing a Go module](https://go.dev/doc/modules/layout)
- [Go: How to Write Go Code](https://go.dev/doc/code)
- [Go: Structured Logging with slog](https://go.dev/blog/slog)
- [Go package documentation: context](https://pkg.go.dev/context)
- [Go package documentation: errors](https://pkg.go.dev/errors)
- [Go package documentation: net/http](https://pkg.go.dev/net/http)
- [Go: Data Race Detector](https://go.dev/doc/articles/race_detector)
