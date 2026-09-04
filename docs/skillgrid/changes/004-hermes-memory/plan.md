# Plan: 004-hermes-memory — Hermes Fact Memory & Agent Skills

## Technical Approach
Extend **003** (not redo **Tiered Storage**, core `semantic_search`, or trail CLI) with **Fact Memory** and **Agent Skill** registry beside `mem_*`. Additive SQL `011_*` after **003** `010_*`; scripts under `.skillgrid/files/skills/`. Fact/skill vectors use **sqlite-vec** on an isolated **Seam**; **003** Pure Go document embeddings stay. Slice 1 includes sandboxed `use_skill` and auto-skill on `mnemonic_commit`. Keep **003** L0/L1/L2 labels — no Hermes L0–L3.

## Architecture Decisions

### Decision: Fact Memory as deep Module
**Module / Interface / Seam / Adapter / Depth**: `facts`; Add/Search/Forget/Decay; SQL+FTS+vec **Seam**; SQLite; deep.
**Choice**: Separate `facts`, `facts_fts`, `fact_embeddings`, `forgetting_events`.
**Alternatives considered**: Overlay on `observations`; Hermes L1 labels.
**Rationale**: Locked intent; no `mem_*` churn; keep **Tiered Storage** terms.

### Decision: sqlite-vec only on fact/skill path
**Module / Interface / Seam / Adapter / Depth**: `vec.Index`; Upsert/SearchKNN; driver **Seam**; sqlite-vec (+ fake); deep.
**Choice**: sqlite-vec for fact/skill KNN; leave **003** `path_embeddings` Pure Go.
**Alternatives considered**: 002 observation BLOB cosine; external vector DB.
**Rationale**: Intent; isolates modernc↔extension risk in one **Adapter**.

### Decision: Agent Skills = FS + table
**Module / Interface / Seam / Adapter / Depth**: `skills`; Write/List/Search/Use; FS+SQL **Seam**; disk+DB; deep.
**Choice**: `files/skills/{name}.{ext}` + `skills`/`skills_fts`/`skill_embeddings`/`skill_usage`.
**Alternatives considered**: SQLite BLOBs only; reuse `.agents/skills`.
**Rationale**: Glossary **Agent Skill**; git-friendly; searchable metadata.

### Decision: Constrained subprocess for use_skill
**Module / Interface / Seam / Adapter / Depth**: `skills.Executor`; Run→IO; exec **Seam**; sandbox (+ deny); deep.
**Choice**: Allowlisted runners, timeout, no network, cwd=skill dir; log usage.
**Alternatives considered**: Docker-only; defer execute.
**Rationale**: Slice-1 acceptance; sandbox risk accepted with RED tests.

### Decision: Commit hooks extend 003 compaction
**Module / Interface / Seam / Adapter / Depth**: AfterCommit on `mnemonic_commit`; fact-extract + optional skill-gen; deep.
**Choice**: Post-commit extract facts; optionally `write_skill`; no tier/trail redo.
**Alternatives considered**: Separate commit tool; always auto-skill.
**Rationale**: Extends **003**; optional skill avoids junk.

## Data Flow

```
fact_add ──▶ facts + FTS + sqlite-vec
fact_search / search_skills ──▶ BM25 ± KNN ──▶ Retrieval Trail
write_skill ──▶ files/skills/* + skills + vec
use_skill ──▶ sandbox exec ──▶ skill_usage
mnemonic_commit (003) ──▶ (+004) facts ± auto-skill
CLI memory|skill ──▶ same Modules
```

## Impacted Files Map
| File | Action | Step | Description |
|------|--------|------|-------------|
| `skillgrid-cli/internal/mnemonic/store/migrations/011_facts_skills.sql` | Create | 01 | facts/FTS/forgetting/embeddings/skills; trail cols |
| `skillgrid-cli/internal/mnemonic/vec/vec.go` | Create | 01 | sqlite-vec **Interface**/loader |
| `skillgrid-cli/internal/mnemonic/store/store.go` | Modify | 01 | Hook vec extension on Open if needed |
| `skillgrid-cli/internal/mnemonic/facts/facts.go` | Create | 01 | Fact Memory **Module** |
| `skillgrid-cli/internal/mnemonic/mcp/tools_facts.go` | Create | 02 | fact_* + trail; RegisterFactTools |
| `skillgrid-cli/internal/mnemonic/skills/skills.go` | Create | 03 | Registry write/list/search |
| `skillgrid-cli/internal/mnemonic/mcp/tools_skills_registry.go` | Create | 03 | write/search/list_skill tools |
| `skillgrid-cli/internal/mnemonic/skills/execute.go` | Create | 04 | Sandbox exec + hybrid search |
| `skillgrid-cli/internal/mnemonic/mcp/tools_skills_exec.go` | Create | 04 | use_skill + hybrid mode |
| `skillgrid-cli/internal/mnemonic/mcp/server.go` | Modify | 04 | Wire Register* for fact+skill tools |
| `skillgrid-cli/internal/mnemonic/mcp/tools_compaction.go` | Modify | 05 | Fact extract + auto-skill (**003** file) |
| `skillgrid-cli/cmd/skillgrid/memory.go` | Create | 05 | memory fact\|forget\|decay |
| `skillgrid-cli/cmd/skillgrid/skill.go` | Create | 05 | skill list\|search\|execute |
| `skillgrid-cli/cmd/skillgrid/main.go` | Modify | 05 | Dispatch memory/skill |

