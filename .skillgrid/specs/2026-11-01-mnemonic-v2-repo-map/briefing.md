# Mnemonic v2 — Repo-Map Function (requirements & intent)

> **STATUS:** `draft` (2026-11-01)

> **Role:** the *what* — requirements, intent, success criteria. The companion
> `blueprint.md` is the *how*. The research anchor the blueprint cites is
> `specs/03-repo-map/000-overview.md` + `findings.md` (the 7-reference fold).

## Problem / Intent

When an agent starts a session in a project it has worked in before, its first minute
is orientation: where do things live, what are the hubs, and what did we already
decide? Today that orientation is either nothing (a fresh context with only AGENTS.md
prose) or a pile of prose the agent must trawl (verbose context files that, per the
agentmd reference, *reduce* task success ~20%).

This change builds **FUNCTION 3 of 5** in mnemonic-v2: the **repo_map** — a single
read-only navigation tool over the repo's **code structure** (the code-index graph,
FUNCTION 2) **and its accumulated memory** (the memory observations, FUNCTION 1). It
ranks the file/symbol tree, labels it for coverage, flags the hubs, budgets its output
in tokens, and is **auto-injected as the agent's first-minute context** via the MCP
session-start hook.

The design was settled in a Plan-mode conversation that walked **seven open-source
references** and folded each in (`findings.md` is the digest): **aider** (signature
lines, dynamic token budget, chat-aware per-turn ranking), **nekocode/agent-codemap**
(the zero-setup floor), **Twigpine/openclaude** (PageRank over an IDF-weighted
reference graph, 1024/2048 token budgets, auto-inject + focus-as-boost),
**cyanheads/repo-map** (the per-file field contract for the emitted doc),
**mikiships/agentmd** (minimal-mode discipline + drift detection), **SarthakMishra/
codemap** (token-optimized docs + the toolkit boundary), and **JordanCoin/codemap**
(the coverage contract, hub warnings, the context protocol, real-rules resolution).

The load-bearing intent, in one line: **the map orients; the agent drills.** It is the
first verb in the retrieval-as-reasoning composition (code-index spec `000` §1.1), not
the whole composition.

## Purpose & Success Criteria

- **Purpose:** give the agent a ranked, coverage-labeled, hub-flagged, token-budgeted
  first-minute map over code structure + memory, auto-injected at session start,
  re-rankable per turn in <100ms, with a CI-able drift check.
