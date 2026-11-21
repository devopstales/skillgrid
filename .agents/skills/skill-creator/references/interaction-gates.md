# Interaction Gates (EGM format)

How the skill (and the skills it creates) talk to the user. This file is the
source of truth for user-interaction structure; SKILL.md cites it.

## Interaction Invariants (mandatory)

All interactive gates (structured question tools such as `ask_question`, if
your harness has one, or plain chat) MUST:

1. contain **strictly one question** (`len(questions) == 1`). Multi-question
   payloads are forbidden.
2. be the **sole tool call in the turn**.
3. provide **clear, action-oriented options** without `(Recommended)`
   prefixes.
4. yield execution immediately upon invocation.

### Open-ended vs Single-Select

Two gate types, never mixed:

- **Open-ended questioning (plain text)** — use for exploration: problem
  drilling, constraints, edge cases, file paths, free-form input. Ask the
  question directly in chat. **Do NOT** use a structured question tool and do
  **NOT** supply pre-canned multiple-choice options while exploring.
- **Single-select gates (structured, `is_multi_select: false`)** — use only
  for a decision between a small set of fixed options: readiness
  confirmation, approval of an artifact, preset/mode selection, handoff
  choice.

### Artifact-First Delivery

Large artifacts (proposals, drafts, briefs, plans, reviews) are written to a
file with the file-writing tool; chat gets the path plus a 2-3 sentence
summary of what changed and its key decisions. The file is the deliverable;
the chat is the notification.

### The Gate Turn (strict generation order)

1. **Visible chat summary FIRST** — conversational text stating what was
   done (which section/artifact was written or updated), 2-3 key decisions
   or findings (one line each), and the path/link to the artifact.
2. **The gate SECOND** — the single-select question, in the same turn.

Never emit a question tool call with empty response text (no empty chat
bubbles). The summary precedes the gate.

### Option Handling

- **Option 0** is the primary forward action (approve / proceed / confirm).
- **Later options** refine, loop back, or end: request revisions, clarify in
  chat, or conclude.
- **Empty write-in guard:** if the user selects "request revisions" (or an
  option needing input) with no write-in text and no inline comments, do not
  loop or fail. Prompt in chat for the specific input, then yield the turn.
- **Approve with inline comments:** apply the comments to the artifact first,
  then advance.
- **Loop back:** stay in the current step, incorporate the input, and
  re-present the same gate when ready.

### Canonical Payloads

Readiness gate (end of open-ended drilling):

```json
{
  "questions": [
    {
      "question": "Anchored hypothesis: [2-3 sentence summary]. How would you like to proceed?",
      "options": [
        "Confirm and proceed to [next step]",
        "Clarify additional constraints or assumptions in chat"
      ],
      "is_multi_select": false
    }
  ]
}
```

Section/approval gate (after each artifact section):

```json
{
  "questions": [
    {
      "question": "Drafted [Section Name] in [artifact]. Please review. How would you like to proceed?",
      "options": [
        "Approve [Section Name] and proceed",
        "Request revisions to [Section Name]"
      ],
      "is_multi_select": false
    }
  ]
}
```

Handoff gate (end of workflow):

```json
{
  "questions": [
    {
      "question": "Workflow complete. [artifact] is finalized. Select next step:",
      "options": [
        "Option A",
        "Option B",
        "Conclude and keep [artifact] for later"
      ],
      "is_multi_select": false
    }
  ]
}
```

## Mermaid Workflow Diagrams

Skills that carry multi-step workflows or branches MUST include a Mermaid
graph in SKILL.md (a `references/` file if SKILL.md stays over it). Rules:

- `flowchart TD` by default; `LR` only when the graph is wide and shallow.
- Every interactive gate is a **diamond** node; every option is a labeled
  edge, and loop-back edges are explicit (`|loop back|`).
- Nodes that are "write artifact" steps are labeled with the artifact
  filename.
- Keep the diagram a map of the same steps the prose defines, one node per
  step or gate — not a second description. If the prose and the diagram
  disagree, the diagram is stale; fix it in the same edit.
- Place the diagram immediately after the overview/intro, before the step
  definitions, so the agent sees the topology before walking the steps.

Example (skill creation flow with gates):

```mermaid
flowchart TD
    A[Capture intent + interview] --> B[Phase 1: Trigger]
    B --> C[Phase 2: Structure - scripts first]
    C --> D[Phase 3: Steering]
    D --> E[Phase 4: Pruning]
    E --> F[Draft SKILL.md]
    F --> G{Readiness gate<br/>confirm draft?}
    G --|confirm| H[Write test cases]
    G --|clarify in chat| A
    H --> I[10-rule audit]
    I --> J{All PASS?}
    J --|FAIL| F
    J --|PASS| K{Handoff gate<br/>next step?}
    K --|iterate| A
    K --|done| L[Skill ships]
```
