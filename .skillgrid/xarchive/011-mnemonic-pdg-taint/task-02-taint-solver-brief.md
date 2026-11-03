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

- [ ] 02.1 `[RED]` Opt-in isolation — `--pdg` index adds taint findings without altering 005/008/010 results (Scenario: Opt-in taint index leaves 005/008/010 results unchanged) — threat: Opt-in isolation
  - [ ] 02.1.a Write failing test — index a fixture repo with `--pdg` that has at least one known source→sink path; assert (1) `taint_findings` rows exist for the known flow; (2) every 005/008/010 `code_*` tool output is byte-for-byte identical to the pre-011 baseline for the same repo; (3) a non-`--pdg` index of the same repo leaves `taint_findings` empty
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintOptInIsolation -count=1` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — taint solver wired into the `--pdg` hook after PDG derivation; findings written to `taint_findings` only when `--pdg` is set
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintOptInIsolation -count=1` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): opt-in taint findings without altering 005/008/010 results`
- [ ] 02.2 `[RED]` Taint core — source→sink path found and persisted (Scenario: Source to sink taint path found)
  - [ ] 02.2.a Write failing test — fixture with a known source (e.g. request param) that flows to a known sink (e.g. SQL exec) through a resolvable data-dependence chain; assert `code_taint` returns a finding with the source kind, sink kind, and the hop-by-hop path; assert the finding is persisted in `taint_findings`
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintSourceToSink -count=1` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation — `pdg/taint.go` default deterministic source/sink sets (sources: request params, env vars, file reads; sinks: SQL exec, shell exec, template render, file write); source→sink reachability solver over the PDG data-dependence edges; `code_taint` tool
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintSourceToSink -count=1` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): intraprocedural source-to-sink taint solver`
- [ ] 02.3 `[RED]` LSP-resolved boundary — a taint path continues through an `LSP_RESOLVED` call boundary instead of "stops at" (Scenario: Taint path continues through an LSP_RESOLVED boundary) — threat: Mnemonic tool surface (LSP edge tier)
  - [ ] 02.3.a Write failing test — fixture where a source reaches a sink only through a member-call boundary that is resolved by the `--lsp` tier (an `LSP_RESOLVED` edge exists); assert (1) the taint path continues through that boundary to the sink and a complete source→sink finding is reported (not truncated); (2) the hop across the `LSP_RESOLVED` boundary carries the `LSP_RESOLVED` label, not `AMBIGUOUS`; (3) the same fixture indexed without `--lsp` (no `LSP_RESOLVED` edge) still truncates at the boundary with an `AMBIGUOUS`/"stops at" note — proving the LSP layer is a feeder, not a dependency
  - [ ] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintLspResolvedBoundary -count=1` — Expected: FAIL
  - [ ] 02.3.c Minimal implementation — solver consults `LSP_RESOLVED` edges when it reaches a call boundary: if a matching `LSP_RESOLVED` edge resolves the callee, the path continues through it (labeling that hop `LSP_RESOLVED`); otherwise it falls back to the `AMBIGUOUS`/"stops at" truncation
  - [ ] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintLspResolvedBoundary -count=1` — Expected: PASS
  - [ ] 02.3.e Commit — `feat(mnemonic): taint path continues through LSP_RESOLVED boundaries`
- [ ] 02.4 `[RED]` Boundary-not-fabricated — a path ending at an unresolved call boundary (no `LSP_RESOLVED` edge) is `AMBIGUOUS`/truncated with "stops at" note, not fabricated (Scenario: Taint path stops at an unresolved boundary) — threat: Mnemonic tool surface
  - [ ] 02.4.a Write failing test — (1) fixture where a source flows to a sink only through a call boundary that is unresolved AND has no `LSP_RESOLVED` edge (callee not resolvable statically or by LSP) → assert the finding is reported with the path truncated at the boundary, the boundary hop marked `AMBIGUOUS`, and a "stops at <boundary>" note; (2) fixture where a source has no path to any sink → assert **no** finding is produced (never fabricated); (3) assert no finding is ever reported with an empty or fabricated hop list
  - [ ] 02.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintBoundaryNotFabricated -count=1` — Expected: FAIL
  - [ ] 02.4.c Minimal implementation — solver truncates the path at the first boundary with no `LSP_RESOLVED` resolution; marks the boundary hop `AMBIGUOUS`; attaches a "stops at <boundary>" note; suppresses findings with no path
  - [ ] 02.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintBoundaryNotFabricated -count=1` — Expected: PASS
  - [ ] 02.4.e Commit — `feat(mnemonic): taint boundary truncation with stops-at note, no fabricated findings`
