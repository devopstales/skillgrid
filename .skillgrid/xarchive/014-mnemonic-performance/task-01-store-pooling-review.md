# Step 01 Review — `store-pooling`

**Reviewer**: Fresh reviewer, no prior context
**Scope**: Cached store handles + WAL lock retry + 020 migration
**Date**: 2026-09-11

## Verdict: `NEEDS FIXES`

## Findings

### F1 — WAL retry loop is not exercised by any test (major)

**File**: `store_pooling_test.go:81-156` (`TestStoreOpenWALRetry`)
**What's wrong**: The test verifies that `busy_timeout=10000` works (a separate mechanism from the retry loop), but does NOT verify that `openWithWALRetry` actually retries with backoff. The contended write goes through `db2` (a raw SQL connection opened via `rawSQLDB`), NOT through `Open()`. So `openWithWALRetry` is never in the path for the contended write.

**Evidence**:
```go
// The contended write goes through db2, not Open():
db2 := rawSQLDB(t, dir, "walproj")  // line 118
_, err = db2.Exec(`INSERT INTO walprobe (v) VALUES ('contended')`)  // line 126
```

`openWithWALRetry` is only called from `Open()` (store.go:117). The test calls `Open()` once at the start (line 85) with no lock held, so it succeeds on attempt 0. The subsequent contended write goes through `db2.Exec(...)`, which uses `busy_timeout=10000` but NOT the retry loop.

**Fix**: Add a test that calls `Open()` while a lock is held. For example:
1. Open a store and hold a write transaction on it.
2. Call `Open()` again for the same path in a goroutine (this will go through `openWithWALRetry`).
3. Verify that the second `Open()` retries with backoff and eventually succeeds after the lock is released.

Alternatively, extract the retry logic into a testable function and test it directly.

### F2 — `isWALBusy` does not match the actual modernc/sqlite error format (major)

**File**: `store.go:155-163`
**What's wrong**: The function matches `"database is locked"`, `"SQLITE_BUSY"`, and `"database table is locked"`, but the actual modernc/sqlite error format is `"The database file is locked (SQLITE_BUSY)"` (verified from the module cache at `modernc.org/sqlite@v1.45.0/error.go:28`). None of the three matched strings appear in this format.

**Evidence** (from `modernc.org/sqlite@v1.45.0/error.go:28`):
```go
sqlite3.SQLITE_BUSY: "The database file is locked (SQLITE_BUSY)",
```

The actual error string contains:
- `"The database file is locked"` (not `"database is locked"`)
- `"(SQLITE_BUSY)"` (not `"SQLITE_BUSY"`)
- No `"database table is locked"` at all

**Fix**: Update `isWALBusy` to match the actual format:
```go
func isWALBusy(err error) bool {
    if err == nil {
        return false
    }
    msg := err.Error()
    return strings.Contains(msg, "database file is locked") ||
        strings.Contains(msg, "SQLITE_BUSY")
}
```

Or, better, use the typed error:
```go
func isWALBusy(err error) bool {
    var sqliteErr *sqlite.Error
    if errors.As(err, &sqliteErr) {
        return sqliteErr.Code() == sqlite3.SQLITE_BUSY
    }
    return false
}
```

### F3 — Migration file name does not match the brief (minor)

**File**: `store/migrations/020_ttl_extraction.sql`
**What's wrong**: The brief specifies `014_ttl_extraction.sql` (line 25, 48), but the implementer created `020_ttl_extraction.sql`. The report explains this (line 22: "the brief's `014_` prefix is the change number; the file sorts after the existing `019_session_relay.sql` so it applies last"), but the brief is explicit about the filename.

**Evidence** (brief line 25):
```
- `internal/mnemonic/store/migrations/014_ttl_extraction.sql` — CREATE: TTL config table, extraction metadata
```

**Fix**: Either rename the file to `014_ttl_extraction.sql` (and update the test accordingly), or update the brief to reflect the actual filename. The current naming is defensible (it sorts after 019), but it deviates from the explicit requirement.

### F4 — Cache key uses `dbPath` but `Open()` validates `projectID` (nit)

**File**: `store.go:97-129`
**What's wrong**: The cache key is `dbPath` (the full store path), which is correct. However, `Open()` validates `projectID` (checks for empty and `..`) before computing `dbPath`. This means two different `projectID`s that produce the same `dbPath` (e.g., `foo` and `foo/../foo`) would share a cache entry, but `foo/../foo` would be rejected by the validation. This is not a bug, but it's worth noting that the validation happens before the cache lookup.

**No fix needed** — this is a nit, not a bug.

### F5 — `TestStoreOpenWALRetry` does not verify the retry count (minor)

**File**: `store_pooling_test.go:136-146`
**What's wrong**: The test verifies that the contended write succeeds after the lock is released, but does not verify that the retry loop actually retried (e.g., by checking the number of attempts or the timing). The test could pass even if the retry loop were broken (e.g., if `openWithWALRetry` succeeded on the first attempt).

**Fix**: Add a check that the elapsed time is consistent with at least one retry (e.g., `elapsed > 50ms`). The test already has a weak check for this (line 141: `elapsed < 50*time.Millisecond`), but it's not strong enough to prove that a retry occurred.

## Evidence Checked

| Check | Verified? | Finding |
|-------|-----------|---------|
| 1. Additive-ness | ✅ | The diff is additive. `Open()` and `Close()` signatures are unchanged. The cache is opt-out via `SKILLGRID_MNEMONIC_DISABLE_CACHE=1`. No existing migrations are modified. |
| 2. Refcount correctness | ✅ | The refcount is atomic (`atomic.Int64`). The eviction path is serialized by `cacheMu`. A losing concurrent open race closes its `db` and reuses the winner. No obvious race conditions. |
| 3. WAL retry | ❌ | **F1**: The retry loop is not exercised by any test. **F2**: `isWALBusy` does not match the actual modernc/sqlite error format. `busy_timeout=10000` is applied on every new connection (store.go:174). |
| 4. Cache key | ✅ | The cache key is the full store path (`dbPath`), not just `projectID`. Two different `dataDir`s with the same `projectID` would not collide. |
| 5. Migration | ✅ | `020_ttl_extraction.sql` uses `CREATE TABLE IF NOT EXISTS` (idempotent). It sorts after `019_session_relay.sql`. The column types are per spec. **F3**: The filename does not match the brief. |
| 6. Test isolation | ✅ | All new tests use `t.TempDir()` (unique paths). The env-bypass test uses `t.Setenv` (cleaned up automatically). |
| 7. TDD discipline | ❌ | **F1**: The WAL test does not verify the retry loop. The cached-handle test does assert that the second `Open()` returns the same `*sql.DB` (line 34), which is good. |
| 8. Pattern fit | ✅ | The code follows existing store test conventions (table-driven where appropriate, naming conventions). The `Open`/`Close` structure is preserved. |

## Approved to merge: **No**

**Reason**: F1 and F2 are both major. The WAL retry loop is a core requirement of the step, but it is not exercised by any test, and `isWALBusy` does not match the actual error format. These need to be fixed before merge.

**Recommended next steps**:
1. Fix `isWALBusy` to match the actual modernc/sqlite error format (F2).
2. Add a test that exercises the WAL retry loop (F1).
3. Consider renaming the migration file to match the brief (F3) or updating the brief.
