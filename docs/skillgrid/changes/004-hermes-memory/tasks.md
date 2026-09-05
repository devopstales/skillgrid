# Tasks: 004-hermes-memory

> **STATUS:** `in-progress` (2026-09-04) — 0/5 steps PASS
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-driven-development (or simple-execution) to implement step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Agents get importance-ranked Fact Memory with forgetting and a path to write, find, and sandboxed-execute Agent Skills beside `mem_*` observations.

**Architecture:** Extend **003** with deep `facts` / `skills` Modules, additive `011_*` SQL, sqlite-vec on an isolated Seam, sandboxed `use_skill`, and optional auto-skill on `mnemonic_commit`. See `change.md` decisions.

**Tech Stack:** Go (`skillgrid-cli`), SQLite + sqlite-vec (fact/skill KNN), MCP, FS under `.skillgrid/files/skills/`

**Spec:** `docs/skillgrid/changes/004-hermes-memory/change.md`

**Acceptance:** `docs/skillgrid/changes/004-hermes-memory/acceptance.feature` (`@step-NN`)

---

## Goal

Agents and operators can add/search/forget/decay ranked facts and write/list/search/execute Agent Skills with hybrid retrieval and Retrieval Trails, without re-delivering Change **003**.

## Out of scope / Non-Goals

- Reimplementing **003** Tiered Storage, `migrate --tier`, core `semantic_search`, or trail CLI
- OpenCode plugin; cloud sync; external vector DB
- Rewriting code-index or web-cache; Changing **002** identity
- Replacing **003** Pure Go document/semantic embedding path
- Hermes L0–L3 labels; confusing Agent Skills with `.agents/skills` packs

## Definition of Done

Change is done only when **all** of the following are true:

- [ ] Every success criterion / DoD checkbox in `change.md` is met
- [ ] Every `@step-NN` Feature in `acceptance.feature` has passing scenarios
- [ ] Every step below has Verdict `PASS` or `PASS WITH WARNINGS`
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] No **Global Constraint** violated
- [ ] Rollback path in `change.md` is still valid (or N/A documented)
- [ ] `## State` status is `done` (set at archive gate)

## Global Constraints

Copy verbatim from `change.md` (Error handling + Non-Goals + stack rules). Every step inherits these — do not restate per step.

- Extends **003** only — no re-delivery of Tiered Storage, core `semantic_search`, or trail CLI
- Fact Memory is a new store beside `mem_*` — not an overlay on observations
- Additive SQL `011_*` after **003** `010_*`; no destructive schema rewrite
- Fact/skill vectors use sqlite-vec only; leave **003** Pure Go `path_embeddings` untouched
- Missing sqlite-vec → fail closed on vec ops only; non-vec open/FTS search still works
- Soft-deleted facts/skills never appear in default list/search
- Decay lowers importance and logs events; purge only below threshold
- Agent Skills = `.skillgrid/files/skills/` + SQL — ≠ `.agents/skills` packs
- New MCP tools are additive JSON-only; `mem_*` / `code_*` / `web_*` shapes unchanged
- `use_skill`: allowlisted runners, timeout, no network, cwd=skill dir; reject path escape
- Keep **003** L0/L1/L2 labels — no Hermes L0–L3
- No OpenCode plugin, cloud sync, or external vector DB in this change
- Base path / module root: `skillgrid-cli/`

---

## State

```yaml
phase: spec          # spec | apply | verify | archive
current_step: 01-facts-schema
status: in_progress  # in_progress | blocked | done
updated: 2026-09-04T20:05:00Z
```

## Step map

| NN | Step | Tag | Blocked by | Acceptance |
|----|------|-----|------------|------------|
| 01 | `facts-schema` | `@step-01` | — | Feature tagged `@step-01` |
| 02 | `fact-tools` | `@step-02` | 01 | Feature tagged `@step-02` |
| 03 | `skills-registry` | `@step-03` | 01 | Feature tagged `@step-03` |
| 04 | `skill-execute-hybrid` | `@step-04` | 02, 03 | Feature tagged `@step-04` |
| 05 | `commit-hooks-cli` | `@step-05` | 02, 03, 04 | Feature tagged `@step-05` |

## Review workload (change-level)

| Field | Value |
|-------|-------|
| Estimated changed lines (change) | ~1600–2200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Delivery strategy | ask-on-risk |

Decision needed before apply: Yes · Chain strategy: pending

---

## 01-facts-schema

### Goal

Facts + FTS + forgetting_events + sqlite-vec embeddings — store open creates Fact Memory without rewriting observations.

### Out of scope / Non-Goals

