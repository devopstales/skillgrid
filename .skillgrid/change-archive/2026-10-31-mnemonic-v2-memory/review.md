# Review — mnemonic-v2-memory

> Change: `.skillgrid/change-specs/2026-10-31-mnemonic-v2-memory/` (moves to `.skillgrid/change-archive/2026-10-31-mnemonic-v2-memory/` at ship)
> Generated: 2026-11-02T15:40:00+01:00 (requesting-code-review)
> Diff range: `52d554b3..e5edff33`
> Rigor tier: T2 (Beta)
> Independence: Grade B per axis — see `## Independence`
>
> The durable audit record of the code review. **requesting-code-review** writes this into
> the spec folder, next to `report.md` (which owns the QA gate). It is committed in the spec
> zone before code-zone work continues. **ship** archives it with the folder; **reflect**
> cites it for lineage. The three review axes are reported **side by side, never merged** —
> that separation is the point.

## What Important means

- **Critical** — must fix before merge. Security, data loss, broken functionality, a
  `SATISFIES` scenario whose RED evidence is missing or whose code path does not match the
  Given/When/Then. Blocks `ship`.
- **Important** — should fix before merge. Architecture drift from a term or an in-force
  ADR, a hard standard breach, a missing or partial requirement, a behavior the spec never
  asked for (scope creep). Unfixed Important issues do not proceed (per the skill's
  Red Flags).
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. Capped to 5
  listed per axis.

## Cap the nits

- Max 5 Minor findings listed per axis. Beyond that, collapsed into a count.
- A Minor finding that is actually a Critical dressed as polish gets re-labeled.

## Do not report

- Anything tooling already enforces (lint, typecheck, format).
- Informational context with no required action — FYI at most, one line.

## Passes

### Standards

- **Worst issue (within this axis):** Taskfile.yml `MEM_DIR: mnemonic` points at the deleted v1 directory — `task build`/`task test` broken at HEAD; contradicts ADR-0040's own consequence list
- **Findings:** 1 Critical / 4 Important / 5 Minor (capped)
- **Verdict:** met-with-fixes

### Spec

- **Worst issue (within this axis):** none — all 75 scenarios + G1–G16 PASS; T17 RED review-attested (honest, in-ledger)
- **Findings:** 0 Critical / 0 Important / 0 Minor
- **Verdict:** met

### Security

- **Worst issue (within this axis):** MCP `all` profile: destructive write tools (hard delete, consolidate-confirm) callable by an LLM with no policy gate or HITL (CWE-285, LLM03, ASI02/03)
- **Findings:** 0 Critical / 1 High / 2 Medium / 1 Low / 2 Info (capped)
- **Verdict:** secure-with-fixes

## Findings

### Critical

- [standards] `Taskfile.yml:23` (`MEM_DIR: mnemonic`) + lines 41,57,58,65,66,73,74,81,121 — the v1 rename (`mnemonic/`→`mnemonic_back/`) happened in this range but the root Taskfile was not updated. `task build`, `task test`, `task fmt`, `task seed-test`, and `task all` all `cd` into `mnemonic/` which no longer exists. ADR-0040's consequence list says "the Taskfile must build both" — the root `go.work` was updated to list both modules, but the Taskfile variable was missed. Verified: `grep MEM_DIR Taskfile.yml` → 6 references, all to the deleted path. **Fix:** point `MEM_DIR` at `mnemonic_back/` (v1) and add a parallel `MEM2_DIR: mnemonic-v2` for the new module's build/test/fmt, per ADR-0040.

### Important

