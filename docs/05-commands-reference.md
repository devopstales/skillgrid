# Skillgrid Commands & Workflows Reference

This document lists every slash command, what it does, and which skills it uses.

## Quick Reference

| Command | Phase | Description |
|---------|-------|-------------|
| `/sdd-init` | Setup | Initialize SDD context, detect stack, bootstrap persistence |
| `/sdd-brainstorm <name>` | Planning | Full planning pipeline: explore → propose → spec → design → tasks |
| `/sdd-explore <topic>` | Planning | Free-form codebase investigation (no code changes) |
| `/sdd-clarify <name>` | Planning | Interactive questioning to sharpen terminology, update CONTEXT.md |
| `/sdd-apply [name]` | Build | **Worker:** implement tasks (all remaining, or one task when called from the loop) — TDD, sequential subagents |
| `/sdd-loop [name]` | Build | **Ralph loop:** one AFK task per fresh invocation — plan → delegate to `/sdd-apply` → reflect → stop (no direct coding) |
| `/sdd-verify [name]` | Verify | **Stage 1:** Spec compliance verification — trace requirements to code/tests |
| `/sdd-review [name]` | Verify | **Stage 2:** Code quality review — style, DRY, error handling, security, maintainability |
| `/sdd-archive [name]` | Archive | Sync delta specs, merge/PR/keep branch — requires verify + review + pre-merge gate passed |
| `/sdd-design-ui [surface]` | Specialist | UI design workshop with high-fidelity skills |
| `/sdd-diagnose <bug>` | Specialist | **4-phase systematic debugging:** reproduce → isolate → root cause → fix → verify |
| `/sdd-openspec-git` | Gate | OpenSpec git discipline gates |

**Norse personas** are not separate commands. The coordinator dispatches them per phase — see **`sdd-<phase>/SKILL.md`** (section *Norse persona invocations*) and **`sdd-persona-delegation.md`**. Guide: **`docs/10-subagent-personas.md`**.

---

## SDD Core Workflow

### `/sdd-init`

**Phase:** Setup (run once per project)

**What it does:** Detects project stack, conventions, and testing capabilities. Bootstraps the active persistence backend (engram, openspec, hybrid, or none). Creates `.skillgrid/` and `openspec/` directory structures. Builds the skill registry. Refreshes semantic indexes.

**Skills used:** `sdd-init`, `skillgrid-skill-registry`, `ccc`, `gitnexus-cli`, `openspec-onboard`

**Input:** None (auto-detects project)

**Output artifacts:**
- `.skillgrid/config.json`
- `.skillgrid/project/ARCHITECTURE.md`
- `.skillgrid/project/PROJECT.md`
- `.skillgrid/project/STRUCTURE.md`
- `.skillgrid/project/SKILL_REGISTRY.md`
- `openspec/config.yaml`
- `openspec/specs/`
- `openspec/changes/`

---

### `/sdd-brainstorm <change-name>`

**Phase:** Planning (start of every new change)

**What it does:** Orchestrates the full planning pipeline. Delegates to sub-agents for each phase in sequence. Integrates UI design sub-flow and optional Beads sync.

**Skills used (sequential pipeline):**

| Step | Skill | Output |
|------|-------|--------|
| 1 | `sdd-explore` + `deep-research` | Web research first, then codebase investigation and approach comparison |
| 2 | `sdd-clarify` | Interactive questioning, CONTEXT.md updates |
| 3 | `sdd-propose` | `proposal.md` — intent, scope, approach, rollback plan |
| 4 | `sdd-spec` | `specs/<domain>/spec.md` — delta specs with Given/When/Then |
| 5 | `sdd-design` | `design.md` — architecture decisions, data flow, file changes; ADRs in `.skillgrid/adr/` when triggers apply |
| 6 | `sdd-design-ui` (conditional) | UI design artifacts if user-facing scope |
| 7 | `sdd-prd` | `.skillgrid/prd/PRD<NN>_<slug>.md` — product requirements |
| 8 | `sdd-tasks` | `tasks.md` — implementation checklist with HITL/AFK labels |
| 9 | `beads-sync` (conditional) | Beads epic + issues if `beads_enable: true` |

**Input:** Change name (kebab-case)

