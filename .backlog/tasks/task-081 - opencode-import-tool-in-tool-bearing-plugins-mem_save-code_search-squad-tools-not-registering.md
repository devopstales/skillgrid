---
id: TASK-081
title: >-
  opencode: import tool in tool-bearing plugins (mem_save/code_search/squad
  tools not registering)
status: done
assignee: []
created_date: '2026-10-18 09:16'
updated_date: '2026-10-18 10:33'
labels: []
dependencies: []
references:
  - plugins/opencode/mnemonic-memory.ts
  - plugins/opencode/mnemonic-codeindex.ts
  - plugins/opencode/skillgrid-squad.ts
priority: low
type: bug
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 all 5 opencode plugins load without ReferenceError
- [ ] #2 tool-bearing plugins register their tools (mem_save, code_search, squad_spawn_task)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 fixed in source plugins/opencode/ + synced to ~/.config/opencode/plugins/; committed 13699f3b
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add import { tool } from "@opencode-ai/plugin" to the 3 tool-bearing source files.
2. Sync to ~/.config/opencode/plugins/.
3. Verify all 5 modules load + tools register.
4. Remove redundant plugin array from opencode.jsonc.
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-10-18 09:17
---
Fixed 13699f3b: 3 source files import tool; synced to home dir; verified all 5 load + 21 tools register. Removed redundant/incorrect plugin array from opencode.jsonc (local plugins auto-load from plugins/ dir).
---

created: 2026-10-18 10:33
---
RESOLVED via dual v1/v2 export (78c78f5c): root cause was opencode v2.0.28 dropping the v1 plugin API — its loader requires module.default = {id, effect|setup}. Added export default define({id, setup}) (define from @opencode-ai/plugin/v2/promise, a no-op identity) alongside the v1 named export in all 5 plugins. Verified on the live v2.0.28 server: opencode plugin list shows all 5 by v2 id, zero 'failed to load plugin'. v2 setup is a fail-open no-op for now (v1 tool.execute.*/experimental.session.compacting hooks have no 1:1 v2 equivalent); mem_*/code_*/squad_* tools remain available via the mnemonic MCP server. Full v2 hook+tool wiring = follow-up.
---
<!-- COMMENTS:END -->
