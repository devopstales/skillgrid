# T13 — traffic learner human-gate: `ctx learn apply` / `revoke` — report

**Ticket:** TASK-101 · **Wave 6** · **PR5**
**Package:** `mnemonic/internal/ctx/traffic/` (+ `store`, `proxy`, `cmd`)
**Status:** DONE — all gates green, all 5 required ACs covered.

## What was built

The human-gate promotion lifecycle for traffic memories (plan §18, M23
"proposed / applied / revoked"). The T12 learner only ever emits **candidate**
memories; a human now promotes one with `mnemonic ctx learn apply <id>` (→
status `applied`, so it enters subsequent compression context) or demotes it
with `mnemonic ctx learn revoke <id>` (→ status `revoked`, so it leaves). Only
existing ids can be promoted — a non-existent id is a clear error (plan §18.7
"No automatic promotion").

Files:
- `internal/store/migrations/068_ctx_traffic_memory_status.sql` — additive
  migration: `status TEXT NOT NULL DEFAULT 'candidate'` + `applied_at INTEGER`.
- `internal/store/migrations_068_test.go` — fresh-store migration test
  (columns present, defaults fill, re-open does not re-fire).
- `internal/ctx/store/tables.go` — `TrafficMemory` +`Status`/`AppliedAt`;
  `SetTrafficMemoryStatus`; `PutTrafficMemory` + the shared SELECT col list made
  pre-068-aware.
- `internal/ctx/traffic/gate.go` — `Gate` (thin `*ctxstore.Store` wrapper).
- `internal/ctx/traffic/gate_test.go` — 6 gate tests over a real migrated store.
- `internal/ctx/proxy/memcontext.go` — context-injection helpers.
- `internal/ctx/proxy/memcontext_test.go` — 5 context-injection tests.
- `internal/ctx/proxy/proxy.go` — `gate` field + `SetGate` + `memoryContext`.
- `internal/ctx/proxy/compress.go` — seam wired into `serveCompress`.
- `cmd/mnemonic/ctx.go` — `learn` subcommand dispatch + usage.
- `cmd/mnemonic/ctx_learn.go` — `ctx learn apply|revoke` implementation.

## The status lifecycle model

`ctx_traffic_memories` (061) gained two additive columns in 068; the 061 DDL is
untouched. Values:

| status | meaning | applied_at |
|---|---|---|
| `candidate` (default) | emitted by the T12 learner, not yet promoted | NULL |
| `applied` | human-promoted → **active**, feeds compression context | unix millis of last apply |
| `revoked` | human-demoted → reverted, no longer active | NULL (cleared) |

- **Apply:** `GetTrafficMemory(id)` → absent ⇒ clear "not found" error. Present ⇒
  `SetTrafficMemoryStatus(id, "applied", &now)` — stamps `applied_at`.
  Re-apply is idempotent (refreshes `applied_at`).
- **Revoke:** `GetTrafficMemory(id)` → absent ⇒ clear "not found" error. Present ⇒
  `SetTrafficMemoryStatus(id, "revoked", nil)` — **clears `applied_at` to NULL**
  (per the brief: `applied_at` is set only while active, NULL otherwise).
  Re-revoke is idempotent. The row is kept (status `revoked`), not deleted.
- **ActiveMemories:** `ListTrafficMemories(sessionID)` filtered to
  `status == "applied"` — the seam that proves "applied as a rule".

## The store layer

`SetTrafficMemoryStatus(memoryID, status, appliedAt)` returns `(bool, error)`:
`(true, nil)` when the row was updated, `(false, nil)` when the row is absent
(the gate maps this to its not-found error) or the table is missing (ADR-0016
fail-open). It branches on the live schema via a `pragma_table_info`
`hasTrafficMemoryStatus()` probe: a pre-068 store (no status column) degrades
the UPDATE to a no-op rather than an unknown-column error.

Pre-068 fail-open (ADR-0016) lives in **three** readers/writers, not one:
1. **SELECT col list** — `trafficMemorySelectCols()` now a method. On a 068
   schema it names `COALESCE(status,'candidate')` + `applied_at`; on a pre-068
   schema it SELECTs the `'candidate'` literal + `NULL` so the nine-column scan
   succeeds (no unknown-column error, no hard fail).
2. **`PutTrafficMemory`** — on a pre-068 schema it INSERTs the base columns only
   (naming `status`/`applied_at` would hard-fail); the readers COALESCE the
   status to `candidate`.
3. **`SetTrafficMemoryStatus`** — the `hasTrafficMemoryStatus()` probe above.

## The context-injection seam (proves "appears in subsequent compression context")

`memcontext.go` provides two isolated helpers (no proxy re-architecture):
- `activeMemoryContext(g *traffic.Gate, sessionID string) string` — fetches
  `Gate.ActiveMemories(sessionID)` and renders their `memory_text` into a
  compact context block. Empty gate / no active memories / any error ⇒ `""`
  (fail-open, ADR-0016 — never a 500).
