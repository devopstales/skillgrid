# 014 Step 20 — Review: Multi-version snapshots + transaction locking

**Verdict: NEEDS FIXES**

Commit `d31bdd8` (8 files: mem.go, mem_snapshot_test.go, config/load.go, memory/service.go, memory/snapshots.go, memory/snapshots_test.go, service/service.go, store/migrations/030_snapshots.sql). Build passes; all 7 step-20 tests pass with `-race` (5 memory + 2 CLI). The feature works end-to-end and the restore is genuinely atomic — but the snapshot is **incomplete**: 4 live observations columns are not serialized, so a restore silently zeroes them.

---

## Findings

### F1 — CRITICAL — Snapshot serialization is incomplete (4 observations columns dropped)

**`internal/mnemonic/memory/snapshots.go:93-99`** (`snapshotCols`) and `:27-58` (`SnapshotRow`).

`SnapshotRow` has **31 fields**, but the `observations` table has **35 columns**. The brief's 20.1 requirement is that a snapshot be a *point-in-time view* — "a restore would lose that column's data" if a column is missing. Four columns are missing from both `SnapshotRow` and `snapshotCols`:

| Missing column | Added by | Live usage (proves it is persisted data, not vestigial) |
|---|---|---|
| `review_after` (TEXT) | `005_review_cycle.sql:5` | `memory/review_cycle.go:36` (SELECT for review-due), `:82`,`:106` (UPDATE) |
| `embedding` (BLOB) | `008_obs_lifecycle.sql:54` | `memory/search_embed.go:25` (SetEmbedding), `retrieval.go:474` (obsEmbedding SELECT) |
| `embedding_model` (TEXT) | `008_obs_lifecycle.sql:55` | `memory/search_embed.go:26` (SetEmbedding) |
| `embedding_created_at` (TEXT) | `008_obs_lifecycle.sql:56` | `memory/search_embed.go:27` (SetEmbedding), `008` partial index `:83` |

Reconstructed full observations schema (001 base 13 + 22 ALTERs = 35 columns). `snapshotCols` (the read AND the restore write share it) is:

```
id, session_id, type, title, content, project, scope, topic_key,
source, normalized_hash, revision_count, prompt_id, created_at, updated_at,
pinned, duplicate_count, last_seen_at, expires_at, tool_name,
owner, visibility, status, retrieval_usage,
importance_score, recency_decay, maturity_tier, provenance, memory_type,
graph_ref, deleted_at                     -- 31 columns
```

**Consequence:** every restore (and therefore every snapshot→modify→restore cycle) silently zeroes `review_after`, `embedding`, `embedding_model`, `embedding_created_at` for *all* project rows, because `restoreUpsertSQL` (`:221-262`) only writes the 31 captured columns and the orphan sweep does not reset them. The 5 memory tests do not catch this because none of them set an embedding or a `review_after` (they use bare `SaveInput`). The implementer's report claim — "serializes the current observations state … (31 fields, pointer types for NULLs)" and "a snapshot is a faithful point-in-time view" — is **false**: 4 of 35 columns are dropped. For a project with embeddings (the hybrid retrieval path, `search_embed.go`), a restore destroys every vector and its provenance.

**Why it slipped past TDD:** RED/GREEN tests for 20.1 only exercise `Save`/`Update` (content + title), never `SetEmbedding` or review scheduling, so the omission is invisible to the suite.

**Fix:** add the 4 columns to `snapshotCols`, `SnapshotRow` (as pointer types: `ReviewAfter *string`, `Embedding []byte` (or base64 string), `EmbeddingModel *string`, `EmbeddingCreatedAt *string`), the `rows.Scan` list (`:170-176`), the `restoreUpsertSQL` column list + `DO UPDATE SET` (`:221-262`), and the `restoreRowDefault` arg list (`:204-216`). Then add a RED test: save an obs, `SetEmbedding`, set a `review_after`, `Snapshot`, `Restore`, and assert `embedding`/`embedding_model`/`embedding_created_at`/`review_after` round-trip. (Note: `embedding BLOB` scanned into `[]byte` round-trips cleanly under modernc; a base64 string field is the safer JSON-serialization choice — `json` cannot encode a raw `[]byte` as an opaque byte string, it base64-encodes it, which is actually fine, but be consistent.)