- [ ] 02.5 `[RED]` Confidence labels — every taint edge is confidence-labeled; path is `EXTRACTED` only when every hop is resolved (Scenario: Every taint hop carries a confidence label)
  - [ ] 02.5.a Write failing test — (1) assert every hop in every `taint_findings` path has a non-empty Confidence Label in `EXTRACTED | INFERRED | AMBIGUOUS | LSP_RESOLVED`; (2) fixture where every hop is a resolved data-dependence → path is `EXTRACTED`; (3) fixture where at least one hop is an unresolved (non-`LSP_RESOLVED`) boundary → path is `INFERRED` or `AMBIGUOUS`, never `EXTRACTED`; (4) fixture where a hop is resolved only via `LSP_RESOLVED` → that hop is labeled `LSP_RESOLVED` and the path is not `EXTRACTED` (it is a resolved boundary, not a resolved data-dependence)
  - [ ] 02.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintConfidenceLabels -count=1` — Expected: FAIL
  - [ ] 02.5.c Minimal implementation — per-hop Confidence Label on the path accepting `LSP_RESOLVED`; path-level label derived from the worst hop (`EXTRACTED` only when all hops are resolved data-dependences, i.e. `EXTRACTED`)
  - [ ] 02.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintConfidenceLabels -count=1` — Expected: PASS
  - [ ] 02.5.e Commit — `feat(mnemonic): confidence-labeled taint findings with LSP_RESOLVED hops`
- [ ] 02.6 `[RED]` Deterministic reproducibility — same source/sink config yields identical findings across repeated runs (Scenario: Repeated taint runs are reproducible)
  - [ ] 02.6.a Write failing test — index the same fixture repo with `--pdg` twice (fresh store each time, same source/sink config); assert the full set of `taint_findings` (source kind, sink kind, path, per-hop labels, ordering-independent) is byte-for-byte identical across the two runs; assert no nondeterminism leaks into findings
  - [ ] 02.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintReproducible -count=1` — Expected: FAIL
  - [ ] 02.6.c Minimal implementation — deterministic source/sink matching + path enumeration (sorted traversal, stable ordering); findings keyed by (source, sink, path) not by run order
  - [ ] 02.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -run TaintReproducible -count=1` — Expected: PASS
  - [ ] 02.6.e Commit — `feat(mnemonic): deterministic taint findings`
- [ ] 02.7 `[RED]` Mnemonic tool surface — `code_taint` registered + 005/008/010 tools stable + bad args rejected (Scenario: code_taint registered and bad args fail) — threat: Mnemonic tool surface
  - [ ] 02.7.a Write failing test — (1) assert `code_taint` is registered with distinct name + optional `--symbol`/`--file`/`--json` params; (2) assert the full set of 005/008/010 `code_*` tool names + required params is unchanged; (3) assert `code_taint` with bad/missing args is rejected with a clear validation error (abort, not invented findings)
  - [ ] 02.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintTool -count=1` — Expected: FAIL
  - [ ] 02.7.c Minimal implementation — `code_taint` tool + server registration without dropping existing `code_*`; arg validation
  - [ ] 02.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run TaintTool -count=1` — Expected: PASS
  - [ ] 02.7.e Commit — `feat(mnemonic): register code_taint tool with stable 005/008/010 surface`
- [ ] 02.8 `[AFK]` `--symbol` / `--file` filter findings; `--json` for CI (Scenario: Taint findings filter by symbol and file) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` — Expected: PASS
- [ ] 02.9 `[AFK]` Non-`--pdg` index returns a clear "run `--pdg`" message, not an error (Scenario: Non-pdg taint query returns run-pdg hint) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` — Expected: PASS
- [ ] 02.10 `[AFK]` Source/sink sets are deterministic and configurable (Scenario: Source and sink sets are configurable) — `Run: go test ./skillgrid-cli/internal/mnemonic/pdg/... -count=1` — Expected: PASS
- [ ] 02.11 `[AFK]` `skillgrid search taint` CLI parity + `skillgrid index --pdg` (Scenario: CLI taint search and pdg flag parity) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` | PASS | | |
| Focused test (CLI) | `go test ./skillgrid-cli/cmd/skillgrid/... -count=1` | PASS | | |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `skillgrid index --pdg` on fixture with known source→sink; `code_taint` returns the finding; cross-boundary finding is `AMBIGUOUS` + "stops at" | PASS | | |
| LSP boundary runtime harness | `skillgrid index --lsp --pdg` on fixture with a resolvable member-call boundary; `code_taint` path continues through it (not "stops at") | PASS | | |
| Byte-for-byte gate | `skillgrid index --pdg` on fixture; diff 005/008/010 tool outputs against pre-011 baseline | PASS | | |
| Rollback boundary | Drop `pdg/taint.go` + `code_taint` + `taint_findings` writes; re-run index | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): intraprocedural source-to-sink taint with code_taint and LSP-resolved boundaries`

---

