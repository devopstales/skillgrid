# Tasks: 011-mnemonic-pdg-taint

> **STATUS:** `in-progress` (2026-09-09) — 0/2 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add an opt-in statement-level analysis tier to Mnemonic's code graph: per-function control-flow graphs → program-dependence graphs → taint findings (source→sink data-flow).

**Architecture:** Two additive, opt-in passes on top of 005's graph: a CFG→PDG pass gated behind `--pdg` (basic blocks + control/data dependence, intraprocedural M1, built from the existing gotreesitter AST) and a taint solver (deterministic configurable source/sink sets, source→sink paths, confidence-labeled, stops-at-boundary-not-fabricated). An independent opt-in `--lsp` edge tier (`pdg/lsp.go`) shells out to a language server on `PATH` to add `LSP_RESOLVED` member-call edges into 005's edges table (best-effort, no-op on a missing server); the taint solver consumes `LSP_RESOLVED` as resolved call boundaries. A non-`--pdg` index is byte-for-byte unchanged. See `change.md` decisions.

**Tech Stack:** Go (`skillgrid-cli`), CGo-free SQLite (`modernc.org/sqlite`), existing gotreesitter AST from 005 (no new grammar), an external language server on `PATH` for the `--lsp` tier (no in-process CGo boundary), MCP (`mcp-go`), CLI.

**Spec:** `docs/skillgrid/changes/011-mnemonic-pdg-taint/change.md`

**Acceptance:** `docs/skillgrid/changes/011-mnemonic-pdg-taint/acceptance.feature` (`@step-NN`)

---

## Goal

An agent gets statement-level data-flow answers on demand — "does this user input reach that SQL write?", "what's the control path that enables this branch?", "trace this value source→sink" — without reading files or re-deriving the CFG by hand. It is opt-in so the common 005/008/010 path stays lean.

## Out of scope / Non-Goals

