# TICKET-03 — Agent Skill registry (FS + SQL + FTS) + Write/List/Search + MCP tools

**Change:** 2026-09-04-hermes-memory · **Work unit:** 2 (PR 2) · **Status:** DONE

## Implementation

### 1. `skills` package — registry Store (Create)
`skillgrid-cli/internal/mnemonic/skills/skills.go`

- `Store{db, root}` + `New(db, root)` — wraps the 011-migrated project store's
  database handle and the workspace root (the `.skillgrid/` scratch-tree owner,
  ContentPlane convention).
- `Skill` struct — the read model for List/Search (id, name, language,
  description, code_path, created_at, updated_at).
- `ErrNameTaken` — sentinel for a `write_skill` name collision with
  `overwrite=false`; surfaced verbatim by the MCP tool.
- `knownLanguages` — language→extension map mirroring the code-index language
  set (`memory.extLanguage` aliases: bash/zsh/shell → sh, md → md, yml → yaml).
  Unknown languages are rejected before any FS or SQL side effect.
- `validName(name)` — single safe basename: no `/`, `\`, NUL, `..`; only ASCII
  alnum, `-`, `_`, `.`; not `.` / `..` (path escape rejects without exec).
- `Write(ctx, name, language, description, code, overwrite) (int64, error)`:
  - Validates name + language up front (no side effects on rejection).
  - Looks up an existing row by name (UNIQUE). Live + `overwrite=false` →
    `ErrNameTaken` touching nothing (not even the FS). Soft-deleted or
    `overwrite=true` → the row is updated in place / resurrected
    (`deleted_at = NULL`, new language/description/code_path/updated_at).
  - SQL + FS inside one transaction: INSERT/UPDATE first, then `MkdirAll` +
    `WriteFile` the payload to `.skillgrid/files/skills/{name}.{ext}` (NOT
    `.agents/skills`). If the FS write fails the tx rolls back (row is the
    identity, file is the payload). If `tx.Commit` fails the just-written file
    is removed. The FTS row is kept in sync by the 011 triggers.
- `List(ctx)` — every live (`deleted_at IS NULL`) skill, newest first
  (`updated_at DESC, id DESC`).
- `Search(ctx, query, limit)` / `SearchWith(ctx, query, limit, includeDeleted)` —
  FTS5 over `skills_fts` (name + description), `bm25()`-ranked, soft-deleted
  excluded by default, `include_deleted` lifts the filter (audit read).
- `buildSkillFTSQuery(query)` — any-term recall: each token double-quoted
  (FTS special chars can't break syntax), OR-joined; blank query → no results.
- `supportedLanguages()` — sorted, deduped language list for the
  unknown-language error message.

### 2. MCP tools — write_skill / list_skills / search_skills (Create)
`skillgrid-cli/internal/mnemonic/mcp/tools_skills.go`

- `registerSkillTools(s)` — registers all three tools.
- `writeSkillTool` / `handleWriteSkill` — required `name`, `language`, `code`;
  optional `description`, `overwrite` (default false), `project` (CWD resolve).
  Returns `{skill_id, name, project, event: "skill_write"}`.
- `listSkillsTool` / `handleListSkills` — returns `{skills: [...], count}`.
- `searchSkillsTool` / `handleSearchSkills` — required `query`; optional
  `limit` (default 20), `include_deleted` (default false), `project`.
  Returns `{skills: [...], count}`.
- All handlers follow the existing `tools_facts.go` pattern: `rootService()` →
  `projectIDFor` → `svc.Open(projectID)` → `skills.New(h.Store().DB, h.Root())`.
- Tool surface 84 → 87.

### 3. Server wiring (Modify)
`skillgrid-cli/internal/mnemonic/mcp/server.go`

- `registerSkillTools(s)` added to both `Start()` and `NewServer()`, matching
  the fact-tools wiring from TICKET-01.

### 4. Tool-count assertions (Modify)
Seven test files updated 84 → 87 (additive surface; no tools removed/renamed):
- `server_test.go` — `want` slice (+ `write_skill`, `list_skills`, `search_skills`) + count
- `tools_code_affected_test.go`
- `tools_code_knowledge_test.go`
- `tools_memory_governance_test.go`
- `tools_memory_layer_args_test.go`
- `tools_memory_layer_test.go`
- `tools_memory_retrieval_test.go`

### Schema (build-on, no new migration)
Migration `011_facts_skills.sql` (TICKET-01) already creates `skills` (name
UNIQUE, deleted_at), `skills_fts` (FTS5 external-content over
name/description/code_path, kept in sync by triggers), and `skill_usage`.
This ticket is purely additive Go code over that schema; migrations ship via
`//go:embed migrations/*.sql` (`store.go:27`).

## Acceptance mapping

