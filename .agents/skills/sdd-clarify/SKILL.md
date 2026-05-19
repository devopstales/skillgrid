---
name: sdd-clarify
description: Clarifying session that challenges your plan against the existing domain model, sharpens terminology, and updates .skillgrid/project/CONTEXT.md as decisions crystallise. Use when user wants to stress-test a plan against their project's language and documented decisions.
license: Apache-2.0
metadata:
  author: devopstales
  version: "1.0"
---

<global-context>

## Persistent glossary (`.skillgrid/project/CONTEXT.md`)

Before starting, read `.skillgrid/project/CONTEXT.md` if it exists.  
If it exists, state: *"I see from CONTEXT.md that you define [term] as [definition]. I will use that as my baseline."*

If the file does not exist, create it (including its parent directory) when the first term is resolved.

### Format for `.skillgrid/project/CONTEXT.md`

```markdown
# Project Context

## Domain
[One‑line description of the main domain]

## Glossary
- **Term**: definition (as agreed by the user)

## Assumptions
- [Any non‑obvious assumptions the agent should remember]

## Success Criteria
- [What "done" looks like from a domain perspective]
```

### Update rules

- When a term is **resolved** (i.e., it is a core domain concept that will be used across the whole project), write it immediately to `Glossary` in `.skillgrid/project/CONTEXT.md`. Use the exact format above.
- When you capture a high‑level assumption about the system, add it to `Assumptions`.
- When you agree on what success looks like for the project, write it to `Success Criteria`.
- Append new entries; never delete existing ones without asking the user.

</global-context>

<what-to-do>

1. **Read `.skillgrid/project/CONTEXT.md`** (if present) and acknowledge its definitions.
2. Interview me relentlessly about every aspect of this plan until we reach a shared understanding. Walk down each branch of the design tree, resolving dependencies between decisions one‑by‑one. For each question, provide your recommended answer.
3. Ask the questions one at a time, waiting for feedback on each question before continuing.
4. If a question can be answered by exploring the codebase, explore the codebase instead.

</what-to-do>

<supporting-info>

## Domain awareness

During codebase exploration, also look for existing documentation:

### File structure

Most repos have a single context:

```
/
├── .skillgrid/
│   ├── project/
│       └── CONTEXT.md <- global glossary & assumptions
│   └── adr/
│       ├── 0001-event-sourced-orders.md
│       └── 0002-postgres-for-write-model.md
└── src/
```

If a `CONTEXT-MAP.md` exists at the root, the repo has multiple contexts. The map points to where each one lives:

```
/
├── .skillgrid/
│   ├── project/
│   │   └── CONTEXT.md
│   └── adr/                          ← system-wide decisions
├── src/
│   ├── ordering/
│   │   ├── CONTEXT.md
│   │   └── docs/adr/                 ← context-specific decisions
│   └── billing/
│       ├── CONTEXT.md
│       └── docs/adr/
```

Create files lazily — only when you have something to write. If no `CONTEXT.md` exists, create one when the first term is resolved. If no `docs/adr/` exists, create it when the first ADR is needed.

## During the session

### Challenge against the glossary

When the user uses a term that conflicts with the existing language in `.skillgrid/project/CONTEXT.md`, call it out immediately: *"Your CONTEXT.md defines 'cancellation' as X, but you seem to mean Y — which is it?"*

### Sharpen fuzzy language

When the user uses vague or overloaded terms, propose a precise canonical term: *"You're saying 'account' — do you mean the Customer or the User? Those are different things."*

### Discuss concrete scenarios

When domain relationships are being discussed, stress-test them with specific scenarios. Invent scenarios that probe edge cases and force the user to be precise about the boundaries between concepts.

### Cross-reference with code

When the user states how something works, check whether the code agrees. If you find a contradiction, surface it: *"Your code cancels entire Orders, but you just said partial cancellation is possible — which is right?"*

### Update `.skillgrid/project/CONTEXT.md` inline

When a term is resolved, update `.skillgrid/project/CONTEXT.md` right there. Don't batch these up — capture them as they happen. Use the format shown above.

Do **not** couple `CONTEXT.md` to implementation details. Only include terms that are meaningful to domain experts.

### Offer ADRs sparingly

Only offer to create an ADR (in `.skillgrid/adr/`) when all three are true:

1. **Hard to reverse** — the cost of changing your mind later is meaningful
2. **Surprising without context** — a future reader will wonder "why did they do it this way?"
3. **The result of a real trade-off** — there were genuine alternatives and you picked one for specific reasons

If any of the three is missing, skip the ADR. Use the format in [template-adr.md](.skillgrid/templates/template-adr.md).

## Return

- Load `skills/_shared/sdd-phase-common.md` for executor protocol.
- **Return:** the standard SDD envelope per `skills/_shared/sdd-return-envelope.md`.

</supporting-info>