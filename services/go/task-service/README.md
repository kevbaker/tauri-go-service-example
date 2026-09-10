# Task Service

The task service owns task validation, CRUD operations, the versioned bridge, configuration loading, structured logging, ordered migrations, and SQLite persistence.

## Run

Desktop-sidecar mode binds an operating-system-assigned loopback port and requires a per-launch token:

```bash
TGS_SERVICE_TOKEN="development-only-secret" \
  go run ./cmd/task-service --database-path ./data/tasks.db
```

Remote-development mode must be selected explicitly and uses a separate token:

```bash
TGS_REMOTE_ACCESS_TOKEN="development-only-secret" \
  go run ./cmd/task-service --config ../../../config/development.yaml
```

The process emits one JSON readiness record on stdout. Logs are written to stderr. The authenticated endpoints are:

- `POST /v1/invoke` for the versioned application envelope;
- `GET /health` for process readiness;
- `POST /v1/shutdown` for bounded graceful shutdown.

Request bodies are limited to 64 KiB and encoded responses to 512 KiB. All endpoints require a bearer token. Browser requests additionally pass the configured origin allowlist.

## Validate

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
```

The Tauri lifecycle integration and remote-development launcher remain outside this module and are not yet implemented.
