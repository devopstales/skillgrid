
# Event Log Rule

For any identified Skillgrid change id, create `.skillgrid/tasks/events/` if needed and append short JSONL events to `.skillgrid/tasks/events/<change-id>.jsonl` when this command starts, completes, blocks, skips, dispatches/receives subagents, or changes workflow state. If a delegated agent cannot write, require it to return a suggested event object and append that event from the parent session before advancing.

## Critical Patterns

### Canonical Paths

```text
.skillgrid/tasks/context_<change-id>.md
.skillgrid/tasks/events/<change-id>.jsonl
.skillgrid/tasks/registry_<change-id>.md
.skillgrid/tasks/research/<change-id>/
```

The handoff file is the rolling state for one change. The event log is the append-only workflow timeline. The compact registry is the short recent index for dispatch-time context injection. The research directory holds long reports, scrapes, browser evidence, or subagent output.

Treat the handoff as the current-state file for the change. It should answer: where are we, what is blocked, what evidence exists, and what should happen next.

### Session Handoff Modes

Use explicit mode names whenever handoff work is requested by any `sdd-*` flow:

- `create`: full handoff snapshot for session transfer or pause.
- `quick`: minimal handoff snapshot for short interruptions.
- `resume`: reload handoff state, run staleness check, then continue from next steps.

Mode guidance:

- Use `create` after major milestones, architecture decisions, broad edits, or before ending a long session.
- Use `quick` for short pauses where full context restatement is unnecessary.
- Use `resume` at session start when a handoff file exists or when continuing interrupted work.

### Handoff Contents

Keep `context_<change-id>.md` concise and skimmable. Copy the canonical blank from **`.skillgrid/templates/template-handoff-context.md`**. Planning logic: **`docs/03-skillgrid-logic.md`**.

Do not turn the handoff into a raw transcript. Link to research files when details are long.

Required sections in every active handoff context file:

- `Goal`
- `Current state`
- `Active artifacts`
- `Decisions`
- `Failed approaches` (what was tried and should not be repeated)
- `HITL blockers`
- `AFK-ready work`
- `Dependency waves`
- `Research index`
- `Last checkpoint` (synced by `checkpoint-record.sh` — see `skillgrid-checkpoints`)
- `Next steps` (ordered immediate continuation actions)

### Operational checkpoints (Tier 1)

Record safe resume points with:

```bash
.skillgrid/scripts/checkpoint-record.sh --change <change-id> --name <label> --trigger <trigger> --phase <phase> --evidence "<summary>"
```

**Mandatory triggers:** `before-apply` (after apply gate, before code), `after-loop` (each `/sdd-loop` reflect), `verify-pass` (PASS verdict), `pre-archive` (before archive ops), `handoff-create` (handoff create mode).

Writes: `.skillgrid/tasks/checkpoints.log`, updates `## Last checkpoint` in `context_<change-id>.md`, appends `node: checkpoint` to `events/<change-id>.jsonl`.

Canonical schema boundaries:

- **State now** lives in `.skillgrid/tasks/context_<change-id>.md`.
- **Timeline** lives in `.skillgrid/tasks/events/<change-id>.jsonl`.
- **Compact recent index** lives in `.skillgrid/tasks/registry_<change-id>.md`.
- **Deep evidence** lives in `.skillgrid/tasks/research/<change-id>/`.

### Compact Registry Discipline

Maintain `.skillgrid/tasks/registry_<change-id>.md` as a short dispatch index:

- latest decisions (with reasons and source paths);
- latest progress and status by slice/task;
- active blockers;
- safe next dispatch candidates;
- seed fields for context injection packets.

The registry should stay concise and represent only recent, high-signal state. Do not duplicate full reports; link to handoff/event/research artifacts.

### Engram State Alignment

When Engram is available, mirror only the compact current-state index to `skillgrid/<change-id>/state`. The handoff remains the in-repo current-state file; the Engram snapshot helps a fresh session find the right files and last known phase after compaction.

Update the snapshot whenever the handoff phase, status, blocker list, active artifacts, or next recommended action changes. Do not copy the full handoff into Engram unless the project explicitly runs in `artifactStore.mode: engram`.

### Required Handoff Event Log

Append workflow events to:

```text
.skillgrid/tasks/events/<change-id>.jsonl
```

Create `.skillgrid/tasks/events/` before the first append if it does not exist. Every Skillgrid command, parent session, and write-capable Skillgrid subagent must append an event when it starts work, completes a phase, blocks, skips a phase, or changes workflow state. If a subagent is read-only or cannot write, it must return a suggested event object and the parent must append it before advancing.

Each line is one JSON object. Keep entries short and stable so the local dashboard can render them:

```json
{"time":"2026-04-28T08:00:00Z","changeId":"planning-dashboard","prd":"PRD02_dashboard.md","node":"apply","phase":"apply","status":"started","summary":"Started AFK slice 2","artifacts":[".skillgrid/tasks/context_planning-dashboard.md"]}
{"time":"2026-04-28T08:12:00Z","changeId":"planning-dashboard","node":"apply","phase":"apply","status":"blocked","summary":"Missing HITL design approval","blocker":"Approve option B preview","artifacts":[".skillgrid/preview/planning-dashboard-options.html"]}
```

Supported fields:

