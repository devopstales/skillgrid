# Skillgrid Release/2 — Engineering Operating System (design briefing)

> **STATUS:** `draft` (2026-10-09) — Release/2 Engineering Operating System Plan folded into this spec as the master vision (Option B, replace in-place). Existing workflow-update reqs 1–24 (rounds 3–12) are preserved as **sub-requirements** under the plan's phased roadmap (33-item initial order). **First slice** (the executable wedge) is TBD — the Release/2 vision re-opens scoping; the prior first-slice cut (reqs 21+19+8) is retained as a candidate but not confirmed under the new vision.
>
> **Master plan:** `.skillgrid/artifacts/skillgrid-workflow-update-plan.md` (1169 lines, 41 sections) — the authoritative vision this briefing operationalizes.
>
> **Topic:** 2026-10-02-skillgrid-workflow-updates
> **Date:** 2026-10-02 (origin); 2026-10-09 (Release/2 fold)
> **Classification:** standard (T2)
> **Build shape:** Full lifecycle (intent → verified delivery)
> **Queued behind:** `2026-10-02-mnemonic-project-init` (serial: do not take `current_change` until that change ships or parks). Sibling queued spec: `2026-10-02-mnemonic-llm-provider`.
> **Findings:** `findings.md` — GSD, BMAD, Cole, Superpowers, Matt, gstack, project-context skill, test-book, subagent execution research. These are **borrow inputs** to the Release/2 vision, not parallel authorities.

---

## 1. Vision and Objective

Skillgrid should evolve from a skill and rules installer into an **artifact-driven engineering operating system for humans and AI agents**.

Its responsibility is to manage the engineering lifecycle from initial intent to verified delivery. It must support product discovery, requirements, behavioral specifications, architecture, implementation planning, execution, review, verification, engineering memory, governance, recovery, and learning.

Skillgrid should not be merely:

- A collection of agent prompts.
- A coding-agent wrapper.
- A task tracker.
- A documentation generator.
- A memory search interface.
- An OpenSpec or other methodology wrapper.
- A WebUI that only displays agent conversations.

The core product is a **deterministic orchestration and governance layer** that connects engineering intent, durable artifacts, execution, evidence, and delivery.

### Guiding principle

> Skillgrid owns the engineering process. Agents perform the work. Mnemonic remembers the engineering. Git records source history. Canonical state records the current reality. Policy controls authority. Independent review challenges results. Verification proves defined claims. Humans retain control over consequential decisions.

### Final definition (Release/2)

**Skillgrid is an artifact-driven engineering operating system that coordinates humans and interchangeable AI agents through a governed, adaptive, recoverable lifecycle—from product intent and discovery through specifications, architecture, implementation, independent review, deterministic verification, engineering memory, approval, and delivery.**

Its differentiator is not code generation. It is the ability to define what success means, choose an appropriate process, preserve why decisions were made, constrain execution, challenge results independently, prove outcomes with evidence, and maintain trustworthy engineering state over time.

---

## 2. Problem / Intent

Skillgrid already copied GSD Core's rhetoric, several BMAD skills, Cole's PIV spine, Superpowers' catalog, and Matt's grill/domain/ticket/deep-module craft, but left *mechanical* sequencing, plan repair, AI-layer honesty, the session-start bootstrap, and two Matt gates as prose or a broken hook. Plans can silently drop requirements, `state.yaml` is a narrative log, the running agent is never told the window is filling, `AGENTS.md` can go stale, `.skillgrid/ARCHITECTURE.md` is listed as an artifact but was never created, agents search with `find` / `ls` / `rg` / `eza` instead of the Mnemonic code index, Cursor starts without the router, a turn that did not land has no named interrupt, and a shallow module still passes review.

The Release/2 plan reframes this: the gap is not more skills — it is that **Skillgrid does not own the engineering lifecycle**. Today the agent's conversation is the workflow state; artifacts are prose; verification is a prompt; memory is advisory. Release/2 makes Skillgrid the deterministic authority over intent → verified delivery, with durable typed artifacts, a policy engine, an execution engine, a verification engine, and Mnemonic as the engineering memory.

---

## 3. Purpose & Success Criteria

- **Purpose:** Make Skillgrid the canonical, deterministic, artifact-driven engineering operating system. Every stage of the lifecycle (intent → discovery → requirements → behavior → design → decision → plan → execution → review → verification → evidence → approval → delivery → learning) is owned by Skillgrid's runtime, not by an agent's memory or a single methodology.
- **Success criteria** (from plan §40):
  - Convert broad goals into appropriately sized engineering work.
  - Preserve one authoritative workflow and artifact structure.
  - Connect product intent, requirements, behavior, design, decisions, tasks, code, and evidence.
  - Select suitable skills and agents without hard-coding one provider.
  - Give agents bounded, context-rich execution packets.
  - Maintain durable state independently of conversations.
  - Produce reproducible, candidate-specific verification evidence.
  - Separate generation, evaluation, verification, authorization, and delivery.
  - Detect stale knowledge and reconcile inconsistent state.
  - Resume work after interruption or agent replacement.
  - Support both greenfield and brownfield repositories.
  - Operate locally with a useful default memory experience.
  - Provide an enterprise WebUI without creating a second workflow authority.
  - Remain useful for small changes without imposing unnecessary process.
  - Support complex, high-risk engineering work with appropriate rigor.
- **Out of scope** (from plan §37, "What Skillgrid Should Not Become"):
  - An OpenSpec wrapper.
  - A hard-coded linear methodology.
  - A coding-only lifecycle.
  - A single-agent orchestration framework.
  - A prompt collection without evaluation.
  - A memory database that silently overrides project truth.
  - A UI-only workflow tracker.
  - A process engine that forces every task through every stage.
  - An LLM-only enforcement system.
  - A system that treats generated documentation as verified truth.
  - A system that equates review approval with delivery authorization.
- **Also out of scope** (from prior findings, retained): replacing Mnemonic with GSD `.planning/` markdown or BMAD `.memlog.md` as the *memory* store; importing GSD's 70+ skill catalog, BMAD's 57-skill/six-module surface, Cole's 34 PIV skill names, Superpowers' 14-skill/14-harness plugin tree, or Matt's remaining catalog; BMM waterfall; CIS/party-mode theater; named personas as the default UX; Python/`uv` as the skill runtime; dark-factory/unattended autonomy; absent-equals-enabled defaults; worktree-by-default parallelism; a single growing `project-context` SKILL.md that copies PRD/ARCHITECTURE/terms/ADRs.

---

## 4. Context

Fits existing patterns: **yes-with-notes**.

The pipeline, `effort-budgets.md`, skill-size budget, `using-skillgrid` router, QA drift scripts, and hook dispatcher already exist. Notes:

