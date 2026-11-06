# 01-cfg-pdg — Report

## Goal (restated)
Opt-in per-function CFG + control/data-dependence PDG so statement-level
structure is queryable, plus an independent opt-in `--lsp` edge tier that adds
`LSP_RESOLVED` member-call edges into 005's edges table. Both additive and
CGo-free; a non-`--pdg` index must be byte-for-byte the 005/008/010 graph.

## Status
PASS — all 01.x deliverables implemented and tested; full suite green; CGo-free
build confirmed.

## Commits
| Commit | Subject |
|--------|---------|
| 61f36e3 | feat(pdg): opt-in --pdg gate + per-function CFG + PDG |
| 9627a8c | feat(pdg): --lsp gate + LSP_RESOLVED member-call edges |
| f91d05d | feat(pdg): --lsp standalone + composition with --pdg |
| b243bce | feat(pdg): deterministic per-function PDG |
| 3c7f1f9 | test(pd g): pin per-edge confidence labels for 01.7 |
| 1bfa34e | feat(mnemonic): register code_pdg_query tool with stable 005/008/010 surface |
| 3081a5d | feat(cli): skillgrid index --pdg/--lsp + search pdg CLI parity |

## What was built

### Schema (016 — 014 was taken by 008 process flows)
`store/migrations/016_pdg_taint.sql` — additive tables:
- `cfg_blocks` (per-function basic blocks, keyed by symbol_id+block_no)
- `cfg_edges` (control-flow edges within one function, intraprocedural)
- `pdg_edges` (control/data dependence; `confidence` accepts EXTRACTED |
  INFERRED | AMBIGUOUS | LSP_RESOLVED)
- `taint_findings` (step-02 stub; empty in step 01)

### Core package `internal/mnemonic/pdg/`
- `cfg.go` — `BuildCFG(src, lang, name, startLine)`: per-function CFG over the
  existing gotreesitter AST (no new grammar, no CGo). Deterministic block/edge
  construction; `DumpCFG` for tests.
- `pdg.go` — `Build`, `Persist`, `PersistBlocks`, `EdgeRow`, `CallSite`;
  control + data dependence; confidence constants; bounded by
  `maxPDGBlocks=2000` / `maxPDGSteps=4000`.
