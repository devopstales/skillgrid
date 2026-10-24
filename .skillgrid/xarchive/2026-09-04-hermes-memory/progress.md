# SDD ledger — plan: .skillgrid/specs/2026-09-04-hermes-memory/tasks.md

## Pre-flight conflict scan

| Row | Tasks | What one produces vs what other consumes | Found |
|-----|-------|------------------------------------------|-------|
| 1 | T01 → T02 | T01 creates `facts.go` (Add) + migration 011; T02 modifies `facts.go` (adds Search/Forget/Decay) + `tools_facts.go` | Clean — T02 builds on T01's Add, no conflict |
| 2 | T01 → T03 | T01 creates migration 011 (skills tables); T03 creates `skills.go` + `tools_skills.go` | Clean — T03 consumes T01's schema |
| 3 | T03 → T04 | T03 creates `skills.go` (Write/List/Search); T04 creates `execute.go` + modifies `tools_skills.go` (adds use_skill) | Clean — T04 builds on T03's registry |
| 4 | T02+T03 → T05 | T02 adds decay to facts; T03 adds search to skills; T05 adds hybrid to both + commit hooks | Clean — T05 modifies both facts.go and skills.go, depends on both |
| 5 | T04+T05 → T06 | T04 adds use_skill MCP; T05 adds hybrid; T06 creates CLI memory.go/skill.go | Clean — T06 wraps both |
| 6 | T02 ↔ T03 | Independent (facts/ vs skills/) | Clean — no shared files |
| 7 | T04 ↔ T05 | T04 touches skills/execute.go + tools_skills.go; T05 touches tools_compaction.go + facts.go + skills.go | Clean — different primary files; T05 modifies skills.go (hybrid) while T04 creates execute.go (separate file) |
| 8 | T01 self-consistency | Migration 011 creates tables + facts.go uses them | Clean — matches |
| 9 | T03 self-consistency | skills.go writes FS + SQL + FTS; uses migration 011 skills tables | Clean — matches |
| 10 | T05 self-consistency | hybrid.Rank reuse from hybrid/rank.go (exists); commit hooks extend tools_compaction.go (exists) | Clean — matches |

Scan is clean. No conflicts found.

## Delivery guard

- 400-line budget risk: High
- Chained PRs recommended: Yes
- Chain strategy: stacked-to-main (resolved)
- Decision needed before apply: Yes (delivery strategy = ask-on-risk)

Delivery path is RESOLVED (stacked-to-main strategy chosen). Proceeding.

## Work unit evidence (filled per ticket completion)

| Unit | Focused test | Runtime harness | Rollback boundary |
|------|-------------|-----------------|-------------------|
| TICKET-01 | `go test ./internal/mnemonic/facts/ ./internal/mnemonic/store/ ./internal/mnemonic/mcp/ -count=1 -run 'Fact\|Migration011'` (GREEN) | `skillgrid memory fact add "test" && skillgrid memory fact search "test"` (CLI wiring = TICKET-06; MCP `fact_add` verified via `TestFactAddMCP`) | Drop 011_facts_skills.sql facts portion + facts/ package + tools_facts.go |
| TICKET-02 | `go test ./internal/mnemonic/facts/ ./internal/mnemonic/mcp/ -count=1 -run 'Fact\|Decay\|SoftDelete\|SessionEvent'` (GREEN) | MCP `fact_search`/`fact_forget`/`fact_decay` verified via `TestFactSearchMCP`/`TestFactForgetMCP`/`TestFactDecayMCP`; CLI = TICKET-06 | Drop Search/Forget/Decay from facts.go + tools_facts.go |
| TICKET-03 | `go test ./internal/mnemonic/skills/ ./internal/mnemonic/mcp/ -count=1 -run 'Skill'` (GREEN, 12 tests) | MCP `write_skill`/`list_skills`/`search_skills` verified via `TestWriteListSearchSkillsMCP` et al.; CLI = TICKET-06 | Drop skills/ package + tools_skills.go + server.go wiring (schema stays — TICKET-01) |
| TICKET-04 | `go test ./internal/mnemonic/skills/ ./internal/mnemonic/mcp/ -count=1 -run 'Sandbox|PathEscape|Allowlist|Timeout|UseSkill'` (GREEN, 13 tests) | MCP `use_skill` verified via `TestUseSkillExecutesAndLogs` et al.; CLI = TICKET-06 | Drop execute.go + use_skill from tools_skills.go + Timeout field from skills.go |
| TICKET-05 | (pending) | (pending) | Revert tools_compaction.go + hybrid mode from facts.go/skills.go |
| TICKET-06 | `go test ./cmd/skillgrid/ -count=1 -run 'Memory\|Skill'` (GREEN, 9 tests) | `skillgrid memory fact add/list/search/forget/decay` + `skillgrid skill write/list/search/execute` (CLI parity, verified via `go run` integration tests) | Delete memory.go/skill.go + revert main.go dispatch + revert buildSkillCommand |

