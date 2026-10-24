## Task 02 — Taint Solver — Report

### Status: COMPLETE (all 11 sub-tasks)

Intraprocedural source→sink taint that consumes step 01's PDG + `LSP_RESOLVED`
edges. Opt-in under `--pdg`; a non-`--pdg` index leaves `taint_findings` empty
and the 005/008/010 graph byte-for-byte unchanged. Pure Go (CGo-free).

### What I implemented per sub-task

- **02.1 Opt-in isolation** — `pdg/taint.go` (solver + `PersistTaint`), wired
  into the `--pdg` hook in `codeindex/pdg_pass.go` AFTER the PDG derivation
  (only writes `taint_findings`). Test `TestTaintOptInIsolation`: a `--pdg`
  index of a fixture with a known GetRequestParam→sqlExec flow persists
  `taint_findings` rows; the 005 graph (files/symbols/edges/chunks) is
  byte-for-byte identical between a `--pdg` and a non-`--pdg` run; a
  non-`--pdg` index leaves `taint_findings` empty.
- **02.2 Taint core** — default deterministic source/sink sets (sources:
  request params, env vars, file reads; sinks: SQL exec, shell exec, template
  render, file write), source→sink reachability over the PDG data-dependence
  edges, `code_taint`-shaped findings with source/sink KIND + hop-by-hop path.
  Test `TestTaintSourceToSink`: unit (finding carries source kind
  `request_param`, sink kind `sql_exec`, non-empty path, first hop = source,
  last hop = sink, complete) + persisted in `taint_findings`.
- **02.3 LSP-resolved boundary** — solver consults `LSP_RESOLVED` edges at a
  call boundary: if the callee is LSP-resolved the path CONTINUES (that hop
  labeled `LSP_RESOLVED`); else it truncates (`AMBIGUOUS` + "stops at").
  Added `CallSite.LSPResolved` (from `pdgCallSites` reading the LSP_RESOLVED
  edge confidence). Test `TestTaintLspResolvedBoundary` (+Unit): the SAME
  fixture indexed WITH `--lsp --pdg` (hermetic `lspResolver` seam, no gopls)
  continues through the cross-file `s.Do` boundary to the sink (complete
  finding, no "stops at"); indexed WITHOUT `--lsp` it truncates (`AMBIGUOUS` +
  "stops at" note). The unit test pins the per-hop `LSP_RESOLVED` label and
  that the path is NOT `EXTRACTED`.
- **02.4 Boundary-not-fabricated** — solver truncates at the first unresolved
  boundary (AMBIGUOUS hop + "stops at <boundary>" note) and suppresses
  findings with no path. Test `TestTaintBoundaryNotFabricated`: (1) unresolved
  boundary → truncated + AMBIGUOUS + "stops at f" (unit + integration);
  (2) a source with no path to any sink → NO finding; (3) no finding with an
  empty hop list.
- **02.5 Confidence labels** — per-hop label on every hop; path-level label =
  worst hop (`EXTRACTED` only when every hop is a resolved data-dependence).
  Test `TestTaintConfidenceLabels`: (1) every hop + path label is one of the
  four allowed values, non-empty; (2) all-resolved chain → `EXTRACTED`;
  (3) unresolved boundary → `AMBIGUOUS`/`INFERRED`, never `EXTRACTED`;
  (4) LSP-resolved hop → `LSP_RESOLVED`, path not `EXTRACTED`.
- **02.6 Deterministic reproducibility** — sorted traversal (lowest-line-first
  DFS), findings deduped by content key and sorted by (source, sink, path).
  Test `TestTaintReproducible`: two fresh-store `--pdg` runs (and two
  `--lsp --pdg` runs) produce byte-for-byte identical `taint_findings`.
- **02.7 Tool surface** — `code_taint` MCP tool (optional `--symbol`/`--file`/
  `--json`), registered in `server.go`; 005/008/010 `code_*` names + required
  params unchanged (tool count 74→75). Test `TestTaintTool`: registration +
  distinct name + optional params; 005/008/010 surface intact; a `--pdg` index
  returns the known finding; a `--symbol` filter narrows / an unknown symbol is
  not-found.
