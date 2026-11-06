# Step 04 — TTL defaults — Review

Change: `014-mnemonic-performance`, step 04 (`04-ttl-defaults`).
Reviewer: fresh, no prior context. Scope: the 5-file step-04 diff + existing code it builds on.

## Verdict: **PASS WITH WARNINGS**

All three sub-tasks deliver. 04.2 was genuinely test-only. Two non-blocking findings
(one behavioral surprise worth documenting, one missing config-level test).

---

## Findings

### F1 — minor — re-save does NOT refresh TTL (dedup + topic-key upsert both preserve the original stamp)

`Save()` has three write paths; only the final INSERT auto-stamps `expires_at`:

1. **Dedup path** (`memory/service.go:282-284`):
   ```go
   if err == nil {
       s.BumpDuplicate(ctx, existingID)
       return existingID, nil   // returns the existing row; NO expires_at re-stamp
   }
   ```
   `BumpDuplicate` (lifecycle.go:104) touches only `duplicate_count`/`last_seen_at`.

2. **Topic-key upsert** (`memory/service.go:309-317`): the `UPDATE ... SET source,
   session_id, type, title, content, normalized_hash, revision_count, last_seen_at,
   updated_at` leaves `expires_at` untouched.

3. **INSERT** (`memory/service.go:337-351`): the only path that auto-stamps.

Consequence: a memory first saved 7d ago and re-saved today keeps its **original**
`expires_at` and retires at the original time — repeated saves do not extend its life.
This is a reasonable "expiry = time since first creation" semantic, but it is **not**
what a reader might assume from "auto-set on save," and it interacts with the 24h dedup
window: an observation re-saved within 24h never gets a fresh stamp, so a long-lived,
frequently re-saved memory silently retires 7d after its first creation. **Flagged per the
brief's "re-stamping on every save" note: this is the *opposite* choice (never re-stamp),
and it is undocumented.**

Fix (no code change required for this review): add a one-line comment at the dedup
branch documenting "re-saves preserve the original expires_at; expiry is anchored to
first creation." If the intended semantic is "each save extends life," the dedup/upsert
paths need a `SET expires_at = now + TTL`.

### F2 — minor — config-level TTL override is implemented but untested

`mnemonic.ttl` is read in `config/load.go:245-249`:
```go
if section.TTL != "" {
    if d, err := time.ParseDuration(section.TTL); err == nil && d > 0 {
        out.TTL = d
    }
}
```
and wired in `service.go:307` (`mem.SetTTL(cfg.TTL)`). This is correct: blank →
keeps `DefaultMemoryTTL` (168h); malformed → `err != nil` so the fallback is kept;
`d <= 0` (e.g. `-5h`) → fallback. No crash path. **But no test covers it.** The report's
"scratch test `TestTTLOverrideFromConfig`" is not in the diff, so the override is
untested at the `config.Load` / `openProject` seam. `TestSaveAutoSetsExpiresAt` exercises
only the `defaultMemoryTTL` path (memory.Service built directly, `SetTTL` never called →
`effectiveTTL()` returns the 7d constant). A regression that drops the `cfg.TTL` wiring
or the `parse` guard would not be caught.

Fix: add a unit test that sets `mnemonic.ttl: 48h` in an `indexing.yaml`, calls
`config.Load`, and asserts `TTL == 48h`; plus a malformed case asserting the 168h
fallback. Cheap and closes the loop on the "configurable via mnemonic.ttl" DoD row.

### F3 — nit — `runMemExpire` ignores the `--dir` it is handed (harmless, but confusing)

`cmd/skillgrid/mem.go` `runMemExpire(svc, dataDir)`:
```go
_ = dataDir // the service already owns the data dir it was opened with
counts, err := svc.RunTTLExpiry(hCtx())
```
The `dataDir` arg is discarded; the sweep operates on the service's own `dataDir` (set
by `openMemService` → `service.New(dd)`). This is functionally correct **because**
`RunTTLExpiry` iterates `ListProjects()` over that same dir — but it makes the signature
lie: the parameter looks load-bearing and isn't. Either drop the `dataDir` parameter or
assert `svc`'s dir matches. Cosmetic.

### F4 — nit — `RunTTLExpiry`'s `os.IsNotExist` guard is dead (double-protected)

