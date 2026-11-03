# SDD ledger — plan: .skillgrid/specs/2026-10-02-mnemonic-project-init/tasks.md

Workspace: `.skillgrid/sdd/2026-10-02-mnemonic-project-init/` (manual slug; `sdd-workspace` basename of tasks.md/blueprint.md would collide)

## Delivery Guard
- Decision needed before apply: No
- Chained PRs recommended: No
- Chain strategy: pending (guard not triggered — Medium budget, no Yes flags)
- 400-line budget risk: Medium
- Ruling: proceed on release/2; resolve chain strategy at ship if still pending

## Pre-flight conflict scan
| Pair / task | Shared surface | Finding | Ruling |
|---|---|---|---|
| T02↔T03↔T04 | `init_cmd.go` | Serial after T01 per tasks.md even though graph only needs T01 | Wave order 02→03→04 mandatory |
| T01→T05 | onboarding skill | T05 can follow T01 but should wait for T04 so skill documents complete CLI | Dispatch T05 after T04 |
| T01 alone | `init_cmd.go`, `main.go`, tests | Help/boot/fatal gates only; index/ingest counts may be zero until later tickets | Door check: BootFile path + exit 0; counts may be 0 |
| Blueprint | no `## Plan Review` | Written before plan-review gate | Proceed; execution review loop still applies |

## Ownership
- Task 1: Owns: `skillgrid-cli/cmd/skillgrid/init_cmd.go`, `init_cmd_test.go`, `main.go` · Needs: none
- Task 2: Owns: `init_boot.go`, `init_cmd.go`, `init_cmd_test.go` · Needs: Task 1
- Task 3: Owns: `init_cmd.go`, `init_cmd_test.go` · Needs: Task 1 (serial after Task 2)
- Task 4: Owns: `init_ingest.go`, `init_cmd.go`, `init_cmd_test.go` · Needs: Task 1 (serial after Task 3)
- Task 5: Owns: `.agents/skills/lifecycle/onboarding/SKILL.md`, `init_cmd_test.go` · Needs: Task 1 (dispatch after Task 4)

## Status
- Queued-behind cleared: `2026-10-02-mnemonic-memory-checkpoint` archived/shipped
- Branch: release/2 @ f739d371
- Task 1: dispatched

## Task 1: complete
- Commit: b5239413
- Review: APPROVED (8047bbfd) — spec PASS, quality approved
- Focused test: `go test ./cmd/skillgrid -run 'TestInitHelpListsFlags|TestInitWritesBootFileAndReportsCounts|TestInitBootFileWriteFailureIsFatal'` → PASS
- Runtime harness: N/A — CLI + store, no HTTP (per work-unit table)
- Rollback boundary: revert b5239413 (init_cmd.go, init_cmd_test.go, main.go dispatch)
- Ruling: park Important help-test drift until T02 shared helper if needed — non-blocking

## Task 2: dispatched

## Task 2: complete
- Commit: 9dd840f5
- Review: APPROVED (f109a306) — spec PASS, quality approved
- Focused test: TestInitUpserts*|Force*|T01 trio → PASS
- Runtime harness: N/A — CLI file upsert
- Rollback: revert 9dd840f5 (init_boot.go + init_cmd wiring)
- Rulings: park orphan-marker / force assert minors

## Task 3: dispatched

## Task 3: complete
- Commit: 03439ce7
- Focused test: `go test ./cmd/skillgrid -count=1 -run 'TestInit'` → PASS (incl. TestInitIndexesTheProject, TestInitIndexFailureIsNonFatal)
- Runtime harness: N/A — CLI + existing RunCodeIndex
- Rollback: revert 03439ce7 (init_cmd.go index wire + tests)
- Note: stub var named `initRunIndex` (CLI already has `runIndex`)

## Task 3: complete
- Commit: 03439ce7
- Review: APPROVED (8b12883b)
- Focused test: TestInit* → PASS
- Runtime harness: N/A — in-process RunCodeIndex
- Rollback: revert 03439ce7

## Task 4: dispatched

## Task 3: complete
- Commit: 03439ce7
- Review: APPROVED (8b12883b)
- Focused test: TestInit* → PASS
- Runtime harness: N/A — in-process RunCodeIndex
- Rollback: revert 03439ce7

## Task 4: dispatched

## Task 4: complete
- Commit: 8763f9b6 (rewrote off polluted ANSI tip)
- Review: APPROVED (3a1d71ba)
- Focused test: TestInitIngests*|Skips*|Missing*|Rejects* → PASS; full TestInit* green after fix
- Runtime harness: N/A — memory.Save via OpenForDirectory
- Rollback: revert 8763f9b6 (init_ingest.go + wiring)
- Note: first GREEN run failed once; fixed then re-verified twice

## Task 5: dispatched

## Task 5: complete
- Commit: 2324c38e (clean commit-tree rewrite; no ANSI / no Co-authored-by)
- Report: task-5-report.md
- Focused test: `go test ./cmd/skillgrid -count=1 -run 'TestOnboardingSkill|TestInit'` → PASS
- Runtime harness: N/A — skill markdown + contract tests
- Rollback: revert 2324c38e (SKILL.md Step 9 + TestOnboardingSkill*)

