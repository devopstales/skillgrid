## 21-handoff

### Goal

Prefix + delta handoff artifacts for agent-to-agent context passing.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-21` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 09, 12

**Files:**
- `internal/mnemonic/memory/handoff.go` — CREATE: prefix + delta handoff artifacts
- `internal/mnemonic/memory/handoff_test.go` — CREATE: handoff tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `handoff` subcommand

**Interfaces:**
- Consumes: session promotion from step 09; DreamExecutor from step 12; project metadata
- Produces: `handoff.latest.json` with stable `prefix` + dynamic `delta`; `mem handoff` CLI

### Tasks

- [x] 21.1 `[RED]` Handoff artifact contains stable prefix with hub summaries and repo file-count
  - [x] 21.1.a Write failing test (`TestHandoffPrefixStable`): generate a handoff artifact; verify the `prefix` section contains hub file summaries and repo file count; modify a non-hub file; regenerate the handoff; verify the `prefix` section is unchanged (stable); modify a hub file; regenerate; verify the `prefix` reflects the hub file change
  - [x] 21.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffPrefixStable'` — Expected: FAIL
  - [x] 21.1.c Minimal implementation — create `handoff.go` with `GenerateHandoff(ctx, projectID) (*Handoff, error)`; the `prefix` section includes hub file summaries (from codeindex) and repo file count; the prefix is computed from stable project metadata and only changes when hub files change
  - [x] 21.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffPrefixStable'` — Expected: PASS
  - [x] 21.1.e Commit — `feat(mnemonic): handoff prefix with stable hub summaries`
- [x] 21.2 `[RED]` Handoff delta contains changed file stubs, risk files, recent events
  - [x] 21.2.a Write failing test (`TestHandoffDeltaDynamic`): generate a handoff; modify 3 files (1 hub, 2 regular); regenerate the handoff; verify the `delta` section lists all 3 changed files with stubs; verify the hub file is flagged as a risk file; verify recent session events are included; verify the delta reflects only changes since the last handoff
  - [x] 21.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffDeltaDynamic'` — Expected: FAIL
  - [x] 21.2.c Minimal implementation — the `delta` section includes: changed file stubs (file path, first 50 lines), risk files (hub files among changes), recent events (last 10 session events), and working set summary; compute the delta at handoff time (not cached) to ensure freshness
  - [x] 21.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffDeltaDynamic'` — Expected: PASS
  - [x] 21.2.e Commit — `feat(mnemonic): handoff delta with changed files and risk analysis`
- [x] 21.3 `[AFK]` Handoff saved to .skillgrid/handoff.latest.json
  - [x] 21.3.a Write failing test (`TestHandoffSavedToFile`): generate a handoff; verify a file `.skillgrid/handoff.latest.json` is created; verify the JSON is valid and contains both `prefix` and `delta` sections; verify a second generation overwrites the file (not appends)
  - [x] 21.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffSavedToFile'` — Expected: FAIL
  - [x] 21.3.c Minimal implementation — in `GenerateHandoff`, serialize the handoff to JSON and write to `.skillgrid/handoff.latest.json` (create the directory if needed); overwrite on each generation; include a `generated_at` timestamp
  - [x] 21.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffSavedToFile'` — Expected: PASS
  - [x] 21.3.e Commit — `feat(mnemonic): save handoff to .skillgrid/handoff.latest.json`
- [x] 21.4 `[AFK]` mem handoff CLI generates handoff artifact
  - [x] 21.4.a Write failing test (`TestMemHandoffCLI`): invoke `mem handoff`; verify it generates the handoff and prints a summary to stdout; verify the file is written; invoke `mem handoff --json` and verify the full JSON is printed to stdout
  - [x] 21.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemHandoffCLI'` — Expected: FAIL
  - [x] 21.4.c Minimal implementation — add `mem handoff` subcommand to `mem.go`; call `GenerateHandoff`; print a human-readable summary by default; support `--json` flag for full JSON output
  - [x] 21.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemHandoffCLI'` — Expected: PASS
  - [x] 21.4.e Commit — `feat(mnemonic): add mem handoff CLI subcommand`

### Verification

Verdict: `PASS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHandoffPrefixStable\|TestHandoffDeltaDynamic\|TestHandoffSavedToFile'` | PASS | PASS | all 3 green |
| Acceptance `@step-21` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | prefix/delta/file/cli scenarios covered |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | PASS | full package green (36s) |
| Rollback boundary | verify handoff is opt-in and computed at generation time | PASS | PASS | GenerateHandoff on demand; cursor in handoff_meta |
| Global Constraints | — | held | held | additive only; no tool contract change; no CGO |
| CLI | `go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemHandoffCLI'` | PASS | PASS | summary + --json green |

> Note: `TestReindexStructuralIsEmbedderFree` (service package) is a
> pre-existing failure that reproduces on clean HEAD (unrelated to step 21).

### Commit

When step DoD is met: `feat(mnemonic): prefix + delta handoff artifacts for agent-to-agent context`

---

