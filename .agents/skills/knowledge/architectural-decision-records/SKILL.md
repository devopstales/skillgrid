---
name: architectural-decision-records
description: Build and sharpen the project's domain model and record its architectural decisions. Use when discussing codebase terminology, editing the terms files in .skillgrid/artifacts/, or authoring, reviewing, updating, or superseding an `### ADR-NNNN` entry in .skillgrid/ASSUMPTIONS.md.
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
challenging terms, inventing edge-case scenarios, and writing the terms and
the decisions down the moment they crystallize. (Merely *reading* the terms
for vocabulary is not this skill — that's a one-line habit any skill can do.
This skill is for when you're *changing* the model, not just consuming it.)

## When to Use

- When making a significant architectural or design decision that should be recorded — a hard-to-reverse, surprising, real trade-off.
- When a decision needs to be reviewed or revisited later — walking the `supersedes` trail to see what is in force.
- When a term is being sharpened, challenged, or cross-referenced against code during a design or interview session.

**When NOT to use:** for trivial or reversible choices that don't warrant a record (e.g. a variable name, a one-line config).

**Config:** Read `.skillgrid/config.yaml` first.
- Use `conventions.artifacts` for the terms directory (default `.skillgrid/artifacts/`) — it holds the terms files (`01-business-terms.md`, `02-technical-terms.md`).
- ADRs are **not separate files**. They live as `### ADR-NNNN` entries inside the single root file `.skillgrid/ASSUMPTIONS.md`, in its `## LOCKED` section (subsections `### In-force set` table, then `### ADR-0001`…`### ADR-NNNN` entries, then `### Locked constraints`). The in-force set is read from the `### In-force set` table, not a separate index.
- Use `adr_style` for the ADR entry's body shape (default `madr-minimal`) — see [ADR Formats](#adr-formats).

## File structure

All skill artifacts live under `.skillgrid/` — never in `src/` — even in a single-context repo. The vocabulary is split into **business** (domain, product, workflow terms) and **technical** (architecture, platform, protocol terms) so a non-monorepo still has a clean home for both vocabularies. The terms live in `.skillgrid/artifacts/`; the decisions live as `### ADR-NNNN` entries in the single root `.skillgrid/ASSUMPTIONS.md` — the durable, cross-change knowledge base.

Single context (most repos):

```
/
└── .skillgrid/
    ├── config.yaml
    ├── state.yaml
    ├── ASSUMPTIONS.md             ← VERIFIED / INFERRED / LOCKED tiers
    │                                (LOCKED: ### In-force set table +
    │                                 ### ADR-0001…NNNN entries +
    │                                 ### Locked constraints)
    ├── ARCHITECTURE.md            ← live repo/program structure
    ├── artifacts/
    │   ├── 01-business-terms.md   ← domain, product, workflow terms
    │   └── 02-technical-terms.md  ← architecture, platform, protocol terms
    ├── spikes/NNN-name/           ← feasibility probes (permanently retained)
    └── specs/                     ← briefing.md, blueprint.md, tasks.md, adr.md (manifest)
```

Multiple contexts (monorepo): a `.skillgrid/artifacts/CONTEXT-MAP.md` lists the
contexts and where their per-context terms live. Per-context terms still live in
`artifacts/01-business-terms.md` / `02-technical-terms.md`, keyed by context name
(a `## <context>` section per file). **ADRs always live as entries in the single
`ASSUMPTIONS.md` `## LOCKED` set** — system-wide and context-specific alike — so
the decision history is one continuous, numbered trail you can read end to end
(prefix the slug with the context if it helps, e.g. `### ADR-0003 <context>-…`).

Create files lazily: only when you have something to write. If no
`artifacts/01-business-terms.md` exists, create the stub table when the first term
is resolved. If `ASSUMPTIONS.md` has no `### In-force set` table yet, create the
`## LOCKED` section when the first ADR is needed. Nothing speculative.

## During the session

### Challenge against the terms

When the user uses a term that conflicts with the existing language in `.skillgrid/artifacts/01-business-terms.md` / `02-technical-terms.md`, call it out immediately. "Your terms define 'cancellation' as X, but you seem to mean Y. Which is it?"

### Sharpen fuzzy language

When the user uses vague or overloaded terms, propose a precise canonical term. "You're saying 'account': do you mean the Customer or the User? Those are different things."

### Discuss concrete scenarios

When domain relationships are being discussed, stress-test them with specific scenarios. Invent scenarios that probe edge cases and force the user to be precise about the boundaries between concepts.

### Cross-reference with code

When the user states how something works, check whether the code agrees. If you find a contradiction, surface it: "Your code cancels entire Orders, but you just said partial cancellation is possible. Which is right?"

### Update the terms inline

When a term is resolved, update the right terms file right there — `artifacts/01-business-terms.md` for domain/product/workflow terms, `artifacts/02-technical-terms.md` for architecture/platform/protocol terms. Don't batch these up — capture them as they happen. Use the format in [templates/business.md](templates/business.md) / [templates/technical.md](templates/technical.md).

Prefer existing project terms (README, domain docs, code names) over new jargon. The terms files are a glossary and nothing else — no implementation details, no specs, no scratch. This is the rule that breaks in the field: left unchecked, models treat "write to the terms" as permission to persist every answer, and the file turns into a running spec.

### Offer ADRs sparingly

This is the ADR-sparingly test the global code standard (`_shared/conventions/code-standards.md`)
references: the ADR is the home for the *per-language idioms* and *hard-to-reverse
decisions* that the global layer deliberately defers. Only offer to create an ADR
when all three are true:

1. **Hard to reverse**: the cost of changing your mind later is meaningful
2. **Surprising without context**: a future reader will wonder "why did they do it this way?"
3. **The result of a real trade-off**: there were genuine alternatives and you picked one for specific reasons

If any of the three is missing, skip the ADR and say which test failed. One decision per ADR — split bundles. Preserve history: supersede an accepted ADR entry rather than rewriting it away. Use the body shape selected by `adr_style` in config (see [ADR Formats](#adr-formats)).

## ADR formats

`adr_style` in `.skillgrid/config.yaml` selects the template. Default is
`madr-minimal`. Onboarding asks the user and stores the choice; this skill
never re-asks once it's set.

| `adr_style` | Template | Shape (body of the `### ADR-NNNN` entry) |
|---|---|---|
| `madr-full` | [templates/madr-full.md](templates/madr-full.md) | Detailed tradeoff record |
| `madr-minimal` | [templates/madr-minimal.md](templates/madr-minimal.md) | Context / Considered Options / Decision Outcome / Consequences |
| `nygard` | [templates/nygard.md](templates/nygard.md) | Classic: Context / Decision / Consequences |
| `y-statement` | [templates/y-statement.md](templates/y-statement.md) | One-sentence decision |
| `custom` | [templates/custom.md](templates/custom.md) | Project-specific — fill in the note in the template |

ADRs always live as `### ADR-NNNN` entries in the single root `ASSUMPTIONS.md`
`## LOCKED` section — there are no per-context ADR subfolders or separate files.
Use sequential numbering: `### ADR-0001`, `### ADR-0002`, etc. Scan the
`### In-force set` table (its "Highest sequence in use" line) for the highest
existing number and increment by one; the sequence is monotonic and never reused.
In a multi-context repo, a context-specific ADR is still an entry in this one set
(prefix the slug with the context if it helps, e.g. `### ADR-0003 <context>-…`),
never a separate file or directory. Every new or superseding ADR gets a row in
the `### In-force set` table in the same edit.

### Fixed heading line (all styles)

Every ADR entry carries a machine-readable heading line regardless of body shape
— this is what makes the in-force set computable. The heading has three fields:

- `### ADR-NNNN — <slug>` — the entry heading
- a `**Status.**` line: `proposed | accepted | deprecated | superseded`
- a `**Date.**` line: `YYYY-MM-DD`
- a `**Supersedes.**` line: `ADR-NNNN`, only when this decision replaces a prior
  in-force ADR (otherwise `—`)

The body then follows the selected `adr_style`. The heading is fixed; the body
is not. The `### In-force set` table mirrors the three fields (`#`, `Status`,
`Supersedes`, `Date`, `In force`) for a single-glance view.

## The IRON RULE — never delete an ADR entry

**You MUST NOT delete a prior accepted ADR entry under any circumstance** — not
its status, not its body, not its date. An accepted ADR entry is a frozen
historical record. To change a decision, author a NEW `### ADR-NNNN` entry whose
`**Status.**` is `accepted` and whose `**Supersedes.**` names the prior one
(`ADR-NNNN`), explain in its Context *why* the prior decision is being revisited,
and **flip the prior row in the `### In-force set` table to "In force: no"**.
Consumers derive what is **currently in force** from the `### In-force set` table
in `ASSUMPTIONS.md` (which walks the `supersedes` links for them): an ADR is in
force when its status is `accepted` **and** no later ADR's `supersedes` points at
it. Superseded entries stay in `ASSUMPTIONS.md`, frozen, readable end to end —
the table row is flipped, the entry is never removed.

## The per-change ADR Review Manifest

Each change produces a manifest at
`.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md` — one per change, alongside
`briefing.md` / `blueprint.md` / `tasks.md`. It is the change's ADR completion
marker and the read source for downstream skills. Template:
[templates/adr-manifest.md](templates/adr-manifest.md).

Process (run at the end of the interview, before the blueprint is written):

1. **Read `.skillgrid/ASSUMPTIONS.md` § `## LOCKED`.** The in-force set is the
   rows in the `### In-force set` table marked "In force: yes". Note the
   "Highest sequence in use" line. (For a full read, open the `### ADR-NNNN`
   entries the table points to.)
2. **Re-examine the design.** Identify decisions that meet ALL of:
   (a) a long-term architectural commitment (pattern, technology, boundary,
   contract) — not a tactical detail; (b) affects future changes beyond this
   one; (c) not already captured by an in-force ADR, or intentionally diverges
   from one (→ the new ADR supersedes it).
3. **Author the qualifying ADR entries** as `### ADR-NNNN — <slug>` in
   `ASSUMPTIONS.md` § `### ADR-0001`… (4-digit, one greater than the highest,
   never reused), each with the fixed heading line, and append each to the
   `### In-force set` table (flip the superseded row's "In force" to `no`). A
   superseding entry sets `**Status.** accepted` + `**Supersedes.** ADR-NNNN`;
   do NOT touch the prior entry.
4. **Write the manifest** at `.skillgrid/specs/<topic>/adr.md`: state that ADR
   review completed, list the in-force ADRs reviewed, and reference every ADR
   entry this change created (pointers to `ASSUMPTIONS.md § ### ADR-NNNN`).
   Pointers only — never duplicate an entry's Context / Decision / Consequences.
5. **If nothing meets the bar, say so explicitly** — "no major durable
   architectural decision introduced; no new ADR entries created." Don't invent
   ADRs to fill the manifest.

## How it runs

- **Underneath `skillgrid:interviewing`** — its primary driver. The interview *is* the grilling session; this skill runs concurrently, writing terms to `.skillgrid/artifacts/01-business-terms.md` / `02-technical-terms.md` and offering ADR entries (to `ASSUMPTIONS.md § ### ADR-NNNN`) as decisions crystallize, so the paper trail is born in the conversation where the trade-off was actually made.
- **Underneath `skillgrid:brainstorming`** — during the codebase feasibility check and design presentation, keep the model sharp; at the end of the interview, run the per-change ADR Review Manifest (write `.skillgrid/specs/<topic>/adr.md`) so the blueprint is constrained by a *verified* in-force set.
- **Directly** — when you want the discipline without the full interview.
- The terms files, the `### ADR-NNNN` entries in `ASSUMPTIONS.md`, and the per-change manifest are then *consumed* by `writing-blueprints`, `slicing`, `subagent-execution`, and `requesting-code-review` (one-line habit: use the vocabulary, and check the work against the in-force ADRs named in the change's `adr.md` manifest).

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll edit the ADR to match what we actually did" | The IRON RULE: an accepted ADR is a frozen historical record. Write a NEW ADR with `supersedes: ADR-NNNN` and explain why. |
| "The decision is obvious, skip the ADR" | Obvious *now* is not obvious to the future reader. If it's hard to reverse and surprising without context, it's an ADR. |
| "I'll write the ADRs after the fact in bulk" | The manifest runs at the end of the interview, *before* the blueprint. Bulk-writes miss the in-force set and the `supersedes` trail. |
| "I'll just bump the status on the old ADR" | Status is immutable on an accepted ADR entry. A superseding entry sets its own `**Status.** accepted`; the prior entry is untouched. |
| "I'll add it to the terms instead — it's faster" | The terms files are a vocabulary, not a decision record. Decisions live as `### ADR-NNNN` entries in `ASSUMPTIONS.md` with a fixed heading and a `supersedes` trail (in-force set in the `### In-force set` table). |

## Red Flags

- An accepted ADR entry's status, body, or date was edited in place instead of superseded.
- A superseding entry's `**Supersedes.**` points at a number that no longer exists, or at two ADRs at once.
- A change's `adr.md` manifest is missing, or lists ADR entries that don't exist in `ASSUMPTIONS.md`.
- The `### In-force set` table disagrees with the `### ADR-NNNN` entries (a row missing, a stale "In force", a wrong highest-sequence line).
- The terms file is absorbing implementation details, specs, or scratch — it's becoming a running spec.
- ADR sequence numbers are reused or out of monotonic order in `ASSUMPTIONS.md`.
- A separate ADR file or per-context ADR subfolder exists — all ADRs live as entries in the one `ASSUMPTIONS.md` `## LOCKED` set.

## Verification

- [ ] Every new ADR exists as a `### ADR-NNNN — <slug>` entry in `.skillgrid/ASSUMPTIONS.md` with a monotonic 4-digit number and the fixed heading (`**Status.**`, `**Date.**`, `**Supersedes.**`).
- [ ] Every new or superseding ADR has a matching row in the `### In-force set` table, and the "Highest sequence in use" line is updated.
- [ ] Every new ADR carries `**Status.** accepted` (or `proposed`) and, when replacing a prior decision, a `**Supersedes.** ADR-NNNN` that resolves to a real entry.
- [ ] No accepted ADR entry was mutated: `git diff` on `ASSUMPTIONS.md` shows only new entries/rows, no edits to previously accepted entries.
- [ ] The per-change manifest exists at `.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md`, lists the in-force ADRs reviewed, and references every ADR entry the change created (pointers to `ASSUMPTIONS.md § ### ADR-NNNN`).
- [ ] If no decision met the bar, the manifest explicitly says "no major durable architectural decision introduced; no new ADR entries created."
- [ ] The terms files contain vocabulary only — no implementation details, no specs, no scratch.
