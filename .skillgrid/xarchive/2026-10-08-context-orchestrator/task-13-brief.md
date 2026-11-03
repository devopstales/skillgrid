# T13 Brief — traffic learner human-gate: `ctx learn apply` / `revoke` (TASK-101)

## Goal
Add the human-gate promotion lifecycle for traffic memories (plan §18, M23 "proposed /
applied / revoked"): `ctx learn apply <id>` promotes a candidate memory to **active** and
makes it appear in subsequent compression context; `ctx learn revoke <id>` removes it.
Only existing candidates can be promoted — a non-existent id is a clear error (the
human-gate, plan §18.7 "No automatic promotion").

This is the ONLY ticket in PR5 that touches the traffic package after T12. T12 is complete
(`internal/ctx/traffic/{rules,learner,evict}.go` + `rules.yaml`, store methods in
`store/tables.go`). Build on it; do not rewrite the extractor.

## The gap you must fill
T12's `ctx_traffic_memories` (061) has NO status column and NO applied_at:
```
memory_id PK, session_id, category, memory_text, evidence_json, confidence, created_at
```
So the candidate/active/reverted lifecycle has nowhere to live yet. The fix is an
ADDITIVE migration + a status-aware store layer. Do NOT change the 061 DDL (already applied
in the wild); add a new migration.

## Implementation

### 1. Migration `068_ctx_traffic_memory_status.sql`
Next free slot is 068 (migrations auto-embed via `//go:embed migrations/*.sql` in store.go).
Add the lifecycle columns idempotently:
```sql
-- 068_ctx_traffic_memory_status.sql — CTX human-gate lifecycle (T13).
-- Adds the candidate/active/reverted lifecycle to ctx_traffic_memories (061).
-- ADD COLUMN IF NOT EXISTS is a no-op if already present (re-run safe).
ALTER TABLE ctx_traffic_memories ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'candidate';
ALTER TABLE ctx_traffic_memories ADD COLUMN IF NOT EXISTS applied_at INTEGER;
```
Values: `candidate` (default, emitted by T12 extractor) → `active` (after apply) →
`reverted` (after revoke). `applied_at` is the UnixNano when activated (NULL for
candidate/reverted).

### 2. Store layer (`mnemonic/internal/ctx/store/tables.go`) — extend, don't replace
- Add `Status string` + `AppliedAt *int64` to the `TrafficMemory` struct (tables.go:91).
  Keep the existing fields in order; append the two new ones at the end.
- Update `PutTrafficMemory`, `GetTrafficMemory`, `ListTrafficMemories`,
  `ListAllTrafficMemories` to read/write `status` (default `'candidate'` when empty) and
  `applied_at` (NULL-safe: scan into `*int64`). All must stay fail-open (ADR-0016) — a
  missing `status` column (a pre-068 store) must degrade gracefully: treat status as
  `'candidate'`, applied_at as nil. Follow the existing `queryFailOpen`/`execFailOpen`
  pattern.
- Add `SetTrafficMemoryStatus(memoryID string, status string, appliedAt *int64) error`:
  `UPDATE ctx_traffic_memories SET status=?, applied_at=? WHERE memory_id=?`. Fail-open
  (no-op) if the table is missing. Return `sql.ErrNoRows`-style "not found" ONLY when the
  table exists but the row does not — the gate needs to distinguish "table missing"
  (fail-open no-op, ADR-0016) from "row absent" (human-gate error).

### 3. Gate logic (`mnemonic/internal/ctx/traffic/` — new `gate.go`)
```go
// Apply promotes a candidate memory to active. Returns an error when the memory id does
// not exist (the human-gate: only existing candidates promote).
func (s *Gate) Apply(memoryID string) error

// Revoke removes (reverts) an active memory: sets status to 'reverted'.
func (s *Gate) Revoke(memoryID string) error

// ActiveMemories returns the active traffic memories that feed compression context.
func (s *Gate) ActiveMemories(sessionID string) ([]*store.TrafficMemory, error)
```
- `Apply`: `GetTrafficMemory(id)` → if nil (absent) return a clear error like
  `fmt.Errorf("traffic memory %q not found (only existing candidates can be applied)", id)`.
  Then `SetTrafficMemoryStatus(id, "active", &now)`. Re-applying an already-active memory
  is an idempotent no-op (refreshes applied_at) — do not error.
- `Revoke`: `GetTrafficMemory(id)` → if nil, clear not-found error. Then
  `SetTrafficMemoryStatus(id, "reverted", nil)`. Re-revoking is idempotent.
- `ActiveMemories`: `ListTrafficMemories(sessionID)` filtered to `status == "active"`
  (or a dedicated `ListActiveTrafficMemories` store query — your choice, but it must
  return ONLY active rows). This is the seam that proves "applied as a rule".
- `Gate` is a thin wrapper over `*store.Store`. Keep it testable (inject the store).

### 4. Context-injection seam (proves "appears in subsequent compression context")
The acceptance AC is "apply marks a candidate promoted and it **appears in subsequent
compression context**". Wire `Gate.ActiveMemories` into the compression path so active
memories are available to the compressor:
- Add a seam in `internal/ctx/proxy` (the compression path) that, given a `*ctxstore.Store`
  (or a `*traffic.Gate`), fetches `ActiveMemories(sessionID)` and appends their
  `memory_text` to the compression context block BEFORE transform. Fail-open: if the gate
  returns no active memories or errors, the context is unchanged (never 500s, ADR-0016).
- Keep it minimal and isolated — a single helper (e.g. `activeMemoryContext(st, sessionID)
  (string, error)`) called from the transform path. Do NOT re-architect the proxy. If the
  transform path already has a clean place to prepend context, use it.
- The KEY proof is the test: a candidate (status=candidate) is NOT in the compression
  context; after `Apply`, it IS; after `Revoke`, it is NOT again.

### 5. CLI (`cmd/mnemonic/ctx.go` or a new `ctx_learn.go`)
Add `mnemonic ctx learn apply <id>` and `mnemonic ctx learn revoke <id>`:
- Resolve the shared store the same way `runCtxProxy` does (`newMnemonicService` +
  `OpenForDirectory` / `OpenAt` + `ctxstore.New(h.Store())`). If no store is available,
  print a fail-open warning and exit 0 (ADR-0016) — but for apply/revoke a nil store means
  the id cannot be found, so exit non-zero with a clear message.
- `apply`: call `Gate.Apply(id)`; on success print the id + "applied"; on not-found print
  the clear error to stderr and exit non-zero.
- `revoke`: same shape.
- Register the `learn` subcommand under `ctx` in the existing `runCtx` dispatcher
  (ctx.go). Follow the existing subcommand dispatch pattern (see `ctx proxy`, `ctx wrap`).

## Constraints
- ADR-0016 fail-open: missing table/column → no-op / default, never an error that 500s the
  proxy. The not-found human-gate error is the ONE exception (it's a user action, not a
  request path).
- ADR-0033: reuse the shared handle (`ctxstore.New(h.Store())`), never a second Open.
- Do NOT change T12's extractor logic (rules/learner/evict) except the additive store
  struct field + status-aware queries.
- Stdlib only. No new dependency, no go.work change.
- Keep `single_open_test.go` green.

## Acceptance gates (all must pass)
- `go build ./...`
- `go test ./internal/ctx/traffic/... -count=1`
- `go test ./internal/ctx/store/... -count=1`
- `go test ./internal/ctx/proxy/... -count=1`
- `go vet ./internal/ctx/traffic/... ./internal/ctx/store/... ./internal/ctx/proxy/... ./cmd/...`
- `gofmt -l` on every file you touched (must be empty)

Required tests (in `internal/ctx/traffic/` + `internal/ctx/proxy/`):
1. **apply marks promoted + appears in context**: seed a `ctx_traffic_memories` row
   (status=candidate) → `ActiveMemories` does NOT include it → `Apply(id)` → `ActiveMemories`
   includes it AND the proxy compression context contains its memory_text.
2. **revoke reverts + disappears**: after apply, `Revoke(id)` → `ActiveMemories` excludes it
   again AND the compression context no longer contains it.
3. **non-candidate id rejected**: `Apply("nope")` → clear "not found" error (and
   `Revoke("nope")` too); status unchanged / no row created.
4. **idempotency**: `Apply` twice → still one active row; `Revoke` twice → no error.
5. **fail-open**: a store missing the 068 status column (or nil store) → `ActiveMemories`
   returns empty, no error; `Apply` on a missing table → no-op or clear message, not a panic.

## Report
Write the FULL report to `.skillgrid/sdd/2026-10-08-context-orchestrator/task-13-report.md`:
what changed (files + functions), the status lifecycle model, the context-injection seam
(how active memories reach compression), the human-gate error, and gate outputs
(pass/fail + counts). Generate `.skillgrid/sdd/2026-10-08-context-orchestrator/task-13.diff`
(git diff of your commits).

## Commit
Conventional subject, e.g. `feat(ctx): learn apply/revoke human gate for traffic memories`.
EXACTLY this body block:
```
[skillgrid-context]
change: 2026-10-08-context-orchestrator
ticket: TASK-101
wave: 6
pr: PR5
```
