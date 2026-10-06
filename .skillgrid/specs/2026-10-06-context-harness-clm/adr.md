# ADR Review Manifest

- **Change:** `2026-10-06-context-harness-clm`
- **Status:** in review
- **Review date:** 2026-10-06

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md` — local SQLite stays the per-machine source of truth; this change adds tables to the per-project store, not a storage swap.
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — fail-open floors hold: the capture gate, `ctx_search`, and the CLM overflow guard all degrade to no-op / raw context on failure and never block the agent.
- `.skillgrid/artifacts/04-adr-0021-pre-tool-policy-fail-open.md` — the capture gate is on actual output in the PostToolUse seam (not a PreToolUse prediction); `SKILLGRID_CTX_BYPASS` is the only forcing mechanism, consistent with the fail-open policy posture.
- `.skillgrid/artifacts/04-adr-0022-host-agent-memory-checkpoint.md` — the `checkpoint` capture mode is the turn-end seam the CLM mirror read reuses; the Memory Checkpoint and the CLM revision are distinct features sharing a seam.
- `.skillgrid/artifacts/04-adr-0024-central-pg-shared-memory-state.md` — the new tables are per-machine derived state (session-scoped `tool_outputs`/`context_revisions`; project-scoped `indexed_files`). `indexed_files` could later join the synced set (like observations); `tool_outputs`/`context_revisions` are session-scoped and stay local. No sync change required.
- Locked constraint "No new dependencies without an ADR" — satisfied: no new Go or Node dependencies; the OpenCode `context` plugin hook is an existing harness surface, not a new dependency.

## New Durable ADRs Created

- `.skillgrid/artifacts/04-adr-0025-context-harness-owner.md` — Context Harness owns the session context lifecycle: absorbs `session_inject` (import path change only, `mem_inject_session` unchanged), adds capture / `ctx_query` / routing / `ctx` CLI; capture gates on actual output. Migration 050.
- `.skillgrid/artifacts/04-adr-0026-indexed-files-reuses-observations-schema.md` — `indexed_files` is a separate project-scoped table reusing the observations schema shape (Second Brain has no table of its own); `ctx_search` fuses sandbox + indexed via RRF. Migration 051.
- `.skillgrid/artifacts/04-adr-0027-clm-context-language-model.md` — CLM layer: Go owns state and decisions (budget, overflow guard, calibration, revision), Node owns the request path (render mirror, apply withhold, read edit); opt-in via the `clm:` config block.
- `.skillgrid/artifacts/04-adr-0028-context-revisions-session-scoped.md` — `context_revisions` is session-scoped audit, purged at session end; raw history + mirror are the source of truth. Migration 052.

## Supersessions

- None. ADR-0022 (checkpoint) and ADR-0024 (central PG) remain in force; this change adds a new owner and layer above them.
