# Mnemonic-v2 Agent Plugins (memory / code-index / knowledge) — Design Briefing (requirements & intent)

> **STATUS:** `sliced` (2026-11-05; amended 2026-11-05: 3-family split per user direction)

> Copy this template to `.skillgrid/specs/2026-11-04-mnemonic-v2-memory-plugins/briefing.md`.
> **Role:** this is the *what* — the validated design/spec: problem, intent,
> requirements, success criteria. The companion `blueprint.md` is the *how*
> (implementation plan). The filename pair is load-bearing: `brainstorming` and
> `resume` key phase state off which of the two files exist. Do not rename.

## Problem / Intent

The mnemonic-v2 server exposes three tool families — `mem_*`/`fact_*` (memory),
`code_*` (code-index), `kb_*` (knowledge-base; the KB spec's "one MCP server,
three tool families" pattern, `2026-10-31-mnemonic-v2-knowledge-base`
`specs/008-transport.md` §1) — and the memory function already emits the
SaveEvent contract consumed by the brain-side wiki
(`/data/git/AI/pi-llm-wiki`). The missing piece is the agent-facing plugin layer
that exposes **all three families** to OpenCode and Pi agents as **three plugins
per host**:

1. **Memory plugin** (engram-based) — `mem_*` + `fact_*` + session/compaction wiring.
2. **Code-index plugin** — `code_*`.
3. **Knowledge-base plugin** — `kb_*` (gated on the KB function shipping).

The existing skillgrid plugin packages cover only a subset of the registered
surface in a single grab-bag package per host. This design specifies the
3-family × 2-host plugin layer.

## Purpose & Success Criteria

- **Purpose:** give OpenCode and Pi agents first-class, engram-parity memory
  tooling plus full code-index and knowledge-base tooling over the
  mnemonic/mnemonic-v2 memory function.
- **Success criteria:**
  1. Plugin↔server contract (`mnemonic-v2/internal/transport/mcp/contract_test.go`
     — the D1-pinned v2 MCP contract) green.
  2. Full agent-facing tool-surface parity per family (memory: engram baseline
     floor ↔ full registered `mem_*`; code-index: full `code_*`;
     knowledge: `kb_*` per KB spec 008) green for both hosts.
  3. Engram-proven behavior port verified (fail-open, retries, compaction recovery,
     session wiring) on the memory plugin; fail-open consistency on all families.
  4. Design approved via brainstorming; spec written.
- **Out of scope:** brain-side wiki implementation (pi-llm-wiki owns it); the
  KB *function* implementation (owned by `2026-10-31-mnemonic-v2-knowledge-base`);
  server-side behavior changes (R3 verifies the existing v2 SaveEvent emission,
  it does not change it); v1 `fact_*` tools (v1-HTTP legacy, retained, not
  re-targeted at v2); plugin packaging for `web_cache_*`, `scan_*`, `dep_*`,
  skill/agent/squad tools (later changes); hosts beyond OpenCode + Pi; the v1
  `mnemonic/` module surface (legacy; the plugins' v1-HTTP wiring is retained
  until the v2 cutover, not extended).

## Context

- **Engine = the v2 module** (`mnemonic-v2/`, module
  `github.com/devopstales/skillgrid/mnemonic-v2`, go 1.26) — the total rewrite of
  `mnemonic/` per `00-foundation/000-architecture.md` ("Mnemonic v2 is a total
  rewrite of `mnemonic/`. It has five functions: memory, code-index, llm-wiki
  (the brain, = the knowledge-base change), opencode-plugin, pi-plugin"). The
  v1 module (`mnemonic/`) is the legacy monolith (its `serve` HTTP API on
  :7438, `fact_*` tools, old `mem_*`/`code_*` surfaces) — still running, but
  not the parity target.
- **MCP surface (the only v2 server transport):** `mnemonic-v2/internal/transport/mcp` —
  spec `004-transport-mcp` ("Transport: MCP stdio" — the MCP stdio server is
  the primary agent interface; `mnemonic serve` is a deprecated no-op in v2).
  Registered today: **28 `mem_*`** (22 agent-profile + 6 admin, D1-pinned by
  `mnemonic-v2/internal/transport/mcp/contract_test.go`), **25 `code_*`**
  (code-index function, ship-pending), **12 `kb_*`** (knowledge-base function,
  DRAFT — not yet registered).
- **Plugins consume the engine only through the MCP surface** (architecture
  §1): "The two plugins are TypeScript packages under `plugins/`, outside the
  Go module, and consume the engine only through the MCP surface (spec
  `004-transport-mcp`)." The existing plugin wiring (HTTP to v1 `:7438`,
  `SKILLGRID_MNEMONIC_HTTP_URL`) is v1-era legacy; the v2 plugin transport is
  MCP stdio to the `mnemonic mcp` process, following the engram runtime
  contract pattern.
