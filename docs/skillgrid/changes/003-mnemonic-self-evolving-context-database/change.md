# Change: 003-mnemonic-self-evolving-context-database — Mnemonic Self-Evolving Context Database

> **STATUS:** `draft` (2026-09-05) — revised after user-gate `questioning` (`interview.md`)
>
> **For agentic workers:** REQUIRED: follow `.agents/skills/_shared/conventions/sdd-structure.md`. This file is WHY + HOW (former intent + plan). Spec phase instantiates `tasks.md` + `acceptance.feature` from the Step Blueprint and per-step WHAT below.
>
> **Interview:** User-gate revise 2026-09-05 — decisions D1–D8 in `interview.md`. Legacy intent/plan folded earlier; this revise supersedes open questions where they conflict.

**Goal:** Upgrade Mnemonic from flat full-token loads to tiered L0/L1/L2 storage, semantic retrieval, explicit session compaction, and trail observability — without rewriting Go MCP + SQLite+FS.

**Architecture:** Keep Go MCP + SQLite+FS hybrid. Add a deep `tiered` module (Generate/Read/Register + non-blocking content-write seam), a Pure Go `Embedder` behind `MNEMONIC_EMBED` (no CGO/onnxruntime), separate `semantic_search` / `load_full_details` (L1-default + corpus filter, default LTM-only), and explicit `mnemonic_commit` that writes LTM/L2 then fires async tiering. Live external writers deferred; seam exercised via commit + `migrate --tier`. SQL `010_*`. Prefer after **002**; teams writers after **001**.

**Tech stack:** Go (`skillgrid-cli`), SQLite (`modernc.org/sqlite`), MCP (`mcp-go`), FS sidecars (`.abstract` / `.overview`), optional Pure Go embeddings behind `MNEMONIC_EMBED` (local preferred; remote HTTP optional; no CGO).

**Research:** none

**Ticket:** `none`

**Depends on:** Prefer after `002-mnemonic-identity-and-parity` (embeddings / vector helpers). Teams completion-hook / external writers after `001-open-sessions-agent-teams` when present — not a hard start gate for schema/FS/retrieval.

---

## Goal

Agents get overview-first recall with full detail on demand, operators can inspect retrieval trails when recall fails, and durable long-term memory is written only via explicit commit — all on the existing Go MCP + SQLite+FS stack.

## Out of scope / Non-Goals

- OpenCode plugin integration
- Python embedding service (Option B)
- CGO / onnxruntime (or other native) local embedders
- Rewriting FTS5, the code index, or the web research cache
- Cloud sync
- Auto-commit on every session end (only explicit `mnemonic_commit`; auto only as a future 001 completion hook)
- Tiering every `mem_save`
- Live external content-write producers this change (teams briefs, observation writers) — seam is ready; production call sites are `mnemonic_commit` + `migrate --tier` only
- Merging `semantic_search` into `mem_search` (separate tools; observation search stays `mem_search`)
- Hard UAT token-% targets (behavioral UAT only)
- Chained PRs (ship as one PR for this change)

## Definition of Done

This change is done only when **all** of the following are true:

- [ ] `migrate --tier` and `mnemonic_commit` create/register L2; async L0/L1 sidecars + path columns appear without blocking L2 durability
- [ ] `skillgrid migrate --tier` backfills L0/L1 without corrupting L2
- [ ] `semantic_search` returns ranked L1 (with abstracts), never L2 bodies, records a retrieval trail, and supports a corpus filter (**default = long-term memories only**; optional all `tiered_contents`)
- [ ] `load_full_details` returns L2 markdown for a search path
- [ ] `mnemonic_commit` persists long-term memory (L2 durable on success; L0/L1 async) with optional source link
- [ ] `skillgrid trail show|recent` shows query, directories, files, and result path
- [ ] Existing `mem_*` / `code_*` / `web_*` tools remain registered and unchanged in contract
- [ ] Every Step Blueprint entry has a matching section in `tasks.md` with Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing `@happy`, `@edge`, and `@failure` scenarios
- [ ] Applicable threat-matrix rows have RED coverage that passed
- [ ] Testing strategy commands below are green
- [ ] Rollback path below is still valid (or N/A documented)
- [ ] Change archived under `docs/skillgrid/archive/003-mnemonic-self-evolving-context-database/`

