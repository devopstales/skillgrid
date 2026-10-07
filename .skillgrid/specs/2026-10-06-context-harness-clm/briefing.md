# Context Harness (with CLM) — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-06)
> **Tier:** T2
> **Path:** New Function (architecture in place; this adds a new subsystem)

## Problem / Intent

Large tool output (a `grep -r` across the repo, a `Read` of a 2,000-line file, a
`WebFetch` of a dense article) flows into the agent's context raw, and context bloat
accumulates until the harness's automatic compaction fires — which summarizes the raw
transcript and discards any edits the agent may have made. Mnemonic's `session_inject`
package retrieves a session's *prior* context but captures nothing and gives the agent no
way to manage its *current* context. This change builds a **Context Harness** that owns
the full context lifecycle — intercept-and-abstract capture, structured query, proactive
index, routing, and a `ctx` CLI — and extends it with the **Context Language Model**
(CLM) layer so the agent gets write access to its own context (arXiv:2609.37725, pi-clm).

## Purpose & Success Criteria

- **Purpose:** Give the agent one owner for its context lifecycle: large output is
  abstracted at the boundary (stored + pointer), the agent can query captured/indexed
  content deterministically, and — when CLM is opted in — the agent can edit its own
  context with ordinary file tools while a Go-side overflow guard keeps the request
  within budget.
- **Success criteria:**
  - Large tool output (> ~4KB) is captured to a session-scoped `tool_outputs` FTS store
    and the agent receives a 200-char summary + a `ctx_search` pointer; small output
    flows through unchanged.
  - `ctx_query` answers counts/lists/existence deterministically over the code index
    with no JS and no line ranges in v1.
  - `ctx index <path>` writes to a project-scoped `indexed_files` table reusing the
    observations schema shape; `ctx_search` fuses `tool_outputs` + `indexed_files` via RRF.
  - `ctx stats/index/search/purge` work from the CLI.
  - A ~80-word Context Routing block is injected by `skillgrid prime`.
  - When CLM is on: a `0600` mirror is rendered before each request; the agent edits it;
    at turn-end the edit is validated and persisted as a `context_revisions` row; the
    overflow guard withholds the oldest tool results when the estimated request exceeds
    `budget − reserve`; calibration corrects the size estimate against the provider count.
  - CLM is off by default; turning it on changes nothing about the existing
    `mem_inject_session` contract.
- **Out of scope:** TUI panel, model-driven compact (`/ctx-compact`), cross-session
  revision history, LLM summarization, line-range `ctx_query`, PreToolUse prediction,
  modifying the `mem_inject_session` MCP contract, a new retrieval engine for
  `indexed_files` (it reuses the observations shape).

## Context

Relevant existing flows this feature must respect:

- **`session_inject` is absorbed, not replaced.** Its public functions
  (`AutoPrepend`, `HybridRetrieve`, `RenderContextBlock`, `EstimateTokens`,
  `ObservationInjectable`) are imported by `cmd/skillgrid/loop_cmd.go`,
  `internal/mnemonic/secondbrain/ask.go`, `internal/mnemonic/mcp/tools_session_inject.go`,
  and `internal/mnemonic/http/toolevents.go`. They move into `context_harness` but keep
  their signatures so importers change only their import path. The `mem_inject_session`
  MCP tool is preserved verbatim (name, output shape, behavior).
- **Capture path.** `hooks/tool-call-capture.js` (PostToolUse) currently truncates
  `tool_output` to `MAX_PREVIEW=500` and POSTs `content_preview` to
  `POST /sessions/{id}/tool-calls` (`http/toolcalls.go` → `memory.RunHook`). The sandbox
  just stops truncating above the threshold and adds a `content` field carrying the full
  output when gated. The `checkpoint` mode is the turn-end seam the CLM mirror read uses.
- **Second Brain schema.** Second Brain has no table of its own —
  `secondbrain/ask.go:74` calls `session_inject.HybridObservations` over
  `observations` + `observations_fts`. "The same that for Second Brain" therefore means
  the `observations` schema shape (ADR-0026).
- **RRF fusion.** `hybrid/rank.go` defines `const RRFK = 60` and `Rank(...)`. `ctx_search`
  reuses it for the two-leg fusion.
- **Token estimator.** `session_inject.EstimateTokens` (summary.go:18) is the deterministic
  size estimator the CLM overflow guard and calibration build on.
