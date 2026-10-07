---
id: TASK-061
title: TICKET-03 Move cmd files + create mnemonic binary
status: ready-for-agent
assignee: []
created_date: '2026-10-07 12:48'
updated_date: '2026-10-07 12:55'
labels: []
milestone: m-8
dependencies:
  - TASK-060
references:
  - .skillgrid/specs/2026-10-06-mnemonic-standalone/briefing.md
modified_files:
  - mnemonic/cmd/mnemonic/**
priority: high
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Move ~45 mnemonic cmd .go files (mcp, mem, search, code_intel, index, init_*, doctor, migrate, policy, session, skill, trail, logs, memory, eval, search_*, handoff_gone, reorder, memhelpers + tests) from skillgrid-cli/cmd/skillgrid/ to mnemonic/cmd/mnemonic/, rewrite imports, and trim the install/sync-repo cases from the moved main.go. This makes mnemonic mcp the thinnest usable standalone binary first.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 cd mnemonic && go build -o /tmp/mnemonic-test ./cmd/mnemonic PASS
- [ ] #2 /tmp/mnemonic-test --help lists all mnemonic subcommands
- [ ] #3 echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | timeout 5 /tmp/mnemonic-test mcp returns JSON with serverInfo
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
created: 2026-10-07 12:55
---
EXECUTOR NOTE: already committed (full subcommand tree in mnemonic/cmd/mnemonic/, binary builds + vet clean). Verify only — re-run acceptance. No new code expected.
---
<!-- COMMENTS:END -->