---

## Problem / why

Mnemonic is flat: full-token loads, fragmented search, no durable compaction, and opaque retrieval. Agents burn context on L2 every time; operators cannot see which directories/files a recall touched. Urgency is high — this builds on **002** (embeddings) and optionally **001** (teams) without waiting on a full rewrite.

## Target users

- **Agent** — overview-first recall; full details on demand
- **Operator** — inspect directories/files when recall fails
- **Urgency:** High — flat loads and opaque trails are daily friction for every Mnemonic consumer

## Business rules

- Modules only; keep Go MCP + SQLite+FS hybrid (no rewrite)
- Pure Go embeddings (Option A); no Python; no CGO/onnxruntime. Feature-flag — offline Pure Go local preferred, optional remote HTTP
- Additive SQL only; content on FS with path columns
- Separate `semantic_search` from `mem_search`; default search body is L1; L2 only via `load_full_details`
- `semantic_search` corpus filter: default **LTM-only**; optional all registered `tiered_contents`
- This change’s producers: `migrate --tier` + `mnemonic_commit` only; external/live writers deferred (seam ready for 001+)
- `mnemonic_commit` explicit; success when LTM row + L2 durable; L0/L1 async (`warn+continue`); no auto-commit on session end
- Auto-tiering must not block writes
- SQL numbering: leave `009_*` for 001; this change ships `010_*`
- Delivery: one PR for steps 01–05
- OpenCode plugin deferred; UAT behavioral (no hard token-%); glossary terms defined in this change

## In scope

- Schema: tier paths, `long_term_memories`, `retrieval_trails`, `path_embeddings` (`010_*`)
- Tiered FS + `migrate --tier` + content-write seam (hook called from `mnemonic_commit`; tested in step 02)
- MCP: `semantic_search` (corpus filter), `load_full_details`, `mnemonic_commit`
- Trail logging + `skillgrid trail` CLI

## Risks & rollback

- **Risk:** Summarization cost/latency — **Mitigation:** Async hooks after commit; `migrate --tier`; stub/heuristic Summarizer
- **Risk:** Couples to teams (001) — **Mitigation:** Generic `tiered_contents` + seam; no teams writers this change
- **Risk:** Embedding quality — **Mitigation:** Pure Go adapter (stub or Pure Go lib); title+L0 fallback when flag off
- **Risk:** Large multi-step change / one PR — **Mitigation:** Five vertical steps with per-step commits; DoD UAT-measurable
- **Risk:** Default LTM-only hides migrate-backfilled paths — **Mitigation:** Corpus filter to all `tiered_contents`; document default
- **Rollback:** Remove new migrations (`010_*`), MCP tools, CLI (`migrate` / `trail`), and tiered/embedder helpers. L2 content stays; drop `.abstract`/`.overview` sidecars if needed. Existing DBs that already applied migrations keep additive tables (harmless if callers stop using them).

## Error handling

| Failure | Behavior | Notes |
|---------|----------|-------|
| Migration fails mid-way | `abort` | Prior observations, FTS, and code index remain readable |
| Summarizer failure during tier generation | `warn+continue` | L2 intact; failure logged; no partial corrupt sidecars as success |
| Embedder unavailable / `MNEMONIC_EMBED` off | `warn+continue` | Title/L0 fallback ranking; trail still recorded |
| `load_full_details` unknown path | `abort` | Clear not-found; no invented content |
| `mnemonic_commit` missing required sources | `abort` | Clear error; no partial `long_term_memories` row |
| Tier generation after successful commit | `warn+continue` | Commit already succeeded; never fail commit solely because tiers lagged |
| Empty retrieval trails store | `warn+continue` | `trail recent` returns empty list, not error |
| Unknown trail id | `abort` | not-found for `trail show` |
| Content-write hook / background tiering | `warn+continue` | Never block the L2 write path |

## Testing strategy

