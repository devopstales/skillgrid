# Step 04 — TTL defaults (04-ttl-defaults) — Report

Change: `014-mnemonic-performance`, step 04.

## Summary

| Sub-task | Status | Commit |
|----------|--------|--------|
| 04.1 Save() auto-sets expires_at | GREEN | `d8f50dd` |
| 04.2 TTLRetire retires only expired | GREEN (test-only) | `d8f50dd` (test) |
| 04.3 `mem expire` CLI | GREEN | `f36eb55` |
| wiring mnemonic.ttl → SetTTL | GREEN | `8fc99cf` |

## 04.1 — Save() auto-sets expires_at (RED → GREEN)

`SaveInput` (memory/service.go) has a new optional field:

```go
// ExpiresAt is an optional RFC3339 expiry. When non-empty it is preserved
// unchanged; when empty, Save() auto-stamps it to now + the configured TTL
// (mnemonic.ttl, default 7 days).
ExpiresAt string
```

In `Save()` (memory/service.go), just before the INSERT:

```go
expiresAt := strings.TrimSpace(in.ExpiresAt)
if expiresAt == "" {
    expiresAt = time.Now().UTC().Add(s.effectiveTTL()).Format(time.RFC3339)
} else if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
    return 0, fmt.Errorf("invalid expires_at: %w", err)
}
```

`expires_at` was added to the INSERT column list and bound via `nullString(expiresAt)`.

**How the 7-day default is read (config key + fallback):**
- New config key `mnemonic.ttl` (a Go duration string, e.g. `168h`/`48h`). It is
  parsed in `config/load.go` (`mnemonicSection.TTL` → `Indexing.TTL`) with a
  `DefaultMemoryTTL = 7 * 24 * time.Hour` fallback. A blank or malformed value
  keeps the default so a bad key never disables expiry.
- The memory `Service` carries a `ttl time.Duration` field; `SetTTL(d)` sets it
  when `d > 0`; `effectiveTTL()` returns it or falls back to the package
  constant `defaultMemoryTTL = 7 * 24 * time.Hour`. A `memory.Service` built
  directly in unit tests (no config) therefore auto-stamps exactly 7d.
- The service layer (`openProject`) calls `mem.SetTTL(cfg.TTL)` right after
  `SetBudget`, so the `mnemonic.ttl` override reaches the production path.
  `Save()`'s unit-test path builds the memory service directly and therefore
  uses the `defaultMemoryTTL` (7d) constant — same value as the config default,
  so behavior is identical either way.

`TestSaveAutoSetsExpiresAt`:
- Save with no expires_at → stored obs has `expires_at` ≈ now+7d (±1h).
- Save with explicit `2030-01-02T03:04:05Z` → preserved unchanged.

RED evidence (implementation temporarily reverted, test kept):
```
--- FAIL: TestSaveAutoSetsExpiresAt (0.33s)
    ttl_defaults_test.go:30: auto expires_at was not set; expected ~now+7d
```
GREEN after implementation:
```
--- PASS: TestSaveAutoSetsExpiresAt
```

## 04.2 — TTLRetire retires only expired (test-only, no code change)

`TTLRetire` (lifecycle.go:145) and the search soft-exclude
(`SearchWithScope`/`SearchOwnerScoped`/`Governance` all carry
`deleted_at IS NULL` plus `expires_at IS NULL OR '' OR strftime(...) > now`)
were **already correct**. No production change was needed — the fix is the test.

`TestTTLRetireOnlyExpired`:
- 3 obs: expired (-1h), expiring in +1h, no expires_at (default cleared via
  `SetExpiresAt(id, "")`).
- `TTLRetire` → returns exactly `1`; only the expired row is soft-deleted
  (`Get` on it returns not-found; the other two remain readable).
- `Search` excludes the retired row but still returns both live rows.

Result: PASS with no change to lifecycle.go (RED only in the sense the test did
not exist; the code already satisfied it).

## 04.3 — `skillgrid mem expire` CLI (RED → GREEN)

**`RunTTLExpiry` on the service** (internal/mnemonic/service/service.go):

```go
func (s *Service) RunTTLExpiry(ctx context.Context) (map[string]int, error) {
	if s == nil {
		return map[string]int{}, fmt.Errorf("service not initialized")
	}
	projects, err := s.ListProjects()
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil // empty sweep, not an error
		}
		return nil, err
	}
	counts := make(map[string]int, len(projects))
	for _, pid := range projects {
		h, cleanup, oerr := s.openProject(pid, ".")
		if oerr != nil {
			counts[pid] = 0 // best-effort: skip missing/corrupt store
			continue
		}
		n, rerr := h.Memory().TTLRetire(ctx)
		if rerr != nil {
			n = 0
		}
		counts[pid] = n
		cleanup()
	}
	return counts, nil
}
```

**`mem expire` subcommand** (cmd/skillgrid/mem.go, added to the manual switch):

```go
case "expire":
    runMemExpire(svc, dataDir)
```

