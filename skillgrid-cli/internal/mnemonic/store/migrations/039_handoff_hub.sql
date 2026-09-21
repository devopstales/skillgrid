-- 039: handoff hub — change snapshots, named checkpoints, handoff refs
-- (change 015-handoff-hub).
--
-- Additive only. The three tables combine the three existing checkpoint
-- sources into one queryable hub:
--
--   change_snapshots — a git-derived, append-only log of EVERY commit for the
--                      project. Reconstructable by replaying `git log`, so it
--                      can never be the sole source of truth; the commit
--                      (and its [skillgrid-context] block) is the durable
--                      record, this table is the index. context_json is the
--                      parsed [skillgrid-context] block (Task/Decisions/
--                      Remaining/Tried) and is NULL when the commit carries
--                      no block. Idempotent upsert on (project, commit).
--
--   checkpoints      — named, INTENTIONAL markers placed before risky actions
--                      (before-apply-<change>, before long pauses, before
--                      validation). Unlike change_snapshots these are placed
--                      by a human/agent, capture dirty state + links to the
--                      active spec dir / PRD / handoff file, and carry a
--                      lifecycle (open | verified | stale | archived) driven
--                      by drift verification.
--
--   handoff_refs     — the join between a handoff artifact (session cleave
--                      bundle, team task completion, or a checkpoint marker)
--                      and the commit range it covers, plus the spec dir it
--                      hands off. This is what makes "this handoff covers
--                      commits X..Y of change Z" explicit.

CREATE TABLE IF NOT EXISTS change_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    branch TEXT NOT NULL,
    "commit" TEXT NOT NULL,
    commit_short TEXT,
    subject TEXT,
    context_json TEXT,
    changed_files TEXT,
    author TEXT,
    committed_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (project, "commit")
);

CREATE INDEX IF NOT EXISTS idx_snapshots_project_time
    ON change_snapshots (project, committed_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS checkpoints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    name TEXT NOT NULL,
    branch TEXT,
    "commit" TEXT,
    dirty INTEGER NOT NULL DEFAULT 0,
    prd_path TEXT,
    spec_dir TEXT,
    handoff_file TEXT,
    evidence TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TEXT NOT NULL,
    verified_at TEXT,
    UNIQUE (project, name)
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_project_status
    ON checkpoints (project, status, created_at DESC);

CREATE TABLE IF NOT EXISTS handoff_refs (
    handoff_id TEXT NOT NULL,
    handoff_type TEXT NOT NULL,
    project TEXT NOT NULL,
    from_commit TEXT,
    to_commit TEXT,
    spec_dir TEXT,
    team_id TEXT,
    task_id TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (handoff_id, handoff_type, project)
);

CREATE INDEX IF NOT EXISTS idx_handoff_refs_project
    ON handoff_refs (project, created_at DESC);