1. Do not invent a second memory store — Mnemonic (Go) is the memory authority.
2. State mutations write the in-repo `state.yaml` the agent would have written — they must not hide project phase in SQLite.
3. Any new gate must honor the existing effort-budget table so `trivial` / `small` fast-track stays light.
4. Serial development stays locked — this folder is queued, not current.
5. Pipeline helpers are JS scripts (existing `scripts/*.mjs` / skill `scripts/` pattern). The Go runtime (Skillgrid core) handles workflow state, policy, and artifact engines; JS scripts handle parse/merge/validate/write helpers.
6. **Release/2 is a runtime evolution**, not a skill-only change. It adds Go runtime components (workflow state, policy engine, artifact engine, execution engine, verification engine) alongside the existing JS helper pattern.

Locked: Go 1.22+, no new dependencies without an ADR, serial one-change, spec-zone before code-zone, repo is source of truth (ADR-0013), SQLite second-brain store (ADR-0012).

---

## 5. Canonical Workflow Authority

The existing Skillgrid SDD structure remains authoritative.

All workflow changes must be reconciled with: `.agents/skills/_shared/rules/sdd-structure.md`

The plan must not introduce a parallel workflow that contradicts the canonical SDD rules. New capabilities, playbooks, artifact types, and adapters must integrate with the existing structure. The `.skillgrid/sdd` directory is an important part of the project's established workflow and must be preserved and extended according to the repository's actual conventions.

OpenSpec, BMAD, Superpowers, intent-driven templates, and other projects are research inputs. They do not become runtime dependencies or workflow authorities by default.

---

## 6. Architectural Invariants

1. **Canonical workflow first:** All capabilities conform to Skillgrid's SDD authority.
2. **Artifacts over conversation history:** Durable engineering state must not depend on an agent remembering previous messages.
3. **Typed interfaces:** Skills and agents exchange structured artifacts, contracts, and findings.
4. **Evidence before state:** A claimed outcome is not a verified outcome.
5. **Authority before automation:** Execution, approval, and delivery have distinct permissions.
6. **Immutable review targets:** Review and verification identify the exact candidate they evaluate.
7. **Independent evaluation:** The generator should not be the sole judge of its own work when risk warrants independence.
8. **Deterministic enforcement:** LLMs may propose and interpret; deterministic systems enforce schemas, permissions, transitions, and mechanical checks.
9. **Adaptive rigor:** Process cost must be proportional to uncertainty, risk, and change impact.
10. **Recoverability:** Work can be resumed, reconciled, rolled back, or sent back to an earlier stage.
11. **Replaceable agents:** No agent vendor or conversation format owns workflow state.
12. **Local-first operation:** The default experience should work locally, including on CPU-only systems.
13. **Traceability:** Requirements, decisions, code, verification, and delivery remain connected.
14. **No unnecessary ceremony:** Introduce process only when its expected engineering value exceeds its cost.

---

## 7. High-Level Architecture

```text
Human Goal
    |
    v
Intent and Problem Framing
    |
    v
Discovery and Decision Readiness
    |
    v
Adaptive Process Sizing
    |
    v
Capability Router
    |
    v
Workflow / Playbook Selection
    |
    v
Skill Composition
    |
    v
Typed Artifact Pipeline
    |
    v
Execution Packet
    |
    v
Policy and Authorization
    |
    v
Agent Adapter
    |
    v
Immutable Candidate
    |
    +----------------------+
    |                      |
    v                      v
Independent Review     Verification
    |                      |
    +-----------+----------+
                |
                v
        Findings and Evidence
                |
                v
      Bounded Correction Loop
                |
                v
        Canonical State Update
                |
                v
      Mnemonic Knowledge Update
                |
                v
      Approval and Delivery
                |
                v
    Post-Delivery Verification
                |
                v
      Reconciliation and Learning
```

Supporting components:

- **Skillgrid Runtime:** Workflow orchestration and state transitions.
- **Capability Router:** Selects capabilities, playbooks, skills, and suitable agents.
- **Policy Engine:** Enforces permissions, risk constraints, and authorization.
- **Artifact Engine:** Validates, versions, links, and transitions engineering artifacts.
- **Execution Engine:** Creates bounded execution packets and coordinates agents.
- **Verification Engine:** Executes checks against identified candidates.
- **Review Engine:** Coordinates independent, evidence-based review.
- **Reconciliation Engine:** Detects inconsistencies between source, artifacts, workflow state, and memory.
- **Mnemonic:** Provides durable engineering memory and semantic retrieval.
- **Git Integration:** Records source history and supports provenance.
- **CLI:** Exposes the core runtime.
- **WebUI:** Provides enterprise visibility and controlled interaction.

---

## 8. Requirements

The Release/2 plan (`.skillgrid/artifacts/skillgrid-workflow-update-plan.md`) is the master requirement set. It is organized into 41 sections; the operational requirement set for this spec is the **33-item Initial Implementation Order** (plan §39), grouped into 8 Phases (plan §38). The existing workflow-update reqs 1–24 (from prior interview rounds 3–12) are **sub-requirements** that map to specific phases.

### 8.1 Master Requirement Set (Release/2 Phases)

#### Phase 1 — Runtime Foundation

| # | Requirement | Sub-reqs (prior) |
|---|-------------|------------------|
| P1.1 | Confirm canonical SDD integration | — |
| P1.2 | Define core artifact schemas and identities | — |
| P1.3 | Implement workflow state and transition validation | Req 2 (deterministic state mutations) |
| P1.4 | Implement Canonical Project State | Req 2 |
| P1.5 | Define execution modes and authorization boundaries | — |
| P1.6 | Add workflow complexity budgeting | — |
| P1.7 | Implement Skill Registry and Skill Runtime Contract | Req 5 (two-stage routing), Req 7 (skill graph) |
| P1.8 | Define Execution Packets | Req 24a (filesystem-as-truth completion) |
| P1.9 | Implement Capability Router and initial playbooks | Req 5 |
| P1.10 | Define the Policy Engine | Req 23 (code-index enforcement policy), Req 17–18 (freeze/careful) |
| P1.11 | Establish the Agent Capability Matrix | — |
| P1.12 | Integrate Mnemonic | — |
| P1.13 | Define evidence and provenance models | — |

#### Phase 2 — Intent and Specification

| # | Requirement | Sub-reqs (prior) |
|---|-------------|------------------|
| P2.1 | Add Discovery capability | — |
| P2.2 | Formalize Goal and Done-When | — |
| P2.3 | Add typed requirements and Behavior Contracts | — |
| P2.4 | Implement Effective Engineering Contract | — |
| P2.5 | Add scenario lifecycle and supersession | — |
| P2.6 | Implement behavioral traceability | — |
| P2.7 | Define architecture and design artifact contracts | Req 22 (ARCHITECTURE.md enforcement) |
| P2.8 | Add ADR lifecycle and lineage | — |
| P2.9 | Add glossary and semantic consistency checks | — |
| P2.10 | Implement artifact transformation validation | — |
| P2.11 | Add brownfield discovery and incremental adoption | Req 20 (project-context map) |

#### Phase 3 — Bounded Execution

