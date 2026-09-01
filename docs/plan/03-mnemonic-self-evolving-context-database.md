# Execution Plan: Upgrading Skillgrid to a Tiered, Self-Evolving, Observable Contextual System

## Goal

Transform skillgrid from a flat storage system into an intelligent, self-evolving, and debuggable contextual database for AI Agents, incorporating tiered storage (L0/L1/L2), hierarchical retrieval, session compaction, and full observability.

**Principle:** Keep the existing Go MCP server and SQLite+FS hybrid as the core. Add new services/scripts in Go (or optionally Python) to handle embeddings, summarization, and vector retrieval. Do **not** rewrite the entire codebase.

---

## Phase 1: Database Schema Extensions (SQLite)

### Step 1.1 — Extend the `tasks` table to store paths for all three tiers

```sql
ALTER TABLE tasks ADD COLUMN abstract_path TEXT;
ALTER TABLE tasks ADD COLUMN overview_path TEXT;
-- Note: brief_path already exists and serves as the L2 (Full Details) tier.
```

### Step 1.2 — Create the `long_term_memories` table for session compaction

```sql
CREATE TABLE long_term_memories (
    id TEXT PRIMARY KEY,
    task_id TEXT,                          -- Optional link back to source task
    title TEXT NOT NULL,
    abstract_path TEXT NOT NULL,           -- L0 (100 tokens)
    overview_path TEXT NOT NULL,           -- L1 (2000 tokens)
    full_path TEXT NOT NULL,               -- L2 (Full markdown)
    tags TEXT,                             -- JSON array for quick filtering
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL
);
```

### Step 1.3 — Create the `retrieval_trails` table for observability

