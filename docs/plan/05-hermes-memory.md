# Centralized Proposal: Hybrid Agent Memory + Skill System for Skillgrid

_Powered by nmemonic SQLite + Filesystem_

## Goal

Transform skillgrid into a complete, self-evolving Agent context platform by integrating Hermes-style memory (facts, forgetting, hybrid retrieval) and skill management (write, search, execute) into a unified hybrid storage system. All data lives in the existing nmemonic SQLite database (`.skillgrid/skillgrid.db`) and filesystem (`.skillgrid/files/`), with no external dependencies.

---

## 1. Four-Layer Memory & Skill Model

| Layer | Name | Storage | Notes |
|-------|------|---------|-------|
| **L0** | Working Memory (Ephemeral) | `workspace/sessions/{session-id}/` | Current session context / scratch space. Not committed; auto-cleaned after the session ends. |
| **L1** | Fact Memory (Structured) | nmemonic SQLite (`facts` table) | Atomic facts from conversations, tasks, documents. Importance scoring, scope isolation, soft deletion, timestamps. |
| **L2** | Document Memory (Human-Readable) | Markdown/text in `files/` | Full task briefs, outputs, memory documents, skill scripts. Committed to Git for versioning and manual editing. |
| **L3** | Semantic Memory (Vector) | nmemonic via `sqlite-vec` | Embeddings of facts and skills for semantic similarity (no external vector DB). Enables hybrid retrieval (BM25 + Cosine). |

---

## 2. NMEMONIC SQLite Schema (all tables in `skillgrid.db`)

### 2.1 Fact Memory (L1)

```sql
CREATE TABLE facts (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,                -- Atomic fact sentence
    scope TEXT NOT NULL DEFAULT '/',      -- '/project-x', '/team-alpha'
    importance REAL DEFAULT 0.5,          -- 0.0 to 1.0 (higher = more critical)
    source_type TEXT,                     -- 'user', 'agent', 'system'
    source_id TEXT,                       -- e.g., 'task-001', 'session-abc'
    tags TEXT,                            -- JSON array: ["auth", "jwt"]
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,                  -- Soft deletion
    metadata TEXT                         -- JSON extra fields
);

CREATE INDEX idx_facts_scope ON facts(scope);
CREATE INDEX idx_facts_importance ON facts(importance DESC);
CREATE INDEX idx_facts_deleted ON facts(deleted_at);
CREATE INDEX idx_facts_source ON facts(source_type, source_id);
```

### 2.2 Full-Text Search (Lexical Retrieval)

```sql
CREATE VIRTUAL TABLE facts_fts USING fts5(content, scope, content=facts);
```

### 2.3 Vector Storage (L3 — Semantic Memory) via sqlite-vec

```sql
CREATE TABLE fact_embeddings (
    fact_id TEXT PRIMARY KEY,
    embedding BLOB,                       -- float32 array
    model TEXT,                           -- e.g., 'text-embedding-3-small'
    FOREIGN KEY (fact_id) REFERENCES facts(id) ON DELETE CASCADE
);
```

### 2.4 Skills (Reusable Agent-Created Tools)

```sql
CREATE TABLE skills (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    file_path TEXT NOT NULL,              -- e.g., 'files/skills/my_script.py'
    language TEXT,                        -- 'python', 'bash', 'prompt', 'js'
    version TEXT DEFAULT '1.0.0',
    scope TEXT DEFAULT '/',
    author_type TEXT,                     -- 'agent', 'user', 'system'
    created_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    metadata TEXT                         -- JSON: dependencies, params, etc.
);

CREATE INDEX idx_skills_scope ON skills(scope);
CREATE INDEX idx_skills_name ON skills(name);
CREATE INDEX idx_skills_deleted ON skills(deleted_at);

CREATE VIRTUAL TABLE skills_fts USING fts5(name, description, file_path);
```

### 2.5 Skill Embeddings (L3 for Skills)