- **Unit:** `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/tiered/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/` — Expected: PASS
- **Integration / acceptance:** `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/cmd/skillgrid/ ./skillgrid-cli/internal/mnemonic/integration/` plus BDD `@step-NN` scenarios in `acceptance.feature` — Expected: PASS (`@step-01`…`@step-05` / `@p0`)
- **Full suite:** `Run: go test ./...` (from repo root / `skillgrid-cli` per module layout) — Expected: PASS
- **Green means:** migrate + commit producers, L1-default search with corpus filter + trails, async tiers, trail CLI; existing `mem_*` contracts unchanged

---

## Step Blueprint

Contract for `sdd-spec`. Do not renumber after `tasks.md` exists. Per-step Out of scope / DoD live under Per-step WHAT (table is summary only).

| NN | Step slug | Goal (one line) | Primary package / entry | Depends on |
|----|-----------|-----------------|-------------------------|------------|
| 01 | `schema-extensions` | Additive SQL for tier paths, long-term memories, trails, embeddings | `skillgrid-cli/internal/mnemonic/store/migrations` | — |
| 02 | `tiered-storage` | L0/L1/L2 FS, tiered module, `migrate --tier`, background hooks | `skillgrid-cli/internal/mnemonic/tiered` | 01 |
| 03 | `semantic-retrieval` | Pure Go embeddings + search/load tools + trail logging | `skillgrid-cli/internal/mnemonic/mcp` | 02 |
| 04 | `session-compaction` | Explicit `mnemonic_commit` → long-term memories with tiers | `skillgrid-cli/internal/mnemonic/service` | 03 |
| 05 | `trail-observability` | `skillgrid trail show\|recent` CLI | `skillgrid-cli/cmd/skillgrid` | 04 |

---

## Technical approach

Add **Tiered Storage**, separate **Semantic Search** (not `mem_search`), compaction into **Long-term Memory**, and **Retrieval Trail** on Go MCP + SQLite+FS. Reuse 002 vector helpers / `MNEMONIC_EMBED`. No CGO embedder. No **001** writers yet: seam ready; producers are `migrate --tier` + `mnemonic_commit` (async tiers). SQL at `010_*`. One PR for 01–05.

## Architecture decisions

### Decision: Generic tier registry

**Module / Interface / Seam / Adapter / Depth:** Module = `tiered`; Interface = Generate/Read/Register; Seam = content-write; Adapter = FS+SQL; Depth = deep
**Choice:** `tiered_contents` + `{path}.abstract` / `.overview` sidecars; live external writers deferred
**Alternatives considered:** `ALTER tasks` now; tier every `mem_save`; broad allowlist of observation writers
**Rationale:** Interview D2 — prove FS + migrate + commit; attach 001+ writers later

### Decision: Pure Go embedder behind flag (no CGO)

**Module / Interface / Seam / Adapter / Depth:** Module = embedder; Interface = `Embedder.Embed`; Seam = real; Adapter = Pure Go local (+ optional remote HTTP); Depth = deep
**Choice:** Option A Pure Go only; `path_embeddings`; reuse 002 blob/cosine helpers; pick stub or Pure Go lib at apply
**Alternatives considered:** Python service; onnxruntime/CGO; always-on remote
**Rationale:** Interview D5′ — keep single-binary story; title/L0 floor when flag off

### Decision: Explicit commit + async tiering via seam

**Module / Interface / Seam / Adapter / Depth:** Module = compaction; Interface = commit; Seam = ContentWriteHook after L2; Adapter = stub/heuristic Summarizer; Depth = deep
**Choice:** Explicit `mnemonic_commit`; success on LTM+L2 durable; fire-and-forget `GenerateTiers`; auto only as future 001 hook
**Alternatives considered:** Every session end; sync await tiers; test-only seam with no product caller
**Rationale:** Interview D4 + D7 — thin producer + non-blocking tiers

### Decision: Separate L1 search with corpus filter + trail

**Module / Interface / Seam / Adapter / Depth:** Module = retrieval tools; Interface = search/load; Seam = trail logger; Adapter = SQLite; Depth = deep
**Choice:** New `semantic_search` / `load_full_details` (not folded into `mem_search`); ranked L1; corpus filter default LTM-only, optional all `tiered_contents`; log **Retrieval Trail**
**Alternatives considered:** Return L2; fold into `mem_search`; search LTM-only with no filter; defer trails
**Rationale:** Interview D1 + D8 — clear L1-only threat surface; migrate-backfill visible when filter widened

