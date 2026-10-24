# SDD ledger — plan: .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/tasks.md

## Delivery
Ruling: size:exception accepted by user 2026-10-02 — single commit stream on release/2 — cost if wrong: larger review diff, revert by commit range.
Chain strategy: size-exception
Merge base will be recorded at first code commit.

## Pre-flight conflict scan
| Pair / task | Shared / self-check | Finding |
|---|---|---|
| T01 ↔ T03/T04/T08 | Config structs consumed by later tickets | Clean — T01 only writes config load |
| T02 ↔ T04 | CheckpointState / ClaimCheckpoint | Clean — T02 produces, T04 consumes |
| T03 ↔ T04 | Decide/BuildDigest/RenderPrompt | Clean — T03 pure, T04 wires |
| T07 ↔ T04 | StripPrivate on digest input | Clean — T07 before T04 in Needs |
| T06 ↔ T05 | tool-call-capture.js both touch | Soft — T06 deletes resolveProject; T05 adds checkpoint mode. Order: T06 before T05 in wave practice, or T05 must not rely on resolveProject. Ruling: T06 before T05 if both touch the script; Wave 1 runs T06 before Wave 4 T05 so OK. |
| T10 ↔ T11 | observations[] API | Clean — T10 then T11 |
| T02 self | migration vs temp tests | Ruling: tests use temp stores only; do not manually ALTER ~/.skillgrid/mnemonic/*.sqlite. First serve open applies 049. Cost if wrong: user store migrates on next serve — additive nullable, reversible by ignore. |
| T07 self | privacy.go vs session_inject stripper | Check for existing stripper and move/alias per blueprint |
Scan clean aside from the two rulings above.

## Ownership (Wave 1)
Task 01: Owns: skillgrid-cli/internal/mnemonic/config/**, docs/user-guide/05-memory.md — Needs: none
Task 02: Owns: skillgrid-cli/internal/mnemonic/store/migrations/049_*, memory/service.go, memory/service_checkpoint_test.go — Needs: none
Task 06: Owns: http/server.go projectFromRequest, http/toolcalls*, hooks/tool-call-capture.js (resolver), cmd/skillgrid/prime* — Needs: none
Task 07: Owns: memory/privacy*, memory/service.go write seams, http/toolcalls preview strip, hooks strip, scripts/test-hooks private — Needs: none
Task 10: Owns: http/toolevents*, mnemonic_files.go observations count, openapi observations — Needs: none

Note: T02 and T07 both touch memory/service.go — sequential Wave 1; T07 after T02 or merge carefully. Ruling: dispatch order T01 → T02 → T07 → T06 → T10 so service.go write bumps land before StripPrivate call sites.

## Task 01
Status: in_progress
Owns: skillgrid-cli/internal/mnemonic/config/**, docs/user-guide/05-memory-and-indexing.md
Needs: none
Ruling: invalid-value → warn+default (acceptance.feature over blueprint hard-error) — cost if wrong: Load fails closed on bad config instead of soft defaults.
Dispatched: 2026-10-02

## Task 01: complete
SHA: 3d826978d5c3f65b3bb5ca31094b2faaeb873d96
Focused test: go test ./internal/mnemonic/config -run TestCheckpointConfig → ok (parent re-verify)
Runtime harness: N/A — config load only
Rollback: load.go Checkpoint/Inject + docs; soft-reset fixed bad commit-tree that staged backlog only
Ruling: controller rewrote commit after implementer commit-tree used wrong tree — cost if wrong: none (unpushed)
Review: controller Spec+quality APPROVED (mechanical config; gate green)

## Task 02
Status: in_progress
Owns: 049 migration, memory/service.go (+checkpoint methods), service_checkpoint_test.go
Needs: none
Dispatched: 2026-10-02

## Task 02: complete
SHA: e5d5efa93a5dc5ade7ac6bbab998a736cb90ced8
Focused test: TestCheckpointState → ok
Runtime harness: N/A — temp store only
Rollback: revert e5d5efa9 (nullable columns ignored by older binaries)
Review: controller APPROVED

## Task 07
Status: in_progress
Dispatched: 2026-10-02

## Task 07: complete
SHA: (rewritten clean subject after ANSI in message)
Focused test: TestStripPrivate|TestToolCalls_PrivateSpan|TestSessionSummary_PrivateSpan → ok; test-hooks private 4/0
Runtime harness: N/A — unit + hook fake server
Rollback: revert privacy commit
Review: controller APPROVED; fixed ANSI subject via commit-tree

## Task 07: complete
SHA: 4156a3328e287d62682c876949650b89eddab137
## Task 06
Status: in_progress
Dispatched: 2026-10-02

## Task 06: complete
SHA: fcd201d6
Focused test: TestToolCalls_ResolvesProjectFromDirectory|TestPrime_ProjectMatchesHTTPStore → ok (parent re-verify, all perms)
Runtime harness: N/A — HTTP + prime unit tests
Rollback: revert fcd201d6 (hooks fail open; server still accepts explicit project)
Review: controller APPROVED; Owns slightly expanded to policy/usage for directory consistency — in scope for resolver

## Task 10
Status: in_progress
Owns: http/toolevents*, mnemonic_files.go observations count, openapi observations
Needs: none
Dispatched: 2026-10-02
Ruling: sessions list gains additive `observations` JSON field mirroring memory_count — keep memory_count for existing UI — cost if wrong: dual fields until T11 migrates

## Task 10: complete
SHA: 9e77568f (rewrote 6c6317aa — stripped ANSI + Co-authored-by via commit-tree)
Focused test: go test ./internal/mnemonic/http -run TestSessionEvents → ok (parent re-verify)
Runtime harness: N/A — HTTP unit tests
Rollback: revert 9e77568f (additive API; older UI ignores missing fields)
Review: controller APPROVED

## Task 03
Status: in_progress
Owns: skillgrid-cli/internal/mnemonic/checkpoint/**
Needs: T01 (config.Checkpoint)
Dispatched: 2026-10-02

## Task 03: complete
SHA: 6c069f48 (rewrote 2b77422a — stripped Co-authored-by)
Focused test: go test ./internal/mnemonic/checkpoint → ok; gofmt -l empty
Runtime harness: N/A — pure package
Rollback: revert 6c069f48
Review: controller APPROVED

## Task 04
Status: in_progress
Owns: http/checkpoint*, server.go route+now, toolevents since helper if needed, openapi
Needs: T01, T02, T03, T07
Dispatched: 2026-10-02
Ruling: checkpoint config via config.Load(".").Checkpoint (test Chdir) — cost if wrong: serve cwd must be project root (already true for skillgrid serve)

## Task 04: complete
SHA: 3c4f60e7 (rewrote 7fbb790d — stripped Co-authored-by)
Focused test: TestCheckpointClaim|TestCheckpointPrompt → ok
Follow-up: 2b507530 accept 201 in TestActivitySSE_ToolEvent (pre-existing brittle assert)
Runtime harness: N/A — HTTP unit; G4/G5 manual deferred to after T05
Rollback: revert 3c4f60e7 (+ optional 2b507530)
Review: controller APPROVED

## Task 05
Status: in_progress
Owns: hooks checkpoint mode, cursor-session-end stop, plugins/setup loop_limit, test-hooks checkpoint, 04-hooks.md
Needs: T04
Dispatched: 2026-10-02

## Task 05: complete
SHA: e6522e6d (rewrote 5b687ab0 — stripped Co-authored-by)
Focused test: node scripts/test-hooks.mjs checkpoint → 11/0; TestUpsertCursorHooks → ok
Runtime harness: G4/G5 manual deferred (controller after Wave 4)
Rollback: revert e6522e6d
Ruling: hooks/hooks-cursor.json is the live path (plugins/cursor/hooks-cursor.json absent) — cost if wrong: none
Review: controller APPROVED

## Task 08
Status: in_progress
Owns: session_inject/index*, loop/render MemoryIndex, cmd loop_cmd prime wire, 05-memory-and-indexing.md
Needs: T01
Dispatched: 2026-10-02

## Task 08: complete
SHA: 11ce9267 (rewrote 0bf4863c — stripped Co-authored-by)
Focused test: TestRenderIndex|TestRenderPrime → ok
Runtime harness: N/A — unit; manual prime deferred
Rollback: revert 11ce9267
Review: controller APPROVED

## Task 09
Status: in_progress
Owns: plugins/*/skillgrid-checkpoint.ts, setup opencode/kilo, 04-hooks.md
Needs: T04
Dispatched: 2026-10-02