- **Success criteria (numeric, from the references):**
  - Warm map / per-turn re-rank: **<100ms** (no re-parse). Cold build = the
    `code_index` cost (≤30s on 2000 files — a code-index acceptance, referenced, not
    re-tested).
  - Default `inject_tokens`: **1024** (aider's `--map-tokens` default), expanded
    **~2× on cold start**. `--tokens` default: **2048**.
  - Hub threshold: **3 importers** (JordanCoin default; config-overridable).
  - PageRank: damping 0.85, max 100 iters, ε=1e-6, **deterministic** (same run, same
    output).
  - Tier 1 emits **signature lines** (parameter + type), not bare identifiers —
    verifiable: the output for a file contains the full `func`/`def`/`function` line.
  - Chat-context prior: `context: ["internal/memory/pipeline.go"]` ranks `pipeline.go`
    + its direct dependents **above** the same call without context (rank delta > 0).
  - Coverage: a file with 3+ importers → `⚠️ HUB` tag + count; a code-index leg that
    times out → `coverage.status: partial` + `issues[]` naming the stale subtree, and
    the map does not read as "no dependencies".
  - `--drift` (the agentmd fold): exit 0 = doc fresh, exit 1 = doc stale vs. live
    state (stamp + content-hash comparison), CI-able JSON.
- **Out of scope:** RAG chat, commits, PRs (the toolkit boundary, the SarthakMishra
  fold); a second vector index (the map reads the code-index's existing vector store,
  D-CI8); the two deferred soft doors (a stored per-file LLM note fallback for files
  with zero memory; stored `code_ref` pointers instead of query-time FTS — both v2.1).

## Requirements

1. **R1 — PageRank + IDF + context-prior ranking (the openclaude + aider fold).**
   - **Current:** no ranking over the file reference graph; the `code_orient` map is
     depth-bounded but unranked.
   - **Target:** deterministic PageRank (damping 0.85, 100 iters, ε=1e-6) over the file
     reference graph, edges weighted by reference count × IDF(symbol name). The
     `context: []` param (recently-read paths/symbols) is a PageRank **prior**;
     `--focus <path>` is a **boost** (a prior on that subtree); `--focus <symbol>` is a
     **neighborhood** query (the cross-graph join). `--focus` is a ranking modifier,
     never a filter (the openclaude fold).
   - **Acceptance:** a seeded graph ranks deterministically (two runs, identical
     order); a common symbol name (`get`) ranks below a rare one at equal degree;
     `context: [path]` raises that path + its direct dependents' rank above the
     no-context run (delta > 0); a warm re-rank completes <100ms.
   - **Acceptance scenarios:** `ranking-deterministic`, `ranking-idf-downweights-common`,
     `ranking-context-prior-raises`, `rerank-under-100ms` → `acceptance.feature`.

2. **R2 — The three-tier renderer: signature lines, token budget, dynamic cold start.**
   - **Current:** `code_orient` returns a symbol TOC (names + kinds), no signature
     lines, no token budget.
   - **Target:** Tier 1 = the ranked symbol tree emitting **signature lines**
     (parameters + types, the `symbols.signature` column — the aider fold); Tier 2 =
     the memory map (topic hubs + per-file developer considerations); Tier 3 = the
     emitted doc. The **token budget is primary** (`--tokens N`, default 2048) with
     `limit` as a hard row cap; the walk is top-down over ranked files and **stops at
     the budget**. The auto-inject budget is `inject_tokens` (default 1024),
     **expanded ~2× on cold start** (session start, no files in context), shrunk
     mid-session (the aider dynamic-budget fold).
   - **Acceptance:** a file's output line contains its full `func`/`def`/`function`
     signature (not a bare name); the 1024-token walk stops at the budget (no
     over-run); the cold-start inject budget is ~2× the mid-session budget.
   - **Acceptance scenarios:** `signature-line-output`, `token-budget-walk-stops`,
     `dynamic-cold-start-budget` → `acceptance.feature`.

3. **R3 — The coverage contract on every answer (the JordanCoin fold).**
   - **Current:** `code_status` reports 4-state freshness + counts (T-CI-4), but a
     graph *read* does not say whether its graph is complete.
   - **Target:** every `repo_map` answer is a **versioned envelope**
     (`schema_version: repo-map/v1`) carrying `coverage: { status:
     complete|partial|unavailable, sources: [{name, status:
     authoritative|mixed|fallback|timeout|unavailable|failed, languages?, detail?}],
     issues: [{code, path, message}] }`. `sources[]` names the **per-language
     extraction tier** (D-TIER: `tree-sitter-rich` = authoritative, `tree-sitter-map` =
     mixed, `regex` = fallback). A timed-out/failed leg returns an **empty result with
     provenance** — not a silent empty graph and not a hard error: a `partial` graph
     never reads as a `complete` one, so "nothing imports this" and "I couldn't tell"
     are different answers (T-CI-4 applied per-answer).
   - **Acceptance:** a complete graph reports `status: complete` + all sources
     `authoritative`; a timed-out code-index leg reports `status: partial` + `issues[]`
     naming the stale subtree, and the map does not read as "no dependencies"; both
     legs missing reports `status: unavailable` + counts (never a silent empty graph).
   - **Acceptance scenarios:** `coverage-complete`, `coverage-partial-timed-out-leg`,
     `coverage-unavailable-both-missing` → `acceptance.feature`.

4. **R4 — The hub flag in the orientation view (the JordanCoin fold).**
   - **Current:** fan-in is visible only through `code_get_callers` counts / the
     blast-radius kind.
   - **Target:** files with fan-in ≥ `hub_threshold` (config, default **3** importers)
     carry a `⚠️ HUB` tag + the importer count **in the orientation view** (every map
     answer that lists that file), not just the blast-radius kind.
   - **Acceptance:** a seeded file with 3 importers carries `⚠️ HUB` + count 3 in the
     default `overview`; a file with 2 importers does not (the threshold is strict
     ≥3); `hub_threshold` is config-overridable.
   - **Acceptance scenarios:** `hub-flag-3-importers`, `hub-threshold-config` →
     `acceptance.feature`.

5. **R5 — The memory join: topic hubs + per-file developer considerations.**
   - **Current:** memory is queryable (`mem_lookup`) but never joined onto the code
     structure.
   - **Target:** Tier 2 = the memory map. The **topic hubs**: the `topic_key` tree →
     typed hubs → top-N by `importance`. The **per-file Developer Consideration**:
     saved observations **FTS-matched to the file's path/package** (the killer feature
     — the notes are accumulated memory, not LLM-generated; the cyanheads fold with
     memory as the source). The join is **query-time FTS** over
     `observations_fts` (the deferred v2.1 door is the stored `code_ref` pointer).
   - **Acceptance:** a seeded observation mentioning `internal/memory/pipeline.go`
     surfaces on `pipeline.go`'s row as its Developer Consideration; the `topics` kind
     returns the `topic_key` tree with the top-N by `importance`; a file with no
     matching observation gets no consideration line (not a fabricated one).
   - **Acceptance scenarios:** `memory-join-surfaces-seeded-observation`,
     `memory-topic-hubs-by-importance`, `memory-join-absent-not-fabricated` →
     `acceptance.feature`.