`service.go:447`:
```go
projects, err := s.ListProjects()
if err != nil {
    if os.IsNotExist(err) {   // unreachable: ListProjects already returns nil on ENOENT
        return map[string]int{}, nil
    }
    return nil, err
}
```
`ListProjects` (service.go:371-378) already converts a missing data dir to
`return []string{}, nil`, so `ListProjects` never returns an `os.IsNotExist` error. The
guard is defensive belt-and-suspenders, not wrong — just redundant. The real
"empty sweep" path (missing dir → empty slice → empty counts map) is taken correctly and
is covered by `TestMemExpireCLI`'s empty-dir case.

---

## Evidence checked (the 7 checks)

**1. 04.1 correctness — VERIFIED.** Auto-set logic (`memory/service.go:337-342`):
```go
expiresAt := strings.TrimSpace(in.ExpiresAt)
if expiresAt == "" {
    expiresAt = time.Now().UTC().Add(s.effectiveTTL()).Format(time.RFC3339)
} else if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
    return 0, fmt.Errorf("invalid expires_at: %w", err)
}
```
- Sets `expires_at` **only** when the input's is empty/blank (`TrimSpace` → `== ""`). ✅
- Explicit value **preserved byte-for-byte** (the `else if` only *validates*, it does not
  rewrite `expiresAt`). ✅
- Default read from config via `effectiveTTL()` → `s.ttl` (set by `SetTTL` from
  `cfg.TTL`) with `defaultMemoryTTL = 7*24h` fallback. ✅
- Invalid config value → fallback (see F2); malformed **input** `expires_at` → hard error
  (not auto-corrected), which is the right call. ✅
- Reran `TestSaveAutoSetsExpiresAt`: **PASS** (asserts ~now+7d within ±1h, and explicit
  `2030-01-02T03:04:05Z` preserved unchanged). ✅

**2. 04.1 config wiring — VERIFIED.** New key `mnemonic.ttl` (Go duration string) in
`config/load.go:96` (`mnemonicSection.TTL`), parsed at `:245-249` into
`Indexing.TTL` (a `time.Duration`), default `DefaultMemoryTTL` (168h) at `:170`. Read
**per-project in `openProject`** (`service.go:293` `cfg := config.Load(root)` →
`:307` `mem.SetTTL(cfg.TTL)`), mirroring the existing `SetBudget` seam exactly.
`SetTTL` (memory/service.go:90-94) only sets when `d > 0`; `effectiveTTL()` (:98-103)
falls back to the 7d constant. Bad value → 7d, no crash. ✅ (config-level test missing — F2.)

**3. 04.2 — WAS IT REALLY TEST-ONLY? — VERIFIED: YES, the existing behavior was already correct.**
- `TTLRetire` (lifecycle.go:145-166) does exactly the brief asks:
  ```sql
  UPDATE observations SET deleted_at = ?
  WHERE project = ? AND (deleted_at IS NULL OR deleted_at = '')
    AND expires_at IS NOT NULL AND expires_at != ''
    AND strftime('%s', expires_at) <= strftime('%s', 'now')
  ```
  i.e. expired (≤ now) → soft-delete; future → untouched; no-expires_at → untouched. ✅
- Search **does** exclude retired/expired. Both `SearchWithScope` (service.go:418-428)
  and `SearchOwnerScoped` (service.go:476-489) carry:
  ```sql
  WHERE observations_fts MATCH ? AND o.deleted_at IS NULL AND o.project = ?
    AND (o.expires_at IS NULL OR o.expires_at = '' OR strftime('%s', o.expires_at) > strftime('%s', 'now'))
  ```
  `deleted_at IS NULL` excludes soft-deleted rows; the `expires_at` clause is a redundant
  belt-and-suspenders (a row is retired → `deleted_at` set → already filtered). No bug. ✅
- Reran `TestTTLRetireOnlyExpired`: **PASS**. It creates all 3 cases (expired −1h,
  future +1h, no-expires_at cleared via `SetExpiresAt(id, "")`), asserts `TTLRetire`
  returns exactly 1, asserts search excludes the retired row and returns both live rows,
  and asserts `Get(retiredID)` is not-found while the other two are readable. ✅
- The brief's premise ("existing behavior was already correct, 04.2 is mostly the test")
  is **confirmed**. No production line in lifecycle.go changed in the diff.

