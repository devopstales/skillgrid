---
name: using-skillgrid
description: Use when starting any skillgrid task to classify the request and pick exactly one first skill from the router table. Use when unsure which skill applies.
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
  based_on: superpowers:using-superpowers
---

<SUBAGENT-STOP>
If you were dispatched as a subagent to execute a specific task, ignore this skill.
</SUBAGENT-STOP>

## Overview

The router. It classifies the request, runs the start gates (config, resume, domain model), and names exactly one first skill. The 1% rule chooses among Router rows. It does not scan the skill catalog.

## When to Use

- At the start of any skillgrid task, to pick the right skill.
- When unsure which skill applies to the current request.

**When NOT to use:** mid-phase, once the correct skill is already running — using-skillgrid routes at the start, it doesn't re-route every step.

## The Rule

Classify the request, then invoke exactly one first skill from the Router table before other work — including clarifying questions, exploring the codebase, or checking files. If the row turns out wrong, stop and reclassify. Do not invoke a second skill "to be safe."

The 1% test asks which Router row matches. It does not ask whether some skill in the catalog might be relevant.

**Before entering plan mode:** a delivery change that has not been brainstormed starts at the Delivery row (`brainstorming`), including the Bounded path.

**Config check:** if `.skillgrid/config.yaml` doesn't exist, the first skill is `onboarding`. All other skills read from this config.

**Resume check:** if `.skillgrid/state.yaml` names an in-flight change, or the newest `.skillgrid/specs/` directory contains in-flight artifacts (an uncompleted `tasks.md`, a latest commit whose `[skillgrid-context]` block has non-empty `Remaining:`, or a ledger under `.skillgrid/sdd/`), the first skill is `resume`. The files say where you left off; the conversation doesn't. A folder under `.skillgrid/archive/` is closed. Do not resume it.

**Domain model check:** if `.skillgrid/` exists, read `.skillgrid/ASSUMPTIONS.md` for the project's understanding, the hard boundaries (`§ Locked constraints`), and the ADRs (`§ LOCKED` — in-force set via the `### In-force set` table) for the area you're touching, plus `artifacts/01-business-terms.md` + `02-technical-terms.md` for the project's vocabulary and `artifacts/README.md` for the topic index. Use that vocabulary in questions and designs, and respect ADRs that already settled decisions. When terms resolve or a hard-to-reverse decision is made, update them via `skillgrid:architectural-decision-records`.

Then announce "Using [skill] to [purpose]" and follow that skill. If it has a checklist, create a todo per item.

## Request class

Pick one class before any skill body loads.

- **Delivery** — a behavior change in this repo. Exactly one Router row. Bounded work (an existing flow, design already obvious) uses `brainstorming`'s Bounded path: a short design in chat, approval, no spec folder. A new function or a new project uses the full planning chain. Fast-track `trivial` / `small` skips brainstorming and blueprints only when `_shared/rules/fast-track.md` says so, and the waiver is recorded in `briefing.md`.
- **Specialist** — the user asked for design, brand, craft, or a scan. Run that named skill. It does not open a spec folder. During a delivery change, these skills wait until the user or the active pipeline skill names them: everything under `design/` except `spike` and `sketch`, all of `craft/`, and the scanners `owasp-security`, `nuclei`, `wapiti`, `akca`, `securing-agentic-ai-tool-invocation`. `qa` and `requesting-code-review` may still call a scanner. Do not set `disable-model-invocation` on design skills; a design request should still find them.
- **Q&A** — a lookup. First skill is `mnemonic` or `research`. No pipeline.

## Context Discipline

Conversation memory does not survive compaction, and no skill may assume it will.

- **Persist before pressure, not after loss.** When a session is clearly long — many subagents dispatched, large tool outputs accumulating, or the harness showing a compaction indicator — STOP and: (1) append the current position to the plan's ledger (during execution) or the spec-zone artifacts (during planning), (2) commit the work unit (`skillgrid:work-unit-commits`) so the session event stream holds the position, (3) if the remaining work is large, run the save path in `skillgrid:resume`.
- **State writes are dual:** the in-repo file is the source of truth; when `mnemonic.enabled: true`, `mem_save` mirrors it under `skillgrid/<topic>/…` as an index and backup. If they disagree, the in-repo file wins.
- **After compaction** ("FIRST ACTION REQUIRED"), re-orient via `skillgrid:resume` before continuing — files first, mnemonic only as fallback.

## Router

The phase order after the first skill is `_shared/rules/sdd-structure.md`. This table names the first skill only.

| Condition | First skill | Then |
|---|---|---|
| Uninitialized (no `.skillgrid/config.yaml`) | `onboarding` | stop for validation |
| Delivery, new function or new project | `brainstorming` | phase order in `sdd-structure.md` |
| Delivery, bounded | `brainstorming` (Bounded path) | stop for approval; no spec folder |
| Delivery, trivial/small (fast-track) | `slicing` (light) | phase order in `sdd-structure.md`; skips per `fast-track.md`; waiver in `briefing.md` |
| Q&A / lookup | `mnemonic` or `research` | no pipeline |
| Spike-only | `spike` | promote to a blueprint if the user keeps the findings |
| Specialist (design, brand, craft, or a scan) | the named skill | no spec folder |
| Mid-change | `resume` | the resume marker in `sdd-structure.md` |

**Uninitialized** when `.skillgrid/config.yaml` does not exist. **Initialized** when it does. Skill-registry, terms files, and ADRs are not init signals.

**Pre-blueprint gates** (delivery, full planning chain only): hard research (external API, costly re-explore) → `research`. Taste, UI, or an unknown shape → `sketch` before locking `blueprint.md`.