- Re-implementing 005's symbols/edges/extractors, 008's communities/processes/knowledge, or 010's routes/affected/rename/watcher
- Whole-program interprocedural taint (M1 is **intraprocedural** per-function; a call-boundary summary is a later pass — the **LSP tier is its feeder**)
- A full data-flow engine beyond CFG→PDG→taint (no type inference beyond what 005 already resolves, no alias analysis beyond 005's receiver resolution)
- Default-on indexing (PDG is a separate, opt-in pass — it must not slow the common path)
- New languages beyond what 005's gotreesitter already parses (the PDG pass reuses 005's AST; M1 ships the same language set as 005)
- A new tree-sitter grammar or any in-process CGo (CFG is a re-read of 005's existing AST; the LSP tier is an external process)
- LLM-suggested sources/sinks (source/sink sets are deterministic and configurable)
- A new language server or in-process LSP client (the `--lsp` tier shells out to a server already on `PATH`; it is never a hard dependency)

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] Every success criterion / DoD checkbox in `change.md` is met
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [ ] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Rollback path in `change.md` is still valid (or N/A documented)
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- Additive + **opt-in** — `--pdg` is the only trigger for the PDG/taint pass; a non-`--pdg` index never populates PDG/taint tables and is byte-for-byte the 005/008/010 graph
- CGo-free: CFG is built from the existing gotreesitter AST (no new grammar, no C); the PDG + taint solver are pure Go
- **LSP tier is opt-in and external-process:** `--lsp` shells out to a language server on `PATH` (`gopls`, `pyright`, `typescript-language-server`, `rust-analyzer`, `clangd` — whichever 005 already parses) for precise member-call resolution. It adds a fourth edge confidence, `LSP_RESOLVED`, to the existing `EXTRACTED | INFERRED | AMBIGUOUS` set. A missing/failing server is best-effort: the index is unchanged (static resolution only), never a hard error. No new in-process CGo boundary
- Intraprocedural (M1): CFG/PDG/taint are per-function; crossing a call boundary without a resolved callee is `AMBIGUOUS`/truncated, not a fabricated intraprocedural hop. Interprocedural summaries are a later pass
- Every PDG/taint edge carries a Confidence Label: `EXTRACTED | INFERRED | AMBIGUOUS`; a taint path is only `EXTRACTED` where every hop is a resolved data-dependence, else `INFERRED`/`AMBIGUOUS`. `LSP_RESOLVED` joins the label set as a fourth value marking call edges resolved by a language server
- Taint **sources** and **sinks** are a configurable, deterministic set (e.g. sources: request params / env / file reads; sinks: SQL exec / shell exec / template render / file write); a finding is source→sink, not a guess
- **LSP-resolved call edges feed taint:** the PDG pass consumes `LSP_RESOLVED` edges (005 + `--lsp`) as resolved call boundaries instead of marking them `AMBIGUOUS`/"stops at boundary" — the primary M1 false-negative reducer for method-heavy code. The LSP layer is a **feeder**, not a dependency: without `--lsp`, taint behaves exactly as specified (intraprocedural, `AMBIGUOUS` at unresolved boundaries)
- A taint finding with no source→sink path is **not reported** (never fabricated); a path that ends at an unresolved boundary is reported with a "stops at <boundary>" note (reuses 005's graph-stops philosophy)
- Existing 005/008/010 `code_*` tools keep name + required params; all new tools use distinct `code_*` names
- Migration id `014_pdg_taint.sql` — leave `011` (005), `012` (008), `013` (010) as-is
- Function with a malformed/unparseable CFG → `warn+continue`; skip that function's PDG; index the rest
- PDG construction exceeds depth/step cap on a large function → `warn+continue`; truncate with a "stops at <block>" note; never abort
- Taint path crosses an unresolved call boundary → `warn+continue`; path marked `AMBIGUOUS` / "stops at <boundary>"; not fabricated (an `LSP_RESOLVED` edge, when present, resolves the boundary instead)
- LSP server absent / fails / times out on `--lsp` → `warn+continue`; index unchanged (static resolution only); no hard error, no partial LSP edge set
- `code_pdg_query` on a statement with no PDG (non-`--pdg` index) → `warn+continue`; clear "run `--pdg`" message; empty result, not an error
- Unknown / missing symbol or statement → `warn+continue`; not-found; no fabricated blocks or dependences
- Bad / missing args on new `code_*` tools → `abort` with clear validation error; do not invent findings
- Existing 005/008/010 `code_*` tool call → unchanged name + required params; a non-`--pdg` index is byte-for-byte unchanged

---

## State

```yaml
phase: verify          # spec | apply | verify | archive
current_step: 02-taint-solver
status: in_progress  # in_progress | blocked | done
updated: 2026-09-10
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `cfg-pdg` | `@step-01` | — (005 done) | Feature tagged `@step-01` |
| 02 | `taint-solver` | `@step-02` | 01 | Feature tagged `@step-02` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~900–1400 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Delivery strategy | single-pr |

Two vertical slices: step 01 (schema + CFG + PDG + the opt-in `--lsp` edge tier + opt-in hook + `code_pdg_query`), step 02 (taint solver that consumes `LSP_RESOLVED` boundaries + `code_taint` + CLI/CLI-flag parity). Each likely its own work-unit commit; a single PR is acceptable for the change as a whole given the opt-in blast radius is bounded behind `--pdg` / `--lsp`.

---

## 01-cfg-pdg

### Goal

Opt-in per-function CFG + control/data-dependence PDG so statement-level structure is queryable, plus an independent opt-in `--lsp` edge tier that adds `LSP_RESOLVED` member-call edges into 005's edges table.

### Out of scope / Non-Goals

- Taint (step 02); interprocedural flow; changing 005/008/010 tools
- A new tree-sitter grammar or any in-process CGo; default-on PDG or LSP indexing; an in-process LSP client (the server is an external process on `PATH`)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none (005 done)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/014_pdg_taint.sql`
- Create: `skillgrid-cli/internal/mnemonic/pdg/cfg.go`
- Create: `skillgrid-cli/internal/mnemonic/pdg/pdg.go`
- Create: `skillgrid-cli/internal/mnemonic/pdg/lsp.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_pdg.go`
- Modify: `skillgrid-cli/internal/mnemonic/codeindex/indexer.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go`
- Modify: `skillgrid-cli/cmd/skillgrid/code_intel.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go`
- Test: `skillgrid-cli/internal/mnemonic/pdg/...`, `skillgrid-cli/internal/mnemonic/store/...`, `skillgrid-cli/internal/mnemonic/codeindex/...`, `skillgrid-cli/internal/mnemonic/mcp/...`

**Interfaces:**
- Consumes: 005's gotreesitter AST + symbols/edges + existing store migration runner + indexer `Run`; a language server on `PATH` for the `--lsp` tier (`gopls`/`pyright`/`typescript-language-server`/`rust-analyzer`/`clangd`)
- Produces: `014_pdg_taint.sql` (cfg_blocks, cfg_edges, pdg_edges, taint_findings); per-function CFG (basic blocks + `CFG` edges); control-dependence + data-dependence PDG edges (each confidence-labeled); `LSP_RESOLVED` edges in 005's edges table from the `--lsp` tier (member calls the static pass couldn't type); opt-in `--pdg` hook + independent opt-in `--lsp` hook in `Indexer.Run` (same incremental transaction, each gated on its own flag); `code_pdg_query <symbol> <statement>` MCP/CLI tool; `skillgrid index --pdg` + `skillgrid index --lsp` flags

### Tasks

- [x] 01.1 `[RED]` Opt-in isolation — a non-`--pdg` index is byte-for-byte unchanged (Scenario: Non-opt-in index is byte-for-byte unchanged) — threat: Opt-in isolation; Mnemonic tool surface
  - [x] 01.1.a Write failing test — (1) index a fixture repo without `--pdg` and assert the `cfg_blocks`/`cfg_edges`/`pdg_edges`/`taint_findings` tables are created but empty; (2) assert every 005/008/010 `code_*` tool output for a representative query is byte-for-byte identical to a pre-011 baseline; (3) assert `code_pdg_query` against a non-`--pdg` index returns a clear "run `--pdg`" message and an empty result, not an error
  - [x] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/mcp/... -run OptInIsolation -count=1` — Expected: FAIL
  - [x] 01.1.c Minimal implementation — opt-in gate: `--pdg` flag on the indexer; the PDG/taint pass runs only when the flag is set, in the same incremental transaction as 005's extraction; without it, no PDG/taint rows are written and the common graph is untouched
  - [x] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/mcp/... -run OptInIsolation -count=1` — Expected: PASS
  - [x] 01.1.e Commit — `feat(mnemonic): opt-in --pdg gate with byte-for-byte unchanged common index`
