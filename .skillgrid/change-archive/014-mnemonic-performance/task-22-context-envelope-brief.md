## 22-context-envelope

### Goal

Working set + intent classification + universal context envelope.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-22` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 08, 13, 21

**Files:**
- `internal/mnemonic/memory/envelope.go` — CREATE: context envelope (working set + intent + skills)
- `internal/mnemonic/memory/envelope_test.go` — CREATE: envelope tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `context` subcommand

**Interfaces:**
- Consumes: improve() from step 08; importance scoring from step 13; handoff from step 21
- Produces: `ContextEnvelope` JSON; working set tracking; `mem context` CLI

### Tasks

- [ ] 22.1 `[RED]` Working set tracks files edited, edit counts, net line deltas
  - [ ] 22.1.a Write failing test (`TestWorkingSetTracking`): simulate editing 3 files (2 in project, 1 outside); record the working set; verify it lists the 2 project files with edit counts and net line deltas; verify the file outside the project is excluded; verify hub file status is flagged for files that are hub files
  - [ ] 22.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestWorkingSetTracking'` — Expected: FAIL
  - [ ] 22.1.c Minimal implementation — create `envelope.go` with `WorkingSet` struct: `Files []WorkingFile`, where `WorkingFile` has `Path`, `EditCount`, `NetLines`, `IsHub bool`; track edits during the session; compute net line delta as (lines added - lines removed)
  - [ ] 22.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestWorkingSetTracking'` — Expected: PASS
  - [ ] 22.1.e Commit — `feat(mnemonic): working set tracking with edit counts and line deltas`
- [ ] 22.2 `[RED]` Intent classification returns exploration/debugging/review/refactor
  - [ ] 22.2.a Write failing test (`TestIntentClassificationInEnvelope`): classify "fix the null pointer in auth.go" as `debugging`; classify "what's in the config module" as `exploration`; classify "check the PR for payment.go" as `review`; classify "extract the validation into a separate function" as `refactor`; verify each classification is correct
  - [ ] 22.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestIntentClassificationInEnvelope'` — Expected: FAIL
  - [ ] 22.2.c Minimal implementation — implement `ClassifyIntent(query string) Intent` using keyword patterns: debugging (fix, bug, error, crash, fail), exploration (what, where, list, show), review (check, review, PR, diff), refactor (extract, rename, move, split); return the best-matching intent; default to `exploration`
  - [ ] 22.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestIntentClassificationInEnvelope'` — Expected: PASS
  - [ ] 22.2.e Commit — `feat(mnemonic): intent classification in context envelope`
- [ ] 22.3 `[RED]` ContextEnvelope JSON contains project metadata + working set + matched skills + handoff refs
  - [ ] 22.3.a Write failing test (`TestContextEnvelopeStructure`): generate a context envelope; verify the JSON contains `project` (name, file count, languages), `working_set` (files, edit counts), `intent` (classification), `matched_skills` (list), `handoff_refs` (path to handoff.latest.json); verify the JSON is valid and parseable; verify the envelope size is under a configurable limit
  - [ ] 22.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestContextEnvelopeStructure'` — Expected: FAIL
  - [ ] 22.3.c Minimal implementation — implement `GenerateContextEnvelope(ctx, projectID) (*ContextEnvelope, error)` that assembles all sections; enforce a configurable max envelope size (`mnemonic.envelope.max_size`, default 64KB); truncate or filter fields if the envelope exceeds the limit
  - [ ] 22.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestContextEnvelopeStructure'` — Expected: PASS
  - [ ] 22.3.e Commit — `feat(mnemonic): universal context envelope JSON`
- [ ] 22.4 `[AFK]` mem context CLI with --envelope flag for universal JSON output
  - [ ] 22.4.a Write failing test (`TestMemContextCLI`): invoke `mem context` and verify a human-readable summary is printed; invoke `mem context --envelope` and verify the full JSON envelope is printed to stdout; verify the JSON is valid
  - [ ] 22.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemContextCLI'` — Expected: FAIL
  - [ ] 22.4.c Minimal implementation — add `mem context` subcommand to `mem.go`; print a summary by default (project name, file count, intent, working set size); support `--envelope` flag for full JSON output
  - [ ] 22.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemContextCLI'` — Expected: PASS
  - [ ] 22.4.e Commit — `feat(mnemonic): add mem context CLI with envelope flag`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestWorkingSetTracking\|TestIntentClassificationInEnvelope\|TestContextEnvelopeStructure'` | PASS | | |
| Acceptance `@step-22` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify envelope size limit prevents oversized output | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): working set, intent classification, and universal context envelope`

---

