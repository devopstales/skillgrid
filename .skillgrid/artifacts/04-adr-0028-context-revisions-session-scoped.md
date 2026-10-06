# context_revisions is session-scoped audit

---
status: "accepted"
supersedes: none
date: 2026-10-06
---

## Context and Problem Statement

The CLM layer (ADR-0027) persists each accepted mirror edit as a `context_revisions` row: which raw prefix it replaces (count + SHA-256 digest), the resulting messages, size estimates, and a per-message edit trace. The question is the table's lifecycle. We have three precedents: `tool_outputs` (session-scoped, purged at session end), `indexed_files` / `observations` (project-scoped, 7-day TTL), and `lifecycle_log` (permanent, append-only audit).

The revision is an *audit record of a transform*, not a *source of truth*. The source of truth is the raw session history (append-only, never rewritten) plus the current mirror. The active revision is re-derivable: given the raw history and the mirror, the transform is the edit the model made. So the table is a record of *what the model did to its context*, not *what the context is*.

That framing decides the lifecycle. A permanent table (c) would grow without bound across sessions for a record that is re-derivable. A project-scoped 7-day TTL (b) would keep revisions from one session visible in the next — but the next session has a different raw history, so a revision anchored to the old session's prefix is not applicable to the new one. Session-scoped (a) matches the anchor: a revision is meaningful only within the session whose raw history it anchors to.

## Considered Options

- Session-scoped: one row per session, purged at session end (like `tool_outputs`)
- Project-scoped, 7-day TTL: revisions persist across sessions (like `indexed_files` / `observations`)
- Permanent: append-only audit history, never purged (like `lifecycle_log`)

Chosen option: "Session-scoped," because the revision is anchored to a specific session's raw history (by count + SHA-256 digest) and is only meaningful within that session. It is an audit record of a transform, not a source of truth — the raw history + mirror are. Session-scoped keeps the store small and matches the anchor.

- `context_revisions` is created by migration `052_context_revisions.sql` (after `049_session_checkpoint.sql`; 050 = `tool_outputs` per ADR-0025, 051 = `indexed_files` per ADR-0026). Columns: `id`, `session_id`, `project_id`, `revision` (int, monotonic per session), `anchor_count` (raw message count the revision replaces), `anchor_digest` (SHA-256 of the anchored raw prefix), `messages` (JSON, the resulting message list), `size_estimate` (int, tokens), `calibration_factor` (float, the corrected estimate factor), `withhold_ids` (JSON, the tool-result ids the overflow guard withheld), `edit_trace` (JSON, per-message edit trace), `created_at`.
- The table is purged at session end, in the same pass as `tool_outputs` (the `ctx purge` path and the session-end hook). `ctx purge` clears both.
- `ctx clm history` (if added in a later change) lists the current session's compaction points — the revisions for the active session. Cross-session revision history is out of scope.
- The active revision is the one with the highest `revision` for the session. On resume, the plugin reconstructs the active revision from the row and validates it against the raw prefix (anchor_count + anchor_digest must match); a mismatch discards the revision and falls back to raw context.

### Consequences

- Good, because the lifecycle matches the anchor: a revision is meaningful only within the session whose raw history it anchors to, and it is purged with that session.
- Good, because the store stays small: revisions are purged at session end, not accumulated across the project.
- Good, because the source of truth is unambiguous: the raw history + mirror, not the revision table. The table is an audit log.
- Bad, because cross-session revision history is lost: `ctx clm history` (if added) shows only the current session. If a need for cross-session compaction history emerges, it is a new ADR that supersedes this.
- Bad, because a revision is re-derivable but re-derivation requires the raw history + mirror to be intact. If the raw history is compacted by OpenCode before the session ends, the anchor no longer validates and the revision is discarded — the audit record is lost even though the row still exists. This is acceptable: the raw history is the source of truth, and a compaction that breaks the anchor is itself a context event.
