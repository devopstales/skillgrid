---
id: TASK-067
title: TICKET-09 Add CI Go jobs
status: done
assignee: []
created_date: '2026-10-07 12:51'
updated_date: '2026-10-07 13:50'
labels: []
milestone: m-8
dependencies:
  - TASK-061
  - TASK-063
references:
  - .skillgrid/specs/2026-10-06-mnemonic-standalone/briefing.md
modified_files:
  - .github/workflows/hub-sync-check.yml
priority: high
type: chore
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add a go-build-test job to .github/workflows/hub-sync-check.yml running go mod tidy + go vet + go build + go test in both mnemonic/ and skillgrid-cli/ (two working-directory steps). Committed — dual working-directory: mnemonic + skillgrid-cli build/test steps present; verify only.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 .github/workflows/hub-sync-check.yml has a go-build-test job with two steps (one per module)
- [ ] #2 python3 -c "import yaml; yaml.safe_load(open('.github/workflows/hub-sync-check.yml'))" PASS
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-10-07 12:58
---
EXECUTOR NOTE: verify-only — .github/workflows/hub-sync-check.yml (and any other CI workflow) should run go build/vet/test in BOTH the mnemonic and skillgrid-cli module dirs. Check the existing CI job already does dual-module testing; add the missing module leg if it only tested one.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
T09 verified: CI workflow has dual-module go build + go test steps for both mnemonic/ and skillgrid-cli/.
<!-- SECTION:FINAL_SUMMARY:END -->
