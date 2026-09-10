CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'done')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX tasks_created_at_idx ON tasks (created_at DESC, id DESC);
CREATE INDEX tasks_status_created_at_idx ON tasks (status, created_at DESC, id DESC);