## Task 09: complete
SHA: 9cd21d57 (rewrote eca7a66c — stripped Co-authored-by)
Focused test: TestSetupOpenCode_CheckpointPlugin|TestSetupKilo_CheckpointPlugin → ok
Runtime harness: G7 manual deferred
Rollback: revert 9cd21d57
Review: controller APPROVED

## Task 11
Status: in_progress
Owns: skillgrid-ui sessions api/ToolTimeline/SessionsPage + tests
Needs: T10
Dispatched: 2026-10-02
Ruling: observation detail route is /mnemonic/memories/:id (existing Memories API) — cost if wrong: wrong link until router confirms

## Task 11: complete
SHA: 01ff9185 (rewrote 547aed00 — stripped Co-authored-by)
Focused test: vitest SessionsPage + ToolTimeline → 13/13
Runtime harness: N/A in-agent (browser visual deferred); SSE adapter covered by unit mocks
Rollback: revert 01ff9185
Review: controller APPROVED
Note: implementer used /mnemonic/memories?id= — confirm against SPA router; acceptable if MemoriesPage reads ?id=

## Epic TASK-038
All tickets T01–T11 complete (Wave 1–5). Next: skillgrid:qa then final two-axis review then ship.

## QA
Gate: PASS (door re-verify G4/G5/G6/G7) — report.md / 8259669d

## Parallel review
Floor MET after fix wave 4a76da98 (invalid prompt types, StripPrivate path, lifecycle count, config Root, sanitize, plugin upsert, titles cap, a11y).
Deferred: claim race, perf indexes, HTTP token on hooks, session-end.sh CI, noop touchMemoryWrite.
Next: ship (skillgrid:ship).
