# Workflow Usage

This guide explains how a new user should operate AISkillGrid from first setup to finished work.

The core habit is simple: choose the command that matches the phase, let it create or update artifacts, then use the artifacts to decide the next command.

## First Run

Start by installing the hub into a target project. After installation, initialize the project workflow:

```text
/sdd-init
```

During initialization, decide:

- Ticketing provider: local, GitHub, GitLab, or Jira.
- Artifact store: disk-first, memory-first, or hybrid.
- PRD workflow: default statuses, provider-style statuses, imported statuses, or custom statuses.
- Index refresh policy: initialize and explicitly refresh GitNexus/ccc indexes during `/sdd-init`.
- Optional persistent memory: Engram.
- Skill registry: `.skillgrid/project/SKILL_REGISTRY.md` for compact rules used in subagent prompts (refresh when skills under `.agents/skills/` change; see `docs/13-memory-and-indexing.md`).

The recommended default for most users is a hybrid model: keep reviewable files in the repository and save concise durable memory summaries.

When Engram is enabled, active changes should also have a compact `skillgrid/<change-id>/state` memory entry. It helps a later session recover the phase, blockers, artifact paths, and next action without trusting chat history.

## First Feature

For a new feature, use this path:

```mermaid
flowchart TD
  Start[Start Feature] --> Clear{Is The Goal Clear}
  Clear -->|No| Explore[sdd-explore]
  Clear -->|Yes| Brainstorm[sdd-brainstorm]
  Explore --> Brainstorm
  Brainstorm --> Loop[sdd-loop]
  Loop --> Apply[sdd-apply]
  Apply --> Verify[sdd-verify]
  Verify --> Archive[sdd-archive]
```

Use `/sdd-explore` when the project is greenfield or the topic needs investigation first (web research via `deep-research`, then codebase — no code changes). Use `/sdd-brainstorm` when you want the full change pipeline (clarify, propose, spec, design, PRD, tasks).

## PRD index, hierarchy, and OpenSpec layout

AISkillGrid takes a deliberate multi-artifact, layered specification approach. It does not treat all specifications as one monolithic document type. Instead, it separates product intent (PRDs), architectural decisions (ADRs), project-level context (PROJECT/ARCHITECTURE/STRUCTURE), and implementation specs (OpenSpec slice specs) into distinct, traceable artifacts. This is a conscious design, not an accidental duplication.

## Artifact Responsibility
- ADR:           Technical decision rationale (`.skillgrid/adr/`)
- PRD:           Product intent, user stories (`.skillgrid/prd/`)
- OpenSpec:      Immutable spec, tasks, acceptance criteria (`openspec/changes/`)
- Beads:         Live task execution and tracking (`.beads/`)

## Flow
```
PRD → OpenSpec proposal → approve → openspec-to-beads → bd ready → execute → OpenSpec archive
```

## Linking
- Each Beads issue must have label `openspec:<change-name>` and include spec reference in description.
- ADRs referenced by both OpenSpec proposals and Beads issues where relevant.

Skillgrid uses one mental model for planning and execution (see `skillgrid-prd-artifacts`, `skillgrid-vertical-slices`, `skillgrid-spec-artifacts`):

| Concept | Jira-style | GitHub-style | Where it lives |
|---------|------------|--------------|----------------|
| Milestone / program slice | Epic | Milestone | `.skillgrid/prd/INDEX.md` — dependency-ordered PRD table **and** optional **Execution snapshot** at the top (current phase, active change/slice, discovered work, session notes). |
| Feature initiative | Task | Issue | `.skillgrid/prd/PRD<NN>_<slug>.md` + `openspec/changes/<change-id>/` |
| Shippable unit | Sub-task | Checklist item | Vertical slice — `openspec/changes/<change-id>/tasks.md` **and** `openspec/changes/<change-id>/specs/<vertical-slice-slug>/spec.md` (see `docs/03-skillgrid-logic.md`) |

**OpenSpec per change** (one folder per PRD-style initiative):

```text
openspec/changes/<change-id>/
  proposal.md
  design.md
  tasks.md
  specs/<vertical-slice-slug>
    spec.md   # slice-scoped requirements + checklist
```

Optional umbrella: `openspec/specs/<change-id>/spec.md` for cross-cutting requirements. There is **no** `.skillgrid/project/TASK.md`; use INDEX snapshot + `tasks.md` + slice specs for “where we are” and sub-task detail.

