---
name: using-skillgrid
description: Use when starting any conversation - establishes how to find and use skills, requiring skill invocation before ANY response including clarifying questions
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: superpowers:using-superpowers
---

<SUBAGENT-STOP>
If you were dispatched as a subagent to execute a specific task, ignore this skill.
</SUBAGENT-STOP>

<EXTREMELY-IMPORTANT>
If you think there is even a 1% chance a skill might apply to what you are doing, you ABSOLUTELY MUST invoke the skill.

IF A SKILL APPLIES TO YOUR TASK, YOU DO NOT HAVE A CHOICE. YOU MUST USE IT.

This is not negotiable. You cannot rationalize your way out of this.
</EXTREMELY-IMPORTANT>

## Overview

The router — the skill that tells an agent which skill to use before any response or action. If even a 1% chance a skill applies, you must invoke it; there is no choice, no rationalizing out. It runs the checks that gate the start (config, resume, domain model), routes the task to the right first skill, and keeps orchestration from turning routing into extra work.

## When to Use

- At the start of any skillgrid task, to pick the right skill.
- When unsure which skill applies to the current request.

**When NOT to use:** mid-phase, once the correct skill is already running — using-skillgrid routes at the start, it doesn't re-route every step.

## The Rule

**Invoke relevant or requested skills BEFORE any response or action** — including clarifying questions, exploring the codebase, or checking files. If it turns out wrong for the situation, you don't have to use it.

**Before entering plan mode:** if you haven't already brainstormed, invoke the skillgrid:brainstorming skill first.

**Config check:** if `.skillgrid/config.yaml` doesn't exist, invoke the skillgrid:onboarding skill first. All other skills read from this config for project-specific settings.

**Resume check:** if the newest `.skillgrid/specs/` directory contains in-flight artifacts (an uncompleted `tasks.md`, a `checkpoint.json` with non-empty `remaining`, or a ledger under `.skillgrid/sdd/`), invoke the skillgrid:resume skill BEFORE any other skill — including clarifying questions. The files say where you left off; the conversation doesn't. A folder that has moved to `.skillgrid/archive/` is **closed**, not in-flight — do not resume it (a new change gets a new dated folder).

**Domain model check:** if `.skillgrid/glossary/` exists, read `business.md` and `technical.md` for the project's vocabulary, and read the ADRs in `.skillgrid/adr/` for the area you're touching. Use that vocabulary in questions and designs, and respect ADRs that already settled decisions. When terms resolve or a hard-to-reverse decision is made, update them via skillgrid:architectural-decision-records.

Then announce "Using [skill] to [purpose]" and follow the skill exactly. If it has a checklist, create a todo per item.

## Context Discipline

Conversation memory does not survive compaction, and no skill may assume it will.

- **Persist before pressure, not after loss.** When a session is clearly long — many subagents dispatched, large tool outputs accumulating, or the harness showing a compaction indicator — STOP and: (1) append the current position to the plan's ledger (during execution) or the spec-zone artifacts (during planning), (2) commit the work unit (skillgrid:work-unit-commits) so `checkpoint.json` carries the position, (3) if the remaining work is large, run the save path in skillgrid:resume.
- **State writes are dual:** the in-repo file is the source of truth; when `mnemonic.enabled: true`, `mem_save` mirrors it under `skillgrid/<topic>/…` as an index and backup. If they disagree, the in-repo file wins.
- **After compaction** ("FIRST ACTION REQUIRED"), re-orient via skillgrid:resume before continuing — files first, mnemonic only as fallback.

## Router

| Condition | First skill | Then |
|---|---|---|
| Uninitialized (no `.skillgrid/config.yaml`) | `onboarding` | stop for validation |
| Initialized + change | `brainstorming` → `writing-blueprints` → `slicing` | `ticketing` → `subagent-execution` → `qa` → `requesting-code-review` → `ship` → `reflect` |
| Initialized + trivial/small (fast-track) | `writing-blueprints` (light) → `slicing` (light) | same tail (light `ship` + light `reflect`); waiver recorded in `briefing.md` |
| Q&A / lookup | *(no pipeline)* | `mnemonic` / code-index / `research` |
| Spike-only | `spike` | promote to blueprint if user keeps findings |
| Mid-change | Resume from `tasks.md` state | `subagent-execution` for unblocked work |