- MCP fact_* / skill_* tools (steps 02–04)
- Sandboxed execute, hybrid mode, commit hooks, CLI (steps 04–05)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/011_facts_skills.sql`
- Create: `skillgrid-cli/internal/mnemonic/vec/vec.go`
- Create: `skillgrid-cli/internal/mnemonic/facts/facts.go`
- Modify: `skillgrid-cli/internal/mnemonic/store/store.go`
- Test: `skillgrid-cli/internal/mnemonic/facts/` · `skillgrid-cli/internal/mnemonic/vec/` · `skillgrid-cli/internal/mnemonic/store/`

**Interfaces:**
- Consumes: none (requires **003** `010_*` already applied)
- Produces: `FactStore` (Add/Search/Forget/Decay); `vec.Index` (Upsert/SearchKNN); migration `011_*` tables including skills schema stubs for later steps

### Tasks

- [ ] 01.1 `[RED]` Store open creates Fact Memory tables without rewriting observations / Tiered Storage intact — Scenario: `Store open creates Fact Memory tables`
  - [ ] 01.1.a Write failing test
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/facts/ -count=1 -run 'FactMemory|Migration011'` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation — `011_facts_skills.sql` + hook migrate on Open
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/facts/ -count=1 -run 'FactMemory|Migration011'` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(mnemonic): add Fact Memory migration 011`
- [ ] 01.2 `[RED]` Missing sqlite-vec fails closed on vec ops only — Scenario: `Missing vector extension fails closed on vector ops`
  - [ ] 01.2.a Write failing test (fake adapter / missing extension)
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/vec/ -count=1 -run 'FailClosed|MissingExt'` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation — `vec.Index` loader + fake; store Open hooks when available
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/vec/ -count=1 -run 'FailClosed|MissingExt'` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(mnemonic): isolate sqlite-vec Seam for fact/skill path`
- [ ] 01.3 `[AFK]` Create `facts` Module implementing Add/Search/Forget/Decay over SQL+FTS+vec Seam — `Run: go test ./skillgrid-cli/internal/mnemonic/facts/ -count=1` — Expected: PASS
- [ ] 01.4 `[AFK]` Re-open idempotent — Scenario: `Re-open is idempotent` — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -count=1 -run 'Idempotent|ReOpen'` — Expected: PASS
- [ ] 01.5 `[AFK]` Leave **003** Pure Go `path_embeddings` path untouched — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/facts/ ./skillgrid-cli/internal/mnemonic/vec/ ./skillgrid-cli/internal/mnemonic/store/ -count=1` | PASS | | |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios for `@step-01` | PASS | | |
| Runtime harness | store Open after **003** migrate | PASS | | |
| Rollback boundary | drop `011_*` / disable vec hook — **003** intact | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): Fact Memory schema and sqlite-vec Seam`

---

## 02-fact-tools

### Goal

MCP `fact_*` tools + Retrieval Trail logging; soft-deleted facts out of default search; `mem_*` unchanged.

### Out of scope / Non-Goals

- Agent Skill registry / execute (steps 03–04)
- Commit hooks and CLI (step 05)
- Hybrid search mode (step 04)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-facts-schema

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_facts.go`
- Test: `skillgrid-cli/internal/mnemonic/mcp/tools_facts_test.go` (or `server_test.go`)

**Interfaces:**
- Consumes: `FactStore` from 01
- Produces: `RegisterFactTools`; MCP `fact_add` / `fact_search` / `fact_forget` / `fact_decay` + trail ids

### Tasks

- [ ] 02.1 `[RED]` Mnemonic tool surface — fact_* registered **and** `mem_save` still dispatches — Scenario: `Fact tools add search and record a Retrieval Trail`
  - [ ] 02.1.a Write failing test in `server_test.go` or `tools_facts_test.go`
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'FactTools|RegisterFact'` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — `tools_facts.go` + `RegisterFactTools` (JSON-only; additive)
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'FactTools|RegisterFact'` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): register MCP fact_* tools`
- [ ] 02.2 `[RED]` Soft-deleted fact absent from default search; `mem_*` shape unchanged — Scenario: `Soft-deleted fact absent from default search`
  - [ ] 02.2.a Write failing test (forget then search)
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'SoftDelete|ForgetSearch'` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation — soft-delete filter + trail (mode + fact ids)
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'SoftDelete|ForgetSearch'` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): soft-delete facts out of default search`
- [ ] 02.3 `[AFK]` Decay lowers importance and logs `forgetting_events`; purge only below threshold — Scenario: `Decay lowers importance and logs events` — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/facts/ -count=1 -run 'Decay'` — Expected: PASS
- [ ] 02.4 `[AFK]` `fact_search` writes Retrieval Trail recording mode and fact ids — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'Trail|FactSearch'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'Fact'` | PASS | | |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios for `@step-02` | PASS | | |
| Runtime harness | MCP fact_add → fact_search trail | PASS | | |
| Rollback boundary | unregister fact tools — `mem_*` intact | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): MCP fact_* tools and Retrieval Trails`