### F2 — MAJOR — RestoreSnapshot is not "BEGIN IMMEDIATE" (the brief's row-level-locking requirement)

**`internal/mnemonic/memory/snapshots.go:301`** — `tx, err := s.store.DB.BeginTx(ctx, nil)`.

20.2 requires writes to use `BEGIN IMMEDIATE` so concurrent writes serialize. The row-lock DSN suffix (`?_txlock=immediate`) is only applied to the *test's* second handle (`openRawSQLDB`, `snapshots_test.go:185`). `RestoreSnapshot` runs on the store's primary handle, whose DSN is bare (`store.go:174`, `sql.Open("sqlite", dbPath)` — no `_txlock`), so under modernc (`tx.go:23-24`: `if !opts.ReadOnly && c.beginMode != "" { sql = "begin "+c.beginMode }`) it issues **`BEGIN DEFERRED`**, not `BEGIN IMMEDIATE`. A concurrent write interleaving the restore's first write can still cause a `SQLITE_BUSY` (deferred promotes to a write lock at the first write, not at BEGIN). It is not *unsafe* (busy_timeout=10000 absorbs it), but the brief's "BEGIN IMMEDIATE transactions for writes" is not honored for the restore path. `Snapshot`'s capture (a single SELECT) is read-only and the insert into `snapshots` is a separate single-row write — also deferred.

**Fix (cheap):** have the service open the store handle with `?_txlock=immediate` (or call `BeginTx` with `&sql.TxOptions{}` after forcing immediate — but modernc only honors the DSN `beginMode`, so the DSN is the real lever). Minimal: pass the row-lock DSN suffix into `store.Open`/`openDatabase` for the primary handle. Alternatively, document that the restore is atomic under DEFERRED + 10s busy_timeout and relax the 20.2 wording. Either way, the current state under-claims the guarantee.

### F3 — MAJOR — The concurrency test does NOT prove "the second write blocks until the first completes"

**`internal/mnemonic/memory/snapshots_test.go:992-1059`** (`TestRowLevelLockingConcurrentWrites`).

Brief 20.2.a: "verify the second write blocks until the first completes; verify no data corruption." The test:
1. Launches two goroutines calling `UpdateRow` concurrently.
2. Asserts neither errors.
3. Asserts final content ∈ {writeA, writeB} (no mix).
4. Asserts a follow-up write succeeds within 3s (lock released).

It does **not** verify the *ordering/blocking* property at all — there are no timestamps, no barrier that confirms B's BEGIN blocked on A's lock, and no assertion that one finished before the other started. With two ~microsecond UPDATEs the "block" window is so small that the test would pass even with plain DEFERRED and no real serialization; it only proves "no corruption," which WAL atomicity would give even without BEGIN IMMEDIATE. The `start` channel (`:1015`,`:1031`) is `close(start)`'d but **never read by the goroutines** — it is dead code, not a release barrier (the goroutines start before `close` anyway). 

**Fix:** make each writer record `time.Now()` at BEGIN and at COMMIT, and assert `max(B.begin) > A.commit` XOR `max(A.begin) > B.commit` (i.e. the two [begin,commit] intervals do not overlap) — that is the actual "second blocks until first completes" property. Also remove or wire the dead `start` channel.

### F4 — MINOR — `UpdateRow` is a new write path that production does not use

**`internal/mnemonic/memory/snapshots.go:470-504`**.

