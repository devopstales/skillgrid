# Mnemonic-v2 Agent Plugins (memory / code-index / knowledge) — Implementation Blueprint (steps, files, verification)

> Copy this template to `.skillgrid/change-specs/2026-11-04-mnemonic-v2-memory-plugins/blueprint.md`.
> **Role:** this is the *how* — the implementation plan: concrete steps, files to touch, code, tests, and verification. The companion `briefing.md` is the *what* (requirements & intent). The filename pair is load-bearing: `brainstorming` and `resume` key phase state off these exact filenames — never rename.

**Status:** APPROVED (2026-11-05, user approval "look good"; amended 2026-11-05: 3-family split per user direction; amended 2026-11-05: pi-research fold R6 — all 7 folds into the memory + code-index plugins on both hosts, excluded from the knowledge-base plugin per user direction)

**Tier:** T2

**Build shape:** Tracer thread

**Goal:** Port the engram agent-memory plugin architecture onto the mnemonic-v2 engine and extend it into **three plugin families per host** (memory / code-index / knowledge-base) for OpenCode and Pi, delivering full tool-surface parity per family (v2 registered sets: 28 `mem_*`, 25 `code_*`, 12 `kb_*`), engram-proven behavior on the memory plugin, and the save-event handoff to the brain-side wiki. The knowledge-base family is gated on the KB function (`2026-10-31-mnemonic-v2-knowledge-base`) shipping its `kb_*` tools.

**Architecture:** Extend the existing `plugins/pi/` and `plugins/opencode/` packages into three families per host. Pi: `@devopstales/pi-mnemonic` (memory family — existing package, codeindex module extracted out), `@devopstales/pi-codeindex` (new), `@devopstales/pi-knowledge` (new, gated). OpenCode: three plugin entries in `@skillgrid/opencode-plugins` (memory / code-index / knowledge). Client plugins consume the v2 engine (`mnemonic-v2/`) through its MCP stdio surface — a spawned `mnemonic mcp` child process per the engram runtime contract pattern (v2 has no HTTP server: `mnemonic serve` is a deprecated no-op; architecture §1: plugins "consume the engine only through the MCP surface"). The existing v1-HTTP wiring (`SKILLGRID_MNEMONIC_HTTP_URL`, :7438) and `fact_*` tools are v1 legacy — retained, not extended. The engram baseline supplies the proven patterns (runtime contract, redaction, retries, self-heal, event wiring, compaction recovery). The memory save pipeline persists, then emits the SaveEvent onto an in-process event bus; the brain-side wiki and the KB function consume it (server-side seam — the plugin is not in the loop).

**Tech Stack:** Go (mnemonic-v2 engine), TypeScript (Pi + OpenCode plugins), MCP stdio, SQLite/Postgres store.

**Spec:** `.skillgrid/change-specs/2026-11-04-mnemonic-v2-memory-plugins/briefing.md` (product *why* lives in the spec and `.skillgrid/knowledge/reference/00-prd.md`; decisions live in `decisions/04-adr-NNNN-slug.md`).

## Terms

None — no new domain terms introduced by this plan beyond the spec's.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- The OpenCode + Pi **memory plugins** expose the full registered v2 `mem_*` surface (28 tools: 22 agent-profile + 6 admin, per `mnemonic-v2/internal/transport/mcp/contract_test.go`; engram baseline as behavior floor) backed by the mnemonic-v2 engine.
- The OpenCode + Pi **code-index plugins** expose the full v2 `code_*` surface (25 tools).
- The OpenCode + Pi **knowledge-base plugins** expose the `kb_*` surface (12 tools per KB spec 008), gated on the KB function's ship.
- The ported memory plugin behaviors preserve engram reliability semantics (redaction, retries, self-heal, fail-open for capture/lifecycle, fail-closed for writes, compaction recovery, passive capture, session wiring); all families share the fail-open seam.
- The memory save pipeline emits a SaveEvent consumed by the brain-side wiki.
- The memory + code-index plugins (both hosts) hold the pi-research fold (R6): capability-gated tool surface (admin `mem_*` hidden by default; `code_*` gated on the server capability probe — no restart), MCP-child lifecycle (bounded drain + force-exit watchdog, shutdown-reason branching), compaction recovery on the named compaction events, described-OFF embedder degradation (one sentence across surfaces, disabled-not-unhealthy), subagent read guard, consumer queue discipline, and the skill + CLI distribution with the exit-code protocol. The knowledge-base plugin is excluded from this fold (user direction).