- **02.8 Filters** — `--symbol`/`--file` filter findings; `--json` for CI.
  Extended `TestTaintTool` (`--file` filter + unknown file not-found + `--json`
  findings array).
- **02.9 Non-`--pdg` hint** — `code_taint` against a non-`--pdg` index returns
  a clear "run `index --pdg`" message, not an error. Test
  `TestTaintToolNotIndexed` (IsError=false, pdg_enabled=false, count 0).
- **02.10 Configurable sets** — `TaintConfig` with `DefaultTaintConfig()`; a
  caller can override sources/sinks. Test `TestTaintConfigurableSets`: default
  set finds the built-in flow; an overridden set changes what matches; an empty
  set finds nothing; the default config is deterministic.
- **02.11 CLI parity** — `skillgrid search taint` (`--symbol`/`--file`/`--json`)
  wired into `search.go` + `search_taint_cli.go`. Test
  `TestSearchTaintCLIParity` (parity with `index --pdg`, filters, `--json`,
  unknown symbol not-found) + `TestSearchTaintCLINotIndexed` (run-`--pdg` hint).

### How the solver consults LSP_RESOLVED boundaries (with/without `--lsp`)

The PDG data-dependence edges (step 01) connect call blocks in source order;
each edge into a call site is a CALL BOUNDARY. The solver labels the hop into a
call site by the site's resolution status:
- **LSP-resolved** (`LSP_RESOLVED` edge written by `--lsp`) → hop
  `LSP_RESOLVED`, path CONTINUES through the boundary (feeder, not a
  dependency).
- **Statically resolved** (to_id set, EXTRACTED) → hop `EXTRACTED` (a resolved
  data-dependence), path continues.
- **Unresolved** (to_id NULL, no LSP edge) → hop `AMBIGUOUS`, path TRUNCATES
  with a "stops at <boundary>" note.

The `LSPResolved` flag is derived in `pdgCallSites` from the edge's confidence
(`to_id IS NOT NULL AND confidence='LSP_RESOLVED'`). Without `--lsp` there is no
`LSP_RESOLVED` edge, so the same cross-file member call stays unresolved and
truncates — proven by `TestTaintLspResolvedBoundary` indexing the same fixture
both ways.

### How the byte-for-byte 005/008/010 baseline is preserved under `--pdg`

The taint pass is ADDITIVE: it runs only when `--pdg` is set, reads the already-
derived PDG rows + call sites, and writes ONLY to `taint_findings`. It never
touches `files`/`symbols`/`edges`/`chunks`. `TestTaintOptInIsolation` asserts
the 005 fingerprint (files/symbols/edges/chunks) of a `--pdg` run equals that of
a non-`--pdg` run of the same fixture, and `TestOptInIsolation` (step 01) still
asserts the pre-011 baseline hash `164fd991779f234` for the pdg fixture. The
step-01 baseline is unchanged (no 005 extractor edits).

### RED→GREEN evidence (RED sub-tasks)

- **02.1** `TestTaintOptInIsolation` — RED: `--pdg: expected taint_findings rows
  ... got 0` (solver not wired / call sites mis-attributed) → GREEN after
  wiring `pdg.Taint` into the hook + fixing `pdgCallSites` to use the callee
  `to_name` (was using the from-symbol name) and the `dedupeEdgeRows` in-place
  filter bug.
- **02.2** `TestTaintSourceToSink` — RED (build failed: `f.SourceKind
  undefined`) → GREEN after adding `SourceKind`/`SinkKind` to `TaintFinding`.
- **02.3** `TestTaintLspResolvedBoundary` — RED: `--lsp: expected a complete
  source->sink finding ... got 0` + `no --lsp: expected a truncated finding ...
  got 0` → GREEN after the solver continues through LSP-resolved boundaries and
  emits a truncated finding at an unresolved one (fixture: cross-file `s.Do` so
  the static pass leaves it unresolved, the LSP seam resolves it).
