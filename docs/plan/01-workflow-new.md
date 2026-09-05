# skillgrid SDD Plan

Spec Driven Development Workflow using `Markdown` structure as spec source and selectable (GitHub, GitLab, Jira, Backlog.md) ticketing system. `Backlog.md` is the default tracker. It uses `Mnemonic` for long-term memory and code indexing.

## skillgrid SDD Workflow (v4)

```
onboard → propose → spec → apply ⇄ verify → archive
```

Optional before locking `change.md` (idea → …):

```
[explore / research.md] → [design-spike] → propose
```

**Entry:** `use-skillgrid` (orchestrator) — detects, routes, resumes; does not reimplement stages.

Idea scope: **feature | bug | refactor | greenfield app** — same pipeline; size only changes how deep research/spike go.

**Verify ↔ apply loop:** agent evidence + human QA plan + code review; human findings become new tasks and return to apply. No seventh top-level stage.

---

## Mapping: 7-phase AI-dev model → Skillgrid (included)

| Phase | Skillgrid home |
|---|---|
| Idea | `use-skillgrid` → propose (any change size) |
| Research | optional `sdd-explore` → `research.md` (change-scoped; rots) |
| Prototype | optional `design-spike` **before** `change.md` approved; marked path listed in change |
| PRD / end state | `sdd-propose` + `questioning` (grill after concrete research/spike when used) |
| Kanban + blockers | `sdd-spec` → `tasks.md` **with blocking edges** + acceptance |
| Execute | `sdd-apply` (sequential default; parallel only unblocked) |
| QA (+ human) | `sdd-verify` (agent proof + `qa-plan` + review) → findings → apply |

---

## Skill taxonomy

Two kinds of skills. Do not duplicate a general capability as a stage-prefixed skill.

| Kind | Rule | Naming |
|---|---|---|
| **Workflow** | Owns a pipeline stage (or is the entry orchestrator). Writes stage artifacts under `docs/skillgrid/changes/` (or onboard skeleton). Invoked by `use-skillgrid` or by the previous stage. | `use-skillgrid`, `sdd-<stage>` |
| **General** | Reusable across stages and outside SDD. Stages **invoke** them; they do not own `## State.phase`. | no `sdd-` prefix (existing Skillgrid names preferred) |

---

## Unifications (decisions)

| Was planned as stage skill | Decision | Becomes |
|---|---|---|
| `sdd-interview` | **Unify** — questioning is cross-cutting (init facts, propose intent, revise after user gate) | general `questioning` (already exists) |
| `sdd-spike` | **Unify** with existing prototype skill | general `design-spike` (Skillgrid name for mattpocock `prototype` / gsd `gsd-spike`) |
| `sdd-architecture` | **Unify** — deep-module vocabulary; decisions stay in `change.md` unless promoted to ADR | general `codebase-design` |
| `sdd-tdd` | **Unify** | general `tdd` |
| `sdd-debug` | **Unify** | general `debugging` |
| `sdd-worktree` | **Unify** | general `isolated-workspace` |
| `sdd-subagents` | **Unify** | general `subagent-execution` + `dispatching-parallel-agents` |
| `sdd-commits` | **Unify** | general (keep/port `work-unit-commits`; do not `sdd-` it) |
| `sdd-review` | **Unify** | general `requesting-code-review` + `review-reception` |
| `sdd-judgment` | **Unify** | general `judgment-day` |
| `sdd-trace` | **Fold into** `sdd-verify` (trace matrix is verify evidence, not a separate stage) | workflow `sdd-verify` step |
| `sdd-finish-branch` | **Unify** | general `finishing-a-development-branch` |
| `sdd-ship` | **Fold into** `sdd-archive` optional path (or invoke finish-branch + tracker PR helpers) | workflow `sdd-archive` option |
| `sdd-retro` | **Fold into** `sdd-archive` + general `mnemonic-memory` | archive step + memory (not `handoff`) |
| `sdd-tasks` | **Absorb into** `sdd-spec` (v3 already: punch-lists + Gherkin in one stage) | workflow `sdd-spec` |
| `sdd-explore` | **Keep as workflow helper** under propose (writes `research.md` for a change) — not a top-level v4 stage | workflow, invoked by `sdd-propose` / `use-skillgrid` |
| `sdd-agent-context`, `sdd-constraints`, `sdd-domain`, `sdd-map-codebase` | **Onboard helpers**, not separate pipeline stages — callable later for refresh | workflow helpers under `sdd-onboard` / `sdd-init` |