The only caller of `UpdateRow` is the concurrency test. The production `mem_*`/update path goes through `Service.Update` (used by `TestSnapshotCreatePointInTime` at `:874-879`), which does **not** take a `db` argument and (per the diff) is not wrapped in `BEGIN IMMEDIATE`. So "row-level locking for concurrent writes" is demonstrated on a method nothing in the product calls. It is not a bug (the brief only asked for the mechanism + a test), but it overstates coverage: two concurrent production `Update` calls are still not routed through `UpdateRow`/immediate.

**Fix (cheap):** route `Service.Update`'s write through the same immediate transaction (or through `UpdateRow`), so the locking is real on the hot path. Or re-scope 20.2 to "the row-lock mechanism is provided and verified" rather than "concurrent writes are serialized."

### F5 — MINOR — `pruneSnapshots` / `pruneSnapshots` indirection + swallowed error

**`internal/mnemonic/memory/snapshots.go:150-153` and `:406-408`**.

`pruneSnapshots(ctx, keep)` is a one-line wrapper around `PruneSnapshots` that adds nothing, and the auto-prune error is swallowed (`_ = perr`) with a comment "log and move on" but **no log call** — a silent prune failure is invisible. Not a correctness issue (the snapshot is already durable), but it violates the project's "evidence before assertions" norm: a storage-bloat guard that can fail silently should at least `slog`/return the error to a sink.

**Fix:** drop the wrapper (call `PruneSnapshots` directly) and either log the error or surface it on the Service for the next op.

### F6 — NIT — `rowLockDsnSuffix` comment is stale

**`internal/mnemonic/memory/snapshots.go:450-457`**.

