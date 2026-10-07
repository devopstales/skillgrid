---
id: TASK-057
title: '[FEATURE] Install semgrep + scanner skills + BDD feature (skillgrid-cli)'
status: needs-triage
assignee: []
created_date: '2026-10-07 11:32'
updated_date: '2026-10-07 11:35'
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
