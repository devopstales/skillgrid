## 04-ttl-defaults

### Goal

Auto-set expires_at on save + scheduled TTL cleanup.

### Out of scope / Non-Goals

- Store pooling, FTS trigram, parallel search, extraction, embedder.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-04` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: auto-set expires_at in Save()
- `internal/mnemonic/service/service.go` — MODIFY: RunTTLExpiry
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `expire` subcommand

**Interfaces:**
- Consumes: cached store handles from step 01; existing `Save()` and `TTLSoftExpiry()` / `TTLRetire()` methods
- Produces: auto-set `expires_at` (7-day default) on save; `RunTTLExpiry()` on Service; `skillgrid mem expire` CLI

### Tasks

- [ ] 04.1 `[RED]` Save() auto-sets expires_at to now + 7 days when not explicitly provided
  - [ ] 04.1.a Write failing test (`TestSaveAutoSetsExpiresAt`): call `Save()` with a `SaveInput` that has no `expires_at` set; verify the stored observation has `expires_at` approximately 7 days in the future; call `Save()` with an explicit `expires_at` and verify it is preserved unchanged
  - [ ] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSaveAutoSetsExpiresAt'` — Expected: FAIL
  - [ ] 04.1.c Minimal implementation — in `Save()` (lifecycle.go), check if `expires_at` is zero/nil; if so, set it to `time.Now().Add(7 * 24 * time.Hour)`; read default TTL from `mnemonic.ttl` config key (7 days default); allow explicit override
  - [ ] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSaveAutoSetsExpiresAt'` — Expected: PASS
  - [ ] 04.1.e Commit — `feat(mnemonic): auto-set expires_at to now+7d on save`
- [ ] 04.2 `[RED]` TTLRetire only retires expired observations, not future ones
  - [ ] 04.2.a Write failing test (`TestTTLRetireOnlyExpired`): create 3 observations — one expired (expires_at in past), one expiring in 1 hour, one with no expires_at; call `TTLRetire`; verify only the expired one is soft-deleted; verify the other two remain active; verify `mem_search` excludes the retired observation
  - [ ] 04.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTTLRetireOnlyExpired'` — Expected: FAIL
  - [ ] 04.2.c Minimal implementation — implement `TTLRetire` to query observations where `expires_at IS NOT NULL AND expires_at < now` and soft-delete them (set `deleted_at`); ensure `TTLSoftExpiry` and `TTLRetire` are operational by default; make cleanup best-effort (skip missing stores, no error)
  - [ ] 04.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTTLRetireOnlyExpired'` — Expected: PASS
  - [ ] 04.2.e Commit — `feat(mnemonic): TTLRetire soft-deletes only expired observations`
- [ ] 04.3 `[AFK]` skillgrid mem expire CLI subcommand
  - [ ] 04.3.a Write failing test (`TestMemExpireCLI`): invoke `skillgrid mem expire` and verify it calls `RunTTLExpiry()` on all projects; verify the CLI outputs a summary of retired observations per project; verify it exits 0 even when a store is missing
  - [ ] 04.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExpireCLI'` — Expected: FAIL
  - [ ] 04.3.c Minimal implementation — add `mem expire` subcommand to `mem.go`; implement `RunTTLExpiry(ctx)` on `Service` that iterates all projects and calls `TTLRetire` on each; output per-project retirement counts; handle missing stores gracefully
  - [ ] 04.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExpireCLI'` — Expected: PASS
  - [ ] 04.3.e Commit — `feat(mnemonic): add mem expire CLI subcommand for TTL cleanup`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSaveAutoSetsExpiresAt\|TestTTLRetireOnlyExpired'` | PASS | | |
| Acceptance `@step-04` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify TTL default is configurable via mnemonic.ttl | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): auto-set expires_at and scheduled TTL cleanup`

---

