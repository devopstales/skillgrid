---
name: questioning
description: "Stress-test a plan, decision, or idea branch by branch before implementation, using a design tree, frontier, and rounds with recommendations. Use when you need to clarify intent, the orchestrator delegates a clarification round (explore/propose/init), or a request must be classified before design."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# questioning

## Purpose

`questioning` is the reusable **questioning / intent-resolution primitive** for Skillgrid SDD. It stress-tests a plan, decision, or idea *before* anyone writes code, by mapping the subject as a design tree and interrogating it branch by branch until nothing is left silently assumed.

It merges two proven techniques:

- **Design tree + frontier + rounds**: ask the whole frontier per round, ship every question with a recommendation, and separate *facts* (agent's job) from *decisions* (user's job).
- **Classify + approval gate**: classify the request by complexity, enforce a hard "no implementation until approved" gate, and propose 2–3 approaches with trade-offs.

It is cross-cutting, not a pipeline phase. Phases invoke it:

- `sdd-init` — to clarify project facts / tracker choice
- `sdd-explore` — to resolve ambiguous requirements before reading code
- `sdd-propose` (Step 0 shaping) — to resolve business rules before writing the intent

## Hard Gate

**Do not invoke any implementation skill, write code, scaffold a project, or take implementation action until you have stated your intent and the user has approved it.** This applies to every path. The approval gate never scales down — a two-sentence design still needs a yes.

## The Core Model

Three ideas carry the technique.

A **design tree** is the model of the subject: each decision branches into the decisions that hang off it.
The **frontier** is the set of decisions whose prerequisites are all settled: the only questions that can honestly be asked *now*.
A **round** is one frontier, asked in full and answered in full.

Inside a round, every question ships in a fixed shape so the user can answer by number:

```
Q1 — **<title>**: <body, may be multiple paragraphs, may include choices>

  Recommendation: <your recommended answer>
```

Rules: ask the whole frontier; never put two questions that depend on each other in the same round; end only when the user confirms **shared understanding**, never when questions run out.

## Classify First

Announce the classification out loud — "this looks **bounded**, so I'll present a short design in chat" — so the user can override:

- **Spike** — feasibility question ("can we…", "is it possible…"). Output is an answer, not kept code. Present the question + probe (2–3 sentences), get a nod, investigate cheaply. Label anything built as throwaway.
- **Bounded** — a well-scoped change to code that already exists: a new flag, a small endpoint, a one-file fix. The flow you're changing is already here to read. Ask the clarifying questions that matter, present a short design **in chat**, STOP, wait for approval. No spec file.
- **Architectural** — new projects, new subsystems, interface-altering restructuring. Full process: questions → 2–3 approaches → sectioned design → written spec → planning.

When in doubt, take the heavier path. Complexity discovered mid-task upgrades the path — stop, say so, step up. Nothing downgrades mid-task.

## Facts vs Decisions

- **Decisions are the user's** — put each to them and wait. Answering your own decisions is a bug.
- **Facts are your job.** When a frontier question needs something the environment settles (filesystem, git, tools), find it yourself — do not ask.
  - In Skillgrid, use the **code-index ladder**: `code_status` → `code_index` (if stale) → `code_search` → `code_read`.
  - If a fact needs non-trivial research, dispatch a sub-agent. Do not block the round: questions *downstream* of running research wait; the rest of the frontier asks now.

## Workflow

1. **Classify** the request, state it out loud, and confirm the subject + scope boundary (in/out).
2. **Explore context** — recover project facts (`mem_search("sdd-init/{project}")` → `mem_get_observation`), check `docs/skillgrid/config.yaml` and the archived changes in `docs/skillgrid/archive/`, run the code-index ladder for repo facts. For multi-subsystem requests, flag if scope needs decomposition first.
3. **Ask clarifying questions** in rounds by frontier. Each question gets a recommendation.
4. **(Architectural only)** Propose 2–3 approaches with trade-offs; lead with your recommendation; apply YAGNI ruthlessly.
5. **Present design** — scale sections to complexity (a few sentences for bounded; 200–300 words per section for architectural). Cover: architecture, components, data flow, error handling, testing. Ask after each section.
6. **Confirmation gate** — when the frontier is empty, STOP. Do not build until the user confirms shared understanding.
7. **(Architectural only) Spec self-review** — placeholder scan, internal consistency, scope check, ambiguity check. Fix inline.

