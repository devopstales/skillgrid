## 11-portable-export

### Goal

COGX-inspired JSON export of observations + graph + embeddings.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-11` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/memory/export.go` — CREATE: COGX-inspired portable JSON export
- `internal/mnemonic/memory/export_test.go` — CREATE: export tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `export` subcommand

**Interfaces:**
- Consumes: observations table; graph edges from codeindex; symbol_embeddings from step 07
- Produces: `ExportProject(ctx, projectID)` method; `mem export` CLI; COGX-inspired JSON format

### Tasks

- [ ] 11.1 `[RED]` ExportProject returns JSON with observations, graph edges, and embeddings
  - [ ] 11.1.a Write failing test (`TestExportProjectStructure`): create observations, graph edges, and symbol embeddings; call `ExportProject(ctx, projectID)`; verify the returned struct has non-empty `Observations[]`, `GraphEdges[]`, `Embeddings[]`; verify each observation record has `id`, `type`, `content`, `metadata`, `embeddings` (base64), `graph_ref`
  - [ ] 11.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportProjectStructure'` — Expected: FAIL
  - [ ] 11.1.c Minimal implementation — create `export.go` with `ExportProject(ctx context.Context, projectID string) (*ExportBundle, error)`; define `ExportBundle` struct with `Observations []ExportRecord`, `GraphEdges []EdgeRecord`, `Embeddings []EmbeddingRecord`; query all three sources; encode embeddings as base64; COGX-inspired format with `id`, `type`, `content`, `metadata`, `embeddings`, `graph_ref` per record
  - [ ] 11.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportProjectStructure'` — Expected: PASS
  - [ ] 11.1.e Commit — `feat(mnemonic): COGX-inspired portable JSON export`
- [ ] 11.2 `[RED]` Export is portable — output can be imported into another instance
  - [ ] 11.2.a Write failing test (`TestExportImportRoundtrip`): export a project with 5 observations, 3 edges, 2 embeddings; write to a JSON file; create a new empty store; import the JSON; verify all 5 observations, 3 edges, and 2 embeddings are present with matching IDs and content
  - [ ] 11.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportImportRoundtrip'` — Expected: FAIL
  - [ ] 11.2.c Minimal implementation — implement `ImportProject(ctx, projectID, reader io.Reader)` that reads the COGX-inspired JSON and inserts records into the store; ensure embeddings are decoded from base64 and stored in `symbol_embeddings`; verify the roundtrip preserves all data
  - [ ] 11.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportImportRoundtrip'` — Expected: PASS
  - [ ] 11.2.e Commit — `feat(mnemonic): export/import roundtrip for portable memory`
- [ ] 11.3 `[AFK]` mem export CLI with --file and --skip-embeddings flags
  - [ ] 11.3.a Write failing test (`TestMemExportCLI`): invoke `mem export` and verify JSON is written to stdout; invoke `mem export --file out.json` and verify the file is created; invoke `mem export --skip-embeddings` and verify the output JSON has no embedding fields; verify streaming for large stores (no OOM)
  - [ ] 11.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExportCLI'` — Expected: FAIL
  - [ ] 11.3.c Minimal implementation — add `mem export` subcommand to `mem.go`; support `--file` flag for file output (stdout by default); support `--skip-embeddings` flag to omit embedding data for smaller payloads; stream JSON output for large stores to avoid memory pressure
  - [ ] 11.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemExportCLI'` — Expected: PASS
  - [ ] 11.3.e Commit — `feat(mnemonic): add mem export CLI with file and skip-embeddings flags`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExportProjectStructure\|TestExportImportRoundtrip'` | PASS | | |
| Acceptance `@step-11` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify export does not modify the source store | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): COGX-inspired portable JSON export with import roundtrip`

---

