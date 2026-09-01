# Skillgrid + Mnemonic Integration Plan (Corrected)

> Self-Evolving, Observable, Infinite-Context Agent Platform

## Goal

Build a unified Agent platform by integrating:

- **Hermes Memory** — structured facts, importance, hybrid retrieval, forgetting.
- **Cleave** — session relay, knowledge compounding, AI self-written handoffs.
- **Existing skillgrid & Mnemonic** — Go MCP server + SQLite + filesystem.

All persistent data is stored in either:

- Mnemonic's SQLite database (L1 facts, L3 vectors, metadata).
- The project filesystem (L2 documents, skills, knowledge).

> **Note:** Mnemonic itself is a Go MCP server, not merely a DB file.

---

## 1. Corrected Directory Structure

```bash
# GLOBAL
~/.skillgrid/
├── mnemonic/                     # Mnemonic MCP server installation
│   ├── mnemonic                  # Go binary
│   ├── skillgrid.db              # SQLite database (managed by Mnemonic)
│   ├── config.yaml               # Mnemonic configuration (scopes, models)
│   └── logs/                     # Mnemonic logs
├── agents/                       # Agent role definitions (global templates)
│   ├── scout.md
│   └── reviewer.md
└── templates/                    # Template files
```

```bash
# PROJECT
$PROJECT/.skillgrid/
├── files/                        # L2 Document Memory (commit to Git)
│   ├── tasks/
│   │   └── {task-id}/
│   │       ├── brief.md
│   │       ├── brief.abstract
│   │       ├── brief.overview
│   │       ├── output.md
│   │       ├── output.abstract
│   │       └── output.overview
│   ├── memories/
│   │   └── memory-{timestamp}.md
│   ├── knowledge/
│   │   └── architecture.md
│   └── skills/
│       ├── my_script.py
│       └── data_processor.js
├── workspace/                    # L0 Working Memory (DO NOT commit)
│   └── sessions/
│       └── {session-id}/
│           ├── context.md
│           └── scratch/
└── .cleave/                      # Cleave-style handoff files (optional)
    ├── PROGRESS.md
    ├── KNOWLEDGE.md
    └── NEXT_PROMPT.md
```

> The Mnemonic binary can be run globally and pointed to a project DB, or run per project with its own DB. For simplicity, assume a **global Mnemonic** with a shared DB that is scope-aware.

---

## 2. Mnemonic SQLite Schema (`skillgrid.db`)

The following tables are added to the existing Mnemonic SQLite schema. They live inside `skillgrid.db` and are managed by the Mnemonic MCP server.

### 2.1 Fact Memory (L1)

```sql
CREATE TABLE facts (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT '/',
    importance REAL DEFAULT 0.5,
    source_type TEXT,                     -- 'user', 'agent', 'system'
    source_id TEXT,
    tags TEXT,                            -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    metadata TEXT
);
CREATE INDEX idx_facts_scope ON facts(scope);
CREATE INDEX idx_facts_importance ON facts(importance DESC);
CREATE INDEX idx_facts_deleted ON facts(deleted_at);

-- Lexical FTS
CREATE VIRTUAL TABLE facts_fts USING fts5(content, scope, content=facts);
```

### 2.2 Vectors (L3 — semantic via sqlite-vec)

```sql
CREATE TABLE fact_embeddings (
    fact_id TEXT PRIMARY KEY,
    embedding BLOB,                       -- float32 array
    model TEXT,
    FOREIGN KEY (fact_id) REFERENCES facts(id) ON DELETE CASCADE
);

CREATE TABLE skill_embeddings (
    skill_id TEXT PRIMARY KEY,
    embedding BLOB,
    model TEXT,
    FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE
);
```

### 2.3 Skills (reusable tools)

```sql
CREATE TABLE skills (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    file_path TEXT NOT NULL,              -- relative to $PROJECT/.skillgrid/files/skills/
    language TEXT,                        -- 'python', 'bash', 'prompt', 'js'
    version TEXT DEFAULT '1.0.0',
    scope TEXT DEFAULT '/',
    author_type TEXT,                     -- 'agent', 'user', 'system'
    created_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    metadata TEXT
);
CREATE INDEX idx_skills_scope ON skills(scope);
CREATE INDEX idx_skills_name ON skills(name);
CREATE VIRTUAL TABLE skills_fts USING fts5(name, description, file_path);
```