---

## 03-skills-registry

### Goal

Skills FS + table + FTS; write/list/search (lexical); soft-delete omits from default list/search.

### Out of scope / Non-Goals

- `use_skill` and hybrid ranking (step 04)
- Commit hooks and CLI (step 05)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-facts-schema

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/skills/skills.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_skills_registry.go`
- Test: `skillgrid-cli/internal/mnemonic/skills/` · `skillgrid-cli/internal/mnemonic/mcp/tools_skills_registry_test.go`

**Interfaces:**
- Consumes: `011_*` skills tables from 01
- Produces: `SkillRegistry` Write/List/Search (lexical); MCP write/search/list tools (no `use_skill` yet)

### Tasks

- [ ] 03.1 `[RED]` Mnemonic tool surface — write/search/list_skill registered **and** `mem_save` still works — Scenario: `Write list and lexical search Agent Skills`
  - [ ] 03.1.a Write failing test
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'SkillRegistry|WriteSkill'` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — `skills.go` + `tools_skills_registry.go` (JSON-only; additive)
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'SkillRegistry|WriteSkill'` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): Agent Skill registry MCP tools`
- [ ] 03.2 `[AFK]` Soft-deleted skills omitted from default list/search — Scenario: `Soft-deleted skills omitted from default list and search` — `Run: go test ./skillgrid-cli/internal/mnemonic/skills/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'SoftDelete|Skill'` — Expected: PASS
- [ ] 03.3 `[AFK]` `overwrite=false` rejects name collision — Scenario: `Overwrite false rejects name collision` — `Run: go test ./skillgrid-cli/internal/mnemonic/skills/ -count=1 -run 'Overwrite|Collision'` — Expected: PASS
- [ ] 03.4 `[AFK]` FS under `.skillgrid/files/skills/{name}.{ext}` + SQL metadata for lexical search — `Run: go test ./skillgrid-cli/internal/mnemonic/skills/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/skills/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'Skill'` | PASS | | |
| Acceptance `@step-03` / `@p0` | BDD / mapped unit scenarios for `@step-03` | PASS | | |
| Runtime harness | write → list → lexical search | PASS | | |
| Rollback boundary | remove skill tools — `mem_*` intact | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): Agent Skill registry write list search`

---

## 04-skill-execute-hybrid

### Goal

Sandboxed `use_skill` + hybrid BM25+sqlite-vec cosine; wire all fact/skill registrars without dropping `mem_*`/`code_*`/`web_*`.

### Out of scope / Non-Goals

- Commit fact extract / auto-skill (step 05)
- Operator CLI (step 05)
- Docker-only sandbox

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-04` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-fact-tools, 03-skills-registry

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/skills/execute.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_skills_exec.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go`
- Test: `skillgrid-cli/internal/mnemonic/skills/` · `skillgrid-cli/internal/mnemonic/mcp/`

**Interfaces:**
- Consumes: `SkillRegistry` from 03; `FactStore` / vec from 01–02
- Produces: `skills.Executor` Use; MCP `use_skill`; hybrid search mode; wired `Register*`

### Tasks

- [ ] 04.1 `[RED]` Documentation-like paths / shell sandbox — `../evil.sh` or unknown language → error, no exec; timeout/cwd-jail/allowlist — Scenario: `Path escape or unknown language rejects without exec`
  - [ ] 04.1.a Write failing adversarial tests
  - [ ] 04.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/skills/ -count=1 -run 'Sandbox|PathEscape|Allowlist'` — Expected: FAIL
  - [ ] 04.1.c Minimal implementation — `execute.go` (allowlist, timeout, no network, cwd=skill dir; log `skill_usage`)
  - [ ] 04.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/skills/ -count=1 -run 'Sandbox|PathEscape|Allowlist'` — Expected: PASS
  - [ ] 04.1.e Commit — `feat(mnemonic): sandboxed Agent Skill executor`
- [ ] 04.2 `[RED]` Mnemonic tool surface — `use_skill` (+ hybrid) registered; `mem_save` still works; soft-deleted facts absent from hybrid fact search — Scenario: `Sandboxed use_skill returns captured IO and logs usage`
  - [ ] 04.2.a Write failing test
  - [ ] 04.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'UseSkill|Hybrid'` — Expected: FAIL
  - [ ] 04.2.c Minimal implementation — `tools_skills_exec.go` + wire `server.go` Register* without dropping mem/code/web
  - [ ] 04.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'UseSkill|Hybrid'` — Expected: PASS
  - [ ] 04.2.e Commit — `feat(mnemonic): use_skill and hybrid search MCP`
- [ ] 04.3 `[AFK]` Hybrid BM25 + sqlite-vec cosine for skills (and facts when mode=hybrid) — Scenario: `Hybrid search ranks skills and facts` — `Run: go test ./skillgrid-cli/internal/mnemonic/skills/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'Hybrid'` — Expected: PASS
- [ ] 04.4 `[AFK]` Sandboxed `use_skill` returns captured IO and logs usage — `Run: go test ./skillgrid-cli/internal/mnemonic/skills/ -count=1 -run 'Use|Usage'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/skills/ ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'UseSkill|Sandbox|Hybrid'` | PASS | | |
| Acceptance `@step-04` / `@p0` | BDD / mapped unit scenarios for `@step-04` | PASS | | |
| Runtime harness | allowlisted skill run + hybrid search | PASS | | |
| Rollback boundary | disable use_skill registrar — host exec impossible | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): sandboxed use_skill and hybrid retrieval`