**Detection — initialized?**

**Uninitialized** when `.skillgrid/config.yaml` does not exist.
**Initialized** when it does. Skill-registry / glossary / ADRs are **not** init signals.

**Pre-blueprint gates:** hard research (external API, costly re-explore) → `research`. Taste/UI/unknown shape → `sketch` **before** locking `blueprint.md`.

## User Gate (mandatory)

After `slicing` writes `tasks.md`:

1. **Implement** → `subagent-execution` (or `simple-execution`)
2. **Revise** → `brainstorming` and/or `writing-blueprints`

Do not auto-execute. The user must confirm the slice before execution begins.

## Resume

| `tasks.md` state | Action |
|---|---|
| missing / no `tasks.md` | `brainstorming` → `writing-blueprints` → `slicing` |
| `tasks.md` exists, no tickets started | user gate → `subagent-execution` |
| Tickets in-progress | resume `subagent-execution` at first incomplete ticket |
| All tickets done, no `qa-report.md` | `qa` |
| `qa-report.md` PASS, no review | `requesting-code-review` |
| Review clean | `receiving-code-review` → `ship` |
| Folder moved to `archive/`, no `report.md` | `reflect` (terminal) |
| `report.md` present in `archive/` | none — cycle complete |

## Skill Priority

When multiple skills apply, process skills come first — they set the approach, then implementation skills (e.g. skillgrid:test-driven-development) carry it out. skillgrid:brainstorming and skillgrid:structured-debugging are skillgrid's most common process skills, but the rule holds for any of them.

- "Let's build X" → skillgrid:brainstorming first, then implementation skills.
- "Fix this bug" → skillgrid:structured-debugging first, then domain skills.

## Orchestration Anti-Patterns

The router is a **persona**, not a skill that calls other skills. These failure modes mean STOP — you've turned routing into extra work:

| Anti-pattern | Why it's wrong | Do this instead |
|---|---|---|
| **Router persona** — the main agent keeps routing *and* does the work itself, so every task re-runs the full pipeline for trivial work. | The router should dispatch, not execute. Doing both doubles context and slows the obvious cases. | Route to the skill, then **step back**. For trivial work, fast-track (the Router's light path), don't run the whole pipeline. |
| **Persona calls persona** — a skill (e.g. `ship`) invokes another *router-level* skill as if it were a tool, nesting orchestration. | Routers set approach; they are not called by other routers. Nesting them creates re-entry loops and duplicated decisions. | Routers call **implementation** skills only. `ship` may fan out to `parallel-code-review` (implementation) but never re-enters `using-skillgrid` or `brainstorming`. |
| **Depth > 1** — a subagent spawns a subagent that spawns a subagent. | Each hop re-reads context and adds a coordination surface; beyond one level the cost exceeds the benefit. | **Max dispatch depth is 1.** The orchestrator dispatches workers; workers do not dispatch. |
| **Paraphrasing hop** — the orchestrator re-summarizes a subagent's result before passing it on, losing fidelity. | The worker's output is already structured; re-narrating it adds loss and tokens. | Pass the worker's **Return Envelope** through verbatim; synthesize only at the final decision (e.g. `ship`'s GO/NO-GO). |
| **Orchestrator that edits** — the coordinating agent goes back and edits code between dispatches. | Mixing coordination with edits breaks the clean "dispatch → collect → decide" loop and muddies who owns the change. | The orchestrator collects verdicts and renders decisions; it edits only to wire up what workers produced. |

Depth rule of thumb: **orchestrator → worker** is the only sanctioned hop. Anything deeper is a signal the task should be sliced differently, not nested further.

**Choosing a pattern — walk this before dispatching:**

