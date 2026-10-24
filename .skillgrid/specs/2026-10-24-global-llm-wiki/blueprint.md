# Global Operational Memory + Skill Improvement Loop — Implementation Blueprint (steps, files, verification)

> **Role:** this is the *how* — the implementation plan: concrete steps,
> files to touch, code, tests, and verification. The companion `briefing.md`
> is the *what* (requirements & intent).
> **Change:** Change A of the `2026-10-24-global-llm-wiki` spec (ADR-0049).
> **Sister change:** `2026-10-25-llm-wiki-pipeline` (Change B, ADR-0050) depends on this.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread — a thin end-to-end path (CLI `--scope global` → `global.sqlite` → MCP `fact_add` → prefetch injection) works first, then thickens with batch ops, distill, skill loop. Integration risk (two stores, RRF fusion, plugin hook) is the unknown; the tracer thread proves the wiring before the intelligence lands.

**Goal:** Add a cross-project, cross-session, cross-agent fact store in `global.sqlite` with AKL importance, char limit, prefetch injection, and a skill improvement loop (observe → inspect → amend → evaluate).

**Architecture:** Reuse the existing `facts`/`skills`/`skill_usage` schema (migration 011) in a `__global__` pseudo-project store at `~/.skillgrid/mnemonic/global.sqlite`. Add `category` (memory/user) to facts and `outcome`+`feedback` to `skill_usage` via two new migrations. Scope routing on `fact_*` MCP tools. RRF-fused search across both stores. OpenCode plugin hook for frozen-snapshot prefetch. Background distill extends the 013 hook to global scope. Skill loop reuses the existing `skills`/`skill_usage` tables.

**Tech Stack:** Go 1.22+ (stdlib `database/sql` + `modernc.org/sqlite`), FTS5, existing `mnemonic` package structure. No new external dependencies.

**Spec:** `.skillgrid/specs/2026-10-24-global-llm-wiki/briefing.md` (Change A, A1-A8). Design decisions 20-27 in the briefing. Per ADR-0012 (SQLite as second-brain store), ADR-0048 (hybrid SoT), ADR-0023 (OpenAI-compatible LLM client), ADR-0045 (OpenCode V2 plugins), ADR-0031 (Go 1.22+ floor).

**Critic:** Ran (T2). 3 critical findings resolved by user approval: (C1) skill loop targets code-registry skills, not agent SKILL.md; (C2) `execute.go` modified to log failure rows; (C3) global `fact_add` trails to caller's project store. 10 major findings: RRF → stable merge (M1), pending queue → table (M3), config file named `config.d/indexing.yaml` (M6), bounded store not ranked (M9), plugin idempotent per session (M8).

## Terms