---

## Workflow skills (pipeline)

Invoked in order by `use-skillgrid` (or resumed from `tasks.md` `## State.phase`).

### `use-skillgrid` — orchestrator (planned)

* **Does:** Detect initialized vs not; classify request (change / Q&A / spike) — change includes feature, bug, refactor, greenfield; announce route; if taste/UI/unknown API shape → route explore and/or `design-spike` **before** propose locks `change.md`; enforce user gate after spec; resume from `## State`; never freestyle a parallel process.
* **Does not:** Write `change.md` / `tasks.md` / code itself; reimplement stage checklists.
* **Calls general:** `questioning`, `design-spike`, `mnemonic-memory` / `investigate` (Q&A path).
* **Origins:** existing `use-skillgrid` + Superpowers `using-superpowers`; gsd-core `gsd-next` / `gsd-progress`.
* **Detail:** see [use-skillgrid plan](#use-skillgrid-orchestrator-plan) below.

### onboard

* `sdd-onboard`
  * **Does:** Stage orchestrator for bootstrap. Greenfield vs brownfield; run helpers in safe order: map (if code) → init → agent-context → constraints → domain; summary + next (`propose` or idle).
  * **Origins:** gsd-core `gsd-onboard`, gentleman-ai `sdd-onboard` (gates without full teach-by-doing cycle).

* `sdd-init`
  * **Does:** Detect project facts; validate; write `docs/skillgrid/` skeleton, registry, agent block, Mnemonic, tracker (Backlog.md default) + triage labels.
  * **Origins:** skillgrid/gentleman `sdd-init`, mattpocock `setup-matt-pocock-skills`.
  * **Calls general:** `questioning` (tracker / ambiguous facts).

* Onboard helpers (not top-level stages; run under onboard/init or on refresh):
  * `sdd-map-codebase` ← brownfield optional map → `docs/skillgrid/codebase/` (primary nav = code-index)
  * `sdd-agent-context` ← `AGENTS.md` full block; other harness files = one-line pointer only
  * `sdd-constraints` ← quality bar into `config.yaml` `rules.*` (no root CONSTRAINTS.md by default)
  * `sdd-domain` ← `docs/skillgrid/glossary/` (sibling of `agents/`; no root CONTEXT.md by default)
  * skill-registry ← **optional** generated index; never an init gate

### propose

* `sdd-propose`
  * **Does:** Stage owner. Reserve `NNN-slug`; if hard explore or taste uncertainty remains, **stop and run** `sdd-explore` / `design-spike` first; then `questioning` (grill after concrete inputs); write `change.md` (user-visible end state + HOW; impl detail may be incomplete); list `Research:` and `Prototype:` paths when present; persist; stop before code.
  * **Origins:** gentleman-ai / skillgrid `sdd-propose`.
  * **Calls general:** `questioning`, `codebase-design`, `design-spike`, `glossary`.
  * **Calls workflow helper:** `sdd-explore` when research missing/needed/stale.

* `sdd-explore` *(helper, not a v4 top-level stage)*
  * **Does:** Optional — only when explore is hard (external API, rare docs, costly re-discovery). Write `research.md` with lifetime header (**this change only; may rot**). Fresh agents **read the cache** instead of re-exploring blindly. Do not promote to `codebase/` or ADRs by default.
  * **Origins:** gentleman-ai `sdd-explore`, gsd-core `gsd-explore`, mattpocock `research`.
  * **Calls general:** `investigate`, `dispatching-parallel-agents` when mapping large surfaces.

### spec

* `sdd-spec`
  * **Does:** Stage owner. Own `NN-*` numbering; write `tasks.md` **with blocking/depends edges** (kanban DAG for parallel apply) **and** `acceptance.feature`; inherit threats as `[RED]`; tracer-bullet tasks; then **stop for user gate**.
  * **Origins:** gentleman-ai `sdd-spec` + `sdd-tasks`, BMAD `bmad-spec`, gsd `gsd-spec-phase` / `gsd-plan-phase`, mattpocock `to-spec` / `to-tickets`, addyosmani planning/spec skills, intent-driven Gherkin/acceptance.
  * **Calls general:** `glossary`, `issue-creation` (when forced), `questioning` only if requirements still ambiguous (prefer revise propose).

### apply

* `sdd-apply`
  * **Does:** Stage owner. Execute unblocked steps/tasks against specs (sequential default; fan out only when blockers clear); use `Prototype:` path if listed; checkbox + State; persist apply-progress; hand off to verify. Designed for low babysitting when research + spike + change + kanban are solid.
  * **Origins:** gentleman-ai `sdd-apply`, mattpocock `implement`, BMAD `bmad-build`, gsd-core `gsd-execute-phase`.
  * **Calls general:** `isolated-workspace`, `tdd`, `debugging`, `subagent-execution` / `dispatching-parallel-agents`, `work-unit-commits`, `simple-execution`.

### verify

* `sdd-verify`
  * **Does:** Stage owner. (1) Agent gate — per-step verdicts, runtime proof for `@step-NN`, traceability in Evidence. (2) Write **human QA plan** (`qa-plan.md` or `## QA plan` in `tasks.md`). (3) Code review via `requesting-code-review` / `judgment-day` as needed. (4) On human QA or review findings → append tasks, set phase to **apply**, do not archive. (5) Archive only when agent gates PASS/WARNINGS, no open tasks, and human QA accepted (or explicitly waived).
  * **Origins:** gentleman-ai `sdd-verify`, superpowers verification + review skills, gsd `gsd-verify-work`, BMAD trace; human QA loop from 7-phase model.
  * **Calls general:** `verification`, `requesting-code-review`, `review-reception`, `judgment-day`.

### archive

* `sdd-archive`
  * **Does:** Stage owner. Gate on verify; `changes/` → `archive/` with readback; optional ship/PR path; extract learnings to Mnemonic; optional finish-branch decision.
  * **Origins:** gentleman-ai `sdd-archive`, gsd `gsd-cleanup` / `gsd-complete-milestone` / `gsd-ship` / `gsd-extract-learnings`, addyosmani `shipping-and-launch`, mattpocock/BMAD retro.
  * **Calls general:** `finishing-a-development-branch`, `mnemonic-memory`.

---

## General skills (cross-cutting)

Already in Skillgrid `.agents/skills/` unless noted. Workflow stages invoke these; do not recreate as `sdd-*`.

| Skill | Role | Typical callers |
|---|---|---|
| `questioning` | Unified intent / decision interview (design tree + frontier + approval gate) | onboard/init, propose, user-gate revise, orchestrator classify |
| `investigate` | High-trust research → markdown | explore, propose, Q&A |
| `design-spike` | Optional prototype **before** locking `change.md` (UI / arch / external smoke); commit marked PROTOTYPE; list path in change; not production; not during apply | orchestrator / propose pre-gate |
| `codebase-design` | Deep modules / seams / testability vocabulary | propose technical approach |
| `glossary` | Term consistency | propose, spec, domain bootstrap |
| `tdd` | Red-green-refactor | apply |
| `debugging` | Root-cause before fix | apply, verify FAIL loop |
| `isolated-workspace` | Worktree / isolation | apply start |
| `subagent-execution` | Fresh agent per plan slice | apply |
| `dispatching-parallel-agents` | 2+ independent slices | explore map, apply |
| `simple-execution` | Inline plan when subagents unnecessary | apply |
| `verification` | Evidence before “done” claims | verify, any completion claim |
| `requesting-code-review` | Ask for review | verify |
| `review-reception` | Process review feedback rigorously | verify → apply |
| `judgment-day` | Adversarial dual review | verify (high-risk) |
| `finishing-a-development-branch` | Merge / PR / discard | archive |
| `mnemonic-memory` | Persist decisions / learnings | all stages |
| `handoff` | Peel off an out-of-scope problem that appeared mid-work into a compact brief for a **subagent** (or separate session) — not for closing the SDD cycle | any stage when a side issue must leave the current context |
| `issue-creation` | Tracker tickets when `force_ticket_creation` | propose / spec |
| `work-unit-commits` | *(port if missing as first-class general)* reviewable commits | apply |

---

## use-skillgrid orchestrator plan

### Purpose

Single entry skill for Skillgrid SDD — counterpart to `using-superpowers`. Agents invoke it **before** inventing a change process. It only **routes**.

### Phase order (v4)

```
use-skillgrid
    │
    ├─ classify: Q&A → mem/code/investigate (no pipeline)
    ├─ classify: spike-only → design-spike (promote to propose if kept)
    ├─ uninitialized? → sdd-onboard (→ sdd-init + helpers) → stop for validation
    ├─ initialized + change (feature|bug|refactor|app)
    │       ├─ need hard research? → sdd-explore → research.md
    │       ├─ need taste / unknown shape? → design-spike → Prototype path
    │       └─ sdd-propose (+ questioning) → sdd-spec (blocking DAG)
    │              → USER GATE
    │              ├─ Implement → sdd-apply ⇄ sdd-verify
    │              │                 (QA plan + review; findings → apply)
    │              │                 → sdd-archive when both gates pass
    │              └─ Revise → questioning and/or sdd-propose
    └─ mid-change → resume from tasks.md ## State.phase
```

### Checklist (planned body)

```
[ ] 1. Classify: change (feature|bug|refactor|app) | Q&A/lookup | spike-only
[ ] 2. Detect initialized? (docs/skillgrid/config.yaml + AGENTS sentinel)
[ ] 3. If NO  → sdd-onboard / sdd-init; stop until user validates
[ ] 4. If YES + change → optional explore / design-spike → sdd-propose (unless Resume)
[ ] 5. After sdd-spec → user gate (Implement | Revise) — never auto-apply
[ ] 6. Apply ⇄ verify (human QA findings re-enter apply) → archive
[ ] 7. Announce: "Using use-skillgrid to <route>"
```

### Detection — initialized?

Uninitialized when **any** of:

1. No `docs/skillgrid/config.yaml`
2. No `<!-- skillgrid-sdd:start -->` … `<!-- skillgrid-sdd:end -->` in `AGENTS.md` / `CLAUDE.md` / `GEMINI.md`

Initialized when `config.yaml` exists (prefer also the AGENTS block).

### Resume map

| `## State.phase` | Action |
|---|---|
| missing / onboard incomplete | `sdd-onboard` / `sdd-init` |
| propose / explore | `sdd-propose` (run explore/spike first if still required) |
| spec | finish `sdd-spec` |
| apply | `sdd-apply` for unblocked `current_step` / tasks |
| verify | `sdd-verify` — on human QA findings → set phase apply |
| archive | `sdd-archive` |

### User gate (mandatory)

After `sdd-spec` writes `tasks.md` + `acceptance.feature`:

1. **Implement** → `sdd-apply`
2. **Revise** → `questioning` and/or `sdd-propose`

### What to drop from current v1 `use-skillgrid`

* Top-level stage name `explore` in the public pipeline (keep as propose helper).
* Direct jump to `sdd-explore` as first post-init step — prefer `sdd-propose`, which pulls explore when needed.
* Any implication that interview/TDD/debug are orchestrator-owned — they are general skills stages load.

### Skill priority

1. `use-skillgrid` — route  
2. `sdd-*` workflow stage — execute  
3. General skills — as the stage skill loads them  

### Router table (planned)

| Condition | First skill | Then |
|---|---|---|
| Uninitialized | `sdd-onboard` → `sdd-init` | After validate, if change stated → `sdd-propose` |
| Initialized + change | optional explore / `design-spike` → `sdd-propose` | `sdd-spec` → gate → `sdd-apply` ⇄ `sdd-verify` → `sdd-archive` |
| Q&A / lookup | *(no pipeline)* | `mnemonic-memory` / code-index / `investigate` |
| Spike-only | `design-spike` | Promote to propose if user keeps findings |
| Mid-change | Resume phase from State | verify findings may force apply |

---

## Compact inventory (after unification)

**Orchestrator:** `use-skillgrid`

**Workflow (7 + helpers):**  
`sdd-onboard`, `sdd-init`, `sdd-propose`, `sdd-explore` (helper), `sdd-spec`, `sdd-apply`, `sdd-verify`, `sdd-archive`  
+ onboard helpers: `sdd-map-codebase`, `sdd-agent-context`, `sdd-constraints`, `sdd-domain`

**General (do not stage-prefix):**  
`questioning`, `investigate`, `design-spike`, `codebase-design`, `glossary`, `tdd`, `debugging`, `isolated-workspace`, `subagent-execution`, `dispatching-parallel-agents`, `simple-execution`, `verification`, `requesting-code-review`, `review-reception`, `judgment-day`, `finishing-a-development-branch`, `mnemonic-memory`, `handoff`, `issue-creation`, `work-unit-commits`

---

## File Structure

Carry forward the v3 change-folder layout; first stage is **onboard**; **explore** is a propose helper. Canonical convention: `.agents/skills/_shared/conventions/sdd-structure.md` (update phase order when implementing v4).

### Multi-harness context model (rethink)

Target harnesses: **OpenCode, Kilo Code, VS Code (Copilot/agent), Cursor**. All share one Skillgrid tree; harness files only **point**, they do not fork project facts.

| Artifact | Verdict | Why |
|---|---|---|
| `docs/skillgrid/config.yaml` | **Required — single SoT** | Stack, tracker, testing, `rules.*` (quality bar), short `context`. Every harness reads the same file via the AGENTS block. No second YAML per tool. |
| `AGENTS.md` (+ optional one-line pointers in harness files) | **Required pointer** | Cross-agent standard. Full Skillgrid block once here; Cursor/OpenCode/Kilo/VS Code follow or get a one-line “see AGENTS.md”. Never duplicate project facts in four root files. |
| `docs/skillgrid/glossary/` | **Required when terms exist; stub on init** | Shared business/technical vocabulary for specs. Sibling of `agents/`, not inside it. Prefer this over a root `CONTEXT.md` glossary dump. |
| `docs/skillgrid/agents/skill-registry.md` | **Optional / generated** | Harnesses already discover `.agents/skills/`. Keep as on-demand index (script), not a gate. Do not block onboard if missing. Drop from “initialized?” detection. |
| `docs/skillgrid/codebase/` | **Optional — brownfield only** | Narrative map when code-index + AGENTS aren’t enough. Skip on greenfield. Refresh with `sdd-map-codebase`; not part of every change. Primary navigation = Mnemonic `code_*` ladder. |
| `CONSTRAINTS.md` (repo root) | **Do not require** | Quality bar lives in `config.yaml` → `rules.*` (tdd, coverage, verify commands, propose/spec gates). A root CONSTRAINTS.md duplicates and drifts. Optional human export only. |
| `CONTEXT.md` (repo root) | **Do not require** | Overlaps `config.yaml` `context:` + glossary. Optional mirror only; cite glossary/`config.yaml`, never a second SoT. |
| `docs/adr/` | **Optional — promote, don’t default** | See [change.md vs ADRs](#changemd-vs-docsadr) below. |

**Initialized?** = `docs/skillgrid/config.yaml` exists + Skillgrid sentinel block in `AGENTS.md` (or chosen primary). Not registry, not CONTEXT, not CONSTRAINTS, not `docs/adr/`.

### change.md vs docs/adr/

Two different lifetimes. Do not write every architecture choice twice.

| | `change.md` (Architecture decisions) | `docs/adr/NNNN-*.md` |
|---|---|---|
| **Role** | Decisions **for this change** — Choice / Alternatives / Rationale next to Step Blueprint | Decisions that **outlive one change** — system invariants future work must obey |
| **Required?** | **Yes** when the change has a non-trivial HOW | **No** — create only on promote |
| **Lifetime** | Lives with the change folder → moves to `archive/<NNN-slug>/` | Stays at repo level across archives |
| **Audience** | Agents applying **this** NNN | Agents on **any** later change |
| **When to use** | Default for all propose-time architecture | Promote when the choice becomes a standing rule (auth model, store, public API shape, monorepo boundary) |

**Rules**

1. **Propose writes decisions only in `change.md`.** Never open `docs/adr/` as part of every change.
2. **Promote to ADR** when (any): the decision constrains multiple future changes; reversing it would be a migration; it belongs in onboarding context for new agents; user explicitly asks to record a standing decision.
3. **On promote:** write `docs/adr/NNNN-slug.md` (MADR-style), link **from** `change.md` (`ADR: docs/adr/…`) and **to** the originating change (`Source: docs/skillgrid/changes|archive/<NNN-slug>/change.md`). Do not delete the change.md decision block — keep a short summary + link.
4. **Archive does not auto-promote.** Moving to `archive/` is not an ADR dump. Retro/learnings may *suggest* promote; human/agent chooses.
5. **Do not init an empty `docs/adr/`.** Create the directory on first promote.
6. **`codebase-design` / glossary** inform wording; they do not replace either artifact.

### Artifact File Structure

```bash
.backlog/                           # default tracker (Backlog.md) — if backlogmd
├── tasks/
├── config.yml
└── …

AGENTS.md                           # primary harness pointer (Skillgrid sentinel block)
# optional one-line pointers only:
#   .cursorrules / opencode / kilo / vscode agent files → "see AGENTS.md"

docs/
└── skillgrid/
    ├── config.yaml                 # REQUIRED SoT — stack, context, tracker, rules.*
    ├── agents/
    │   ├── issue-tracker.md        # tracker conventions for this repo
    │   ├── triage-labels.md
    │   └── skill-registry.md       # OPTIONAL generated index
    ├── glossary/                   # vocabulary SoT — sibling of agents/ (not nested)
    │   ├── business.md
    │   └── technical.md
    ├── codebase/                   # OPTIONAL brownfield map
    │   └── *.md
    ├── changes/
    │   └── <NNN-slug>/
    │       ├── research.md         # optional; lifetime = this change (rots)
    │       ├── change.md           # lists Research: / Prototype: when present
    │       ├── tasks.md            # State + steps + blocking edges + verify + QA plan section
    │       ├── acceptance.feature
    │       ├── qa-plan.md          # optional file; or ## QA plan inside tasks.md
    │       └── interview.md        # optional questioning log
    └── archive/
        └── <NNN-slug>/
```

### Who writes what

| Skill | Creates / updates | Path |
|---|---|---|
| `sdd-init` / `sdd-onboard` | skeleton + config | `docs/skillgrid/config.yaml`, `agents/` stubs, `changes/`, `archive/`, `AGENTS.md` block |
| `sdd-map-codebase` | optional map | `docs/skillgrid/codebase/` (brownfield) |
| `sdd-agent-context` | harness pointer(s) | `AGENTS.md` full block; other harness files = one-line pointer |
| `sdd-constraints` | quality bar **in config** | `config.yaml` `rules.*` — not a separate CONSTRAINTS.md by default |
| `sdd-domain` | vocabulary | `docs/skillgrid/glossary/{business,technical}.md` |
| `sdd-explore` | research (change-scoped; may rot) | `changes/<NNN-slug>/research.md` |
| `sdd-propose` | reserves NNN; change (+ Architecture decisions; `Research:` / `Prototype:` links) | `changes/<NNN-slug>/change.md` |
| *(promote)* | standing ADR | `docs/adr/NNNN-*.md` — only when decision outlives this change; link both ways |
| `questioning` | optional interview log | `changes/<NNN-slug>/interview.md` |
| `design-spike` | marked PROTOTYPE (commit OK); answer → change | near target code; path cited from `change.md` |
| `sdd-spec` | tasks + acceptance + **blocking DAG** | `tasks.md`, `acceptance.feature` |
| `sdd-apply` | checkboxes + State | `tasks.md` |
| `sdd-verify` | verdicts + trace + **human QA plan**; findings → apply | `tasks.md` `### Verification`; `qa-plan.md` or `## QA plan` |
| `sdd-archive` | move folder | `changes/<NNN-slug>/` → `archive/<NNN-slug>/` |
| `handoff` | out-of-scope brief | OS temp — **not** the change folder |
| *(optional)* registry script | generated index | `agents/skill-registry.md` — never a workflow gate |

### Notes

* **NNN** — `sdd-propose`. **NN** — `sdd-spec`. Never reuse / renumber.
* `tasks.md` carries **blocking/depends** edges for parallel apply.
* `research.md` lifetime = this change; may rot; do not promote by default.
* Human QA lives under verify (`qa-plan`); findings re-enter apply — not a separate stage.
* No `steps/` tree; no companion `*-glossary-reference.md`.
* Templates: `.agents/skills/_shared/templates/template-{change,tasks}.md` + `template-acceptance.feature` (extend for Depends / QA plan when implementing).
* Archive = **pure move** of the change folder.
* One facts file (`config.yaml`), one vocab tree (`glossary/`), one entry pointer (`AGENTS.md`). Everything else is optional or generated.

### Sub-Agent Scratch (optional runtime)

Ephemeral only — durable truth is `docs/skillgrid/` + Mnemonic:

```bash
.skillgrid/sdd/
└── <change-id>/
    ├── context/
    ├── phase-status/
    ├── work-unit/
    └── progress/
```

## Memory Topics (Mnemonic)

Always **hybrid** — filesystem + Mnemonic. Prefer Mnemonic for skill-index snapshots if you want search without committing `skill-registry.md`.

```
sdd-init/{project}
sdd/{project}/issue_tracker
sdd/{project}/testing-capabilities
sdd/{project}/changelog
sdd/<NNN-slug>/research
sdd/<NNN-slug>/change
sdd/<NNN-slug>/tasks
sdd/<NNN-slug>/spec
sdd/<NNN-slug>/grill
sdd/<NNN-slug>/issue-creation
sdd/<NNN-slug>/apply-progress
sdd/<NNN-slug>/verification
sdd/<NNN-slug>/qa-plan
sdd/<NNN-slug>/archive-report
```

## Ticketing

* Default: **Backlog.md** (`.backlog/`)
* Selectable: GitHub / GitLab / Jira / Backlog.md
* `issue-creation` maps propose/spec → tracker when `force_ticket_creation` is set