| Acceptance criterion (step-03) | Evidence |
|---|---|
| Write list search and execute Agent Skills (write/list/search portion) | `TestWriteSkillStoresFSAndSQL`, `TestListReturnsNonDeletedSkills`, `TestSearchExcludesSoftDeleted` + MCP `TestWriteListSearchSkillsMCP` |
| Overwrite false rejects name collision | `TestWriteOverwriteFalseRejects`, `TestWriteSkillOverwriteFalseRejectsMCP` |
| Path escape or unknown language rejects without exec | `TestWriteUnknownLanguageRejects`, `TestWritePathEscapeRejects`, `TestWriteSkillUnknownLanguageMCP` — all before any FS/SQL side effect |
| Soft-deleted skills omitted from list/search by default | `TestListReturnsNonDeletedSkills`, `TestSearchExcludesSoftDeleted` (with `include_deleted` audit read) |
| Soft-deleted names are free (resurrected on overwrite write) | `TestWriteOverwriteTrueReplaces`, `TestListReturnsNonDeletedSkills` (resurrect path via UPDATE `deleted_at = NULL`) |
| `memory_type="skill"` observations untouched | no changes to `memory/` or `tools_memory*.go`; full mcp package green |

## TDD evidence (RED → GREEN)

**RED** (before implementation):
```
# ./internal/mnemonic/skills/
skills_test.go: undefined: Store, New, Skill, ErrNameTaken, Write, List, Search, SearchWith
# ./internal/mnemonic/mcp/
tools_skills_test.go: undefined: writeSkillTool, handleWriteSkill, listSkillsTool,
    handleListSkills, searchSkillsTool, handleSearchSkills
```

**GREEN** (after implementation, `-count=1 -run 'Skill' -v`):
```
ok  .../internal/mnemonic/skills  1.1s  (8 tests)
ok  .../internal/mnemonic/mcp     2.4s  (4 skill MCP tests)
```

Fails-when gate
(`go test ./internal/mnemonic/skills/ ./internal/mnemonic/mcp/ -count=1 -run 'Skill'`)
**PASS**, exit 0.

Full `./internal/mnemonic/...` suite: all packages pass except two
pre-existing failures unrelated to this change (see Concerns).

## 4-pass self-review

1. **Completeness** — every acceptance bullet has a named test; no stubs.
   `Search` is a thin wrapper over `SearchWith` (same pattern as `facts.Search`
   / `facts.SearchWith`).
2. **Re-read** — `skills.go` + `tools_skills.go` alignment clean; stray double
   blank line removed; `gofmt -l` clean on all files touched by this ticket
   (the remaining `gofmt -l` hits in `mcp/` are pre-existing).
3. **Defects** — `validName` rejects NUL/`/`/`\`/`..` and non-identifier runes;
   language lookup lowercases + trims; `Write` validates before opening the tx
   so unknown language / path escape never touch FS or SQL. `WriteFile` 0644
   inside `MkdirAll` 0755 matches the store's file conventions.
4. **Full run** — focused gate re-run GREEN; `go vet` on both packages clean
   except the pre-existing self-assignment at `tools_code_pdg.go:241`.

## Lazy / gate evidence

- No new dependencies (stdlib only; reuses the 011 schema + triggers and the
  `memory.extLanguage` alias set).
- `skills_fts` is an external-content FTS5 table kept by the 011 triggers —
  no manual FTS inserts in Go (DRY with the facts leg).
- `buildSkillFTSQuery` reuses the same quoted-token convention as
  `facts.buildFactFTSQuery`.
- Tool surface grew additively (84→87); no existing tools removed or renamed.

## Files

- `skillgrid-cli/internal/mnemonic/skills/skills.go` (Create — Store: Write/List/Search)
- `skillgrid-cli/internal/mnemonic/skills/skills_test.go` (Create — 8 tests)
- `skillgrid-cli/internal/mnemonic/mcp/tools_skills.go` (Create — 3 new tools)
- `skillgrid-cli/internal/mnemonic/mcp/tools_skills_test.go` (Create — 4 MCP tests)
- `skillgrid-cli/internal/mnemonic/mcp/server.go` (Modify — `registerSkillTools` wiring)
- `skillgrid-cli/internal/mnemonic/mcp/server_test.go` (Modify — count + want slice)
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_affected_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_knowledge_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_governance_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_args_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_retrieval_test.go` (Modify — count)

## Concerns

- **Pre-existing `http/tracker` test failures** (`TestPhase2_CLI_Failure`,
  `TestPhase2_BadOutput`) — gh CLI timeout under parallel load; pass in
  isolation. Unrelated to this change (carried over TICKET-01/02).
- **Pre-existing `wiki/skillgrid_test.go` failures** — from untracked wiki
  files (`internal/mnemonic/wiki/skillgrid*.go`) not introduced by this ticket.
- **Pre-existing `go vet` finding** at `tools_code_pdg.go:241` (self-assignment)
  and pre-existing `gofmt -l` hits in several `mcp/` files — not from this change.
- The "execute" half of the step-03 scenario is TICKET-04 (sandboxed
  `use_skill` executor); this ticket ships the registry leg only, per the
  ticket split.