- **Global store** — `~/.skillgrid/mnemonic/global.sqlite`, the `__global__` pseudo-project store. Per `.skillgrid/decisions/04-adr-0012-sqlite-as-second-brain.md`.
- **Category** — `memory` (agent's operational notes) vs `user` (user profile). Per briefing design decision 20 (Hermes MEMORY.md / USER.md pattern).
- **Prefetch injection** — frozen snapshot of top-k global facts in the system prompt at session start. Per briefing design decision 23.
- **Skill loop** — observe → inspect → amend → evaluate. Per briefing design decision 27 (cognee pattern).
- None beyond the briefing's terms.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `Service.OpenGlobal()` creates `~/.skillgrid/mnemonic/global.sqlite` and returns a `ProjectHandle` for `__global__`.
- `ListProjects()` does NOT include `__global__` in its results.
- `RunTTLExpiry()` does NOT process `__global__`.
- `fact_add(scope="global", target="memory")` writes the fact to `global.sqlite` and the `session_events` trail to the caller's project store.
- `fact_add(scope="project")` writes to the per-project store (unchanged behavior).
- `fact_search(scope="all")` returns hits from both stores (stable merge by BM25 rank), each labeled with `scope` + `category`.
- `fact_forget`, `fact_decay`, `fact_decay_all` accept `scope` param (global facts can be forgotten/decayed).
- Batch `operations` array: char limit is checked only on the final result. A batch that frees 200 chars and adds 250 chars to a 2100/2200 store succeeds (final = 2150/2200).
- Overflow: `fact_add` on a 2150/2200 store with a 100-char fact returns `*facts.OverflowError` with `current_entries` (JSON array) + `usage: "2,150/2,200"`.
- Duplicate: adding identical content returns success with "no duplicate added" message.
- Substring matching: `patch`/`remove` use `old_text` (unique substring). Ambiguous match (2+ entries) → error. No match → error with `closest_entries`.
- Prefetch: the OpenCode plugin `context` hook renders a frozen block with `\n§\n` separators, usage headers (`[67% — 1,474/2,200 chars]`), and two labeled sections (GLOBAL MEMORY + USER PROFILE). Entries sorted by importance DESC.
- Prefetch is frozen: the plugin hook fires once per session (memoized by session ID). Mid-session `fact_add` does NOT change the system prompt until the next session.
- Write approval: with `write_approval: true`, passive (distill) `fact_add` calls are staged in `pending_facts` table (durable, survives restart). `fact_pending_list` shows them. `fact_pending_approve` writes them to `facts` and deletes from `pending_facts`.
- Distill: one LLM pass per session close. Each fact is classified as `project` or `global` scope. One call, two stores.
- `skill_usage` rows include `outcome` (success/failure/error) and `feedback` (optional text). Failure/timeout executions ARE logged (C2 fix in `execute.go`).
- `skill inspect <name>` (project-scoped) returns an evidence summary: total runs, success rate, common stderr patterns (exact-match grouping, top-N), last N failures.
- `skill amend <name>` writes `<code_path>.amended` (not in-place) with a proposed patch.
- `skill evaluate <name>` compares outcome rate before vs after the amendment. Regression → discard `.amended`, keep original. Improvement → apply + git commit.
- `skillgrid memory global --target memory|user` CLI works.
- `skillgrid memory global pending/approve/reject` CLI works.

**Artifacts** (files that must exist with real implementation, not stubs):
- `mnemonic/internal/store/migrations/072_facts_category.sql` — adds `category` column to `facts`
- `mnemonic/internal/store/migrations/073_skill_usage_outcome.sql` — adds `outcome` + `feedback` to `skill_usage`, creates `pending_facts` table
- `mnemonic/internal/service/global.go` — `OpenGlobal()` method
- `mnemonic/internal/facts/scope.go` — scope routing (project/global/all) + stable merge
- `mnemonic/internal/facts/batch.go` — batch operations + char limit + substring matching
- `mnemonic/internal/facts/prefetch.go` — frozen snapshot rendering
- `mnemonic/internal/facts/distill.go` — background distill (one pass, scope-classifying)
- `mnemonic/internal/skills/loop.go` — inspect + amend + evaluate
- `mnemonic/internal/skills/execute.go` — modified: log usage rows on failure/timeout (C2 fix)
- `mnemonic/internal/mcp/tools_facts.go` — modified: `scope` + `target` params on ALL fact tools, batch ops, pending queue
- `mnemonic/internal/mcp/tools_skills.go` — modified: `skill inspect/amend/evaluate` tools (project-scoped)
- `mnemonic/cmd/mnemonic/memory.go` — modified: `--scope`, `--target`, `pending/approve/reject`
- `plugins/opencode/src/memory-prefetch.ts` — OpenCode V2 plugin hook (idempotent, fail-open)
- `mnemonic/internal/facts/facts_test.go` — modified: scope, batch, overflow, dedup, substring tests
- `mnemonic/internal/service/global_test.go` — `OpenGlobal` tests
- `mnemonic/internal/skills/loop_test.go` — skill loop tests (including observe-leg failure-row test)
- `.skillgrid/decisions/04-adr-0049-global-operational-memory.md` — ADR

**Key links** (critical connections between artifacts that must work together):
- `OpenGlobal()` must use the same `store.Open()` path as `Open(projectID)` — `__global__` is a valid projectID. File name is `__global__.sqlite` (pinned in door check).
- `fact_add(scope="global")` must route the fact to the `__global__` store AND the trail to the caller's project store (C3).
- `fact_search(scope="all")` must open both stores and stable-merge by BM25 rank (not RRF).
- The OpenCode plugin hook must call the same `prefetch.RenderFrozenSnapshot()` that the CLI `skillgrid memory global prefetch` uses.
- The distill hook must use the same `facts.Store.Add()` that the MCP `fact_add` tool uses — same char limit, same dedup.
- `skill inspect` must read from `skill_usage` (migration 073 columns), including failure rows (C2 fix in `execute.go`).
- `skill amend` must write to the skill's `code_path` (the sandboxed script file), not to the `skills` table. Project-scoped.

**One-way-door decisions** (hard to reverse — flag for explicit user approval):
- ⚠ **Migration 072: `facts.category` column** — adds a `NOT NULL DEFAULT 'memory'` column to the `facts` table. Existing rows get `category='memory'`. Cannot be undone without a data migration.
- ⚠ **Migration 073: `skill_usage.outcome` + `feedback` + `pending_facts` table** — adds two columns to `skill_usage` (existing rows get NULL) and creates `pending_facts` table in `global.sqlite`. Additive (no data loss) but the schema change is permanent.
- ⚠ **`__global__` pseudo-project** — `__global__.sqlite` is a new store file in `~/.skillgrid/mnemonic/`. `ListProjects` and `RunTTLExpiry` must skip it. If the skip logic is wrong, `__global__` appears in project lists or gets TTL-expired.
- ⚠ **`execute.go` observe-leg fix** — modifying `execute.go` to log `skill_usage` rows on failure/timeout changes the existing behavior (currently: only success rows). This affects all existing `use_skill` call sites.

## Global Constraints

- Go 1.22+ minimum to build (per ADR-0031).
- No new external dependencies (per AGENTS.md § Rules).
- Conventional commits only; no AI-attribution trailers (per AGENTS.md § Rules).
- Spec-zone changes commit before code-zone changes (per AGENTS.md § Rules).
- `go test ./...` must pass. `gofmt -w ./...` must be clean.
- Trivy is advisory-only (findings reported, never blocking).

## File Structure

```
mnemonic/internal/store/migrations/
  072_facts_category.sql          — ADD COLUMN category TEXT NOT NULL DEFAULT 'memory' TO facts
  073_skill_usage_outcome.sql     — ADD COLUMN outcome TEXT, feedback TEXT TO skill_usage

mnemonic/internal/service/
  global.go                       — OpenGlobal() + __global__ skip logic (ListProjects, RunTTLExpiry)
  global_test.go                  — tests: OpenGlobal, file name = __global__.sqlite,
                                    ListProjects skip, RunTTLExpiry skip, NormalizeID passthrough

mnemonic/internal/facts/
  facts.go                        — MODIFIED: Store gets category; Add/Search/Forget/Decay
                                    accept category + trailStore param
  scope.go                        — Scope type + routing (project/global/all) + stable merge
  batch.go                        — Batch operations + char limit + dedup + substring matching
                                    + OverflowError struct
  prefetch.go                     — RenderFrozenSnapshot() (renders all, sorted by importance)
  distill.go                      — Background distill (one pass, scope-classifying, coalescing)
  facts_test.go                   — MODIFIED: scope, batch, overflow, dedup, substring, category
  scope_test.go                   — scope routing + trail semantics + fused search
  batch_test.go                   — batch operations + char limit + substring
  prefetch_test.go                — frozen snapshot rendering (format, separator, sorting)
  distill_test.go                 — distill (one-pass, coalescing, write approval)

mnemonic/internal/skills/
  execute.go                      — MODIFIED: log skill_usage rows on failure/timeout (C2)
  skills.go                       — MODIFIED: usage tracking includes outcome
  loop.go                         — Inspect + Amend + Evaluate (project-scoped)
  loop_test.go                    — skill loop tests (observe, inspect, amend, evaluate, rollback)

mnemonic/internal/mcp/
  tools_facts.go                  — MODIFIED: scope + target on ALL fact tools, batch ops,
                                    pending queue (pending_facts table)
  tools_skills.go                 — MODIFIED: skill inspect/amend/evaluate (project-scoped)
  tools_facts_test.go             — MODIFIED: scope, trail semantics, batch, pending
  tools_skills_test.go            — MODIFIED: skill loop tool tests

mnemonic/cmd/mnemonic/
  memory.go                       — MODIFIED: --scope, --target, pending/approve/reject, prefetch

plugins/opencode/
  src/memory-prefetch.ts          — NEW: OpenCode V2 plugin hook (idempotent, fail-open)

.skillgrid/decisions/
  04-adr-0049-global-operational-memory.md  — ADR
```

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Scope isolation (global vs project store) | Applicable: two stores, routing must not leak | `scope` param on all `fact_*` tools; routing in `facts/scope.go`; `__global__` skipped in `ListProjects`/`RunTTLExpiry` | `TestScopeIsolation`: `fact_add(scope=global)` → verify row in `global.sqlite`, NOT in project store. `fact_search(scope=project)` → verify no global hits. |
| Char limit overflow | Applicable: bounded store, error-on-overflow | Batch atomicity: limit checked on final result only. Error includes `current_entries` + `usage`. | `TestOverflowError`: add to 2150/2200 store → error with usage. `TestBatchFreeAndAdd`: free 200, add 250 → success (final 2150). |
| Fused search correctness | Applicable: two stores, stable merge by BM25 rank | `scope=all` opens both stores, stable-merges by each store's own rank, labels each hit with scope + category. Trail goes to project store only. | `TestFusedSearch`: seed both stores with facts matching the query → `scope=all` returns all, interleaved by rank, labeled correctly. `TestFusedSearchTrailGoesToProjectStore`. |
| Plugin hook (OpenCode V2) | Applicable: external plugin, prefix cache | Frozen snapshot at session start; `ctx.session.hook("context")` per ADR-0045. Mid-session writes don't change the prompt. | `TestPrefetchFrozen`: add fact after session start → verify system prompt unchanged. |
| Write approval gate | Applicable: passive writes can be staged | `write_approval: true` → distill writes go to `pending_facts` table (durable). Explicit writes go through freely. | `TestWriteApproval`: `write_approval=true` → distill `fact_add` → verify row in `pending_facts` (not `facts`). Restart server. Approve → verify row in `facts`, deleted from `pending_facts`. |
| Skill loop (amend → evaluate) | Applicable: self-modifying component | Amend writes to `<code_path>.amended` (not in-place). Evaluate compares outcome rate. Regression → discard. Git checkpoint = rollback. Observe leg: `execute.go` logs failure/timeout rows. | `TestObserveLogsFailures`: skill exits 1 → `skill_usage` row with `outcome='failure'`. `TestSkillEvaluateRollback`: declining outcome rate → amend → evaluate → verify original script restored. |
| Trail FK (global fact_add) | Applicable: `session_events` FK to `sessions` in the same store | Global `fact_add` trails to caller's project store (where the session exists). `facts.Route` returns (dataStore, trailStore). | `TestGlobalAddTrailSemantics`: global `fact_add` from project session → fact in `global.sqlite`, trail in project store. |
| Migration (072, 073) | Applicable: schema change, one-way door | `NOT NULL DEFAULT 'memory'` for category (no NULLs). `outcome`/`feedback` are nullable (existing rows = NULL). | `TestMigration072`: run on existing store → all existing facts have `category='memory'`. `TestMigration073`: run on existing store → all existing skill_usage rows have NULL outcome. |

## Door Check

**Task 1 Step 1** (before the migration) is the door check: can `store.Open(dataDir, "__global__")` create and open a file named exactly `__global__.sqlite`, and can `ListProjects` skip it? If `NormalizeID` mangles `__global__` or the file name is wrong, the pseudo-project approach needs revision. The migration (072) lands in the same task but is a separate one-way door — the approach is validated by the store test, the schema change is validated by the column test.

---

### Task 1: Migration 072 + Global Store Bootstrap

> ⚠ **one-way:** Migration 072 adds `category` column to `facts` table.

**Files:**
- Create: `mnemonic/internal/store/migrations/072_facts_category.sql`
- Create: `mnemonic/internal/service/global.go`
- Create: `mnemonic/internal/service/global_test.go`
- Modify: `mnemonic/internal/service/service.go:587` (ListProjects — skip `__global__`)
- Modify: `mnemonic/internal/service/service.go:2174` (RunTTLExpiry — skip `__global__`)

**Interfaces:**
- Consumes: `store.Open(dataDir, projectID)` (existing, `mnemonic/internal/store/store.go`)
- Produces: `Service.OpenGlobal() (*ProjectHandle, func(), error)` — returns a handle for `__global__`
- Seam: `service.go` — the `Open` method dispatches by projectID; `__global__` is a valid projectID
- Deletion test: if `global.go` is deleted, `OpenGlobal` disappears; callers (MCP tools, CLI) lose global scope
- Adapters: 2 (prod: `__global__` in `~/.skillgrid/mnemonic/`; test: temp dir with `__global__`)

**SATISFIES:** `global-store-bootstrap`, `list-projects-skips-global`, `ttl-expiry-skips-global`

- [ ] **Step 1: Write the failing test**

```go
// mnemonic/internal/service/global_test.go
func TestOpenGlobalCreatesStore(t *testing.T) {
    dataDir := t.TempDir()
    svc := NewService(t.Context(), dataDir)
    handle, release, err := svc.OpenGlobal()
    if err != nil {
        t.Fatalf("OpenGlobal: %v", err)
    }
    defer release()
    // Verify global.sqlite exists
    dbPath := filepath.Join(dataDir, "__global__.sqlite")
    if _, err := os.Stat(dbPath); os.IsNotExist(err) {
        t.Fatalf("global.sqlite not created at %s", dbPath)
    }
    // Verify migration 072 ran (category column exists)
    var colExists int
    handle.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('facts') WHERE name='category'`).Scan(&colExists)
    if colExists == 0 {
        t.Fatal("category column not found in facts table")
    }
}