**Output artifacts:**
- `openspec/changes/<name>/proposal.md`
- `openspec/changes/<name>/specs/<domain>/spec.md`
- `openspec/changes/<name>/design.md`
- `openspec/changes/<name>/tasks.md`
- `.skillgrid/prd/PRD<NN>_<name>.md`
- Optional: ADRs, Beads epic/issues, UI previews

---

### `/sdd-explore <topic>`

**Phase:** Planning (standalone investigation)

**What it does:** Free-form exploration of a topic or feature idea. Reads codebase, compares approaches, provides recommendation. Makes NO code changes. When the topic is an architectural decision, drafts or reviews an ADR in `.skillgrid/adr/` using `architectural-decision-records`.

**Skills used:** `sdd-explore`, `deep-research` (first search: Exa MCP → Tavily → Firecrawl), `parallel-delegate`, `exa-search`, `architectural-decision-records` (ADR-focused topics)

**Input:** Topic to explore, or decision topic / ADR path for ADR authoring or review

**Output:** Exploration findings (engram topic or `exploration.md`); or ADR file(s) and `ARCHITECTURE.md` updates for ADR-focused runs

---

### `/sdd-clarify <change-name>`

**Phase:** Planning (interactive questioning)

**What it does:** Challenges the plan against the existing domain model. Sharpens terminology, resolves ambiguity, updates `.skillgrid/project/CONTEXT.md`.

**Skills used:** `sdd-clarify`

**Input:** Change name

**Output:** Updated `CONTEXT.md`, clarified terminology

---

### `/sdd-apply [change-name]`

**Phase:** Implementation (worker)

**What it does:** Writes code and tests from `tasks.md`, specs, and design. When you invoke it directly, it works through **remaining** incomplete tasks (granular planning, sequential subagents, TDD, two-stage review) until done or blocked.

When invoked **from `/sdd-loop`**, it implements **only the one task** named in the delegation prompt, then returns.

**Skills used:** `sdd-apply`, `granular-planning`, `sequential-agent-executor`, `enforced-tdd-protocol`

**Preflight:** `.skillgrid/scripts/sdd-gate.sh apply --change <name>` (canonical gate)

**Output:** Code, tests, updated `tasks.md`, commits, standard return envelope

---

### `/sdd-loop [change-name]`

**Phase:** Build — Ralph loop (orchestrator only)

**Full documentation:** [SDD Ralph Loop](11-sdd-ralph-loop.md)

