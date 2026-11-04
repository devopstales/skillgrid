# Mnemonic-v2 Memory Plugins — Implementation Blueprint (steps, files, verification)

> Copy this template to `.skillgrid/specs/2026-11-04-mnemonic-v2-memory-plugins/blueprint.md`.
> **Role:** this is the *how* — the implementation plan: concrete steps, files to touch, code, tests, and verification. The companion `briefing.md` is the *what* (requirements & intent). The filename pair is load-bearing: `brainstorming` and `resume` key phase state off these exact filenames — never rename.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread

**Goal:** Port the engram agent-memory plugin architecture onto the mnemonic/mnemonic-v2 memory function for OpenCode and Pi, delivering full agent-facing memory tool-surface parity, engram-proven behavior, and a save-event handoff to the brain-side wiki.

**Architecture:** Extend the existing `plugins/pi/mnemonic/` and `plugins/opencode/` packages; client plugins call the mnemonic/mnemonic-v2 HTTP server. The engram baseline supplies the proven patterns (runtime contract, redaction, retries, self-heal, event wiring, compaction recovery). The memory save pipeline persists, then emits the SaveEvent onto an in-process event bus; the brain-side wiki consumes it (server-side seam — the plugin is not in the loop).

**Tech Stack:** Go (mnemonic/mnemonic-v2 server), TypeScript (Pi + OpenCode plugins), MCP, SQLite/Postgres store.

**Spec:** `.skillgrid/specs/2026-11-04-mnemonic-v2-memory-plugins/briefing.md` (product *why* lives in the spec and `.skillgrid/knowledge/reference/00-prd.md`; decisions live in `decisions/04-adr-NNNN-slug.md`).

## Terms

None — no new domain terms introduced by this plan beyond the spec's.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- The OpenCode + Pi memory plugins expose the full engram-baseline agent-facing memory tool surface backed by mnemonic/mnemonic-v2.
- The ported plugin behaviors preserve engram reliability semantics (redaction, retries, self-heal, fail-open for capture/lifecycle, fail-closed for writes, compaction recovery, passive capture, session wiring).
- The memory save pipeline emits a SaveEvent consumed by the brain-side wiki.

**Artifacts** (files that must exist with real implementation, not stubs):
- `plugins/pi/mnemonic/api.js` — HTTP/redaction/TypeBox helpers.
- `plugins/pi/mnemonic/index.ts` — tool registration + event wiring.
- `plugins/pi/mnemonic/mnemonic-memory.ts` — mem_* tools.
- `plugins/pi/mnemonic/mnemonic-facts.ts` — fact_* tools.
- `plugins/pi/mnemonic/mnemonic-codeindex.ts` — code_* tools.
- `plugins/pi/mnemonic/ctx-sandbox.ts` — ctx_* sandbox.
- `plugins/pi/mnemonic/skillgrid-events.ts`, `skillgrid-sdd-events.ts`, `skillgrid-compaction.ts` — event/compaction wiring.
- `plugins/opencode/mnemonic-memory.ts`, `plugins/opencode/mnemonic-codeindex.ts`, `plugins/opencode/skillgrid-compaction.ts`, `plugins/opencode/skillgrid-context.ts`, `plugins/opencode/skillgrid-events.ts`, `plugins/opencode/skillgrid-squad.ts`, `plugins/opencode/skillgrid-v2sse.ts`, `plugins/opencode/tools/*` — OpenCode plugin wiring.
- `.skillgrid/specs/2026-11-04-mnemonic-v2-memory-plugins/acceptance.feature`.

**Key links:** plugin tool call → redact → validate → dispatch → persist → SaveEvent → wiki ingestion.

**One-way-door decisions:** None identified.

## Global Constraints

- Go 1.22+ minimum to build (ADR-0031).
- No new dependencies without an ADR.
- TDD baseline: basic TDD (failing test before code) is the baseline; strict-TDD machinery off by default.
- Test: `go test ./...`; Build: `go build ./...`.

## File Structure

- `plugins/pi/mnemonic/*` — Pi memory plugin modules.
- `plugins/opencode/*` — OpenCode memory plugin modules + tools.
- `mnemonic/internal/http/plugin_route_contract_test.go` — plugin↔mux route contract owner.

## Tool-surface parity mapping

Critic F1 resolution (user decision): add full-parity tasks + mapping table.