## User Gate (mandatory)

After `slicing` writes `tasks.md`:

1. **Implement** → `subagent-execution` (or `simple-execution`)
2. **Revise** → `brainstorming` and/or `writing-blueprints`

Do not auto-execute. The user must confirm the slice before execution begins.

## Resume

Planning position, when `resume` is the first skill:

| State | Action |
|---|---|
| No `tasks.md` | `brainstorming` (full or Bounded — see Request class) |
| `tasks.md` exists, no tickets started | user gate |
| Tickets in progress | `subagent-execution` at the first incomplete ticket |

The tail (qa, review, ship, reflect) is the Resume markers section of `_shared/rules/sdd-structure.md`. Do not restate it here.

## Skill Priority

When several Router rows could match, pick the one class above. Inside a delivery change, process skills come first and implementation skills (for example `skillgrid:test-driven-development`) carry the work out.

- "Let's build X" → Delivery → `brainstorming`.
- "Fix this bug" → `structured-debugging`, then domain skills.
- "Design this" / "scan this" → Specialist. Do not open a spec folder.

## Orchestration Anti-Patterns

The router is a persona, not a skill that calls other skills. These failure modes mean STOP — you've turned routing into extra work:

| Anti-pattern | Why it's wrong | Do this instead |
|---|---|---|
| **Router persona** — the main agent keeps routing and does the work itself, so every task re-runs the full pipeline for trivial work. | The router should dispatch, not execute. Doing both doubles context and slows the obvious cases. | Route to the first skill, then step back. For trivial work, use the fast-track row. |
| **Persona calls persona** — a skill (for example `ship`) invokes another router-level skill as if it were a tool. | Routers set approach. They are not called by other routers. | Routers call implementation skills only. `ship` may fan out to `parallel-code-review` but never re-enters `using-skillgrid` or `brainstorming`. |
| **Depth > 1** — a subagent spawns a subagent that spawns a subagent. | Each hop re-reads context. Beyond one level the cost exceeds the benefit. | Max dispatch depth is 1. Workers do not dispatch. |
| **Paraphrasing hop** — the orchestrator re-summarizes a subagent's result before passing it on. | The worker's output is already structured. | Pass the worker's Return Envelope through verbatim. Synthesize only at the final decision (for example `ship`'s GO/NO-GO). |
| **Orchestrator that edits** — the coordinating agent edits code between dispatches. | Mixing coordination with edits muddies who owns the change. | The orchestrator collects verdicts and renders decisions. It edits only to wire up what workers produced. |
| **Catalog scan** — a delivery change invokes a design, craft, or scanner skill because its description might apply. | Those skills are specialist. They are not Router rows for a delivery change. | Stay on the Delivery row. A specialist runs when the user or the active pipeline skill names it. |

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

| | Fan-out (subagents) | Teams (teammates that message each other) |
|---|---|---|
| Sub-agents see | The same diff, different lenses | A shared task list and each other's findings |
| Output | Independent reports → one merge (a verdict on a known artifact) | Adversarial debate → consensus (an investigation among competing hypotheses) |
| Use for | `parallel-code-review`, `ship`'s GO/NO-GO | `structured-debugging` when a single agent would pick the first plausible theory and stop |

A fan-out produces a verdict on something you already have. A team produces a root cause among things you don't yet know. Do not wrap a team-style investigation in a fan-out.

## Red Flags

These thoughts mean STOP — you're rationalizing:

| Thought | Reality |
|---------|---------|
| "This is just a simple question" | Classify it. Q&A is a Router row. Delivery is a different row. |
| "I need more context first" | The Router row comes before exploring. |
| "Let me explore the codebase first" | The first skill tells you how to explore. |
| "A skill exists for this, so I'll load it" | During a delivery change, only the Router row's first skill loads. Specialist skills wait until named. |
| "I remember this skill" | Skills evolve. Read the current version of the one row you picked. |
| "The skill is overkill" | Use the Bounded row or the fast-track row. Do not scan the catalog for a lighter skill. |
| "I'll just do this one thing first" | Pick the row, then do the work inside that skill. |
| "I'll run the design skill and the pipeline" | One class. A design request does not also open a spec folder. |

## Platform Adaptation

If your harness appears here, read its reference file for special instructions:

- Codex: `references/codex-tools.md`
- Pi: `references/pi-tools.md`
- OpenCode: `references/opencode-tools.md`
- Antigravity: `references/antigravity-tools.md`
- Hermes Agent: `references/hermes-tools.md`

## User Instructions

User instructions (CLAUDE.md, AGENTS.md, GEMINI.md, and direct requests) take precedence over skills, which in turn override default behavior. Only skip skill workflows when your human partner has explicitly told you to.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll just pick a skill by name, skip the router" | The Router table maps condition → first skill. Picking by name misses the gates. |
| "The user gate is optional, I'll proceed" | The user confirms the slice before execution begins. |
| "I'll run two skills at once to be safe" | One first skill. A second skill runs when that skill names it. |
| "The router can just do the work itself" | The router dispatches. It does not execute. |
| "I remember this skill, no need to read it" | Read the current version of the first skill. |
| "The 1% rule means load every plausible skill" | The 1% rule picks one Router row. |

## Verification

- [ ] The request was classified as Delivery, Specialist, or Q&A before any skill body loaded.
- [ ] The Router selected exactly one first skill (one row, no catalog scan).
- [ ] The start gates passed: config check, resume check, and domain model check.
- [ ] The User Gate was passed — the user confirmed the slice before execution began.
- [ ] No specialist skill ran during a delivery change unless the user or the active pipeline skill named it.
- [ ] The chosen skill was announced ("Using [skill] to [purpose]").