6. **R6 — The cross-graph kinds: focus/neighborhood, blast-radius, temporal.**
   - **Current:** the graph reads (`code_impact`, `code_path`) and the memory reads
     (`mem_lookup`) are separate verbs; no single call joins them.
   - **Target:** the `kind` parameter (progressive disclosure): `overview` (default),
     `packages`, `symbols`, `topics`, **`focus`/neighborhood** (a `--focus <symbol>`
     cross-graph join: the symbol's neighborhood from the graph + the observations
     that reference it), **`blast-radius`** (the code reverse-BFS via the existing
     `graph.Impact` + "which observations reference this"), **`temporal`** (weekly
     bins: code re-indexes from `files.indexed_at` + memory growth from
     `observations.created_at`).
   - **Acceptance:** a `blast-radius` kind on a seeded symbol returns the code
     reverse-BFS set + the observations that reference it; a `temporal` kind returns
     weekly bins for both legs; a `focus <symbol>` returns the symbol's neighborhood
     (the cross-graph join), distinct from a `focus <path>` boost (same argument,
     different shape → the MCP `mode` param removes the ambiguity).
   - **Acceptance scenarios:** `kind-blast-radius-cross-graph`,
     `kind-temporal-weekly-bins`, `focus-mode-disambiguation` → `acceptance.feature`.

7. **R7 — The context envelope + session-start hook + per-turn re-rank (the JordanCoin
   context-protocol + aider per-turn folds).**
   - **Current:** the MCP server has a write queue + the `code_*` family; no
     session-start injection, no per-turn re-rank.
   - **Target:** the session-start MCP hook call returns **one**
     `ContextEnvelope`: the map (Tier 1+2, token-budgeted at `inject_tokens`,
     expanded ~2× on cold start) + the coverage contract + the hub warnings + a memory
     summary (top-3 recent decisions + topic hubs) + the `--for <intent>` prior + the
     compact/full budget marker. The hook **re-ranks per turn** if the conversation
     context shifted (new files read / symbols mentioned) — the PageRank re-run with
     new priors, **<100ms**. Auto-injection is opt-in (a config flag); the on-demand
     `repo_map` MCP tool is always available.
   - **Acceptance:** the session-start call returns exactly one envelope containing the
     map + coverage + hubs + memory summary + intent prior; a per-turn re-rank after a
     new file is read completes <100ms and raises that file's rank; auto-injection is
     off when the config flag is off.
   - **Acceptance scenarios:** `context-envelope-one-call`,
     `session-start-auto-inject-flag`, `per-turn-rerank-under-100ms` →
     `acceptance.feature`.