## Round Format

```
Subject: {plan / decision / idea being clarified}
Classification: {spike | bounded | architectural} — say why
Scope: {what's in / out}

---

Round 1 — frontier is {N} decisions:

Q1 — **<title>**: <body, may include choices>

  Recommendation: <recommended answer>

---

Q2 — **<title>**: <body>

  Recommendation: <recommended answer>

---

Reply by number (e.g. "1 yes, 2 the second option, 3 no — here's why").
```

After the user answers, recompute the frontier and ask the next round.

## Worked Example

Input: *"We should let customers save checkout drafts so they can come back later."*

```
Subject: Persist checkout drafts for later completion
Classification: architectural — new subsystem (draft storage, lifecycle, conflict resolution)
Scope: in — save/recover draft at any step; out — saved-cart sync, scheduled reminders

---

Round 1 — frontier is 2 decisions:

Q1 — **Where does draft state live?**: local storage + backend, or backend-only?
  Recommendation: backend-only. Local-only loses drafts on device switch and blocks recovery flows.

Q2 — **When is a draft discarded?**: on completed checkout, on timeout, or manual delete only?
  Recommendation: keep on completed checkout (re-entry), discard via explicit timeout (30 days), manual delete always.

---

Reply by number.
```

User answers `1 backend-only, 2 discard on timeout only`. Frontier advances: storage choice unblocks *conflict resolution*; discard rule does not. Round 2 asks only the unblocked branch.

## Red Flags (don't skip these gates)

| Thought | Reality |
|---|---|
| "Too simple to need a design" | Simple means a short design, not no design. Present it, then stop. |
| "I'll start while they read" | The gate is approval, not design length. Present, then wait. |
| "I understand this kind of app, so it's bounded" | Bounded measures the repo, not your familiarity. No existing flow to change → architectural. |
| "The spike works, so I'll keep the code" | A spike's output is an answer. Keeping the code is a new request — classify it. |

## Persistence (SDD / Mnemonic mode)

If the orchestrator launches you with a change name:

```
Mnemonic topic: sdd/{NNN-slug}/grill
Filesystem:     docs/skillgrid/changes/{NNN-slug}/interview.md
Mode:            hybrid (default)
```

- Start once: `sid = skillgrid-mnemonic_mem_session_start(title: "sdd/{change-name}/grill")`.
- After each round, append decisions + questions to `interview.md` and `mem_save` the transcript (upsert via `topic_key`).
- Recovery: `mem_search(query: "sdd/{change-name}/grill")` → `mem_get_observation(id)` for full content. Never rely on search previews.
- At session end: `mem_session_summary` then `mem_session_end`.

In `none` mode, return the transcript inline only.

## Gotchas

- **The frontier is judgment, not a computed graph.** You may put two questions in one round then discover one should have changed the other — say so and reopen that branch next round.
- **Recommendation arguing against the question** — when your recommendation disputes how the question is framed, the user should answer the *recommendation*. Say so when it happens.
- **No answer caps.** If a session runs very long, scope is the real cause — break it up and question the pieces.
- **Don't answer your own decisions.** Under a "resolve-this-ticket" frame, the task reads as license to keep moving. Decisions stay the user's.
- **Don't act on agreement without confirmation.** The session is not done when questions run out — it finishes when the user confirms shared understanding.
- **One question at a time is supported for bounded work**, but batch the frontier for deeper sessions. Don't let the format choice mask a skipped gate.
- **Mnemonic `mem_search` returns 300-char previews** — always `mem_get_observation(id)` before relying on a prior transcript.