- **Migrations.** Latest is `049_session_checkpoint.sql`. `046`/`047` are taken
  (`046_entity_aliases.sql`, `047_session_agent.sql`). This change adds
  `050_tool_outputs.sql`, `051_indexed_files.sql`, `052_context_revisions.sql`.
- **OpenCode seam.** OpenCode v2 exposes a `context` plugin hook that modifies
  `event.system`/`event.messages`/`event.tools` immediately before model dispatch; "changes
  affect only the outgoing model call, not persisted history." That is the exact seam the
  CLM mirror render + withhold apply needs.
- `.skillgrid/ASSUMPTIONS.md` § Locked constraints and `.skillgrid/ARCHITECTURE.md` §5
  (packages), §7 (hybrid), §8.1 (session injection), §9 (storage), §10 (transport).

## Approaches Considered

- **Chosen:** Absorb `session_inject` into a new `context_harness` package that owns
  retrieval, render, capture, query, routing, the `ctx` CLI, and the CLM layer. Public
  functions keep their signatures; importers change import path only. CLM is a
  Go-owns-state / Node-owns-request-path split. Rationale: the context lifecycle is one
  concern; splitting it across sibling packages forces every capability to re-import the
  retrieval engine and the FTS store.
- **Rejected — sibling package:** `context_harness` next to `session_inject` leaves the
  lifecycle split across two packages that need the same store and token-cost model.
- **Rejected — thin facade:** a `context_harness` facade delegating to an unchanged
  `session_inject` avoids the move but keeps two owners of the same lifecycle, and the
  capture/query/CLI code still lands in the facade, so the split is cosmetic.
- **Rejected — full CLM in Node:** pi-clm parity (Node computes budget, overflow,
  calibration, persists via HTTP) puts deterministic, testable math in the thin request
  path and re-implements the token estimator in JS. Go already owns the estimator and the
  capture path that sees turn boundaries and provider counts.