- Engram baseline: `/data/git/AI/engram/plugin/pi` (Pi package
  `gentle-engram@0.6.0`), `/data/git/AI/engram/plugin/opencode` — the 27-tool
  MCP surface + 23-tool `agent` profile (per `004-transport-mcp` reference
  line).
- skillgrid plugin packages: `plugins/pi/mnemonic/` (`@devopstales/pi-mnemonic`),
  `plugins/opencode/` (mnemonic-*, skillgrid-*, squad-*, code-*, fact-* tools).
- Brain-side: `/data/git/AI/pi-llm-wiki` (wiki_observe, wiki_recall, wiki_capture_source,
  …).
- **Code-index function:** `2026-10-31-mnemonic-v2-code-index` — QA pass, review
  met-with-fixes, fix-loop green, ship-pending. Registers the 25 v2 `code_*`
  tools (code_index/search/read/grep/orient/explore/impact/affected/deps/
  dep_impact/path/packages/circles/navigates/rename/resolve/route/status/
  embedding_status/unresolved/watch/sql_schema/docs/explain/configs).
- **Knowledge-base function:** `2026-10-31-mnemonic-v2-knowledge-base` (DRAFT, queued
  behind code-index ship; the "llm-wiki" function in the v2 five-function plan).
  `specs/008-transport.md` finalizes the `kb_*` surface: **12 tools** — agent
  profile: `kb_query`, `kb_read`, `kb_search`, `kb_generate`, `kb_validate`,
  `kb_lint`, `kb_status`, `kb_cite_source`; admin profile: `kb_reindex`,
  `kb_index`, `kb_bootstrap`, `kb_rebuild_meta`. Writes are async (`run_id` +
  poll, `409` on double-start); reads synchronous; degradation envelope
  (INV-3) on every tool; KB writes through a dedicated write queue.
- **fact_* (8 tools) is v1-only** — the v2 surface has no fact family (in v2,
  durable facts are memory observations). The existing fact tools remain
  v1-HTTP legacy, retained but outside this change's parity target.
- **Current plugin coverage (subset, v1-HTTP era):** OpenCode package exposes
  6 `mem_*` + 5 `code_*` + 4 `fact_*` + squad tools against v1 :7438; Pi
  package bundles memory + facts + codeindex + ctx-sandbox modules in one
  extension. Full parity not in place for any family.

 - **Fits existing patterns:** yes — the v2 MCP tool contract is pinned
   (`mnemonic-v2/internal/transport/mcp/contract_test.go`, D1); existing skillgrid
   plugin packages already carry the api/memory/facts/codeindex/compaction/context/events
   module shape; engram baseline supplies proven behavior patterns (27-tool MCP
   surface, 23-tool agent profile — the v2 design already mirrors it). Port =
   reconcile engram tool names onto the v2 registered set, then reuse the proven
   patterns over the v2 MCP stdio transport. The 3-family split mirrors the
   server's own tool-family architecture (KB spec 008 §1).

## Approaches Considered

- **Chosen:** Approach A — port engram plugin architecture into the existing
  skillgrid plugin packages; full parity per family; OpenCode + Pi; target
  `mnemonic`/`mnemonic-v2`. 2026-11-05 amendment (user direction): deliver as
  **three plugin families per host** — Pi: `@devopstales/pi-mnemonic` (memory,
  existing package, codeindex module extracted), `@devopstales/pi-codeindex`
  (new), `@devopstales/pi-knowledge` (new, gated); OpenCode: three plugin
  entries in `@skillgrid/opencode-plugins` (memory / code-index / knowledge).
- **Rejected:** Approach B — build a fresh adapter against the mnemonic-v2 API;
  Approach C — thin adapter / delegate to the engram server.

## Requirements

### Requirement: Full agent-facing memory tool-surface parity

