---
name: architectural-decision-records
description: Build and sharpen the project's domain model and record its architectural decisions. Use when discussing codebase terminology, editing the glossary in .skillgrid/glossary/, or drafting, reviewing, updating, or superseding an ADR in .skillgrid/adr/.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: mattpocock-skills:domain-modeling + intent-driven-template:architectural-decision-records + skillgrid-v2:sdd-onboard/domain
---

# Architectural Decision Records

**Announce at start:** "I'm using the skillgrid:architectural-decision-records skill to sharpen the domain model and record decisions."

## Overview

Actively build and sharpen the project's domain model as you design, and record
its load-bearing architectural decisions. This is the *active* discipline:
challenging terms, inventing edge-case scenarios, and writing the glossary and
the decisions down the moment they crystallize. (Merely *reading* the glossary
for vocabulary is not this skill — that's a one-line habit any skill can do.
This skill is for when you're *changing* the model, not just consuming it.)

## When to Use

- When making a significant architectural or design decision that should be recorded — a hard-to-reverse, surprising, real trade-off.
- When a decision needs to be reviewed or revisited later — walking the `supersedes` trail to see what is in force.
- When a glossary term is being sharpened, challenged, or cross-referenced against code during a design or interview session.

**When NOT to use:** for trivial or reversible choices that don't warrant a record (e.g. a variable name, a one-line config).

**Config:** Read `.skillgrid/config.yaml` first.
- Use `conventions.glossary` for the glossary directory (default `.skillgrid/glossary/`) — it holds `business.md` and `technical.md`.
- Use `conventions.adr` for the ADR directory (default `.skillgrid/adr/`).
- Use `adr_style` for the ADR template (default `madr-minimal`) — see [ADR Formats](#adr-formats).

## File structure

All skill artifacts live under `.skillgrid/` — never in `src/` — even in a single-context repo. The glossary is split into **business** (domain, product, workflow terms) and **technical** (architecture, platform, protocol terms) so a non-monorepo still has a clean home for both vocabularies.

Single context (most repos):

```
/
└── .skillgrid/
    ├── config.yaml
    ├── glossary/
    │   ├── business.md          ← domain, product, workflow terms
    │   └── technical.md         ← architecture, platform, protocol terms
    ├── adr/
    │   ├── 0001-event-sourced-orders.md
    │   └── 0002-postgres-for-write-model.md
    └── specs/                   ← briefing.md, blueprint.md, tasks.md
```

Multiple contexts (monorepo): a `.skillgrid/glossary/CONTEXT-MAP.md` lists the
contexts and where their per-context glossary files live. Per-context
glossary terms still live under `.skillgrid/glossary/`, keyed by context name.
**ADRs always live in the single `.skillgrid/adr/` folder** — system-wide and
context-specific alike — so the decision history is one continuous, numbered
trail you can read end to end.

Create files lazily: only when you have something to write. If no
`.skillgrid/glossary/` exists, create the stub tables when the first term is
resolved. If no `.skillgrid/adr/` exists, create it when the first ADR is
needed. Nothing speculative.

## During the session

### Challenge against the glossary

When the user uses a term that conflicts with the existing language in `.skillgrid/glossary/`, call it out immediately. "Your glossary defines 'cancellation' as X, but you seem to mean Y. Which is it?"

### Sharpen fuzzy language

When the user uses vague or overloaded terms, propose a precise canonical term. "You're saying 'account': do you mean the Customer or the User? Those are different things."

### Discuss concrete scenarios

When domain relationships are being discussed, stress-test them with specific scenarios. Invent scenarios that probe edge cases and force the user to be precise about the boundaries between concepts.

### Cross-reference with code

When the user states how something works, check whether the code agrees. If you find a contradiction, surface it: "Your code cancels entire Orders, but you just said partial cancellation is possible. Which is right?"

### Update the glossary inline

When a term is resolved, update the right glossary file right there — `business.md` for domain/product/workflow terms, `technical.md` for architecture/platform/protocol terms. Don't batch these up — capture them as they happen. Use the format in [templates/business.md](templates/business.md) / [templates/technical.md](templates/technical.md).

Prefer existing project terms (README, domain docs, code names) over new jargon. The glossary is a glossary and nothing else — no implementation details, no specs, no scratch. This is the rule that breaks in the field: left unchecked, models treat "write to the glossary" as permission to persist every answer, and the file turns into a running spec.

### Offer ADRs sparingly

Only offer to create an ADR when all three are true:

1. **Hard to reverse**: the cost of changing your mind later is meaningful
2. **Surprising without context**: a future reader will wonder "why did they do it this way?"
3. **The result of a real trade-off**: there were genuine alternatives and you picked one for specific reasons

If any of the three is missing, skip the ADR and say which test failed. One decision per ADR — split bundles. Preserve history: supersede an accepted ADR rather than rewriting it away. Use the template selected by `adr_style` in config (see [ADR Formats](#adr-formats)).

## ADR formats

`adr_style` in `.skillgrid/config.yaml` selects the template. Default is
`madr-minimal`. Onboarding asks the user and stores the choice; this skill
never re-asks once it's set.

| `adr_style` | Template | Shape |
|---|---|---|
| `madr-full` | [templates/madr-full.md](templates/madr-full.md) | Detailed tradeoff record |
| `madr-minimal` | [templates/madr-minimal.md](templates/madr-minimal.md) | Context / Considered Options / Decision Outcome / Consequences |
| `nygard` | [templates/nygard.md](templates/nygard.md) | Classic: Status / Context / Decision / Consequences |
| `y-statement` | [templates/y-statement.md](templates/y-statement.md) | One-sentence decision |
| `custom` | [templates/custom.md](templates/custom.md) | Project-specific — fill in the note in the template |

ADRs always live in the single `.skillgrid/adr/` folder — there are no
per-context ADR subfolders — and use sequential numbering: `0001-slug.md`,
`0002-slug.md`, etc. Scan the directory for the highest existing number and
increment by one; the sequence is monotonic and never reused. In a
multi-context repo, a context-specific ADR is still a file in this one folder
(prefix the slug with the context if it helps, e.g. `0003-ordering-...md`),
never a separate directory.

### Fixed header (all styles)

Every ADR carries a machine-readable header regardless of body style — this is
what makes the in-force set computable. For `madr-full` / `madr-minimal` /
`nygard` it is YAML frontmatter; for `y-statement` / `custom` a leading
blockquote line. The header has three fields:

- `status`: `proposed | accepted | deprecated | superseded`
- `date`: `YYYY-MM-DD`
- `supersedes`: `ADR-NNNN`, only when this decision replaces a prior in-force
  ADR (otherwise empty / `none`)

The body then follows the selected `adr_style`. The header is fixed; the body
is not.

## The IRON RULE — ADRs are immutable once accepted

**You MUST NOT edit a prior accepted ADR under any circumstance** — not its
status, not its body, not its date. The accepted ADR is a frozen historical
record. To change a decision, write a NEW ADR whose `status` is `accepted` and
whose `supersedes` names the prior one (`supersedes: ADR-NNNN`), and explain in
its Context *why* the prior decision is being revisited. Consumers derive what
is **currently in force** by walking `supersedes` links across
`.skillgrid/adr/`: an ADR is in force when its status is `accepted` **and** no
later ADR's `supersedes` points at it. Superseded ADRs stay in the folder,
frozen, readable end to end.

## The per-change ADR Review Manifest

Each change produces a manifest at
`.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md` — one per change, alongside
`briefing.md` / `blueprint.md` / `tasks.md`. It is the change's ADR completion
marker and the read source for downstream skills. Template:
[templates/adr-manifest.md](templates/adr-manifest.md).

Process (run at the end of the interview, before the blueprint is written):

1. **Read every ADR in `.skillgrid/adr/`.** Build the in-force set by walking
   `supersedes` links. Note the highest sequence number in use.
2. **Re-examine the design.** Identify decisions that meet ALL of:
   (a) a long-term architectural commitment (pattern, technology, boundary,
   contract) — not a tactical detail; (b) affects future changes beyond this
   one; (c) not already captured by an in-force ADR, or intentionally diverges
   from one (→ the new ADR supersedes it).
3. **Create the qualifying repo ADRs** at `.skillgrid/adr/NNNN-slug.md` (4-digit,
   one greater than the highest, never reused), each with the fixed header. A
   superseding ADR sets `status: accepted` + `supersedes: ADR-NNNN`; do NOT
   touch the prior file.
4. **Write the manifest** at `.skillgrid/specs/<topic>/adr.md`: state that ADR
   review completed, list the in-force ADRs reviewed, and reference every repo
   ADR this change created. Pointers only — never duplicate a repo ADR's
   Context / Decision / Consequences.
5. **If nothing meets the bar, say so explicitly** — "no major durable
   architectural decision introduced; no new ADR files created." Don't invent
   ADRs to fill the manifest.

## How it runs

- **Underneath `skillgrid:interviewing`** — its primary driver. The interview *is* the grilling session; this skill runs concurrently, writing terms to `.skillgrid/glossary/` and offering ADRs as decisions crystallize, so the paper trail is born in the conversation where the trade-off was actually made.
- **Underneath `skillgrid:brainstorming`** — during the codebase feasibility check and design presentation, keep the model sharp; at the end of the interview, run the per-change ADR Review Manifest (write `.skillgrid/specs/<topic>/adr.md`) so the blueprint is constrained by a *verified* in-force set.
- **Directly** — when you want the discipline without the full interview.
- The glossary, repo ADRs, and the per-change manifest are then *consumed* by `writing-blueprints`, `slicing`, `subagent-execution`, and `requesting-code-review` (one-line habit: use the vocabulary, and check the work against the in-force ADRs named in the change's `adr.md` manifest).

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll edit the ADR to match what we actually did" | The IRON RULE: an accepted ADR is a frozen historical record. Write a NEW ADR with `supersedes: ADR-NNNN` and explain why. |
| "The decision is obvious, skip the ADR" | Obvious *now* is not obvious to the future reader. If it's hard to reverse and surprising without context, it's an ADR. |
| "I'll write the ADRs after the fact in bulk" | The manifest runs at the end of the interview, *before* the blueprint. Bulk-writes miss the in-force set and the `supersedes` trail. |
| "I'll just bump the status on the old ADR" | Status is immutable on an accepted ADR. A superseding ADR sets its own `status: accepted`; the prior file is untouched. |
| "I'll add it to the glossary instead — it's faster" | The glossary is a vocabulary, not a decision record. Decisions live in `.skillgrid/adr/` with a fixed header and a `supersedes` trail. |

## Red Flags

- An accepted ADR's `status`, body, or `date` was edited in place instead of superseded.
- A superseding ADR's `supersedes` points at a number that no longer exists, or at two ADRs at once.
- A change's `adr.md` manifest is missing, or lists repo ADRs that don't exist in `.skillgrid/adr/`.
- The glossary file is absorbing implementation details, specs, or scratch — it's becoming a running spec.
- ADR sequence numbers are reused or out of monotonic order in `.skillgrid/adr/`.
- A per-context ADR subfolder exists under `.skillgrid/adr/` — all ADRs live in the one folder.

## Verification

- [ ] Every new ADR file exists in `.skillgrid/adr/` with a monotonic 4-digit number and the fixed header (`status`, `date`, `supersedes`).
- [ ] Every new ADR carries `status: accepted` (or `proposed`) and, when replacing a prior decision, a `supersedes: ADR-NNNN` that resolves to a real file.
- [ ] No accepted ADR was mutated: `git diff` on `.skillgrid/adr/` shows only new files, no edits to previously accepted ADRs.
- [ ] The per-change manifest exists at `.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md`, lists the in-force ADRs reviewed, and references every repo ADR the change created.
- [ ] If no decision met the bar, the manifest explicitly says "no major durable architectural decision introduced; no new ADR files created."
- [ ] The glossary files contain vocabulary only — no implementation details, no specs, no scratch.
