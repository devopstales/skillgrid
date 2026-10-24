# 014 Step 20 — Re-review: fix for `20-snapshots` (review round 2)

**Verdict: PASS**

Fix commit `ba49344` (`fix(mnemonic): complete snapshot columns and prove row-lock blocking`) resolves all 3 blocking findings (F1 CRITICAL, F2 MAJOR, F3 MAJOR) plus the F5 minor. The working tree matches the commit; the 6 step-20 tests pass with `-race`; `gofmt -l` is clean on both touched files.

All verification below was run against the working tree at `ba49344` (clean `git status`), not just the diff.

---

## F1 (CRITICAL) — resolved: YES

All 4 layers now carry the 4 missing columns (`review_after`, `embedding`, `embedding_model`, `embedding_created_at`), verified in the working tree, not just the struct:

| Layer | Location | Evidence |
|---|---|---|
| `SnapshotRow` struct | `snapshots.go:59-62` | `ReviewAfter *string`, `Embedding []byte`, `EmbeddingModel *string`, `EmbeddingCreated *string` |
| `snapshotCols` (shared read+restore) | `snapshots.go:104` | `... graph_ref, deleted_at, review_after, embedding, embedding_model, embedding_created_at` |
| `rows.Scan` list | `snapshots.go:181-182` | `&r.ReviewAfter, &r.Embedding, &r.EmbeddingModel, &r.EmbeddingCreated` |
| `restoreUpsertSQL` cols + VALUES + `DO UPDATE SET` | `snapshots.go:244-245`, `:280-283` | INSERT list extended; VALUES `?` count 31→35; `ON CONFLICT ... DO UPDATE SET review_after/embedding/embedding_model/embedding_created_at` |
| `restoreRowDefault` args | `snapshots.go:226-227` | `row.ReviewAfter, blobOrNULL(row.Embedding), row.EmbeddingModel, row.EmbeddingCreated` |

`snapshotCols` is now 35 columns = the full observations schema (001 base 13 + 22 ALTERs). The new RED test `TestSnapshotPreservesEmbeddingsAndReviewAfter` (`snapshots_test.go:398-477`) is real (not a stub): it `Save`s an obs, `SetEmbedding` (BLOB + `test-embedder/v1` + created_at) and `SetReviewAfter("2027-01-15 12:00:00")`, asserts the pre-snapshot state is non-empty, `Snapshot`s, then `Update`s content AND clears both the embedding and the review schedule, `RestoreSnapshot`s, and asserts the embedding triplet (`bytes.Equal`) + `embedding_model` + `embedding_created_at` + `review_after` all survive. This directly catches the "dropped column restores as NULL" defect. PASS under `-race` (2.14s).

## F2 (MAJOR) — resolved: YES

`RestoreSnapshot` now opens the transaction explicitly with `BEGIN IMMEDIATE`, the FIRST statement of the tx (before the per-row upserts, the orphan sweep, and the FTS rebuild) at `snapshots.go:330`:

```go
if _, err := s.store.DB.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
    return fmt.Errorf("begin immediate restore: %w", err)
}
```

followed by a `committed` flag + deferred `ROLLBACK` (`:333-340`), the upserts/sweep/FTS on `s.store.DB` (`:346`,`:355`,`:363`), and a final explicit `COMMIT` (`:367`). A new narrow `txHandle` interface (`:201`) replaces `*sql.Tx` in `restoreRowFn`/`restoreRowDefault`/`sweepOrphanRows` so the pooled `*sql.DB` (satisfying it) is the transaction target. The fix report documents the caveat: this relies on `MaxOpenConns(1)` (every `ExecContext` lands on the in-transaction connection) and bypasses `database/sql` `*sql.Tx` bookkeeping — safe here, more fragile if the pool ever exceeds 1 connection. The idiomatic long-term fix (DSN `_txlock=immediate` on the primary handle via `store.Open`) is noted as out of scope. This is an acceptable, documented trade-off, not a defect. `TestSnapshotRestoreAtomic` still passes, proving atomicity is preserved.

## F3 (MAJOR) — resolved: YES

