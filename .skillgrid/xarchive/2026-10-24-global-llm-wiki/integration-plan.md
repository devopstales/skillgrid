# Integration Plan: Global LLM-Wiki into the Skillgrid Workflow

> **STATUS:** `draft` — planning the SDD pipeline path for this change.
> **Change slug:** `2026-10-24-global-llm-wiki`
> **Current phase:** `spec` (briefing.md exists, all 8 open questions resolved)

## Where We Are

The briefing is complete: 27 design decisions, 15 research sources, 2 changes (A + B), all open questions resolved. The spec is `ready-for-tasks`.

## SDD Pipeline Path

```
briefing.md (DONE)
    ↓
[1] writing-blueprints → blueprint.md
    ↓
[2] blueprint-validation (critic subagent + user gate)
    ↓
[3] acceptance-test-authoring → acceptance.feature
    ↓
[4] slicing → tasks.md (dependency graph + execution waves)
    ↓
[5] ticketing → Backlog.md (backlog MCP)
    ↓
[6] execution → subagent-execution (or simple-execution)
    ↓
[7] qa → report.md (QA half)
    ↓
[8] requesting-code-review → review.md
    ↓
[9] receiving-code-review
    ↓
[10] ship → xarchive/
    ↓
[11] reflect → report.md (full) + mem_session_summary
```

## Step-by-Step Plan

### [1] writing-blueprints

**Input:** `briefing.md` (complete)

**Output:** `blueprint.md` in the change dir

**Key decisions to make during blueprinting:**

| Decision | Options | Leaning |
|---|---|---|
| **Change split** | Single change (A+B together) vs two sequential changes (A then B) | **Two changes**: `2026-10-24-global-memory` (A) + `2026-10-25-llm-wiki-pipeline` (B). A is smaller (~1100-1500 lines), unblocks memory immediately, creates `global.sqlite` which B needs. B depends on A. |
| **ADR numbering** | ADR-0049 + ADR-0050 vs single ADR | **Two ADRs**: ADR-0049 (global memory + skill loop), ADR-0050 (wiki pipeline + layout move). Each ADR is scoped to its change. |
| **Migration numbering** | 071 (wiki) + 072 (category) + 073 (skill_usage.outcome) vs combined | **Three migrations**: 072 (facts.category), 073 (skill_usage.outcome + feedback), 074 (wiki_pages + wiki_links + wiki_fts). Number sequentially. |
| **Layout move timing** | During A or during B | **During B** (step B8). The layout move is wiki-specific; A doesn't need it. |
| **Web-UI scope** | Full (4 pages + API + server) vs minimal (search + review only) | **Full**: WikiPage, WikiReviewPage, WikiGraphPage, WikiDashboardPage. The Web-UI is what makes HITL review practical. |

**Blueprint structure (per change):**
- Goal + scope
- Architecture (diagrams from briefing)
- Step table (A1-A8 / B1-B9) with file paths
- Migration plan
- ADR draft (to be written at spec-zone commit)
- Risk register
- Acceptance criteria (to be expanded in [3])

### [2] blueprint-validation

**Who:** Critic subagent + user gate

**Checks:**
- Does the blueprint cover all 27 design decisions?
- Are file paths consistent with the codebase?
- Is the migration numbering correct (071 is `add_knowledge_class`, next free is 072)?
- Does the dependency graph make sense (A before B)?
- Are the ADR references correct (0012, 0048, 0023, 0031, 0045)?
- Is the edit scope clear (which files each ticket touches)?

**Gate:** User approves the blueprint before proceeding.

### [3] acceptance-test-authoring

**Input:** `blueprint.md`

**Output:** `acceptance.feature` (BDD scenarios)

**Scenario gates (per change):**

**Change A (global memory):**
- G1: Global store bootstrap — `__global__` pseudo-project, `global.sqlite` created, `ListProjects` skips it
- G2: Scope routing — `fact_add scope=global` writes to `global.sqlite`, `scope=project` writes to per-project store
- G3: Fused search — `scope=all` RRF-fuses both stores, labels hits with scope + category
- G4: Batch atomicity — `operations` array, limit checked on final result only, partial failure rolls back
- G5: Char limit — overflow returns error with `current_entries` + `usage`, agent can batch free+add
- G6: Prefetch injection — frozen block in system prompt, `§` separators, usage headers, two targets (memory/user)
- G7: Write approval — passive writes staged, pending list, approve/reject
- G8: Skill loop — `skill_usage.outcome` tracked, `skill inspect` detects patterns, `skill amend` proposes patch, `skill evaluate` rolls back on regression
- G9: CLI — `--scope global --target memory|user`, `pending/approve/reject`

