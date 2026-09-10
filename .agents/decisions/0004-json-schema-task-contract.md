# ADR 0004: JSON Schema task contract

- Status: accepted
- Date: 2026-09-10

## Context

Task requests and responses cross TypeScript, Rust, HTTP, and Go boundaries. Hand-maintaining equivalent data-transfer types in each language would allow field names, nullability, operation names, limits, and protocol versions to drift.

The Go domain also needs semantics that a general type generator cannot represent safely. In particular, task updates distinguish a missing `description` from an explicit JSON `null`.

## Decision

Use [`contracts/task-api.schema.json`](../../contracts/task-api.schema.json) as the canonical task wire contract. It uses JSON Schema Draft 2020-12 and defines the five task operations, their payloads, task response data, stable errors, and the versioned request and response envelopes.

Generate the TypeScript data-transfer types into `packages/node/task-core-library/src/generated`. Generated files are committed, never edited by hand, and checked for drift during type checking.

Keep the idiomatic Go domain and bridge structs handwritten. A Go contract test compiles the same canonical schema and validates JSON produced by those structs, including explicit-null updates and RFC 3339 timestamps. This preserves Go domain behavior while making schema compatibility executable.

Runtime constants needed by TypeScript, such as task statuses and operation names, remain small exported values. A test compares them directly with the canonical schema enums.

## Consequences

- One language-neutral artifact documents task JSON and produces the TypeScript DTOs.
- Go can retain `time.Time` and its explicit missing-versus-null update representation.
- Schema, generated TypeScript, runtime constants, and Go JSON shapes fail tests when they drift.
- Domain validation still remains in Go; JSON Schema does not replace business rules such as trimming whitespace.
- System health and public-configuration operations remain outside this task-specific schema.