The comment says "The `%26` is the URL-encoded '&'", but `rowLockDsnSuffix = "?_txlock=immediate"` contains no `%26` — the comment describes a multi-param DSN that was never used. The report (concern #2) correctly explains the single-param choice, but the code comment references a `%26` that doesn't exist in the constant.

**Fix:** delete the `%26` sentence or rephrase.

### F7 — NIT — SnapshotRow field misalignment (gofmt)

**`internal/mnemonic/memory/snapshots.go:51-53`**.

```
ImportanceScore *float64
RecencyDecay   *float64
MaturityTier   *string
```
`ImportanceScore` breaks the gofmt-aligned block (its type starts a new alignment group). `go build` passes but `gofmt -l` flags the struct. Cosmetic.

**Fix:** run `gofmt -w` on the file.

### F8 — NIT — Commit-message framing vs the brief's per-task commits

The brief specified 4 per-task commit messages (`feat(mnemonic): point-in-time store snapshots`, `…row-level locking…`, `…auto-prune…`, `…mem snapshot CLI`). The implementer merged all into one `feat(mnemonic): multi-version snapshots with transaction locking` and documented the rationale. Acceptable; noting for the record.

---

## Evidence checked (the 9 checks)

1. **Snapshot serialization completeness — INCOMPLETE.** Counted `SnapshotRow` (31 fields, `:27-58`) against the reconstructed observations schema (35 columns: 001 base 13 + `ALTER TABLE observations ADD COLUMN` in 004, 005, 007, 008×7, 009, 017×4, 022, 025×3, 027, 028). **4 columns missing: `review_after`, `embedding`, `embedding_model`, `embedding_created_at`.** Confirmed each is *live* (read/written by production code, not vestigial): `review_cycle.go:36/82/106`, `search_embed.go:25-27`, `retrieval.go:474`, and `008_obs_lifecycle.sql:83` partial index. **Verdict: the snapshot does NOT cover all columns — F1 is the blocking finding.**
2. **RestoreSnapshot atomicity — STRONG.** `RestoreSnapshot` (`:271-342`) runs one `BeginTx`/`Commit` with a `defer` rollback on any error; the per-row upsert, the orphan sweep (`sweepOrphanRows`, `:359`), and the FTS rebuild are all inside the tx. `TestSnapshotRestoreAtomic` swaps a failing `restoreRowFn` seam and proves the store is left exactly pre-restore. PASS.
3. **state_hash integrity — PASS.** Hash = SHA-256 of the serialized BLOB (`:132`); on restore `:293` recomputes and errors on mismatch *before* touching observations. `TestSnapshotCreatePointInTime:869-871` asserts `sha256Hex(data) == state_hash`.
4. **Row-level locking — PARTIAL.** `BEGIN IMMEDIATE` is real on the row-lock handle (verified modernc `tx.go:23-24` issues `begin immediate`; `_txlock=immediate` is a valid value, `sqlite.go:187-193`); `busy_timeout=5000` is set (`rowLockTimeout`, `snapshots_test.go:191`). But (a) the restore uses the *primary* handle → DEFERRED, not IMMEDIATE (F2); (b) the concurrency test proves no-corruption but **not** the blocking/serialization property (F3); (c) `UpdateRow` is test-only, production `Update` is not routed through it (F4).
5. **snapshots migration — PASS.** `030_snapshots.sql` is `CREATE TABLE IF NOT EXISTS`, purely additive, columns exactly `id INTEGER PK AUTOINCREMENT, project_id TEXT NOT NULL, state_hash TEXT NOT NULL, data BLOB NOT NULL, created_at TIMESTAMP NOT NULL`. No ALTER/CREATE on observations. PASS.
6. **Snapshot retention — PASS.** `PruneSnapshots(ctx, keep)` (`:380`) deletes all but the `keep` most-recent by id (correct under AUTOINCREMENT monotonicity); auto-prune runs after each `Snapshot` (`:150`); `SetSnapshotRetention` configurable, default 10; wired via `config/load.go:403` → `service/service.go:388`. `TestSnapshotAutoPrune` verifies 20→5 and explicit `PruneSnapshots(2)`→2. PASS.
7. **CLI — PASS.** `mem snapshot create|restore|list` (`mem.go:54-123`) wired to `Snapshot`/`RestoreSnapshot`/`ListSnapshots`. Missing/bad id → `error: mem snapshot restore requires a snapshot id` (exit 2) / `error: invalid snapshot id` (exit 2). `TestMemSnapshotCLI` + `TestMemSnapshotCLIBadArgs` pass.
8. **Additive-ness / tool contract — PASS.** The diff's only deletions are `printMemUsage` help-text reflow. No `mem_*`/`code_*`/`web_*` MCP tool file is touched by the commit (8 files, none in `internal/mnemonic/mcp/`). `modernc.org/sqlite v1.45.0` stays; no CGO (modernc is pure Go). New table + new functions + new CLI subcommand + new config key, all additive.
9. **TDD discipline — PASS with a gap.** RED→GREEN is evidenced for all 4 sub-tasks (report + the tests exist and fail-before-implement). The tests verify (a) point-in-time snapshot+restore (5→6→5), (b) concurrent writes (no-corruption + lock-release), (c) the CLI. **Gap:** no test seeds an embedding or a `review_after` before a snapshot, which is exactly why F1's 4-column omission is invisible. TDD was disciplined about *what it tested*, but the test set is not a column-completeness oracle.

**Test/build evidence (run in this review, not just the report's):**
- `go build ./...` → clean.
- `go test ./internal/mnemonic/memory/ -race -run 'TestSnapshot|TestRowLevel|TestListSnapshots'` → 5/5 PASS.
- `go test ./cmd/skillgrid/ -run 'TestMemSnapshot'` → 2/2 PASS.

---

## Approved to merge: NO

F1 (4 dropped columns) is a data-loss-on-restore defect that the brief's own 20.1 scenario is meant to prevent, and it is invisible to the current suite. Fix F1 (add the 4 columns + a column-completeness test), F2 (route restore through IMMEDIATE), and F3 (make the concurrency test actually prove serialization) — F4/F5/F6/F7 can be folded in. Re-run the suite + `gofmt` after the fix.