## Data flow

```mermaid
flowchart TD
  migrate["migrate --tier"] --> l2["FS L2 full markdown"]
  commit["mnemonic_commit"] --> ltm["long_term_memories + L2"]
  ltm -->|"async GenerateTiers"| sidecars["L0 .abstract + L1 .overview + SQL paths"]
  migrate -->|"GenerateTiers"| sidecars
  agent["Agent MCP"] --> search["semantic_search"]
  search --> corpus["corpus filter: LTM default / all tiered"]
  corpus --> embed["Embedder / title+L0 fallback"]
  embed --> l1["Ranked L1 + trail row"]
  agent --> load["load_full_details"]
  load --> l2
  op["Operator CLI"] --> trail["trail show|recent"]
  trail --> trails["retrieval_trails"]
  search --> trails
```

## File layout

```
skillgrid-cli/
├── cmd/skillgrid/
│   ├── main.go                 # Dispatch migrate + trail
│   ├── migrate.go              # runMigrate --tier
│   └── trail.go                # trail show|recent
└── internal/mnemonic/
    ├── store/migrations/
    │   └── 010_tiered_context.sql
    ├── tiered/
    │   ├── tiered.go           # Generate/read L0/L1/L2 + register
    │   ├── summarizer.go       # Summarizer interface + adapters
    │   └── hook.go             # Non-blocking ContentWriteHook
    ├── embedder/
    │   └── embedder.go         # Embedder + local/remote adapters
    ├── memory/
    │   └── embedding.go        # Shared Vector/blob/cosine with 002
    ├── service/
    │   ├── service.go          # Search/load/trail facade
    │   └── compaction.go       # Commit → long_term_memories
    └── mcp/
        ├── server.go           # Register retrieval + compaction
        ├── tools_retrieval.go  # semantic_search, load_full_details
        └── tools_compaction.go # mnemonic_commit
```

## Impacted files map

| File | Action | Step | Description |
|------|--------|------|-------------|
| `skillgrid-cli/internal/mnemonic/store/migrations/010_tiered_context.sql` | Create | 01 | `tiered_contents`, `long_term_memories`, `retrieval_trails`, `path_embeddings` |
| `docs/skillgrid/agents/glossary/technical.md` | Modify | 01 | Tiered Storage / Semantic Search / Retrieval Trail terms (apply-time; not edited in this migration) |
| `docs/skillgrid/agents/glossary/business.md` | Modify | 01 | Long-term Memory term (apply-time; not edited in this migration) |
| `skillgrid-cli/internal/mnemonic/tiered/tiered.go` | Create | 02 | Generate/read L0/L1/L2 |
| `skillgrid-cli/internal/mnemonic/tiered/summarizer.go` | Create | 02 | Summarizer **Interface** + adapters |
| `skillgrid-cli/internal/mnemonic/tiered/hook.go` | Create | 02 | Non-blocking content-write **Seam** |
| `skillgrid-cli/cmd/skillgrid/migrate.go` | Create | 02 | `runMigrate` (`--tier`) |
| `skillgrid-cli/internal/mnemonic/embedder/embedder.go` | Create | 03 | Embedder + local/remote adapters |
| `skillgrid-cli/internal/mnemonic/memory/embedding.go` | Modify | 03 | Share vector helpers with embedder |
| `skillgrid-cli/internal/mnemonic/service/service.go` | Modify | 03 | Search/load/trail facade |
| `skillgrid-cli/internal/mnemonic/mcp/tools_retrieval.go` | Create | 03 | `semantic_search`, `load_full_details` |
| `skillgrid-cli/internal/mnemonic/mcp/tools_compaction.go` | Create | 04 | `mnemonic_commit` |
| `skillgrid-cli/internal/mnemonic/service/compaction.go` | Create | 04 | Commit → `long_term_memories` |
| `skillgrid-cli/internal/mnemonic/mcp/server.go` | Modify | 04 | Register retrieval + compaction |
| `skillgrid-cli/cmd/skillgrid/trail.go` | Create | 05 | `trail show\|recent` |
| `skillgrid-cli/cmd/skillgrid/main.go` | Modify | 05 | Dispatch migrate + trail |