**Change B (wiki pipeline):**
- G1: Vault init — `knowledge/` created at global + project, `.obsidian/` initialized, `raw/` + `proposed/` + `comparisons/` zones exist
- G2: FTS index — `wiki_pages`/`wiki_links`/`wiki_fts` tables, `wiki index` rebuilds from markdown, `wiki search` returns page paths
- G3: Ingest — new file in `raw/` → credential redaction → LLM extraction → confidence score → `proposed/` (or auto-approve if score ≥ threshold + routine)
- G4: Contradiction — conflicting content → `comparisons/` page + `contested: true` on affected pages
- G5: Two-phase review — approve records decision (no vault change), apply moves to gold + reindex + commit
- G6: Lint — frontmatter valid, links unbroken, orphans reported, index complete; lint must not regress (DGM gate)
- G7: Query — `wiki_query` FTS search, top-k pages with path + title + snippet + scope
- G8: Distill — explicit `fact_add scope=global` + passive session-end sweep
- G9: Migration — existing bundle content moved to `knowledge/`, all path references updated, `git mv` preserves history
- G10: Web-UI — 4 pages render, search works, review approve/reject works, graph renders, dashboard shows counts

### [4] slicing

**Input:** `blueprint.md` + `acceptance.feature`

**Output:** `tasks.md` with dependency graph + execution waves

**Change A tasks (est. 8-10 tickets):**

| Ticket | Task | Wave | Depends on |
|---|---|---|---|
| A-T01 | Migration 072: `facts.category` column | W1 | — |
| A-T02 | `Service.OpenGlobal()` + `__global__` bootstrap | W1 | — |
| A-T03 | Scope + category params on `fact_*` tools | W2 | A-T02 |
| A-T04 | Fused search (RRF, scope+category labels) | W2 | A-T03 |
| A-T05 | Batch operations + char limit + dedup | W2 | A-T03 |
| A-T06 | Prefetch injection (plugin hook + MCP resource) | W3 | A-T05 |
| A-T07 | Background distill (extends 013) + write approval | W3 | A-T05 |
| A-T08 | Migration 073: `skill_usage.outcome` + `feedback` | W1 | — |
| A-T09 | Skill loop: inspect + amend + evaluate | W3 | A-T08 |
| A-T10 | CLI: `--scope global --target`, `pending/approve/reject` | W4 | A-T03, A-T06, A-T07 |
| A-T11 | ADR-0049 + tests | W4 | A-T01..A-T10 |

**Change B tasks (est. 12-15 tickets):**

| Ticket | Task | Wave | Depends on |
|---|---|---|---|
| B-T01 | Migration 074: `wiki_pages` + `wiki_links` + `wiki_fts` | W1 | A complete |
| B-T02 | Vault init: `knowledge/` at global + project, `.obsidian/` | W1 | — |
| B-T03 | FTS index builder: walk vault, parse frontmatter + links | W2 | B-T01, B-T02 |
| B-T04 | CLI: `wiki index` + `wiki search` | W2 | B-T03 |
| B-T05 | Credential redaction (deterministic regex) | W2 | — |
| B-T06 | Ingest pipeline: pre-flight → redact → LLM → validate → route | W3 | B-T03, B-T05 |
| B-T07 | Contradiction detection + `comparisons/` + `contested: true` | W3 | B-T06 |
| B-T08 | HITL review: two-phase (approve → apply) + CLI + MCP tools | W4 | B-T06 |
| B-T09 | `wiki lint` (extends `okf-lint.mjs`) + DGM gate | W4 | B-T03 |
| B-T10 | `wiki_query` MCP tool (FTS, scope, RRF) | W4 | B-T03 |
| B-T11 | Distill: explicit + passive (extends 013 to global) | W4 | A-T07 |
| B-T12 | `web_cache` → `raw/` bridge (`--from-cache`) | W4 | B-T06 |
| B-T13 | Migration: bundle root → `knowledge/` + path reference sweep | W5 | B-T02 |
| B-T14 | Web-UI: WikiPage + WikiReviewPage + WikiGraphPage + WikiDashboardPage | W5 | B-T03, B-T08 |
| B-T15 | Web-UI: REST API + Go server endpoints | W5 | B-T03 |
| B-T16 | Mnemonic skill update (Step 3.6, Tools, Subcommands) | W5 | B-T04, B-T08, B-T10 |
| B-T17 | ADR-0050 + tests | W5 | B-T01..B-T16 |

**Execution waves:**
- **W1:** Foundations (migrations, bootstrap) — 3-4 parallel tickets
- **W2:** Core pipeline (routing, search, indexing) — 3-4 parallel tickets
- **W3:** Intelligence (ingest, distill, skill loop) — 3-4 parallel tickets
- **W4:** Trust + query (HITL, lint, query, distill) — 4-5 parallel tickets
- **W5:** Surface (migration, Web-UI, skill docs, ADR) — 4-5 parallel tickets

### [5] ticketing

**Tool:** `backlog` MCP (Backlog.md)