| Engram baseline tool | Mnemonic-v2 registered tool | Notes |
|---|---|---|
| mem_search | mem_search | same |
| mem_search_prompts | mem_search_prompts | same |
| mem_save | mem_save | same |
| mem_update | mem_update | same |
| mem_consolidate | mem_consolidate | same |
| mem_suggest_topic_key | mem_suggest_topic_key | same |
| mem_save_prompt | mem_save_prompt | same |
| mem_session_summary | mem_session_summary | same |
| mem_session_finalize | mem_session_finalize | same |
| mem_context | mem_context | same |
| mem_get_observation | mem_get_observation | same |
| mem_session_start | mem_session_start | same |
| mem_session_end | mem_session_end | same |
| mem_current_project | mem_current_project | same |
| mem_doctor | mem_doctor | same |
| mem_capture_passive | mem_capture_passive | same |
| mem_review | mem_review | same |
| mem_judge | mem_judge | same |
| mem_judge_many | mem_judge_many | same |
| mem_compare | mem_compare | same |
| mem_list_projects | mem_list_projects | same |
| mem_pin | mem_pin | same |
| mem_unpin | mem_unpin | same |
| — (engram had no lookup) | mem_lookup | mnemonic-v2 delta: retrieval ladder |

### Task 1: Port engram memory tool-surface parity into the Pi plugin (full agent-facing set)

**Files:**
- Create: `plugins/pi/mnemonic/mnemonic-memory.ts` (extend), `mnemonic-facts.ts`, `mnemonic-codeindex.ts`, `ctx-sandbox.ts`, `api.js`, `index.ts` (extend).
- Test: `plugins/pi/mnemonic/*` parity + save/lookup tests.

**Interfaces:**
- Consumes: mnemonic/mnemonic-v2 HTTP API (`/observations`, `/memory/observations/{id}`, `/search`).
- Produces: registered `mem_*` tools per the mapping table.
- Seam: `plugins/pi/mnemonic/api.js`.
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `happy path — agent saves a new memory through the plugin tool`

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

### Task 2: Port engram reliability behavior into the OpenCode plugin (fail-open split, retry, self-heal)

Critic F2 resolution (user decision): split capture-fails-open vs write-fails-closed.

**Files:**
- Create: `plugins/opencode/mnemonic-memory.ts` (extend), `plugins/opencode/tools/mem_*.ts` (extend), `plugins/opencode/skillgrid-compaction.ts` (extend).

**Interfaces:**
- Consumes: mnemonic/mnemonic-v2 HTTP API.
- Produces: capture/lifecycle fail-open; write fail-closed (single attempt, ambiguous-outcome recovery, no blind retry).
- Seam: plugin tool handler.
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `failure state — memory server unreachable through the plugin`

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

### Task 3: Port engram behavior portability into the Pi plugin (compaction recovery, passive capture)

**Files:**
- Create: `plugins/pi/mnemonic/skillgrid-compaction.ts` (extend), `plugins/pi/mnemonic/skillgrid-events.ts` (extend).

**Interfaces:**
- Consumes: mnemonic/mnemonic-v2 session/compaction API.
- Produces: compaction recovery + passive capture redaction.
- Seam: plugin event wiring.
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `happy path — compaction recovery persists prior work`

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

### Task 4: Wire server-side SaveEvent → brain-side wiki consumer (rewritten per F3)

Critic F3 resolution (user decision): rewrite Task 4 around the server-side event-bus seam.

**Files:**
- Create: `plugins/pi/mnemonic/skillgrid-events.ts` (extend) / server-side save-event publication + wiki consumer wiring.

**Interfaces:**
- Consumes: memory save pipeline SaveEvent contract (server-side bus).
- Produces: wiki ingestion entry (brain-side `wiki_observe` / capture).
- Seam: server-side event bus.
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `happy path — save emits a SaveEvent for the wiki`

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

## Blueprint validation (after writing, before proceeding)

1. **Critic subagent (mandatory — Tier T2).** Dispatch a dedicated critic subagent with the blueprint content; structured critique (issues + severity).
2. **User review (always mandatory).** Present the blueprint + critic findings to the user; for each issue, present options; update blueprint per decision; re-present final blueprint; user must approve the final blueprint before `acceptance.feature` is written.

Never proceed to the next artifact without the user approving the final blueprint.
