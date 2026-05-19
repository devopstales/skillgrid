---
description: Implement SDD tasks — writes code following specs and design
---

You are an SDD sub-agent. Read the skill file at `.agents/skills/sdd-apply/SKILL.md` FIRST, then follow its instructions exactly.

The sdd-apply skill supports TDD workflow (RED-GREEN-REFACTOR cycle). Write a failing test first, then implement the minimum code to pass, then refactor.

CONTEXT:
- Working directory: !`echo -n "$(pwd)"`
- Current project: !`echo -n "$(basename $(pwd))"`
- Artifact store mode: hybrid

TASK:
Implement tasks for the active SDD change.

**Direct `/sdd-apply`:** Remaining incomplete tasks until done or blocked.

**From `/sdd-loop`:** Implement **only** the single task line in the delegation prompt; then return.

GATE:
- Before any implementation, run:
  `.skillgrid/scripts/sdd-gate.sh apply --change {change-name}`
- If this script exits non-zero, stop immediately and return:
  ```
  status: "blocked"
  executive_summary: "Gate failure — see sdd-gate errors above"
  next_recommended: "Fix gate failures in tasks.md or artifacts before proceeding"
  ```
- Do NOT attempt manual label checks or artifact checks — the gate script is the single source of truth.

VDD — DECOMPOSE (before implementation):
- Read `.agents/skills/vdd-decompose/SKILL.md`
- If `openspec/changes/{change-name}/tasks.md` exists but has not been decomposed to beads:
  - Read the change artifacts (proposal, tasks, design) to understand scope
  - Decompose complex tasks into atomic beads (Epics → Issues → Sub-issues) with dependency chains
  - Run proactive gap detection (missing tests, rollbacks, monitoring)
  - Store beads at `.beads/{change-name}-beads.json`

ENGRAM PERSISTENCE (artifact store mode: engram):
CRITICAL: mem_search returns 300-char PREVIEWS, not full content. You MUST call mem_get_observation(id) for EVERY artifact.
STEP A — SEARCH (get IDs only):
  mem_search(query: "sdd/{change-name}/spec", project: "{project}") → save spec_id
  mem_search(query: "sdd/{change-name}/design", project: "{project}") → save design_id
  mem_search(query: "sdd/{change-name}/tasks", project: "{project}") → save tasks_id
STEP B — RETRIEVE FULL CONTENT (mandatory):
  mem_get_observation(id: spec_id) → full spec
  mem_get_observation(id: design_id) → full design
  mem_get_observation(id: tasks_id) → full tasks (keep tasks_id for updates)
Update tasks as you complete them:
  mem_update(id: {tasks-observation-id}, content: "{updated tasks with [x] marks}")
Save progress:
  mem_save(title: "sdd/{change-name}/apply-progress", topic_key: "sdd/{change-name}/apply-progress", type: "architecture", project: "{project}", content: "{progress report}")
FILESYSTEM PERSISTENCE:
  Reade .agents/skills/_shared/skillgrid-handoff.md for filesystem persistence instructions.

For each task:
1. **Read `.skillgrid/project/CONTEXT.md`** if it exists. Note any relevant glossary terms, assumptions, or success criteria before proceeding.
2. Read the relevant spec scenarios (acceptance criteria)
3. Read the design decisions (technical approach)
4. Read existing code patterns in the project
5. Write the code (write failing test first, then implement, then refactor)
6. **Execute the TDD skill** (`.agents/skills/skillgrid-tdd/SKILL.md`) – follow its Red‑Green‑Refactor loop exactly.
7. Mark the task as complete [x]

ENFORCEMENT CONTRACT:
- Canonical enforcement is centralized in `.agents/skills/_shared/sdd-enforcement-contract.md`.
- This workflow MUST apply that shared contract for:
  - phase routing and stop conditions
  - mandatory skill-gate checks
  - two-stage review gate
  - standard return envelope
- Apply-specific emphasis:
  - enforce task-label validation before implementation starts
  - enforce slice-level acceptance-criteria presence before coding
  - any critical finding must block progression and produce deterministic remediation in `next_recommended`

Return the standard SDD envelope per `.agents/skills/_shared/sdd-return-envelope.md`.