```
Is the work one perspective on one artifact?
├── Yes → Direct invocation (a single skill). Stop.
└── No  → Will the same composition repeat?
         ├── No  → Ad-hoc, single skill. Stop.
         └── Yes → Are the sub-tasks independent?
                  ├── No  → Sequential pipeline, user-driven between steps.
                  └── Yes → Parallel fan-out with a merge step in the main context.
                           If the merge would not fit in the main context, fall back to a single skill.
```

**Verdict vs investigation — pick the right primitive:**

| | **Fan-out (subagents)** | **Teams (teammates that message each other)** |
|---|---|---|
| Sub-agents see | The same diff, different lenses | A shared task list *and* each other's findings |
| Output | Independent reports → one merge (a **verdict** on a known artifact) | Adversarial debate → **consensus** (an **investigation** among competing hypotheses) |
| Use for | `parallel-code-review`, `ship`'s GO/NO-GO | `structured-debugging` when a single agent would pick the first plausible theory and stop |

A fan-out produces a *verdict on something you already have*; a team produces a *root cause among things you don't yet know*. Do not wrap a team-style investigation in a fan-out — subagents can't challenge each other, and the debate is what makes it work.

## Red Flags

These thoughts mean STOP—you're rationalizing:

| Thought | Reality |
|---------|---------|
| "This is just a simple question" | Questions are tasks. Check for skills. |
| "I need more context first" | Skill check comes BEFORE clarifying questions. |
| "Let me explore the codebase first" | Skills tell you HOW to explore. Check first. |
| "I can check git/files quickly" | Files lack conversation context. Check for skills. |
| "Let me gather information first" | Skills tell you HOW to gather information. |
| "This doesn't need a formal skill" | If a skill exists, use it. |
| "I remember this skill" | Skills evolve. Read current version. |
| "This doesn't count as a task" | Action = task. Check for skills. |
| "The skill is overkill" | Simple things become complex. Use it. |
| "I'll just do this one thing first" | Check BEFORE doing anything. |
| "This feels productive" | Undisciplined action wastes time. Skills prevent this. |
| "I know what that means" | Knowing the concept ≠ using the skill. Invoke it. |

## Platform Adaptation

If your harness appears here, read its reference file for special instructions:

- Codex: `references/codex-tools.md`
- Pi: `references/pi-tools.md`
- Antigravity: `references/antigravity-tools.md`
- Hermes Agent: `references/hermes-tools.md`

## User Instructions

User instructions (CLAUDE.md, AGENTS.md, GEMINI.md, etc, direct requests) take precedence over skills, which in turn override default behavior. Only skip skill workflows or instructions when your human partner has explicitly told you to.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll just pick a skill by name, skip the router" | The Router table maps condition → first skill. Picking by name misses the gates (config, resume, domain model) that run before routing. |
| "The user gate is optional, I'll proceed" | The User Gate is mandatory: the user confirms the slice before execution begins. Do not auto-execute. |
| "I'll run two skills at once to be safe" | Skill Priority: process skills come first, then implementation. Running two at once re-enters routing and nests orchestration. |
| "The router can just do the work itself" | The router is a persona that dispatches, not executes. Doing both doubles context and re-runs the full pipeline for trivial work. |
| "I remember this skill, no need to read it" | Skills evolve. The Rule requires invoking the current version, not the one in memory. |

## Verification

- [ ] The Router selected exactly one first skill for the task (a `Router` row matches, no guessing by name).
- [ ] The required start gates passed: config check (`onboarding` if uninitialized), resume check (in-flight artifacts), and domain model check (glossary/ADRs read).
- [ ] The User Gate (mandatory) was passed — the user confirmed the slice before execution began.
- [ ] Skill Priority was respected and no Orchestration Anti-Patterns (router executing, persona-calls-persona, depth > 1, orchestrator that edits).
- [ ] The chosen skill was announced/invoked ("Using [skill] to [purpose]"), or its skip is documented (e.g. a meta/always-on skill).