- **Rejected — no mirror:** only the overflow guard + calibration ("runs without the
  model" half) gives no agent write access, which is the core CLM capability.

## Requirements

1. **Capture gate (intercept-and-abstract):** Large tool output is stored to the
   `tool_outputs` sandbox and the agent receives a summary + pointer; small output
   flows through unchanged.
   - **Current:** `tool-call-capture.js` truncates `tool_output` to 500 chars;
     `http/toolcalls.go` carries only `content_preview`. Nothing is stored for retrieval.
   - **Target:** When actual output > threshold (default ~4KB) and no `SKILLGRID_CTX_BYPASS`,
     the full output is stored to `tool_outputs` and the agent gets a 200-char summary +
     `ctx_search <query>` pointer. Bypass forces the full text through. Output ≤ threshold
     is unchanged.
   - **Acceptance:** A >4KB tool output produces a `tool_outputs` row and a
     summary+pointer to the agent; a <4KB output produces no `tool_outputs` row and the
     original text; `SKILLGRID_CTX_BYPASS=1` on a >4KB output forces the full text.
   - **Acceptance scenario:** `happy path large-output-captured` → `acceptance.feature`

2. **Structured query (`ctx_query`):** The agent can run deterministic counts/lists/
   existence checks over the code index with no JS and no line ranges in v1.
   - **Current:** `code_search`/`code_*` MCP tools exist but are a different contract
     (search, not structured query); no `ctx_query`.
   - **Target:** `ctx_query symbols --path <p> --kind <k> --name <n>` (and counts,
     existence) answers deterministically from the code index.
   - **Acceptance:** A `ctx_query` for a known symbol returns the correct count/list/
     existence without executing JS; an unknown symbol returns an empty/falsy result, not
     an error.
   - **Acceptance scenario:** `happy path ctx-query-known-symbol` → `acceptance.feature`

3. **Proactive index (`ctx index` + `indexed_files`):** The agent or operator can index a
   file/directory into a project-scoped store reusing the observations schema shape.
   - **Current:** No `indexed_files` table; `ctx index` does not exist.
   - **Target:** `ctx index <path>` reads, chunks (~2KB), and upserts rows into
     `indexed_files` (keyed by `source_path` + `chunk_index`, bounded to ~100 chunks per
     call). 7-day TTL like observations.
   - **Acceptance:** `ctx index docs/guide.md` creates `indexed_files` rows for the file's
     chunks; re-running upserts (no duplicate rows); a >100-chunk file is bounded.
   - **Acceptance scenario:** `happy path ctx-index-file` → `acceptance.feature`

4. **Fused search (`ctx_search`):** `ctx_search` fuses the session-scoped sandbox and the
   project-scoped index via RRF.
   - **Current:** No `ctx_search`; `tool_outputs` and `indexed_files` do not exist.
   - **Target:** `ctx_search <query>` queries `tool_outputs_fts` + `indexed_files_fts` and
     fuses with `hybrid.Rank` (RRFK=60); results carry provenance (sandbox vs indexed).
   - **Acceptance:** A query matching both a captured output and an indexed file returns
     both, fused, with per-leg provenance; a query matching only one leg returns that leg.
   - **Acceptance scenario:** `happy path ctx-search-fused` → `acceptance.feature`

5. **`ctx` CLI:** `ctx stats`, `ctx index <path>`, `ctx search <query>`, and `ctx purge`
   work from the command line.
   - **Current:** No `ctx` command group.
   - **Target:** `skillgrid ctx stats` reports sandbox + index counts; `ctx index`/`search`
     as above; `ctx purge` clears `tool_outputs` (session-scoped) and `context_revisions`.
   - **Acceptance:** `ctx stats` returns row counts for `tool_outputs` and `indexed_files`;
     `ctx purge` empties `tool_outputs` and `context_revisions` for the session.
   - **Acceptance scenario:** `happy path ctx-purge-clears-sandbox` → `acceptance.feature`

6. **Context Routing block:** `skillgrid prime` injects a ~80-word tool map.
   - **Current:** `prime` injects the Memory Index and session context; no routing block.
   - **Target:** `prime` output includes a Context Routing block mapping intent to tool:
     counts/lists → `ctx_query`, retrieve from captured output → `ctx_search`, proactive
     index → `ctx index`, memory → `mem_*`. Advisory, not blocking.
   - **Acceptance:** `prime` output contains the four intent→tool mappings and does not
     block the session.
   - **Acceptance scenario:** `happy path prime-routing-block` → `acceptance.feature`

7. **CLM opt-in (default off):** CLM is off by default; enabling it requires config.
   - **Current:** No CLM; no `clm:` config block.
   - **Target:** `clm.enabled` defaults to `false`. When off, no mirror is rendered, no
     overflow guard runs, no budget notes appear, and `context_revisions` stays empty.
   - **Acceptance:** With default config, a session runs with no mirror file and no
     `context_revisions` rows; with `clm.enabled: true`, a mirror is rendered before the
     first request.
   - **Acceptance scenario:** `happy path clm-off-by-default` → `acceptance.feature`

8. **CLM mirror render + model edit:** When CLM is on, a `0600` mirror is rendered before
   each request and the agent can edit it with ordinary file tools.
   - **Current:** No mirror.
   - **Target:** The Node plugin renders the effective context to a `0600` temp file
     (`[[LIVE_CONTEXT ...]]` header + `[[CTX_TURN ...]]` blocks, nonce-bound ids) via the
     OpenCode `context` hook; the agent edits it during its turn.
   - **Acceptance:** With CLM on, a mirror file exists and is mode 0600 before the first
     request; a model edit to the file is observable at turn-end.
   - **Acceptance scenario:** `happy path clm-mirror-rendered` → `acceptance.feature`

9. **Revision capture + validation:** At turn-end, the model's mirror edit is validated
   and persisted as a `context_revisions` row, then activated for the next request.
   - **Current:** No `context_revisions`.
   - **Target:** The `checkpoint` capture path reads the mirror, compares mtime/hash to
     the last render (skip when unchanged), and POSTs an edit to Go; Go validates
     (nonce match, legal message sequence, tool-call-group repair), persists a
     `context_revisions` row, and activates it. The next request uses the revision.
   - **Acceptance:** A changed mirror at turn-end produces a `context_revisions` row whose
     anchor (count + digest) matches the raw prefix and is active on the next request; an
     unchanged mirror produces no new row; a nonce mismatch discards the edit.
   - **Acceptance scenario:** `happy path clm-revision-captured` → `acceptance.feature`

10. **Overflow guard:** When the estimated request exceeds `budget − reserve`, the oldest
    tool results are withheld.
    - **Current:** No budget enforcement.
    - **Target:** Go computes the withhold decision (which tool-result ids to swap for
      one-line notes) when the calibrated estimate exceeds `budget − reserve`; the decision
      is stored on the revision and applied by the Node plugin at render.
    - **Acceptance:** With a request over budget, the oldest tool results are swapped for
      one-line notes in the rendered context; a request under budget withholds nothing.
    - **Acceptance scenario:** `happy path clm-overflow-withholds` → `acceptance.feature`

11. **Calibration:** The size estimate is corrected against the provider's token count.
    - **Current:** `EstimateTokens` is a fixed ~4 chars/token.
    - **Target:** Each request's provider token count corrects the per-session estimate
      factor, stored on the revision, so dense content does not slip past the budget.
    - **Acceptance:** After a request whose provider count differs from the estimate, the
      stored calibration factor reflects the correction and the next estimate uses it.
    - **Acceptance scenario:** `happy path clm-calibration-corrects` → `acceptance.feature`

12. **Session-scoped purge + resume:** `context_revisions` and `tool_outputs` are purged
    at session end; on resume the active revision is reconstructed and validated.
    - **Current:** Neither table exists.
    - **Target:** `ctx purge` and the session-end hook clear `tool_outputs` +
     `context_revisions`; on resume the plugin reconstructs the highest `revision` row and
     validates anchor_count + anchor_digest against the raw prefix — a mismatch discards
     the revision and falls back to raw context.
    - **Acceptance:** `ctx purge` empties both tables; a resumed session with a valid anchor
     restores the revision; a resumed session with a broken anchor (raw history compacted)
     falls back to raw context.
    - **Acceptance scenario:** `happy path clm-resume-validates-anchor` → `acceptance.feature`

13. **CLM config:** All CLM settings read from the `clm:` block with env overrides.
    - **Current:** No `clm:` block.
    - **Target:** `clm.enabled`, `clm.budget`, `clm.reserve`, `clm.reminders`, `clm.guard`,
      `clm.cap` read from `.skillgrid/config.yaml`; env overrides
      (`SKILLGRID_CTX_CLM`, `SKILLGRID_CTX_CLM_BUDGET`, `SKILLGRID_CTX_CLM_RESERVE`) win.
    - **Acceptance:** A `clm:` block is honored; an env override for `budget` wins over the
      config value; an absent `clm:` block yields all defaults (off).
    - **Acceptance scenario:** `happy path clm-config-honored` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:**
  - NEW `internal/mnemonic/context_harness/` — absorbs `session_inject` (retrieval,
    render, autoprepend, summary, privacy, index) and adds `capture.go` (sandbox gate),
    `query.go` (`ctx_query`), `search.go` (`ctx_search` two-leg RRF), `routing.go`
    (Context Routing block), `clm/` sub-package (`mirror.go`, `overflow.go`,
    `calibrate.go`, `validate.go`, `revision.go`).
  - NEW `internal/mnemonic/store/migrations/050_tool_outputs.sql`,
    `051_indexed_files.sql`, `052_context_revisions.sql`.
  - CHANGED `hooks/tool-call-capture.js` — stop truncating above the threshold (add a
    `content` field for gated output); extend `checkpoint` mode to read the CLM mirror at
    turn-end and POST the edit.
  - CHANGED `http/toolcalls.go` — accept `content` (full output) on the capture route;
    route gated output to the sandbox.
  - CHANGED `cmd/skillgrid/` — add `ctx` command group (`stats/index/search/purge`);
    `prime` gains the Context Routing block; `loop_cmd.go` import path
    `session_inject` → `context_harness`.
  - CHANGED importers — `secondbrain/ask.go`, `mcp/tools_session_inject.go`,
    `http/toolevents.go` change import path only (signatures unchanged).
  - NEW OpenCode plugin module (`.opencode/plugins/` or staged Hub Content) — the `context`
    hook that renders the mirror, applies the withhold decision, and reads the edit.
  - CHANGED `config` — parse the `clm:` block with defaults + env overrides.

- **Interfaces:**
  - `context_harness` public surface keeps `session_inject` signatures:
    `AutoPrepend(...)`, `HybridRetrieve(...)`, `RenderContextBlock(...)`,
    `EstimateTokens(string) int`, `ObservationInjectable(...)`.
  - `ctx_query` request shape: `{ query: "symbols"|"counts"|"exists", path, kind, name, ... }`
    → `{ count, items[], exists }` (no line ranges in v1).
  - `ctx_search` returns fused hits with `{ source: "sandbox"|"indexed", score, snippet }`.
  - Mirror format: `[[LIVE_CONTEXT version=1 revision=N document=<nonce> baseline=<digest>]]`
    header then `[[CTX_TURN document=<nonce> index=I role=R id=<id> protected=false]]`
    blocks. Nonce = f(session, anchor digest, revision) so ids are stable across accepted
    edits.
  - `context_revisions` row: `id, session_id, project_id, revision, anchor_count,
    anchor_digest, messages (JSON), size_estimate, calibration_factor, withhold_ids (JSON),
    edit_trace (JSON), created_at`.
  - Capture edit POST: `{ session_id, document (nonce), mirror_text, mtime, hash }` → Go
    validates + persists, returns `{ revision, active }`.

- **Data flow:**
  - Capture: harness tool → `tool-call-capture.js` (gate on actual size) →
    `POST /sessions/{id}/tool-calls` (now with `content`) → `context_harness` writes
    `tool_outputs` row + returns summary+pointer.
  - Query: agent → `ctx_query`/`ctx_search`/`ctx` CLI → `context_harness` → code index /
    FTS store → deterministic result.
  - CLM request path: OpenCode `context` hook → Node plugin renders mirror (0600) from the
    active revision + raw history, applies Go's withhold decision → outgoing model call.
  - CLM turn-end: `checkpoint` mode reads mirror (mtime/hash gate) → POST edit → Go
    validates + persists `context_revisions` + activates → next request uses it.
  - Provider token count from the capture corrects the calibration factor per session.

- **Error handling:** Fail-open floors hold. A capture-path error never blocks the tool
  call (fire-and-forget, exit 0). A mirror-read or edit-validation failure discards the
  edit and falls back to raw context (the revision is not activated). An overflow-guard
  or calibration error leaves the estimate at the prior factor and withholds nothing.
  `ctx purge` on a missing table is a no-op. CLM off = zero behavior change.

- **Dependencies:** `context_harness` depends on `store`, `codeindex` (for `ctx_query`),
  `hybrid` (RRF), and `config` only. The Node plugin depends on the HTTP API + the temp
  mirror path. No new Go dependencies; no new Node dependencies beyond what
  `tool-call-capture.js` already uses (`fetch`, `fs`, `os`, `path`).

- **Prototype snippets:** None (the design is pinned by ADR-0025…0028 + the verified
  capture seam; no throwaway probe produced a load-bearing snippet).

## Testing Decisions

- **What makes a good test:** Assert external behavior — the `tool_outputs` row, the
  summary+pointer, the fused result, the `context_revisions` row, the withhold decision —
  not internal helpers. The CLM budget math, overflow-withhold, mirror validation, and
  calibration are pure functions over a message list: unit-test them directly without a
  Node plugin or an OpenCode session.
- **Modules to test:**
  - `context_harness/capture_test.go` — gate threshold, bypass, small-output passthrough.
  - `context_harness/query_test.go` — counts/list/existence against a seeded code index.
  - `context_harness/search_test.go` — two-leg RRF fusion, provenance, single-leg.
  - `context_harness/routing_test.go` — the four intent→tool mappings present.
  - `context_harness/clm/` — `mirror` render (format + 0600), `overflow` withhold
    decision, `calibrate` correction, `validate` nonce/sequence, `revision` persistence +
    anchor check + resume reconstruction.
  - `http` — the capture route accepts `content` and stores gated output.
  - `config` — `clm:` block parsing + env overrides + defaults.
- **Prior art:** `session_inject/*_test.go` (retrieval/render), `hybrid/rank.go` tests
  (RRF), `memory` integration tests (seeding a store then asserting over HTTP), and the
  `internal/mnemonic/integration/` seed tests.
- **Edge cases:** Output exactly at the threshold; `SKILLGRID_CTX_BYPASS` on a >4KB output;
  a >100-chunk `ctx index`; a query matching both legs vs one leg; an unchanged mirror at
  turn-end (skip POST); a nonce mismatch; an anchor mismatch on resume (raw history
  compacted); CLM off with all defaults; an env override beating the config value.

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None (scope/metric unchanged; this is a subsystem).
- `.skillgrid/ASSUMPTIONS.md`: In-force set already carries ADR-0025…0028 (added during
  the interview). No further edit.
- `.skillgrid/ARCHITECTURE.md`: Update §5 (packages: add `context_harness`, note
  `session_inject` absorbed), §8.1 (rename to Context Harness; add capture/query/routing/
  CLI/CLM rows), §9 (migration table: add `050`/`051`/`052`), §10 (transport: capture route
  carries `content`; new OpenCode `context` plugin hook), §14 (config: `clm:` block), §16
  (hooks: `tool-call-capture.js` gain + checkpoint CLM read).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | PASS   | Context lifecycle + CLM, one owner |
| Boundary Clarity    | 0.90  | 0.70 | PASS   | Absorb `session_inject`; Go/Node split |
| Constraint Clarity  | 0.85  | 0.65 | PASS   | Fail-open floors; `mem_inject_session` unchanged; Go 1.22+ |
| Acceptance Criteria | 0.85  | 0.70 | PASS   | 13 falsifiable requirements + gates |
| **Clarity**         | **0.13** | ≤0.20 | PASS | 1 − mean(0.90,0.90,0.85,0.85) |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | Where do the new context capabilities live? | Absorb `session_inject` into `context_harness`; public functions keep signatures; `mem_inject_session` unchanged (ADR-0025). |
| 2     | Which store for `ctx index`? | Separate `indexed_files` reusing the observations schema shape (Second Brain has no table of its own); `ctx_search` fuses via RRF (ADR-0026). |
| 3     | CLM scope + split? | v1 = mirror + overflow guard + calibration; Go owns state/decisions, Node owns the request path (ADR-0027). |
| 4     | `context_revisions` lifecycle? | Session-scoped audit, purged at session end; raw history + mirror are the source of truth (ADR-0028). |
| 5     | Config + opt-in? | CLM off by default; all config in `clm:` block with env overrides. |
| 6     | Capture gate + `ctx_query` scope? | Gate on actual output size in the PostToolUse seam; `ctx_query` = counts/lists/existence, no line ranges in v1. |

## Open Questions & Assumptions

- **Assumption:** The OpenCode `context` hook is available in the target OpenCode version
  (v2 plugins). If the harness in use predates it, the CLM mirror render degrades to a
  no-op (CLM off-equivalent) and the overflow guard still runs from the capture path.
  This is a known seam dependency, not a blocker.
- **Assumption:** The provider token count is available in the capture path for
  calibration. If a harness does not expose it, calibration stays at the prior factor
  (fail-open) and the overflow guard uses the uncorrected estimate.
- **Assumption:** The mirror is a prompt-injection surface (injected text can induce the
  model to rewrite its own constraints). The system prompt stays out of the mirror and
  authored roles are lowered to plain text. Known limitation per ADR-0027, not a fix.

## Decisions (ADR)

- `.skillgrid/artifacts/04-adr-0025-context-harness-owner.md` — Context Harness absorbs
  `session_inject`; four capabilities; capture gates on actual output.
- `.skillgrid/artifacts/04-adr-0026-indexed-files-reuses-observations-schema.md` —
  `indexed_files` is a separate project-scoped table reusing the observations schema shape;
  `ctx_search` fuses both legs with RRF.
- `.skillgrid/artifacts/04-adr-0027-clm-context-language-model.md` — CLM layer: Go owns
  state/decisions (budget, overflow, calibration, revision), Node owns the request path
  (render mirror, apply withhold, read edit); opt-in via `clm:` config.
- `.skillgrid/artifacts/04-adr-0028-context-revisions-session-scoped.md` —
  `context_revisions` is session-scoped audit (migration 052), purged at session end.

## Terms

- Context Harness — `.skillgrid/artifacts/02-technical-terms.md`
- Intercept-and-Abstract — same file
- Sandbox Store — same file (migration 050)
- Output Sandbox Gate — same file
- ctx_query — same file
- indexed_files — same file (migration 051)
- Context Routing — same file
- ctx CLI — same file
- Context Language Model (CLM) — same file
- Context Mirror — same file
- Context Revision — same file
- Overflow Guard — same file
- Context Calibration — same file
- CLM Config — same file
