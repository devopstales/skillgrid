---
name: questioning
description: "Grill the programmer relentlessly until you share a model of the problem. Use when a brainstorm request is ambiguous, when a design rests on unresolved assumptions, or when the user wants to be stress-tested on a plan, design, briefing, or spec."
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
  based_on: skillgrid:brainstorm (questioning reference extracted) + grill-me 1.0 (merged)
  created_by: human
  related_skills: [brainstorm, blueprint]
  version: "1.1"
---

# Questioning

## Overview

Map the problem as a **design tree** and work its frontier in rounds until you share a model of the problem. The glossary and ADRs are written *during* the interview, not backfilled after.

This skill is the interview loop, extracted from `skillgrid:brainstorm` as a
standalone concern. `brainstorm` classifies the path and drives the design;
this skill owns the questioning structure — the design tree, the frontier
rounds, the clarity gate, and the paper trail.

**Announce at start:** "I'm using the skillgrid:questioning skill to work the
design tree."

**Position in the chain:** `skillgrid` → `brainstorm` → **`questioning`** → `research` → …
(per `_shared/rules/sdd-structure.md` § Phase Order). This skill runs inline
inside `brainstorm`. It must complete (clarity gate passed, frontier empty)
before `research` or `prototype` begin, and before `brainstorm` hands off to
`blueprint`.

## When to Use

- When a brainstorm request is ambiguous or the programmer wants to stress-test
  their thinking.
- When the user wants to be stress-tested on a plan, design, briefing, or spec —
  an existing design that needs every branch of the decision tree resolved
  before implementation.
- When a design rests on unresolved assumptions that need to be put to the
  programmer as decisions, not questions.
- When the frontier (the set of open questions that block progress) is
  non-empty and needs structured work.

**Skip it** when the request is fully specified and unambiguous — the clarity
gate is the exit, not the entry; don't interview for the sake of it. Also skip
it when the user asked a single factual question, or the design is already
approved and only implementation remains. Grilling is for the *design*
decision, not for the *build* decision.

**Config:** Read `.skillgrid/config.yaml` before starting. Use
`conventions.specs_root` (default `.skillgrid/change-specs/`) for the topic
folder. If `mnemonic.enabled: true`, mirror transitions with
`mem_save(topic_key: skillgrid/<topic>/briefing, ...)` (upsert).

## The Design Tree

The design tree is the structure of the problem, not a to-do list. Each node is
either:

- **A fact** — you look it up (code, docs, a quick read). Facts are not
  questions; they are answers you go get.
- **A decision** — the programmer must choose. Decisions carry a trade-off
  and a recommendation.
- **A constraint** — a boundary the design must respect. Constraints are
  recorded, not resolved.

The **frontier** is the set of open decision nodes whose prerequisites are
all settled. You work the frontier one round at a time.

## Rounds

**One round at a time.** Ask the whole frontier, wait for answers, recompute,
next round. Don't ask questions whose prerequisites are still open.

### Traversal modes

Two modes for working the tree, chosen by the situation:

- **Frontier rounds (batch)** — the default for brainstorm interviews. Work
  the whole frontier in one round: one gate turn per open decision (one
  structured tool call per decision — the invariants forbid multi-question
  payloads), and the programmer may answer in any order.
- **One at a time (depth-first)** — when the user wants to be stress-tested
  on a *specific* plan, design, briefing, or spec. Walk the tree depth-first:
  the highest-leverage decision first (the one whose answer changes the most
  downstream work), then its dependencies, then the leaves. One question at a
  time, each with a recommended answer — the user accepts, overrules, or asks
  for more detail. A branch is resolved when the user has picked an option
  and the pick is recorded in the running summary. Stop when every branch is
  resolved or the user says "stop".

If a question can be answered by exploring the codebase or the project
wiki, explore first and state the finding; ask the user only for the
branch the code cannot settle (intent, trade-off, scope).

### Round structure

1. **Recompute the frontier.** After each round of answers, re-derive which
   nodes are now open. A node becomes open when all its parent decisions are
   settled.
2. **Present the frontier.** Number each open decision. For each, state:
   - the question (what decision is needed)
   - the trade-off (what each option costs)
   - your recommended answer (with the reasoning in one line)
3. **Wait for answers.** Each open decision is one gate turn (one structured
   tool call — see Rules); batch mode is several such gate turns in one
   round, not one payload. The programmer may answer in any order.
4. **Record the answers.** For each answered decision:
   - Write the resolved term to the terms file
     (`.skillgrid/projectwiki/glossary/01-business-terms.md` for
     domain/product/workflow, `02-technical-terms.md` for
     architecture/platform) — challenge conflicts, sharpen fuzzy language,
     cross-reference the code.
   - If a decision clears the ADR bar (hard to reverse + surprising without
     context + real trade-off), offer to record it as
     `.skillgrid/projectwiki/decisions/NNNN-slug.md` and add a pointer row to
     the `## LOCKED` ADR index in
     `.skillgrid/projectwiki/ASSUMPTIONS.md`.
