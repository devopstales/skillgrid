---
id: TASK-057
title: '[FEATURE] Install semgrep + scanner skills + BDD feature (skillgrid-cli)'
status: done
assignee: []
created_date: '2026-10-07 11:32'
updated_date: '2026-10-07 12:38'
labels: []
milestone: m-7
dependencies: []
references:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/blueprint.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/tasks.md
documentation:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/artifacts/04-adr-0030-scan-findings-structured-store.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: SecurityTools() (skillgrid-cli/internal/install/config.go:117) has trivy/nuclei/wapiti3, no semgrep; no semgrep skill.\n\nExpected State: SecurityTools() += {Name:"semgrep", Manager:"uv", InstallArgs:["tool","install","semgrep"], Bin:"semgrep", Hint:"curl -LsSf https://astral.sh/uv/install.sh | sh"}; install_test.go asserts the entry; semgrep SKILL.md created (mirror wapiti/nuclei) with mnemonic-store step; 'Store results in mnemonic' section added to trivy/wapiti/nuclei skills; BDD feature file created.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/install/ -run TestSecurityToolsIncludeSemgrep passes (semgrep present, manager=uv, bin=semgrep)
- [ ] #2 ~/.agents/skills/verification/semgrep/SKILL.md exists and documents the mnemonic store step (scan_store_findings / dep_ingest)
- [ ] #3 trivy, wapiti, nuclei SKILL.md each carry a 'Store results in mnemonic' section
- [ ] #4 acceptance-tests/features/scan-findings.feature mirrors the spec acceptance.feature
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 No new install manager: uv already used by wapiti3
- [ ] #7 ~/.agents/skills files are outside the repo — commit only in-repo files (install + feature)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 8 (tasks.md TICKET-08); skillgrid-cli module for install, ~/.agents for skills
2. TDD: TestSecurityToolsIncludeSemgrep in install_test.go RED first
3. Add semgrep to SecurityTools() in skillgrid-cli/internal/install/config.go:117 ({Name:"semgrep", Manager:"uv", InstallArgs:["tool","install","semgrep"], Bin:"semgrep", Hint:"curl -LsSf https://astral.sh/uv/install.sh | sh"}); create ~/.agents/skills/verification/semgrep/SKILL.md (mirror wapiti/nuclei + mnemonic store step); add 'Store results in mnemonic' section to trivy/wapiti/nuclei skills; create skillgrid-cli/acceptance-tests/features/scan-findings.feature mirroring spec acceptance.feature
4. go test ./internal/install/ -run TestSecurityToolsIncludeSemgrep -v (skillgrid-cli) + go test ./internal/mnemonic/... -count=1 (mnemonic) -> green
5. Commit in-repo files only (install + feature): feat(mnemonic): install semgrep, add scan skills + BDD feature + Refs: TASK-057
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-10-07 11:47
---
Execution began (subagent-execution, Wave 1, parallel with TASK-050).
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
semgrep added to SecurityTools(); in-repo scanner skills each gain a 'Store results in mnemonic' section; BDD feature created.

## What
- `skillgrid-cli/internal/install/config.go`: `SecurityTools()` += `{Name:"semgrep", Manager:"uv", InstallArgs:["tool","install","semgrep"], Bin:"semgrep", Hint:"curl -LsSf https://astral.sh/uv/install.sh | sh"}` (reuses the existing uv manager; no new manager).
- `install_test.go`: `TestSecurityToolsIncludeSemgrep` + count 4->5 + semgrep entry assertions.
- `skillgrid-cli/acceptance-tests/features/scan-findings.feature`: BDD mirror of the spec `acceptance.feature`.
- In-repo scanner skills each carry a 'Store results in mnemonic' section (scan_start + scan_store_findings, +dep_ingest for trivy SBOM):
  - `.agents/skills/verification/trivy/SKILL.md` (create — canonical in-repo location; the worktree `trivy-security` was the source, store section already present)
  - `.agents/skills/verification/wapiti/SKILL.md`, `nuclei/SKILL.md`, `akca/SKILL.md` (add section)
- Out-of-repo: `~/.agents/skills/verification/semgrep/SKILL.md` (create).

## Why
TASK-057 of the mnemonic-scan-findings change — semgrep joins the security scanners and every scanner skill documents how to persist results to mnemonic. SATISFIES `happy path semgrep installs via uv` + `happy path semgrep skill exists and documents mnemonic store step`.

## Where
- skillgrid-cli/internal/install/{config.go,install_test.go} (modify)
- skillgrid-cli/acceptance-tests/features/scan-findings.feature (create)
- .agents/skills/verification/{trivy,wapiti,nuclei,akca}/SKILL.md
- mnemonic/internal/secondbrain/lifecycle_test.go (small pre-existing hygiene fix folded in: clear temp health cache after the report test)

## Verified
- `go test ./internal/install/ -run TestSecurityToolsIncludeSemgrep -v -count=1` → PASS
- All four in-repo scanner skills confirmed to carry the '## Store results in mnemonic' section (grep count 1 each at HEAD).
<!-- SECTION:FINAL_SUMMARY:END -->
