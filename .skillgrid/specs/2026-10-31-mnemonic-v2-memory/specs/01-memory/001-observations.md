# 001 — Observations (the core fact row + save pipeline)

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `002-schema` (§2.2), `003-intelligence`, `004-transport-mcp`
**Reference:** Engram `internal/store/store.go` (`AddObservation`, `insertObservationTx`,
`AddObservationParams`), replay-safe saves, the save-conflict relevance gate; mnemonic v1
two-phase async commit.

---

## 1. Capability

The observation is the **unit of memory**. This spec defines the row (in `002-schema`
§2.2) and — the load-bearing part — the **save pipeline**: how a `mem_save` becomes a
durable, deduped, synced, wiki-notified fact in one atomic local transaction.

## 2. The save pipeline (the contract)

`mem_save` / `POST /observations` / CLI `mem save` all funnel into
`Store.AddObservation(ctx, AddObservationParams)`. The pipeline, in order:

```
1. VALIDATE      title, type, content non-empty; scope ∈ {project,personal,global};
                 type ∈ vocabulary; topic_key format (if given) kebab-case ≤2 levels.
2. RESOLVE       project: explicit > session > cwd/config (INV-7: don't invent).
                 session: must be a live session for this project.
3. REDACT        strip <private>…</private> (config `memory.private_tag`);
                 truncate content to `memory.content_max_bytes` with
                 TruncationMetadata (Engram).
4. PRE-FILTER    conflict candidates: OR-based FTS on title within project+scope,
                 then the deterministic relevance gate (003-facts-relations §3).
                 Candidate set is bounded (≤ per-source limit).  [floor: lexical only]
5. DEDUP         LLM Classify(newContent, candidates) → DedupDecision.
                 [degraded: hash dedup — normalized_hash+project+scope+type+title]
                 verdict: add | update(CandidateID) | delete(CandidateID) | noop
6. UPSERT/INSERT (a) topic_key set & matches project+scope+topic_key → UPDATE in place,
                     revision_count++, snapshot pre-state into observation_versions.
                 (b) no match → INSERT new observation (sync_id minted).
                 (c) DEDUP verdict update/delete → merge into CandidateID instead.
                 (d) compute review_after (type→months, 002-schema §2.2.2) and
                     importance (on-the-fly formula, 002-schema §2.2.3) for the row.
7. ENQUEUE       if sync enabled & project enrolled: enqueue an upsert mutation
                 (sync journal). Order: upsert before any related deletes.
8. COMMIT        one local transaction: observation + FTS + versions + sync mutation.
                 Any failure rolls back all of it.
9. RESULT        return {id, sync_id, revision_count, dedup: {verdict, reason,
                 merged_into}, candidates[], judgment_required}.
10. EVENT        emit SaveEvent (004-transport-mcp §5) to the event bus.
```

**Steps 1–8 are synchronous and local.** Step 9 (the wiki handoff) is **asynchronous**
with respect to the save's return — the save returns before the wiki consumer runs
(mnemonic v1's two-phase commit idea, but the async phase is the *wiki*, not the store).

## 3. The write queue

MCP writes are serialized through a **shared memory write queue** (Engram
`internal/mcp/write_queue.go`). `mem_save`, `mem_update`, `mem_judge_many`, and
`mem_consolidate` are all queued writes. This is what makes the transaction + sync
journal atomic from the agent's perspective even under concurrent tool calls.

## 4. Replay-safe saves (idempotency)

`AddObservationParams.OperationID` optionally binds a save to a durable local result in
`observation_save_operations` (Engram):

- Fingerprint = `v1:` SHA-256 of length-prefixed, normalized request fields (incl.
  redacted/truncated content), computed **before** session ownership resolution.
- **Identical replay** → returns the committed id; dedup/upsert/sync run once.
- **Changed payload** → `ErrObservationOperationConflict` (409).
- **Deleted observation** → `ErrObservationOperationExpired` (410); the ledger row
  survives as a tombstone (`observation_id` nullable, `ON DELETE SET NULL`).
- **No operation id** → existing behavior, no ledger row.

This is what lets a plugin retry a `mem_save` after a flaky transport without
double-writing.

## 5. The save event (the eager wiki handoff)

After commit, the save emits a `SaveEvent` (defined in `004-transport-mcp` §5). The
contract on the **memory side**:

- The event carries `sync_id, project, scope, type, topic_key, entity, importance,
  degraded` — enough for the wiki to do **entity-targeted** regeneration.
- The memory function **does not import** the wiki package. The event bus is the
  boundary (dependency inversion: memory emits, wiki consumes).
- **Failure isolation:** if the wiki consumer is absent/errors, the save still
  succeeded. The event is logged + queued for wiki-side retry. A wiki failure never
  fails a save.
- **Eager, but bounded:** the wiki's regeneration is incremental + entity-targeted by
  the wiki's contract (OPEN-11). The memory side only guarantees the event is emitted
  exactly once per committed save.

## 6. Updates and deletes

- **`mem_update`** (partial, by id): mandatory `expected_project` ownership assertion,
  validated *inside the same transaction* as the mutation (Engram's
  `UpdateObservationForProject`). A content change snapshots the pre-state into
  `observation_versions` first.
- **`mem_delete`** (by id): soft-delete (`deleted_at`) by default; `--hard` for
  physical. Mandatory `expected_project`. Soft-deleted rows are excluded from
  search/context/recent/timeline but their ids remain for audit.
- **Ownership assertion is a consistency check, not authn/authz** (Engram). The stored
  owner fallback (`mem_get_observation`) is read-only.

## 7. Invariants

INV-1 (provenance column set at insert, immutable), INV-6 (sync_id minted at insert),
INV-7 (agent curates), INV-4 (commit is local; sync journal is additive).

## 8. Open items

None.