8. **R8 — The `--emit` doc + `--drift` (the cyanheads + agentmd folds).**
   - **Current:** no emitted orientation doc; no drift check.
   - **Target:** `--emit` writes `.skillgrid/repo-map.md`: the cyanheads **per-file
     field contract** (Description = the signature-line floor, Developer
     Consideration = the memory observation, Maintenance Flag (Stable/Volatile/
     Generated/Unknown) from `files.content_hash` churn + the observation type mix,
     Architectural Role (Entrypoint/Service/API Route/Utility/Test/Configuration/…)
     from `edges` in/out-degree + kind, Critical Dependencies (top-N by confidence
     rank × fan-in, with one-line reasons)) + a **header stamp** (`generated_at` + a
     content hash). **Gitignored by default**; a config flag `repo_map.commit: true`
     opts in to committing. **`--drift`** (the agentmd fold): exit 0 = doc fresh, exit
     1 = doc stale vs. live state (stamp + content-hash comparison), CI-able JSON.
   - **Acceptance:** `--emit` writes the doc with all five per-file fields + the header
     stamp; the doc is gitignored by default and committed when `repo_map.commit` is
     true; `--drift` exits 0 on a fresh doc, 1 after the tree changes (the hash
     mismatch), and emits CI-able JSON.
   - **Acceptance scenarios:** `emit-doc-per-file-fields`, `emit-gitignore-default-
     commit-flag`, `drift-exit-codes` → `acceptance.feature`.

9. **R9 — The MCP tool + CLI group + AGENTS.md trigger-table + onboarding re-emit.**
   - **Current:** the MCP server's two-profile mechanism + the `code_*` family exist
     (code-index); the AGENTS.md block has a `### Memory` line but no map instruction.
   - **Target:** (1) the MCP `repo_map` tool (JSON; `mode`/`kind`/`focus`/`tokens`/
     `context`) on the agent profile; (2) the **session-start auto-injection hook**
     (the context envelope, the primary path); (3) the CLI `mnemonic repo map` (JSON +
     `--emit` + `--drift` + `--focus` + `--tokens` + `--kind` + `--context`); (4) the
     **emitted doc** re-emitted by onboarding when the code-index freshness flips
     `fresh→lag` or the memory schema version bumps. The **AGENTS.md instruction** is a
     **lean trigger-table** added INSIDE the existing `<!-- skillgrid:start -->` block
     in `.agents/skills/_shared/agent-config/block.md` (the single source of truth the
     CLI renders into every enrolled project): a compact When → Command table + a
     one-line trust rule, **not** a prose paragraph (block.md "Keep this block lean":
     navigation, not conventions; a line is added only on an observed failure; it
     loads on every prompt). The MCP hook is the PRIMARY path; the AGENTS.md
     instruction is the FALLBACK for harnesses without the hook. No new top-level
     section: it folds into the existing `### Memory` line or adds a minimal
     `### Code Map` sub-block of ≤6 lines.
   - **Acceptance:** the `repo_map` MCP tool + the CLI `mnemonic repo map` return
     byte-identical JSON for the same args; the block.md diff adds a ≤6-line
     trigger-table (no prose paragraph, no new top-level section); onboarding re-emits
     the doc on the `fresh→lag` flip.
   - **Acceptance scenarios:** `mcp-cli-byte-identical`, `agents-md-lean-trigger-table`,
     `onboarding-reemit-on-lag` → `acceptance.feature`.

10. **R10 — The doctor map-health surface + QA measurability.**
    - **Current:** `doctor` reports the store health; no map health.
    - **Target:** the `doctor` surface gains a **map-health** line (the coverage status
      of the last map read + the leg status: code-index present/lag/absent, memory
      present/empty). **QA measurability (the scenario the brief names):** a seeded
      observation for a seeded file surfaces in the map's consideration join (the
      end-to-end proof the memory join works against a real store).
    - **Acceptance:** `doctor` reports the map health (coverage status + leg status);
      a seeded observation for a seeded file surfaces in the map output (the QA
      scenario).
    - **Acceptance scenarios:** `doctor-map-health`, `qa-consideration-join-seeded` →
      `acceptance.feature`.

