---
name: sdd-loop
id: sdd-loop
category: Workflow
description: Controlled build loop for AFK-safe slices with evidence capture and reassessment
agent: odin
subtask: true
---

You are the SDD coordinator loop controller (Odin-primary). Run a bounded Build Loop for the active change and stop deterministically when a gate requires it.

CONTEXT:
- Working directory: !`echo -n "$(pwd)"`
- Current project: !`echo -n "$(basename $(pwd))"`
- Change name (optional): $ARGUMENTS
- Artifact store mode: hybrid

TASK:
Execute one-or-more safe loop iterations using this contract:
1. Pick one AFK-safe slice.
2. Execute implementation.
3. Capture verification evidence.
4. Reassess risk and blockers.
5. Continue or stop.

MANDATORY PRECHECK (fail closed):
- Locate active change artifacts and ensure required inputs exist:
  - `openspec/changes/{change-name}/specs/**/spec.md` (or equivalent active spec path)
  - `openspec/changes/{change-name}/design.md`
  - `openspec/changes/{change-name}/tasks.md`
- Run:
  `.skillgrid/scripts/validate-task-labels.sh openspec/changes/{change-name}/tasks.md`
- If validation fails, return `status: blocked` with validator output and `next_recommended` set to fix labels.
- If required artifacts are missing, return `status: failed` with missing paths listed.

LOOP SAFETY RULES:
- Pick exactly one AFK-ready slice per iteration.
- Never execute `[HITL]` tasks in unattended mode.
- Stop immediately on unresolved HITL blockers.
- After each iteration, reassess before deciding to continue.
- Do not run unbounded autonomy; each continuation decision must be explicit.

EXECUTION MODEL:
- Delegate implementation to `/sdd-apply` for the selected slice.
- Capture evidence from implementation outputs (tests/build/checks/review notes).
- If reassessment identifies architectural ambiguity, security concern, spec conflict, or go/no-go uncertainty, invoke `/sdd-persona-board --preset <architecture|security|ux-content|go-no-go-release|risk-acceptance>` before proceeding.
- If `/sdd-persona-board` (or `/sdd-board` alias) returns conflict or critical findings, set `status: blocked` and require HITL.

HARD BLOCK SEMANTICS:
- `tyr` critical finding => block progression.
- `heimdall` critical finding => block progression.
- unresolved persona conflict => block progression.
- missing gate output/evidence => block progression.

FILESYSTEM PERSISTENCE:
- Read `.agents/skills/_shared/skillgrid-handoff.md` and apply:
  - handoff updates in `.skillgrid/tasks/context_<change-id>.md`
  - event logging in `.skillgrid/tasks/events/<change-id>.jsonl`
  - slice summary updates after each iteration

ENFORCEMENT CONTRACT:
- Canonical enforcement is centralized in `.agents/skills/_shared/sdd-enforcement-contract.md`.
- This workflow MUST apply:
  - phase routing and stop conditions
  - mandatory skill-gate checks
  - two-stage review gate
  - standard return envelope

RETURN FORMAT:
- `status`: `completed | blocked | failed`
- `executive_summary`:
  - `overview`
  - `used_tokens` (`input`, `output`, `total`, or `not_available`)
- `artifacts`: loop evidence paths, updated handoff path, event log path, optional board report paths
- `next_recommended`: deterministic next safe action (`continue loop`, `run /sdd-board`, `resolve HITL`, `fix task labels`, `run /sdd-verify`)
- `risks`: open/accepted risks or explicit `none`