5. **Commit** the glossary and ADR changes after each round (spec zone). The
   committed glossary + ADRs ARE the in-flight index.

## Exit: Clarity Gate

Before declaring done, the clarity gate must pass:

- **Every resolved term** is in the glossary.
- **Every decision that cleared the ADR bar** was either recorded or
  explicitly declined.
- **The frontier is empty** — no open decision nodes with all prerequisites
  settled.
- **Clarity score ≤ 0.20** with all dimensions at or above their minimums.
  A dimension below its minimum is marked ⚠ and treated as an assumption in
  the Clarity Report.

A glossary untouched by a long interview means the ADR half got skipped;
go back and backfill, or say why there was nothing to record.

Record the final scores in the design brief's **Clarity Report** section.

### Output: the resolved decision tree

The output of the interview is the resolved decision tree — one line per
decision: what was decided, what was rejected and why, and what it costs if
wrong. This tree is the input to `skillgrid:brainstorm` (if the plan is new)
or to the `briefing.md`'s interview log (if the plan exists).

## Rules

- **One round at a time.** Ask the whole frontier, wait for answers,
  recompute, next round. Don't ask questions whose prerequisites are still
  open.
- **Harness-native structured tool first; the EGM gate format is the
  contract.** Decision questions go through the harness's native
  structured-question tool — `question` in opencode, `AskUserQuestion` in
  Claude Code, the equivalent in other agents — never a pre-canned
  multiple-choice list typed into chat, and never an open-ended question
  that is really a decision among fixed options. The format follows
  `skill-creator/references/interaction-gates.md` (cited, not restated):
  strictly **one question per tool call** (multi-question payloads are
  forbidden — a frontier of N decisions is N sequential gate turns, not one
  payload); the question call is the **sole tool call in the turn**;
  options are **clear and action-oriented, without `(Recommended)`
  prefixes**; the turn **yields immediately** on invocation. Gate-turn
  order is fixed: the **visible chat summary first** (which decision this
  is, the trade-off, your recommended answer with one line of reasoning),
  then the gate, same turn — never an empty chat bubble before the tool
  call. Free-form exploration (problem drilling, constraints, edge cases)
  stays plain-text chat — no structured tool while exploring.
- **Maintain the domain model as you go.** As terms resolve, write them to
  the glossary inline. As a load-bearing decision clears the ADR bar, offer
  to record it. The interview ends with the glossary and any ADRs already
  reflecting what was decided — not a to-do to backfill.
- **Facts are looked up, not asked.** If a question has an answer in the
  codebase or the docs, go get it. Never ask the programmer a fact you could
  have looked up yourself.
- **Decisions carry a recommendation.** Every open decision you present
  comes with your recommended answer and the reasoning. The programmer
  approves or overrides; they don't design from scratch.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The programmer knows their domain, I'll just ask open-ended questions" | Open-ended without a frontier is unfocused. The design tree gives the questions structure and the frontier gives them priority. |
| "I'll ask everything at once to save time" | Batch mode is fine, but the frontier must be computed first. Asking a question whose prerequisite is still open wastes the round. |
| "The glossary can be filled in after the interview" | The glossary is the paper trail. A glossary that only changes after the interview is a summary, not the session's output. |
| "This decision is too small for an ADR" | Run the three-part test (hard to reverse, surprising without context, real trade-off). If any is true, offer it. The programmer declines; you don't skip. |
| "The clarity gate is optional if we're on schedule" | The gate is the exit condition, not a checkpoint. An interview without a passed gate is incomplete. |

## Red Flags

- One round after another with no new frontier — recompute or the frontier is
  empty and you haven't.
- The glossary is untouched by a long interview: the domain model half was
  skipped.
- Declaring "done" when the frontier is empty but the clarity gate hasn't
  been scored.
- A decision recorded as made that the programmer never actually put to.
- A decision asked in plain chat when the harness has a structured tool — or
  a multi-question payload, or options carrying `(Recommended)` prefixes:
  the gate format in `interaction-gates.md` is the contract, and the native
  tool is the delivery mechanism.
- The clarity gate passed on a self-score while a dimension was under 8/10
  confidence in the programmer's own confirmation.

## Verification

- [ ] The design tree was mapped before the first question was asked
- [ ] Every round presented the full frontier with recommendations
- [ ] Every resolved term is in the glossary (committed after each round)
- [ ] Every ADR-cleared decision was recorded or explicitly declined
- [ ] The clarity gate passed: clarity ≤ 0.20, all dimensions ≥ minimums,
  frontier empty
- [ ] The programmer was never asked for a fact you could have looked up
  yourself
- [ ] Every decision gate used the harness's native structured-question tool
  in EGM gate format: one question per call, sole tool call in the turn,
  action-oriented options without `(Recommended)` prefixes, chat summary
  before the gate in the same turn, and the turn yielded immediately
- [ ] When the bridge was used: each posted question is a `type: decision`
  observation (JSON `content` + `state`), shared to `team`, with a stable
  `topic_key`; answered questions were read back via `mem_search` and folded
  into the tree (or the programmer confirmed a chat-only interview)