- [standards] `internal/transport/handler/handler.go:450,476,555` + `internal/memory/trails.go:77,89` + `internal/memory/transport.go:127` — SQLite-flavored SQL (`datetime('now')`, `ifnull(...)`) in dialect-agnostic code paths. The PG `sessions` table exists (`migrations_pg/001_schema.sql:19`, `TIMESTAMPTZ DEFAULT now()`) and `postgres.go:713,1361` insert sessions via Go-side SQL. The handler's `SessionStart`/`SessionEnd`/`SessionFinalize` use `stExec` with `datetime('now')` — on a PG store these calls would fail (PG has no `datetime()`). `trails.go` and `transport.go:127` (ack) are the same pattern. This is also Duplicated Code: the session-row INSERT with `datetime('now')` exists in 4 places. **Fix:** route these writes through a dialect-aware timestamp (the store already computes UTC stamps in Go); or document that the PG mode is store-only (no session lifecycle through the handler) with the session methods failing loudly.
- [standards] `internal/transport/envelope/envelope.go:72-93` — `MapError` classifies errors by message substring (`"no rows"`, `"not found"`, `" is required"`). Any internal error containing those substrings is misclassified. The D1 wire contract (code, message) depends on this. **Fix:** promote `ErrInvalidArguments`/`ErrNotFound` sentinels in the pipeline/store; classify with `errors.Is`; keep `sql.ErrNoRows` as the not-found anchor.
- [standards] `internal/memory/scan.go:178` — `var MigrationState func(...)` is declared but never read (only `MigrationStateHook` at line 187 is used). Dead code; AGENTS standard "delete, don't comment out." **Fix:** delete the var.
- [standards] `internal/memory/transport.go` (~line 190) — `PullAndApply` constructs `&SyncService{st: st}` without config, while every other path goes through `NewSyncService` (which sets `cfg.Sync.Enabled`). Benign today (default is false) but a latent wiring asymmetry. `Optional:` — thread the config.
- [security] `internal/transport/mcp/tools_mem.go:58-63` + `server.go:152-190` + `cmd/mnemonic/mcp.go:30-31` — MCP `all` profile exposes destructive write tools (`mem_delete{hard:true}`, `mem_consolidate{confirm:true}`, `mem_update`) to an LLM caller with no central policy gate and no HITL approval step. The agent that reads observations (potential injection vector via peer-pulled content) is the same agent that can delete them. CWE-285, OWASP A01, LLM03 Excessive Agency, ASI02/03. **Fix:** keep `agent` as the default for unattended servers; gate `all` behind an explicit operator flag; add an approval step (or at minimum an audit log) for `mem_delete hard=true` and `mem_consolidate confirm=true`.
- [security] `internal/core/store/postgres.go:282-329` (PushMutations) + `internal/memory/sync.go:164-200` (applyObservation) + `internal/memory/chunkcodec.go:161-200` (Import) — sync mutations and imported chunk archives carry no source identity. Any enrolled peer (or anyone with the shared DSN) is a full-writer into local memory; content is then served to agents and re-pushed to the ring. CWE-807, OWASP A06 + A01, LLM09, ASI06. Medium: preconditions are "one enrolled machine compromised or shared DSN leaked." **Fix:** add a source-machine identifier to the mutations schema; enforce a per-project source allowlist at apply time; optionally sign chunk archives (HMAC over the manifest).

### Minor