**What it does:** Runs **one iteration** per invocation ([Ralph pattern](https://ghuntley.com/ralph/)): pick the next `[Label: AFK]` task from `tasks.md`, delegate it to `/sdd-apply` with explicit single-task scope, append learnings to `ralph-loop-state.md` / `progress.txt`, then **stop**. Does not write application code.

**Not the same as `/sdd-apply`:** loop = conductor; apply = musician.

**Architecture (one iteration):**
```
PLAN → DELEGATE (/sdd-apply, one task) → REFLECT → STOP
```

**AFK multi-iteration driver:**
```bash
.skillgrid/scripts/sdd-ralph-loop.sh <change-name> [max-iterations]
```
Uses `SDD_RALPH_AGENT` (`claude` | `opencode` | `cursor`). Stops when output contains `<promise>COMPLETE</promise>` ([snarktank/ralph](https://github.com/snarktank/ralph), [aihero Ralph guide](https://www.aihero.dev/getting-started-with-ralph)).

**Completion:** `<promise>COMPLETE</promise>` when all AFK tasks are `[x]`; then run `/sdd-verify`.

**Memory between iterations:** `tasks.md`, git commits, `.skillgrid/tasks/research/<change>/ralph-loop-state.md`, `progress.txt`

**Skills used:** orchestration only; execution via `sdd-apply`

---

### `/sdd-verify [change-name]`

**Phase:** Verification — Stage 1 (Spec Compliance)

**What it does:** Validates that implementation fully satisfies the slice specification. Builds traceability matrix: every requirement → concrete evidence (code/test file:line). Does NOT review code quality — that's `sdd-review`.

**Skills used:** `sdd-verify` (orchestrator), `spec-compliance-verifier` (core logic)

**Process:**
1. Read all slice specs for the change
2. For each slice: parse requirements, gather evidence from codebase
3. Build traceability table (requirement → evidence location)
4. Determine verdict: **PASS** (all satisfied) / **FAIL** (missing requirements) / **PARTIAL** (incomplete)
5. Save report: `openspec/changes/<id>/verification/<slice>-report.md`

**Verdict meanings:**
- **PASS** → proceed to `sdd-review`
- **FAIL** → fix missing requirements via `sdd-apply`, then re-run
- **PARTIAL** → treat as needing work (some requirements incomplete)

**Gate:** `sdd-archive` checks `.skillgrid/state/verification_status = passed` — requires PASS to proceed.

---

### `/sdd-review [change-name]`

**Phase:** Verification — Stage 2 (Code Quality)

**What it does:** Reviews implementation for code health independent of spec compliance. Evaluates readability, DRY, error handling, test quality, security, performance, maintainability. Produces severity-tagged issues (CRITICAL/IMPORTANT/MINOR) and APPROVED/CHANGES_REQUESTED verdict.

**Skills used:** `sdd-review` (orchestrator), `code-quality-reviewer` (core logic)

**Process:**
1. Confirm `sdd-verify` passed (pre-check)
2. Invoke `code-quality-reviewer` on changed files
3. Categorize issues by severity
4. Provide concrete fixes for each issue
5. Verdict: **APPROVED** (zero CRITICAL, zero or resolved IMPORTANT) or **CHANGES_REQUESTED**

**Review loop:** If issues found → implementer fixes → re-run `sdd-review --re-review` until APPROVED.

**Flags:**
- `--slice <slug>` — review specific slice only
- `--re-review` — focus on previously flagged items
- `--reviewer <persona>` — delegate review to persona (e.g., `thor`, `heimdall`)
- `--force` — skip already-reviewed check

**Gate:** `sdd-archive` requires APPROVED review report present.

---

### `/sdd-archive [change-name]`

**Phase:** Archiving (after both verification gates pass)

**What it does:** Three-gate precheck before archiving:
1. **Spec compliance** (`sdd-verify` passed)
2. **Code quality** (`sdd-review` approved)
3. **Pre-merge verification** (tests green, lint clean, working tree clean, branch mergeable, security scan)

Then executes user-chosen disposition: merge to main / open PR / keep branch / discard.

**Skills used:** `sdd-archive` (orchestrator), `pre-merge-verification` (gate)

**Input:** Optional change name

**Disposition** (from config or prompt):
- `merge` — merge branch to main, push
- `pr` — push branch, create PR via `gh`
- `keep` — leave branch, no merge
- `discard` — delete branch, keep local changes if any

**Post-archive:**
- Move `openspec/changes/<id>/` → `openspec/archive/YYYY-MM-DD-<id>/`
- Update `.skillgrid/prd/INDEX.md` if PRD-linked (mark tasks complete)
- Clear active change state

**Gate enforcement:** If any precheck fails → `status: blocked`, `next_recommended` lists fixes. No archive until all three gates pass.

---

## Specialist Commands

### `/sdd-design-ui [surface]`

**What it does:** UI design workshop. Generates visual direction options, compares across accessibility, responsive, cost, and design-system fit. Optionally produces high-fidelity previews.

**Skills used:** `sdd-ui-design`, `sdd-design`, `engram-ui-elements`, `engram-visual-language`, `design-taste-frontend`, `frontend-ui-engineering`, `high-end-visual-design`, `impeccable`, `superdesign`, `huashu-design`, `image-to-code`

**Input:** Surface or change name

**Output:** UI design direction, preview artifacts (via `preview.sh`)

---

### `/sdd-diagnose <bug>`

**Phase:** Specialist (Debug)

**What it does:** Systematic debugging using 4-phase protocol:
1. **Root Cause Investigation** — gather evidence, reproduce consistently, instrument multi-component boundaries, trace data flow backward
2. **Pattern Analysis** — find working examples, compare, read references completely
3. **Hypothesis & Test** — generate 3–5 ranked falsifiable hypotheses, test minimally one variable at a time
4. **Implementation** — create failing regression test (if seam exists), apply minimal fix, verify, cleanup

**Enforcement:** NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST. Three-fixes threshold: if 3+ attempted fixes fail → STOP → question architecture → escalate to `sdd-architecture-review`.

**Skills used:** `sdd-diagnose` (orchestrator), systematic-debugging methods integrated into skill

**Input:** Bug report, test failure, unexpected behavior

**Output:** Diagnostic report with evidence, root cause, fix, verification, post-mortim. Saved to `.skillgrid/tasks/research/<issue-id>-diagnosis.md`

**Integration:** After diagnosis, optional `sdd-apply` to implement thorough fix if initial fix was minimal. If architecture friction found → `sdd-architecture-review`.

---

### `/sdd-openspec-git`

**What it does:** Enforces OpenSpec git gates — ensures proposal/apply/archive cross main branch correctly, commits are explicit. Never auto-commits or merges.

**Skills used:** `openspec-git-discipline`

**Input:** Optional context or change name

**Output:** Git facts, gate outcomes, next recommended action

---

## Norse personas (per-phase, in skills)

Each SDD phase skill and workflow includes **Norse persona invocations (coordinator)** — which `persona` + `capability` to dispatch before the phase completes. Protocol: **`.agents/skills/_shared/sdd-persona-delegation.md`**.

Example (`sdd-verify`): required `tyr` → `spec-compliance`; conditional `heimdall`, `frigg`, `loki`.

---

## Utility Skills (Auto-Loaded by Context)

These skills are not invoked directly as commands but are auto-loaded when the orchestrator detects relevant context:

| Skill | Trigger Context |
|-------|-----------------|
| `skillgrid-tdd` | Writing tests, `--tdd` flag, RED/GREEN/REFACTOR |
| `skillgrid-vertical-slices` | Breaking down work, creating issues, planning implementation |
| `skillgrid-skill-registry` | Initializing Skillgrid, refreshing project context |
| `micro-plan` | Quick fix, "few steps", "simple plan", small operational plan |
| `parallel-delegate` | Multiple independent files, parallel research passes |
| `deep-research` | First-search orchestration for explore/brainstorm (Exa → Tavily → Firecrawl) |
| `exa-search` | Exa MCP tools (`web_search_exa`, `web_fetch_exa`) |
| `ccc` | Semantic code search, indexing after changes |
| `gitnexus-cli` | Analyzing/indexing repos, generating wikis |
| `full-output-enforcement` | Tasks requiring exhaustive, unabridged output |
| `markdown-converter` | Converting PDF, DOCX, PPTX, XLSX to markdown |
| `karpathy-guidelines` | Writing/reviewing/refactoring code to avoid LLM pitfalls |

## Engram Guardrail Skills (Auto-Loaded by Project)

These enforce project-specific conventions for the Engram project. They are auto-loaded when relevant files are touched:

| Skill | Trigger |
|-------|---------|
| `engram-project-structure` | Creating files, packages, handlers, templates, tests |
| `engram-commit-hygiene` | Commit creation, review, branch cleanup |
| `engram-testing-coverage` | Implementing behavior changes |
| `engram-server-api` | Route, handler, payload, status code changes |
| `engram-dashboard-htmx` | HTMX attributes, partial updates, forms |
| `engram-ui-elements` | Dashboard UI components, connected browsing flows |
| `engram-visual-language` | Typography, spacing, color, visual identity changes |
| `engram-tui-quality` | Bubbletea/Lipgloss model, update, view changes |
| `engram-gentleman-bubbletea` | Go files in `installer/internal/tui/` |
| `engram-architecture-guardrails` | System boundaries, ownership, state flow changes |
| `engram-business-rules` | Sync behavior, project controls, permissions |
| `engram-plugin-thin` | Plugin scripts/hooks changes |
| `engram-docs-alignment` | Code changes affecting user/contributor behavior |
| `engram-cultural-norms` | Starting work, reviewing changes, defining conventions |
| `engram-memory-protocol` | Decisions, bugfixes, discoveries, session closure |
| `engram-backlog-triage` | Auditing issues/PRs, triaging backlog |
| `engram-issue-creation` | Creating GitHub issues, reporting bugs |
| `engram-branch-pr` | Creating PRs, opening PRs, preparing branches |
| `engram-pr-review-deep` | Reviewing PRs as reviewer |
| `engram-sdd-flow` | SDD or multi-phase implementation planning |

---

## Workflow Diagram

```
/sdd-init
    │
    ▼
/sdd-brainstorm <name>
    ├── sdd-explore     → deep-research (web) then codebase investigation
    ├── sdd-clarify     → terminology, CONTEXT.md
    ├── sdd-propose     → proposal.md
    ├── sdd-spec        → specs/<domain>/spec.md
    ├── sdd-design      → design.md (+ ADRs in `.skillgrid/adr/` when triggers apply)
    ├── sdd-design-ui   → UI artifacts (if user-facing)
    ├── sdd-prd         → PRD<NN>_<name>.md
    ├── sdd-tasks       → tasks.md (HITL/AFK labels, granular, TDD-compliant)
    └── beads-sync      → Beads epic + issues (if enabled)
        │
        ▼
/sdd-apply <name>
    ├── granular-planning check   → tasks are atomic (2–5 min each)
    ├── sequential-agent-executor → per-task subagent dispatch
    │   ├── Implementer (RED/GREEN/REFACTOR)
    │   ├── Spec compliance review (stage 1)
    │   └── Code quality review (stage 2)
    └── TDD evidence collected
        │
        ▼
/sdd-verify <name>           [STAGE 1: SPEC COMPLIANCE]
    ├── Parse slice specs → requirements
    ├── Gather evidence (code, tests, config)
    ├── Build traceability matrix
    └── Verdict: PASS / FAIL / PARTIAL
        │
        ▼ (PASS)
/sdd-review <name>           [STAGE 2: CODE QUALITY]
    ├── Analyze: style, DRY, errors, tests, security, perf
    ├── Severity-tag issues (CRITICAL/IMPORTANT/MINOR)
    └── Verdict: APPROVED / CHANGES_REQUESTED
        │
        ▼ (APPROVED)
pre-merge-verification        [FINAL GATE]
    ├── sdd-verify passed ✓
    ├── sdd-review approved ✓
    ├── tests green ✓
    ├── lint clean ✓
    ├── working tree clean ✓
    ├── branch mergeable ✓
    └── security scan ✓ (if enabled)
        │
        ▼ (ALL PASS)
/sdd-archive <name>
    ├── Choose disposition: merge / PR / keep / discard
    ├── Execute disposition
    └── Archive artifacts → openspec/archive/
        │
        ▼
beads-retrospective
    ├── analyze patterns, tech debt
    └── suggest new OpenSpec proposals
```

---

## Return envelope (all phases)

Every `sdd-*` command and phase skill ends with the **same structured envelope** (JSON or YAML). Canonical contract: [`.agents/skills/_shared/sdd-return-envelope.md`](../.agents/skills/_shared/sdd-return-envelope.md).

| Field | Required | Notes |
|-------|----------|-------|
| `status` | yes | `completed`, `blocked`, or `failed` |
| `executive_summary` | yes | `overview` (1–3 sentences), `used_tokens` |
| `detailed_report` | no | Verbose matrix, checks, or phase-specific detail |
| `artifacts` | yes | Paths written or validated |
| `next_recommended` | yes | Next safe command or action |
| `risks` | yes | List or `none` |
| `skill_resolution` | yes | How project standards were loaded |

On `blocked` or `failed`, also include stop condition, failing gate id, and missing evidence (see contract). Extensions: verify/VDD converge fields, loop `<promise>COMPLETE</promise>`, persona subagent fields — all documented in the contract file.

---

## Utility Scripts

Located in `.skillgrid/scripts/` (see [scripts README](../.skillgrid/scripts/README.md)):

| Script | Purpose |
|--------|---------|
| `checkpoint-record.sh` | Tier 1 checkpoint — see [Checkpoints](14-checkpoints.md) (`skillgrid checkpoint` wraps this) |
| `validate-task-labels.sh` | Validates `[Label: AFK|HITL]` and `[Budget: safe|RISK]` in tasks.md |
| `preview.sh` | Scaffolds preview artifacts for UI design |
| `handoff-create.sh` | Creates a new handoff record |
| `handoff-resume.sh` | Resumes from a handoff record |
| `handoff-validate.sh` | Validates handoff record integrity |
| `handoff-check-staleness.sh` | Checks if handoff is stale |
| `handoff-resume.sh list` | Lists session handoffs under `.skillgrid/handoffs/` |
| `handoff-registry-init.sh` | Initializes `registry_<change-id>.md` dispatch index |
