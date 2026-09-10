package migrations

import "embed"

// Files contains the service's ordered SQLite migrations.
//
//go:embed *.sql
var Files embed.FS
