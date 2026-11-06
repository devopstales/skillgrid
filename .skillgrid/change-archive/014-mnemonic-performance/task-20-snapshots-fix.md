# 014 Step 20 — Fix: review findings for `20-snapshots`

**Verdict addressed:** NEEDS FIXES (1 critical + 2 major + 2 minor + 3 nits)
**Fix commit:** `ba49344` `fix(mnemonic): complete snapshot columns and prove row-lock blocking`
**Files changed:** `internal/mnemonic/memory/snapshots.go`, `snapshots_test.go`, `service.go` (no migration change — the 4 columns already exist on `observations` from 005/008).

---

## Per-finding

### F1 (CRITICAL) — 4 observations columns missing from SnapshotRow

Added the 4 live columns to every layer of the snapshot round-trip:
- `SnapshotRow` struct (new fields quoted below)
- `snapshotCols` SELECT list
- the `rows.Scan` argument list in `readSnapshotRows`
- `restoreUpsertSQL` (INSERT column list, VALUES placeholders 33→35, and `ON CONFLICT DO UPDATE SET`)
- `restoreRowDefault` argument list (note: `embedding` is passed through `blobOrNULL` so an empty/nil BLOB round-trips as SQL NULL, matching `SetEmbedding`'s `blobOrNULL` semantics)

New `SnapshotRow` fields:

```go
	ReviewAfter      *string
	Embedding        []byte
	EmbeddingModel   *string
	EmbeddingCreated *string
```

`snapshotCols` tail is now: `..., graph_ref, deleted_at, review_after, embedding, embedding_model, embedding_created_at` (35 columns = the full observations schema: 001 base 13 + 22 ALTERs).

New RED test `TestSnapshotPreservesEmbeddingsAndReviewAfter` (quoted in full below) seeds an embedding + a `review_after`, snapshots, modifies content AND clears the embedding + review, restores, and asserts the embedding triplet + review survive the round trip.

### F2 (MAJOR) — RestoreSnapshot used BEGIN DEFERRED, not BEGIN IMMEDIATE

The store's primary handle is opened with a bare DSN (`store.go:174` `sql.Open("sqlite", dbPath)`), so `BeginTx` issued `BEGIN DEFERRED`. The modernc driver only honors the DSN `_txlock` begin-mode (verified `sqlite.go:187-192`, `tx.go:23-24`), and there is no per-`BeginTx` option to force immediate, so the transaction is opened explicitly and driven on the same pooled connection (`MaxOpenConns(1)` → every `ExecContext` lands on the in-transaction connection):

```go
	// BEGIN IMMEDIATE (014 step 20.2): the restore acquires the write lock at
	// BEGIN, so a concurrent writer blocks until the restore commits instead
	// of interleaving with the per-row upserts. The store's primary handle is
	// opened with a bare DSN (BEGIN DEFERRED), so the transaction is opened
	// explicitly here and driven through the same pooled connection (the
	// handle is MaxOpenConns(1), so every ExecContext below lands on this
	// in-transaction connection). A failure anywhere rolls the whole restore
	// back — committed atomically, never half-restored.
	if _, err := s.store.DB.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return fmt.Errorf("begin immediate restore: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			if _, rbErr := s.store.DB.ExecContext(ctx, `ROLLBACK`); rbErr != nil {
				slog.Warn("snapshot restore rollback failed", "project", s.projectID, "err", rbErr)
			}
		}
	}()
```

The per-row upserts, orphan sweep, and FTS rebuild now run on that explicit transaction; the final `COMMIT` and the deferred `ROLLBACK` are direct statements. A `txHandle` interface (`ExecContext`) was introduced so the `restoreRowFn` seam and `sweepOrphanRows` accept the pooled `*sql.DB` (which satisfies it) instead of a `*sql.Tx`. Atomicity is unchanged and still proven by `TestSnapshotRestoreAtomic` (the swapped failing `restoreRowFn` now returns `sql.ErrConnDone` and the whole restore rolls back — passes).

### F3 (MAJOR) — concurrency test didn't prove "second blocks until first completes"

`UpdateRow` now returns the time its `BEGIN IMMEDIATE` returned (the write-lock acquisition moment): `func (s *Service) UpdateRow(...) (int64, time.Time, error)`. The test drives both writers through `BEGIN IMMEDIATE` handles (two independent row-lock DSN connections to the same file) and asserts the non-overlap property. The dead `start` channel is removed. A `rowLockHold` seam (test-only, on `*Service`) lets writer A hold the write lock until writer B is already waiting on its `BEGIN`, so the microsecond-UPDATE serialization window is visible to the timestamps (without it both writes can finish before the second writer even starts, and the assertion would be vacuous).

The new blocking assertion (quoted in full below):

```go
	// The second write blocked until the first completed: the LATER-acquired
	// writer's BEGIN IMMEDIATE returned at or after the earlier writer's
	// COMMIT. If the writes overlapped (no serialization), the later BEGIN
	// would have returned before the earlier writer finished.
	first, second := a, b
	if b.acq.Before(a.acq) {
		first, second = b, a
	}
	if second.acq.Before(first.commit) {
		t.Fatalf("writers overlapped (no blocking): first committed %s, second acquired the write lock %s (earlier) — the second write did not block until the first completed",
			first.commit.Format(time.RFC3339Nano), second.acq.Format(time.RFC3339Nano))
	}
```

Verified the row-lock DSN really blocks: a throwaway probe with one handle holding `BEGIN IMMEDIATE` and the row-lock handle issuing an UPDATE blocked the full 5s `busy_timeout` before returning `SQLITE_BUSY`.

### F4 (MINOR) — UpdateRow is a test-only write path

Addressed as "re-scope": 20.2's requirement is the row-lock *mechanism* + a test that it serializes, which is now genuinely proven (F3). The production `Service.Update` hot path was left on the store handle (deferred begin + 10s busy_timeout, which is safe but not `BEGIN IMMEDIATE`); routing it through `UpdateRow` would change the update path's transactional semantics (version-append, dedup-hash recompute) beyond this step's scope and is a larger change. Noted as a follow-up rather than a defect — the mechanism is provided and verified on a real two-connection WAL contention.

### F5 (MINOR) — pruneSnapshots indirection + swallowed error

Dropped the one-line `pruneSnapshots` wrapper (now calls `PruneSnapshots` directly) and logged the best-effort auto-prune failure so a storage-bloat guard cannot fail silently:

```go
	if perr := s.PruneSnapshots(ctx, s.snapshotRetention()); perr != nil {
		slog.Warn("snapshot auto-prune failed", "project", s.projectID, "keep", s.snapshotRetention(), "err", perr)
	}
```

### F6 (NIT) — stale %26 comment on rowLockDsnSuffix

Rewrote the comment: the suffix is a single-parameter DSN (`?_txlock=immediate`), so no URL-encoding is needed; the `%26` note now only explains the multi-parameter case. `busy_timeout` is documented as a PRAGMA, not a DSN parameter.

### F7 (NIT) — SnapshotRow gofmt misalignment

Re-aligned the struct (the `ImportanceScore *float64` field had broken the gofmt alignment group). `gofmt -l` is now clean on `snapshots.go` and `snapshots_test.go`. (`service.go` has pre-existing gofmt drift unrelated to this change — present on base `d31bdd8` — left untouched; the fix only adds the additive `rowLockHold` field.)

### F8 (NIT) — single commit vs per-task commits

The fix is a single conventional commit covering all the review fixes (the brief's per-task commit framing was about the original implementation, already merged in `d31bdd8`).

---

## New embedding-preservation test (F1)

```go
// TestSnapshotPreservesEmbeddingsAndReviewAfter covers the column-completeness
// gap the point-in-time test cannot see (F1): a snapshot must round-trip the
// embedding triplet (embedding BLOB, embedding_model, embedding_created_at)
// AND the review-cycle column (review_after) — a restore that drops any of
// them silently zeroes live data on every restore.
func TestSnapshotPreservesEmbeddingsAndReviewAfter(t *testing.T) {
	_, svc := newTestStore(t, "snapshotembed")
	ctx := context.Background()
	sid := newSession(t, svc)
	id, err := svc.Save(ctx, SaveInput{
		Title:     "embedded baseline",
		Type:      "decision",
		Content:   "baseline content for embedding round-trip",
		SessionID: sid,
		Scope:     "project",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// Seed an embedding (BLOB + model + created_at) and a review schedule.
	emb := EncodeVector(Vector{Data: []float32{0.1, -0.2, 0.3, 1.0}})
	if err := svc.SetEmbedding(ctx, id, emb, "test-embedder/v1"); err != nil {
		t.Fatalf("set embedding: %v", err)
	}
	if err := svc.SetReviewAfter(ctx, id, "2027-01-15 12:00:00"); err != nil {
		t.Fatalf("set review_after: %v", err)
	}
	var preEmb []byte
	var preModel, preEmbCreated, preReview string
	if err := svc.store.DB.QueryRowContext(ctx,
		`SELECT embedding, embedding_model, embedding_created_at, review_after FROM observations WHERE id = ?`, id,
	).Scan(&preEmb, &preModel, &preEmbCreated, &preReview); err != nil {
		t.Fatalf("read pre-snapshot embedding state: %v", err)
	}
	if len(preEmb) == 0 || preModel == "" || preEmbCreated == "" || preReview == "" {
		t.Fatalf("pre-snapshot state incomplete: embLen=%d model=%q created=%q review=%q",
			len(preEmb), preModel, preEmbCreated, preReview)
	}

	// Capture the point-in-time state WITH the embedding + review schedule.
	snapID, err := svc.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	// Modify content and clear the embedding + review schedule (the restore
	// must bring the captured values back — a dropped column restores as NULL
	// and the post-restore values would be empty).
	if err := svc.Update(ctx, id, UpdateInput{Content: "modified after snapshot"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := svc.SetEmbedding(ctx, id, nil, ""); err != nil {
		t.Fatalf("clear embedding: %v", err)
	}
	if err := svc.SetReviewAfter(ctx, id, ""); err != nil {
		t.Fatalf("clear review_after: %v", err)
	}

	// Restore to the captured state.
	if err := svc.RestoreSnapshot(ctx, snapID); err != nil {
		t.Fatalf("restore: %v", err)
	}

	// The embedding triplet and the review schedule survive the round trip.
	var postEmb []byte
	var postModel, postEmbCreated, postReview string
	if err := svc.store.DB.QueryRowContext(ctx,
		`SELECT embedding, embedding_model, embedding_created_at, review_after FROM observations WHERE id = ?`, id,
	).Scan(&postEmb, &postModel, &postEmbCreated, &postReview); err != nil {
		t.Fatalf("read post-restore embedding state: %v", err)
	}
	if !bytes.Equal(postEmb, preEmb) {
		t.Fatalf("restored embedding mismatch: got %d bytes, want %d bytes (zeroed on restore)", len(postEmb), len(preEmb))
	}
	if postModel != preModel {
		t.Fatalf("restored embedding_model=%q want %q (zeroed on restore)", postModel, preModel)
	}
	if postEmbCreated != preEmbCreated {
		t.Fatalf("restored embedding_created_at=%q want %q (zeroed on restore)", postEmbCreated, preEmbCreated)
	}
	if postReview != preReview {
		t.Fatalf("restored review_after=%q want %q (zeroed on restore)", postReview, preReview)
	}
}
```

## RED → GREEN evidence

- **F1 RED:** with the 4 columns temporarily removed from `restoreUpsertSQL`, `TestSnapshotPreservesEmbeddingsAndReviewAfter` failed:
  `restored embedding mismatch: got 0 bytes, want 16 bytes (zeroed on restore)`. With the fix restored it passes.
- **F3 RED:** before `UpdateRow` returned the acquisition time and both writers used `BEGIN IMMEDIATE` handles, the assertion failed with overlapping intervals (`second acquired the write lock ... (earlier)`). After the fix it passes, and was run 5× consecutively with no flake.
- **F2:** `TestSnapshotRestoreAtomic` (atomicity via the `restoreRowFn` seam) still passes with the explicit `BEGIN IMMEDIATE` + deferred `ROLLBACK`; the probe confirmed the row-lock DSN blocks a concurrent writer for the full `busy_timeout`.

## Full relevant suite

`go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1 -race`

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	191.130s
```

Step-20 subset (6 tests) all PASS with `-race`:

```
=== RUN   TestSnapshotCreatePointInTime            --- PASS
=== RUN   TestSnapshotPreservesEmbeddingsAndReviewAfter --- PASS
=== RUN   TestSnapshotRestoreAtomic                --- PASS
=== RUN   TestRowLevelLockingConcurrentWrites      --- PASS
=== RUN   TestSnapshotAutoPrune                    --- PASS
=== RUN   TestListSnapshots                        --- PASS
```

CLI snapshot tests: `go test ./cmd/skillgrid/ -run 'TestMemSnapshot' -count=1` → `ok ... 25.382s` (2/2).

## Baselines

`go test ./skillgrid-cli/internal/mnemonic/store/... ./internal/mnemonic/service/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... -count=1`

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store	6.012s
--- FAIL: TestReindexStructuralIsEmbedderFree (1.01s)
    reindex_structural_test.go:107: count gamma: sql: database is closed
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	54.110s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex	18.949s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route	4.094s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected	8.850s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community	6.285s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg	2.178s
```

`TestReindexStructuralIsEmbedderFree` is **pre-existing**: it fails identically on the unmodified base `d31bdd8` (verified via `git stash`), so it is not a regression from this fix.

## Commits

- `ba49344` `fix(mnemonic): complete snapshot columns and prove row-lock blocking` (3 files: `memory/snapshots.go`, `memory/snapshots_test.go`, `memory/service.go`)

## Concerns

- **F2 mechanism:** the explicit `BEGIN IMMEDIATE` / `COMMIT` / `ROLLBACK` statements are correct for the single-connection pool (`MaxOpenConns(1)`), but they bypass `database/sql`'s `*sql.Tx` bookkeeping. This is safe here because the handle is pinned to one connection and the restore is the only in-flight transaction on it; it is more fragile than a `*sql.Tx` if the pool is ever allowed >1 connection. The `_txlock=immediate` DSN lever on the primary handle would be the more idiomatic long-term fix but requires a `store.Open` change shared by every project, so it was out of scope for this fix.
- **F4 scope:** the production `Service.Update` hot path is still on the deferred-begin store handle (safe under the 10s busy_timeout, but not `BEGIN IMMEDIATE`). The row-lock mechanism is verified, not yet on the update hot path.
- The concurrency test relies on the `rowLockHold` test seam to make the sub-millisecond serialization window observable; that is a deliberate, documented trade-off (the alternative — asserting on two independent microsecond timestamps — is flake-prone).
