# Task Brief — T03 (TASK-090): ctx: config block via ADR-0036 config.d

- Ticket: TASK-090 (type: feature, priority: medium)
- Change: 2026-10-08-context-orchestrator (CTX v1.11)
- Wave 1 · PR1 (foundation) · base = release/2
- Reversibility: reversible (new package, off-by-default consumers)
- Precondition: `findConfigD` shape already proven in `mnemonic/internal/config/pricing.go` (ADR-0036)
- SATISFIES: config-precedence, config-defaults, config-clm-unchanged

## Scope
Create `mnemonic/internal/ctx/config/` that loads a `ctx:` block using the ADR-0036 config.d convention: **builtin → `~/.skillgrid/config.d/ctx.yaml` → first-found walking up**. Reuse the `findConfigD` shape from `mnemonic/internal/config/pricing.go`. Defaults for all keys; per-key merge with repo-local winning. The `clm:` block is STILL read from `.skillgrid/config.yaml` (unchanged) — do not move it.

## CtxConfig keys (all need safe defaults)
port, modes (passthrough/compress), retry, thresholds, wrap agents, judgment, observability. (Mirror the config surface the blueprint specifies for CTX; if a key's exact set is under-specified, use the keys the later tickets reference and default every one.)

## Acceptance Criteria
1. `go test ./internal/ctx/config/...` green: repo-local key wins over machine-local over builtin.
2. Absent repo-local falls to machine-local; neither present → defaults.
3. `clm` block still resolves from `.skillgrid/config.yaml` (assert a sentinel value).

## Definition of Done
- Tests pass for touched packages; lint/format pass; edge cases covered; no new warnings; spec/docs updated if behavior changes.

## Implementation Plan
1. Create `mnemonic/internal/ctx/config/config.go` mirroring the `findConfigD` shape from `mnemonic/internal/config/pricing.go`.
2. Define the `CtxConfig` struct with all keys + safe defaults.
3. Implement `Load(ctx)`: builtin defaults → `~/.skillgrid/config.d/ctx.yaml` → first-found walking up; per-key merge, repo-local wins.
4. Tests: repo-local beats machine-local beats builtin; absent repo-local falls to machine-local; neither → defaults.
5. Sentinel test proving the `clm:` block still resolves from `.skillgrid/config.yaml`.

## Gates
- G1: `cd mnemonic && go build ./...` → EXPECT exit 0
- G2: `cd mnemonic && go test ./internal/ctx/config/...` → EXPECT exit 0
- G3: `cd mnemonic && go test ./internal/config/...` → EXPECT exit 0 (clm resolution unchanged — no regression)
- Fails-when: `go test ./internal/ctx/config/...` exit != 0 OR clm sentinel test fails

## ADR Constraints (do not re-litigate)
- ADR-0036: config.d convention — builtin → `~/.skillgrid/config.d/<name>.yaml` → first-found walking up, repo-local wins.
- ADR-0033: ctx is a `mnemonic` subcommand.

## Owns (edit only under these)
- `mnemonic/internal/ctx/config/{config,load}.go` + test fixtures

## Work Unit Evidence (fill in report)
| Unit | Focused test | Runtime harness | Rollback boundary |
|------|--------------|-----------------|-------------------|
| T03 ctx config | `go test ./internal/ctx/config/...` | N/A (pure config loader, no runtime boundary) | `git revert <sha>` (new internal/ctx/config package) |