**Artifacts** (files that must exist with real implementation, not stubs):
- `plugins/pi/mnemonic/api.js` — shared MCP-stdio client / redaction / TypeBox helpers (memory family; reused shape for the other Pi packages).
- `plugins/pi/mnemonic/index.ts` — memory tool registration + event wiring.
- `plugins/pi/mnemonic/mnemonic-memory.ts` — mem_* tools (full v2 set).
- `plugins/pi/mnemonic/mnemonic-facts.ts` — fact_* tools (v1-HTTP legacy, retained, out of parity).
- `plugins/pi/mnemonic/ctx-sandbox.ts` — ctx_* sandbox.
- `plugins/pi/mnemonic/skillgrid-events.ts`, `skillgrid-sdd-events.ts`, `skillgrid-compaction.ts` — event/compaction wiring.
- `plugins/pi/codeindex/package.json`, `index.ts`, `codeindex-tools.ts` — new Pi package, full v2 `code_*` set.
- `plugins/pi/knowledge/package.json`, `index.ts`, `knowledge-tools.ts` — new Pi package, `kb_*` set (gated).
- `plugins/opencode/mnemonic-memory.ts` — memory entry (mem_* tools).
- `plugins/opencode/mnemonic-codeindex.ts` — code-index entry (full v2 `code_*` set).
- `plugins/opencode/mnemonic-knowledge.ts` — knowledge entry (`kb_*` set; gated).
- `plugins/opencode/tools/mem_*.ts`, `code_*.ts`, `kb_*.ts` — OpenCode tool definitions (`tools/fact_*.ts` v1-legacy, retained).
- `plugins/opencode/test/*` — parity + fail-open tests (node --experimental-strip-types; the package `test` script runs all test files).
- `.skillgrid/change-specs/2026-11-04-mnemonic-v2-memory-plugins/acceptance.feature`.

**Key links:** plugin tool call → redact → validate → dispatch → persist → SaveEvent → wiki/KB ingestion.

**One-way-door decisions:** None identified.

## Global Constraints

- Go 1.22+ minimum to build (ADR-0031).
- No new dependencies without an ADR.
- TDD baseline: basic TDD (failing test before code) is the baseline; strict-TDD machinery off by default.
- Test: `go test ./...` (v2 contract + gates via `go test ./mnemonic-v2/... -count=1`); Build: `go build ./...`; plugin tests: `node --experimental-strip-types` — the OpenCode package `test` script is updated to run ALL test files (currently single-file); the Pi packages get a `test` script on the same runner (defined in Tasks 1/5/6).
- Serial development: waves run in order; the knowledge family (wave 3) additionally requires the KB function shipped.

## File Structure

- `plugins/pi/mnemonic/*` — Pi memory plugin modules.
- `plugins/pi/codeindex/*` — Pi code-index plugin (new package).
- `plugins/pi/knowledge/*` — Pi knowledge-base plugin (new package, gated).
- `plugins/opencode/*` — OpenCode plugin entries + tools (3 families).
- `mnemonic-v2/internal/transport/mcp/contract_test.go` — D1-pinned v2 MCP tool contract (mem_* family; code/kb family contracts per their changes). The v1 `mnemonic/internal/http/plugin_route_contract_test.go` is legacy (v1 HTTP era) — not the parity owner.

## Tool-surface parity mapping

Critic F1 resolution (user decision): add full-parity tasks + mapping table.

### Memory family (`mem_*`) — engram baseline floor, full v2 registered set target

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
| mem_doctor | mem_doctor | same (admin profile) |
| mem_capture_passive | mem_capture_passive | same |
| mem_review | mem_review | same (admin profile) |
| mem_judge | mem_judge | same |
| mem_judge_many | mem_judge_many | same |
| mem_compare | mem_compare | same |
| mem_list_projects | mem_list_projects | same |
| mem_pin | mem_pin | same |
| mem_unpin | mem_unpin | same |
| — (engram had no lookup) | mem_lookup | mnemonic-v2 delta: retrieval ladder |

