# Observations are bi-temporal; the save path classifies each write as Add/Update/Delete/Noop

---
status: "accepted"
supersedes: none
date: 2026-09-24
---

## Context and Problem Statement

A memory system that only appends facts never answers the question "what was true
when?" Consider: in session 1 the agent saves "We use SQLite for storage." In
session 3 the stack changes and the agent saves "We migrated to Postgres." Both
observations are live, both rank in search, and the agent gets both — with no way
to tell that the second supersedes the first, when the supersession happened, or
what was true before it.

The codebase already has the *lifecycle flag* (`status = 'superseded'` in
`governance.go:27`) and the *forward pointer* (`supersedes` edge in
`memory_relations`, migration 006), but neither is **temporal**: there is no
`valid_at` (when did this fact become true) or `invalid_at` (when did it stop
being true), and no **back-pointer** (`superseded_by`) to walk the chain
backward. Supersession is manual (HTTP `POST .../status`) and never auto-fires on
save. The save cascade (`service.go:425-603`) is a binary dedup: "is this a dup?
(merge) or not? (insert)." It has no Delete arm and no explicit classification —
the agent cannot see *what happened* when it calls `mem_save`.

Two reference systems (mnemonic-ai, Rust; Mnemon, Go) both converged on the same
fix: bi-temporal columns on the fact row plus a save-time classifier that decides
Add/Update/Delete/Noop and produces the supersede chain automatically.

## Considered Options

- **A. Full bi-temporal + AUDN classifier.** Add `valid_at`, `invalid_at`,
  `superseded_by` columns to `observations`. Extend the `DedupLLM` seam from a
  binary `(bool, int64)` to a 4-way `DedupDecision` struct. The Delete arm
  produces the supersede chain (soft-delete old, set `invalid_at` +
  `superseded_by`, add `supersedes` edge). The `mem_save` MCP response gains
  `action` + `superseded_id` fields. This is the complete pattern.

- **B. Columns only, manual supersede.** Add the three columns but keep
  supersede manual (status + edge, as today). No auto-supersede on save. The
  columns enable future bi-temporal queries without a migration later.
  Cheaper now, but the agent still has to remember to supersede — the same
  failure that prompted this ADR.

- **C. Keep status + edge, skip bi-temporal.** The `status = 'superseded'` +
  `supersedes` edge mechanism is sufficient. We don't have facts that become
  true then false over time; we have facts that get refined. The refinement is
  the topic-key upsert path (versioned, not superseded).
  Cheapest, but loses the "what was true at time T" query and the automatic
  supersede chain. The agent can't answer "did we always use Postgres?" or
  "when did we switch from SQLite to Postgres?"

Chosen option: **A. Full bi-temporal + AUDN classifier**, because the
bi-temporal columns are what make the supersede chain *queryable* (time-travel
queries, "what was true at T"), and the AUDN classifier is what makes the
supersede chain *automatic* (the save path produces it, the agent doesn't have
to remember). Option B ships the columns but not the automation — the failure
persists. Option C avoids both costs but also the capability.

### Implementation

- `store/migrations/043_bitemporal.sql` — new file (matching the 040-042
  separate-file pattern): `ALTER TABLE observations ADD COLUMN valid_at TEXT;
  ALTER TABLE observations ADD COLUMN invalid_at TEXT; ALTER TABLE observations
  ADD COLUMN superseded_by INTEGER;` + partial index `idx_obs_invalid_at`.
- `memory/service.go` — `Observation` struct: add `ValidAt string`,
  `InvalidAt string`, `SupersededBy sql.NullInt64`. `obsSelectCols`: add 3
  columns. `scanObservations`: add 3 scan vars. 4 inline FTS/admin SELECTs
  (service.go:689, 748; governance.go:446, 492): add 3 columns to each.
  `Save` INSERT: add 3 columns (valid_at = created_at, invalid_at = NULL,
  superseded_by = NULL).
- `memory/types.go` — `DedupLLM` interface: add `Classify` method returning
  `DedupDecision{Verdict, TargetID, Confidence, Reason}`. Keep `Dedup` as
  deprecated wrapper. `runDedupCheck`: call `Classify` instead of `Dedup`.
- `memory/service.go` — new `SaveWithAction(ctx, in) (SaveResult, error)`
  method. `Save` delegates to it and extracts `.ID`. `SaveResult{ID, Action,
  SupersededID}`. The 4-way routing in `SaveWithAction`: noop →
  `BumpDuplicate` + return; add → INSERT; update → topic-key upsert or new
  upsert-by-id; delete → `MarkSuperseded` + INSERT new.
- `memory/lifecycle.go` — new `MarkSuperseded(ctx, id, supersededByID)` method.
  Pattern: `UPDATE observations SET invalid_at = ?, superseded_by = ?, status =
  'superseded' WHERE id = ? AND project = ? AND deleted_at IS NULL`.
- 7 primary read sites — add bi-temporal filter:
  `AND (o.invalid_at IS NULL OR o.invalid_at = '' OR strftime('%s',
  o.invalid_at) > strftime('%s', 'now'))`. Sites: service.go:696, 755, 792,
  827, 847; governance.go:453, 498.
- `mcp/tools_memory.go` — `handleMemSave`: call `SaveWithAction` instead of
  `Save`. Add `action` + `superseded_id` to the response map (additive JSON).

### Consequences

- Good, because the agent can answer "what was true at time T?" — a time-travel
  query over the supersede chain.
- Good, because the save path produces the supersede chain automatically — the
  agent doesn't have to remember to call `mem_judge` or set status.
- Good, because the `action` field in the `mem_save` response tells the agent
  what happened (add/update/delete/noop) — the agent can adjust its behavior
  (e.g., "I just deleted a fact, I should verify the new one is correct").
- Bad, because the `DedupLLM` interface changes (binary → 4-way). Existing test
  mocks break. Mitigation: keep `Dedup` as a deprecated wrapper; add `Classify`
  as a new method. Existing implementations keep compiling.
- Bad, because 7 read sites need the bi-temporal filter. If one is missed,
  superseded observations leak into results. Mitigation: the filter is
  additive (a new `AND` clause); a missed site is a test failure, not a data
  corruption. The acceptance test "superseded observations do not appear in
  search results" catches it.
- Bad, because the 4 inline FTS SELECTs duplicate the column list instead of
  using `obsSelectCols`. If the 3 new columns are added to `obsSelectCols` but
  not to these 4, `scanObservations` fails at runtime with a column-count
  mismatch. Mitigation: the migration task explicitly lists all 5 SELECT sites.

### Revisit criteria

Revisit this ADR (write a superseding ADR) when **any** of:

1. The observation store moves off SQLite (e.g., to a graph database) and the
   bi-temporal model is replaced by a native temporal edge.
2. The AUDN classifier proves too aggressive (false "delete" verdicts
   superseding facts that were still true) and we need to gate it behind a
   confidence threshold or a user-confirmation step.
 3. The bi-temporal columns become a performance bottleneck on search (the
    `strftime('%s', ...)` comparison in the search filter prevents the partial
    index `idx_obs_invalid_at` from being used for the "is invalid_at in the
    future" branch; the `IS NOT NULL` branch can use the index) and we need to
    switch to a stored integer timestamp or a computed column.
