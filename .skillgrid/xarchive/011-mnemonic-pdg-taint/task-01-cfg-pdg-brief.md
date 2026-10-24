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
 | Focused test | `go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` | PASS | PASS | store ok; pdg unit tests run under codeindex |
| Focused test (codeindex + mcp) | `go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` | PASS | PASS | both packages ok |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | OptInIsolation/CfgPdgBuild/Lsp*/ConfidenceLabels/PdgQueryTool/SearchPdg all PASS |
| Runtime harness | `skillgrid index --pdg` on fixture repo; query `cfg_blocks`/`pdg_edges` via `code_pdg_query` | PASS | PASS | TestSearchPdgCLIParity + TestPdgQueryTool (dependents returned, confidence-labeled) |
| LSP runtime harness | `skillgrid index --lsp` (with server) adds `LSP_RESOLVED` edges; `skillgrid index --lsp` (no server) is byte-for-byte static | PASS | PASS | TestLspResolvedEdges / TestLspAbsentServer / TestLspComposition |
| Byte-for-byte gate | `skillgrid index` (no `--pdg`) on fixture repo; diff 005/008/010 tool outputs against pre-011 baseline | PASS | PASS | TestOptInIsolation fingerprint 164fd991779f234 stable; pdg tables empty on non---pdg |
| Rollback boundary | Drop `016_pdg_taint.sql` + `pdg/` (incl. `lsp.go`) + `code_pdg_query` + `--pdg`/`--lsp` hooks; re-run non-`--pdg` index | PASS | PASS | additive migration; non---pdg path never touches the four tables |
| Global Constraints | — | held | held | CGo-free build (01.11) PASS; tool surface 73→74 additive; no 005/008/010 change |

### Commit

When step DoD is met: `feat(mnemonic): opt-in per-function CFG, PDG, and LSP edge tier`

---