```sql
CREATE TABLE skill_embeddings (
    skill_id TEXT PRIMARY KEY,
    embedding BLOB,
    model TEXT,
    FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE
);
```

### 2.6 Observability (Retrieval Trails)

```sql
CREATE TABLE retrieval_trails (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    query TEXT NOT NULL,
    scope TEXT,
    retrieval_mode TEXT,                  -- 'lexical', 'semantic', 'hybrid'
    directories_traversed TEXT,           -- JSON array
    facts_retrieved TEXT,                 -- JSON array of fact IDs
    documents_retrieved TEXT,             -- JSON array of file paths
    skills_retrieved TEXT,                -- JSON array of skill IDs
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2.7 Forgetting/Decay Log

```sql
CREATE TABLE forgetting_events (
    id TEXT PRIMARY KEY,
    fact_id TEXT,
    old_importance REAL,
    new_importance REAL,
    reason TEXT,                          -- 'time_decay', 'access_decay', 'manual'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2.8 Skill Usage Log (Optional)

```sql
CREATE TABLE skill_usage (
    id TEXT PRIMARY KEY,
    skill_id TEXT,
    called_by TEXT,
    arguments TEXT,                       -- JSON
    output TEXT,
    execution_time_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (skill_id) REFERENCES skills(id)
);
```

---

## 3. Filesystem Layout (L0 + L2)

```bash
$PROJECT/.skillgrid/
├── skillgrid.db                         # nmemonic (SQLite + vectors)
├── files/                               # L2 Document Memory (Commit to Git)
│   ├── tasks/                           # Existing task briefs/outputs
│   │   └── {task-id}/
│   │       ├── brief.md                 # L2 Full details
│   │       ├── brief.abstract           # L0 100-token summary
│   │       ├── brief.overview           # L1 2000-token overview
│   │       ├── output.md
│   │       ├── output.abstract
│   │       └── output.overview
│   ├── memories/                        # Long-term memory documents
│   │   └── memory-{timestamp}.md        # Generated by mnemonic_commit
│   ├── knowledge/                       # Static knowledge base (optional)
│   │   └── architecture.md
│   └── skills/                          # Agent-written skills
│       ├── my_script.py
│       ├── data_processor.js
│       └── git-helper.sh
└── workspace/                           # L0 Working Memory (DO NOT commit)
    └── sessions/
        └── {session-id}/
            ├── context.md
            └── scratch/
```

---

## 4. MCP Tools (Exposed to AI Agents via the skillgrid MCP Server)

### 4.1 Memory Tools (Facts + Retrieval)

**`fact_add`**

- **Input:** `text` (string), `scope` (string, optional), `source_type` (string, optional).
- **Logic:**
  1. Call the LLM to extract atomic facts with importance scores.
  2. Insert into the `facts` table.
  3. Compute the embedding and store it in `fact_embeddings` (L3).
  4. Return the fact IDs.

**`fact_search`**

- **Input:** `query` (string), `scope` (string, optional), `limit` (int, default 10), `mode` (`lexical` | `semantic` | `hybrid`, default `hybrid`).
- **Logic:**
  1. Lexical: FTS5 match on `facts_fts`.
  2. Semantic: compute the query embedding → sqlite-vec nearest neighbors.
  3. Hybrid: combine and re-rank by relevance + importance.
  4. Log the trail to `retrieval_trails`.
  5. Return facts with metadata.

**`fact_forget`**

- **Input:** `fact_id` (string) or `query` (string).
- **Logic:** Soft-delete matching facts (set `deleted_at`).

**`fact_decay`**

- **Input:** `days_threshold` (int, default 30), `decay_factor` (float, default 0.95).
- **Logic:** Lower the importance of old facts; log to `forgetting_events`; auto-purge very low ones.

### 4.2 Skill Management Tools

**`write_skill`**

- **Input:** `name` (string), `description` (string), `code` (string), `language` (string), `scope` (string, optional), `dependencies` (JSON, optional), `overwrite` (bool).
- **Logic:**
  1. Write the code to `files/skills/{name}.{extension}`.
  2. Insert into the `skills` table.
  3. Compute the embedding and store it in `skill_embeddings` (L3).
  4. Call `fact_add` on the description to create a memory of this skill.
  5. Return the `skill_id`.

**`search_skills`**

- **Input:** `query` (string), `scope` (string, optional), `limit` (int, default 5), `mode` (`lexical` | `semantic` | `hybrid`, default `hybrid`).
- **Logic:** Similar hybrid retrieval as `fact_search`, but over skills. Return skills with metadata and `file_path`.

**`use_skill`**

- **Input:** `skill_id` (or `skill_name`), `arguments` (JSON).
- **Logic:**
  1. Retrieve `file_path` from the `skills` table.
  2. Read the script content.
  3. Execute in a sandboxed environment (subprocess/Docker).
  4. Capture the output and log to `skill_usage`.
  5. Return stdout/stderr.

**`list_skills`**

- **Input:** `scope` (string, optional), `limit` (int).
- **Returns:** list of skills (name, description, language, created_at).

### 4.3 Composite Tools (Integration)

**`mnemonic_commit` (Enhanced)**

- **Input:** `task_id` (string), `title` (string), `content` (string, optional).
- **Logic:**
  1. Read `brief.md` and `output.md` from the task.
  2. Write the full memory to `files/memories/memory-{timestamp}.md`.
  3. Generate `.abstract` and `.overview` tiers.
  4. Insert into the `long_term_memories` table.
  5. Call `fact_add` on the content to extract atomic facts (L1).
  6. **Optional:** ask the LLM if a reusable skill emerges from this work; if yes, call `write_skill` automatically.
  7. Update the task status to `archived`.

---

## 5. Background Processes (Self-Evolution & Maintenance)

- **Auto-Skill Generation (Hermes-like):** when `mnemonic_commit` runs, the LLM analyzes the completed task. If a reusable pattern/script is detected, auto-generate a skill — the agent learns and grows its toolkit over time.
- **Memory Decay Scheduler:** a daily cron job runs `fact_decay`; low-importance facts (>90 days old) are automatically soft-deleted.
- **Index Rebuilding:** rebuild FTS indexes and vectors after batch imports or migrations.

---

## 6. Observability & CLI Commands

```bash
# Memory
skillgrid memory fact list --scope <scope>
skillgrid memory fact add --text "..." --scope "/project-x"
skillgrid memory fact search "query"
skillgrid memory trail recent --limit 10
skillgrid memory forget --fact-id <id>
skillgrid memory decay --run

# Skills
skillgrid skill list --scope <scope>
skillgrid skill show <skill_id>
skillgrid skill search "query"
skillgrid skill execute <skill_id> --args '{"key":"value"}'
skillgrid skill delete <skill_id>

# Vectors
skillgrid vector rebuild --table facts
skillgrid vector rebuild --table skills
```

---

## 7. Implementation Roadmap (6 Weeks)

### Week 1 — Foundation & Schema

- [ ] Enable sqlite-vec in the Go SQLite driver.
- [ ] Run all SQL migrations (facts, skills, vectors, FTS, trails).
- [ ] Write Go helpers: `StoreEmbedding`, `SearchVector`, FTS search.
- [ ] Write filesystem helpers for the skills directory.

### Week 2 — Core Memory Tools

- [ ] Implement `fact_add` (LLM atomic extraction + embedding).
- [ ] Implement `fact_search` (lexical + semantic hybrid).
- [ ] Implement `fact_forget` and `fact_decay`.
- [ ] Log trails in `retrieval_trails`.

### Week 3 — Skills Core

- [ ] Implement `write_skill` (file write + DB + embedding).
- [ ] Implement `search_skills` (hybrid).
- [ ] Implement `use_skill` (sandboxed execution + logging).
- [ ] Implement `list_skills`.

### Week 4 — Integration & Auto-Skills

- [ ] Enhance `mnemonic_commit` with auto-skill generation.
- [ ] Add fact extraction to `mnemonic_commit`.
- [ ] Preload top facts and skills into the agent system prompt on startup.

### Week 5 — CLI & Observability

- [ ] Build all CLI commands.
- [ ] Add `skills_retrieved` to `retrieval_trails`.
- [ ] Create a migration script to backfill facts from existing tasks.

### Week 6 — Testing & Optimization

- [ ] End-to-end test: Task → Completion → Memory → Auto-Skill → Reuse.
- [ ] Optimize vector search (indexing, batch embedding).
- [ ] Write documentation and examples.

---

## 8. Key Advantages of This Centralized Proposal

- [x] **Unified Hybrid Storage:** nmemonic SQLite for all metadata + vectors; the filesystem for all human-readable documents and scripts.
- [x] **Hermes-Style Memory:** structured facts with importance, scope, soft deletion, forgetting, and hybrid retrieval (BM25 + Vector).
- [x] **Skill Writing & Reuse:** agents can write, store, discover, and execute skills across sessions, enabling self-evolution.
- [x] **L3 Vector Store Inside nmemonic:** no external vector DB; uses the sqlite-vec extension for in-database semantic search.
- [x] **Full Observability:** `retrieval_trails` logs every search path, making debugging simple and transparent.
- [x] **Self-Evolution:** `mnemonic_commit` automatically extracts facts and optionally generates new skills from completed work.
- [x] **Git-Friendly:** all L2 documents are plain text files that can be versioned, reviewed, and manually edited.
- [x] **Minimal External Dependencies:** only sqlite-vec required; everything else is already in the Go stack.
- [x] **OpenCode Plugin Ready:** a thin plugin can visualize tasks, facts, skills, and trails without duplicating logic.

---

## 9. Architecture Diagram (Conceptual)

```text
                    +-------------------+
                    |  Agent / User     |
                    +--------+----------+
                             |
                             v
                    +-------------------+
                    |  MCP Tools        |  (Go)
                    |  - fact_add       |
                    |  - fact_search    |
                    |  - write_skill    |
                    |  - search_skills  |
                    |  - use_skill      |
                    |  - mnemonic_commit|
                    +--------+----------+
                             |
          +------------------+---------------------+
          |                                           |
          v                                           v
+---------------------------+            +-----------------------------+
|   nmemonic SQLite         |            |   Filesystem                |
|   (skillgrid.db)          |            |   $PROJECT/.skillgrid/files/|
|                           |            |                             |
|   - facts (L1)            |            |   - tasks/ (L2)             |
|   - skills (L1)           |            |   - memories/ (L2)          |
|   - fact_embeddings (L3)  |            |   - skills/ (L2)            |
|   - skill_embeddings (L3) |            |   - knowledge/ (L2)         |
|   - facts_fts             |            |                             |
|   - skills_fts            |            |   $PROJECT/.skillgrid/      |
|   - retrieval_trails      |            |   workspace/ (L0)           |
|   - forgetting_events     |            |                             |
+---------------------------+            +-----------------------------+
```

---

## 10. Conclusion

This centralized proposal unifies Hermes-style memory management with skill-writing capabilities, all built on top of the existing skillgrid infrastructure (Go MCP + nmemonic SQLite + filesystem). It adds:

- Structured, importance-ranked facts.
- Hybrid lexical + semantic retrieval (BM25 + sqlite-vec).
- Soft deletion and memory forgetting/decay.
- Agent-writable, reusable skills (scripts/prompts/tools).
- Auto-generation of skills from completed tasks (self-evolution).
- Full observability via retrieval trails.
- No external dependencies beyond sqlite-vec.

The implementation is incremental (add tables, add tools, add CLI) and does not require rewriting any existing code. Each phase builds on the previous one.