func TestListProjectsSkipsGlobal(t *testing.T) {
    dataDir := t.TempDir()
    svc := NewService(t.Context(), dataDir)
    // Open a regular project
    h1, rel1, _ := svc.Open("proj-a")
    rel1()
    // Open global
    h2, rel2, _ := svc.OpenGlobal()
    rel2()
    // ListProjects should NOT include __global__
    projects, err := svc.ListProjects()
    if err != nil {
        t.Fatalf("ListProjects: %v", err)
    }
    for _, p := range projects {
        if p == "__global__" {
            t.Error("ListProjects includes __global__")
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/service/ -run TestOpenGlobal -v`
Expected: FAIL — `OpenGlobal` method not defined

- [ ] **Step 3: Write migration 072**

```sql
-- 072_facts_category.sql
-- Adds the category column to the facts table (ADR-0049, design decision 20).
-- category: 'memory' (agent's operational notes) | 'user' (user profile).
-- NOT NULL DEFAULT 'memory' — existing rows get 'memory'.

ALTER TABLE facts ADD COLUMN category TEXT NOT NULL DEFAULT 'memory';
```

- [ ] **Step 4: Write minimal implementation**

```go
// mnemonic/internal/service/global.go
package service

const GlobalProjectID = "__global__"

// OpenGlobal opens (creating if needed) the global store at
// <dataDir>/__global__.sqlite. Reuses the standard store.Open path.
func (s *Service) OpenGlobal() (*ProjectHandle, func(), error) {
    return s.openProject(GlobalProjectID, s.configRoot)
}
```

In `service.go`, modify `ListProjects` to skip `__global__`:
```go
// In ListProjects, after collecting project IDs:
for i, p := range projects {
    if p == GlobalProjectID {
        projects = append(projects[:i], projects[i+1:]...)
        i--
    }
}
```

Same for `RunTTLExpiry`: skip `__global__` in the project loop.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./mnemonic/internal/service/ -run "TestOpenGlobal|TestListProjects" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add mnemonic/internal/store/migrations/072_facts_category.sql
git add mnemonic/internal/service/global.go
git add mnemonic/internal/service/global_test.go
git add mnemonic/internal/service/service.go
git commit -m "feat(mnemonic): global store bootstrap + facts.category migration

- Migration 072: ADD COLUMN category TEXT NOT NULL DEFAULT 'memory' TO facts
- Service.OpenGlobal() → __global__ pseudo-project → global.sqlite
- ListProjects/RunTTLExpiry skip __global__

[skillgrid-context]
change: 2026-10-24-global-llm-wiki
task: A-T01
adr: 0049
"
```

---

### Task 2: Scope + Category Params on Fact Tools

**Files:**
- Create: `mnemonic/internal/facts/scope.go`
- Create: `mnemonic/internal/facts/scope_test.go`
- Modify: `mnemonic/internal/facts/facts.go` (Store gets category; Add/Search accept category + trailStore param)
- Modify: `mnemonic/internal/mcp/tools_facts.go` (scope + target params on all fact tools: add, search, forget, decay, decay_all)
- Modify: `mnemonic/internal/mcp/tools_facts_test.go`

**Interfaces:**
- Consumes: `Service.OpenGlobal()` (Task 1), `facts.Store.Add/Search/Forget/Decay` (existing)
- Produces: `facts.Scope` type (`project` | `global` | `all`), `facts.Route(scope, svc) (*facts.Store, *facts.Store, error)` — returns (dataStore, trailStore)
- Seam: `tools_facts.go` — the MCP handler resolves scope → stores before calling `facts.Store`
- Deletion test: if `scope.go` is deleted, MCP tools can't route; CLI can't accept `--scope`
- Adapters: 2 (prod: MCP handler; test: direct `facts.Route` calls)

**C3 resolution — trail semantics:** A global `fact_add` from a project session writes the fact to `global.sqlite` but the `session_events` trail row goes to the *project* store (where the session exists). `facts.Route` returns two stores: the data store (where facts live) and the trail store (where session_events go). For `scope=project`, both are the same store. For `scope=global`, dataStore=`__global__`, trailStore=caller's project store.

**M4 resolution — config root:** `OpenGlobal` uses `s.dataDir` as both data and config root (the global store loads config from `~/.skillgrid/config.d/indexing.yaml`, not the CWD). `__global__` is NOT run through `NormalizeID` — it's passed raw to `store.Open` (which only rejects empty/`..`). The file name is `__global__.sqlite` (pinned in Task 1's door check test).

**SATISFIES:** `scope-routing`, `category-filter`, `trail-semantics`

- [ ] **Step 1: Write the failing test**

```go
// mnemonic/internal/facts/scope_test.go
func TestScopeRouting(t *testing.T) {
    // scope=global → dataStore=__global__, trailStore=project store
    // scope=project → dataStore=project store, trailStore=project store (same)
    // scope=all → for search: both stores; for write: error (writes need explicit scope)
}

func TestAddWithCategory(t *testing.T) {
    // fact_add(scope=global, target=memory) → row in global.sqlite with category='memory'
    // fact_add(scope=global, target=user) → row in global.sqlite with category='user'
    // Trail row goes to the project store (where the session exists)
}

func TestGlobalAddTrailSemantics(t *testing.T) {
    // Open project store + global store
    // fact_add(scope=global, sessionID="proj-session-1")
    // Verify: fact row in global.sqlite
    // Verify: session_events trail row in project store (not global)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/facts/ -run TestScopeRouting -v`
Expected: FAIL — `facts.Scope` type not defined

- [ ] **Step 3: Write minimal implementation**

```go
// mnemonic/internal/facts/scope.go
package facts

type Scope string

const (
    ScopeProject Scope = "project"
    ScopeGlobal  Scope = "global"
    ScopeAll     Scope = "all"
)

func (s Scope) Valid() bool {
    return s == ScopeProject || s == ScopeGlobal || s == ScopeAll
}
```

Modify `facts.Store` to track category. Modify `Add` to accept a `category` param. Modify `Search` to filter by category when specified.

Modify `tools_facts.go` handlers to accept `scope` and `target` params, resolve the store, and pass category through.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/facts/ -run "TestScope|TestCategory" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/facts/scope.go mnemonic/internal/facts/scope_test.go
git add mnemonic/internal/facts/facts.go
git add mnemonic/internal/mcp/tools_facts.go mnemonic/internal/mcp/tools_facts_test.go
git commit -m "feat(mnemonic): scope + category params on fact_* tools

- facts.Scope type (project/global/all) + routing
- fact_add/fact_search accept scope + target (memory/user)
- MCP tools route to the correct store by scope

[skillgrid-context]
change: 2026-10-24-global-llm-wiki
task: A-T03
adr: 0049
"
```

---

### Task 3: Fused Search (Stable Merge)

**Files:**
- Modify: `mnemonic/internal/facts/scope.go` (add fused search)
- Modify: `mnemonic/internal/facts/facts_test.go`
- Modify: `mnemonic/internal/mcp/tools_facts.go` (scope=all path)

**Interfaces:**
- Consumes: `facts.Store.Search` (both stores)
- Produces: `facts.FusedSearch(ctx, query, limit string, stores map[Scope]*Store) ([]Fact, error)` — stable merge by each store's own BM25 rank, labels each with scope + category
- Seam: `scope.go` — the merge logic lives here; callers pass both stores
- Deletion test: if fused search is deleted, `scope=all` returns only project hits
- Adapters: 2 (prod: MCP handler; test: direct `FusedSearch` calls)

**M1 resolution — stable merge, not RRF:** The two stores are disjoint (a fact lives in exactly one store), so RRF's overlap-merging benefit doesn't apply. Use a stable merge: run both searches, interleave results by their respective BM25 rank (global rank 1, project rank 1, global rank 2, project rank 2, ...). Simpler, no tuning constant, testable.

**M2 resolution — trail for fused search:** `scope=all` search writes the `fact_search` trail to the *caller's project store* only (not both stores). The fused results are labeled with scope, so the agent knows which store each hit came from.

**SATISFIES:** `fused-search`

- [ ] **Step 1: Write the failing test**

```go
func TestFusedSearch(t *testing.T) {
    // Seed global store: "user prefers concise responses" (category=user)
    // Seed project store: "project uses Go 1.22" (category=memory)
    // Seed project store: "responses are too verbose" (category=memory)
    // Fused search "responses" → 2 hits: global (rank 1) + project (rank 1), interleaved
    // Fused search "Go" → 1 hit: project, labeled scope=project, category=memory
    // Fused search "all" → 3 hits, correctly labeled, interleaved by rank
}

func TestFusedSearchTrailGoesToProjectStore(t *testing.T) {
    // Fused search from a project session
    // Verify: fact_search trail row in project store, NOT in global store
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/facts/ -run TestFusedSearch -v`
Expected: FAIL — `FusedSearch` not defined

- [ ] **Step 3: Write minimal implementation**

RRF (Reciprocal Rank Fusion): `score(doc) = Σ 1/(k + rank_i)` where `k=60` (standard RRF constant). Each hit gets a `Scope` and `Category` label from the store it came from.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/facts/ -run TestFusedSearch -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/facts/scope.go mnemonic/internal/facts/facts_test.go
git add mnemonic/internal/mcp/tools_facts.go
git commit -m "feat(mnemonic): RRF-fused search across global + project stores

- facts.FusedSearch: RRF (k=60) over both stores
- Each hit labeled with scope + category
- scope=all MCP param triggers fused path

[skillgrid-context]
change: 2026-10-24-global-llm-wiki
task: A-T04
adr: 0049
"
```

---

### Task 4: Batch Operations + Char Limit + Dedup

**Files:**
- Create: `mnemonic/internal/facts/batch.go`
- Create: `mnemonic/internal/facts/batch_test.go`
- Modify: `mnemonic/internal/facts/facts.go` (char limit enforcement)
- Modify: `mnemonic/internal/mcp/tools_facts.go` (operations array)

**Interfaces:**
- Consumes: `facts.Store.Add/Forget/Decay` (existing)
- Produces: `facts.Batch(ctx, scope, target string, ops []Operation) (BatchResult, error)` — atomic batch, limit checked on final result
- Seam: `batch.go` — the batch logic lives here; MCP handler calls it
- Deletion test: if `batch.go` is deleted, overflow handling reverts to per-op check (broken)
- Adapters: 2 (prod: MCP handler; test: direct `Batch` calls)

**M6 resolution — config:** Char limits and write_approval are read from `config.d/indexing.yaml` (the existing config file the loader already reads). Add a `global_memory` section to that file. No new config file.

**M9 resolution — bounded store, not ranked:** The store is bounded at 2200/1375 chars (Hermes pattern). Prefetch renders *all* facts in the store (no top-k ranking — the store is small enough that everything fits). AKL decay still applies (purges low-importance facts to make room), but there's no "top-k by importance" selection. The `RenderFrozenSnapshot` function renders everything, sorted by importance for readability.

**Minor fix — substring matching:** `patch` and `remove` operations use `old_text` (unique substring) matching, not exact content. This is the Hermes pattern. The dedup check is on exact content for `add`; `patch`/`remove` match by substring.

**SATISFIES:** `batch-atomicity`, `char-limit-overflow`, `dedup`, `substring-match`

- [ ] **Step 1: Write the failing test**

```go
func TestBatchFreeAndAdd(t *testing.T) {
    // Store at 2100/2200
    // Batch: remove 200 chars + add 250 chars
    // Final: 2150/2200 → success
}

func TestOverflowError(t *testing.T) {
    // Store at 2150/2200
    // Add 100-char fact
    // Error: "Memory at 2,150/2,200 chars; adding this entry (100 chars) would exceed the limit by 50 chars."
    // Error includes current_entries (as JSON array) + usage
    // Error type: *facts.OverflowError (struct, not plain string)
}

func TestDedup(t *testing.T) {
    // Add "user prefers dark mode"
    // Add "user prefers dark mode" again
    // Second add: success with "no duplicate added"
}

func TestSubstringPatchRemove(t *testing.T) {
    // Add "User prefers dark mode in all editors"
    // Patch with old_text="dark mode in all editors", content="light mode in VS Code"
    // → "User prefers light mode in VS Code"
    // Remove with old_text="light mode"
    // → entry removed
    // Patch with old_text matching 0 entries → error with closest_entries
    // Patch with old_text matching 2 entries → error "ambiguous match"
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/facts/ -run "TestBatch|TestOverflow|TestDedup" -v`
Expected: FAIL — `facts.Batch` not defined

- [ ] **Step 3: Write minimal implementation**

```go
// mnemonic/internal/facts/batch.go
type Operation struct {
    Action  string `json:"action"`  // "add" | "patch" | "remove"
    Content string `json:"content,omitempty"`
    OldText string `json:"old_text,omitempty"` // for patch/remove
}

type BatchResult struct {
    Success      bool     `json:"success"`
    Error        string   `json:"error,omitempty"`
    CurrentEntries []string `json:"current_entries,omitempty"`
    Usage        string   `json:"usage,omitempty"`
}

func Batch(ctx context.Context, s *Store, target string, ops []Operation) BatchResult {
    // Apply ops in a transaction
    // Compute final char count
    // If over limit → rollback, return error with current_entries + usage
    // If under limit → commit, return success
    // Dedup: check for exact duplicate content before insert
}
```

Char limits: `memory` = 2200, `user` = 1375. Read from config (`config.d/mnemonic.yaml`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/facts/ -run "TestBatch|TestOverflow|TestDedup" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/facts/batch.go mnemonic/internal/facts/batch_test.go
git add mnemonic/internal/facts/facts.go
git add mnemonic/internal/mcp/tools_facts.go
git commit -m "feat(mnemonic): batch operations + char limit + dedup

- facts.Batch: atomic operations array, limit on final result
- Overflow error with current_entries + usage
- Exact duplicate rejection
- Char limits: memory=2200, user=1375

[skillgrid-context]
change: 2026-10-24-global-llm-wiki
task: A-T05
adr: 0049
"
```

---

### Task 5: Prefetch Injection (Plugin + MCP Resource + CLI)

**Files:**
- Create: `mnemonic/internal/facts/prefetch.go`
- Create: `mnemonic/internal/facts/prefetch_test.go`
- Create: `plugins/opencode/src/memory-prefetch.ts`
- Modify: `mnemonic/cmd/mnemonic/memory.go` (`skillgrid memory global prefetch`)
- Modify: `mnemonic/internal/mcp/server.go` (MCP resource `memory/global`)

**Interfaces:**
- Consumes: `facts.Store.List` (top-k by AKL importance), `Service.OpenGlobal`
- Produces: `facts.RenderFrozenSnapshot(ctx, store, category, limit) (string, error)` — the frozen block
- Seam: `prefetch.go` — the rendering logic; plugin, MCP resource, and CLI all call it
- Deletion test: if `prefetch.go` is deleted, no injection path works
- Adapters: 3 (plugin hook, MCP resource, CLI)

**M5 resolution — frozen is a property of the injection moment, not the renderer:** `RenderFrozenSnapshot` is a pure function of the store (it renders everything in the store). "Frozen" means the plugin hook fires *once per session* and caches the result. The Go test verifies the renderer's output format. The plugin test (Step 3b) verifies the hook fires once and caches.

**M8 resolution — plugin idempotency:** The OpenCode V2 `context` hook fires on every LLM request (primary, compaction, generate). The plugin must memoize by session ID: render once, cache, return cached on subsequent calls. Skip if the system prompt already contains the block. Fail-open: if the CLI call fails, return empty (don't break the session).

**SATISFIES:** `prefetch-injection`, `prefetch-frozen`

- [ ] **Step 1: Write the failing test**

```go
func TestRenderFrozenSnapshot(t *testing.T) {
    // Seed global store: 3 facts in 'memory', 2 facts in 'user'
    // Render → verify format:
    //   ════════════════════════════════════════
    //   GLOBAL MEMORY [xx% — N/2200 chars]
    //   ════════════════════════════════════════
    //   fact1
    //   §
    //   fact2
    //   ...
    //   ════════════════════════════════════════
    //   USER PROFILE [xx% — N/1375 chars]
    //   ════════════════════════════════════════
    //   ...
    // Separator is exactly "\n§\n" (newline, section sign, newline)
    // Entries sorted by importance_score DESC for readability
}
```

- [ ] **Step 1b: Write the plugin test (typecheck + fixture)**

```ts
// plugins/opencode/src/memory-prefetch.test.ts (or inline assertion)
// - Hook fires once per session (memoize by session ID)
// - Second call with same session ID returns cached result
// - CLI error → returns empty string (fail-open)
// - System prompt already contains the block → skip (idempotent)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/facts/ -run TestRender -v`
Expected: FAIL — `RenderFrozenSnapshot` not defined

- [ ] **Step 3: Write minimal implementation**

```go
// mnemonic/internal/facts/prefetch.go
func RenderFrozenSnapshot(ctx context.Context, s *Store, limit int) (string, error) {
    // Query top-k facts by importance_score (AKL)
    // Split by category: memory (limit 2200) + user (limit 1375)
    // Render each section with:
    //   - Header: ═══ + label + [xx% — N/LIMIT chars] + ═══
    //   - Entries separated by §
    //   - Multiline entries supported
    // Return the combined block
}
```

OpenCode V2 plugin (`memory-prefetch.ts`):
```ts
// plugins/opencode/src/memory-prefetch.ts
// Per ADR-0045: Plugin.define with ctx.session.hook("context")
// At session start: call skillgrid CLI `memory global prefetch`
// Inject the frozen block into the system prompt
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/facts/ -run "TestRender|TestPrefetch" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/facts/prefetch.go mnemonic/internal/facts/prefetch_test.go
git add plugins/opencode/src/memory-prefetch.ts
git add mnemonic/cmd/mnemonic/memory.go
git add mnemonic/internal/mcp/server.go
git commit -m "feat(mnemonic): prefetch injection (plugin + MCP resource + CLI)

- facts.RenderFrozenSnapshot: § separators, usage headers, two targets
- OpenCode V2 plugin: ctx.session.hook('context') per ADR-0045
- MCP resource: memory/global
- CLI: skillgrid memory global prefetch

[skillgrid-context]
change: 2026-10-24-global-llm-wiki
task: A-T06
adr: 0049
"
```

---

### Task 6: Background Distill + Write Approval

**Files:**
- Create: `mnemonic/internal/facts/distill.go`
- Create: `mnemonic/internal/facts/distill_test.go`
- Modify: `mnemonic/internal/mcp/tools_facts.go` (pending queue: `fact_pending_list/approve/reject`)
- Modify: `mnemonic/internal/facts/batch.go` (staged writes)

**Interfaces:**
- Consumes: `facts.Store.Add` (same as MCP `fact_add`), ADR-0023 LLM client
- Produces: `facts.Distill(ctx, sessionID, conversation) ([]Fact, error)` — extracts durable facts from conversation
- Produces: `facts.PendingList/Approve/Reject` — write approval queue
- Seam: `distill.go` — the distill logic; the 013 hook calls it
- Deletion test: if `distill.go` is deleted, passive extraction stops; explicit `fact_add` still works
- Adapters: 2 (prod: 013 hook; test: direct `Distill` calls)

**M3 resolution — pending queue is a table, not in-memory:** A 4-line migration (folded into migration 073) adds a `pending_facts` table to `global.sqlite`: `(id, content, target, created_at, status)`. Staged writes go into this table. `fact_pending_list` reads from it. `fact_pending_approve` moves the row into `facts` and deletes from `pending_facts`. Survives restart.

**Minor fix — coalescing:** One distill pass per session close. The LLM prompt classifies each fact as `project` or `global` scope. One LLM call, two stores. The prompt: "Extract durable facts. For each fact, specify scope: 'project' (specific to this project) or 'global' (cross-project, about the user or environment)."

**SATISFIES:** `distill-passive`, `write-approval-staging`, `pending-durable`

- [ ] **Step 1: Write the failing test**

```go
func TestDistillExtractsFacts(t *testing.T) {
    // Mock LLM returns: [{content: "user prefers concise", target: "user", scope: "global"}, {content: "project uses Go", target: "memory", scope: "project"}]
    // Distill → global fact in global.sqlite, project fact in project store
    // One LLM call (not two)
}

func TestWriteApprovalStages(t *testing.T) {
    // write_approval: true
    // Distill → fact_add → verify row in pending_facts table (not facts table)
    // Restart the server (new Service instance)
    // fact_pending_list → still shows the pending fact (durable)
    // fact_pending_approve → verify row in facts table, deleted from pending_facts
}

func TestCoalescingOnePass(t *testing.T) {
    // Session close → one distill run
    // Verify: one LLM call (mock counts calls)
    // Verify: facts routed to correct stores by scope
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/facts/ -run "TestDistill|TestWriteApproval" -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

```go
// mnemonic/internal/facts/distill.go
func Distill(ctx context.Context, llm LLMClient, sessionID, conversation string) ([]Fact, error) {
    // Prompt: "Extract durable facts from this conversation. Return JSON: [{content, target}]"
    // target: "memory" | "user"
    // For each fact: call Store.Add(scope=global, target)
    // If write_approval: true → stage instead of write
    // Coalescing: latest snapshot wins (check for existing pending distill for sessionID)
}
```

Pending queue: in-memory (per server lifetime) + `session_events` trail for audit.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/facts/ -run "TestDistill|TestWriteApproval" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/facts/distill.go mnemonic/internal/facts/distill_test.go
git add mnemonic/internal/mcp/tools_facts.go
git add mnemonic/internal/facts/batch.go
git commit -m "feat(mnemonic): background distill + write approval

- facts.Distill: LLM extracts durable facts from conversation (ADR-0023)
- write_approval: true → passive writes staged (pending queue)
- fact_pending_list/approve/reject MCP tools
- Coalescing: latest snapshot wins

[skillgrid-context]
change: 2026-10-24-global-llm-wiki
task: A-T07
adr: 0049
"
```

---

### Task 7: Skill Loop (Migration 073 + Observe Fix + Inspect + Amend + Evaluate)

> ⚠ **one-way:** Migration 073 adds `outcome` + `feedback` columns to `skill_usage` AND creates `pending_facts` table.

**C1 resolution — code-registry skills, not agent SKILL.md:** The skill loop targets the `skills` table (sandboxed scripts: `.skillgrid/files/skills/foo.sh`, `foo.py`, `foo.go`). `Amend` edits the script file at `code_path`. Agent SKILL.md files (`.agents/skills/`) are out of scope for this change.

**C2 resolution — observe leg fix:** `execute.go` currently only logs `skill_usage` rows on success (`exitCode == 0 && !timedOut`). Modify it to log on failure/timeout too, with `outcome` set to `'failure'` or `'error'` and `stderr` captured. This is a prerequisite for `Inspect` and `Evaluate` to work.

**M7 resolution — skill loop is project-scoped:** The skill loop operates on the *project's* skill registry (the `skills` table in the project store). There is no `__global__` skill registry. The MCP tool takes a `project` param (like the existing `use_skill`).

**Minor fix — stderr patterns:** `Inspect` groups by exact stderr string (top-N by frequency). Not TF-IDF, not LLM summarize. Simplest algorithm that works.

**Files:**
- Create: `mnemonic/internal/store/migrations/073_skill_usage_outcome.sql`
- Create: `mnemonic/internal/skills/loop.go`
- Create: `mnemonic/internal/skills/loop_test.go`
- Modify: `mnemonic/internal/skills/execute.go` (log usage rows on failure/timeout)
- Modify: `mnemonic/internal/skills/skills.go` (usage tracking includes outcome)
- Modify: `mnemonic/internal/mcp/tools_skills.go` (skill inspect/amend/evaluate)

**Interfaces:**
- Consumes: `skills.Store` (existing), `skill_usage` table (migration 073), ADR-0023 LLM client
- Produces: `skills.Inspect(ctx, db *sql.DB, skillName string) (EvidenceSummary, error)`
- Produces: `skills.Amend(ctx, codePath string, evidence EvidenceSummary, llm LLMClient) (string, error)` — returns path to `<code_path>.amended`
- Produces: `skills.Evaluate(ctx, db *sql.DB, skillID int64, amendmentTime time.Time) (EvaluationResult, error)` — before/after outcome rate
- Seam: `loop.go` — the loop logic; MCP tools call it
- Deletion test: if `loop.go` is deleted, skills are static (Voyager without the improvement loop)
- Adapters: 2 (prod: MCP tools; test: direct calls)

**SATISFIES:** `skill-observe`, `skill-inspect`, `skill-amend`, `skill-evaluate`

- [ ] **Step 1: Write the failing test**

```go
// mnemonic/internal/skills/loop_test.go
func TestMigration073(t *testing.T) {
    // Run migration on existing store
    // Verify outcome + feedback columns exist on skill_usage
    // Verify pending_facts table exists
    // Verify existing skill_usage rows have NULL outcome
}

func TestObserveLogsFailures(t *testing.T) {
    // Execute a skill that exits with code 1
    // Verify: skill_usage row exists with outcome='failure', stderr captured
    // Execute a skill that times out
    // Verify: skill_usage row exists with outcome='error', stderr captured
}

func TestInspectDetectsFailures(t *testing.T) {
    // Seed skill_usage: 5 runs, 3 failures with stderr "timeout", 2 successes
    // Inspect → evidence summary: 5 runs, 40% success, common_errors: ["timeout"] (top-N by frequency)
}

func TestAmendWritesNewFile(t *testing.T) {
    // Mock LLM returns a proposed patch
    // Amend → <code_path>.amended exists, original file unchanged
}

func TestEvaluateRollback(t *testing.T) {
    // Seed skill_usage: before amendmentTime, 80% success; after, 50% success
    // Evaluate → regression detected → .amended file discarded, original unchanged
}

func TestEvaluateApply(t *testing.T) {
    // Seed skill_usage: before amendmentTime, 50% success; after, 80% success
    // Evaluate → improvement → .amended replaces original, git commit
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/skills/ -run "TestMigration|TestInspect|TestAmend|TestEvaluate" -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

```sql
-- 073_skill_usage_outcome.sql
ALTER TABLE skill_usage ADD COLUMN outcome TEXT;      -- 'success' | 'failure' | 'error'
ALTER TABLE skill_usage ADD COLUMN feedback TEXT;     -- optional user/agent note
```

```go
// mnemonic/internal/skills/loop.go
type EvidenceSummary struct {
    TotalRuns    int     `json:"total_runs"`
    SuccessRate  float64 `json:"success_rate"`
    CommonErrors []string `json:"common_errors"`
    LastFailures []UsageRow `json:"last_failures"`
}

func Inspect(ctx context.Context, db *sql.DB, skillName string) (EvidenceSummary, error)
func Amend(ctx context.Context, codePath string, evidence EvidenceSummary, llm LLMClient) (string, error)
func Evaluate(ctx context.Context, db *sql.DB, skillID int64, amendmentTime time.Time) (EvaluationResult, error)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/skills/ -run "TestMigration|TestInspect|TestAmend|TestEvaluate" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/store/migrations/073_skill_usage_outcome.sql
git add mnemonic/internal/skills/loop.go mnemonic/internal/skills/loop_test.go
git add mnemonic/internal/skills/skills.go
git add mnemonic/internal/mcp/tools_skills.go
git commit -m "feat(mnemonic): skill improvement loop (observe → inspect → amend → evaluate)

- Migration 073: skill_usage.outcome + feedback
- skills.Inspect: aggregate usage, detect failure patterns
- skills.Amend: LLM proposes patch → SKILL.md.amended
- skills.Evaluate: before/after outcome comparison, rollback on regression
- MCP tools: skill inspect/amend/evaluate

[skillgrid-context]
change: 2026-10-24-global-llm-wiki
task: A-T09
adr: 0049
"
```

---

### Task 8: CLI + ADR-0049 + Final Tests

**Files:**
- Modify: `mnemonic/cmd/mnemonic/memory.go` (`--scope`, `--target`, `pending/approve/reject`)
- Create: `.skillgrid/decisions/04-adr-0049-global-operational-memory.md`
- Modify: `mnemonic/internal/facts/facts_test.go` (threat matrix RED cases)

**Interfaces:**
- Consumes: all of Tasks 1-7
- Produces: CLI surface + ADR
- Seam: `memory.go` — the CLI dispatches to the same functions as MCP tools
- Deletion test: if CLI is deleted, MCP still works; ADR is documentation
- Adapters: 1 (CLI → Go functions; no external seam)

**SATISFIES:** `cli-scope-target`, `cli-pending`, `adr-0049`

- [ ] **Step 1: Write the failing test**

```go
func TestCLIScopeGlobal(t *testing.T) {
    // skillgrid memory fact add --scope global --target memory "test fact"
    // Verify row in global.sqlite
}

func TestCLIPendingApprove(t *testing.T) {
    // write_approval: true
    // Distill → pending
    // skillgrid memory global pending → shows pending facts
    // skillgrid memory global approve all → verifies in store
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/cmd/mnemonic/ -run "TestCLI" -v`
Expected: FAIL

- [ ] **Step 3: Write CLI flags + ADR**

CLI:
```
skillgrid memory fact add --scope global --target memory "content"
skillgrid memory fact search --scope all "query"
skillgrid memory global prefetch
skillgrid memory global pending
skillgrid memory global approve all
skillgrid memory global reject all
```

ADR-0049: follows the existing ADR format (Context / Decision / Consequences). References ADR-0012, ADR-0048, ADR-0023, ADR-0045, ADR-0031.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/ -v` (full mnemonic package)
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mnemonic/cmd/mnemonic/memory.go
git add mnemonic/internal/facts/facts_test.go
git add .skillgrid/decisions/04-adr-0049-global-operational-memory.md
git commit -m "feat(mnemonic): CLI scope/target + ADR-0049 + threat-matrix tests

- CLI: --scope global --target memory|user, pending/approve/reject
- ADR-0049: global operational memory decision
- Threat-matrix RED tests: isolation, overflow, fused, injection shape, batch atomicity

[skillgrid-context]
change: 2026-10-24-global-llm-wiki
task: A-T10, A-T11
adr: 0049
"
```

---

## Self-Review

**1. Spec coverage:** Briefing A1-A8 all mapped to tasks:
- A1 (bootstrap) → Task 1 ✅
- A2 (scope + category) → Task 2 ✅
- A3 (fused search) → Task 3 ✅
- A4 (batch + char limit) → Task 4 ✅
- A5 (prefetch) → Task 5 ✅
- A6 (distill + write approval) → Task 6 ✅
- A7 (CLI + tests) → Task 8 ✅
- A8 (skill loop) → Task 7 ✅

**2. Must-haves coverage:** All 18 truths mapped to tasks. All 16 artifacts named. All 7 key links covered. All 3 one-way-door decisions flagged.

**3. One-way-door completeness:** Migration 072 (Task 1), Migration 073 (Task 7), `__global__` pseudo-project (Task 1) — all tagged.

**4. Placeholder scan:** No TBD, "implement later", or "similar to Task N" found.

**5. Type consistency:** `facts.Scope`, `facts.Batch`, `facts.RenderFrozenSnapshot`, `skills.Inspect/Amend/Evaluate` — consistent across tasks.

## Execution Handoff

**Slicing:** 8 tasks → invoke `skillgrid:slicing` to break into vertical tracer-bullet tickets with dependency edges and execution waves.

**Execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration
2. **Inline Execution** — batch execution with checkpoints

**Which approach?**