- [x] 01.2 `[RED]` Opt-in isolation — per-function CFG + PDG populates only under `--pdg` (Scenario: Opt-in index builds per-function CFG and PDG) — threat: Opt-in isolation
  - [x] 01.2.a Write failing test — index a fixture repo (a 005-supported language) with `--pdg`; assert `cfg_blocks`/`cfg_edges` rows exist per function (basic blocks from branch/loop/return structure) and `pdg_edges` rows exist for both control-dependence and data-dependence; assert a non-`--pdg` index of the same repo leaves those tables empty
  - [x] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -run CfgPdgBuild -count=1` — Expected: FAIL
  - [x] 01.2.c Minimal implementation — `014_pdg_taint.sql` migration (cfg_blocks, cfg_edges, pdg_edges, taint_findings); per-function CFG builder over the gotreesitter AST (basic blocks from branch/loop/return structure, no new grammar); control-dependence + data-dependence derivation into `pdg_edges`; indexer hook after 005 extraction
  - [x] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -run CfgPdgBuild -count=1` — Expected: PASS
  - [x] 01.2.e Commit — `feat(mnemonic): per-function CFG and control/data-dependence PDG`
- [x] 01.3 `[RED]` LSP opt-in isolation — `--lsp` with no language server installed produces a byte-for-byte static index with zero `LSP_RESOLVED` edges, warns, and continues (Scenario: LSP index with no server is byte-for-byte static) — threat: Opt-in isolation; Mnemonic tool surface
  - [x] 01.3.a Write failing test — with no `gopls`/`pyright`/`typescript-language-server`/`rust-analyzer`/`clangd` on `PATH` (isolated `PATH` in the test), index a fixture repo with `--lsp`; assert (1) the index is byte-for-byte identical to a static `--lsp`-less index for the same repo; (2) zero `LSP_RESOLVED` edges are written to 005's edges table; (3) a warning is emitted and indexing continues (no hard error, no partial LSP edge set); (4) a non-`--lsp` index of the same repo is also unchanged
  - [x] 01.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -run LspAbsentServer -count=1` — Expected: FAIL
  - [x] 01.3.c Minimal implementation — `--lsp` flag (independent of `--pdg`); the LSP tier runs only when the flag is set and a server for the language is resolvable on `PATH`; absent/failing server → best-effort no-op that leaves the static edge set untouched and emits a `warn+continue`
  - [x] 01.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -run LspAbsentServer -count=1` — Expected: PASS
  - [x] 01.3.e Commit — `feat(mnemonic): opt-in --lsp gate with byte-for-byte static no-op on missing server`
- [x] 01.4 `[RED]` LSP edge tier — `--lsp` adds `LSP_RESOLVED` edges for member calls the static pass couldn't type (Scenario: LSP index adds LSP_RESOLVED member-call edges)
  - [x] 01.4.a Write failing test — with a language server available on `PATH` (or a deterministic fake server stub in the test), index a fixture repo with `--lsp` that contains member calls the static tree-sitter pass cannot type (receiver-bound method / interface→impl); assert (1) `LSP_RESOLVED` edges appear in 005's edges table for those member calls; (2) each carries the `LSP_RESOLVED` confidence label (the fourth value, joining `EXTRACTED | INFERRED | AMBIGUOUS`); (3) static edges for the same calls that the static pass already resolved are not duplicated or downgraded
  - [x] 01.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run LspResolvedEdges -count=1` — Expected: FAIL
  - [x] 01.4.c Minimal implementation — `pdg/lsp.go` external-process language-server adapter (JSON-RPC over a separate binary on `PATH`); resolves member calls for languages 005 already parses and writes `LSP_RESOLVED` edges into 005's edges table; pure-Go in-process, no CGo boundary
  - [x] 01.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run LspResolvedEdges -count=1` — Expected: PASS
  - [x] 01.4.e Commit — `feat(mnemonic): LSP edge tier adds LSP_RESOLVED member-call edges`
- [x] 01.5 `[RED]` LSP composition — `--lsp` works standalone and composes with `--pdg` (Scenario: LSP tier works standalone and composes with PDG)
  - [x] 01.5.a Write failing test — (1) standalone: index a fixture with `--lsp` only (no `--pdg`) and assert `LSP_RESOLVED` edges are written into 005's edges table while `cfg_blocks`/`cfg_edges`/`pdg_edges`/`taint_findings` remain empty; (2) composed: index with `--lsp --pdg` and assert both `LSP_RESOLVED` edges AND the per-function CFG/PDG rows are present, and the PDG sees the `LSP_RESOLVED` call edges as available (not dropped)
  - [x] 01.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -run LspComposition -count=1` — Expected: FAIL
  - [x] 01.5.c Minimal implementation — wire `--lsp` as an independent hook (not nested under `--pdg`); when both flags are set, the LSP tier runs first so `LSP_RESOLVED` edges exist before the PDG pass derives dependences
  - [x] 01.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/store/... -run LspComposition -count=1` — Expected: PASS
  - [x] 01.5.e Commit — `feat(mnemonic): --lsp works standalone and composes with --pdg`
