# Step 01 Re-Review (fix round) — `store-pooling`, scope F1 + F2

**Reviewer**: Fresh reviewer (scoped re-review)
**Scope**: Only F1 (retry loop untested) and F2 (`isWALBusy` wrong error matching). F3/F4/F5 not in scope.
**Fix diff**: `review-e2ff708..8c20c92.diff` (1 commit `8c20c92`)
**Date**: 2026-09-11

## Verdict: `PASS`

Both major findings are resolved with runtime-verified fixes. `go vet` clean; full
`internal/mnemonic/store` package passes; the three relevant tests pass 5/5 with
`-count=5`.

---

## F2 resolved? **Yes**

`isWALBusy` now prefers the driver's typed error and only falls back to text. Final
implementation (store.go:158-171):

```go
func isWALBusy(err error) bool {
	if err == nil {
		return false
	}
	var typed *sqlite.Error
	if errors.As(err, &typed) {
		return typed.Code() == sqlite3.SQLITE_BUSY
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLITE_BUSY") ||
		strings.Contains(msg, "The database file is locked") ||
		strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked")
}
```

- **Typed driver error**: `errors.As(err, &typed)` → `typed.Code() == sqlite3.SQLITE_BUSY`.
  Verified the import resolves: `sqlite3 "modernc.org/sqlite/lib"` exports
  `SQLITE_BUSY = 5` (lib/sqlite_darwin_arm64.go:2516) and `modernc.org/sqlite`
  defines `Error.Code() int` (error.go:21). `openDatabase` wraps with `%w`
  (`apply %s: %w`, store.go:189), so the `*sqlite.Error` survives `errors.As`.
- **Text fallback present + sensible**: retains the real rendered busy message
  `"The database file is locked"` (exactly matches error.go:28
  `"The database file is locked (SQLITE_BUSY)"`) plus legacy `"database is locked"`.
  The prior dead `"SQLITE_BUSY"` substring is kept as belt-and-suspenders.
- **Comment now accurate**: no longer claims the wrong format; it states the typed
  path is preferred and cites v1.45.0 error.go/conn.go as verified.
- **Classification correct**: `TestIsWALBusyClassification` proves the busy path is
  live — it hard-fails unless `errors.As(busy,&typed)` yields `Code()==5`, then
  asserts `isWALBusy(busy)==true`. By that same typed branch, a permanent error
  (SQLITE_CANTOPEN / SQLITE_CORRUPT) has a different code and is classified
  non-busy; a non-`*sqlite.Error` (the "unrelated" `no such table: foo` case) also
  returns false. The precondition check means a CANTOPEN/CORRUPT would fail the
  test if it wrongly surfaced as code 5.

## F1 resolved? **Yes**

`TestStoreOpenWALRetryLoop` (store_pooling_test.go:171-236) genuinely exercises
`openWithWALRetry`:

- **Goes through `Open()`**: the goroutine calls `Open(dir, "retryproj")`, whose
  cold path reaches `openWithWALRetry` (store.go:119). It is a cold open — the cache
  is disabled via `t.Setenv(envCacheDisable, "1")` (store.go:114 skips
  `acquireCached`), so no pooled handle is reused.
- **Lock held during open**: a separate `*sql.DB` holder opens the same path, runs
  `CREATE TABLE`, then `Begin()` + `INSERT` to take the WAL write lock, and the lock
  is held until 80ms. The first `openDatabase` attempt's `PRAGMA journal_mode=WAL`
  hits that lock (journal-mode transition under lock) and returns a typed SQLITE_BUSY.
- **Asserts success after release**: `select { case s := <-opened: ... }` — on a
  non-retrying open the busy error would propagate to `openErr` and the test would
  `t.Fatalf`.
- **Asserts a backoff occurred**: `elapsed < 50*time.Millisecond` → fatal. Observed
  elapsed across 5 runs: 394ms / 250ms / 248ms / 247ms / 251ms — comfortably above
  the floor. The 2nd attempt succeeds (its `journal_mode=WAL` is a no-op on a
  already-WAL db, so it isn't blocked by the 80ms release), consistent with the
  implementer's "2 BUSY attempts then success" note.
- **Existing `TestStoreOpenWALRetry` retained** for the busy_timeout=10000 path
  (unchanged, still passes).

### Residual skeptic note (why it still passes)
The 50ms floor is *necessary but not sufficient* proof of a retry: a single attempt
whose `busy_timeout` (10s) absorbed the lock held ~80ms would also clear the floor.
However that gap is closed by the companion `TestIsWALBusyClassification`, which
proves `isWALBusy` returns true on the real wrapped busy error — i.e. that
`openWithWALRetry`'s break condition (`if !isWALBusy(err) { break }`) is *not* hit
on the first attempt, so the retry is structurally guaranteed. Together the two
tests cover F1. Not a blocking concern; flagged only for completeness.

## Any new issues introduced by the fix

- **N/A — no blocking issues.**
  - Import change `_ "modernc.org/sqlite"` → `"modernc.org/sqlite"` (value import) is
    correct: the blank import only ran registration; the driver self-registers in its
    init, and the new named import is required for `*sqlite.Error`. Compiles, vet
    clean, no driver double-registration.
  - `sqlite3 "modernc.org/sqlite/lib"` is a legitimate public package (exports
    `SQLITE_BUSY`). Slightly heavier than re-deriving the constant, but idiomatic
    and correct.
  - No regression: full package `go test -count=1 ./internal/mnemonic/store/` passes;
    the three targeted tests pass 5/5.
  - Timing flakiness of `TestStoreOpenWALRetryLoop`: the floor is a 50ms lower bound
    against an observed ~250ms, so it is generous; the 5s upper timeout is also
    generous. Not flaky in 5 runs.

## Approved to merge: **Yes** (for F1 + F2)

F1 and F2 are both properly fixed and runtime-verified. (F3/F4/F5 were out of scope
for this re-review.)
