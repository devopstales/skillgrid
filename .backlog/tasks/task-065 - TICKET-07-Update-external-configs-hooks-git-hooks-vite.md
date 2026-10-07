---
id: TASK-065
title: TICKET-07 Update external configs + hooks + git-hooks + vite
status: done
assignee: []
created_date: '2026-10-07 12:50'
updated_date: '2026-10-07 13:50'
labels: []
milestone: m-8
dependencies:
  - TASK-061
references:
  - .skillgrid/specs/2026-10-06-mnemonic-standalone/briefing.md
modified_files:
  - config.d/mcp.yaml
  - plugins/cursor/mcp.json
  - hooks/cursor-session-start.sh
  - hooks/opencode-session-start.sh
  - git-hooks/post-commit
  - git-hooks/post-checkout
  - skillgrid-ui/vite.config.ts
  - scripts/sync-mnemonic-rule.sh
priority: high
type: chore
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Repoint all external config to the mnemonic binary: config.d/mcp.yaml ([mnemonic, mcp]), plugins/cursor/mcp.json, hooks/{cursor,opencode}-session-start.sh (serve/prime), git-hooks/{post-commit,post-checkout} (index/pages), skillgrid-ui/vite.config.ts (outDir: ../mnemonic/internal/http/ui/dist), scripts/sync-mnemonic-rule.sh (fix stale plugins/_shared/ path). mcp.yaml, cursor mcp.json, hooks already verified updated; verify all 7 and patch scripts/sync-mnemonic-rule.sh if the stale path remains.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 config.d/mcp.yaml, plugins/cursor/mcp.json, hooks/cursor-session-start.sh, hooks/opencode-session-start.sh, git-hooks/post-commit, git-hooks/post-checkout, skillgrid-ui/vite.config.ts all reference mnemonic for mnemonic commands
- [ ] #2 no 'skillgrid mcp|serve|index|prime|pages' remains in those files
- [ ] #3 cd skillgrid-ui && npm run build outputs to mnemonic/internal/http/ui/dist and cd mnemonic && go build -tags ui ./cmd/mnemonic PASS
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
EXECUTOR NOTE: verify/finish — scripts/sync-mnemonic-rule.sh + hooks/*.sh + config.d/mcp.yaml + plugins/cursor/mcp.json should all reference the mnemonic module path/binary. Spot-check each for any stale skillgrid-cli/internal/mnemonic reference; patch if found. Mostly already committed.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
T07 verified: scripts/sync-mnemonic-rule.sh reads plugins/opencode/memory-protocol.md; stale plugins/_shared/ comment fixed.
<!-- SECTION:FINAL_SUMMARY:END -->
