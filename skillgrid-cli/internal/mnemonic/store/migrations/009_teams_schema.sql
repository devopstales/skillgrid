-- 009: Hybrid agent teams control-plane tables (metadata + file paths only).
-- Content lives under {project}/.skillgrid/files/; SQLite stores paths/status.
-- Additive only — does not rewrite observations or existing mnemonic tables.

CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS team_members (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    role TEXT NOT NULL,
    system_prompt_path TEXT,
    model TEXT,
    status TEXT NOT NULL DEFAULT 'idle',
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    parent_task_id TEXT,
    title TEXT NOT NULL,
    brief_path TEXT NOT NULL,
    output_path TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    assigned_to TEXT,
    created_by TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    memory_context_path TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    completed_at TEXT,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_task_id) REFERENCES tasks(id),
    FOREIGN KEY (assigned_to) REFERENCES team_members(id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_priority
    ON tasks (status, priority DESC, created_at ASC);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    from_agent TEXT NOT NULL,
    to_agent TEXT NOT NULL,
    subject TEXT,
    body_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unread',
    in_reply_to TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    read_at TEXT,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (from_agent) REFERENCES team_members(id),
    FOREIGN KEY (to_agent) REFERENCES team_members(id),
    FOREIGN KEY (in_reply_to) REFERENCES messages(id)
);

CREATE TABLE IF NOT EXISTS task_results (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    output_path TEXT NOT NULL,
    summary TEXT,
    token_usage INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS reviews (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    reviewer_id TEXT NOT NULL,
    review_type TEXT NOT NULL,
    passed INTEGER NOT NULL,
    comments_path TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (reviewer_id) REFERENCES team_members(id)
);
