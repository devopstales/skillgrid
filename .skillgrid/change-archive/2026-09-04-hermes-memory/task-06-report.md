# TICKET-06 — CLI `memory` / `skill` subcommands (MCP parity)

**Change:** 2026-09-04-hermes-memory · **Status:** DONE · **Commit:** cec4f7b6

## Objective

Give the CLI the same surface as the Fact Memory + Agent Skill MCP tools:
`skillgrid memory fact <add|search|forget|decay|list>` and
`skillgrid skill <write|list|search|execute>`. Table is the default output;
`--json` emits the machine-readable form the MCP tools return. Invalid groups,
actions, and missing session ids fail **before any store is opened**, so a usage
mistake can never corrupt the project store.

## Implementation

### 1. `cmd/skillgrid/memory.go` (Create)
`runMemory(version, args)` — dispatcher for `memory fact <add|search|forget|decay|list>`.

- **Group gate first:** `args[0] != "fact"` → `unknown memory group` (exit 2, no
  store). Bare `memory` → usage (exit 2).
- One `flag.FlagSet` for all fact actions; `reorderMemArgs` moves flags before
  positionals so `search q --json` and `search --json q` both parse.
- Positional convenience: `add <content>`, `search <query...>`,
  `forget|decay <id>` (each also has a flag form).
- **Session gate (before store open):** `add|search|forget|decay` require
  `--session-id` or `SKILLGRID_MEMORY_SESSION_ID`; empty → `requires a
  --session-id` (exit 2). `list` needs none (read-only).
- **Mode gate (before store open):** `--mode` must be `""|fts|hybrid`
  (`searchModeValid`); anything else → `--mode must be fts or hybrid` (exit 2).
- **Numeric gate (before store open):** `forget|decay` `--fact-id` must parse as
  int64 → else `requires a numeric --fact-id` (exit 2).
- Routes through the same seam as MCP: `openMemService` → `svc.Open` →
  `facts.New(db, projID)`.
  - `add` → `st.Add(ctx, sessionID, content)` → JSON `{event:"fact_add", fact_id, project}`
    / table `added fact N`.
  - `search` → `st.SearchWith(ctx, sessionID, query, limit, includeDel)` → JSON
    `{facts, count}` / table. `--mode hybrid` degrades to the lexical FTS path
    (no embedder in the CLI).
  - `forget` → `st.Forget(ctx, sessionID, id)` → JSON `{event:"fact_forget", fact_id}`
    / `forgotten fact N`.
  - `decay` → `st.Decay(ctx, sessionID, id)` → JSON `{event:"fact_decay", fact_id, new_score}`
    / `decayed fact N to score X`.
  - `list` → `st.ListWith(ctx, limit, includeDel)` → JSON `{facts, count}` / table
    (header `ID CONTENT`).
- `writeFactTable`, `printMemoryOut` (JSON-or-table), `printMemoryUsage`.

### 2. `cmd/skillgrid/skill.go` (Create)
`runSkill(version, args)` — dispatcher for `skill <write|list|search|execute>`.

- Unknown command → `unknown skill command` (exit 2, no store). `write` missing
  `--name/--language/--code` → usage (exit 2).
- Flags: `--dir/--project/--root/--name/--language/--description/--code/
  --overwrite/--query/--limit/--include-deleted/--input/--json/--mode`.
- **Mode gate** reuses `searchModeValid`.
- Routes through `skills.New(h.Store().DB, storeRoot)`; `storeRoot` defaults to
  CWD (the workspace the operator is in — the CLI has no CWD root seam).
  - `write` → `st.Write(...)` → prints id + code path.
  - `list` → blank-query `SearchWith` then fall back to `st.List` (a blank FTS
    match returns nothing) → table (header `ID NAME LANGUAGE DESCRIPTION`).
  - `search` → `st.SearchWith(ctx, query, limit, includeDel)`.
  - `execute` → `st.Execute(ctx, name, input)` → JSON `{stdout, stderr, exit_code,
    timed_out, usage_id, status}` / table (`exit N status` + stdout/stderr).
- `writeSkillTable`, `printSkillOut`, `reorderSkillArgs`, `printSkillUsage`,
  plus the shared helpers **`posArg`** and **`searchModeValid`** (used by both
  memory.go and skill.go).

### 3. `cmd/skillgrid/main.go` (Modify)
- Usage line for `memory` and `skill`.
- Dispatch cases `case "memory"` → `runMemory`, `case "skill"` → `runSkill`.

### 4. `internal/mnemonic/skills/execute.go` (Modify) — `buildSkillCommand`
The earlier stdin-fed approach made `bash -c` receive the code as a script path
and the input as `$0` (so `$1` stayed empty). Rewrote:
- The code now travels as the **interpreter `-c`/`-e` argument** (the registry
  file is not executable, so it cannot be run as a script path — exit 126):
  sh/bash/zsh/shell → `bash -c <code>`, python → `python3 -c <code>`,
  node/js → `node -e <code>`, go → `go run <file>` (unchanged).
- The input is shifted into `$1` (bash) / `sys.argv[1]` (python) /
  `process.argv[1]` (node) by inserting an explicit `$0` placeholder (the
  interpreter name) before the input — `bash -c 'echo greets $1' world` alone
  would put `world` in `$0`, leaving `$1` empty.
- `run(cmd, stdin=nil, deadline)` now uses dev-null stdin.