Full target = the v2 registered `mem_*` set: **28 tools** (22 agent-profile +
6 admin: mem_delete, mem_doctor, mem_merge_projects, mem_review, mem_stats,
mem_timeline — per `mnemonic-v2/internal/transport/mcp/contract_test.go`,
the D1-pinned authority). The parity test reads the v2 registration as the
source of truth — the table above is the engram anchor; all 24 anchor rows
(23 engram + `mem_lookup`) exist in the v2 set. The v1 `fact_*` set (8 tools:
fact_add, fact_search, fact_forget, fact_decay, fact_decay_all,
fact_pending_list, fact_pending_approve, fact_pending_reject) is v1-HTTP
legacy — retained in the plugins, out of the parity target (in v2, durable
facts are memory observations).

### Code-index family (`code_*`) — full v2 registered set (25 tools)

code_affected, code_circles, code_configs, code_dep_impact, code_deps,
code_docs, code_embedding_status, code_explain, code_grep, code_impact,
code_index, code_navigates, code_orient, code_packages, code_path, code_read,
code_rename, code_resolve, code_route, code_search, code_semantic,
code_sql_schema, code_status, code_unresolved, code_watch.

Parity test asserts plugin-registered set == v2-registered `code_*` set (both
hosts). Staleness workflow preserved: code_status → (stale) code_index →
code_search → code_read.

### Knowledge family (`kb_*`) — per KB spec `specs/008-transport.md` (gated)

| Tool | Profile | Purpose |
|---|---|---|
| kb_query | agent | Stage-1 deep retrieval (wiki-first) |
| kb_read | agent | Stage-2 full page / section |
| kb_search | agent | Raw FTS over kb_pages_fts |
| kb_generate | agent | Run generation (async: run_id + poll; 409 double-start) |
| kb_validate | agent | Pin-check against live code |
| kb_lint | agent | Deterministic health (+ fix) |
| kb_status | agent | Vault health (read-only derived state) |
| kb_cite_source | agent | Record external citation (write queue) |
| kb_reindex | admin | Full FTS rebuild |
| kb_index | admin | Incremental reindex |
| kb_bootstrap | admin | Scaffold vault (idempotent) |
| kb_rebuild_meta | admin | Rebuild checkpoint + backlink cache |

Plugins consume the `kb_*` tools through the same MCP stdio server as the
other families (KB spec 008 §1); reads synchronous, writes async (`run_id` +
status poll, `409` on double-start); degradation envelope (`degraded: true` +
reason) surfaced, never silently floored.

## pi-research fold (R6, 2026-11-05)

Source: `github.com/Lincoln504/pi-research` (Pi extension + knowledge store,
v1.7.24). User direction: all seven folds land on the **separate OpenCode + Pi
plugins** (memory + code-index families); the knowledge-base plugin is
**not** modified by this fold. Task assignments:

- **PR-1 Capability-gated tool surface** → Tasks 1, 2, 5. Pi: register the
  family's tools unconditionally, re-register with `exposure: 'hidden'` when
  the capability is absent (`pi` has no unregister API; re-registration is
  the documented withdrawal, and a later re-register back to `direct`
  re-activates like a new tool). Gate on the LIVE seam capability probe, not
  on startup registration — a capability change takes effect next
  `session_start` without a restart. OpenCode: no exposure API →
  registration-time gate + self-guarding "unavailable" response envelope.
  Admin `mem_*` (6): hidden by default, exposed via the `all` profile.
  `code_*`: hidden/self-guarding until the server reports the code-index
  family registered.
- **PR-2 Lifecycle discipline** → Tasks 1, 2, 3. MCP stdio child on
  `session_shutdown`: branch on `event.reason` — `quit` → bounded drain
  (≤5s) + force-exit watchdog (`forceExitAfter` ~30s; teardown of ONNX/store
  can hang); reload/resume/fork → drain without exit. Timers `unref()`'d so a
  short host run isn't held open. Compaction recovery: hook the named
  `session_before_compact` / `session_compact` events (not only
  before_agent_start inference).
- **PR-3 Described-OFF degradation** → Task 5 (plugin-visible half). When the
  semantic leg is unavailable (embedder package missing / model failed),
  `code_search` / `code_semantic` / `code_status` responses AND the health
  surface carry ONE identical sentence naming the missing dependency + the
  repair command; the run still serves FTS-leg results; the condition is
  reported as *disabled, not unhealthy* so it never blocks a run.
  Availability probed by resolution, memoized per process, fail-fast (no
  retry storm).
