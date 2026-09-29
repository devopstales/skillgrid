---
name: interviewing
description: Interview the user relentlessly about a plan, decision, or idea until you reach a shared understanding, while maintaining the project's domain model (glossary + ADRs). Use when brainstorming needs to sharpen a design, or when the user wants to stress-test their thinking.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: mattpocock-skills:grilling
---

# Interviewing

**Announce at start:** "I'm using the skillgrid:interviewing skill to grill this until we share a model."

## Overview

Grill the user relentlessly until you and the user share a model of the
problem. Mapping the problem as a **design tree** and working its frontier in
rounds is how vague intent becomes a decision with a paper trail — the glossary
and ADRs are written *during* the interview, not backfilled after.

## When to Use

- When a request is ambiguous and you must clarify before acting.
- At the start of a non-trivial task with underspecified requirements.
- When the user wants to stress-test their own thinking on a design.

**When NOT to use:** when the request is fully specified and unambiguous — the
clarity gate is the exit, not the entry; don't interview for the sake of it.

Interview the user relentlessly until you reach a shared understanding. Map the
problem as a **design tree**: every decision branches into the decisions that
hang off it.

**This skill drives `skillgrid:architectural-decision-records` underneath it** (the same
relationship as Matt Pocock's `grill-with-docs`). The interview is where terms
crystallize and decisions are made — so the paper trail is written *here*, in
the conversation, not batched at the end:

- **Terms:** the moment a domain term is resolved, write it to the terms file
  (`conventions.artifacts`, default `.skillgrid/artifacts/01-business-terms.md`
  for domain/product/workflow, `02-technical-terms.md` for architecture/platform)
  — challenge conflicts, sharpen fuzzy language, cross-reference the code. A
  terms file that only changes after the interview is a summary, not the
  session's output.
- **ADRs:** the moment a decision clears the bar (hard to reverse + surprising
  without context + real trade-off), offer to record it as a `### ADR-NNNN`
  entry in `.skillgrid/ASSUMPTIONS.md` (§ LOCKED) and add a row to the
  `### In-force set` table. The "why" is freshest right now; a future reader
  needs exactly this moment's reasoning.

If `conventions.artifacts` is set in `.skillgrid/config.yaml`, use that path
for the terms files. The terms files and the ADR entries are created lazily on
first use.

Work the tree in **rounds**. The **frontier** is every decision whose
prerequisites are already settled — the questions you can ask _now_ without
guessing at answers you haven't heard yet. Ask the whole frontier in one round:
number each question and give your recommended answer. Then wait for the
user's answers before the next round.

Format a round like so:

```
❓ **Q1** — **<question title>**: <question body, may include multiple choices>

➡️ <your recommended answer>

---

❓ **Q2** — **<question title>**: <question body>

➡️ <your recommended answer>
```

Each round the user's answers reshape the tree: settled decisions push the
frontier outward and unblock questions that depended on them. Recompute the
frontier and ask the next round. A question whose answer depends on another
question still open in this round belongs to a _later_ round, not this one.

**Post each question to the decision bridge** (so the dashboard's Decisions
view is the live inbox — see `docs/user-guide/10-decision-companion.md`). For
every frontier question you put to the user, also `mem_save` it as a governed
`type: decision` observation, then widen it so the team-scoped dashboard reader
sees it (the frozen `mem_save` tool has no visibility argument, so `mem_share`
after saving):

```
mem_save(
  type: "decision",
  title: "<question title>",
  topic_key: "interview/<slug>/<question-id>",   # <slug>=the interview, <question-id>=this Q (stable)
  content: {
    "question": "<question body>",
    "options": [ { "id": "a", "label": "<choice>" }, { "id": "b", "label": "<choice>" } ],
    "recommended": "<id of your recommended answer>",
    "state": "pending"
  })
mem_share(id, target: "team")
```

Rules:

- **`content` is JSON, not markdown** — `mem_save` normally takes markdown, but
  a decision's `content` MUST be the structured payload above (the dashboard
  parses it; markdown or a missing `state` is not an interview decision and is
  excluded from the inbox).