## Step WHAT

### Step 01-facts-schema — What it delivers
- As operator: open creates **Fact Memory** tables without rewriting observations.
- Given a DB after **003**, migrate leaves prior rows and **Tiered Storage** intact.
- Edge: re-open idempotent; missing sqlite-vec fails closed on vec ops only.

### Step 02-fact-tools — What it delivers
- As agent: `fact_add`/`fact_search`/`fact_forget`/`fact_decay`; soft-deleted facts out of default search.
- Given search, a **Retrieval Trail** records mode and fact ids.
- Edge: decay lowers importance, logs events; purge only below threshold.

### Step 03-skills-registry — What it delivers
- As agent: write/list/search **Agent Skills** (lexical); FS + SQL metadata.
- Soft-deleted skills omitted from default list/search.
- Edge: overwrite=false rejects name collision.

### Step 04-skill-execute-hybrid — What it delivers
- As agent: sandboxed `use_skill` returns captured IO and logs usage.
- Hybrid BM25 + sqlite-vec cosine for skills (and facts when mode=hybrid).
- Edge: bad language/timeout/path escape → error, no host-wide exec.

### Step 05-commit-hooks-cli — What it delivers
- As agent: `mnemonic_commit` keeps **003** behaviour and adds fact extract ± auto-skill.
- As operator: `skillgrid memory` / `skillgrid skill` (incl. execute) match MCP.
- Edge: skip auto-skill when no reusable pattern; trail CLI unchanged.

## Interfaces / Contracts

```go
type FactStore interface {
  Add(ctx context.Context, in FactAdd) ([]Fact, error)
  Search(ctx context.Context, q FactQuery) (FactHits, trailID string, err error)
  Forget(ctx context.Context, sel FactSelector) (int, error)
  Decay(ctx context.Context, p DecayParams) (DecayResult, error)
}
type SkillRegistry interface {
  Write(ctx context.Context, in SkillWrite) (Skill, error)
  List(ctx context.Context, scope string, limit int) ([]Skill, error)
  Search(ctx context.Context, q SkillQuery) (SkillHits, trailID string, err error)
  Use(ctx context.Context, idOrName string, args json.RawMessage) (SkillRun, error)
}
// Additive MCP fact_* / skill_*; mem_* code_* web_* unchanged
```

## Mnemonic Integration
New tools: `fact_add`, `fact_search`, `fact_forget`, `fact_decay`, `write_skill`, `search_skills`, `list_skills`, `use_skill`. Extend **003** `mnemonic_commit` (facts ± auto-skill) and **Retrieval Trail** with fact/skill ids. No `mem_save`/`code_*` shape change. Topic: `sdd/004-hermes-memory/plan`.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests | Owner step |
|---|---|---|---|---|
| Documentation-like paths | Applicable — executable **Agent Skill** scripts | Allowlisted runners; reject path escape | `../evil.sh` / unknown lang → error, no exec | 04 |
| Git repository selection | N/A: no git cwd authority | — | — | — |
| Commit state | N/A: no git commit automation | — | — | — |
| Push state | N/A: no push | — | — | — |
| PR commands | N/A: no PR CLI | — | — | — |
| **Mnemonic tool surface** | Applicable — new fact_*/skill_* | Additive; `mem_*` unchanged; JSON-only | Tools registered; `mem_save` works; soft-deleted fact absent from search | 02, 03, 04 |
| **Shared-convention drift** | N/A: no `_shared/conventions/*` edits | — | — | — |

Shell/subprocess sandbox owned by step 04 RED (timeout, cwd jail, allowlist).

## Migration / Rollout
Requires **003** `010_*`. Additive `011_facts_skills.sql`. Fail closed if sqlite-vec missing on vec search. Rollback: drop 011 + new tools/CLI.

## Open Questions
- [ ] Tracker: `task-001` (plan ticket; backlog CLI Bun SIGILL — file created via emergency fallback).
- [ ] sqlite-vec on modernc (extension vs CGO) — pick in step 01; **Seam** isolates.