Verified present today: migrations through `008_*.sql`; `cmd/skillgrid/main.go`; `internal/mnemonic/{store,memory,mcp,service}` including `memory/embedding.go`. Not yet present (expected Create): `tiered/`, `embedder/`, `migrate.go`, `trail.go`, `010_*.sql`, retrieval/compaction tool files.

## Per-step WHAT

Observable behavior each step must deliver (feeds Gherkin). Not implementation HOW.

### Step 01 — `schema-extensions`

**Goal:** Additive SQL for tier paths, long-term memories, trails, embeddings
**Out of scope:** FS sidecars, MCP tools, CLI
**Definition of Done:** Store open creates tier/memory/trail/embedding tables without rewriting existing rows; migrate 008→010 is idempotent

- As operator: store open creates tier/memory/trail/embedding tables without rewriting existing rows
- Given existing DB, when migrate runs, observations/FTS/code index stay intact
- Edge: schema 008 → 010 once; re-open idempotent
- Failure: failed migration leaves prior data intact

### Step 02 — `tiered-storage`

**Goal:** L0/L1/L2 FS, tiered module, `migrate --tier`, non-blocking content-write seam
**Out of scope:** Semantic search tools, compaction MCP, trail CLI, live external writers
**Definition of Done:** Seam + migrate yield sidecars without blocking; summarizer failure preserves L2; product wire of seam is step 04 (`mnemonic_commit`)

- As writer via content-write seam (test harness in this step): L2 save later yields L0/L1 sidecars + path columns without blocking
- Given L2 files, `skillgrid migrate --tier` backfills L0/L1; L2 unchanged
- Edge: summarizer failure leaves L2 intact; error logged

### Step 03 — `semantic-retrieval`

**Goal:** Pure Go embeddings + search/load tools + trail logging, L1-default results
**Out of scope:** `mnemonic_commit`, trail CLI, changing existing `mem_*` contracts, CGO embedders
**Definition of Done:** Search returns ranked L1 only; corpus filter defaults to LTM; load returns L2; embeddings-off fallback still records trail

- As agent: `semantic_search` returns ranked L1 (with abstracts), not L2
- Given default corpus, results are from long-term memories only; with filter widened, registered tiered paths are included
- Given a result path, `load_full_details` returns L2 markdown
- Edge: embeddings off/empty → title/L0 fallback; trail still recorded
- Failure: unknown path rejects full-detail load
- **Threat (Mnemonic tool surface):** `semantic_search` body is L1-only; L2 only via `load_full_details`

### Step 04 — `session-compaction`

**Goal:** Explicit `mnemonic_commit` → long-term memories; hooks async tiering
**Out of scope:** Trail CLI; auto-commit on session end; external writers beyond commit
**Definition of Done:** Explicit commit persists LTM + durable L2; tiers async; session end alone does not write; missing sources abort; `mem_save` still registered

- As agent: `mnemonic_commit` persists **Long-term Memory** with durable L2 and optional source link; L0/L1 appear asynchronously via the content-write seam
- Session end alone does not auto-commit
- Edge: missing sources → clear error; no partial write
- Edge: summarizer/tier failure after commit does not roll back the LTM/L2 write
- **Threat (Mnemonic tool surface):** after registering new tools, existing `mem_save` remains registered and callable

### Step 05 — `trail-observability`

**Goal:** `skillgrid trail show|recent` CLI
**Out of scope:** New MCP tools; schema changes
**Definition of Done:** recent/show expose query paths; empty store lists nothing; unknown id is not-found

- As operator: `trail recent` / `trail show <id>` show query, directories, files, result path
- Empty store → empty list, not error
- Edge/failure: unknown id → not-found

## Threat matrix

Mark each row `Applicable` or `N/A: reason`. Applicable rows name an owning step and propagate into RED tasks + acceptance scenarios.