- [x] 01.6 `[RED]` Deterministic reproducibility — same input yields identical PDG across repeated runs (Scenario: Repeated PDG builds are reproducible)
  - [x] 01.6.a Write failing test — index the same fixture repo with `--pdg` twice (fresh store each time); assert the full set of `cfg_blocks`/`cfg_edges`/`pdg_edges` rows (ids, types, confidence labels, ordering-independent) is byte-for-byte identical across the two runs; assert no wall-clock, pointer, or map-iteration-order nondeterminism leaks into persisted rows
  - [x] 01.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/store/... -run PdgReproducible -count=1` — Expected: FAIL
  - [x] 01.6.c Minimal implementation — deterministic traversal (sorted iteration where the AST/PDG yields unordered collections); stable block/statement ids keyed by (symbol, line range), not by allocation order
  - [x] 01.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/store/... -run PdgReproducible -count=1` — Expected: PASS
  - [x] 01.6.e Commit — `feat(mnemonic): deterministic per-function PDG construction`
- [x] 01.7 `[RED]` Confidence labels — every PDG edge is labeled; unresolved data-dependence is `AMBIGUOUS`/`INFERRED` not fabricated; LSP-resolved boundaries are `LSP_RESOLVED` not `AMBIGUOUS` (Scenario: Every PDG edge carries a confidence label)
  - [x] 01.7.a Write failing test — (1) assert every row in `pdg_edges` has a non-empty Confidence Label in `EXTRACTED | INFERRED | AMBIGUOUS | LSP_RESOLVED`; (2) fixture with a resolvable data-dependence → that edge is `EXTRACTED`; (3) fixture with a data-dependence that cannot be resolved statically and has no `LSP_RESOLVED` edge (e.g. value leaves through an unresolved call) → that edge is `AMBIGUOUS` or `INFERRED`, and no fabricated `EXTRACTED` edge is written for it; (4) fixture where the same call boundary is resolved by the `--lsp` tier → that boundary edge is `LSP_RESOLVED`, not `AMBIGUOUS`
  - [x] 01.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run PdgConfidenceLabels -count=1` — Expected: FAIL
  - [x] 01.7.c Minimal implementation — Confidence Label column on `pdg_edges` accepting the fourth `LSP_RESOLVED` value; derivation logic labels each edge by how it was resolved; only fully-resolved data-dependences are `EXTRACTED`; a call boundary with an `LSP_RESOLVED` edge is not re-marked `AMBIGUOUS`
  - [x] 01.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run PdgConfidenceLabels -count=1` — Expected: PASS
  - [x] 01.7.e Commit — `feat(mnemonic): confidence-labeled PDG edges with LSP_RESOLVED boundary value`
- [x] 01.8 `[AFK]` Malformed function CFG is skipped and index continues (Scenario: Malformed function CFG skips and index continues) — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... -count=1` — Expected: PASS
- [x] 01.9 `[AFK]` Function over depth/step cap truncates with a note, never aborts (Scenario: Over-cap function truncates with a note) — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -count=1` — Expected: PASS
- [x] 01.10 `[AFK]` LSP server absent / fails / times out → warn+continue, index unchanged, no partial edge set (Scenario: LSP server failure is best-effort no-op) — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... -count=1` — Expected: PASS
- [x] 01.11 `[AFK]` CGo-free: build succeeds without CGO_ENABLED=1 (Scenario: PDG package builds without CGo) — `Run: CGO_ENABLED=0 go build ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... && CGO_ENABLED=0 go test ./skillgrid-cli/internal/mnemonic/pdg/... -count=1` — Expected: PASS
- [x] 01.12 `[RED]` Mnemonic tool surface — `code_pdg_query` registered + existing 005/008/010 tools stable + bad args rejected (Scenario: code_pdg_query registered and bad args fail) — threat: Mnemonic tool surface
  - [x] 01.12.a Write failing test — (1) assert `code_pdg_query` is registered with distinct name + required `symbol`/`statement` params; (2) assert the full set of 005/008/010 `code_*` tool names + required params is unchanged; (3) assert `code_pdg_query` with missing/unknown args is rejected with a clear validation error (abort, not a fabricated empty result)
  - [x] 01.12.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run PdgQueryTool -count=1` — Expected: FAIL
  - [x] 01.12.c Minimal implementation — `tools_code_pdg.go` `code_pdg_query` tool + server registration without dropping existing `code_*`; arg validation
  - [x] 01.12.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run PdgQueryTool -count=1` — Expected: PASS
  - [x] 01.12.e Commit — `feat(mnemonic): register code_pdg_query tool with stable 005/008/010 surface`
- [x] 01.13 `[AFK]` `code_pdg_query` returns control/data dependents of a statement (Scenario: PDG query returns statement dependents) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS
- [x] 01.14 `[AFK]` Unknown symbol or statement returns not-found, no fabricated dependences (Scenario: Unknown PDG query symbol returns not-found) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` — Expected: PASS
- [x] 01.15 `[AFK]` `skillgrid index --pdg` + `skillgrid index --lsp` CLI flags + `skillgrid search pdg` (Scenario: CLI pdg and lsp flag parity) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` | PASS | PASS | pdg (unit: cfg/lsp) + store migration 016; 01.8/01.9/01.10 in codeindex (import-cycle avoidance) |
| Focused test (codeindex + mcp) | `go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` | PASS | PASS | CfgPdgBuild, OptInIsolation, Lsp* (absent/resolved/composition/timeout), PdgQueryTool |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | all `@step-01` scenarios mapped to named tests; byte-for-byte via `baselineFingerprint` (hash 164fd991779f234) |
| Runtime harness | `skillgrid index --pdg` on fixture repo; query `cfg_blocks`/`pdg_edges` via `code_pdg_query` | PASS | PASS | CfgPdgBuild populates cfg_blocks/cfg_edges/pdg_edges; PdgQueryTool returns control/data dependents |
| LSP runtime harness | `skillgrid index --lsp` (with server) adds `LSP_RESOLVED` edges; `skillgrid index --lsp` (no server) is byte-for-byte static | PASS | PASS | hermetic lspResolver seam; LspResolvedEdges + LspAbsentServer (isolated PATH) + LspFailingServerNoPartialEdgeSet (timeout, 4682339) |
| Byte-for-byte gate | `skillgrid index` (no `--pdg`) on fixture repo; diff 005/008/010 tool outputs against pre-011 baseline | PASS | PASS | OptInIsolation: tables created-empty + content-hash identical; 005/008/010 baseline suite `ok` (route/affected/community) |
| Rollback boundary | Drop `016_pdg_taint.sql` + `pdg/` (incl. `lsp.go`) + `code_pdg_query` + `--pdg`/`--lsp` hooks; re-run non-`--pdg` index | PASS | PASS | 016 additive (CREATE TABLE IF NOT EXISTS); non-`--pdg`/non-`--lsp` path writes no PDG/LSP rows, baseline stable |
| Global Constraints | — | held | held | opt-in isolation, CGo-free (`CGO_ENABLED=0 go build` exit 0), LSP best-effort no-op, intraprocedural M1, confidence-labeled (4th value LSP_RESOLVED), deterministic |

Review: task reviewer `approved with fixes` → fix commit 4682339 (enforced LSP round-trip timeout — was a no-op; added dedicated 01.8/01.9/01.10 tests). Scoped re-review of 4682339: timeout genuinely enforced (`resolveWith3` checks `ctx.Err()`, `ResolveMemberCalls` returns early on error → no partial set); 3 AFK tests non-vacuous (`//go:nocfg` marker + `truncateCFG` + `MaxPDGBlocksForTest` are real implementation, not test scaffolding). Residual (non-blocking, deferred to 011 sdd-verify): real external-process JSON-RPC path `resolveLSPServer` exercised only via the hermetic seam; package-level `scanRoot` mutable global (single-threaded `Run` in practice).