- **`recommended` is your `➡️` answer**, encoded as an option `id` — the UI
  highlights it. Free-text questions still get options (the stated choices, or
  the realistic ones you'd put to them); never a decision with zero options.
- **`topic_key` is stable per question** — re-saving the same key updates the
  question in place (upsert + version), which is how you revise a recommendation
  mid-interview.
- **`visual` (optional):** when a question is clearer shown than told and the
  user has accepted the visual companion, write throwaway HTML to
  `.skillgrid/prototype/<topic>/<variant>.html` and add `"visual": "<topic>/<variant>.html"`
  to `content` — the card renders it in a sandboxed iframe.
- **Read the answer back by polling.** After the user answers in the dashboard
  (or in chat), poll `mem_search` on a word of the question; when the returned
  `content.state` is `"answered"`, fold `answeredOption` / `answerNote` into the
  tree and clear the question. A chat answer the user never posts to the
  dashboard: record it by re-saving the same `topic_key` with
  `state: "answered"` + the chosen option so the durable trail matches.
- **Skip persistence** when the interview is purely conversational and the user
  has no dashboard — the bridge is a convenience view, not a requirement for the
  clarity gate.

**Round state:** the glossary and ADRs capture the *output*, but the in-flight
position — which round you are on, which questions are settled, which are
still open — lives in conversation only. A crashed interview loses it. The
decision bridge is the durable exception: every posted question is a governed
`type: decision` observation with a version history, so the open/settled
position is recoverable from the store, not just the transcript.

- When the topic has a spec dir, **commit the glossary and ADR changes after
  each round** (spec zone). The committed glossary + ADRs ARE the in-flight
  index — the glossary gets every term the moment it resolves, and the ADRs
  capture each settled decision. No separate `state.md` is maintained.
- If `mnemonic.enabled: true`, mirror the transition with
  `mem_save(topic_key: skillgrid/<topic>/briefing, ...)` (upsert).
- On resume (skillgrid:resume), read the glossary + ADRs **and the decision
  bridge** to recompute the frontier: `GET /mnemonic/decisions?state=pending`
  (or `mem_search` on the interview slug) lists what is still open — `pending`
  rows are the unsettled frontier, `answered` rows are settled with their
  recorded choice. What is in the glossary is settled; what is not is open.

## Rules

- **Finding facts is your job, never the user's.** When a frontier question
  needs a fact from the environment (filesystem, codebase, tools), look it up
  yourself (or dispatch a subagent to find it). Don't ask the user for anything
  you could find out. Don't block on it: a running lookup is an unsettled
  prerequisite, so only the questions downstream of it wait; ask the rest of
  the frontier now.
- **The decisions are the user's.** Put each decision to them and wait. Never
  decide for them.
- **One round at a time.** Ask the whole frontier, wait for answers, recompute,
  next round. Don't ask questions whose prerequisites are still open.
- **Maintain the domain model as you go (skillgrid:architectural-decision-records).** As terms
  resolve, write them to the glossary inline. As a load-bearing decision
  clears the ADR bar, offer to record it. The interview ends with the glossary
  and any ADRs already reflecting what was decided — not a to-do to backfill.

## Exit: clarity gate

The interview is **not** done when the frontier is empty. It's done when the
frontier is empty **AND** the clarity gate passes.

Score each dimension 0.0 (completely unclear) to 1.0 (crystal clear):

| Dimension           | Weight | Minimum | What it measures                              |
|---------------------|--------|---------|-----------------------------------------------|
| Goal Clarity        | 35%    | 0.75    | Is the outcome specific and measurable?       |
| Boundary Clarity    | 25%    | 0.70    | What's in scope vs out of scope?              |
| Constraint Clarity  | 20%    | 0.65    | Performance, compatibility, data requirements?|
| Acceptance Criteria | 20%    | 0.70    | How do we know it's done?                     |

**Clarity score** = 1.0 − (0.35×goal + 0.25×boundary + 0.20×constraint + 0.20×acceptance)

**Gate:** clarity ≤ 0.20 AND all dimensions ≥ their minimums → shared
understanding reached.

If the gate fails, identify which dimension is lowest, ask targeted questions
to raise it, and re-score. Repeat until the gate passes.

**Domain-model completeness (exit check):** before declaring done, confirm the
paper trail is current — every resolved term is in the glossary, and every
decision that cleared the ADR bar was either recorded or explicitly declined.
A glossary untouched by a long interview means the architectural-decision-records half got
skipped; go back and backfill, or say why there was nothing to record.

Record the final scores in the design brief's **Clarity Report** section.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I can infer the intent" | The decisions are the user's. Put each one to them and wait — inference is a guess, not a decision. |
| "One question is enough" | The frontier is every decision whose prerequisites are settled. Ask the whole frontier; one question leaves the rest unblocked-by-nothing. |
| "The user will just say 'you decide'" | If they do, that *is* the answer — record it. But you may not decide for them first; the ask is the decision point. |
| "I'll look it up later, after the answers" | Finding facts is your job. A running lookup is an unsettled prerequisite — only the questions downstream of it wait, the rest of the frontier goes out now. |

## Red Flags

- A question asked whose prerequisite is still open in the same round.
- The user is asked to provide a fact you could have looked up yourself.
- One round after another with no new frontier — recompute or the frontier is empty and you haven't.
- The glossary is untouched by a long interview: the domain model half was skipped.
- Declaring "done" when the frontier is empty but the clarity gate hasn't been scored.
- A decision recorded as made that the user never actually put to.
- A posted "decision" whose `content` is markdown or has no `state` — that is a
  decision *record*, not an interview decision; the inbox excludes it (the
  dashboard filters on the payload, not the `type` column).
- A decision saved but not `mem_share`d to `team` — `mem_save` defaults to
  `private`, and the dashboard reader is team-scoped, so a private decision is
  never listed.

## Verification

- [ ] The clarity gate passed — clarity ≤ 0.20 AND every dimension ≥ its minimum, with final scores recorded in the Clarity Report.
- [ ] Each question targeted a real decision point with a recommended answer, not a yes/no rubber-stamp.
- [ ] No frontier question was asked before its prerequisites were settled.
- [ ] The user was never asked for a fact you could have looked up yourself.
- [ ] The paper trail is current: every resolved term is in the glossary, and every ADR-cleared decision was recorded or explicitly declined.
- [ ] When the bridge was used: each posted question is a `type: decision` observation (JSON `content` + `state`), shared to `team`, with a stable `topic_key`; answered questions were read back via `mem_search` and folded into the tree (or the user confirmed a chat-only interview).