- `time`: ISO timestamp.
- `changeId`: OpenSpec/Skillgrid change id.
- `prd`: optional PRD filename.
- `node`: workflow node or command id, for example `plan`, `apply`, `test`, `validate`.
- `phase`: Skillgrid phase.
- `status`: `started`, `progress`, `passed`, `failed`, `blocked`, `completed`, or `skipped`.
- `summary`: one-line event summary.
- `blocker`: optional HITL or technical blocker.
- `artifacts`: optional paths to PRDs, handoff files, previews, reports, checks, or research.
- `external`: optional issue key or URL.
- `agent` / `subagent`: optional subagent name for the dashboard Subagents view.
- `role`: optional role such as `implementer`, `spec-reviewer`, `code-quality-reviewer`, `design-critic`, or `security-auditor`.
- `task`: optional short delegated task label.
- `output`: optional primary output path when it is not already listed in `artifacts`.

Do not replace `context_<change-id>.md` with events. The event log is the timeline; the handoff remains the current-state summary.

### Enforcement Contracts

Apply enforcement to all `sdd-*` phases handled by `sdd-orchestrator` using:

- `skills/_shared/sdd-enforcement-contract.md`
- `skills/_shared/sdd-return-envelope.md`
- `skills/_shared/sdd-label-gate-contract.md`

This handoff document remains focused on state, timeline, and evidence tracking. Enforcement semantics are centralized in the shared contracts above.

### Persona dispatch records

When Norse persona subagents run for a phase (see `sdd-<phase>/SKILL.md` and `sdd-persona-delegation.md`), the coordinator records merged outcomes in the handoff before advancing:

```markdown
## Persona merge: <phase> — <change-id>

Personas invoked (persona → capability):
Report paths:
Accepted findings / decision:
Rejected options:
Conflicts:
HITL required: yes/no
Artifacts updated:
Next safe action:
```

Persona events use the same JSONL log with `node: "persona"`, `status: "dispatched" | "completed" | "blocked"`, `persona`, `capability`, `output`, `findings_severity`, and `hitlRequired` when known. Personas advise; the coordinator, user, PRD, and OpenSpec artifacts remain authoritative.

### Slice Completion Summary

After every completed vertical slice, delegated task, review loop, or blocked apply attempt, update the handoff with a compact reassessment:

```markdown
## Last slice summary

- **Slice/task:** <identifier or title>
- **Result:** completed | blocked | needs-review | reverted
- **Evidence:** <test command, preview, checkpoint, review report, or research file>
- **Changed assumptions:** <none, or what changed since planning>
- **Blockers:** <none, or HITL/technical blocker>
- **Next recommended action:** <next command or slice>
```

This replaces long transcript-style progress logs. Keep details in research files and link them from the summary.

### Research Spill Files

Use `.skillgrid/tasks/research/<change-id>/` for:

- external web research
- documentation extracts
- browser testing reports
- subagent reports
- large comparison tables
- security or performance findings that exceed a short summary

Each research file should include:

- topic
- date or “as of” note when relevant
- sources or local files inspected
- findings
- recommendation
- open questions

Template:

```markdown
# Research: <topic>

- **Change:** `<change-id>`
- **Date:** <YYYY-MM-DD>
- **Question:** <decision this research informs>
- **Method:** repo search | web search | docs lookup | browser test | subagent review

## Sources

- `<local path>` — <why inspected>
- [Source title](https://example.com) — accessed <date>

## Findings

- <Evidence-backed finding>

## Recommendation

<What should the parent session do next?>

## Open questions

- <Question and owner, or `None`>

## Handoff update

Add to `.skillgrid/tasks/context_<change-id>.md`:

- `<this file>` — <one-line finding>
```

### Subagent Contract

Every subagent prompt for a Skillgrid change should include:

- the handoff path
- the event log path
- the compact registry path
- the PRD path
- the OpenSpec change path when present
- the expected output file under `research/<change-id>/`
- a requirement to append a short event when write-capable, or return a suggested event object when read-only
- a requirement to return a short summary with file paths

Parent session must pass an explicit **context injection packet** (no raw transcript dump):

- objective (one sentence);
- constraints and non-goals;
- exact task/slice id;
- owned files/edit boundaries;
- required artifacts to read (ordered list of paths);
- expected output path and return format;
- verification command with expected pass condition.

### Sequencing Gate

Before dispatching multiple subagents, answer:

**Will any agent need to read another agent's output before it can produce correct work?**

- If **yes**, dispatch sequentially in dependency order.
- If **no**, parallel dispatch is allowed only when file ownership is non-overlapping and merge verification is planned.

### Retry Policy For Missing/Inconsistent Artifacts

When expected artifacts are missing, empty, or inconsistent with return claims, use this bounded retry ladder before continuing:

1. **Attempt 1 (clarify):** resend the same task with explicit missing artifact list, exact output path, and required return format.
2. **Attempt 2 (tighten):** reduce scope to one artifact/output and restate ownership boundaries and verification command.
3. **Attempt 3 (route):** switch reviewer/implementer persona or model tier and request the same bounded deliverable.
4. **Escalate:** if still failing, mark workflow `blocked`, append event log with failure details, and require HITL decision.

Required checks on each retry:

- artifact path exists;
- artifact is non-empty;
- artifact is referenced in handoff/event log;
- summary claims match produced files.

Do not run unbounded retries. Maximum automated retries per missing/inconsistent artifact set: **3**.

After a subagent returns, the parent session must read the handoff and cited research files before editing product code or changing workflow state.

### Parent Session Rule

The parent session owns implementation decisions. Subagents gather evidence, critique, test, or draft bounded artifacts. The parent reconciles their output against PRD and OpenSpec artifacts.

## Commands

```bash
ls .skillgrid/tasks
ls .skillgrid/tasks/research/<change-id>
```