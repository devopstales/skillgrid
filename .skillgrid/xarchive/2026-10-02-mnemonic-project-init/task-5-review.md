# Task 5 review — TICKET-05 Onboarding skill calls skillgrid init

**Base:** `8763f9b6` · **Head:** `2324c38e`  
**Diff:** `task-5.diff` · **Brief:** `task-5-brief.md` · **Report:** `task-5-report.md`  
**Overall:** APPROVED

### Spec Compliance

- ✅ Spec compliant
- Parent verified `TestOnboardingSkill|TestInit` PASS — accepted (not re-run here).

Checked against brief / TICKET-05 / blueprint Task 5:

| Requirement | Evidence |
|---|---|
| Onboarding skill names `skillgrid init` | SKILL.md Step **9. Project init** after Commit / before Step 4 Verify |
| Optional `--force` when user asked to rebuild preamble | Step 9 bullet matches brief verbatim |
| Extra docs via `--docs <path>` | Step 9 bullet matches brief verbatim |
| No second ingest procedure | Explicit “Do not walk files or call `mem_save` for docs”; forbidden phrase absent |
| Missing CLI → tell user, continue onboarding | Final Step 9 bullet |
| `TestOnboardingSkillCallsSkillgridInit` | Diff matches brief Step 1 |
| `TestOnboardingSkillHasNoSecondIngest` | Diff matches brief Step 1 (sentinel `walk docs/ and mem_save each file`) |
| Files: `SKILL.md`, `init_cmd_test.go` | Diff stat: 2 files, +36 |

### Strengths

- Exact brief Step 9 text; placement is correct (end of Write/Commit, before Verify).
- Contract tests are thin and stable; `onboardingSkillPath` resolves repo-root skill from `skillgrid-cli/cmd/skillgrid`.
- Report correctly distinguishes Step 7 config `mem_save` from docs ingest (not a second ingest procedure).
- Commit subject matches brief: `docs(onboarding): finish with skillgrid init`.

### Issues

#### Critical (Must Fix)

_(none)_

#### Important (Should Fix)

_(none)_

#### Minor (Nice to Have)

1. **Merge Mode** — re-run path still does not mention `skillgrid init`. Out of TICKET-05 scope; optional later if brownfield re-onboarding should also hand off to the CLI.
2. **Deletion-test sentinel** — only blocks the exact phrase `walk docs/ and mem_save each file`. Adequacy matches the brief’s prescribed test; a broader “no docs walk + mem_save” check would be stronger but is not required.

### Assessment

**Task quality:** Approved

**Reasoning:** Skill contract hands off to `skillgrid init` with optional `--force` / `--docs`, forbids a second docs ingest, and both acceptance tests match the brief. TDD RED→GREEN evidence in the report is credible; parent gate PASS. No blockers.
