# Writing Rules, EGM Gates, and Mermaid Diagrams

## 5b. Writing Rules

Rules beyond what `_shared/craft/skill-anatomy.md` covers:

1. **Script-first.** Never write a SKILL.md line that references a script you
   haven't built and tested first.
2. **One contract per script.** One job, one exit-code contract, one stdout
   format. If a script does two things, split it.
3. **The skill line is the interface.** The agent sees command + exit-code
   meanings, not the script's internals. If the line needs to explain
   internals, the script is doing too much.
4. **Judgment stays prose, and stays short.** 1-3 sentences. Longer means it's
   a deterministic procedure in disguise — re-run the classification test.
5. **No English if-statements.** "If missing, error. If stale, drift. If
   current, continue." is three branches → one script with three exit codes.

### Red Flags

- A skill line > 2 sentences describing a procedure (not a judgment)
- "If X, do Y. If Z, do W." in prose (deterministic branching → script)
- References a script path that doesn't exist yet
- A line explaining *why* the script does what it does (internals leaked)
- A judgment paragraph > 3 sentences (procedure in disguise)
- No scripts where the process is clearly algorithmic (parse, filter, count)
- Copying a procedure instead of referencing the script that already does it

### Common Rationalizations

If you (or the user) push back on extracting a script, keeping the skill
small, or skipping the spine sections, load `references/rationalizations.md`
— the rationalizations and the reality for each.

---

## 5c. User Interaction (EGM Gates)

User interactions in skills follow the EGM (Elephant-Goldfish Model)
interaction format — the same structure the skillgrid EGM skills
(`brainstorm`, `blueprint`) and the shared Goldfish persona briefs in
`.agents/agents/goldfish.md` use. The full protocol, including canonical
payloads, is `references/interaction-gates.md`; this section is the contract
summary.

### Interaction Invariants (mandatory)

All interactive gates (structured question tools such as `ask_question`, if
your harness has one, or plain chat) MUST:

1. contain **strictly one question** (`len(questions) == 1`) — multi-question
   payloads are forbidden;
2. be the **sole tool call in the turn**;
3. provide **clear, action-oriented options** without `(Recommended)`
   prefixes;
4. yield execution immediately upon invocation.

### Open-Ended vs Single-Select

- **Open-ended (plain text)** — exploration: problem drilling, constraints,
  edge cases, file paths, free-form input. Ask in chat. No structured
  question tool, no pre-canned multiple-choice options while exploring.
- **Single-select (structured, `is_multi_select: false`)** — decisions among a
  small set of fixed options: readiness confirmation, section approval,
  preset/mode selection, handoff choice.

### The Gate Turn (strict generation order)

1. **Visible chat summary FIRST** — state what was written or updated,
   2-3 key decisions or findings (one line each), and the path to the
   artifact.
2. **The gate SECOND** — the single-select question, same turn.

Never emit a question tool call with empty response text (no empty chat
bubbles).

### Option Handling

- **Option 0** is the primary forward action (approve / proceed / confirm).
- **Later options** refine, loop back, or end (request revisions, clarify in
  chat, conclude).
- **Empty write-in guard:** "request revisions" with no write-in text and no
  inline comments → prompt in chat for the specific input, then yield. Do not
  loop or fail.
- **Approve with inline comments:** apply the comments to the artifact first,
  then advance.
- **Artifact-first:** large artifacts (proposals, drafts, briefs, plans,
  reviews) go to a file; chat gets the path plus the summary, never the full
  text.

When authoring or editing a skill that has interactive steps, give it gates in
this format. Load `references/interaction-gates.md` for the canonical
readiness / approval / handoff payloads.

---

## 5d. Mermaid Workflow Diagrams

A skill with multi-step workflows or branches carries a Mermaid graph of its
workflow in SKILL.md (in a `references/` file if that keeps SKILL.md small).
Rules:

- `flowchart TD` by default; `LR` only for wide, shallow graphs.
- Every interactive gate is a **diamond** node; every option is a labeled
  edge, and loop-back edges are explicit (`|loop back|`).
- "Write artifact" nodes are labeled with the artifact filename.
- The diagram maps the same steps the prose defines, one node per step or
  gate. If prose and diagram disagree, the diagram is stale: fix it in the
  same edit.
- Place the diagram after the overview/intro, before the step definitions, so
  the agent sees the topology before walking the steps.

Example (this skill's own flow, §1-§7):

```mermaid
flowchart TD
    A[Capture intent + interview] --> B[Phase 1: Trigger<br/>invocation mode]
    B --> C[Phase 2: Structure<br/>classify steps, scripts first]
    C --> D[Phase 3: Steering<br/>leading words, legwork]
    D --> E[Phase 4: Pruning<br/>deletion test, token audit]
    E --> F[Draft SKILL.md<br/>with gates + mermaid map]
    F --> G{Readiness gate<br/>confirm draft?}
    G --|confirm| H[Write evals<br/>evals/evals.json]
    G --|clarify in chat| A
    H --> I[10-rule audit]
    I --> J{All PASS?}
    J --|FAIL: fix| F
    J --|PASS| K[Run + evaluate<br/>skill vs no-skill]
    K --> L{User satisfied?}
    L --|iterate: feedback loop| A
    L --|done| M[Skill ships]
```

---

**Decision gates:**

| Condition | Action |
|---|---|
| SKILL.md > 500 lines, or a reference chains behind another | Split / flatten into `references/` — one concern per file, one level deep. |
| A step is deterministic | It gets a tested, executable script with a documented exit-code contract (0/1/2) — not prose. |
| A skill needs user input at a gate | EGM format: one question, the sole tool call, no "(Recommended)" prefix. |
| A multi-step workflow where order matters | Mermaid diagram with gate diamonds, labeled option edges, explicit loop-backs. |
| The TDD gate is skipped (no failing test before the skill edit) | RED first — a skill is code that shapes agent behavior. |
| `skill-size-budget.mjs` exits non-zero | Not shippable — prune or re-tier before the audit runs. |
| Audit findings (any FAIL) | Fix, re-run the audit — findings are reported as a table (Rule / Status / Finding / Fix). |