| Boundary / threat | Applicable? | Owning step | Planned RED coverage |
|-------------------|-------------|-------------|----------------------|
| Documentation-like paths | N/A: no executable classification | — | — |
| Git repository selection | N/A: no git cwd authority | — | — |
| Commit state | N/A: no git commit automation | — | — |
| Push state | N/A: no push automation | — | — |
| PR commands | N/A: no PR CLI | — | — |
| Mnemonic tool surface (`mem_*` / `code_*` / `web_cache_*`) | Applicable | 03 | `semantic_search` response body is L1-only (overview + abstract; no L2 markdown body); L2 reachable only via `load_full_details{path}` |
| Mnemonic tool surface (`mem_*` / `code_*` / `web_cache_*`) | Applicable | 04 | After registering new tools, existing `mem_save` remains registered and callable (additive-only; no `mem_*` contract break) |
| Shared-convention drift | N/A: no `_shared/conventions/*` edits this phase | — | — |

## Migration / rollout

- **Legacy supersession:** Legacy `intent.md`, `plan.md`, `*-glossary-reference.md`, and `steps/` were folded into this `change.md` plus change-level `tasks.md` + `acceptance.feature`. **User-gate questioning 2026-09-05** (`interview.md`) revised producers, embedder, corpus filter, and delivery.
- Additive `010_*`; leave `009_*` for 001. `MNEMONIC_EMBED` (+ optional remote HTTP); Pure Go only.
- Backfill via `migrate --tier`; commit hooks async tiers for new LTM L2.
- Prefer after 002; external/teams writers after 001.
- Ship steps 01–05 as **one PR** (per-step commits still recommended inside the branch).

## Open questions

- [ ] Exact Pure Go embedder adapter for apply (hash/deterministic stub vs a named Pure Go library) — pick in step 03; does not block schema/FS.
- [x] ~~onnxruntime-go vs stub~~ — closed: Pure Go only; no CGO/onnxruntime (interview D5′).
- [x] ~~Search surface / tier writers / delivery~~ — closed in `interview.md` (D1–D8).

## Glossary

| Term | Definition | Glossary file |
|------|------------|---------------|
| **Change** | A self-contained SDD unit tracked by a 3-digit NNN number, authored as `change.md` through archive. | business |
| **Step** | A sequenced unit of an SDD change that ships one testable behaviour slice. | business |
| **Tiered Storage** | L0/L1/L2 filesystem layout plus SQL path columns for progressive recall. | technical |
| **Semantic Search** | Ranked L1 retrieval via embeddings with title/L0 fallback when embeddings are off or empty. | technical |
| **Retrieval Trail** | Logged directories, files, query, and result path for each semantic search. | technical |
| **Long-term Memory** | Durable compacted memory persisted by explicit `mnemonic_commit` with L0/L1/L2 paths. | business |
| **Module** | Anything with an interface and an implementation; scale-agnostic. | technical |
| **Interface** | Everything a caller must know to use a module correctly (e.g. Summarizer / Embedder / ContentWriteHook). | technical |
| **Seam** | The location at which a module's interface lives (content-write and embedder seams). | technical |
| **Adapter** | A concrete implementation behind an interface (local/remote embedder; heuristic/stub summarizer). | technical |
| **Depth** | How much a module hides behind its interface; deep modules preferred for tiered + retrieval. | technical |
| **Topic Key** | Stable string used to upsert Mnemonic observations (`sdd/003-…/{change,tasks,spec}`). | technical |
| **Artifact Store Mode** | Persistence contract for SDD artifacts; `hybrid` (filesystem + Mnemonic) only. | technical |

<!-- Fold new terms here; also upsert docs/skillgrid/agents/glossary/{business,technical}.md at apply-time. No companion *-glossary-reference.md. -->

## Author self-review

- [x] **Goal**, **Out of scope / Non-Goals**, and **Definition of Done** are filled and testable
- [x] **Error handling** and **Testing strategy** are filled
- [x] Non-goals match Global Constraints that will appear in `tasks.md`
- [x] Rollback plan is present
- [x] Step Blueprint covers a vertical-slice sequence (no horizontal-only layers)
- [x] Every Impacted Files row maps to exactly one step
- [x] Every applicable threat row names an owning step
- [x] Glossary terms reused or defined; no companion reference file