Canonical **blank files** (`template-*.md` under **`.skillgrid/templates/`**) and a consolidated explanation live in **`docs/03-skillgrid-logic.md`**.

## Shared Understanding Before Planning

Do not rush vague intent into a PRD. For ambiguous work, the agent should question the user or domain expert until both sides have the same understanding of the goal, scope, non-goals, risks, and tradeoffs.

Good questioning is direct and incremental:

- ask one meaningful question at a time;
- include a recommended answer when useful;
- record accepted answers as decisions;
- record unknowns as open questions or HITL blockers;
- stop when the remaining ambiguity is low enough to write durable artifacts.

This is HITL work. Do not send product alignment into an unattended Build Loop.

## Working In Slices

AISkillGrid prefers small vertical slices. Each slice should be understandable, implementable, and verifiable.

Every task should carry one enforceable execution label:

- `[AFK]` when the agent can safely proceed with clear instructions and verification.
- `[HITL]` when a human decision is required before work continues.

This distinction is enforced as a workflow gate, not a soft convention:

- unlabeled tasks are not apply-ready and should be routed back to breakdown;
- `[HITL]` tasks should stop unattended execution until a human resolves the blocker;
- only `[AFK]` tasks should enter `/sdd-apply`;
- `/sdd-verify` should fail slices that bypassed labeling or do not record the label reason.

Automation hook:

- run `.skillgrid/scripts/sdd-gate.sh <phase> --change <change-name>` as the hard precheck (`apply`, `verify`, etc.);
- label validation is included inside the gate script — treat gate exit 1 as blocking (`blocked` in apply, `failed` in verify).

Vertical slices should create early feedback. Prefer a thin path through the necessary layers over a horizontal plan where one phase only changes schema, another only changes API, and another only changes UI. A good first slice behaves like a tracer bullet: small enough to build safely, but complete enough to prove direction.

Each apply-ready slice should include:

- acceptance criteria;
- blockers and unblocks;
- `[HITL]` or `[AFK]` reason;
- vertical-slice or horizontal-setup classification, with justification for horizontal setup work;
- relevant files and artifacts;
- context budget or split trigger;
- fresh-agent input list with exact artifact and source/test paths;
- verification command;
- TDD expectation or explicit non-TDD exception.

Treat the PRD and OpenSpec artifacts as the destination: they define what done means. Treat `tasks.md`, issues, handoff, event logs, and checkpoints as the journey: they define how agents move toward done.

### Operational checkpoints

Before apply (after the apply gate), after each `/sdd-loop` iteration, on verify pass, and before archive, the coordinator runs `.skillgrid/scripts/checkpoint-record.sh` so the repo records branch, SHA, and phase-bound evidence. That gives the next agent a git-anchored resume point without relying on chat history.

See **[Checkpoints (Tier 1)](18-checkpoints.md)** for flags, log format, triggers, and dashboard behavior. The journey should be a Kanban/DAG of independently grabbable slices, not only a linear checklist. Record `blockedBy` and `unblocks` relationships so independent work can be grouped into safe dependency waves.

## Smart Zone And Context Rot

Large context windows are useful for retrieval, but coding quality still drops when the session carries too much unrelated or stale context. AISkillGrid treats this as context rot.

Use the smart zone deliberately:

- keep the system prompt and always-loaded rules small;
- move durable state into PRDs, OpenSpec changes, handoff files, events, and research files;
- split large tasks before implementation;
- dispatch fresh subagents with bounded artifact paths instead of pasting chat history;
- review implementation in a fresh context when risk is meaningful.

If a slice needs broad chat memory, whole-repo rereading, or multiple unrelated subsystems, it is too large for AFK apply. Route it back to breakdown.

Context budget gate: before apply, confirm the slice can be executed by a fresh agent using durable artifacts and a bounded file list. If the agent would need broad chat history, whole-repo reading, or multiple unrelated subsystems, split the slice or route back to breakdown.

## Operating Guardrails

Keep the fixed context surface small. Load optional skills, personas, research, and project standards only when the current phase needs them. Use `.skillgrid/project/SKILL_REGISTRY.md` as a compact pull-based index, and push only the relevant standards into reviewer prompts.

Use quality gates before validation and finish:

- no silent scope reduction from PRD/OpenSpec/task intent;
- no schema, migration, or API drift without an updated artifact;
- no missing verification for behavior-changing work;
- no unresolved security, credential, destructive-action, merge, release, or product-signoff risk;
- no stale release docs, README guidance, rules, or project context for shipped behavior.