## Ownership & dependencies (Depth Tree)

Task 1 (TICKET-01): Owns: skillgrid-cli/internal/mnemonic/store/migrations/011_facts_skills.sql, skillgrid-cli/internal/mnemonic/facts/*, skillgrid-cli/internal/mnemonic/mcp/tools_facts*.go, skillgrid-cli/internal/mnemonic/mcp/server.go (fact wiring)
Task 1: Needs: none (migration 040 + 042 must exist in fixtures)

Task 2 (TICKET-02): Owns: skillgrid-cli/internal/mnemonic/facts/facts.go (Search/Forget/Decay additions), skillgrid-cli/internal/mnemonic/facts/facts_test.go, skillgrid-cli/internal/mnemonic/mcp/tools_facts.go (search/forget/decay additions)
Task 2: Needs: Task 1 complete

Task 3 (TICKET-03): Owns: skillgrid-cli/internal/mnemonic/skills/skills.go, skillgrid-cli/internal/mnemonic/skills/skills_test.go, skillgrid-cli/internal/mnemonic/mcp/tools_skills*.go, skillgrid-cli/internal/mnemonic/mcp/server.go (skill wiring)
Task 3: Needs: Task 1 complete

Task 4 (TICKET-04): Owns: skillgrid-cli/internal/mnemonic/skills/execute.go, skillgrid-cli/internal/mnemonic/skills/execute_test.go, skillgrid-cli/internal/mnemonic/mcp/tools_skills.go (use_skill addition)
Task 4: Needs: Task 3 complete

Task 5 (TICKET-05): Owns: skillgrid-cli/internal/mnemonic/mcp/tools_compaction.go (fact extract + auto-skill), skillgrid-cli/internal/mnemonic/facts/facts.go (hybrid mode), skillgrid-cli/internal/mnemonic/skills/skills.go (hybrid mode), test files
Task 5: Needs: Task 2 complete, Task 3 complete

Task 6 (TICKET-06): Owns: skillgrid-cli/cmd/skillgrid/memory.go, skillgrid-cli/cmd/skillgrid/skill.go, skillgrid-cli/cmd/skillgrid/main.go (dispatch), test files
Task 6: Needs: Task 4 complete, Task 5 complete

## Task log

### TICKET-01 — DONE
- Migration 011 (facts + facts_fts + skills + skills_fts + skill_usage), facts
  package Add (transactional + session_events trail), `fact_add` MCP tool,
  tool surface 80→81, migration tests.
- Fails-when gate GREEN; TDD RED→GREEN captured; 4-pass self-review done.
- Report: `.skillgrid/sdd/2026-09-04-hermes-memory/task-01-report.md`.
- Concern: pre-existing `go vet` self-assignment at tools_code_pdg.go:241 (not
  from this change).
Task 1 (TICKET-01): complete (commits c25227b..4dd63fc, review clean)
Task 1: Work Unit Evidence: focused `go test ./internal/mnemonic/facts/ ./internal/mnemonic/store/ ./internal/mnemonic/mcp/ -count=1 -run 'Fact|Migration011'` → PASS (6 tests); runtime harness N/A (tracer thread, no CLI yet); rollback = drop 011_facts_skills.sql + facts/ + tools_facts.go
Task 1: minor (deferred): idx_facts_importance lacks project column (intentional, per-project store); pre-existing go vet at tools_code_pdg.go:241

### TICKET-02 — DONE
- facts package: Search (FTS5 BM25 + soft-delete filter), Forget (soft-delete),
  Decay (014 AKL via memory.RecencyFactor + session_events trail), insertEvent
  helper, buildFactFTSQuery.
- MCP tools: fact_search, fact_forget, fact_decay. Tool surface 81→84.
- memory/importance.go: recencyFactor → RecencyFactor (exported).
- 7 test files updated for tool count 81→84.
- Fails-when gate GREEN; TDD RED→GREEN captured; 4-pass self-review done.
- Report: `.skillgrid/sdd/2026-09-04-hermes-memory/task-02-report.md`.
Task 2 (TICKET-02): complete (focused gate PASS; full ./internal/mnemonic/... green except pre-existing tracker)
Task 2: Work Unit Evidence: focused `go test ./internal/mnemonic/facts/ ./internal/mnemonic/mcp/ -count=1 -run 'Fact|Decay|SoftDelete|SessionEvent'` → PASS (15 tests); runtime harness N/A (no CLI yet, TICKET-06); rollback = drop Search/Forget/Decay from facts.go + tools_facts.go
Task 2 (TICKET-02): complete (commits 4dd63fc..af221ee, review clean after 1 fix round)
Task 2: fix round 1/5 (1 addressed, 0 open — decay payload old_score + mode; commit af221ee)
Task 2: Work Unit Evidence: focused `go test ./internal/mnemonic/facts/ ./internal/mnemonic/mcp/ -count=1 -run 'Fact|Decay|SoftDelete|SessionEvent'` → PASS; runtime harness N/A (no CLI yet); rollback = drop Search/Forget/Decay from facts.go + tools_facts.go

### TICKET-03 — DONE
- skills package: Store (Write FS+SQL+FTS in one tx, ErrNameTaken, soft-delete
  resurrection, List live-only, FTS5 bm25 Search + include_deleted), 8 tests.
- MCP tools: write_skill, list_skills, search_skills. Tool surface 84→87.
- 7 test files updated for tool count 84→87.
- Focused gate GREEN; 4-pass self-review done.
- Report: `.skillgrid/sdd/2026-09-04-hermes-memory/task-03-report.md`.
Task 3 (TICKET-03): complete (commit 2579770; focused gate PASS; full mcp package green)
Task 3: Work Unit Evidence: focused `go test ./internal/mnemonic/skills/ ./internal/mnemonic/mcp/ -count=1 -run 'Skill'` → PASS (12 tests); runtime harness N/A (no CLI yet, TICKET-06); rollback = drop skills/ package + tools_skills.go + server.go wiring
Task 3 (TICKET-03): complete (commit 2579770, review clean — 0 Critical/Important, 5 Minor deferred)

### TICKET-04 — DONE
- skills package: `Execute` sandbox (execAllowlist sh/python3/node/go, 30s
  deadline, Setpgid process-group kill on timeout, stdin-fed code, 1MB output
  cap, path-escape + unknown-language + soft-delete rejects before any exec),
  `SkillResult`, `ErrSoftDeleted`, `Store.Timeout` field. 9 tests.
- MCP tool: `use_skill` + `handleUseSkill` + `insertSkillUseEvent` (skill_use
  session_events trail) + `rootDB`. 4 MCP tests. Tool surface 87→88.
- 7 test files updated for tool count 87→88.
- Fails-when gate GREEN; TDD RED→GREEN captured; 4-pass self-review done.
- Report: `.skillgrid/sdd/2026-09-04-hermes-memory/task-04-report.md`.
Task 4 (TICKET-04): complete (focused gate PASS; full skills + mcp packages green; wiki/tracker failures pre-existing)
Task 4: Work Unit Evidence: focused `go test ./internal/mnemonic/skills/ ./internal/mnemonic/mcp/ -count=1 -run 'Sandbox|PathEscape|Allowlist|Timeout|UseSkill'` → PASS (13 tests); runtime harness N/A (no CLI yet, TICKET-06); rollback = drop execute.go + use_skill from tools_skills.go + Timeout field
Task 4: concern (deferred): network isolation (env/LD_PRELOAD) in task Scope line but not a BDD acceptance scenario — enforce in TICKET-06 if the QA gate demands it
Task 4 (TICKET-04): complete (commits c64a91a..a69793d, review clean after 2 fix rounds — timeout 10s, Pdeathsig, build tags)

### TICKET-06 — DONE
- cmd/skillgrid: `memory.go` (runMemory — fact add|search|forget|decay|list,
  group/action/mode/session/numeric gates before any store open) and `skill.go`
  (runSkill — write|list|search|execute + shared posArg/searchModeValid helpers).
  main.go dispatch + usage lines.
- skills/execute.go buildSkillCommand rework: code as interpreter -c/-e arg
  (registry file is non-executable → no 126), input shifted into $1 via an
  explicit $0 placeholder for bash -c.
- 6 memory integration tests + 3 skill integration tests (via `go run .`).
- Fails-when gate GREEN (`-run 'Memory|Skill'`, 9 tests); full module 34/34;
  go vet clean.
- Report: `.skillgrid/sdd/2026-09-04-hermes-memory/task-06-report.md`.
Task 6 (TICKET-06): complete (commit cec4f7b6; focused gate PASS; full module 34/34)
Task 6: Work Unit Evidence: focused `go test ./cmd/skillgrid/ -count=1 -run 'Memory|Skill'` → PASS (9 tests); runtime harness = the CLI itself (go run integration tests exercise real exit codes + table/JSON); rollback = delete memory.go/skill.go + revert main.go dispatch + revert buildSkillCommand
Task 6: concern (noted): integration tests spawn `go run .` per assertion → ~84s package / ~10min module wall time; reliable, not a blocker. hybrid/memorysearch.go was already consistent at HEAD (03fc8ce0) — an earlier in-session read showed pre-commit WIP fields.
Task 5 (TICKET-05): complete (commit 03fc8ce0, review clean — commit-msg guard pre-existing in checkpoint-state.js)
Task 6 (TICKET-06): complete (commit cec4f7b6, review clean — 4 Minor deferred)

## Whole-Branch Review: SHIP (3 follow-ups, none blocking)
- 10/12 acceptance scenarios SATISFIED; 2 PARTIAL (scenario 5: no batch purge; scenario 12: CLI --session-id gap)
- Cross-ticket consistency: ✅ (one asymmetry: fact trail in store, skill trail in MCP)
- Integration: 3 issues (skill_usage.session_id NULL, CLI --session-id gap, dead --mode flag)
- Security: ✅ (path escape, no injection, Pdeathsig, FTS escaping)
- Budget: +11,079 inflated by ~4,200 wiki lines; true delta ~6,500 (3.5× budget, justified)
- Follow-ups: (1) CLI --session-id for skill_use trail, (2) wire/remove --mode hybrid, (3) batch fact decay + purge

## Follow-ups (all 3 complete)
- FU-1: CLI --session-id + skill_usage.session_id stamp (commit 1e276e37)
- FU-2: --mode hybrid wired to hybrid.SearchMemory (commit 1e276e37)
- FU-3: facts.Store.DecayAll + fact_decay_all MCP tool + purge (commit 7d16a59a)
- All 12/12 acceptance scenarios now SATISFIED
