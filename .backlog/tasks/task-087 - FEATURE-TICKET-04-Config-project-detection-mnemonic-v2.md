---
id: TASK-087
title: '[FEATURE] TICKET-04 Config + project detection (mnemonic-v2)'
status: done
assignee: []
created_date: '2026-11-01 12:44'
updated_date: '2026-11-01 16:25'
labels: []
milestone: m-11
dependencies:
  - TASK-084
  - TASK-086
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
YAML + env configuration with the spec 006-config resolution order (flag > env > file > default) and validation with actionable errors, plus project detection (explicit -> MNEMONIC_PROJECT -> cwd -> git binding -> basename) with --all semantics. CLI and MCP server both resolve their operating project through this package, so its contract gates every command surface.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Precedence matrix: flag beats env beats file beats default for data_dir, sync.dsn, embedder config
- [ ] #2 Invalid YAML -> error naming the file and line
- [ ] #3 Project resolution returns the right project for each selector case; --all lists every enrolled project file
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
author: orchestrator
created: 2026-11-01 15:56
---
Wave 2 (2026-11-01): implementer dispatched.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Config load (flag>env>file>default, per-key zero/empty merge, pointer booleans for false-set) + validation (actionable YAML file+line, postgres-no-DSN, local-no-base_url) + project detection (explicit→MNEMONIC_PROJECT→cwd git toplevel→basename, --all semantics, Canonicalize). yaml.v3 v3.0.1 (ADR-0008). 11 tests. Fix round 1: go mod tidy (yaml direct require), seam field assertion, timeout-borrow comment. Commits ed38985e, 93a0e350. Refs: TASK-087
<!-- SECTION:FINAL_SUMMARY:END -->
