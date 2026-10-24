## 01-store-pooling

### Goal

Cached store handles + WAL lock retry.

### Out of scope / Non-Goals

- FTS trigram, parallel search, TTL, extraction, embedder.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: —

**Files:**
- `internal/mnemonic/store/store.go` — MODIFY: add handle cache, refcount, WAL retry
- `internal/mnemonic/store/migrations/014_ttl_extraction.sql` — CREATE: TTL config table, extraction metadata

**Interfaces:**
- Consumes: existing `store.Open()` / `store.Close()` API; `PRAGMA busy_timeout` support
- Produces: cached `ProjectHandle` via `sync.Map`; refcounted close; WAL lock retry with exponential backoff (50ms, 100ms, 200ms)

### Tasks

- [ ] 01.1 `[RED]` Cached handle reuse eliminates N+1 store opens
  - [ ] 01.1.a Write failing test (`TestStoreOpenReusesCachedHandle`): open a store for project A, open again for project A, verify the second open returns the same handle (same underlying `*sql.DB`); then open project B and verify it gets a different handle
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenReusesCachedHandle'` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation — add `sync.Map` handle cache in `store` package keyed by project ID; `Open()` checks cache first and returns existing handle with refcount incremented; `Close()` decrements refcount and removes from cache when it reaches 0
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenReusesCachedHandle'` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(mnemonic): add cached store handles with refcounted close`
- [ ] 01.2 `[RED]` WAL lock retry with exponential backoff
  - [ ] 01.2.a Write failing test (`TestStoreOpenWALRetry`): simulate a WAL lock by opening a second connection to the same store file while a write transaction is in progress; verify `Open()` retries with backoff (50ms, 100ms, 200ms) and eventually succeeds; verify `PRAGMA busy_timeout=10000` is set on the connection
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenWALRetry'` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation — add WAL lock retry loop in `Open()` with exponential backoff (50ms, 100ms, 200ms); fail after 3 attempts; set `PRAGMA busy_timeout=10000` on every new connection; add health-check on cache reuse (ping before returning cached handle)
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenWALRetry'` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(mnemonic): add WAL lock retry with exponential backoff`
- [ ] 01.3 `[AFK]` Migration for TTL config and extraction metadata tables
  - [ ] 01.3.a Write failing test (`TestMigration014TTLExtraction`): run migration `014_ttl_extraction.sql` on a fresh store; verify `ttl_config` table exists with `key`/`value` columns; verify `extraction_metadata` table exists with expected schema
  - [ ] 01.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestMigration014TTLExtraction'` — Expected: FAIL
  - [ ] 01.3.c Minimal implementation — create `internal/mnemonic/store/migrations/014_ttl_extraction.sql` with `CREATE TABLE IF NOT EXISTS ttl_config (key TEXT PRIMARY KEY, value TEXT NOT NULL)` and `CREATE TABLE IF NOT EXISTS extraction_metadata (id INTEGER PRIMARY KEY, session_id TEXT, content_hash TEXT, extracted_at TIMESTAMP, model TEXT)`
  - [ ] 01.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestMigration014TTLExtraction'` — Expected: PASS
  - [ ] 01.3.e Commit — `feat(mnemonic): add 014 migration for TTL config and extraction metadata`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/store/ -run 'TestStoreOpenReusesCachedHandle\|TestStoreOpenWALRetry'` | PASS | | |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/store/` | PASS | | |
| Rollback boundary | verify `SKILLGRID_MNEMONIC_DISABLE_CACHE=1` bypasses cache | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): cached store handles with refcount and WAL lock retry`

---

