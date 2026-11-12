---
id: TASK-020
title: '[FEATURE] NL capture: remember-this intent detection over mem_save'
status: needs-triage
assignee: []
created_date: '2026-09-30 09:45'
labels:
  - mnemonic
  - second-brain
dependencies: []
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Detect user intent to persist a memory and auto-invoke mem_save with an inferred observation type, so capturing a second-brain memory is natural language, not an explicit tool call.

Reference: brobertsaz/claude-os "The Magic" — 'remember this...' → saved with a type tag (Architecture/Decision/Pattern); 'what did we decide?' → search; 'how did we fix that?' → find past solutions. Mnemonic is missing only the intent-detection layer; the store (mem_save) already exists.

Scope:
- A thin capture layer that recognizes recall/persist intents from user (and agent) utterances and routes them to the existing mem_save / mem_search surface with an inferred `type` (decision|architecture|bugfix|pattern|preference|discovery|learning).
- Infer type from phrasing + content heuristics; never block on ambiguity (fall back to default type).
- Reuse existing stripPrivateTags + project scope. No new store.

Acceptance:
- Saying "remember this: <fact>" produces a mem_save with a sensible inferred type, no explicit tool call by the user.
- "what did we decide about <topic>" routes to mem_search and surfaces the stored observation.
- Fallback path saves with a default type when intent is ambiguous (no hard fail).
- No new dependency (locked constraint).

Source: .skillgrid/artifacts/08-second-brain-roadmap.md (change B, P0). Reference project: https://github.com/brobertsaz/claude-os.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