| # | Requirement | Sub-reqs (prior) |
|---|-------------|------------------|
| P3.1 | Implement execution packet creation and validation | Req 24a |
| P3.2 | Add thin agent adapters | — |
| P3.3 | Implement dependency-aware work graphs | — |
| P3.4 | Add context handoffs and resume points | Req 1 (context monitor), Req 6 (correct course) |
| P3.5 | Support interleaved specialist execution | Req 24 (subagent execution) |
| P3.6 | Implement immutable candidate creation | — |
| P3.7 | Add bounded correction work orders | — |
| P3.8 | Implement deterministic workflow reconciliation | — |

#### Phase 4 — Review and Verification

| # | Requirement | Sub-reqs (prior) |
|---|-------------|------------------|
| P4.1 | Implement the Verification Engine | Req 19 (validation matrix), Req 4 (Nyquist) |
| P4.2 | Add immutable verification targets and receipts | — |
| P4.3 | Add independent review pipelines | Req 24c (goal verifier) |
| P4.4 | Add quality floors and escalation | — |
| P4.5 | Implement minimal-change correction | — |
| P4.6 | Add behavioral verification adapters | — |
| P4.7 | Implement contract conformance tests | — |
| P4.8 | Add evidence-based state transitions | — |

#### Phase 5 — Engineering Memory and Code Index

| # | Requirement | Sub-reqs (prior) |
|---|-------------|------------------|
| P5.1 | Establish Mnemonic memory contracts | — |
| P5.2 | Add default CPU embedding capability | — |
| P5.3 | Add optional GPU, Ollama, and remote providers | — |
| P5.4 | Implement repository and code indexing | Req 23 (code-index enforcement) |
| P5.5 | Add symbol-aware indexing where practical | Req 23 |
| P5.6 | Implement code-linked engineering knowledge | — |
| P5.7 | Add deterministic knowledge attestation | — |
| P5.8 | Add memory drift and supersession | — |
| P5.9 | Implement change-to-knowledge impact analysis | — |
| P5.10 | Add engineering traceability graph | — |
| P5.11 | Add Git, pre-commit, and CI validation | — |

#### Phase 6 — CLI, Runtime Operations, and Recovery

| # | Requirement | Sub-reqs (prior) |
|---|-------------|------------------|
| P6.1 | Complete CLI command families | — |
| P6.2 | Implement `skillgrid doctor` | — |
| P6.3 | Add `skillgrid explain` and `skillgrid validate` | — |
| P6.4 | Add workflow and memory inspection commands | — |
| P6.5 | Add verification and rollback commands | — |
| P6.6 | Implement dry-run and recovery mechanisms | — |
| P6.7 | Add operational diagnostics and audit events | — |
| P6.8 | Verify local-first and offline operation | — |

#### Phase 7 — Enterprise WebUI

| # | Requirement | Sub-reqs (prior) |
|---|-------------|------------------|
| P7.1 | Implement `skillgrid serve` | — |
| P7.2 | Define and expose the runtime API | — |
| P7.3 | Build the Project Dashboard and Workflow Explorer | — |
| P7.4 | Build the Artifact Explorer and lineage views | — |
| P7.5 | Add Discovery, Requirements, and Behavior Contract workspaces | — |
| P7.6 | Add Architecture and ADR workspaces | — |
| P7.7 | Add Agent and Execution workspaces | — |
| P7.8 | Add Review and Verification workspaces | — |
| P7.9 | Add Mnemonic and Knowledge Graph views | — |
| P7.10 | Add Policy, Approval, and Audit workspaces | — |
| P7.11 | Add Recovery and Reconciliation views | — |

#### Phase 8 — Enterprise Governance

| # | Requirement | Sub-reqs (prior) |
|---|-------------|------------------|
| P8.1 | Add SSO and RBAC integration | — |
| P8.2 | Implement organization and project isolation | — |
| P8.3 | Add policy administration | — |
| P8.4 | Add skill and adapter supply-chain governance | — |
| P8.5 | Add enterprise audit and provenance | — |
| P8.6 | Add organization-level knowledge controls | — |
| P8.7 | Validate security and offline deployment requirements | — |

### 8.2 Sub-Requirements (Prior Workflow-Update Reqs 1–24)

The following are the detailed requirements from prior interview rounds (3–12). They are **sub-requirements** of the master phases above. Their locked decisions, acceptance criteria, and interview provenance are preserved here. The **first slice** (the executable wedge) is TBD under the Release/2 vision — the prior cut (21+19+8) is a candidate but not confirmed.

#### Req 1 — Agent-facing context monitor

The running agent is warned when remaining context is low; hooks fail open and never block the tool.