`UpdateRow` now returns the time its `BEGIN IMMEDIATE` returned (`snapshots.go:318`, `acq := time.Now()` at `:531` — after `BeginTx`, before the UPDATE). The dead `start` channel is removed (no `start := make`/`close(start)` remains in the test). The test (`snapshots_test.go:272-399`) now:
- drives BOTH writers through `BEGIN IMMEDIATE` handles (`firstDB` and `secondDB`, both from `openRawSQLDB`, which appends `rowLockDsnSuffix` = `_txlock=immediate` + `busy_timeout=5000` — `snapshots_test.go:523-540`), so the contention is real, not vacuous;
- uses a `rowLockHold` test-only seam (on `*Service`, `service.go:120-14`; nil in production) to make writer A hold the write lock until writer B is already waiting on its `BEGIN`, making the sub-millisecond serialization window observable;
- asserts the blocking (non-overlap) property at `snapshots_test.go:376-379`:

```go
first, second := a, b
if b.acq.Before(a.acq) { first, second = b, a }
if second.acq.Before(first.commit) {
    t.Fatalf("writers overlapped (no blocking): first committed %s, second acquired the write lock %s (earlier) ...",
        first.commit.Format(time.RFC3339Nano), second.acq.Format(time.RFC3339Nano))
}
```

i.e. the LATER-acquired writer's `BEGIN IMMEDIATE` returned at or after the earlier writer's COMMIT — exactly the "second blocks until first completes" property. This is stronger than the original no-corruption-only check (it would fail under plain DEFERRED / no serialization). PASS under `-race` (2.28s). The `rowLockHold` seam is a deliberate, documented trade-off for flake resistance.

## No regressions

- `go test ./internal/mnemonic/memory/ -run 'TestSnapshot|TestRowLevel|TestListSnapshots' -count=1 -race` → all 6 PASS:
  `TestSnapshotCreatePointInTime`, `TestSnapshotPreservesEmbeddingsAndReviewAfter`, `TestSnapshotRestoreAtomic`, `TestRowLevelLockingConcurrentWrites`, `TestSnapshotAutoPrune`, `TestListSnapshots`.
- The 5 original step-20 tests (the 4 above minus the new one) still pass; the new F1 test was added. The fix modified `snapshots.go` but no original test was broken.
- `gofmt -l snapshots.go snapshots_test.go` → clean.
- The one baseline failure `TestReindexStructuralIsEmbedderFree` (`service` package) is pre-existing (fails identically on base `d31bdd8` per the fix report, verified via `git stash`) and outside the step-20 scope.

## `-race` clean

The concurrency test (`TestRowLevelLockingConcurrentWrites`) passes under `-race`. The `coordMu`/`coordCond` coordination and the two independent row-lock connections are properly synchronized; no data races reported.

## Remaining findings (non-blocking)

- **F4 (MINOR, re-scoped):** the production `Service.Update` hot path is still on the deferred-begin store handle (safe under the 10s busy_timeout, not `BEGIN IMMEDIATE`); the row-lock mechanism is verified on `UpdateRow`, not yet on the update hot path. The fix correctly re-scopes 20.2 to "mechanism provided + verified." Acceptable; candidate for a follow-up.
- **F6 (NIT) — resolved:** the stale `%26` comment on `rowLockDsnSuffix` was rewritten (single-param DSN, `busy_timeout` documented as a PRAGMA) — folded in.
- **F7 (NIT) — resolved:** `SnapshotRow` realigned; `gofmt` clean.
- **F5 (MINOR) — resolved:** the `pruneSnapshots` wrapper was dropped (calls `PruneSnapshots` directly) and the auto-prune failure is now logged via `slog.Warn` (`snapshots.go:120-121`) instead of silently swallowed.
- **F2 long-term (not a defect):** the explicit `BEGIN IMMEDIATE`/`COMMIT`/`ROLLBACK` on the pooled `*sql.DB` is more fragile than a `*sql.Tx` if `MaxOpenConns` ever exceeds 1; the DSN-based fix on the primary handle is the cleaner path but out of scope. Documented in the fix report.

## Approved to merge: YES

All 3 blocking findings (F1, F2, F3) are resolved with real, non-stub evidence; the 6 step-20 tests pass with `-race`; gofmt is clean; no regressions in the touched files. The remaining items are a re-scoped minor (F4) and documented long-term notes, none of which block this change.