## Context

- **The memory change** (`2026-10-31-mnemonic-v2-memory`, in-flight) is the substrate:
  the `Store` interface (`internal/core/store/store.go`), the `SQLiteStore` with the
  `RawQuery`/`Probe`/`ProbeInt`/`RawExec` seams (`sqlite.go` — `RawQuery(query,
  args...) ([][]any, error)` for multi-row reads, `Probe(query, ...) (bool, error)`,
  `ProbeInt(query, ...) (int, error)`), the `observations` table (`sync_id`,
  `topic_key`, `importance`, `retrieval_usage`, `pinned`, `type`, `title`, `content`,
  `created_at`), the `observations_fts` FTS5 external-content table (trigram over
  `title, content, tool_name, type, project, topic_key`), the `sessions` +
  `session_trails` tables, and the `EventBus`/`SaveEvent` (the save handoff).
- **The code-index change** (`2026-10-31-mnemonic-v2-code-index`, queued) is the other
  leg: the `files`/`chunks`/`symbols`/`edges`/`index_freshness` tables (7 tables,
  additive), the `internal/codeindex/graph` package (`Orient`/`Impact`/`Callees`/
  `Callers`), the 4-state freshness model (`fresh`/`lag`/`empty`/`unknown`), and the
  D-TIER per-language extraction tier report.
- **The AGENTS.md block** (`.agents/skills/_shared/agent-config/block.md`) is the
  single source of truth the `skillgrid init`/onboarding CLI renders into every
  enrolled project's AGENTS.md. Its "Keep this block lean" rule: navigation, not
  conventions; reference don't write; grow only on an observed failure.