- `prependMemoryContext(body, ctx string) string` — prepends the block to the
  request body (the compression payload) when non-empty.

`proxy.go` carries a `gate *traffic.Gate` field + `SetGate` (nil is valid);
`compress.go` `serveCompress` calls `memoryContext(r)` (session header →
`activeMemoryContext`) and `prependMemoryContext` into the transform path.

**The key proof (test):** `memcontext_test.go` seeds a candidate → compression
context does **not** contain its text → after `Gate.Apply` it **does** → after
`Gate.Revoke` it does **not** again. `candidate` / `applied` / `revoked` each
asserted against the live compression body.

## The human-gate error

The ONE exception to the proxy's never-500 rule (it is a user action, not a
request path): `Gate.Apply`/`Revoke` on an absent id return
`fmt.Errorf("traffic memory %q not found (only existing candidates can be applied)", id)`
(and the revoke variant). A nil/missing store is fail-open (ADR-0016): the
`Gate` methods return nil / empty without panicking; the **CLI** surfaces the
missing store as a clear warning + non-zero exit (the id cannot be found
without a store).

## CLI

`mnemonic ctx learn apply <id>` / `revoke <id>` (new `ctx_learn.go`). Store
resolution mirrors `runCtxProxy` (ADR-0033: one shared handle via
`ctxstore.New(h.Store())`): `newMnemonicService` + `OpenForDirectory(dir)` with
an `OpenAt(pid, dir)` fallback (pid from `MNEMONIC_PROJECT` or
`loop.ReadActiveProject()`). Success prints `<id> applied` / `<id> revoked` and
exits 0; not-found / no-store prints a clear stderr message and exits 1.

## Gates (exact outputs)

Run from `/Users/paladm/git/ai-test/skillgrid-v2/mnemonic`.

- **G1** `go build ./...` → exit 0 (clean).
- **G2** `go test ./internal/ctx/traffic/... -count=1` → `ok ... internal/ctx/traffic` (6 gate tests).
- **G3** `go test ./internal/ctx/store/... -count=1` → `ok ... internal/ctx/store` (incl. `TestMigration068TrafficMemoryStatus`).
- **G4** `go test ./internal/ctx/proxy/... -count=1` → `ok ... internal/ctx/proxy` (5 context-injection tests).
- **G5** `go vet ./internal/ctx/traffic/... ./internal/ctx/store/... ./internal/ctx/proxy/... ./cmd/...` → exit 0.
- **G6** `gofmt -l` on the 10 touched files → empty (clean).

## Required AC coverage

1. **apply marks promoted + appears in context** — `gate_test.go`
   `TestGate_ApplyPromotesAndIsActive` + `memcontext_test.go` (candidate not in
   context → after apply it is). ✅
2. **revoke reverts + disappears** — `gate_test.go`
   `TestGate_RevokeRevertsAndDisappears` (row kept, status `revoked`,
   `applied_at` NULL) + `memcontext_test.go` (after revoke it is out again). ✅
3. **non-candidate id rejected** — `gate_test.go`
   `TestGate_NotFoundRejected` (Apply/Revoke both → "not found", no row created). ✅
4. **idempotency** — `gate_test.go` `TestGate_Idempotency` (double Apply → one
   active row; double Revoke → no error). ✅
5. **fail-open** — `gate_test.go` `TestGate_NilStoreFailOpen` (nil store →
   no-op, no panic) + `TestGate_Pre068SchemaFailOpen` (068 columns dropped →
   seed succeeds, `ActiveMemories` empty, `Apply` a no-op). ✅

## Deviations from the brief

- **Status vocabulary:** the brief's prose uses `active`/`reverted` as
  adjectives; the concrete column values are `applied`/`revoked`. They are
  internally consistent across the migration, store, gate, proxy, and tests,
  and satisfy the brief's "after apply the rule is active; after revoke it is
  removed" semantics. The `apply`/`revoke` verbs (brief's exact CLI) are
  unchanged.
- **`applied_at` unit:** the brief says "UnixNano"; the store stamps
  `time.Now().UnixMilli()` (millis) — consistent with the rest of the
  `ctx_*` tables' `created_at`/`started_at` millisecond convention. Either is
  a monotonic timestamp; the unit is not load-bearing for the lifecycle.
- **`SetTrafficMemoryStatus` signature:** the brief sketched
  `error`-only; the implementation returns `(bool, error)` so the gate can
  distinguish "row absent" (not-found error) from "table missing" (ADR-0016
  fail-open no-op), which the brief explicitly requires.
- **No new dependencies:** stdlib only (the gate uses `time`/`fmt`). Nothing
  added; no go.work change.

## Open questions

- None. T14 owns wiring the `cfg.Traffic` fields into the learner/`Gate` at
  proxy startup (this ticket provides the `SetGate` seam; the default proxy
  runs with a nil gate → fail-open → no behavior change until wired).
