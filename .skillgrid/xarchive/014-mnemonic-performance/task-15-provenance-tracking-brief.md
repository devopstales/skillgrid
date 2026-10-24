## 15-provenance-tracking

### Goal

Provenance chain metadata on observations.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-15` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 05

**Files:**
- `internal/mnemonic/memory/provenance.go` — CREATE: provenance chain tracking
- `internal/mnemonic/memory/provenance_test.go` — CREATE: provenance tests
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: provenance column
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `provenance` subcommand

**Interfaces:**
- Consumes: LLM extraction from step 05; session context; curation pipeline
- Produces: `provenance` JSON column on observations; `mem provenance` CLI

### Tasks

- [ ] 15.1 `[RED]` Provenance JSON chain stores session_id → curate_command → source_files → LLM_reasoning
  - [ ] 15.1.a Write failing test (`TestProvenanceChainStorage`): save an observation with a full provenance chain (session_id, curate_command, source_files, llm_reasoning); retrieve the observation; verify the `provenance` JSON column contains all four fields with correct values; verify the JSON is valid and parseable
  - [ ] 15.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceChainStorage'` — Expected: FAIL
  - [ ] 15.1.c Minimal implementation — add `provenance` (TEXT, JSON) column to observations via migration; define `Provenance` struct with `SessionID`, `CurateCommand`, `SourceFiles []string`, `LLMReasoning string` fields; serialize to JSON on save; parse on read
  - [ ] 15.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceChainStorage'` — Expected: PASS
  - [ ] 15.1.e Commit — `feat(mnemonic): add provenance JSON chain to observations`
- [ ] 15.2 `[RED]` Provenance is immutable once set during curation
  - [ ] 15.2.a Write failing test (`TestProvenanceImmutability`): save an observation with provenance; attempt to update the observation without changing provenance; verify the provenance is unchanged; attempt to update with a different provenance; verify the update is rejected (or the provenance is preserved); verify the original provenance is intact
  - [ ] 15.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceImmutability'` — Expected: FAIL
  - [ ] 15.2.c Minimal implementation — in the update path, check if `provenance` is already set (non-NULL); if so, preserve the existing value regardless of the update input; log a warning if a different provenance is attempted; only the initial save can set provenance
  - [ ] 15.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceImmutability'` — Expected: PASS
  - [ ] 15.2.e Commit — `feat(mnemonic): make provenance immutable after initial set`
- [ ] 15.3 `[AFK]` mem provenance CLI returns the full provenance chain
  - [ ] 15.3.a Write failing test (`TestMemProvenanceCLI`): save an observation with a full provenance chain; invoke `mem provenance <observation_id>`; verify the output shows all four chain elements (session_id, curate_command, source_files, llm_reasoning); verify the output is formatted as a readable chain (not raw JSON); verify missing provenance returns a clear message
  - [ ] 15.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemProvenanceCLI'` — Expected: FAIL
  - [ ] 15.3.c Minimal implementation — add `mem provenance <observation_id>` subcommand to `mem.go`; query the observation and parse its `provenance` JSON; format the output as a readable chain; include provenance summary in `mem list` output
  - [ ] 15.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemProvenanceCLI'` — Expected: PASS
  - [ ] 15.3.e Commit — `feat(mnemonic): add mem provenance CLI subcommand`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestProvenanceChainStorage\|TestProvenanceImmutability'` | PASS | | |
| Acceptance `@step-15` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify observations without provenance still work | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): provenance chain metadata with immutability`

---