---

## 05-commit-hooks-cli

### Goal

Commit fact extraction ± optional auto-skill; `skillgrid memory` / `skillgrid skill` CLI parity with MCP; trail CLI unchanged.

### Out of scope / Non-Goals

- Redo Tiered Storage migrate or trail CLI
- Always-on auto-skill (must skip when no reusable pattern)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-05` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-fact-tools, 03-skills-registry, 04-skill-execute-hybrid

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_compaction.go`
- Create: `skillgrid-cli/cmd/skillgrid/memory.go`
- Create: `skillgrid-cli/cmd/skillgrid/skill.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go`
- Test: `skillgrid-cli/internal/mnemonic/mcp/` · `skillgrid-cli/cmd/skillgrid/`

**Interfaces:**
- Consumes: FactStore, SkillRegistry, Executor from 01–04; existing **003** `mnemonic_commit`
- Produces: AfterCommit fact extract ± optional `write_skill`; CLI `memory` / `skill` subcommands

### Tasks

- [ ] 05.1 `[RED]` `mnemonic_commit` preserves **003** behaviour and adds fact extract ± auto-skill — Scenario: `Commit extracts facts and optional auto-skill`
  - [ ] 05.1.a Write failing test on compaction hook
  - [ ] 05.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'CommitFact|AutoSkill|Compaction'` — Expected: FAIL
  - [ ] 05.1.c Minimal implementation — extend `tools_compaction.go` (no tier/trail CLI redo)
  - [ ] 05.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'CommitFact|AutoSkill|Compaction'` — Expected: PASS
  - [ ] 05.1.e Commit — `feat(mnemonic): commit fact extract and optional auto-skill`
- [ ] 05.2 `[AFK]` Skip auto-skill when no reusable pattern; trail CLI unchanged — Scenario: `Skip auto-skill when no reusable pattern` — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ -count=1 -run 'SkipAutoSkill|NoPattern'` — Expected: PASS
- [ ] 05.3 `[AFK]` Create `memory.go` — `skillgrid memory` fact|forget|decay parity with MCP — Scenario: `CLI memory and skill match MCP or fail cleanly` — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -count=1 -run 'Memory'` — Expected: PASS
- [ ] 05.4 `[AFK]` Create `skill.go` + register in `main.go` — `skillgrid skill` list|search|execute parity — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -count=1 -run 'Skill'` — Expected: PASS
- [ ] 05.5 `[AFK]` Invalid CLI actions fail without corrupting Fact Memory — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -count=1 -run 'Memory|Skill'` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/cmd/skillgrid/ -count=1 -run 'Commit|Memory|Skill'` | PASS | | |
| Acceptance `@step-05` / `@p0` | BDD / mapped unit scenarios for `@step-05` | PASS | | |
| Runtime harness | mnemonic_commit + CLI memory/skill | PASS | | |
| Rollback boundary | revert compaction hook + CLI — **003** intact | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): commit hooks and memory/skill CLI`

---

## Archive gate checklist

- [ ] Change-level **Definition of Done** fully checked
- [ ] No unchecked `- [ ]` under any `### Tasks`
- [ ] Every step Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated
- [ ] `## State` status is `done` and phase is `archive` (set by verify/archive)
- [ ] STATUS banner updated to `complete`