- **02.4** `TestTaintBoundaryNotFabricated` — GREEN (truncation + no-fabrication
  already implemented by the 02.3 solver change; test pins the behavior).
- **02.5** `TestTaintConfidenceLabels` — GREEN (per-hop + worst-hop labeling
  already implemented; test pins the four fixtures).
- **02.6** `TestTaintReproducible` — GREEN (sorted traversal + content-key
  dedupe + stable sort already implemented; test pins byte-for-byte stability).
- **02.7** `TestTaintTool` — RED (build: `codeTaintTool`/`handleCodeTaint`
  undefined, `code_taint` not registered, tool count 74→75) → GREEN after the
  tool + registration + count updates.

### Verification

Full relevant suite:
```
go test ./internal/mnemonic/pdg/... ./internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg   1.497s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp  33.346s
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid         13.477s
```
Plus the codeindex taint/integration suite (where the solver tests live):
```
go test ./internal/mnemonic/codeindex/ -run 'TestTaint|TestLspComposition' -count=1
PASS: TestLspComposition, TestTaintBoundaryNotFabricated, TestTaintConfidenceLabels,
      TestTaintConfigurableSets, TestTaintLspResolvedBoundary, TestTaintLspResolvedBoundaryUnit,
      TestTaintOptInIsolation, TestTaintReproducible, TestTaintSourceToSink
```
CGo-free:
```
CGO_ENABLED=0 go build ./internal/mnemonic/pdg/...   → OK
```
Step-01 baseline + 005/008/010 suite still pass:
```
go test ./internal/mnemonic/codeindex/ -run 'TestOptInIsolation|TestTaintOptInIsolation' -count=1
PASS (pre-011 baseline 164fd991779f234 intact)
go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... -count=1
ok  .../route      1.309s
ok  .../affected   8.457s
ok  .../community  2.487s
```

### Commits

| SHA | Subject |
|-----|---------|
| 13695c2 | feat(mnemonic): opt-in taint findings without altering 005/008/010 results |
| 54a116f | feat(mnemonic): intraprocedural source-to-sink taint solver |
| c504233 | feat(mnemonic): taint path continues through LSP_RESOLVED boundaries |
| 8db2b9d | feat(mnemonic): taint boundary truncation with stops-at note, no fabricated findings |
| f22e408 | feat(mnemonic): confidence-labeled taint findings with LSP_RESOLVED hops |
| 8171708 | feat(mnemonic): deterministic taint findings |
| fbb0351 | feat(mnemonic): register code_taint tool with stable 005/008/010 surface |
| 5bc8fd2 | test(mnemonic): taint filters, run-pdg hint, and configurable source/sink sets |
| c38f769 | feat(cli): skillgrid search taint CLI parity |

### Concerns

1. **Persisted path vs stored schema.** The `taint_findings` schema stores
   source/sink/label/note but not the full hop-by-hop path. The persisted `note`
   carries the "stops at" marker; the full path is reconstructed from the PDG on
   query (the solver re-walks the dataDeps). The `code_taint` tool currently
   returns the persisted row (source/sink/label/note) — the hop-by-hop path is
   available from the solver directly. If the acceptance scenarios require the
   full hop list in the tool/CLI output, the path would need to be persisted
   (e.g. a JSON path column) or the tool re-walks the PDG per finding.
2. **Source detection is name-substring.** The default source/sink matching is
   name-substring based (e.g. "param"/"env"/"sql"). It is deterministic and
   configurable, but a call whose name happens to contain a substring (e.g.
   "sqlExec" matches both `sql_exec` and the generic "sql") is classified by the
   FIRST matching class in config order. This is acceptable for the M1 scope but
   worth noting for precision.
3. **Truncated finding's sink_name is the boundary callee.** When a path
   truncates at a non-sink boundary, the finding's `sink_name` is set to the
   boundary callee (a "truncated sink") so the finding is still keyed and
   non-empty. This is a modeling choice (the path "stops at" that boundary); the
   `StopsAt` note + `AMBIGUOUS` label make it unambiguous that it is not a real
   sink.
