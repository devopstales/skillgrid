# Hybrid Agent Teams Architecture (SQLite Metadata + Filesystem Content)

## Principle

- **SQLite** = "Control Plane" — metadata, state, indexes, foreign keys, paths.
- **Filesystem** = "Data Plane" — prompts, code, outputs, reviews as `.md` files.
- SQLite stores FILE PATHS; the AI reads/writes Markdown files directly on disk.
- Links to the existing skillgrid "mnemonic" memory via `memory_context` columns.

---

## 1. Directory Structure

```bash
# GLOBAL
~/.skillgrid/teams/
├── agents/
│   ├── scout.md          # System prompt for the scout role
│   └── reviewer.md       # System prompt for the reviewer role
└── templates/
    └── task_brief_template.md
```

```bash
# PROJECT
$PROJECT/.skillgrid/
├── skillgrid.db          # SQLite database (all metadata)
└── files/                # Data plane (commit this to Git!)
    ├── tasks/
    │   └── {task-id}/
    │       ├── brief.md      # Task description (written by Lead)
    │       ├── output.md     # Final deliverable (written by Agent)
    │       └── artifacts/    # Extra files (JSON specs, patches)
    ├── messages/
    │   └── {msg-id}/
    │       └── body.md       # Agent-to-agent message content
    └── reviews/
        └── {task-id}/
            ├── spec_review.md    # Spec compliance review
            └── code_review.md    # Code quality review
```

---

## 2. SQLite Schema (paths point to `files/`)

```sql
-- Teams
CREATE TABLE teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Team members (Agent roles)
CREATE TABLE team_members (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    role TEXT NOT NULL,              -- lead, scout, developer, qa, reviewer
    system_prompt_path TEXT,         -- e.g., '~/.skillgrid/agents/scout.md'
    model TEXT,
    status TEXT DEFAULT 'idle',      -- idle, working, blocked, waiting
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
);

-- Tasks (content in files/)
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    parent_task_id TEXT,             -- for subtask decomposition
    title TEXT NOT NULL,             -- short title (searchable in DB)
    brief_path TEXT NOT NULL,        -- e.g., 'files/tasks/task-001/brief.md'
    output_path TEXT,                -- e.g., 'files/tasks/task-001/output.md'
    status TEXT DEFAULT 'pending',   -- pending, in-progress, review_spec, review_code, done, failed
    assigned_to TEXT,
    created_by TEXT,
    priority INTEGER DEFAULT 0,
    memory_context_path TEXT,        -- link to existing skillgrid memory file
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_task_id) REFERENCES tasks(id),
    FOREIGN KEY (assigned_to) REFERENCES team_members(id)
);

-- Agent inbox messages (content in files/)
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    from_agent TEXT NOT NULL,
    to_agent TEXT NOT NULL,
    subject TEXT,                    -- short subject (searchable)
    body_path TEXT NOT NULL,         -- e.g., 'files/messages/msg-abc/body.md'
    status TEXT DEFAULT 'unread',    -- unread, read, archived
    in_reply_to TEXT,                -- threading
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (from_agent) REFERENCES team_members(id),
    FOREIGN KEY (to_agent) REFERENCES team_members(id),
    FOREIGN KEY (in_reply_to) REFERENCES messages(id)
);

-- Task results (summary in SQL, full output in files/)
CREATE TABLE task_results (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    output_path TEXT NOT NULL,       -- e.g., 'files/tasks/task-001/output.md'
    summary TEXT,                    -- 1-sentence summary stored in DB for quick glance
    token_usage INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Two-stage reviews (content in files/)
CREATE TABLE reviews (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    reviewer_id TEXT NOT NULL,
    review_type TEXT NOT NULL,       -- 'spec_compliance' or 'code_quality'
    passed BOOLEAN NOT NULL,
    comments_path TEXT NOT NULL,     -- e.g., 'files/reviews/task-001/spec_review.md'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (reviewer_id) REFERENCES team_members(id)
);
```

---

## 3. Atomic Operations (Keep SQL & FS in Sync)

**Write operation** (e.g. creating a task):

1. Generate `task_id` and directory path.
2. Write the Markdown content to the filesystem (e.g. `brief.md`).
3. If the file write fails → abort (no SQL change).
4. Insert/update the SQLite record with the file path.
5. If the SQL insert fails → delete the orphaned file (rollback).

**Read operation** (e.g. reading a task brief):

1. Query SQLite to get the `brief_path`.
2. Read the file from disk using the path.
3. Return raw Markdown content directly to the LLM (no transformation).

---

## 4. MCP Tools (Exposed to AI Agents)