The system SHALL expose the full registered v2 memory tool surface (28 `mem_*`:
22 agent-profile + 6 admin, per `mnemonic-v2/internal/transport/mcp/contract_test.go`)
through the OpenCode and Pi memory plugins, backed by the mnemonic-v2 engine.
The engram baseline (27-tool MCP surface, 23-tool `agent` profile) is the
behavior floor, not the ceiling: the target is the full registered v2 set, with
the engram mapping table as the parity anchor and `mem_lookup` as the
mnemonic-v2 delta. The v1 `fact_*` family is out of the parity target (v1-HTTP
legacy; in v2 durable facts are memory observations).

- **Current:** engram baseline exposes a 27-tool memory surface;
  mnemonic-v2 registers 28 `mem_*`; skillgrid plugins cover a subset against
  the v1 HTTP API; full parity not yet in place.
- **Target:** OpenCode + Pi memory plugins register the full v2 `mem_*` set
  (agent profile exposed by default; admin tools via the `all` profile/ops
  path).
- **Acceptance:** parity mapping test green (plugin-registered set == v2
  registered `mem_*` set, both hosts); v2 MCP contract green.

### Requirement: Engram-proven behavior port

The system SHALL port engram-proven plugin behaviors (redaction, retries, self-heal,
fail-open delivery, compaction recovery, passive capture, session wiring) into the
memory plugin packages without behavior drift.

- **Current:** skillgrid plugins implement variants (skillgrid-events, skillgrid-compaction,
  ctx-sandbox).
- **Target:** engram architecture patterns ported.
- **Acceptance:** reliability/compaction/session behaviors verified by ported tests.

### Requirement: Memory→wiki handoff contract

The memory save pipeline SHALL emit the SaveEvent consumed by the brain-side wiki.

- **Current:** memory server emits SaveEvent; wiki consumes via `wiki_observe` /
  `wiki_capture_source`.
- **Target:** save → SaveEvent → wiki ingestion intact through the plugins.
- **Acceptance:** the save pipeline emits a SaveEvent carrying the observation
  identity on successful persist, and the brain-side wiki can ingest it via its
  capture tool without failing the memory save.

### Requirement: Code-index plugin parity

The system SHALL expose the full v2 `code_*` tool surface (25 tools, per the
registered set of the `2026-10-31-mnemonic-v2-code-index` change) through the
OpenCode and Pi code-index plugins.

- **Current:** OpenCode package exposes 5 v1-era `code_*` tools; Pi bundles a
  codeindex module in the memory package; full v2 `code_*` parity not in place.
- **Target:** both hosts register the full v2 `code_*` set; staleness workflow
  (code_status → code_index → code_search) preserved; fail-open consistent with
  the memory plugin (shared api seam).
- **Acceptance:** parity mapping test green (plugin-registered set == v2
  registered `code_*` set, both hosts); v2 MCP contract green.

### Requirement: Knowledge-base plugin

The system SHALL expose the `kb_*` tool surface (12 tools per
`2026-10-31-mnemonic-v2-knowledge-base` `specs/008-transport.md`) through the
OpenCode and Pi knowledge-base plugins.

- **Gated:** server-side `kb_*` routes ship with the KB function (queued behind
  code-index ship). This requirement is verifiable — and its scenarios gated —
  once that change lands.
- **Current:** no `kb_*` plugin exposure; KB function DRAFT.
- **Target:** agent-profile tools (`kb_query`, `kb_read`, `kb_search`,
  `kb_generate`, `kb_validate`, `kb_lint`, `kb_status`, `kb_cite_source`)
  exposed on both hosts; admin tools available via the `all` profile/ops path;
  HTTP mirror routes consumed; async write semantics honored (`run_id` + poll,
  `409` on double-start); degradation envelope (`degraded: true` + reason)
  surfaced to the agent, never silently floored.
- **Acceptance:** parity mapping test green (plugin-registered set == `kb_*` set
  from KB spec 008, both hosts); KB route contract green (KB change's
  contract owner).

## Implementation Decisions