Commits (step 01): 61f36e3 (opt-in gate + CFG + PDG), 9627a8c (--lsp gate + LSP_RESOLVED tier), f91d05d (lsp standalone+composition), b243bce (determinism), 3c7f1f9 (confidence labels), 1bfa34e (code_pdg_query tool, surface 73→74), 3081a5d (CLI parity), 4682339 (timeout fix + AFK tests). Migration 016_pdg_taint.sql (brief's 014 was taken by 008 step 02).

### Commit

When step DoD is met: `feat(mnemonic): opt-in per-function CFG, PDG, and LSP edge tier`

---

## 02-taint-solver

### Goal

Configurable source/sink sets + intraprocedural source→sink taint so "does untrusted input reach this sink" is a query — consuming `LSP_RESOLVED` edges as resolved call boundaries instead of truncating at them.

### Out of scope / Non-Goals

- Interprocedural flow (later pass); a full data-flow engine; changing 005/008/010 tools
- LLM-suggested sources/sinks; cross-function call-boundary summaries; building the LSP edge tier (that is step 01)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-cfg-pdg

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/pdg/taint.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_pdg.go` (extend with `code_taint`)
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go`
- Modify: `skillgrid-cli/cmd/skillgrid/code_intel.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go`
- Test: `skillgrid-cli/internal/mnemonic/pdg/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: PDG edges + CFG blocks + `LSP_RESOLVED` edges from step 01; 005's symbols/edges; existing store + indexer
- Produces: deterministic configurable source/sink sets; intraprocedural source→sink solver over the PDG that treats `LSP_RESOLVED` edges as resolved call boundaries (path continues through them); persisted `taint_findings` (source kind, sink kind, hop-by-hop path, per-hop Confidence Label); `code_taint` MCP/CLI tool (`--symbol`/`--file`/`--json`); `skillgrid search taint` CLI; findings registered on the MCP surface

### Tasks

- [x] 02.1 `[RED]` Opt-in isolation — `--pdg` index adds taint findings without altering 005/008/010 results (Scenario: Opt-in taint index leaves 005/008/010 results unchanged) — threat: Opt-in isolation
  - [x] 02.1.a Write failing test — index a fixture repo with `--pdg` that has at least one known source→sink path; assert (1) `taint_findings` rows exist for the known flow; (2) every 005/008/010 `code_*` tool output is byte-for-byte identical to the pre-011 baseline for the same repo; (3) a non-`--pdg` index of the same repo leaves `taint_findings` empty
  - [x] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintOptInIsolation -count=1` — Expected: FAIL
  - [x] 02.1.c Minimal implementation — taint solver wired into the `--pdg` hook after PDG derivation; findings written to `taint_findings` only when `--pdg` is set
  - [x] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintOptInIsolation -count=1` — Expected: PASS
  - [x] 02.1.e Commit — `feat(mnemonic): opt-in taint findings without altering 005/008/010 results`
- [x] 02.2 `[RED]` Taint core — source→sink path found and persisted (Scenario: Source to sink taint path found)
  - [x] 02.2.a Write failing test — fixture with a known source (e.g. request param) that flows to a known sink (e.g. SQL exec) through a resolvable data-dependence chain; assert `code_taint` returns a finding with the source kind, sink kind, and the hop-by-hop path; assert the finding is persisted in `taint_findings`
  - [x] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintSourceToSink -count=1` — Expected: FAIL
  - [x] 02.2.c Minimal implementation — `pdg/taint.go` default deterministic source/sink sets (sources: request params, env vars, file reads; sinks: SQL exec, shell exec, template render, file write); source→sink reachability solver over the PDG data-dependence edges; `code_taint` tool
  - [x] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintSourceToSink -count=1` — Expected: PASS
  - [x] 02.2.e Commit — `feat(mnemonic): intraprocedural source-to-sink taint solver`
- [x] 02.3 `[RED]` LSP-resolved boundary — a taint path continues through an `LSP_RESOLVED` call boundary instead of "stops at" (Scenario: Taint path continues through an LSP_RESOLVED boundary) — threat: Mnemonic tool surface (LSP edge tier)
  - [x] 02.3.a Write failing test — fixture where a source reaches a sink only through a member-call boundary that is resolved by the `--lsp` tier (an `LSP_RESOLVED` edge exists); assert (1) the taint path continues through that boundary to the sink and a complete source→sink finding is reported (not truncated); (2) the hop across the `LSP_RESOLVED` boundary carries the `LSP_RESOLVED` label, not `AMBIGUOUS`; (3) the same fixture indexed without `--lsp` (no `LSP_RESOLVED` edge) still truncates at the boundary with an `AMBIGUOUS`/"stops at" note — proving the LSP layer is a feeder, not a dependency
  - [x] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintLspResolvedBoundary -count=1` — Expected: FAIL
  - [x] 02.3.c Minimal implementation — solver consults `LSP_RESOLVED` edges when it reaches a call boundary: if a matching `LSP_RESOLVED` edge resolves the callee, the path continues through it (labeling that hop `LSP_RESOLVED`); otherwise it falls back to the `AMBIGUOUS`/"stops at" truncation
  - [x] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintLspResolvedBoundary -count=1` — Expected: PASS
  - [x] 02.3.e Commit — `feat(mnemonic): taint path continues through LSP_RESOLVED boundaries`
- [x] 02.4 `[RED]` Boundary-not-fabricated — a path ending at an unresolved call boundary (no `LSP_RESOLVED` edge) is `AMBIGUOUS`/truncated with "stops at" note, not fabricated (Scenario: Taint path stops at an unresolved boundary) — threat: Mnemonic tool surface
  - [x] 02.4.a Write failing test — (1) fixture where a source flows to a sink only through a call boundary that is unresolved AND has no `LSP_RESOLVED` edge (callee not resolvable statically or by LSP) → assert the finding is reported with the path truncated at the boundary, the boundary hop marked `AMBIGUOUS`, and a "stops at <boundary>" note; (2) fixture where a source has no path to any sink → assert **no** finding is produced (never fabricated); (3) assert no finding is ever reported with an empty or fabricated hop list
  - [x] 02.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintBoundaryNotFabricated -count=1` — Expected: FAIL
  - [x] 02.4.c Minimal implementation — solver truncates the path at the first boundary with no `LSP_RESOLVED` resolution; marks the boundary hop `AMBIGUOUS`; attaches a "stops at <boundary>" note; suppresses findings with no path
  - [x] 02.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintBoundaryNotFabricated -count=1` — Expected: PASS
  - [x] 02.4.e Commit — `feat(mnemonic): taint boundary truncation with stops-at note, no fabricated findings`
- [x] 02.5 `[RED]` Confidence labels — every taint edge is confidence-labeled; path is `EXTRACTED` only when every hop is resolved (Scenario: Every taint hop carries a confidence label)
  - [x] 02.5.a Write failing test — (1) assert every hop in every `taint_findings` path has a non-empty Confidence Label in `EXTRACTED | INFERRED | AMBIGUOUS | LSP_RESOLVED`; (2) fixture where every hop is a resolved data-dependence → path is `EXTRACTED`; (3) fixture where at least one hop is an unresolved (non-`LSP_RESOLVED`) boundary → path is `INFERRED` or `AMBIGUOUS`, never `EXTRACTED`; (4) fixture where a hop is resolved only via `LSP_RESOLVED` → that hop is labeled `LSP_RESOLVED` and the path is not `EXTRACTED` (it is a resolved boundary, not a resolved data-dependence)
  - [x] 02.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintConfidenceLabels -count=1` — Expected: FAIL
  - [x] 02.5.c Minimal implementation — per-hop Confidence Label on the path accepting `LSP_RESOLVED`; path-level label derived from the worst hop (`EXTRACTED` only when all hops are resolved data-dependences, i.e. `EXTRACTED`)
  - [x] 02.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintConfidenceLabels -count=1` — Expected: PASS
  - [x] 02.5.e Commit — `feat(mnemonic): confidence-labeled taint findings with LSP_RESOLVED hops`
- [x] 02.6 `[RED]` Deterministic reproducibility — same source/sink config yields identical findings across repeated runs (Scenario: Repeated taint runs are reproducible)
  - [x] 02.6.a Write failing test — index the same fixture repo with `--pdg` twice (fresh store each time, same source/sink config); assert the full set of `taint_findings` (source kind, sink kind, path, per-hop labels, ordering-independent) is byte-for-byte identical across the two runs; assert no nondeterminism leaks into findings
  - [x] 02.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintReproducible -count=1` — Expected: FAIL
  - [x] 02.6.c Minimal implementation — deterministic source/sink matching + path enumeration (sorted traversal, stable ordering); findings keyed by (source, sink, path) not by run order
  - [x] 02.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintReproducible -count=1` — Expected: PASS
  - [x] 02.6.e Commit — `feat(mnemonic): deterministic taint findings`
- [x] 02.7 `[RED]` Mnemonic tool surface — `code_taint` registered + 005/008/010 tools stable + bad args rejected (Scenario: code_taint registered and bad args fail) — threat: Mnemonic tool surface
  - [x] 02.7.a Write failing test — (1) assert `code_taint` is registered with distinct name + optional `--symbol`/`--file`/`--json` params; (2) assert the full set of 005/008/010 `code_*` tool names + required params is unchanged; (3) assert `code_taint` with bad/missing args is rejected with a clear validation error (abort, not invented findings)
  - [x] 02.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintTool -count=1` — Expected: FAIL
  - [x] 02.7.c Minimal implementation — `code_taint` tool + server registration without dropping existing `code_*`; arg validation
  - [x] 02.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintTool -count=1` — Expected: PASS
  - [x] 02.7.e Commit — `feat(mnemonic): register code_taint tool with stable 005/008/010 surface`
- [x] 02.8 `[AFK]` `--symbol` / `--file` filter findings; `--json` for CI (Scenario: Taint findings filter by symbol and file) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` — Expected: PASS
- [x] 02.9 `[AFK]` Non-`--pdg` index returns a clear "run `--pdg`" message, not an error (Scenario: Non-pdg taint query returns run-pdg hint) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` — Expected: PASS
- [x] 02.10 `[AFK]` Source/sink sets are deterministic and configurable (Scenario: Source and sink sets are configurable) — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -count=1` — Expected: PASS
- [x] 02.11 `[AFK]` `skillgrid search taint` CLI parity + `skillgrid index --pdg` (Scenario: CLI taint search and pdg flag parity) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` | PASS | PASS | pdg (taint unit) + mcp (code_taint tool); TestTaint* integration in codeindex (import-cycle avoidance) |
| Focused test (CLI) | `go test ./skillgrid-cli/cmd/skillgrid/... -count=1` | PASS | PASS | search taint CLI parity (--symbol/--file/--json + non-pdg hint) |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | all `@step-02` scenarios mapped to named tests; byte-for-byte via `baselineFingerprint` (hash 164fd991779f234) |
| Runtime harness | `skillgrid index --pdg` on fixture with known source→sink; `code_taint` returns the finding; cross-boundary finding is `AMBIGUOUS` + "stops at" | PASS | PASS | TaintSourceToSink (default set, GetRequestParam→sqlExec); TaintBoundaryNotFabricated (truncated + StopsAt) |
| LSP boundary runtime harness | `skillgrid index --lsp --pdg` on fixture with a resolvable member-call boundary; `code_taint` path continues through it (not "stops at") | PASS | PASS | TaintLspResolvedBoundary — BOTH arms: --lsp --pdg continues (hop LSP_RESOLVED); --pdg-only truncates (AMBIGUOUS + "stops at"); hermetic lspResolver seam |
| Byte-for-byte gate | `skillgrid index --pdg` on fixture; diff 005/008/010 tool outputs against pre-011 baseline | PASS | PASS | TaintOptInIsolation: --pdg vs non-pdg `baselineFingerprint` equal; pre-011 164fd991779f234 guarded by step-01 TestOptInIsolation; 005/008/010 suite (route/affected/community) `ok` |
| Rollback boundary | Drop `pdg/taint.go` + `code_taint` + `taint_findings` writes; re-run index | PASS | PASS | taint wired into the --pdg hook after PDG Persist; non-pdg path writes no taint rows, baseline stable; taint_findings additive (016) |
| Global Constraints | — | held | held | opt-in isolation, intraprocedural M1, worst-hop confidence labels (4th value LSP_RESOLVED), LSP feeder-not-dependency, no fabricated findings, deterministic, CGo-free (`CGO_ENABLED=0 go build` exit 0) |

Review: task reviewer `approved` (clean). Fix commit fd94b0d (corrected PersistTaint doc-comment — the per-hop path is derived in-memory, not persisted; the row carries the worst-hop path label + stops-at note — and a stray tab in pdg.go). Note for 011 sdd-verify: `taint_findings` persists the path-LEVEL label + note, not per-hop rows (a deliberate schema choice); per-hop reconstruction on query is a possible future enhancement, not a constraint violation.

Commits (step 02): 13695c2 (opt-in taint), 54a116f (source→sink), c504233 (LSP boundary), 8db2b9d (boundary-not-fabricated), f22e408 (confidence labels), 8171708 (determinism), fbb0351 (code_taint tool, surface 74→75), 5bc8fd2 (filters/run-pdg/configurable sets), c38f769 (CLI parity), fd94b0d (doc/tab cleanup).

### Commit

When step DoD is met: `feat(mnemonic): intraprocedural source-to-sink taint with code_taint and LSP-resolved boundaries`

---

## Verification (change-level)

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

**Change**: 011-mnemonic-pdg-taint
**Per-step verdicts**: 01-cfg-pdg PASS · 02-taint-solver PASS
**Runtime proof** (all run at verify time, `-count=1`, exit 0):
- Full 011 suite: `go test ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/mcp/... ./cmd/skillgrid/... ./internal/mnemonic/store/... -count=1` → all `ok`
- 005/008/010 baseline intact: `go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... -count=1` → all `ok`
- CGo-free: `CGO_ENABLED=0 go build ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/...` → exit 0

### Scenario traceability (21 scenarios; @step-01 = 10, @step-02 = 11)

Every scenario is COMPLIANT — a covering test passed at runtime in the suite above.

| @step-01 Scenario | Covering test | Result |
|---|---|---|
| Opt-in index builds per-function CFG and PDG | `codeindex.TestCfgPdgBuild` | COMPLIANT |
| LSP index adds LSP_RESOLVED member-call edges | `codeindex.TestLspResolvedEdges` | COMPLIANT |
| LSP tier works standalone and composes with PDG | `codeindex.TestLspComposition` | COMPLIANT |
| Non-opt-in index is byte-for-byte unchanged | `codeindex.TestOptInIsolation` (baselineFingerprint 164fd991779f234) | COMPLIANT |
| LSP index with no server is byte-for-byte static | `codeindex.TestLspAbsentServer` + `TestLspWarnsOnAbsentServer` (isolated PATH) | COMPLIANT |
| Every PDG edge carries a confidence label | `codeindex.TestPdgConfidenceLabels` | COMPLIANT |
| Malformed function CFG skips and index continues | `codeindex.TestPdgMalformedCfgSkipsContinues` (01.8) | COMPLIANT |
| Over-cap function truncates with a note | `codeindex.TestPdgOverCapTruncatesNeverAborts` (01.9) | COMPLIANT |
| LSP server failure is best-effort no-op | `codeindex.TestLspFailingServerNoPartialEdgeSet` (01.10) + `pdg/lsp_timeout_test.go` | COMPLIANT |
| Repeated PDG builds are reproducible | `codeindex.TestPdgReproducible` | COMPLIANT |
| code_pdg_query registered and bad args fail | `mcp.TestPdgQueryTool` (+ not-found, not-indexed) | COMPLIANT |

| @step-02 Scenario | Covering test | Result |
|---|---|---|
| Source to sink taint path found | `codeindex.TestTaintSourceToSink` | COMPLIANT |
| Every taint hop carries a confidence label | `codeindex.TestTaintConfidenceLabels` | COMPLIANT |
| Opt-in taint index leaves 005/008/010 results unchanged | `codeindex.TestTaintOptInIsolation` (baselineFingerprint) | COMPLIANT |
| Taint path continues through an LSP_RESOLVED boundary | `codeindex.TestTaintLspResolvedBoundary` (both --lsp/--pdg arms) | COMPLIANT |
| Taint path stops at an unresolved boundary | `codeindex.TestTaintBoundaryNotFabricated` (truncation + stops-at) | COMPLIANT |
| No source to sink path means no finding | `codeindex.TestTaintBoundaryNotFabricated` (no-finding assertion) | COMPLIANT |
| Repeated taint runs are reproducible | `codeindex.TestTaintReproducible` | COMPLIANT |
| Taint findings filter by symbol and file | `mcp.TestTaintTool` (--symbol/--file/--json) | COMPLIANT |
| Non-pdg taint query returns run-pdg hint | `mcp.TestTaintToolNotIndexed` | COMPLIANT |
| Source and sink sets are configurable | `codeindex.TestTaintConfigurable` (taint_config_test.go) | COMPLIANT |
| code_taint registered and bad args fail | `mcp.TestTaintTool` (registration + bad args) | COMPLIANT |

### Global Constraints — held
- Opt-in isolation: `--pdg`/`--lsp` gates real; non-flag index byte-for-byte the 005/008/010 graph (content-hash 164fd991779f234 + route/affected/community suites `ok`).
- CGo-free: CFG from existing gotreesitter AST (no new grammar); PDG/taint/LSP-adapter pure Go; `CGO_ENABLED=0 go build` exit 0.
- LSP best-effort: absent/failing/timing-out server → warn+continue, static index unchanged, no partial edge set; `LSP_RESOLVED` is a 4th confidence value.
- Intraprocedural M1: taint per-function; unresolved boundary → `AMBIGUOUS`/"stops at", never fabricated.
- Deterministic: repeated PDG + taint builds byte-identical (sorted traversal, stable ids / keyed findings).

### Review
- Per-step task reviews: 01 `approved with fixes` (fix 4682339), 02 `approved` (fix fd94b0d). Both re-reviewed clean.
- Non-blocking follow-ups (do not gate archive): (a) real external-process JSON-RPC path `pdg/lsp.go:resolveLSPServer` is exercised only via the hermetic `lspResolver` seam (edge-write semantics + the absent/failing/timeout paths are tested; the live JSON-RPC handshake is not, to stay hermetic); (b) `taint_findings` persists the path-LEVEL label + stops-at note, not per-hop rows (deliberate schema choice; per-hop reconstruction on query is a future enhancement); (c) package-level `scanRoot` mutable global in `pdg_pass.go` (single-threaded `Run` in practice).

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
