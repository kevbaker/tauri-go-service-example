# Contracts

`task-api.schema.json` is the canonical JSON wire contract for task CRUD. It uses JSON Schema Draft 2020-12 and owns task DTOs, request payloads, operation names, response envelopes, and application error shapes.

After editing it, regenerate and check the committed TypeScript bindings:

```bash
npm run contract:generate
npm run contract:check
```

The Go service intentionally keeps handwritten domain and bridge types. Its contract tests compile this same schema and validate JSON marshaled from those types, preserving Go-specific behavior such as the difference between an omitted update field and an explicit `null`.
