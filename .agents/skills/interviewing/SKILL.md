---
name: interviewing
description: Interview the user relentlessly about a plan, decision, or idea until you reach a shared understanding, while maintaining the project's domain model (glossary + ADRs). Use when brainstorming needs to sharpen a design, or when the user wants to stress-test their thinking.
# based on mattpocock-skills:grilling
---

# Interviewing

Interview the user relentlessly until you reach a shared understanding. Map the
problem as a **design tree**: every decision branches into the decisions that
hang off it.

**This skill drives `skillgrid:architectural-decision-records` underneath it** (the same
relationship as Matt Pocock's `grill-with-docs`). The interview is where terms
crystallize and decisions are made — so the paper trail is written *here*, in
the conversation, not batched at the end:

- **Glossary:** the moment a domain term is resolved, write it to the glossary
  (`conventions.glossary`, default `.skillgrid/glossary/`) — challenge conflicts,
  sharpen fuzzy language, cross-reference the code. A glossary that only changes
  after the interview is a summary, not the session's output.
- **ADRs:** the moment a decision clears the bar (hard to reverse + surprising
  without context + real trade-off), offer to record it in `.skillgrid/adr/`. The
  "why" is freshest right now; a future reader needs exactly this moment's
  reasoning.

If `conventions.glossary` / `conventions.adr` are set in `.skillgrid/config.yaml`,
use those paths. Both files are created lazily on first use.

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

**Round state:** the glossary and ADRs capture the *output*, but the in-flight
position — which round you are on, which questions are settled, which are
still open — lives in conversation only. A crashed interview loses it.

- When the topic has a spec dir, keep `state.md` current: after each round,
  update `Open:` (the unsettled frontier) and `Decisions:` (what settled this
  round), and bump the round count. It is the in-flight index of the glossary,
  not a duplicate of it — the glossary still gets every term the moment it
  resolves.
- If `mnemonic.enabled: true`, mirror the transition with
  `mem_save(topic_key: skillgrid/<topic>/state, ...)` (upsert).
- On resume, read `state.md` first (skillgrid:resume) — recompute the frontier
  from the settled list, then continue with the next round.

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