**4. 04.3 RunTTLExpiry — VERIFIED.** `service.go:440-468`:
- Iterates **all** projects via `ListProjects()` (reads every `*.sqlite` in the data dir). ✅
- Missing store → `openProject` error → `counts[pid] = 0; continue` (best-effort skip, no
  abort). A single store's `TTLRetire` error → `n = 0`, sweep continues. ✅
- Empty/missing data dir → `ListProjects` returns `[]string{}, nil` → empty counts, `nil`
  error. ✅ (The `os.IsNotExist` branch at :447 is dead — see F4.)
- Returns per-project counts map. CLI `runMemExpire` prints `{"retired": {...}, "total": N}`
  via `printJSON` and exits 0 (only exits 1 on a hard `RunTTLExpiry` error). ✅
- Manual switch, not cobra (`mem.go:80-81` `case "expire": runMemExpire(svc, dataDir)`). ✅
- Reran `TestMemExpireCLI` + `TestRunTTLExpiryDirect`: **PASS**. CLI test asserts both
  projects appear, per-project counts `"memcli-expire-exp": 1` / `"memcli-expire-live": 0`,
  idempotent second run, and exit 0 on a fresh empty data dir. ✅

**5. Additive-ness / tool contract — VERIFIED: no `mem_*`/`code_*`/`web_*` signature changed.**
- Diff touches 5 files; none is an MCP tool registration file. `Save` is an internal
  method; `RunTTLExpiry`/`SetTTL`/`effectiveTTL` are new seams.
- `mem_save` MCP response is `{"id", "project"}` (`mcp/tools_memory.go:260-264`) — the
  auto-set `expires_at` does **not** leak into the tool's return shape.
- `mem expire` is a **new** CLI subcommand (manual switch), additive.
- The one observable shape change: `mem_search`/`mem_context`/`mem_get_observation` now
  *exclude* observations that are past expiry (and the `Observation` JSON type already
  carried `expires_at,omitempty`). This is the *intended* 04.2 behavior, not a regression —
  and `omitempty` means the field is only present when set, so no new always-present key. ✅

**6. TDD discipline — VERIFIED.**
- `TestSaveAutoSetsExpiresAt`: asserts ~7d out **with a ±1h tolerance** (`d > time.Hour ||
  d < -time.Hour`) AND explicit value preserved. ✅
- `TestTTLRetireOnlyExpired`: all 3 cases (expired/1h/none), asserts ONLY the expired is
  retired (count == 1), the other two remain readable, and search excludes the retired. ✅
- `TestMemExpireCLI`: per-project counts + exit 0 on missing/empty store. ✅
- I independently re-ran all four; all PASS.

**7. Edge cases — VERIFIED (both behave sensibly, one is a documentation gap → F1).**
- **Explicitly-past `expires_at`**: preserved, not auto-retired on save. I wrote a throwaway
  test (`TestEdgeExplicitPastPreserved`): saved with `ExpiresAt = now − 2h`, re-read the
  row → `expires_at = 2026-09-11T12:36:00Z` (in the past), `Get` succeeds. It will be
  retired on the **next** `mem expire` sweep (`TTLRetire` picks up `expires_at <= now`).
  Correct. ✅
- **Same observation saved twice**: **NOT** re-stamped. Dedup path returns the existing row
  (`service.go:282-284`, only `BumpDuplicate`); topic-key upsert path leaves `expires_at`
  alone (`:309-317`). So expiry is anchored to **first creation**, and repeated saves do not
  extend life. This is a defensible semantic but **undocumented** and the opposite of
  "re-stamp on every save" — flagged as F1.

---

## Global Constraints — held
- **Additive only**: `expires_at` column pre-existed; `SaveInput.ExpiresAt` is a new
  optional field; `TTLRetire`/`TTLSoftExpiry` unchanged (only tested); `mem expire` new;
  `mnemonic.ttl` new config key with default. No schema rewrite. ✅
- **Tool contracts unchanged**: no `mem_*`/`code_*`/`web_*` signature or response shape
  changed (see check 5). ✅
- **TTL defaults to 7 days, configurable via `mnemonic.ttl`**: implemented + wired; config
  parse guarded. (config-level test missing — F2.) ✅
- **modernc.org/sqlite stays / no CGO**: untouched by the diff. ✅

## Approved to merge: **yes** (with the two minor follow-ups — F1 documentation, F2 config
test — tracked as nits/minors; neither blocks the step-04 DoD).
