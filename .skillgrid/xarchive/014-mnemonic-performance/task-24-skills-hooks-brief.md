## 24-skills-hooks

### Goal

Skills framework + lifecycle hooks for context injection.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-24` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 05, 09

**Files:**
- `internal/mnemonic/memory/skills.go` — CREATE: skills framework + lifecycle hooks
- `internal/mnemonic/memory/skills_test.go` — CREATE: skills tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `skills` and `hook` subcommands

**Interfaces:**
- Consumes: LLM extraction from step 05; session promotion from step 09; intent classification
- Produces: skills as observations with `skill` memory type; lifecycle hooks (session-start, pre-edit, prompt-submit, session-stop); `mem skills` and `mem hook` CLI

### Tasks

- [ ] 24.1 `[RED]` Skills are stored as observations with skill memory type and matched by intent
  - [ ] 24.1.a Write failing test (`TestSkillsMatchingByIntent`): create 3 skills — one for `debugging` intent, one for `review` intent, one for `exploration` intent; classify a query as `debugging`; run skill matching; verify only the debugging skill is returned; verify skills are stored as observations with `memory_type: skill`
  - [ ] 24.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSkillsMatchingByIntent'` — Expected: FAIL
  - [ ] 24.1.c Minimal implementation — create `skills.go` with `MatchSkills(ctx, intent Intent, mentionedFiles []string, projectLangs []string) ([]Skill, error)`; skills are observations with `memory_type: skill` and Markdown content; matching is based on intent, mentioned files, and project languages; return matched skills sorted by relevance
  - [ ] 24.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSkillsMatchingByIntent'` — Expected: PASS
  - [ ] 24.1.e Commit — `feat(mnemonic): skills framework with intent-based matching`
- [ ] 24.2 `[RED]` Lifecycle hooks inject context at the right moments
  - [ ] 24.2.a Write failing test (`TestLifecycleHookSessionStart`): configure a `session-start` hook that injects relevant memories; trigger the hook; verify it returns matched memories and skills; trigger `pre-edit` hook with a file path; verify it returns risk analysis for that file; trigger `session-stop` hook; verify it triggers distillation
  - [ ] 24.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestLifecycleHookSessionStart'` — Expected: FAIL
  - [ ] 24.2.c Minimal implementation — implement `RunHook(ctx, hookType string, payload HookPayload) (HookResult, error)` for 4 hook types: `session-start` (inject relevant memories + skills), `pre-edit` (inject risk analysis for the file), `prompt-submit` (classify intent), `session-stop` (trigger distillation); hooks are opt-in per project
  - [ ] 24.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestLifecycleHookSessionStart'` — Expected: PASS
  - [ ] 24.2.e Commit — `feat(mnemonic): lifecycle hooks for context injection`
- [ ] 24.3 `[AFK]` Hooks are opt-in per project with a timeout
  - [ ] 24.3.a Write failing test (`TestHooksOptInWithTimeout`): disable hooks for a project; trigger a hook; verify no hook runs; enable hooks; trigger a hook that sleeps 10 seconds; verify it times out after the configured timeout (default 30s) and returns a timeout error; verify the timeout is configurable
  - [ ] 24.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHooksOptInWithTimeout'` — Expected: FAIL
  - [ ] 24.3.c Minimal implementation — add `mnemonic.hooks.enabled` config per project (default `false`); wrap hook execution in a `context.WithTimeout` (default 30s); on timeout, return a descriptive error and log a warning; make the timeout configurable via `mnemonic.hooks.timeout`
  - [ ] 24.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHooksOptInWithTimeout'` — Expected: PASS
  - [ ] 24.3.e Commit — `feat(mnemonic): make hooks opt-in with configurable timeout`
- [ ] 24.4 `[AFK]` mem skills and mem hook CLI subcommands
  - [ ] 24.4.a Write failing test (`TestMemSkillsAndHookCLI`): invoke `mem skills list` and verify all skills are listed; invoke `mem skills add "name" "content"` and verify a skill is created; invoke `mem hook list` and verify configured hooks are shown; invoke `mem hook run session-start` and verify the hook executes
  - [ ] 24.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSkillsAndHookCLI'` — Expected: FAIL
  - [ ] 24.4.c Minimal implementation — add `mem skills list`, `mem skills add <name> <content>`, `mem hook list`, and `mem hook run <type>` subcommands to `mem.go`; wire them to the skills and hooks modules
  - [ ] 24.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemSkillsAndHookCLI'` — Expected: PASS
  - [ ] 24.4.e Commit — `feat(mnemonic): add mem skills and mem hook CLI subcommands`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestSkillsMatchingByIntent\|TestLifecycleHookSessionStart\|TestHooksOptInWithTimeout'` | PASS | | |
| Acceptance `@step-24` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify hooks are opt-in per project with timeout | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): skills framework and lifecycle hooks for context injection`

---