- **PR-4 Consumer queue discipline** → Task 4. Per-observation in-flight
  lock map (TOCTOU on the same fact), bounded drain with a shared teardown
  budget, drop-count surfaced (metrics/response), no unbounded queue growth.
- **PR-5 Subagent read guard** → Task 5. `code_read` from subagent/sandbox
  contexts: confine to the project root (realpath-resolved, symlink-aware) +
  secret-file denylist (`.env*`, `id_*`, `*.pem`, `*.tfstate`, shell
  history, credentials files); denial returns `read denied: <reason>` — a
  prompt injection on read content cannot exfiltrate a credential.
- **PR-6 Skill + CLI dual distribution** → Task 1 (packaging; post-ship
  step, spec-locked now). `agent-skill/` in the pi-mnemonic package:
  SKILL.md with "Use for X / Not for Y" description discipline,
  per-command timeouts with queue headroom, single-background-mechanism
  rule; zero-dep `run.mjs` launcher with ordered engine resolution and exit
  codes (78 = engine/config/model missing — prints the repair + config file
  locations); manifest-tracked installer (install / uninstall / status;
  symlink into detected agents; never clobber a foreign skill; uninstall
  removes exactly what it created; stale links GC'd). Host-agnostic: the
  OpenCode plugin (Task 2) does not duplicate it.
- **PR-7 Prompt-cache steering (forward-looking, no task)** → note for future
  squad/research features: steering goes into the *user* message (keeps the
  system-prompt cacheable prefix byte-identical across parallel subagent
  sessions); "CRITICAL: NO SUBAGENTS" prompt discipline when a tool
  orchestrates internally.

### Cross-change folds (owned elsewhere — not tickets of this change)

- **Code-index change at ship** (`2026-10-31-mnemonic-v2-code-index`,
  fix-loop-green): PR-3 server-side half — NaN/Inf vector gate before vec0
  insert (a correct-width NaN vector breaks ANN ordering and passes length
  checks; guard shared by the normal path and any re-embed migration);
  embedder model change → `backup` (default) / `drop` / `re-embed`
  migration with re-embed falling back to backup on failure — data is never
  dropped without explicit `drop`.
- **KB change** (`2026-10-31-mnemonic-v2-knowledge-base`, advisory):
  scope-as-column (one physical store, scope columns filtered at query time,
  not separate folders), TTL eviction at open within the current scope,
  per-model chunk config (chunk size / pooling / prefixes defined by the
  model, schema built around it).
- **Squad/research (future)**: PR-7.

## Execution waves

- **Wave 1 — memory family (both hosts):** Tasks 1–4. Tracer: engram port.
- **Wave 2 — code-index family (both hosts):** Task 5. Precondition:
  `2026-10-31-mnemonic-v2-code-index` shipped.
- **Wave 3 — knowledge family (both hosts):** Task 6. Precondition:
  `2026-10-31-mnemonic-v2-knowledge-base` shipped (`kb_*` routes live).

### Task 1: Port engram memory tool-surface parity into the Pi plugin (full agent-facing set)

**Files:**
- Create: `plugins/pi/mnemonic/mnemonic-memory.ts` (extend), `mnemonic-facts.ts`, `ctx-sandbox.ts`, `api.js`, `index.ts` (extend).
- Test: `plugins/pi/mnemonic/*` parity + save/lookup tests.

**Interfaces:**
- Consumes: the v2 MCP stdio server (`mnemonic mcp`): `mem_*` tools (agent profile default; admin via `all`).
- Produces: registered `mem_*` tools — the full v2 set (28) per the mapping table.
- Seam: `plugins/pi/mnemonic/api.js` (MCP stdio client + redaction).
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `happy path — agent saves a new memory through the plugin tool`; `edge case — agent looks up an unknown topic through the plugin tool`

**Fold (R6):** PR-1 — the 6 admin `mem_*` tools are registered but
`exposure: 'hidden'` by default; enabling the `all` profile re-registers
them `direct` (session_start + live config), no pi restart. PR-2 — the MCP
stdio child gets the shutdown-reason lifecycle (bounded drain ≤5s +
force-exit watchdog on `quit`; drain-without-exit on reload/resume/fork),
unref'd timers; the seam exposes the server capability probe (families
present per `tools/list`) that PR-1 gating reads. PR-6 — package ships
`agent-skill/` (SKILL.md discipline: "Use for / Not for" description,
per-command timeouts, single-background-mechanism rule), zero-dep `run.mjs`
launcher (ordered engine resolution; exit 78 = engine/config missing with
repair + config paths), manifest-tracked installer (install/uninstall/
status; never clobbers a foreign skill; stale links GC'd). Post-ship
packaging step, spec-locked here.

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

### Task 2: Port engram reliability behavior into the OpenCode plugin (fail-open split, retry, self-heal)

Critic F2 resolution (user decision): split capture-fails-open vs write-fails-closed.

**Files:**
- Create: `plugins/opencode/mnemonic-memory.ts` (extend), `plugins/opencode/tools/mem_*.ts` (extend to full v2 set), `plugins/opencode/skillgrid-compaction.ts` (extend). `tools/fact_*.ts` stay v1-legacy (untouched).

**Interfaces:**
- Consumes: the v2 MCP stdio server (`mnemonic mcp`): `mem_*` tools.
- Produces: capture/lifecycle fail-open; write fail-closed (single attempt, ambiguous-outcome recovery, no blind retry).
- Seam: plugin tool handler.
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `failure state — memory server unreachable through the plugin`; `failure state — retry and self-heal after transient failure`

**Fold (R6, OpenCode equivalents):** PR-1 — no exposure API on OpenCode:
registration-time gate + self-guarding "unavailable" response envelope for
gated tools (admin `mem_*` on the same profile rule as Pi). PR-2 — the host
lifecycle hooks drive the same bounded drain + force-exit watchdog for the
shared stdio child. PR-6 is host-agnostic and shipped once by Task 1 — this
task does not duplicate the skill.

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

### Task 3: Port engram behavior portability into the Pi plugin (compaction recovery, passive capture)

**Files:**
- Create: `plugins/pi/mnemonic/skillgrid-compaction.ts` (extend), `plugins/pi/mnemonic/skillgrid-events.ts` (extend).

**Interfaces:**
- Consumes: the v2 MCP stdio server (`mnemonic mcp`): session/compaction `mem_*` tools.
- Produces: compaction recovery + passive capture redaction.
- Seam: plugin event wiring.
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `happy path — compaction recovery persists prior work`; `edge case — passive capture redaction`

**Fold (R6):** PR-2 — compaction recovery is additionally triggered from the
named `session_before_compact` / `session_compact` events (deterministic
trigger, not only before_agent_start inference); the recovery persists prior
work in both trigger paths.

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

### Task 4: Wire server-side SaveEvent → brain-side wiki consumer (rewritten per F3)

Critic F3 resolution (user decision): rewrite Task 4 around the server-side event-bus seam.

**Files:**
- Create: `plugins/pi/mnemonic/skillgrid-events.ts` (extend).
- Verify (no server behavior change): the v2 memory function already emits SaveEvent per `004-transport-mcp` §5 — this task adds the emission-verification test, not the emission.
- Cross-repo (out of tree): the brain-side wiki consumer lives in `/data/git/AI/pi-llm-wiki` (wiki_observe / wiki_capture_source); this change pins the event contract and verifies ingestion end-to-end against that consumer — the consumer implementation stays the wiki repo's.

**Interfaces:**
- Consumes: the v2 memory save pipeline's SaveEvent (server-side bus; verify-only).
- Produces: wiki ingestion entry (brain-side `wiki_observe` / capture).
- Seam: server-side event bus.
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `happy path — save emits a SaveEvent for the wiki`; `edge case — wiki vault not initialized`; `failure state — wiki ingestion failure does not lose the memory save`

**Fold (R6):** PR-4 — the plugin-side consumer wiring uses a per-observation
in-flight lock map (TOCTOU when the same fact is re-emitted), bounded drain
with a shared teardown budget (≤5s; the un-drained remainder is counted, not
hung), and drop-count surfacing — the queue never grows unbounded.

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

### Task 5: Port full `code_*` parity into both code-index plugins (new Pi package + OpenCode entry)

**Files:**
- Create: `plugins/pi/codeindex/package.json`, `plugins/pi/codeindex/index.ts`, `plugins/pi/codeindex/codeindex-tools.ts` (new package; api seam shape reused from `plugins/pi/mnemonic/api.js`).
- Create: `plugins/opencode/mnemonic-codeindex.ts` (extend to full v2 set), `plugins/opencode/tools/code_*.ts` (extend to 25).
- Extract: `plugins/pi/mnemonic/mnemonic-codeindex.ts` → `plugins/pi/codeindex/` (memory package drops the codeindex module; `pi.extensions` unchanged otherwise).
- Test: parity mapping tests (both hosts) + staleness workflow + fail-open.
- Packaging: `@devopstales/pi-codeindex` gets `publishConfig` (GitHub npm, same as pi-mnemonic); `@devopstales/pi-mnemonic` bumps 0.1.1 → 0.2.0 with a README note that `code_*` moved to pi-codeindex (existing installers wanting code tools add the new package).

**Interfaces:**
- Consumes: the v2 MCP stdio server's `code_*` tools (code-index change shipped before this wave).
- Produces: registered `code_*` tools — the full 25-tool v2 set per the mapping table.
- Seam: plugin tool handler + shared api seam.
- Deletion test: pass-through.
- Adapters: ≥2.

**SATISFIES:** `happy path — agent searches code through the code-index plugin`; `edge case — stale index refresh workflow`; `failure state — memory server unreachable through the code-index plugin`

**Fold (R6, both hosts):** PR-1 — the 25 `code_*` tools are registered but
hidden (Pi `exposure`) / self-guarding (OpenCode) until the seam capability
probe reports the code-index family registered; re-evaluated at
session_start. PR-3 — described-OFF: when the semantic leg is unavailable,
`code_search` / `code_semantic` / `code_status` responses AND the health
surface carry one identical sentence naming the missing embedder + the
repair; FTS-leg results still served; reported as disabled, not unhealthy
(availability probed by resolution, memoized, fail-fast). PR-5 — subagent/
sandbox-context `code_read` is confined to the project root
(realpath-resolved) with a secret-file denylist (`.env*`, `id_*`, `*.pem`,
`*.tfstate`, shell history, credentials files); denial =
`read denied: <reason>`.

- [ ] **Step 1: Write the failing parity test (plugin set == registered set, both hosts)**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Commit**

### Task 6: Port `kb_*` surface into both knowledge plugins (gated on KB function ship)

**Files:**
- Create: `plugins/pi/knowledge/package.json`, `plugins/pi/knowledge/index.ts`, `plugins/pi/knowledge/knowledge-tools.ts` (new package, gated).
- Create: `plugins/opencode/mnemonic-knowledge.ts` (new entry), `plugins/opencode/tools/kb_*.ts`.
- Test: parity mapping against KB spec 008 tool table (both hosts); async write semantics (run_id + poll, 409 double-start); degradation envelope surfacing; read tools synchronous.
- Packaging: `@devopstales/pi-knowledge` gets `publishConfig` (GitHub npm, same as pi-mnemonic) on first publish.

**Interfaces:**
- Consumes: the v2 MCP stdio server's `kb_*` tools (pinned by the KB change's contract).
- Produces: agent-profile `kb_*` tools on both hosts; admin tools via `all` profile/ops path.
- Seam: plugin tool handler + shared api seam.
- Deletion test: pass-through.
- Adapters: ≥2.

**Precondition:** `2026-10-31-mnemonic-v2-knowledge-base` shipped. Until then this task's
parity test asserts against the KB spec 008 tool table (spec-pinned constants, 12 tools), not a
live server; the live half runs green only once the `kb_*` tools are registered.

**SATISFIES:** `happy path — agent queries the knowledge base through the plugin`; `edge case — async KB write does not double-start`; `failure state — degraded KB run surfaces the envelope, never fails the agent`

- [ ] **Step 1: Write the failing parity test (spec-pinned kb_* table, both hosts)**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation**
- [ ] **Step 4: Run test to verify it passes (live half when KB routes land)**
- [ ] **Step 5: Commit**

## Blueprint validation (after writing, before proceeding)

1. **Critic subagent (mandatory — Tier T2).** Dispatch a dedicated critic subagent with the blueprint content; structured critique (issues + severity).
2. **User review (always mandatory).** Present the blueprint + critic findings to the user; for each issue, present options; update blueprint per decision; re-present final blueprint; user must approve the final blueprint before `acceptance.feature` is written.

Never proceed to the next artifact without the user approving the final blueprint.