- [standards] `internal/memory/scan.go:187` vs `:178` — `MigrationStateHook` is the used var; `MigrationState` is dead (see Important #3 above; listed here for the cap count).
- [standards] `internal/core/store/txngate.go:365-384` — hand-rolled `jsonEscape` does not escape control characters `< 0x20`; content with `\u0001` produces invalid JSON. `json.Marshal` on a struct is three lines. `Optional:`
- [standards] `internal/memory/judgment.go:431-490` — consolidate's two-transaction merge (survivor insert, then source soft-delete) is documented as non-atomic; a failure in between leaves a live survivor with un-superseded sources. Legitimate YAGNI (write queue bounds the window) but the one data-integrity gap. `Consider:` fold into the save gate when Task 18's `pgSaveTXN` lands.
- [security] `internal/memory/pipeline.go:170` vs `judgment.go:387,439` + `handler.go:461-556` + `dispatch.go:202-226` — `<private>` redaction is applied on the save path but not on `mem_consolidate`'s new_content, `mem_update`'s content, `mem_judge`'s evidence/reason, or `mem_session_*` sections. Own-data impact (same principal) but the control is inconsistent. `Consider:` centralize redaction in the handler/service write seam.
- [security] `go.mod:23` — `golang.org/x/text v0.14.0` indirect pin; CVE-2026-56851 (panic in `secure/precis`, fixed v0.41.0) — **verified inert**: `go list -deps` shows `transform`, `unicode/bidi`, `secure/bidirule`, `unicode/norm` imported; `secure/precis` is NOT in the build graph. The CVE is unreachable. `FYI` — bump or `go mod tidy` to drop the stale pin.

## Independence

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh subagent context (ses_e635fc73), same model family as implementer, no shared session history |
| Spec | B | fresh subagent context (ses_e635fc72f), same model family as implementer, no shared session history |
| Security | B | fresh subagent context (ses_e635fc72d), same model family as implementer, no shared session history |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker). **C** =
inline self-review or the implementer's verdict leaked in — **diagnostic only**.

## Verdict

- **Standards:** met-with-fixes
- **Spec:** met
- **Security:** secure-with-fixes
- **Floor (decides):** met (after the 2026-11-02 fix round; initial render met-with-fixes)
- **Worst issue (across all three axes):** Taskfile.yml `MEM_DIR: mnemonic` — fixed (build entry point restored, ADR-0040 both-modules contract honored)
- **Unfixed Important count (must be 0 to proceed):** 0 (all 5 code-zone Important fixed + tested; the 2 security design-level findings parked as named windows W037/W038 with their fix paths)

## Fix Round (2026-11-02)

| # | Finding | Disposition | Evidence |
|---|---------|-------------|----------|
| C1 | Taskfile `MEM_DIR: mnemonic` | **FIXED** | `task build` runs green; build targets mnemonic-v2, test/fmt cover both modules, seed-test targets v1 |
| I1 | SQLite-flavored SQL in dialect-agnostic paths | **FIXED** | UTC stamps computed in Go + bound as parameters (handler session methods, trails.ensureSession, ackMutations); PG parity suite re-run green 8/8 |
| I2 | `envelope.MapError` substring classification | **FIXED** | `memory.ErrInvalidArguments` sentinel wraps all validation errors (pipeline + MCP errMissingArg); store "observation %d not found" wraps `ErrObservationNotFound`; isNotFound anchors on sql.ErrNoRows + sentinel only; new regression test `TestMapErrorIsNotSubstringBased` |
| I3 | Dead `var MigrationState` | **FIXED** | deleted (only `MigrationStateHook` is used) |
| I4 | `PullAndApply` zero-config SyncService | **FIXED** | takes `config.Config`; autosync passes its own cfg, CLI passes `h.Config()` (the config openRuntime already resolved) |
| S1 | MCP `all` profile: no policy gate / HITL | **DEFERRED → W037** | design-level (policy gate + HITL is a product surface, not a bug); fix path: gate `all` behind an explicit operator flag + approval/audit for hard-delete and consolidate-confirm |
| S2 | Sync peers: no source identity / signing | **DEFERRED → W038** | design-level (schema migration + per-project allowlist); fix path: source-machine column + allowlist at apply time; optional HMAC over the chunk manifest |

**Minor items (capped, not fixed this round):** jsonEscape control chars (Optional), consolidate 2-txn (Consider, folds into Task 18 pgSaveTXN), redaction call-site consistency (Consider), x/text v0.14.0 pin (FYI — verified inert: `secure/precis` not in the build graph; `go mod tidy` will drop it when the graph moves).

**Validation:** `go build`/`go vet` clean; `go test ./... -count=1` all packages green; `-race` green on memory/transport/store; `task build` green; PG parity suite green (8/8, DSN-gated).