**Format:** One Backlog task per ticket. Title: `[A|B]-T0N: <short description>`. Body: task brief + file paths + acceptance criteria reference.

### [6] execution

**Tool:** `subagent-execution` (preferred for parallel waves) or `simple-execution` (serial)

**Per-ticket flow:**
1. Implementer subagent: TDD (RED → GREEN → REFACTOR)
2. Work-unit commit (conventional commit, `[skillgrid-context]` block)
3. Reviewer subagent: spec compliance + code quality
4. Update `tasks.md` checkbox + Backlog task status

**Threat matrix:** Each ticket's tests include threat-matrix RED cases (isolation, overflow, injection shape, etc.) per the existing skillgrid convention.

### [7] qa

**Tool:** `qa` skill

**Checks:**
- `go build ./...` (exit 0)
- `go test ./...` (all pass)
- `gofmt -w ./...` (clean)
- `trivy fs . --scanners vuln` (advisory only)
- `okf-lint.mjs` (advisory — wiki lint for Change B)
- Verification-gap audit (every acceptance gate has a test)
- TDD evidence audit (RED → GREEN → REFACTOR in commit history)

**Output:** `report.md` (QA half: test plan + gate verdict + evidence)

### [8] requesting-code-review

**Tool:** `requesting-code-review` skill (or `parallel-code-review` for large diffs)

**Reviewers:**
- Spec compliance: does the code match the blueprint?
- Code quality: naming, error handling, test coverage
- Security: credential redaction patterns, scope isolation, write approval gate

**Output:** `review.md` (findings + dispositions)

### [9] receiving-code-review

**Tool:** `receiving-code-review` skill

**Flow:** For each finding: verify (receiving-code-review protocol), fix or park, update `review.md` disposition.

### [10] ship

**Tool:** `ship` skill

**Actions:**
- Merge to base branch
- Update `state.yaml` (remove change entry)
- Move change dir to `xarchive/`
- Update Backlog (close tasks)
- `git mv` preserves history

### [11] reflect

**Tool:** `reflect` skill

**Actions:**
- Write `report.md` (full: QA half + review half + lessons)
- `mem_session_summary` (persist session learnings)
- `mem_session_end`
- Environment retro (sweep for tool/config improvements)
- Lift durable findings to `knowledge/reference/` + `log.md`

## Pre-conditions Before [1]

| Pre-condition | Status | Action |
|---|---|---|
| Briefing complete | ✅ | — |
| All open questions resolved | ✅ | — |
| ADR-0048 exists (layout to be amended) | ✅ | — |
| Migration 071 is `add_knowledge_class` (next free: 072) | ✅ | Verified |
| `skills`/`skill_usage` tables exist (migration 011) | ✅ | Verified |
| 013 distill hook exists | ✅ | Verified |
| ADR-0023 (LLM client) exists | ✅ | Verified |
| ADR-0045 (OpenCode V2 plugins) exists | ✅ | Verified |
| `okf-lint.mjs` exists | ✅ | Verified (60 pre-existing findings) |
| Web-UI structure (`skillgrid-ui/src/features/`) | ✅ | Verified |

## Sequencing Decision: Two Changes

**Change A: `2026-10-24-global-memory`** (ADR-0049)
- ~1100-1500 lines
- 11 tickets, 4 waves
- Unblocks: Hermes-style memory, skill improvement loop
- Creates: `global.sqlite`, `facts.category`, `skill_usage.outcome`

**Change B: `2026-10-25-llm-wiki-pipeline`** (ADR-0050)
- ~2200-3200 lines
- 17 tickets, 5 waves
- Unblocks: LLM-wiki pipeline, HITL review, Web-UI, layout move
- Depends on: Change A (uses `global.sqlite`)

**Why two changes:**
1. A is smaller and self-contained — it can ship independently
2. B depends on A (needs `global.sqlite` for the global wiki index)
3. The layout move (B8) is wiki-specific — it doesn't affect A
4. Two ADRs are cleaner than one mega-ADR
5. If A ships and B gets delayed, the user still has global memory + skill loop

## state.yaml Updates

**Before [1]:**
```yaml
changes:
  2026-10-24-global-llm-wiki:
    phase: spec
    status: in-progress
```

**After [1] (blueprint written):**
```yaml
changes:
  2026-10-24-global-llm-wiki:
    phase: blueprint
    status: in-progress
```

**After split into two changes:**
```yaml
changes:
  2026-10-24-global-memory:
    phase: blueprint
    status: in-progress
  2026-10-25-llm-wiki-pipeline:
    phase: spec
    status: queued
    blocked(spec): depends on 2026-10-24-global-memory
```

**After A ships:**
```yaml
changes:
  2026-10-25-llm-wiki-pipeline:
    phase: blueprint
    status: in-progress
# 2026-10-24-global-memory removed (shipped)
```
