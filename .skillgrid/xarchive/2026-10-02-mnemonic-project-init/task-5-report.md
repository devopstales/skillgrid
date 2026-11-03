# Task 5 report — TICKET-05 Onboarding skill calls skillgrid init

## Status

**DONE**

## What was implemented

Onboarding Step 3 gains **9. Project init (deterministic finish)**:

- Run `skillgrid init` from project root (`--force` only if user asked to rebuild preamble).
- Pass user-named extra docs as `--docs <path>`.
- No file-walk / `mem_save` for docs in the skill — CLI owns ingest + code index.
- If CLI missing from PATH: tell user to install and re-run; continue onboarding.

Contract tests in `init_cmd_test.go` read `.agents/skills/lifecycle/onboarding/SKILL.md`.

## SATISFIES

- happy path onboarding skill calls skillgrid init
- skill passes extra docs through
- skill does not contain a second ingest procedure

Acceptance: `TestOnboardingSkillCallsSkillgridInit`, `TestOnboardingSkillHasNoSecondIngest`.

## TDD Evidence

**RED** (tests only, skill unchanged):

```bash
cd skillgrid-cli && go test ./cmd/skillgrid -count=1 -run 'TestOnboardingSkill'
```

```
--- FAIL: TestOnboardingSkillCallsSkillgridInit (0.01s)
    init_cmd_test.go:294: onboarding must name skillgrid init
FAIL
```

**GREEN** (after Step 9 in SKILL.md):

```bash
cd skillgrid-cli && go test ./cmd/skillgrid -count=1 -run 'TestOnboardingSkill|TestInit'
```

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	129.860s
```

## Gates

| Gate | Command | Exit | EXPECT |
|------|---------|------|--------|
| G11 skill names init | `go test ./cmd/skillgrid -count=1 -run TestOnboardingSkillCallsSkillgridInit` | 0 | PASS |
| G12 no second ingest | `go test ./cmd/skillgrid -count=1 -run TestOnboardingSkillHasNoSecondIngest` | 0 | PASS |
| all TestOnboardingSkill\|TestInit | `go test ./cmd/skillgrid -count=1 -run 'TestOnboardingSkill\|TestInit'` | 0 | PASS |

(Run from `skillgrid-cli/`.)

## Files changed

- `.agents/skills/lifecycle/onboarding/SKILL.md` — Step 9 project init handoff
- `skillgrid-cli/cmd/skillgrid/init_cmd_test.go` — two skill-contract tests + `onboardingSkillPath`

## Commit / Checkpoint

`2324c38ebfcb8551f3a8d8dd705924f047f729a2` — `docs(onboarding): finish with skillgrid init`

(On protected `release/2`; rewritten with `git commit-tree` from a clean msg file — no ANSI / no Co-authored-by. `GIT_DIR`/`GIT_WORK_TREE` unset. Single commit: skill + tests; zone guard N/A — no `.skillgrid/specs/`.)

```
[skillgrid-context]
Task: TICKET-05 — onboarding skill calls skillgrid init; no second ingest.
Decisions: Step 9 after Write/Commit; pass --docs through; no file-walk
mem_save for docs; missing CLI → tell user, continue onboarding.
Remaining: none (project-init tickets complete).
Tried: TDD RED (skillgrid init absent) → add Step 9 GREEN; TestOnboardingSkill|TestInit PASS.
```

## Concerns

- None. Existing Step 7 `mem_save` for onboarding config facts is unchanged (not a docs ingest procedure).