- **Modules to build/modify:**
  - Pi: `plugins/pi/mnemonic/*` (amend — memory family: api.js, index.ts,
    mnemonic-memory.ts, mnemonic-facts.ts (v1-legacy, retained), ctx-sandbox.ts,
    skillgrid-events.ts, skillgrid-sdd-events.ts, skillgrid-compaction.ts);
    `plugins/pi/codeindex/` (new package); `plugins/pi/knowledge/` (new
    package, gated).
  - OpenCode: `plugins/opencode/*` (amend — three entries: mnemonic-memory.ts,
    mnemonic-codeindex.ts, mnemonic-knowledge.ts (new) + `tools/mem_*`,
    `tools/code_*`, `tools/kb_*`; `tools/fact_*` v1-legacy retained).
  - Contract owners: `mnemonic-v2/internal/transport/mcp/contract_test.go`
    (D1-pinned `mem_*` contract; code-family contract per the code-index
    change); KB family pinned by the KB change's contract. The v1
    `mnemonic/internal/http/plugin_route_contract_test.go` is legacy (v1
    HTTP era) — not the parity owner.
- **Interfaces:** tool names (full per-family sets, v2 registered), MCP stdio
  protocol (engram runtime contract pattern), profiles (agent default; admin
  via `all`).
- **Data flow:** agent tool call → redact → validate → dispatch (MCP
  `tools/call`) → persist → SaveEvent → wiki/KB ingestion (KB ingestion is the
  KB function's subscription, not the plugin's).
- **Error handling:** fail-open, retries, self-heal, honest reporting.
- **Dependencies:** none new (MCP stdio JSON-RPC over a spawned
  `mnemonic mcp` child process — stdio child process + JSON, no extra runtime
  deps; if an MCP client library becomes necessary that is a new-dependency
  ADR).

## Testing Decisions

- **What makes a good test:** external behavior only (tool outcome, fail-open, recovery).
- **Modules to test:** plugin tool handlers (parity mapping per family), route
  contract, reliability paths, handoff.
- **Prior art:** `mnemonic/internal/http/plugin_route_contract_test.go`, engram
  plugin tests, `plugins/opencode/test/` (node --experimental-strip-types).

## Impact on Global Docs

- None — no product-scope change. (ADR candidate for the plugin-port decision,
  if recorded.)

- `.skillgrid/knowledge/reference/00-prd.md`: None
- `.skillgrid/knowledge/ASSUMPTIONS.md`: ADR candidate (plugin port decision;
  3-family split mirrors KB spec 008 §1)
- `.skillgrid/knowledge/ARCHITECTURE.md`: None

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.80  | 0.75 | OK     |                                    |
| Boundary Clarity    | 0.80  | 0.70 | OK     |                                    |
| Constraint Clarity  | 0.75  | 0.65 | OK     |                                    |
| Acceptance Criteria | 0.55  | 0.70 | **WARN**| acceptance criteria authored post-design-approval |

**Clarity:** 0.26 (above gate) — Acceptance dimension below minimum → ⚠, listed in
Open Questions & Assumptions.

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | v1 toolset scope        | Full engram surface                |
| 1     | host scope              | OpenCode + Pi                      |
| 2     | adapter source          | Port engram architecture           |
| 2     | tool surface delta      | Full parity, vs mnemonic/mnemonic-v2 |
| 2     | server binding          | mnemonic/mnemonic-v2               |
| 3     | implementation placement| Extend existing skillgrid plugins  |
| 3     | brain-side source       | /data/git/AI/pi-llm-wiki           |
| 4 (2026-11-05) | plugin split       | 3 families: memory / code-index / knowledge, both hosts |
| 4 (2026-11-05) | KB plugin target    | KB function `kb_*` (gated on brain ship) |
| 4 (2026-11-05) | spec structure        | One amended spec (3 families × 2 hosts) |
| 4 (2026-11-05) | parity source of truth | v2 module (`mnemonic-v2/internal/transport/mcp`), MCP stdio; v1 surface + `fact_*` legacy |

## Open Questions & Assumptions

- **Question:** final acceptance criteria set (authored post design-approval).
- **Assumption:** design approval ("look good") satisfies the design gate; acceptance
  criteria finalize the verifiable success criteria above.
- **Assumption:** R5 (knowledge-base plugin) scenarios are gated on the KB
  function's ship; wave 3 of execution lands after it.

## Decisions (ADR)

- ADR candidate: port engram plugin architecture as the baseline for the OpenCode + Pi
  memory plugins (hard-to-reverse + surprising + real trade-off) — record via
  `skillgrid:architectural-decision-records` if cleared.

## Terms

- None (no new domain terms introduced by this design).