For risky changes, add a second-opinion review from an independent specialist or model and record overlapping and unique findings. For destructive commands or production-impacting work, warn explicitly and restrict edits to the intended path or stop for `[HITL]`.

## When To Use Explore

Use:

```text
/sdd-explore
```

when the project is brownfield or the agent needs to understand architecture before planning. Exploration should produce project knowledge, not implementation changes.

## When To Use Design

Use:

```text
/sdd-design-ui
```

when the work includes user-facing UI, layout, interaction, visual direction, or product experience decisions.

Design work should create durable direction so later implementation does not depend on memory or taste guesses.

## When To Use Import

There is no dedicated `/skillgrid-import` command in the current workflow surface. Import/normalization should be handled during `/sdd-explore` and `/sdd-brainstorm`.

## During Implementation

Use:

```text
/sdd-loop
```

for controlled implementation from an approved task list.

`/sdd-loop` is the **Ralph loop orchestrator**: one AFK task per invocation (`plan → delegate to /sdd-apply → reflect → stop`). Use it as the default AFK continuation entrypoint. For unattended runs: `.skillgrid/scripts/sdd-ralph-loop.sh <change> [max]`.

Full guide: **[SDD Ralph Loop](10-sdd-ralph-loop.md)**.

Use `/sdd-apply` directly when you want a **single session** to chew through many tasks (sequential subagents, full pipeline) without the per-iteration Ralph controller.

The agent should:

- Read the active PRD.
- Read the technical change artifacts.
- Read the handoff.
- Implement the next task or slice.
- Run focused verification.
- Update state and evidence.
- Stop on unclear scope or HITL blockers.

For behavioral code, implementation should follow TDD:

1. **RED:** write one focused failing test and confirm it fails for the expected reason.
2. **GREEN:** implement the smallest change that makes the test pass.
3. **Refactor:** clean up only after the behavior is proven.

Do not delete, weaken, or bypass tests to get green.

Controlled continuation is handled by `/sdd-loop` in small, verified increments. It is not an excuse for unbounded autonomous work.

```mermaid
flowchart TD
  State[Read Durable State] --> Pick[Pick Next Safe Unit]
  Pick --> Gate{AFK Ready}
  Gate -->|No| Stop[Stop For HITL Or Breakdown]
  Gate -->|Yes| Apply[Apply One Slice]
  Apply --> Evidence[Capture Evidence]
  Evidence --> PersonaGate{Required persona reports}
  PersonaGate --> Update[Update Handoff And Events]
  Update --> Reassess[Reassess Context And Risk]
  Reassess --> Pick
```

## Verification And Review

Use:

```text
/sdd-verify
```

Verification reconciles specs, code review, execution evidence, and sign-off.

This is where AISkillGrid becomes more than a productivity wrapper. It helps users keep the speed of AI while preserving review discipline.

Delegated implementation must pass a double review gate before the parent marks the slice complete:

1. **Spec compliance review:** verify PRD/OpenSpec/task traceability, acceptance criteria, and slice boundaries.
2. **Code quality review:** verify correctness, maintainability, architecture, security, performance, tests, and local conventions.

Execution enforcement when both agent families are configured:

- Use phase executor agents (`sdd-*`) to perform each phase.
- Use Nordic personas for gate/review authority.
- Minimum hard gate: `tyr` on verify, plus `heimdall` for security/release-sensitive slices.

Ordering matters. Do not accept code quality review before spec compliance passes. If either review returns required changes, fix the issue, rerun focused verification, and repeat the same review stage. Critical or important findings block completion until fixed, explicitly accepted with rationale, or converted into follow-up work.

For user-facing behavior, add UAT notes or a manual QA checklist after automated evidence. Failed UAT should create focused fix tasks rather than a vague “try again” loop.

When a phase needs independent judgment, the coordinator dispatches Norse personas listed in that phase's **`sdd-<phase>/SKILL.md`** (*Norse persona invocations*). Each subagent returns one report; the coordinator merges, records decisions in the handoff, and either continues or marks HITL.

Protocol: **`.agents/skills/_shared/sdd-persona-delegation.md`**. On conflict during verify, dispatch extra invocations from **`sdd-verify/SKILL.md`** (e.g. `loki`, `heimdall`).

Durable state:

- reports under `.skillgrid/tasks/research/<change-id>/`;
- handoff in `.skillgrid/tasks/context_<change-id>.md`;
- JSONL events in `.skillgrid/tasks/events/<change-id>.jsonl`.