| Tool Name | Purpose | SQL Table Affected |
|---|---|---|
| `team_spawn_task` | Lead creates a new task + `brief.md` | `tasks` (INSERT) |
| `agent_pull_next_task` | Agent claims highest priority task | `tasks` (UPDATE status) |
| `agent_read_task` | Agent reads `brief.md` from disk | SELECT … + fs read |
| `agent_submit_output` | Agent writes `output.md` + updates task | `task_results` + `tasks` |
| `agent_send_message` | Agent writes `body.md` to another agent | `messages` (INSERT) |
| `agent_read_inbox` | Agent reads unread messages | `messages` (SELECT + fs read) |
| `agent_submit_review` | Reviewer writes `review.md` | `reviews` (INSERT) |
| `agent_mark_done` | Mark task complete | `tasks` (UPDATE) |

---

## 5. Workflow Example (End-to-End)

1. **LEAD AGENT** spawns a task:
   - Writes `files/tasks/task-001/brief.md`.
   - Inserts into `tasks` with `brief_path = 'files/tasks/task-001/brief.md'`, status = `pending`.

2. **DEVELOPER AGENT** pulls next task:
   - `SELECT * FROM tasks WHERE status='pending' ORDER BY priority LIMIT 1`.
   - Reads `brief_path` from disk (raw Markdown), works on the task (writes code, etc.).
   - Writes `files/tasks/task-001/output.md`.
   - `UPDATE tasks SET status='review_spec', output_path='...'`.

3. **REVIEWER AGENT** performs Spec Review:
   - Reads `brief.md` AND `output.md` from disk.
   - Writes `files/reviews/task-001/spec_review.md`; `INSERT INTO reviews (passed=TRUE/FALSE, comments_path='...')`.
   - If passed → `UPDATE tasks SET status='review_code'`; if failed → `status='in-progress'` (rework).

4. **REVIEWER AGENT** performs Code Quality Review:
   - Same pattern, writes `code_review.md`.
   - If both pass → `UPDATE tasks SET status='done'`.

5. **HUMAN** (optional) can manually edit any `.md` file in VSCode:
   - The agent reads the updated file on the next cycle.
   - Git tracks all changes (audit trail + revert capability).

---

## 6. Integration With Existing Skillgrid Memory

- The existing "mnemonic" memory tables (e.g. `memories`, `embeddings`) stay untouched.
- Add a `memory_context_path` column to the `tasks` table.
- When the LEAD spawns a task, it queries the memory table, finds relevant context, writes that context to a file (or links to it), and stores the path.
- The DEVELOPER agent automatically reads the memory context along with the brief, enabling context-aware coding without overloading the SQLite DB with large BLOBs.

---

## 7. Implementation Roadmap

**Phase 1 — Database Schema Migration**

- Alter existing `skillgrid.db` to add the new tables (`teams`, `tasks`, `messages`, …).
- Add `_path` columns and drop old `TEXT` content columns.

**Phase 2 — Filesystem Utilities (Python module)**

- Write `file_store.py` with helper functions:
  - `write_task_brief(task_id, content)`
  - `read_markdown(path)`
  - `write_output(task_id, content)`
  - `write_message(msg_id, content)`
  - `write_review(task_id, review_type, content)`
- Ensure path resolution works for both global (`~/.skillgrid`) and project (`$PROJECT/.skillgrid`).

**Phase 3 — MCP Tool Updates**

- Replace direct SQL read/write of content with calls to `file_store`.
- Keep SQLite for metadata updates only.
- Add new tools: `team_spawn_task`, `agent_read_inbox`, `agent_submit_review`.

**Phase 4 — One-Time Migration (if upgrading from the old system)**

- Script to extract all existing `description` fields from SQL, write them to `files/tasks/{task-id}/brief.md`, and update `brief_path` columns.

**Phase 5 — CLI Commands**

```bash
skillgrid team list
skillgrid task show {task-id}      # prints brief.md + output.md
skillgrid inbox --agent {id}       # shows unread messages
skillgrid review {task-id}         # shows review comments
```

---

## 8. Key Advantages of This Hybrid Approach

- [x] **SQLite handles concurrency** (ACID transactions, row-level locking).
- [x] **Filesystem stores massive content** (code diffs, long prompts) efficiently.
- [x] **Human-friendly:** edit Markdown files directly in VSCode.
- [x] **Git-native:** track the AI's work via `git diff` and revert bad outputs.
- [x] **Lightweight MCP:** only file paths are transmitted over the MCP protocol.
- [x] **Memory-integrated:** link tasks to existing skillgrid memory via paths.
- [x] **Scalable:** SQLite stays small; filesystem can grow indefinitely.
- [x] **No binary blobs:** all data is plain text (Markdown) for maximum transparency.