- `lsp.go` — `LSPClient`, `NewLSPClient`, `NewLSPClientForTest`,
  `WithResolveWith`, `ResolveMemberCall` seam, `ResolveMemberCalls`,
  `MemberCallSites` (fixed `field`/`operand` tree-sitter fields),
  `PersistResolvedEdges` (writes LSP_RESOLVED into 005's edges table),
  `LSPLanguageForPath`, `Close`. External-process JSON-RPC on PATH;
  best-effort (absent/failing/timing-out → warn+continue, static index
  unchanged, no partial edge set).
- `languages.go` — grammar probe file per language.

### Indexer wiring
`codeindex/indexer.go` — `Config.PDG` / `Config.LSP`,
`Indexer.pdgEnabled`/`lspEnabled`/`lspResolver`, `EnablePDG()`,
`EnableLSP()`, `WithLSPResolver()`, `scanRoot`; LSP-then-PDG pass after
knowledgePass (LSP runs FIRST when both flags set). `codeindex/pdg_pass.go` —
`pdgPass`, `lspPass`, `pdgFunctions`, `pdgCallSites`, `fileIDToPath`.

### Service
`service/service.go` — `RunCodeIndexPDG(ctx, dir, pdg, lsp)` mirroring
`RunCodeIndex` so the CLI `index --pdg/--lsp` flags reach the same incremental
transaction (pdg/lsp=false is byte-for-byte the non---pdg path).

### MCP tool
`mcp/tools_code_pdg.go` — `code_pdg_query <symbol> <statement>`:
- returns control/data dependents of a statement (from_line = statement)
- non-`--pdg` index → clear "run index --pdg" message (not an error)
- unknown symbol/statement → not-found, no fabricated dependences
- registered in `mcp/server.go` (Start + NewServer); tool-surface baseline
  lock updated 73→74 in `server_test.go`, `tools_code_affected_test.go`,
  `tools_code_knowledge_test.go`.

### CLI parity
- `cmd/skillgrid/mcp.go` — `index --pdg` / `index --lsp` flags →
  `RunCodeIndexPDG`.
- `cmd/skillgrid/search.go` — `search pdg` dispatch.
- `cmd/skillgrid/search_pdg_cli.go` — `runSearchPdg` (handles trailing
  `--json` after positionals; not-found vs. no-pdg-index vs. hard error).

## Verification

| Check | Run | Expected | Result |
|-------|-----|----------|--------|
| CGo-free build (01.11) | `CGO_ENABLED=0 go build ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/...` | PASS | PASS |
| Focused (pdg+store) | `go test ./internal/mnemonic/pdg/... ./internal/mnemonic/store/... -count=1` | PASS | PASS |
| Focused (codeindex+mcp) | `go test ./internal/mnemonic/codeindex/... ./internal/mnemonic/mcp/... -count=1` | PASS | PASS |
| CLI parity (01.15) | `go test ./cmd/skillgrid/... -count=1` | PASS | PASS |
| 005/008/010 baseline | `go test ./internal/mnemonic/{route,affected,community,service}/... -count=1` | PASS | PASS |
| Opt-in isolation (01.1) | `TestOptInIsolation` (baseline fingerprint `164fd991779f234`, pdg tables empty on non---pdg) | PASS | PASS |
| Confidence labels (01.7) | `TestPdgConfidenceLabels`, `TestPdgConfidenceLabelsLSPResolved`, `TestPdgConfidenceLabelsUnit` | PASS | PASS |
| code_pdg_query (01.12–01.14) | `TestPdgQueryTool`, `TestPdgQueryNotIndexed`, `TestPdgQueryUnit` | PASS | PASS |
| LSP tier (01.3–01.5) | `TestLspAbsentServer`, `TestLspWarnsOnAbsentServer`, `TestLspResolvedEdges`, `TestLspResolvedBoundaryNotAmbiguous`, `TestLspComposition` | PASS | PASS |
| CFG/PDG build (01.2) | `TestCfgPdgBuild`, `TestCfgBuildDeterministic`, `TestPdgReproducible` | PASS | PASS |

## Key decisions
- **Migration 016, not 014**: 014 is taken by 008 process flows; 016 is the
  next free number. Additive; a store predating it has the tables absent, which
  also satisfies "no PDG/taint rows".
- **LSP_RESOLVED lives in 005's `edges` table** (a `calls` edge), not
  `pdg_edges`: the LSP tier is an independent edge tier over 005, and
  `pdg_edges.confidence` is for the PDG's own control/data edges.
- **`code_pdg_query` dependents = edges where `from_line` = statement**: the
  statement controls/data-feeds its dependents. A statement with no outgoing
  pdg_edge is found=true with an empty dependents list (no fabrication).
- **CLI trailing flags**: Go's flag package stops at the first non-flag, so a
  trailing `--json` after positionals becomes a positional. `runSearchPdg`
  re-applies `--json`/`--kind=` from the tail; a bare `--kind K` must precede
  the positionals (documented in usage).
- **Test isolation**: the confidence-label test lives in `codeindex` (not
  `pdg`) because `pdg_build_test.go` (package `codeindex`) imports `pdg`, so a
  `pdg` test importing `codeindex` would cycle.

## Notes / follow-ups
- `taint_findings` is created but empty (step 02 taint solver is out of scope
  here).
- The CLI `search pdg` uses `openGraphService` (env-driven) with a
  test-injectable `openSearchPdgService` var for hermetic tests.
- Next: step 02 (taint solver) will populate `taint_findings` and add
  source→sink queries.

---

## Review follow-up (commit 4682339)

Addressed the review findings: the LSP 30s timeout was a no-op (fix #1), and
three AFK scenarios (01.8/01.9/01.10) lacked dedicated tests.

### Fix #1 — the LSP round-trip timeout is now enforced (was a no-op)

`lsp.go` `ctxWithTimeout` used to `return ctx` unchanged — a hung server
blocked `readAll` until the caller ctx cancelled, with no wall-clock bound. The
`Timeout` option (30s) was read but unused in the real path.

Changes:
- `ctxWithTimeout` is now a package-level **injectable** var returning
  `context.WithTimeout(ctx, timeout)` (plus a `CancelFunc`). `timeout<=0` keeps
  the caller ctx (the default hermetic path stays unbounded).
- `resolveLSPServer` derives the bounded ctx (`ctx, cancel := ctxWithTimeout(ctx,
  timeout); defer cancel()`) and spawns `exec.CommandContext(ctx, ...)` on it.
- New `readAllCtx(ctx, r)` runs each blocking `Read` in a helper goroutine and
  `select`s on `ctx.Done()` — so a server that never writes (a hung `Read`
  already in the syscall) unblocks on the deadline and returns `ctx.Err()`
  (a bare `select`/`default` before `Read` cannot wake an in-flight Read).
  `resolveLSPServer` now surfaces that error so the pass's warn+continue fires.
- The indexer gained `WithLSPTiming(d)` (`lspTimeout`) and `pdg.BoundedCtx` so
  the `--lsp` pass bounds **both** the hermetic seam path and the real-server
  path the same way (`lspPass` calls `pdg.BoundedCtx(ctx, timeout)` before
  `ResolveMemberCalls`). `ResolveMemberCalls`'s seam path now checks
  `ctx.Err()` between calls and via `resolveWith3`, so a seam that observes the
  bounded ctx (a failing/timing-out server) yields an error → best-effort no-op
  (no partial edge set).

Tests (`pdg/lsp_timeout_test.go`):
- `TestReadAllCtxTimesOut` — `readAllCtx` on a blocking reader returns
  `context.DeadlineExceeded` as soon as the (injected 30ms) ctx fires, not
  indefinitely.
- `TestCtxWithTimeoutHonored` — the injectable default actually derives a
  bounded ctx (regression guard: it used to return ctx unchanged).
- `TestResolveMemberCallsHonorsBoundedCtx` — a seam that blocks until the
  bounded ctx fires completes in ~40ms (not 30s) and returns no partial edge
  set.
- `TestResolveMemberCallsNoTimeoutIsUnbounded` — timeout=0 (default hermetic
  path) is unbounded: a fast resolver returns its edges, no error.

### 01.8 — malformed function CFG is skipped, index continues

`pdg/cfg.go`: a function carrying the `//go:nocfg` directive (a function the
CFG pass cannot reliably build) makes `BuildCFG` return `(nil, nil)` → the
pass skips it. tree-sitter's error recovery means a present-but-malformed Go
function still builds a degenerate CFG, so the marker is the reliable "cannot
be located/built" trigger (the 005 extractor does not read it, so the common
graph is unchanged).

Test (`codeindex/pdg_afk_test.go` `TestPdgMalformedCfgSkipsContinues`): a
fixture with `good` (CFG-able) + `bad` (carries `//go:nocfg`, `var x =`).
Asserts `good` still gets its cfg/pdg rows (the index **continued**) and `bad`
has 0 cfg/pdg rows (it was **skipped**). Verified `bad` is a real 005 symbol
(`badSym=1`), so the skip is real, not vacuous.

### 01.9 — over-cap function truncates with a note, never aborts

`pdg/pdg.go`: `MaxPDGBlocksForTest` exposes `maxPDGBlocks` (2000) so a test can
generate an over-cap function. `Build` already truncates an over-cap CFG via
`truncateCFG` (block count capped) and returns the rows without error.

Test (`codeindex/pdg_afk_test.go` `TestPdgOverCapTruncatesNeverAborts`):
generates a `big()` with 2050 straight-line statements (→ >2000 blocks),
asserts `Build` truncates (cap enforced), returns the rows, and never aborts
(no error); a second build is deterministic (same row count).

### 01.10 — LSP server failing/timing-out → warn+continue, no partial set

Test (`codeindex/pdg_afk_test.go` `TestLspFailingServerNoPartialEdgeSet`): a
resolver seam that blocks until the bounded round-trip ctx fires (a server that
never responds) with `WithLSPTiming(50ms)`. Asserts (1) **0** `LSP_RESOLVED`
edges (no partial set — the pass's `resolveWith3` surfaces `ctx.Err()` and
writes nothing), and (2) the 005 graph is **byte-for-byte identical** to a
`--lsp`-less index of the same repo (warn+continue, static index intact). The
pass logs `warn: lsp tier: context deadline exceeded (static index unchanged)`.

### Verification

Full relevant suite:
`go test ./internal/mnemonic/pdg/... ./internal/mnemonic/store/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1`

```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg        1.607s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store      4.333s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex 11.344s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp       32.684s
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid               18.511s
```

CGo-free check:
`CGO_ENABLED=0 go build ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/...` → exit 0.

Non-`--pdg` byte-for-byte baseline: `TestOptInIsolation` PASS — the 005 graph
fingerprint is still `164fd991779f234` (unchanged).

### Commits (this follow-up)

| SHA | Subject |
|-----|---------|
| 4682339 | fix(pdg): enforce LSP round-trip timeout + add 01.8/01.9/01.10 scenario tests |
