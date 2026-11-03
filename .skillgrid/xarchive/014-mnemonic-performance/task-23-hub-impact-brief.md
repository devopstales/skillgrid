## 23-hub-impact

### Goal

Hub file identification + impact analysis + risk scores.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-23` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 10, 13

**Files:**
- `internal/mnemonic/memory/hub.go` — CREATE: hub file identification + impact analysis + risk scores
- `internal/mnemonic/memory/hub_test.go` — CREATE: hub tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `graph --risk` flag

**Interfaces:**
- Consumes: temporal graph from step 10; importance scoring from step 13; codeindex symbols and edges
- Produces: `hub_score` on symbols; `AnalyzeImpact()`; `risk_score` on observations; `mem graph --risk` CLI

### Tasks

- [ ] 23.1 `[RED]` Hub files are identified by 3+ importers
  - [ ] 23.1.a Write failing test (`TestHubFileIdentification`): create a codeindex with 10 files, one of which is imported by 5 other files (hub), and the rest imported by 0-2 files; run hub identification; verify only the file with 3+ importers is flagged as a hub; verify the `hub_score` (import count / total files) is computed correctly
  - [ ] 23.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHubFileIdentification'` — Expected: FAIL
  - [ ] 23.1.c Minimal implementation — create `hub.go` with `IdentifyHubFiles(ctx, projectID) ([]HubFile, error)` that queries the codeindex `edges` table to count importers per file; flag files with 3+ importers as hubs; compute `hub_score = importers / total_files`; store `hub_score` on the symbol
  - [ ] 23.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHubFileIdentification'` — Expected: PASS
  - [ ] 23.1.e Commit — `feat(mnemonic): hub file identification with 3+ importer threshold`
- [ ] 23.2 `[RED]` AnalyzeImpact identifies hub files among changed files
  - [ ] 23.2.a Write failing test (`TestAnalyzeImpactHubFiles`): create a change set with 5 modified files, 2 of which are hub files; call `AnalyzeImpact(changedFiles)`; verify the result flags the 2 hub files as high-impact; verify the non-hub files are marked low-impact; verify the impact analysis includes the number of dependent files for each hub file
  - [ ] 23.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestAnalyzeImpactHubFiles'` — Expected: FAIL
  - [ ] 23.2.c Minimal implementation — implement `AnalyzeImpact(ctx, changedFiles []string) ([]ImpactResult, error)` that cross-references changed files with the hub file list; for each hub file in the change set, count dependent files; return an impact result with `file`, `isHub`, `dependentCount`, `impactLevel` (high/medium/low)
  - [ ] 23.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestAnalyzeImpactHubFiles'` — Expected: PASS
  - [ ] 23.2.e Commit — `feat(mnemonic): impact analysis for hub file changes`
- [ ] 23.3 `[RED]` risk_score on observations based on hub file involvement
  - [ ] 23.3.a Write failing test (`TestRiskScoreOnObservations`): create an observation referencing a hub file; create another observation referencing a non-hub file; verify the hub-file observation has a higher `risk_score`; verify the risk score is proportional to the hub file's `hub_score`; verify the risk score is updated when the hub file's import count changes
  - [ ] 23.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRiskScoreOnObservations'` — Expected: FAIL
  - [ ] 23.3.c Minimal implementation — add `risk_score` (FLOAT) column to observations; compute it as the `hub_score` of the referenced file (0 for non-hub files); update risk scores periodically or on hub file changes; make the threshold configurable
  - [ ] 23.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRiskScoreOnObservations'` — Expected: PASS
  - [ ] 23.3.e Commit — `feat(mnemonic): risk_score on observations based on hub file involvement`
- [ ] 23.4 `[AFK]` mem graph --risk CLI shows high-risk hub files
  - [ ] 23.4.a Write failing test (`TestMemGraphRiskCLI`): create observations with varying risk scores; invoke `mem graph --risk`; verify the output lists high-risk hub files sorted by risk score; verify only files above the risk threshold are shown; verify the threshold is configurable via `--threshold` flag
  - [ ] 23.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemGraphRiskCLI'` — Expected: FAIL
  - [ ] 23.4.c Minimal implementation — add `--risk` flag to the `mem graph` subcommand; query observations ordered by `risk_score` DESC; filter by a configurable threshold (default 0.5); display file path, risk score, and dependent count
  - [ ] 23.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemGraphRiskCLI'` — Expected: PASS
  - [ ] 23.4.e Commit — `feat(mnemonic): add mem graph --risk CLI for high-risk hub files`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHubFileIdentification\|TestAnalyzeImpactHubFiles\|TestRiskScoreOnObservations'` | PASS | | |
| Acceptance `@step-23` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify hub analysis is opt-in with minimum importers threshold | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): hub file identification, impact analysis, and risk scores`

---