```go
func runMemExpire(svc *service.Service, dataDir string) {
	_ = dataDir // the service already owns the data dir it was opened with
	counts, err := svc.RunTTLExpiry(hCtx())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	out := map[string]any{"retired": counts}
	total := 0
	for _, n := range counts {
		total += n
	}
	out["total"] = total
	printJSON(out)
}
```

Usage string updated to `<layers|governance|share|search|context|timeline|expire>`
with an `expire` line. It ignores `--project` (cross-project sweep) and always
exits 0 on a successful sweep (missing/empty store → empty counts, no error).

`TestMemExpireCLI`:
- seeds two projects (one expired obs, one live obs) → `mem expire` output lists
  both projects with per-project counts `memcli-expire-exp: 1` /
  `memcli-expire-live: 0` and `total: 1`.
- a second `mem expire` run is idempotent (still exit 0, still lists projects).
- a fresh empty data dir → exit 0 (graceful, no store).
`TestRunTTLExpiryDirect`: asserts `RunTTLExpiry` returns `{exp:1, live:0}`.

RED evidence (before `RunTTLExpiry`/`expire` existed):
```
cmd/skillgrid/mem_expire_test.go:108:21: svc.RunTTLExpiry undefined
    (type *service.Service has no field or method RunTTLExpiry)
```
GREEN after implementation:
```
--- PASS: TestMemExpireCLI
--- PASS: TestRunTTLExpiryDirect
```

## Global Constraints — held

- **Additive only**: `expires_at` column already existed; `SaveInput.ExpiresAt`
  is a new optional field; `TTLRetire` behavior unchanged (just tested); `mem
  expire` is a new subcommand. No schema rewrite.
- **Tool contracts unchanged**: no `mem_*`/`code_*`/`web_*` response shape
  changed. `RunTTLExpiry` is a new method; `SetTTL`/`effectiveTTL` are new
  private/public memory seams.
- **TTL defaults to 7 days, configurable via `mnemonic.ttl`**: verified
  `config.Load` returns 168h default, 48h when `mnemonic.ttl: 48h`, and 168h
  when malformed (scratch test `TestTTLOverrideFromConfig` PASS).
- **`modernc.org/sqlite` stays; no CGO**: untouched.

## Verification

### Focused step-04 tests
```
$ go test ./internal/mnemonic/memory/ -run 'TestSaveAutoSetsExpiresAt|TestTTLRetireOnlyExpired' -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	3.356s

$ go test ./cmd/skillgrid/ -run 'TestMemExpireCLI|TestRunTTLExpiryDirect' -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid
```

### Full relevant suites
```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	23.331s

$ go test ./skillgrid-cli/cmd/skillgrid/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	163.468s
```

### Baselines (005/008/010/011/013)
```
$ go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/... \
    ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/affected/... \
    ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/pdg/... \
    ./skillgrid-cli/internal/mnemonic/codeindex/... -count=1

ok  	.../internal/mnemonic/store	10.158s
--- FAIL: TestReindexStructuralIsEmbedderFree (2.13s)   [PRE-EXISTING, see below]
ok  	.../internal/mnemonic/route	6.829s
ok  	.../internal/mnemonic/affected	29.027s
ok  	.../internal/mnemonic/community	2.495s
ok  	.../internal/mnemonic/pdg	14.176s
ok  	.../internal/mnemonic/codeindex	38.077s
```

Service package minus the pre-existing failure:
```
$ go test ./internal/mnemonic/service/ -count=1 -skip 'TestReindexStructuralIsEmbedderFree'
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	44.038s
```

### Pre-existing failure (not caused by step 04)
`TestReindexStructuralIsEmbedderFree` (service/reindex_structural_test.go:107,
"count gamma: sql: database is closed") **also fails at baseline commit
`c44c78d`** (before any step-04 change), confirmed by `git checkout c44c78d --`
on the four step-04 files and re-running the test in isolation. It is unrelated
to TTL — it lives in the code-index reindex path.

## Concerns

- **`TestReindexStructuralIsEmbedderFree`** is a pre-existing baseline failure
  (confirmed at `c44c78d`) in the service package; it does not block step 04.

## Commits

- `d8f50dd` feat(mnemonic): auto-set expires_at to now+7d on save
  - `internal/mnemonic/config/load.go` (mnemonic.ttl key + DefaultMemoryTTL)
  - `internal/mnemonic/memory/service.go` (SaveInput.ExpiresAt, SetTTL,
    effectiveTTL, Save() auto-stamp)
  - `internal/mnemonic/memory/ttl_defaults_test.go` (04.1 + 04.2 tests)
- `f36eb55` feat(mnemonic): add mem expire CLI subcommand for TTL cleanup
  - `internal/mnemonic/service/service.go` (RunTTLExpiry)
  - `cmd/skillgrid/mem.go` (expire case + runMemExpire + usage)
  - `cmd/skillgrid/mem_expire_test.go` (TestMemExpireCLI + TestRunTTLExpiryDirect)
- `8fc99cf` feat(mnemonic): wire mnemonic.ttl config into openProject SetTTL
  - `internal/mnemonic/service/service.go` (1-line additive wiring of the
    `mnemonic.ttl` override)