---

## OpenSpec CLI supplement (integrated from former opsx-apply)

Implement tasks from an OpenSpec change.

**Input**: Optionally specify a change name (e.g., `/sdd-apply add-auth`). If omitted, check if it can be inferred from conversation context. If vague or ambiguous you MUST prompt for available changes.

**Steps**

1. **Select the change**

   If a name is provided, use it. Otherwise:
   - Infer from conversation context if the user mentioned a change
   - Auto-select if only one active change exists
   - If ambiguous, run `openspec list --json` to get available changes and use the **AskUserQuestion tool** to let the user select

   Always announce: "Using change: <name>" and how to override (e.g., `/sdd-apply <other>`).

2. **Check status to understand the schema**
   ```bash
   openspec status --change "<name>" --json
   ```
   Parse the JSON to understand:
   - `schemaName`: The workflow being used (e.g., "spec-driven")
   - Which artifact contains the tasks (typically "tasks" for spec-driven, check status for others)

3. **Get apply instructions**

   ```bash
   openspec instructions apply --change "<name>" --json
   ```

   This returns:
   - Context file paths (varies by schema)
   - Progress (total, complete, remaining)
   - Task list with status
   - Dynamic instruction based on current state

   **Handle states:**
   - If `state: "blocked"` (missing artifacts): show message, suggest using `/sdd-brainstorm`
   - If `state: "all_done"`: congratulate, suggest archive
   - Otherwise: proceed to implementation

4. **Read context files**

   Read the files listed in `contextFiles` from the apply instructions output.
   The files depend on the schema being used:
   - **spec-driven**: proposal, specs, design, tasks
   - Other schemas: follow the contextFiles from CLI output

5. **Show current progress**

   Display:
   - Schema being used
   - Progress: "N/M tasks complete"
   - Remaining tasks overview
   - Dynamic instruction from CLI

6. **Implement tasks (loop until done or blocked)**

   For each pending task:
   - Show which task is being worked on
   - Make the code changes required
   - Keep changes minimal and focused
   - Mark task complete in the tasks file: `- [ ]` → `- [x]`
   - Continue to next task

   **Pause if:**
   - Task is unclear → ask for clarification
   - Implementation reveals a design issue → suggest updating artifacts
   - Error or blocker encountered → report and wait for guidance
   - User interrupts

7. **On completion or pause, show status**

   Display:
   - Tasks completed this session
   - Overall progress: "N/M tasks complete"
   - If all done: suggest archive
   - If paused: explain why and wait for guidance

**Output During Implementation**

```
## Implementing: <change-name> (schema: <schema-name>)

Working on task 3/7: <task description>
[...implementation happening...]
✓ Task complete

Working on task 4/7: <task description>
[...implementation happening...]
✓ Task complete
```

**Output On Completion**

```
## Implementation Complete

**Change:** <change-name>
**Schema:** <schema-name>
**Progress:** 7/7 tasks complete ✓

### Completed This Session
- [x] Task 1
- [x] Task 2
...

All tasks complete! You can archive this change with `/sdd-archive`.
```

**Output On Pause (Issue Encountered)**

```
## Implementation Paused

**Change:** <change-name>
**Schema:** <schema-name>
**Progress:** 4/7 tasks complete

### Issue Encountered
<description of the issue>

**Options:**
1. <option 1>
2. <option 2>
3. Other approach

What would you like to do?
```

**Guardrails**
- Keep going through tasks until done or blocked
- Always read context files before starting (from the apply instructions output)
- If task is ambiguous, pause and ask before implementing
- If implementation reveals issues, pause and suggest artifact updates
- Keep code changes minimal and scoped to each task
- Update task checkbox immediately after completing each task
- Pause on errors, blockers, or unclear requirements - don't guess
- Use contextFiles from CLI output, don't assume specific file names

**Fluid Workflow Integration**

This skill supports the "actions on a change" model:

- **Can be invoked anytime**: Before all artifacts are done (if tasks exist), after partial implementation, interleaved with other actions
- **Allows artifact updates**: If implementation reveals design issues, suggest updating artifacts - not phase-locked, work fluidly

