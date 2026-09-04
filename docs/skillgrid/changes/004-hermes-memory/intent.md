# Intent: 004-hermes-memory — Hermes Fact Memory & Agent Skills

## Business Problem
Agents lack importance-ranked **Fact Memory** with forgetting, and a path to write/find/execute **Agent Skills**. Extends **003**; not a **Tiered Storage** redo.

## Target Users & Situations
- **Agent** — recall facts; write/search/execute **Agent Skills**.
- **Operator** — CLI inspect/forget/decay/execute; debug **Retrieval Trail**s.
- Urgency: High after **003**; builds on **002**/**003**.

## Business Rules
- **Extends** **003** — no re-delivery of tiered FS, core `semantic_search`, core `mnemonic_commit`, or trail CLI.
- **Fact Memory** is a **new store** beside `mem_*` observations (not an overlay).
- Additive SQL + FS; Go MCP + SQLite + FS; no external vector DB.
- Fact/skill vectors require **sqlite-vec** (beyond **003** Pure Go here; **003** document/semantic path may stay).
- Soft-delete facts; decay then purge below threshold.
- **Agent Skills** = FS + `skills` table — ≠ `.agents/skills` packs.
- Slice 1: sandboxed `use_skill` + auto-skill on `mnemonic_commit` (sandbox risk accepted).
- Keep **003** L0/L1/L2 **Tiered Storage** terms; no Hermes L0–L3 labels.

## Success Criteria (UAT-level)
- [ ] Add/search/forget facts; soft-deleted out of default search.
- [ ] Decay lowers importance and logs events; CLI can trigger.
- [ ] Write/list/search **Agent Skills**; lexical and hybrid modes.
- [ ] Sandboxed `use_skill` returns captured output.
- [ ] `mnemonic_commit` extracts facts and may auto-generate an **Agent Skill**.
- [ ] Fact/skill search extends **003** **Retrieval Trail**s.
- [ ] `go test ./...` passes for touched packages.

## Scope

### In Scope
- **Fact Memory** + MCP fact_add|search|forget|decay (FTS5, sqlite-vec, forgetting).
- **Agent Skill** registry + write/search/list/`use_skill`; hybrid BM25+cosine (sqlite-vec).
- Fact extraction + auto-skill on `mnemonic_commit`; CLI memory/skill (incl. execute).

### Out of Scope
- Reimplementing **003** **Tiered Storage**, `migrate --tier`, `semantic_search` core, trail CLI.
- OpenCode plugin; cloud sync; external vector DB; code-index/web-cache rewrite; **002** identity.
- Replacing **003** Pure Go document/semantic embedding path.

## Step Blueprint
- `01-facts-schema`: Facts + FTS + forgetting_events + sqlite-vec embeddings.
- `02-fact-tools`: MCP fact_* + trail logging.
- `03-skills-registry`: Skills FS + table + FTS; write/search/list.
- `04-skill-execute-hybrid`: Sandboxed `use_skill` + hybrid BM25+cosine (sqlite-vec).
- `05-commit-hooks-cli`: Commit fact extraction + auto-skill; memory/skill CLI.

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `…/mnemonic/store/migrations/` | New | facts, skills, forgetting, sqlite-vec |
| `…/mnemonic/` | New/Modified | fact/skill modules; sqlite-vec |
| `…/mnemonic/mcp/` | Modified | fact_* / skill_* incl. `use_skill` |
| `skillgrid-cli/` CLI | Modified | memory / skill (incl. execute) |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Duplicate **003** work | High | Out-of-scope; depend on **003** |
| Skill execute sandbox | High | Accepted; constrain runtime, log usage |
| sqlite-vec vs **003** Pure Go | Med | Isolate to fact/skill path |
| Term clash with `.agents/skills` | Med | Glossary: **Agent Skill** |

## Rollback Plan
Drop new migrations, tools, and CLI; leave **003** intact.

## Dependencies
- **003** required; **002** preferred; sqlite-vec for facts/skills.
- Source: `docs/plan/05-hermes-memory.md`. Approved 2026-09-04.