- **In-force ADRs:** the next ADR number is confirmed against
  `.skillgrid/ASSUMPTIONS.md` § LOCKED (highest in use: 0042, the goldfish plan review;
  the code-index change claims 0043–0046 for its tree-sitter/fsnotify/onnxer/Store-
  extension decisions — this change's new ADR, if any, starts at 0047).

## Implementation Decisions

- **No LLM in the first cut.** All per-file fields are derived deterministically from
  the graph + the memory (the minimal-mode discipline: the map is a read, not a
  generation).
- **No new tables, no new migration, no `Store` interface extension.** The read goes
  through the existing seams + the `graph` package (RM-INV-1; the ADR below).
- **The join mechanism is query-time FTS** over `observations_fts` (title/content/
  topic_key matched to the file's path/package). The stored `code_ref` pointer is the
  deferred v2.1 door.
- **The MCP hook is primary; the AGENTS.md trigger-table is the fallback.** A harness
  without the hook still gets the map via the lean table.

## Testing Decisions

- **What makes a good test:** a seeded code-index graph (files/symbols/edges) + a
  seeded memory store (observations with topic_key + path mentions) exercise the full
  composition end-to-end. The ranking is table-driven over small seeded graphs
  (deterministic: two runs, identical order). The coverage contract is table-driven
  over the three status values + the per-language tier. The token budget is asserted
  by walking a ranked file list and checking the walk stops at the budget.
- **Held-out tests (the reference's number is the acceptance target):** the
  <100ms re-rank (a benchmark-style assert over a seeded graph, no re-parse); the
  signature-line output (a file's row contains the full `func` line); the context
  prior (rank delta > 0 with vs. without `context`); the hub flag (3 importers → tag);
  the coverage `partial` (a timed-out leg + the `issues[]` naming the stale subtree);
  the drift exit codes (0 fresh, 1 stale).
- **Prior art:** the code-index change's held-out test pattern (T-CI-1…4) + the memory
  change's table-driven store tests.

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None (FUNCTION 3 of 5 is already the PRD's shape).
- `.skillgrid/ASSUMPTIONS.md`: one new ADR pointer row under `§ LOCKED` — the
  repo-map read surface is a view, not a store extension (no tables, no migration, no
  `Store` method; reads via the seams + the `graph` package). Record:
  `artifacts/04-adr-0047-repo-map-view-not-store.md` (the number is confirmed against
  the code-index change's 0043–0046 claim at T0).
- `.skillgrid/ARCHITECTURE.md`: None (no new component or data store; the new domain
  package is `mnemonic-v2/internal/repomap/`).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | ok     | the 7-reference fold is explicit   |
| Boundary Clarity    | 0.88  | 0.70 | ok     | map orients; agent drills          |
| Constraint Clarity  | 0.85  | 0.65 | ok     | no LLM, no tables, no Store ext    |
| Acceptance Criteria | 0.87  | 0.70 | ok     | all 10 requirements numeric        |
| **Clarity**         | 0.13  | ≤0.20| ok     |                                      |

**Interview log (the Plan-mode conversation that settled the design):**

| Round | Question summary                  | Decision locked                              |
|-------|-----------------------------------|----------------------------------------------|
| 1     | Code structure too, or memory only? | Both — the map is over code structure (the code-index graph) AND memory (the observations). |
| 2     | AGENTS.md every-run instruction?  | Yes — a lean trigger-table inside the existing `skillgrid:start` block + the MCP session-start hook as the primary path. |
| 3     | LLM in the first cut?             | No — the map is a pure read; per-file fields are derived from the graph + memory. The two LLM doors (stored note fallback, stored code_ref) are deferred to v2.1. |
| 4     | The join mechanism?               | Query-time FTS over `observations_fts` (the floor); the stored `code_ref` pointer is the deferred v2.1 door. |
| 5     | New tables / a Store extension?   | No — a pure view over the existing seams + the `graph` package (ADR-0047). |

## Open Questions & Assumptions

- **Assumption:** the code-index `graph` package exposes `Orient`/`Impact`/`Callees`/
  `Callers` with signatures that let the repo-map call them read-only (confirmed
  against the code-index blueprint's Task 4; the exact Go signatures are pinned at T0
  when code-index ships). If a method is missing, the repo-map reads the `edges`
  table through the `RawQuery` seam directly (RM-INV-1 still holds — no `Store`
  extension).
- **Assumption:** the `hub_threshold`, `inject_tokens`, `repo_map.commit`, and the
  auto-inject flag are config keys under the memory change's existing `config` load
  (no new config dep).
- **Question (open, non-blocking):** whether the `--for <intent>` prior should
  classify the intent (a light keyword match over the graph's package/symbol names)
  or be a raw string the PageRank prior consumes. Default for this change: a raw
  string consumed as a prior (no classifier — the minimal-mode discipline).

## Decisions (ADR)

- `artifacts/04-adr-0047-repo-map-view-not-store.md` — the repo-map read surface is a
  **view, not a store extension**: no new tables, no new migration, no `Store`
  interface method. The read goes through the existing `RawQuery`/`Probe`/`ProbeInt`/
  `RawExec` seams + the code-index `graph` package methods. The rejected alternative
  is a `repo_map_nodes` cache table with invalidation (or extending `Store` with
  repo-map methods) — a cache table adds a second source of truth + an invalidation
  bug class the index already owns (the code-index IS the cache, the openclaude fold);
  a `Store` extension reshapes a contract three functions share (the ADR-0045 one-way
  door) for a read that the seams already serve.

## Terms

- **the repo-map:** the first-minute orientation surface — a single read-only, ranked,
  token-budgeted view over the code-index graph + the memory store.
- **the coverage contract:** the versioned per-answer envelope
  (`status`/`sources`/`issues`) that makes a partial graph never read as a complete
  one (the JordanCoin fold; T-CI-4 applied per-answer).
- **Developer Consideration:** the per-file "the one thing a contributor is most likely
  to get wrong," sourced from a saved memory observation (FTS-matched to path/
  package) in the first cut (the cyanheads fold, memory as the source).
- **the context envelope:** the single session-start return (map + coverage + hubs +
  memory summary + intent prior + budget) that is auto-injected (the JordanCoin
  context protocol).
- **the two deferred soft doors:** (a) the stored per-file LLM note fallback
  (`file_notes`); (b) the stored `code_ref` pointer. Named, not built (v2.1).