For the full multi-agent operating model, see `08-multi-agent-work.md`. It covers personas, dependency waves, handoff and event logs, the subagent orchestration skill, parallel branch lanes, and parallelism rules.

CI-ready default progression is verify-first:

```text
/sdd-loop (or /sdd-apply) -> /sdd-verify -> /sdd-archive
```

Do not archive directly after implementation. `sdd-archive` should fail closed if verification artifacts are missing or unresolved critical findings remain.

## Finishing Work

Use:

```text
/sdd-archive
```

when implementation has passed verification.

Finish should handle closure tasks such as:

- Updating final PRD status.
- Archiving or syncing specs.
- Marking completed PRDs and journey artifacts closed or archived so stale docs are not mistaken for current architecture.
- Cleaning up previews or checkpoints when appropriate.
- Preparing git or PR handoff when requested.
- Running explicit post-merge index refresh (`ccc index` and `npx gitnexus analyze`, preserving `--embeddings` mode when already enabled).
- Confirming docs and evidence are not stale.
- Saving final Engram closure/state summaries when memory is available.

If your team intentionally shares Engram memories through git, run:

```bash
engram sync
```

after significant finish work, and use:

```bash
engram sync --import
```

on another machine after cloning. Review `.engram/` before committing because it may contain prompts, decisions, and sensitive project context.

## Resuming Work

When returning after an interruption, inspect durable artifacts and then continue with either:

```text
/sdd-explore <change-name>
```

or:

```text
/sdd-verify
```

The agent should inspect the durable state:

- Project config.
- PRD index.
- Active handoff files.
- Event logs.
- Relevant memory.
- Skill registry.
- Open change artifacts.

Then it should recommend the next command.

If Engram returns memory search hits, the agent should retrieve full observations before relying on them. Search previews are not enough for requirements, blocker state, task status, or user decisions.

### Session handoff helpers

For explicit session-to-session transfer, use the handoff helper scripts:

```bash
.skillgrid/scripts/handoff-create.sh full <slug> [continues-from]
.skillgrid/scripts/handoff-create.sh quick <slug>
.skillgrid/scripts/handoff-resume.sh list
.skillgrid/scripts/handoff-resume.sh [latest|.skillgrid/handoffs/<file>.md] [max-age-days]
.skillgrid/scripts/handoff-validate.sh .skillgrid/handoffs/<file>.md
.skillgrid/scripts/handoff-check-staleness.sh .skillgrid/handoffs/<file>.md [max-age-days]
```

These files are intentionally agent-agnostic and live in `.skillgrid/handoffs/` so another agent/session can resume without replaying full chat history.

Mode contract:

- `create` (`full`) writes a complete transfer-ready handoff.
- `quick` writes a compact interruption handoff.
- `resume` loads a handoff, runs validation and staleness checks, and then continues from explicit `Next steps`.

For active changes, initialize and maintain a compact dispatch registry:

```bash
.skillgrid/scripts/handoff-registry-init.sh <change-id>
```

Use this file as the short index for parent-agent context injection packets to subagents, instead of repeating long report/history reads.

## What To Expect After Each Phase

| Phase | Expected Output |
|---|---|
| Init | Project config and persistence bootstrapping |
| Explore | Project map, architecture notes, and approach analysis |
| Brainstorm | Clarified scope plus generated proposal/spec/design/PRD/tasks |
| Apply | TDD-backed code changes, evidence, updated task progress |
| Verify | Verification report with completeness/correctness/coherence checks |
| Archive | Synced specs and archived completed change |

## Daily Usage Pattern

```mermaid
flowchart LR
  Check[Check State] --> Choose[Choose Command]
  Choose --> Act[Run Phase]
  Act --> Evidence[Capture Evidence]
  Evidence --> Update[Update Artifacts]
  Update --> Reassess[Reassess Next Step]
  Reassess --> Choose
```

The best way to use AISkillGrid is not to memorize every command. It is to trust the phase model. Ask what state the work is in, run the command for that state, and let the artifacts guide the next move.

## Why This Feels Different

Many AI coding tools help with a single task. AISkillGrid helps with the lifecycle around the task.

That is the full-solution advantage:

- New users get a guided path.
- Experienced users get control.
- Teams get consistent process across IDEs.
- Agents get durable context.
- Reviewers get evidence.
- Work can resume without rebuilding the story from chat.

This is how AI-assisted development becomes repeatable engineering practice.