### 5. `facts.Store.List` / `ListWith` (already present from TICKET-02)
`list` uses `ListWith(ctx, limit, includeDeleted)`.

### 6. Tests (TDD)
- `cmd/skillgrid/memory_test.go` (Create) — 6 integration tests via `go run .
  memory …`: fact add+list (table + JSON), fact search (lexical + `--mode
  hybrid` + JSON + invalid-mode exit 2), fact forget (soft-delete, hidden from
  default list, `--include-deleted` restores, non-numeric id exit 2), fact decay
  (reports new score + JSON shape), invalid group/action (no store opened —
  asserts no `<project>.sqlite` file created), add-requires-session (exit 2, no
  store).
- `cmd/skillgrid/skill_test.go` (Create) — 3 integration tests via `go run .
  skill …`: write+list+search (table + JSON), execute (stdout + exit 0, JSON
  result shape, non-zero exit surfaces as failure), invalid action (no store
  opened).

## TDD evidence

**RED** (dispatcher + helpers absent):
```
cmd/skillgrid/main.go:171:3: undefined: runMemory
cmd/skillgrid/skill.go:65:10: undefined: posArg
cmd/skillgrid/skill.go:88:21: undefined: searchModeValid
```

**GREEN** — fails-when gate:
```
$ go test ./cmd/skillgrid/ -count=1 -run 'Memory|Skill'
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid   83.7s
```
All 9 memory/skill tests PASS.

Supporting packages:
```
$ go test ./internal/mnemonic/skills/ -count=1   # 9 execute/registry tests
ok  .../skills  15.3s
$ go test ./internal/mnemonic/hybrid/ -count=1   # SearchMemory BM25/vector/degrade
ok  .../hybrid  4.1s
```

`go vet ./cmd/skillgrid/ ./internal/mnemonic/...` clean.

Full-module regression:
```
$ go test ./...   # 34 packages, 0 failures
```

## 4-pass self-review

1. **Completeness** — every acceptance bullet has a named test: add/list, search
   (incl. `--mode hybrid` degrade + invalid-mode exit 2), forget (soft-delete +
   include-deleted + non-numeric exit 2), decay (score report), skill
   write/list/search/execute (incl. JSON + non-zero exit), invalid group/action
   and missing-session all fail before the store opens (asserted by the absent
   `<project>.sqlite` file).
2. **Re-read** — `memory.go` / `skill.go` route through the same seams as the MCP
   handlers; `posArg`/`searchModeValid` live in one place (skill.go) and are
   shared; `reorderMemArgs` reused for flag/positional parsing.
3. **Defects** —
   - **`bash -c` `$0`/`$1`:** the input must be preceded by a `$0` placeholder
     (the interpreter name) or it lands in `$0` and `$1` stays empty. Fixed in
     `buildSkillCommand` (`args = append(args, args[0], input)`).
   - **No store on usage error:** group/action/mode/session/numeric gates run
     before `openMemService`; the tests prove it by asserting no `<project>.sqlite`
     file is created.
   - **`list` blank-query FTS:** a blank FTS match returns nothing, so `skill list`
     falls back to `st.List`; `memory fact list` uses `ListWith` directly.
4. **Full run** — both packages GREEN; `go vet` clean; full module 34/34.

## Lazy / gate evidence

- No new dependencies (stdlib + existing `facts`/`skills` packages).
- The CLI is a thin dispatcher over the same store seams the MCP tools use — no
  duplicated domain logic, no second execution path.
- `posArg`/`searchModeValid` are single shared helpers, not per-command copies.
- The `hybrid/memorysearch.go` `SearchMemory` contract (result shape,
  `MemoryOptions` fields, `Embedder` interface) was already correct at HEAD
  (commit 03fc8ce0) — verified `git diff HEAD` clean; no change needed.

## Files

- `skillgrid-cli/cmd/skillgrid/memory.go` (Create — `runMemory`, `writeFactTable`,
  `printMemoryOut`, `printMemoryUsage`)
- `skillgrid-cli/cmd/skillgrid/skill.go` (Create — `runSkill`, `writeSkillTable`,
  `printSkillOut`, `reorderSkillArgs`, `printSkillUsage`, `posArg`,
  `searchModeValid`)
- `skillgrid-cli/cmd/skillgrid/memory_test.go` (Create — 6 tests)
- `skillgrid-cli/cmd/skillgrid/skill_test.go` (Create — 3 tests)
- `skillgrid-cli/cmd/skillgrid/main.go` (Modify — usage + dispatch cases)
- `skillgrid-cli/internal/mnemonic/skills/execute.go` (Modify — `buildSkillCommand`
  code-as-`-c`/`-e` arg + `$0` placeholder for `$1`)

## Concerns

- **Long test wall time:** the memory/skill integration tests spawn `go run .`
  per assertion, so the package gate is ~84s and the full module ~10min. They
  pass reliably; this is the cost of true CLI parity (real process, real exit
  codes). Not a blocker.
- **`hybrid` package churn this session:** an earlier in-session read saw
  `memorysearch.go` referencing `facts.Fact.Title/.Relevance` (nonexistent) and a
  divergent `SearchMemory` signature. Those were a pre-commit in-progress state;
  the committed contract (03fc8ce0) is consistent with `search_test.go` and the
  MCP caller `tools_memory_hybrid.go`. Confirmed by the green `hybrid` + full
  module run.