- **Maps to:** P3.4 (context handoffs and resume points)
- **Current:** Hooks record tool calls and checkpoint memory on stop/idle. No agent-facing headroom warning.
- **Target:** Advisory WARNING/CRITICAL at documented remaining-% (GSD's 35/25 is the starting number); debounce; point at `resume`/checkpoint, not at a new ceremony.
- **Acceptance:** TBD after interview (which harness events, which signal).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 2 — Deterministic state mutations (script-owned)

Phase, current change, and blockers change only through a command that writes `state.yaml`.

- **Maps to:** P1.3 (workflow state and transition validation), P1.4 (Canonical Project State)
- **Current:** `progress` is a concatenated narrative; drift scripts exist but do not advance state. Planning/`resume` skills describe file-inventory in prose.
- **Target:** A JS script (same family as `state-drift-check.mjs`) validates pointer fields against the spec folder and writes structured updates (BMAD `sprint_plan.py` logic: agent decides *which* change; script owns parse/merge/validate/write). Atomic write; JSON stdout including errors; `--dry-run`. Agents do not append free prose to `progress` as the state machine.
- **Acceptance:** TBD (validate/sync/advance equivalents; script tests).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 3 — Plan-check before execute

A planner that drops a requirement cannot pass slicing.

- **Maps to:** P4.1 (Verification Engine — behavioral traceability)
- **Current:** Human Go/Revise after `tasks.md`. No mechanical coverage or scope-recovery loop.
- **Target:** Requirements-coverage check + scope-reduction recovery (prohibit/check/re-inject), max 3 iterations, before the user gate.
- **Acceptance:** TBD.
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 4 — Nyquist contract

Every in-scope requirement names a runnable test command before implementation.

- **Maps to:** P4.1 (Verification Engine), P2.3 (typed requirements and Behavior Contracts)
- **Current:** Acceptance features and TDD exist; there is no per-requirement command contract written before code.
- **Target:** Wave-0 scaffolding for missing signals; later validate-phase may add tests only, not production code.
- **Acceptance:** TBD (attach to `acceptance.feature`, `#### Gates`, or both).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 5 — Two-stage skill routing

Eager skill listing is a handful of routers, not the full catalog.

- **Maps to:** P1.7 (Skill Registry and Skill Runtime Contract), P1.9 (Capability Router)
- **Current:** `using-skillgrid` routes; Cursor still sees the full skill list.
- **Target:** Namespace routers (workflow/project/quality/context/manage/ideate or a Skillgrid-native set) with short keyword descriptions; concrete skills stay directly invocable where the loader requires it.
- **Acceptance:** TBD (token/entry count before vs after on the target runtime).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 6 — Correct course

A named mid-change path writes a change proposal when the plan is wrong while tickets are in flight.

- **Maps to:** P3.4 (context handoffs and resume points), P3.8 (workflow reconciliation)
- **Current:** `resume` continues the plan; `reflect` only runs after ship. No impact pass across briefing/blueprint/tasks/ADRs.
- **Target:** Load those artifacts, assess impact, propose start-over/patch briefing/redo blueprint/reticket — human picks. Does not silently edit the plan.
- **Acceptance:** TBD (`resume` mode vs new skill).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 7 — Skill graph + completion markers

Next required skill and "started vs done" are queryable.

- **Maps to:** P1.7 (Skill Registry and Skill Runtime Contract)
- **Current:** `using-skillgrid` routes in prose; `resume` infers phase from file presence.
- **Target:** A small committed graph (phase, precedes, required, outputs). File presence means started; a finalization marker means done. Help/resume reads the graph.
- **Acceptance:** TBD (CSV/YAML vs generated from frontmatter).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 8 — Script contract on new pipeline helpers

Repeating work is a tested script, not skill prose.

- **Maps to:** P1.7 (Skill Registry and Skill Runtime Contract), P6.1 (CLI command families)
- **Current:** `deterministic-boundary.md` + `skill-write` state the rule. QA/ship/citations call scripts. `using-skillgrid`, `interviewing`, `writing-blueprints`, `slicing`, `resume` do not.
- **Target:** New helpers (state, phase-from-artifacts, requirement-coverage, catalog page) are `.mjs` next to the skill or under repo `scripts/`, following BMAD's contract in JS: structured stdout, atomic writes, tests beside the script (`scripts/test-*.mjs`), no interactive prompts. Skill line names the judgment/script split. No Python/`uv`. No new Go CLI for these jobs.
- **Acceptance:** Each extracted helper has a JS test; the calling SKILL.md is one line to run it plus judgment. `skill-write` quotes "agent decides *which*; script owns everything after."
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 9 — Rules-file drift check

After a change, `AGENTS.md` (and the locked-constraint mirror) is still true and no longer than it needs to be.

- **Maps to:** P2.9 (glossary and semantic consistency checks), P3.8 (workflow reconciliation)
- **Current:** `state-drift-check.mjs` watches `state.yaml` vs the spec folder. Nothing holds always-loaded rule claims against the diff. Wrong rules are worse than missing ones.
- **Target:** Pre-merge/QA advisory: flag only a now-false claim, a moved architecture-map path, or one new durable invariant. Mechanical path-claim check is a `.mjs`; the agent writes the one-line fix. Most diffs need no edit.
- **Acceptance:** TBD (attach to `qa`, `requesting-code-review`, or a pre-merge hook).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 10 — Declared file coupling

An edit of a coupled file cannot proceed until its declared dependencies were read this session.

- **Maps to:** P1.10 (Policy Engine)
- **Current:** GSD-style "read before edit" is advisory. Precommit zone guard is spec-vs-code, not file-vs-file (schema↔migration, route↔schema).
- **Target:** A fail-open JS hook plus a committed coupling map (Cole `dependencies.json` logic). No config → allow. Not Python/`uv`.
- **Acceptance:** TBD (first-slice hook vs follow-up; who authors the first couplings).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 11 — AI-layer opportunity from a real run

A finished run names the smallest durable change that would have prevented the failure — rule, skill, hook, or script.

- **Maps to:** P3.8 (workflow reconciliation), P5.8 (memory drift and supersession)
- **Current:** `reflect` writes a sourced retro after ship. It does not map the next primitive.
- **Target:** Reactive scan first, using the change's artifacts + Mnemonic `session_events`. Do not add a second `logs/agent-actions.jsonl` trail. Proactive log-window scan stays a follow-up.
- **Acceptance:** TBD (`reflect` section vs a separate invoke).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 12 — Ablate always-loaded rules (named follow-up)

Measure whether always-loaded instructions still change outcomes before cutting them.

- **Maps to:** P1.6 (workflow complexity budgeting)
- **Current:** Skill-size budgets shrink SKILL.md. Nothing experiments on `AGENTS.md` / always-loaded rules.
- **Target:** Map the layer (always-loaded/on-demand/enforcement). Same probe task, both arms, throwaway worktrees, working tree untouched. User picks the probe. JS `map_layer` at most in a later slice; no Python/`uv` bake-off in the first slice.
- **Acceptance:** Follow-up — listed so it is not dropped; not first-slice unless interview pulls it.
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 13 — Harness bootstrap

SessionStart injects `using-skillgrid` on every supported harness so skills are not inert files.

- **Maps to:** P1.7 (Skill Registry and Skill Runtime Contract), P3.2 (thin agent adapters)
- **Current:** `session-start.js` intends this but reads a path that does not exist (router lives under `lifecycle/`). Cursor SessionStart injects Mnemonic prime only.
- **Target:** One JSON shape per platform (Cursor `additional_context`, Claude nested, else SDK), tested in `.mjs`. Cursor gets router + prime. A clean "let's build X" session routes to `brainstorming` before code. Skills stay action-named, not tool-named. Do not rewrite SKILL.md per harness.
- **Acceptance:** TBD (first slice with routers vs later harness-install ticket; one hook vs two on Cursor).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 14 — Skill-behavior TDD (named follow-up)

A skill change is proven by a pressure scenario — fail without the skill, pass with it.

- **Maps to:** P4.7 (contract conformance tests), P1.7 (Skill Runtime Contract)
- **Current:** `skill-write` enforces the deterministic boundary and anatomy. It does not run a baseline-fail pass. Hook-shape tests exist for some scripts, not for SessionStart JSON.
- **Target:** Pressure scenario for skill text; hook-shape unit tests in `.mjs` (Superpowers `test-session-start.sh` logic). No Drill/`uv`/tmux in the first slice.
- **Acceptance:** Follow-up — listed so it is not dropped; not first-slice unless interview pulls it.
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 15 — wait-what

A named human interrupt re-pitches the last turn when it did not land.

- **Maps to:** P3.4 (context handoffs and resume points)
- **Current:** `resume` continues from files. Interviewing builds vocabulary up front. Nothing handles "I did not understand that."
- **Target:** User-invoked (or a `resume`/interviewing mode): re-pitch in the project terms, short, no new work. Not a second handoff format.
- **Acceptance:** TBD (new skill vs mode).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 16 — Deep-module gate

A blueprint names the seam and the interface test; review fails a new shallow module that is not called out.

- **Maps to:** P4.3 (independent review pipelines), P2.7 (architecture and design artifact contracts)
- **Current:** Vocabulary lives in `code-standards.md`, blueprint `codebase-design.md`, `ponytail`, and `99-logics.md`. Agents still ship busy interfaces. No gate.
- **Target:** Attach to `writing-blueprints` + `requesting-code-review`. Prefer existing/highest/fewest seams (`to-spec` quiz). Merges with Nyquist (req 4): the test command is at the interface. No TypeScript dependency-cruiser; no new Go dep without an ADR.
- **Acceptance:** TBD (blueprint only, review only, or both).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 17 — Edit freeze (proposed from gstack; not locked)

A user-activated session lock blocks writes outside one directory.

- **Maps to:** P1.10 (Policy Engine)
- **Current:** Zone guard is spec-vs-code at commit time. Cole coupling (req 10) is read-before-edit pairs. Nothing scopes "only this folder this session."
- **Target:** JS PreToolUse hook. Off → allow. On → block Edit/Write outside the path. Complements Cole coupling. Not Bun, not a persona.
- **Acceptance:** TBD (own hook vs a mode on the coupling hook; first slice vs follow-up).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 18 — Destructive-command careful (proposed from gstack; not locked)

Shell policy warns or hard-denies catastrophic commands.

- **Maps to:** P1.10 (Policy Engine)
- **Current:** `cursor-policy.sh` / `tool-call-capture.js policy` (ADR-0021) does not match `rm -rf`, `DROP TABLE`, force-push, or `kubectl delete`.
- **Target:** Extend that policy (or a sibling `hooks/careful.js`). MEDIUM → warn + override; root/home recursive delete and default-branch force-push hard-deny. JS. Fail open if the hook errors.
- **Acceptance:** TBD (extend existing policy vs new hook).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 19 — Technical validation plan (locked)

Before implementation, the change says how it will be proven — not only that tests will exist.

- **Maps to:** P4.1 (Verification Engine), P2.3 (Behavior Contracts)
- **Current:** Briefing lists requirements; blueprint Must-Haves list observable truths; Nyquist (req 4) will name a command; QA runs after code. This briefing itself still says "Acceptance: TBD" on most reqs. There is no matrix of happy/edge/failure, layer, command, and falsifier written *before* slicing. Agents implement, then invent the proof.
- **Target:** A committed `## Validation` (on the blueprint, or a sibling `validation.md` if interview splits it) that plan-check treats as required. One row per in-scope requirement: happy path, edge, failure; the check (command or hook); the layer (unit/script/gate/browser); what result would falsify it; out-of-scope (what we are *not* proving). Merges with Nyquist (the command column), Must-Haves (truths become rows), deep-module gate (check sits at the interface), and `99-logics.md` (happy/edge/failure; a green suite is not the verdict). A coverage script fails slicing if a requirement has no row. Do not import gstack `/plan-eng-review` persona or autoplan.
- **Acceptance:** TBD (section on `blueprint.md` vs `validation.md`; whether briefing may keep TBD until this section exists).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 20 — Project-context map (optional, not decided)

A fresh feature agent always knows the project exists, and loads only the the file the task needs. Not a second dump of the project.

- **Maps to:** P2.11 (brownfield discovery and incremental adoption)
- **Current:** `onboarding` already comes from BMAD `bmad-project-context`. `AGENTS.md` is the always-on pointer. `using-skillgrid` already reads config, resume, then the domain model. Observed misses become one line in `AGENTS.md` or a mnemonic note. There is no dedicated map skill. There is no post-ship reconcile that updates `ARCHITECTURE.md` and terms from the real repo while keeping handwritten lines and flagging conflicts.
- **What already replaces the video skill** (do not copy these into a `SKILL.md`):

  | Video skill holds | Use this instead |
  |---|---|
  | What the app does | `00-prd.md` + `ASSUMPTIONS.md` Product |
  | Pages / structure | `ARCHITECTURE.md` + code index |
  | Shared words | `01-business-terms.md` / `02-technical-terms.md` |
  | Hard decisions | `04-adr-NNNN-slug.md` (index in `ASSUMPTIONS.md`) |
  | What is locked | `ASSUMPTIONS.md` Locked constraints → `AGENTS.md` `### Rules` |
  | Stack | `.skillgrid/config.yaml` |
  | Always-on pointer | `AGENTS.md` `## Skillgrid` block |
  | Load on demand | Open the file, or `mem_search` → `mem_get_observation` |
  | After a real miss | One line in `AGENTS.md`, or a mnemonic note |

- **Target if taken:** Interview picks one: skip; a thin map skill (when to open which file); or map + post-ship reconcile of `ARCHITECTURE.md` and terms (handwritten lines stay; conflicts flagged, not overwritten). The skill body is a map only. Existing files stay the store.
- **Do not do (even if taken):** A growing `SKILL.md` that restates PRD/ARCHITECTURE/terms/ADRs. A one-file `CONTEXT.md`. Replacing Mnemonic or the committed artifacts with a skill body.
- **Acceptance:** TBD. Not first-slice unless interview pulls it.
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 21 — Scope pushback (locked)

When the user piles independent outcomes into one change, the agent stops and cuts — it does not refine a fat plan.

- **Maps to:** P1.6 (workflow complexity budgeting), P2.1 (Discovery capability)
- **Current:** Brainstorming already says "flag multiple subsystems immediately" and YAGNI. Interviewing scores Boundary Clarity. Serial one-change and "smallest usable whole" are locked. None of that *blocks* a briefing that keeps growing (this folder is the exhibit: six research sources, 20+ reqs, still one change). gstack's advantage is the pushback: office-hours reframes and recommends the narrowest wedge; CEO-review has Hold Scope/Scope Reduction as named modes.
- **Target:** A mandatory gate in `brainstorming` / `interviewing` (and again before `writing-blueprints`): if the request or draft briefing names more than one independently shippable outcome, stop. List the pile. Recommend one first slice (smallest usable whole) and queue or drop the rest. Human picks hold/cut/split. Do not write `blueprint.md` until the pick is recorded in the interview log. Default mode is **Hold/Reduce**, not Expand (Skillgrid is serial; gstack Expansion stays out). No CEO persona. No `/office-hours` import.
- **Acceptance:** TBD (clarity-gate dimension vs a hard stop in `using-skillgrid`; how "independently shippable" is counted).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 22 — ARCHITECTURE.md is created and kept current (proposed, queued)

`.skillgrid/ARCHITECTURE.md` exists once the repo has structure, and a change that adds or moves a component cannot ship without updating it.

- **Maps to:** P2.7 (architecture and design artifact contracts)
- **Current:** `AGENTS.md` lists `.skillgrid/ARCHITECTURE.md` as "Live repo/program structure", but the file does not exist in this repo. `onboarding` says it is "created lazily" (and may create it empty). Only the brainstorming **New Project** path writes it from `templates/ARCHITECTURE.md`. The **New Function** path updates it "only when they exist", so a brownfield repo never gets one. `ship` and `reflect` do not mention it. No script checks it. Skills that "read ARCHITECTURE.md if present" silently skip it.
- **Target:** (a) **Creation:** a deterministic `.mjs` scaffolds `ARCHITECTURE.md` from `templates/ARCHITECTURE.md` plus real repo facts (top-level dirs, modules, entrypoints from the code index), with section stubs the agent fills. Handwritten lines are never overwritten. Onboarding/reconcile and the first brainstorming of a brownfield repo run it when the file is absent or empty. (b) **Updating:** a drift script (same family as `state-drift-check.mjs`; overlaps req 9 rules-drift path claims) compares paths and components named in `ARCHITECTURE.md` with the tree and the change's diff. It flags a named path that no longer exists, and a new top-level component or store in the diff that the file does not name. QA/pre-ship reports it; the agent writes the one-line fix. (c) **User validation:** a structural update made by an agent is marked for review and does not count as accepted until the user confirms it. Not a dump of the code tree. Not a second store: the code index stays the source of facts, and `ARCHITECTURE.md` is the human-readable map.
- **Acceptance:** TBD (QA WARNING vs ship-blocking; generator inputs: code index vs `git ls-files`; whether a missing file blocks `writing-blueprints` for New Function; how the user-confirmed status is written).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 23 — Code search goes through the Mnemonic code index (proposed, queued)

When the project is indexed, an agent answers code-structure and code-search questions with the skillgrid MCP (`code_explore`, then `code_search` / `code_read`), not shell `find`, `locate`, `ls -R`, `rg`, `grep -r`, `fd`, `tree`, or `eza --tree`.

- **Maps to:** P5.4 (repository and code indexing), P5.5 (symbol-aware indexing), P1.10 (Policy Engine)
- **Current:** The Mnemonic prime says "Call code_explore before rg. rg is the escape hatch when code_explore is empty." The `mnemonic.mdc` rule describes the code-index ladder. Both are advisory, and agents still reach for `find`/`ls`/`rg`/`eza`. The enforcement point exists: `hooks/hooks-cursor.json` routes `beforeShellExecution`, `beforeMCPExecution`, and `beforeReadFile` through `cursor-policy.sh` → `tool-call-capture.js policy` → `/policy/evaluate` (ADR-0021, effects `block`/`warn`/`guide`/`allow`, fail open). But this repo has no `.skillgrid/policy.yaml`, so every command is allowed with no guidance. `AGENTS.md` has no code-index rule. Native agent search tools (Cursor `Grep`/`Glob`) do not pass through `beforeShellExecution`.
- **Target:** (a) A shipped default policy rule for `command_exec` that matches the search commands above and returns `guide` with a one-line pointer to `code_explore`/`code_search`/`code_read`. It escalates to `warn` after a repeat in the same session. It never blocks when the index is stale, missing, or Mnemonic is down (fail open, ADR-0021). Escape hatch: `code_explore` returned nothing for the query, or the target is not indexed (non-code files, `.skillgrid/` artifacts, gitignored paths). (b) A tested `.mjs` (or the policy test suite) holds the match list and the allow cases so the rule is not regex prose. (c) A one-line `AGENTS.md` rule and the `using-skillgrid` / `codebase-inspection` text point at the ladder. (d) Measure it: count of shell-search calls vs `code_*` calls per session from Mnemonic `session_events` (no second log), before and after.
- **Do not do:** Hard-deny every `rg` (it is the named escape hatch). Block `git ls-files`/`ls` of a single directory. Import a second search tool. Add a dependency.
- **Acceptance:** TBD (`guide` vs `warn` vs `block` threshold; how the hook learns the index is healthy without a slow round-trip; whether native `Grep`/`Glob` can be covered on Cursor or only on OpenCode/Kilo `tool.before`; whether the default ships in `~/.skillgrid/policy.yaml` via install or the repo `.skillgrid/policy.yaml`).
- **Acceptance scenario:** pending `acceptance.feature`

#### Req 24 — Subagent execution: filesystem-as-truth + post-wave gate + goal verifier (proposed; part (a) in first slice, parts 2–3 queued)

`subagent-execution` already matches GSD's per-task shape (fresh implementer + per-task review + two-axis final review); the gap is the three GSD mechanisms the skill's orchestrator lacks. Not wave fan-out, not worktree-by-default (both stay out — serial constraint).

- **Maps to:** P3.1 (execution packet creation), P3.5 (interleaved specialist execution), P4.3 (independent review pipelines — goal verifier)
- **Current:** `subagent-execution` dispatches fresh implementers per task and trusts the return message plus the ledger for completion; verification between tasks is the task reviewer; integration breakage between tasks is only caught by the end-of-change `skillgrid:qa` gate; nothing checks the change *goal* (not just task completion) — the Validation matrix (req 19) is checked for row coverage, not re-verified at the end by an independent reader. `parallel-execution/concurrent-leaves` is the lightweight fan-out case; the lease/wave machinery for 4+ leaves is explicitly "a future orchestrator skill, out of scope" (subagent-execution:264-267).
- **Target:** (a) **Filesystem-as-truth completion** (in first slice; edits `subagent-execution` + `parallel-execution` only, no new machinery): before a task is marked complete, the orchestrator spot-checks the worker's claimed artifacts on disk and the git log for its commits (the GSD `SUMMARY.md` + `git log --grep=<task-id>` move); a worker whose claims do not match disk or git is treated as not done, and the return message alone is never the completion signal. This strengthens the existing parent re-verify (Decompose & Gate) — the re-run covers the `#### Gates` oracles; this adds the artifact/commit check for harnesses where a subagent completion signal may be lost. (b) **Post-wave build+test gate** (queued behind the first slice): after each wave's tasks land, run `testing.runner` + `commands.build` before the next wave dispatches; a failing gate means the wave's integration is broken — stop, fix, re-run — instead of carrying breakage to the end-of-change QA. (c) **Goal verifier + gap-closure loop** (queued behind the first slice, synergetic with req 19): a separate verifier subagent (fresh context, most capable model per model-selection) re-runs the change's Validation matrix rows against the live tree and checks the briefing goal, not task checkboxes; gaps route to a named gap-closure pass (new tickets or a briefing patch via the correct-course path, req 6), then re-verify. The verifier is advisory-to-blocking: a gap it names cannot be parked without a ledger ruling.
- **Do not do:** Wave fan-out (waves sequential, plans-in-wave parallel) — fights the serial one-change lock; worktree-by-default isolation for every task; importing GSD's `STATE.md`/`REQUIREMENTS.md` split (state.yaml + briefing + Mnemonic already own that); replacing `simple-execution` (it stays the no-subagent-harness fallback, per its line 22).
- **Acceptance:** TBD (part (a) attaches to `subagent-execution`'s Decompose & Gate + Red Flags; part (b) to the wave transition in `subagent-execution` / `parallel-execution`; part (c) to the Final QA Gate boundary — a new `verification` subagent vs a `skillgrid:qa` mode; how the verifier's gap list maps to ticket IDs; whether (b) runs on `trivial`/`small` fast-track or only `standard`/`max` per the effort-budget table).
- **Acceptance scenario:** pending `acceptance.feature`

---

## 9. Implementation Decisions

- **Runtime (locked 2026-10-02, extended by Release/2):** JS scripts for parse/merge/validate/write helpers (steal BMAD's logic: JSON stdout including errors, atomic write, `--dry-run`, tests beside the script, "agent decides *which*; script owns everything after"). **Go runtime** for workflow state, policy engine, artifact engine, execution engine, and verification engine (Release/2 §38–39). Not Python/`uv`. Go 1.22+. No new dependencies without an ADR.
- **Modules to build/modify (Phase 1 first):** Go runtime skeleton (workflow state, transition validation, Canonical Project State); artifact schema definitions (YAML/JSON); policy engine (permissions, risk constraints, authorization); execution packet definition; capability router + initial playbooks; skill registry + runtime contract; Mnemonic integration (memory contracts, evidence/provenance models); JS helpers for state mutations, requirement coverage, validation-row coverage, rules-drift path-claims, SessionStart JSON shapes.
- **Interfaces:** each Go engine exposes a clean internal API; each JS helper prints one JSON object on stdout; exit 0 = pass, 1 = findings/fail, 2 = usage. Writes use temp + replace. `--dry-run` for mutating operations.
- **Data flow:** disk artifacts remain source of truth; Mnemonic mirrors as an index (existing dual-write rule). Canonical Project State is in-repo `state.yaml` (extended); Canonical Engineering State is in Mnemonic + committed artifacts.
- **Error handling:** new hooks fail open (ADR-0021 posture). A state script that cannot validate must not invent phase. Policy engine failures are advisory (warn), not blocking, unless the risk level warrants.
- **Dependencies:** no new dependency without an ADR. Use the existing `yaml` package where YAML is parsed.

---

## 10. Testing Decisions

Prior art for the locked JS contract: `scripts/test-state-drift.mjs`, `scripts/test-skill-size-budget.mjs`, `scripts/test-state-lock.mjs`. New helpers get a sibling `scripts/test-<name>.mjs`. SessionStart bootstrap gets a shape test (Cursor `additional_context` only / Claude nested only / SDK `additionalContext` only).

**Release/2 testing strategy** (plan §35):

- **Unit tests:** Artifact schemas, policy rules, state transitions, and individual capabilities.
- **Contract tests:** Runtime APIs, agent adapters, memory operations, and verification interfaces.
- **Workflow tests:** Complete and partial lifecycle scenarios, backtracking, recovery, and supersession.
- **Behavioral evaluations:** Whether skills and workflows produce expected outcomes in representative and adversarial cases.
- **Integration tests:** Mnemonic, Git, CLI, WebUI, model providers, and supported agent integrations.
- **End-to-end tests:** User goal through execution, review, verification, approval, and delivery.
- **Regression tests:** Prevent known failures, policy bypasses, invalid state transitions, and workflow rationalizations.
- **Cross-binding conformance:** Ensure different implementations satisfy the same normative contract.

Golden fixtures should cover success, failure, interruption, conflicting changes, invalid artifacts, missing evidence, and unauthorized operations.

**Validation planning (req 19) — starter matrix for *this* change** (rows fill when the first slice is cut; empty cells are the gap this req exists to close):

| Req | Happy | Edge | Failure | Check | Falsifier |
|-----|-------|------|---------|-------|-----------|
| 24a filesystem-as-truth | worker claims done; artifact on disk + commit in git log → marked complete | worker claims done; commit present, artifact missing → not done, fix round | completion signal lost (no return message) → spot-check decides, never waits | `subagent-execution` Decompose & Gate spot-check (script or rule, TBD per req 8) | task marked complete on return message alone |
| 8 script contract | helper JSON stdout, exit 0/1/2 | `--dry-run` writes nothing | missing args → exit 2 | `node scripts/test-<name>.mjs` | test file absent or asserts prose |
| 4 Nyquist | each in-scope req names a command | wave-0 scaffold only | req with no command | coverage `.mjs` vs briefing IDs | slicing passes with a blank command |
| 13 bootstrap | SessionStart injects router | missing file → INFO degrade | wrong JSON shape (double-inject) | `scripts/test-session-start-shape.mjs` | Cursor session has prime only |
| 19 this matrix | every locked req has a row | follow-ups (12, 14) marked out-of-scope | locked req with no row | same coverage `.mjs` | `tasks.md` exists, Validation empty |
| 21 pushback | pile of ≥2 shippable outcomes → stop + one recommended wedge | human picks hold / cut / split; pick logged | agent writes `blueprint.md` on a fat briefing | interview log + blueprint absence | blueprint exists, interview log has no pick |

A green helper suite is not the verdict for this change. The fresh check is: a planner that drops a validation row cannot pass slicing, and a "let's build X" session still hits `brainstorming` (req 13) when that slice lands.

---

## 11. Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: **Release/2 updates the PRD** — the product description changes from "skill installer" to "artifact-driven engineering operating system." The PRD needs a revision to reflect the new vision.
- `.skillgrid/ASSUMPTIONS.md`: **Release/2 will add ADRs** for the Go runtime architecture, artifact schema decisions, policy engine design, and verification engine design. Each is a one-way door requiring an ADR.
- `.skillgrid/ARCHITECTURE.md`: **Release/2 creates and maintains** `.skillgrid/ARCHITECTURE.md` (req 22). The high-level architecture diagram (plan §4) becomes the canonical architecture reference.
- `.skillgrid/artifacts/06-research-findings.md`: Durable distillation of the GSD, BMAD, colemedin-skills, Superpowers, and Matt comparisons (written with the prior draft). Release/2 findings (plan §2: "OpenSpec, BMAD, Superpowers, intent-driven templates, and other projects are research inputs") are captured here.
- `.skillgrid/artifacts/skillgrid-workflow-update-plan.md`: **The master plan** — 1169 lines, 41 sections. This briefing operationalizes it.

---

## 12. Clarity Report

Interview **not started** for the Release/2 vision. The prior 12 rounds (3–12) covered the workflow-update scope; the Release/2 fold re-opens scoping. Clarity gate has not run on the new vision.

| Dimension           | Score | Min  | Status | Notes |
|---------------------|-------|------|--------|-------|
| Goal Clarity        | —     | 0.75 | ⚠      | Release/2 vision is clear (plan §1, §40, §41); first slice TBD |
| Boundary Clarity    | —     | 0.70 | ⚠      | Catalogs / dark factory / Python hooks / Drill / skip-interview rejected; attach points open; Phase 1 scope needs cutting |
| Constraint Clarity  | —     | 0.65 | ⚠      | Serial + no-second-store + JS-scripts + Go 1.22+ + no-new-deps-without-ADR locked; Phase 1 implementation order is a "starting dependency sequence, not a prohibition" (plan §39) |
| Acceptance Criteria | —     | 0.70 | ⚠      | Reqs 1–24 have acceptance TBD; Release/2 phases need per-phase acceptance; validation matrix (req 19) is the mechanism |
| **Clarity**         | —     | ≤0.20| ⚠      | Do not write `blueprint.md` until interview exits on the Release/2 first slice |

**Interview log (prior rounds 3–12, retained as provenance):**

| Round | Question summary | Decision locked |
|-------|------------------|-----------------|
| 0     | Findings captured | 2026-10-02 from gsd-core, BMAD, colemedin-skills, superpowers, mattpocock-skills, and gstack |
| 1     | Script runtime?   | **JS scripts; steal BMAD logic.** Not Python/`uv`. Not a new Go CLI for these helpers. |
| 2     | Scripts vs guessing? | **Scripts over guessing.** Same input → same output is a `.mjs` run, not skill prose the model re-derives. |
| 3     | Cole on the plan? | **Yes.** Rules-file drift, declared file coupling, and reactive opportunity-scan are in-scope (reqs 9–11). Ablation is a named follow-up (req 12). Do not import the 34-skill catalog. |
| 4     | Superpowers on the plan? | **Yes.** Harness bootstrap is in-scope (req 13). Skill-behavior TDD is a named follow-up (req 14). Do not import the 14-skill catalog or Drill. |
| 5     | Matt on the plan? | **Yes.** `wait-what` and the deep-module gate are in-scope (reqs 15–16). Do not import wayfinder, skip-interview `to-spec`, `CONTEXT.md`, TS dep-cruiser, or a second handoff. |
| 6     | Technical validation planning lacking? | **Yes — lock it.** Req 19: a Validation matrix before implement (happy/edge/failure, check, falsifier). Must-Haves + Nyquist are not enough. Not gstack `/plan-eng-review`. |
| 7     | Karpathy-loop project-context skill? | **Optional, not decided.** Req 20. Skip / map-only skill / map + post-ship reconcile. Existing files stay the store. No dump SKILL.md. |
| 8     | gstack's real advantage? | **Scope pushback.** When the user wants too many things, stop and cut (Hold/Reduce). Locked as req 21. Not freeze/careful, not a CEO persona, not Expansion-as-default. |
| 9     | Which wedge is the first slice? | **First slice:** cut — reqs 21, 19, 8. Scope pushback + required Validation matrix + tested `.mjs` contract. Queue the rest. |
| 10    | Enforce ARCHITECTURE.md and the code index? | **Added as proposed reqs 22 and 23, queued behind the first slice.** Enforcement points: ARCHITECTURE scaffold + drift script; ADR-0021 policy hook `guide`/`warn` for shell search. |
| 11    | What can Skillgrid learn from test-book? | **Six borrow patterns captured** (2026-10-04): evaluator-independence grade, `manifest.yaml` spine, `evals.json`, packet schema, checkpoint-then-choice recovery, `allowed-tools` frontmatter. No new req; all attach to existing reqs or follow-ups. |
| 12    | Should Skillgrid run everything in subagents? | **One req 24 (2026-10-07):** (a) filesystem-as-truth completion lands in the first slice; (b) post-wave gate and (c) goal verifier are queued. **Rejected:** wave fan-out and worktree-by-default. |

**Release/2 fold (round 13, 2026-10-09):**

| Round | Question summary | Decision locked |
|-------|------------------|-----------------|
| 13    | How to integrate the Release/2 plan? | **Option B: replace in-place.** Existing reqs 1–24 become sub-requirements under the plan's phased roadmap (33-item initial order). The plan is the master vision; the prior reqs are preserved as provenance and mapped to specific phases. First slice TBD — the Release/2 vision re-opens scoping. |

---

## 13. Open Questions & Assumptions

- **Locked (round 13):** Release/2 plan is the master vision. Existing reqs 1–24 are sub-requirements mapped to phases. First slice TBD.
- **Locked (round 9, prior):** first slice was 21+19+8. This is a **candidate** under the Release/2 vision, not confirmed. The Release/2 plan's Phase 1 (Runtime Foundation) is a larger scope that may subsume or reorder the prior first slice.
- **Locked:** Scripts over guessing. Repeating procedures are JS `.mjs`; the agent does not re-derive them. Steal BMAD's contract. Not Python/`uv`. Go runtime for engines.
- **Locked:** Go 1.22+, no new dependencies without an ADR, serial one-change, spec-zone before code-zone, repo is source of truth (ADR-0013), SQLite second-brain store (ADR-0012).
- **Assumption:** Serial constraint holds — this change stays queued behind `mnemonic-project-init`.
- **Assumption:** Mnemonic remains the memory/code store; GSD intel/learnings files are not imported.
- **Question:** Phase 1 (Runtime Foundation) is 13 sub-requirements — what is the first executable slice? Is it the prior 21+19+8 cut, or a new slice from the Release/2 plan?
- **Question:** Does the Go runtime (workflow state, policy engine, artifact engine) land in the `mnemonic/` module or a new `skillgrid-runtime/` module?
- **Question:** What is the artifact schema format? YAML (matching existing `.skillgrid/` conventions) or JSON?
- **Question:** Does the Policy Engine extend the existing `cursor-policy.sh` / `tool-call-capture.js policy` (ADR-0021) or is it a new Go component?
- **Question:** Does the WebUI (Phase 7) replace or extend the existing embedded SPA?
- **Question:** What is the agent adapter interface? How do existing harness integrations (Cursor, OpenCode, Kilo) map to the Agent Capability Matrix?
- **Question (prior, retained):** All prior open questions from rounds 3–12 remain open (see prior briefing sections).

---

## 14. Decisions (ADR)

None from the Release/2 fold yet. The following will require ADRs when implemented:

- Go runtime module structure (new module vs `mnemonic/` submodule).
- Artifact schema format (YAML vs JSON).
- Policy Engine design (extension of ADR-0021 vs new component).
- Verification Engine design (candidate identity, receipt format).
- Agent Adapter interface.
- WebUI runtime API contract.

---

## 15. Terms

New terms from the Release/2 plan that may enter the glossary:

- **Effective Engineering Contract** — the resolved definition of what the system is currently expected to do (plan §11).
- **Execution Packet** — the boundary between Skillgrid orchestration and an agent (plan §17).
- **Immutable Candidate** — an immutable snapshot submitted for evaluation (plan §19).
- **Verification Receipt** — target identity, checks, results, and evidence (plan §20).
- **Canonical Project State** — what is happening now (plan §24).
- **Canonical Engineering State** — the current engineering model (plan §24).
- **Capability Router** — selects capabilities, playbooks, skills, and suitable agents (plan §4).
- **Bounded Correction Loop** — failed review/verification produces an explicit Correction Work Order (plan §21).
- **Workflow Complexity Budget** — prevents unnecessary skills, agents, reviews, and artifacts from accumulating (plan §8).
- **Behavior Contract** — observable and testable behavior (plan §10).
- **Adaptive Process Sizing** — workflow matches the task (plan §8).

These are proposed, not locked. If the interview keeps any of these as Skillgrid terms, add them to `02-technical-terms.md` then.
