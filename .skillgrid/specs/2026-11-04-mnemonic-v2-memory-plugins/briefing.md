# Mnemonic-v2 Memory Plugins — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-04)

> Copy this template to `.skillgrid/specs/2026-11-04-mnemonic-v2-memory-plugins/briefing.md`.
> **Role:** this is the *what* — the validated design/spec: problem, intent,
> requirements, success criteria. The companion `blueprint.md` is the *how*
> (implementation plan). The filename pair is load-bearing: `brainstorming` and
> `resume` key phase state off which of the two files exist. Do not rename.

## Problem / Intent

The mnemonic-v2 memory function — the agent-side durable fact store with save
pipeline, retrieval ladder, sessions, facts, and code context — exists in
skillgrid (`mnemonic` + `mnemonic-v2` modules, MCP transport, HTTP routes) and
already emits the SaveEvent contract consumed by the brain-side wiki
(`/data/git/AI/pi-llm-wiki`). The missing piece is the agent-facing plugin layer
that exposes this function to OpenCode and Pi agents. This design specifies that
plugin layer.

## Purpose & Success Criteria

- **Purpose:** give OpenCode and Pi agents first-class, engram-parity memory
  tooling over the mnemonic-v2 memory function.
- **Success criteria:**
  1. Plugin↔server route contract (`mnemonic/internal/http/plugin_route_contract_test.go`) green.
  2. Full agent-facing memory tool-surface parity (engram baseline ↔ mnemonic-v2) green.
  3. Engram-proven behavior port verified (fail-open, retries, compaction recovery, session wiring).
  4. Design approved via brainstorming; spec written.
- **Out of scope:** brain-side wiki implementation (pi-llm-wiki owns it); server-side
  behavior changes; hosts beyond OpenCode + Pi.

## Context

- Mnemonic-v2 memory function: `mnemonic/` + `mnemonic-v2/` Go modules; MCP tool set
  in `internal/transport/mcp` (mem_* set incl. `mem_lookup`, `mem_merge_projects`,
  `mem_delete`, `mem_stats`, `mem_timeline`, `mem_review`, `mem_doctor`,
  `mem_pin`/`unpin`, `mem_judge*`, `mem_compare`, `mem_consolidate`); HTTP routes
  pinned by `internal/http/plugin_route_contract_test.go`.
- Engram baseline: `/data/git/AI/engram/plugin/pi` (Pi package `gentle-engram@0.6.0`),
  `/data/git/AI/engram/plugin/opencode`.
- skillgrid plugin packages: `plugins/pi/mnemonic/` (`@devopstales/pi-mnemonic`),
  `plugins/opencode/` (mnemonic-*, skillgrid-*, squad-*, code-*, fact-* tools).
- Brain-side: `/data/git/AI/pi-llm-wiki` (wiki_observe, wiki_recall, wiki_capture_source,
  …).

 - **Fits existing patterns:** yes — plugin↔server route contract already pinned
   (`mnemonic/internal/http/plugin_route_contract_test.go`); existing skillgrid plugin
   packages already carry the api/memory/facts/codeindex/compaction/context/events module
   shape; engram baseline supplies proven behavior patterns. Port = reconcile engram tool
   names onto mnemonic-v2's registered set, then reuse the proven patterns.

## Approaches Considered

- **Chosen:** Approach A — port engram plugin architecture into the existing
  skillgrid plugin packages; full parity; OpenCode + Pi; target `mnemonic`/`mnemonic-v2`.
- **Rejected:** Approach B — build a fresh adapter against the mnemonic-v2 API;
  Approach C — thin adapter / delegate to the engram server.

## Requirements

### Requirement: Full agent-facing memory tool-surface parity

The system SHALL expose the full engram-baseline agent-facing memory tool surface
through the OpenCode and Pi plugins, backed by the mnemonic/mnemonic-v2 memory
function.

- **Current:** engram baseline exposes a 27-tool memory surface; mnemonic/mnemonic-v2
  registers a ~26-tool `mem_*` set + code tools; skillgrid plugins cover a subset; full
  parity not yet in place.
- **Target:** OpenCode + Pi plugins register the full baseline surface.
- **Acceptance:** parity mapping test green (engram tool names ↔ mnemonic-v2 registered
  tools); plugin route contract green.

### Requirement: Engram-proven behavior port

The system SHALL port engram-proven plugin behaviors (redaction, retries, self-heal,
fail-open delivery, compaction recovery, passive capture, session wiring) into the
existing skillgrid plugin packages without behavior drift.

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

## Implementation Decisions

- **Modules to build/modify:** `plugins/pi/mnemonic/*`, `plugins/opencode/*`; contract
  owner `mnemonic/internal/http/plugin_route_contract_test.go`.
- **Interfaces:** tool names (full baseline), HTTP routes (pinned), protocol text.
- **Data flow:** agent tool call → redact → validate → dispatch → persist → SaveEvent →
  wiki ingestion.
- **Error handling:** fail-open, retries, self-heal, honest reporting.
- **Dependencies:** none new.

## Testing Decisions

- **What makes a good test:** external behavior only (tool outcome, fail-open, recovery).
- **Modules to test:** plugin tool handlers, route contract, reliability paths, handoff.
- **Prior art:** `mnemonic/internal/http/plugin_route_contract_test.go`, engram plugin tests.

## Impact on Global Docs

- None — no product-scope change. (ADR candidate for the plugin-port decision, if recorded.)

- `.skillgrid/knowledge/reference/00-prd.md`: None
- `.skillgrid/knowledge/ASSUMPTIONS.md`: ADR candidate (plugin port decision)
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

## Open Questions & Assumptions

- **Question:** final acceptance criteria set (authored post design-approval).
- **Assumption:** design approval ("look good") satisfies the design gate; acceptance
  criteria finalize the verifiable success criteria above.

## Decisions (ADR)

- ADR candidate: port engram plugin architecture as the baseline for the OpenCode + Pi
  memory plugins (hard-to-reverse + surprising + real trade-off) — record via
  `skillgrid:architectural-decision-records` if cleared.

## Terms

- None (no new domain terms introduced by this design).