### 2.4 Session Relay (Cleave-style handoffs)

```sql
CREATE TABLE session_handoffs (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    task_id TEXT,
    session_number INTEGER,
    progress_path TEXT,
    knowledge_path TEXT,
    next_prompt_path TEXT,
    context_usage_percent REAL,
    cost_usd REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE session_archives (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    task_id TEXT,
    handoff_id TEXT,
    summary TEXT,
    full_log_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2.5 Observability

```sql
CREATE TABLE retrieval_trails (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    query TEXT NOT NULL,
    scope TEXT,
    retrieval_mode TEXT,
    directories_traversed TEXT,
    facts_retrieved TEXT,
    documents_retrieved TEXT,
    skills_retrieved TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE forgetting_events (
    id TEXT PRIMARY KEY,
    fact_id TEXT,
    old_importance REAL,
    new_importance REAL,
    reason TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE skill_usage (
    id TEXT PRIMARY KEY,
    skill_id TEXT,
    called_by TEXT,
    arguments TEXT,
    output TEXT,
    execution_time_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (skill_id) REFERENCES skills(id)
);
```

---

## 3. MCP Tools (Exposed by Mnemonic — extended server)

The Mnemonic MCP server is extended to provide the following tools. Clients (skillgrid CLI, OpenCode plugin, Claude Desktop) call these via the MCP protocol.

### 3.1 Memory Tools

| Tool | Description |
|---|---|
| `fact_add` | Extracts atomic facts from text via LLM, stores them with importance + scope; generates the embedding and stores it in `fact_embeddings`. |
| `fact_search` | Hybrid search (FTS5 + vector) with scope filtering and importance ranking; logs the trail to `retrieval_trails`. |
| `fact_forget` | Soft-deletes facts (sets `deleted_at`). |
| `fact_decay` | Lowers importance of old facts; logs to `forgetting_events`. |

### 3.2 Skill Tools

| Tool | Description |
|---|---|
| `write_skill` | Writes the script to `.skillgrid/files/skills/`; inserts metadata into `skills` and the embedding into `skill_embeddings`. |
| `search_skills` | Hybrid search for skills. |
| `use_skill` | Reads and executes the skill script in a sandbox, returns output; logs usage to `skill_usage`. |
| `list_skills` | Lists skills by scope. |

### 3.3 Session Relay Tools

| Tool | Description |
|---|---|
| `session_handoff` | Generates `PROGRESS.md`, `KNOWLEDGE.md`, `NEXT_PROMPT.md` in the project's `.cleave/`; records the handoff in `session_handoffs`. |
| `session_resume` | Reads the handoff files and prepares a new session prompt — effectively restarts the task with full context. |
| `session_status` | Returns current session stats (cost, context, handoff count). |

### 3.4 Composite Tools

| Tool | Description |
|---|---|
| `mnemonic_commit` | After a task is done, writes a memory document, extracts facts, and optionally detects reusable skills and calls `write_skill`. |
| `knowledge_compact` | Consolidates knowledge: keeps high-importance facts, rolls off old session logs, updates `KNOWLEDGE.md`. |

---

## 4. How the Pieces Fit Together

```text
┌─────────────────────────────────────────────────────────────────────┐
│                    MCP Clients                                      │
│   skillgrid CLI, OpenCode plugin, Claude Desktop, etc.             │
│   (They call Mnemonic tools via stdio or HTTP)                     │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Mnemonic MCP Server                           │
│         (Go binary – extended with new tables & tools)             │
│  ┌───────────────┐ ┌───────────────┐ ┌─────────────────────────┐  │
│  │ Memory tools  │ │ Skills tools  │ │ Session relay tools     │  │
│  │ fact_add      │ │ write_skill   │ │ session_handoff         │  │
│  │ fact_search   │ │ search_skills │ │ session_resume          │  │
│  │ fact_forget   │ │ use_skill     │ │ session_status          │  │
│  │ fact_decay    │ │ list_skills   │ │                         │  │
│  └───────────────┘ └───────────────┘ └─────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │ Composite tools: mnemonic_commit, knowledge_compact         │  │
│  └─────────────────────────────────────────────────────────────┘  │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    skillgrid.db (SQLite)                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────────┐  │
│  │ facts    │ │ skills   │ │ handoffs │ │ fact_embeddings      │  │
│  │ (L1)     │ │ (L1)     │ │ (relay)  │ │ skill_embeddings (L3)│  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                           │
│  │ retrieval│ │ forgetting│ │ skill_   │                            │
│  │ trails   │ │ events    │ │ usage    │                            │
│  └──────────┘ └──────────┘ └──────────┘                            │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Filesystem (L2)                                  │
│  $PROJECT/.skillgrid/files/                                         │
│  ├── tasks/                                                         │
│  ├── memories/                                                      │
│  ├── skills/                                                        │
│  └── knowledge/                                                     │
│  Also: $PROJECT/.skillgrid/workspace/ (L0)                          │
│        $PROJECT/.skillgrid/.cleave/ (handoff)                       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Background Processes

- **Auto-skill generation:** during `mnemonic_commit`, use the LLM to detect reusable patterns; if any, automatically call `write_skill`.
- **Memory decay scheduler:** regularly run `fact_decay` to lower the importance of old facts.
- **Index rebuilding:** after bulk import, rebuild FTS and vector indexes.
- **Handoff watchdog:** monitor context usage and trigger `session_handoff` when limits are exceeded (Cleave-style).

---

## 6. CLI Commands (skillgrid CLI wrappers)

```bash
# Memory
skillgrid memory fact add --text "..." --scope "/project-x"
skillgrid memory fact search "query" --scope "/project-x"
skillgrid memory fact list --scope "/project-x"
skillgrid memory fact forget --fact-id <id>
skillgrid memory decay --run

# Skills
skillgrid skill write --name "my-skill" --lang python --file ./myscript.py
skillgrid skill search "token validation"
skillgrid skill execute <skill_id> --args '{"key":"value"}'
skillgrid skill list --scope "/project-x"

# Session relay
skillgrid session handoff --task-id task-001
skillgrid session resume --handoff-id handoff-001
skillgrid session status

# Compact & trails
skillgrid compact --scope "/project-x"
skillgrid trail recent --limit 10
skillgrid trail show <trail-id>
```

---

## 7. Implementation Roadmap (8 weeks)

**Week 1-2 — Schema & extension of Mnemonic**

- Add sqlite-vec support to the Mnemonic build.
- Run migrations to create all new tables.
- Write helper functions in Go for vector storage, FTS, and file I/O.

**Week 3-4 — Memory tools**

- Implement `fact_add`, `fact_search`, `fact_forget`, `fact_decay`.
- Ensure `retrieval_trails` logging.

**Week 5-6 — Skills tools**

- Implement `write_skill`, `search_skills`, `use_skill`, `list_skills`.
- Add auto-skill generation to `mnemonic_commit`.

**Week 7 — Session relay tools**

- Implement `session_handoff`, `session_resume`, `session_status`.
- Implement `knowledge_compact`.

**Week 8 — CLI, OpenCode plugin integration & testing**

- Build CLI wrappers.
- Optional: create an OpenCode plugin that calls Mnemonic tools.
- End-to-end test: task → completion → memory → skill → handoff → resume.

---

## 8. Key Benefits

- [x] **Unified storage:** Mnemonic manages all metadata + vectors in SQLite; the filesystem holds all human-editable documents.
- [x] **Structured facts** with importance, scope, soft-delete.
- [x] **Hybrid retrieval** (BM25 + vector) inside the same DB.
- [x] **Memory decay and forgetting.**
- [x] **Agents can write, search, and execute skills.**
- [x] **Auto-skill generation** from completed work.
- [x] **Session relay** for infinite context (Cleave-style).
- [x] **Full observability** via retrieval trails.
- [x] **No external services** — only sqlite-vec.
- [x] **Git-friendly:** all L2 files are plain text.
- [x] **MCP-based** — works with any MCP client (Claude, OpenCode, etc.).

---

## 9. Next Steps

1. Clone/extend the Mnemonic repository.
2. Add the new SQL migrations.
3. Implement the core tools in Go.
4. Update the skillgrid CLI to call the new MCP tools.
5. (Optional) Build an OpenCode plugin that uses these tools.
6. Test with real projects.