```sql
CREATE TABLE retrieval_trails (
    id TEXT PRIMARY KEY,
    session_id TEXT,                       -- OpenCode session or agent run ID
    query TEXT NOT NULL,
    directories_traversed TEXT NOT NULL,   -- JSON array, e.g., ["tasks", "memories"]
    files_read TEXT NOT NULL,              -- JSON array of absolute/relative paths
    final_result_path TEXT,                -- The path returned to the agent
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## Phase 2: Filesystem Enhancements (Tiered L0/L1/L2)

### Step 2.1 — Establish the new file naming convention

For any content file (e.g. `brief.md`), generate two additional files:

| Tier | Size | Path |
|------|------|------|
| **L2** — Full Details (original) | full | `files/tasks/{task-id}/brief.md` |
| **L0** — Abstract | ~100 tokens | `files/tasks/{task-id}/brief.abstract` |
| **L1** — Overview | ~2000 tokens | `files/tasks/{task-id}/brief.overview` |

### Step 2.2 — Create the `tiered_storage` utility in Go

**File:** `internal/storage/tiered.go`

Functions to implement:

- `GenerateTiers(content string, contentPath string) error`
- `ReadTier(path string, tier string) (string, error)` — `tier` = `"l0" | "l1" | "l2"`

Implementation logic:

1. Take the full content (string).
2. Call the LLM with prompt: *"Summarize this in 1 sentence (max 100 tokens)."* → write to `{path}.abstract`.
3. Call the LLM with prompt: *"Provide a structured overview with key sections (max 2000 tokens)."* → write to `{path}.overview`.
4. The original content remains as `{path}` (L2).

---

## Phase 3: Background Tiering Service (Auto-Generation)

### Step 3.1 — Add a file watcher or lifecycle hook

In the MCP server, when a task is created (`brief.md` written) or completed (`output.md` written), trigger a background goroutine.

Hook locations:

- In `team_spawn_task`: after writing `brief.md` → spawn `GenerateTiers(brief.md)`.
- In `agent_submit_output`: after writing `output.md` → spawn `GenerateTiers(output.md)`.

### Step 3.2 — Implement a fallback for existing data

**Command:** `skillgrid migrate --tier`

- Iterates over all existing tasks with non-null `brief_path`.
- Reads the file content.
- Generates `.abstract` and `.overview` if they don't exist.
- Updates the SQLite `abstract_path` and `overview_path` columns.

---

## Phase 4: Implement New MCP Tools (Advanced Retrieval & Memory)

### Tool 4.1 — `semantic_search` (replaces naive full-text search)

**Input:**

- `query`: string
- `max_results`: integer (default: 3)

**Logic:**

1. (Directory search) Compute the embedding of the query.
2. Compare against embeddings of task titles and task `.abstract` (L0) files.
3. Retrieve the top 5 most relevant DIRECTORY IDs (e.g. `task-001`, `task-015`).
4. (In-directory search) Within those 5 directories, recursively scan all `.abstract` and `.overview` files.
5. Re-rank the results based on relevance.
6. Return only the `.overview` (L1) content by default. Do **not** return L2 yet.
7. Log the entire traversal path to `retrieval_trails`.

**Return JSON:**

```json
{
  "results": [
    {
      "task_id": "task-001",
      "overview": "The authentication system uses JWT with refresh tokens...",
      "abstract": "JWT-based auth implemented",
      "full_path": "files/tasks/task-001/brief.md"
    }
  ]
}
```

### Tool 4.2 — `load_full_details`

**Input:**

- `path`: string (the `full_path` from `semantic_search`)

**Logic:**

1. Read the full file from the filesystem.
2. Return the raw Markdown content.

**Purpose:** Allows the agent to lazily load L2 only when needed, saving tokens.

### Tool 4.3 — `mnemonic_commit` (Session Compaction / Self-Evolution)

**Input:**

- `task_id`: string
- `lessons_learned`: string (optional, can auto-generate)

**Logic:**

1. Fetch the task record (`brief_path`, `output_path`).
2. Read both files.
3. Call the LLM with prompt:

   > Given the task brief and final output, extract:
   > - What was done.
   > - What challenges were faced.
   > - What worked well.
   > - What failed or could be improved.
   > Write this as a concise memory entry.

4. Generate a new ID: `memory-{timestamp}`.
5. Write the result to `files/memories/memory-{timestamp}.md`.
6. Call `GenerateTiers` on this new file (creates `.abstract`, `.overview`).
7. Insert a record into the `long_term_memories` table.
8. Update the source task status to `archived`.

---

## Phase 5: Observability & Debugging

### Step 5.1 — Instrument all retrieval functions

In any function that reads from SQLite or the filesystem to answer a query, ensure you call:

```go
func LogTrail(query, directories, files, resultPath string) {
    // Insert into retrieval_trails table
}
```

### Step 5.2 — Add a CLI introspection command

**`skillgrid trail show <id>`**

- Prints the exact path the AI browsed.
- Prints the query that was used.
- Prints which files were read.

**`skillgrid trail recent`**

- Lists the last 10 trails.

This allows you to debug exactly why an AI missed a specific piece of information.

---

## Phase 6: Architectural Integration (Keeping the MCP Core)

### Option A — Pure Go implementation (recommended)

- Implement vector similarity using a lightweight Go library (e.g. `go-bert` for embeddings) or an external OpenAI/Cohere API.
- Embedding vectors can be stored in a new SQLite table:

  ```sql
  CREATE TABLE embeddings ( path TEXT PRIMARY KEY, vector BLOB );
  ```

- This avoids external dependencies.

### Option B — Hybrid microservice (if you prefer Python)

- Keep the Go MCP server for orchestration, file writes, and task management.
- Spin up a lightweight Python FastAPI service that handles:
  - Generating embeddings (using sentence-transformers).
  - Performing vector searches.
- The MCP server calls this service via HTTP when `semantic_search` is invoked.

---

## Phase 7: Implementation Roadmap & Priority

### Week 1 — Foundation

- [ ] Run SQL migrations to add columns/tables.
- [ ] Write the `tiered.go` utility (`GenerateTiers`, `ReadTier`).
- [ ] Write the migration script `skillgrid migrate --tier`.
- [ ] Run migration on existing projects to backfill `.abstract`/`.overview`.

### Week 2 — Retrieval Engine

- [ ] Implement vector storage (SQLite + embedding generation).
- [ ] Implement the `semantic_search` MCP tool (Directory-First logic).
- [ ] Implement the `load_full_details` MCP tool.
- [ ] Add observability logging (`LogTrail`).

### Week 3 — Self-Evolution & UI

- [ ] Implement the `mnemonic_commit` MCP tool.
- [ ] Add lifecycle hooks to auto-trigger compaction on task completion.
- [ ] Implement the `skillgrid trail` CLI commands.
- [ ] Test the full flow: Task creation → Work → Completion → Memory Commit → Future Search.

### Week 4 — Optimization & OpenCode Plugin (optional but recommended)

- [ ] Build a thin OpenCode plugin to visualize the task tree and search trails.
- [ ] Add a sidebar showing recent `retrieval_trails` for debugging.
- [ ] Add command palette actions to trigger `mnemonic_commit` manually.

---

## Final Checklist: What Skillgrid Gains From This Upgrade

- [x] **Tiered Storage (L0/L1/L2)** → Up to 90% token savings on retrieval.
- [x] **Directory-First Search** → Retrieves coherent context, not fragmented text.
- [x] **Progressive Loading** → Lazy-load heavy files only when needed.
- [x] **Session-to-Memory Compaction** → The system learns and evolves over time.
- [x] **Full Observability** → Every AI decision is traceable and debuggable.
- [x] **Minimal Code Change** → Core Go MCP server remains intact; only add new modules.
- [x] **OpenCode Plugin (Optional)** → Native UI for managing the memory graph.
