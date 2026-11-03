---
name: architectural-decision-records
description: Build and sharpen the project's domain model and record its architectural decisions. Use when discussing codebase terminology, editing the terms files in .skillgrid/artifacts/, or authoring, reviewing, updating, or superseding an ADR file at .skillgrid/artifacts/04-adr-NNNN-slug.md.
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
- ADRs are **separate files**: `.skillgrid/artifacts/04-adr-NNNN-slug.md`, one decision per file. `.skillgrid/ASSUMPTIONS.md` § `## LOCKED` is the ADR index: one row per decision (a short description + the path to the record). `status`/`supersedes`/`date` live in the file's frontmatter, not the row. The table does not hold the body. The in-force set is read from that table.
- Use `adr_style` for the ADR entry's body shape (default `madr-minimal`) — see [ADR Formats](#adr-formats).

## File structure

All skill artifacts live under `.skillgrid/` — never in `src/` — even in a single-context repo. The vocabulary is split into **business** (domain, product, workflow terms) and **technical** (architecture, platform, protocol terms) so a non-monorepo still has a clean home for both vocabularies. The terms and the ADR files live in `.skillgrid/artifacts/`. `.skillgrid/ASSUMPTIONS.md` stores the path to each ADR.

Single context (most repos):

```
/
└── .skillgrid/
    ├── config.yaml
    ├── state.yaml
    ├── ASSUMPTIONS.md             ← VERIFIED / INFERRED / LOCKED tiers
    │                                (LOCKED: the ADR index, one pointer row
    │                                 per in-force decision; operational rules
    │                                 live in AGENTS.md, not here)
    ├── ARCHITECTURE.md            ← live repo/program structure
    ├── artifacts/
    │   ├── 01-business-terms.md   ← domain, product, workflow terms
    │   ├── 02-technical-terms.md  ← architecture, platform, protocol terms
    │   └── 04-adr-NNNN-slug.md    ← one locked decision per file
    ├── prototypes/NNN-name/           ← feasibility probes (permanently retained)
    └── specs/                     ← briefing.md, blueprint.md, tasks.md, adr.md (manifest)
```

Multiple contexts (monorepo): a `.skillgrid/artifacts/CONTEXT-MAP.md` lists the
contexts and where their per-context terms live. Per-context terms still live in
`artifacts/01-business-terms.md` / `02-technical-terms.md`, keyed by context name
(a `## <context>` section per file). **ADRs always live as files in `artifacts/`**,
system-wide and context-specific alike, so the decision history is one continuous,
numbered trail (prefix the slug with the context if it helps, e.g.
`04-adr-0003-<context>-….md`). `ASSUMPTIONS.md` holds the path.

Create files lazily: only when you have something to write. If no
`artifacts/01-business-terms.md` exists, create the stub table when the first term
is resolved. If `ASSUMPTIONS.md` has no `## LOCKED` ADR index yet, create the
`## LOCKED` section when the first ADR file is written, and add the pointer row.
Nothing speculative.

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

This is the ADR-sparingly test the global code standard (`_shared/rules/code-standards.md`)
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

| `adr_style` | Template | Shape (body of `artifacts/04-adr-NNNN-slug.md`) |
|---|---|---|
| `madr-full` | [templates/madr-full.md](templates/madr-full.md) | Detailed tradeoff record |
| `madr-minimal` | [templates/madr-minimal.md](templates/madr-minimal.md) | Context / Considered Options / Decision Outcome / Consequences |
| `nygard` | [templates/nygard.md](templates/nygard.md) | Classic: Context / Decision / Consequences |
| `y-statement` | [templates/y-statement.md](templates/y-statement.md) | One-sentence decision |
| `custom` | [templates/custom.md](templates/custom.md) | Project-specific — fill in the note in the template |

Write the body to `.skillgrid/artifacts/04-adr-NNNN-slug.md`. Use sequential
numbering: `0001`, `0002`, etc. Scan the `## LOCKED` ADR index (its "Highest
sequence in use" line) for the highest existing number and increment by one; the
sequence is monotonic and never reused. In a multi-context repo, prefix the slug
with the context (`04-adr-0003-<context>-….md`). Every new or superseding ADR
gets a pointer row in the `## LOCKED` ADR index in the same edit. The row is a
short description + the path. The file is the body.

When `ticketing.type` is `backlogmd`, copy the ADR into
`.backlog/decisions/` in the same edit (see [Backlog.md copy](#backlogmd-copy)).
The artifact file stays the record (ADR-0019). The decision file is the tracker
copy so `backlog decision list` / `backlog search` can find it.

### Fixed heading (all styles)

Every ADR file carries machine-readable status fields regardless of body shape:

- `# <title>` — the file title
- frontmatter `status`: `proposed | accepted | deprecated | superseded`
- frontmatter `date`: `YYYY-MM-DD`
- frontmatter `supersedes`: `ADR-NNNN`, only when this decision replaces a prior
  in-force ADR (otherwise `none`)

The body then follows the selected `adr_style`. The `## LOCKED` ADR index row
carries only a short description and the path — `status`/`supersedes`/`date` live
in the file's frontmatter, not the row.

## The IRON RULE — never delete an ADR file

**You MUST NOT delete a prior accepted ADR file under any circumstance** — not
its status, not its body, not its date, not its row. An accepted ADR file is a
frozen historical record. To change a decision, author a NEW
`04-adr-NNNN-slug.md` whose status is `accepted` and whose `supersedes` names
the prior one, explain in its Context *why* the prior decision is being
revisited, and **flip the prior row in the `## LOCKED` ADR index to "In force:
no"**. Consumers derive what is **currently in force** from that index:
an ADR is in force when its status is `accepted` **and** no later ADR's
`supersedes` points at it. Superseded files stay on disk. The index row is
flipped. The file is never removed.

## The per-change ADR Review Manifest

Each change produces a manifest at
`.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md` — one per change, alongside
`briefing.md` / `blueprint.md` / `tasks.md`. It is the change's ADR completion
marker and the read source for downstream skills. Template:
[templates/adr-manifest.md](templates/adr-manifest.md).

Process (run at the end of the interview, before the blueprint is written):

1. **Read `.skillgrid/ASSUMPTIONS.md` § `## LOCKED`.** The in-force set is the
   rows in the ADR index marked "In force: yes". Note the
   "Highest sequence in use" line. Open the linked file when you need the body.
2. **Re-examine the design.** Identify decisions that meet ALL of:
   (a) a long-term architectural commitment (pattern, technology, boundary,
   contract) — not a tactical detail; (b) affects future changes beyond this
   one; (c) not already captured by an in-force ADR, or intentionally diverges
   from one (→ the new ADR supersedes it).
3. **Author the qualifying ADR files** as `artifacts/04-adr-NNNN-slug.md`
   (4-digit, one greater than the highest, never reused), each with the fixed
    frontmatter, and append a pointer row to the `## LOCKED` ADR index (flip
    the superseded row's "In force" to `no`). A superseding file sets
   `status: accepted` and `supersedes: ADR-NNNN`. Do NOT edit the prior file.
   If `ticketing.type` is `backlogmd`, copy each new or superseding file into
   `.backlog/decisions/` and refresh the superseded copy's `status` to
   `superseded`.
4. **Write the manifest** at `.skillgrid/specs/<topic>/adr.md`: state that ADR
   review completed, list the in-force ADRs reviewed, and reference every ADR
   file this change created (pointers to `artifacts/04-adr-NNNN-slug.md`).
   Pointers only — never duplicate a file's Context / Decision / Consequences.
5. **If nothing meets the bar, say so explicitly** — "no major durable
   architectural decision introduced; no new ADR files created." Don't invent
   ADRs to fill the manifest.

## How it runs

- **Underneath `skillgrid:interviewing`** — its primary driver. The interview *is* the grilling session; this skill runs concurrently, writing terms to `.skillgrid/artifacts/01-business-terms.md` / `02-technical-terms.md` and offering ADR files (`artifacts/04-adr-NNNN-slug.md`, plus a path row in `ASSUMPTIONS.md`) as decisions crystallize, so the paper trail is born in the conversation where the trade-off was actually made.
- **Underneath `skillgrid:brainstorming`** — during the codebase feasibility check and design presentation, keep the model sharp; at the end of the interview, run the per-change ADR Review Manifest (write `.skillgrid/specs/<topic>/adr.md`) so the blueprint is constrained by a *verified* in-force set.
- **Directly** — when you want the discipline without the full interview.
- The terms files, the ADR files, and the per-change manifest are then *consumed* by `writing-blueprints`, `slicing`, `subagent-execution`, and `requesting-code-review` (one-line habit: use the vocabulary, and check the work against the in-force ADRs named in the change's `adr.md` manifest).

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll edit the ADR to match what we actually did" | The IRON RULE: an accepted ADR is a frozen historical record. Write a NEW ADR with `supersedes: ADR-NNNN` and explain why. |
| "The decision is obvious, skip the ADR" | Obvious *now* is not obvious to the future reader. If it's hard to reverse and surprising without context, it's an ADR. |
| "I'll write the ADRs after the fact in bulk" | The manifest runs at the end of the interview, *before* the blueprint. Bulk-writes miss the in-force set and the `supersedes` trail. |
| "I'll just bump the status on the old ADR" | Status is immutable on an accepted ADR file. A superseding file sets its own `status: accepted`; the prior file is untouched. |
| "I'll add it to the terms instead — it's faster" | The terms files are a vocabulary, not a decision record. Decisions live as `artifacts/04-adr-NNNN-slug.md` with a pointer row in the `## LOCKED` ADR index. |
| "I'll paste the body into ASSUMPTIONS.md" | `ASSUMPTIONS.md` stores the path. The body is the file. |

## Red Flags

- An accepted ADR file's status, body, or date was edited in place instead of superseded.
- A superseding file's `supersedes` points at a number that no longer exists, or at two ADRs at once.
- A change's `adr.md` manifest is missing, or lists ADR files that don't exist.
- The `## LOCKED` ADR index disagrees with the files (a row missing, a stale "In force", a path that doesn't resolve, a wrong highest-sequence line).
- The terms file is absorbing implementation details, specs, or scratch — it's becoming a running spec.
- ADR sequence numbers are reused or out of monotonic order.
- An ADR body was pasted into `ASSUMPTIONS.md`. The table stores the path.
- An ADR file has no matching `.backlog/decisions/decision-NNN` copy when `ticketing.type` is `backlogmd`.

## Backlog.md copy

When `.skillgrid/config.yaml` has `ticketing.type: backlogmd`, every ADR file
has a tracker copy under `.backlog/decisions/`. Do this in the same edit that
writes the artifact — a missing copy is an incomplete ADR.

Write the decision file yourself so the Backlog id stays aligned with the ADR
number (`ADR-0024` → `decision-024`). Do **not** use `backlog decision create`
for these copies: it assigns the next free id and the pair drifts.

```
.backlog/decisions/decision-NNN - ADR-NNNN-slug.md
```

`NNN` is the ADR number zero-padded to the project's `zero_padded_ids` (this
repo: 3). Quote `title` in the decision frontmatter — titles often contain
colons. The file shape:

```markdown
---
id: decision-NNN
title: "ADR-NNNN <title from the artifact H1>"
date: 'YYYY-MM-DD'
status: accepted
---

Source: `.skillgrid/artifacts/04-adr-NNNN-slug.md`

<verbatim artifact file>
```

Rules:

- The artifact is the record. If the copy disagrees, recopy from the artifact
  (ADR-0013 / ADR-0019).
- A new ADR → a new decision file. A superseding ADR → a new decision file
  **and** set the prior copy's `status` to `superseded`. Do not delete a copy.
- Onboarding copies any `04-adr-*.md` that has no matching `decision-NNN` file.
- Other trackers (`gh` / `glab` / `jira`) do not get a copy.

## Verification

- [ ] Every new ADR exists as `.skillgrid/artifacts/04-adr-NNNN-slug.md` with a monotonic 4-digit number and frontmatter (`status`, `date`, `supersedes`).
- [ ] When `ticketing.type` is `backlogmd`, `.backlog/decisions/decision-NNN - ADR-NNNN-slug.md` exists, lists the Source path, and carries the same status as the artifact.
- [ ] Every new or superseding ADR has a matching pointer row in the `## LOCKED` ADR index, and the "Highest sequence in use" line is updated.
- [ ] Every new ADR carries `status: accepted` (or `proposed`) and, when replacing a prior decision, a `supersedes` that resolves to a real file.
- [ ] No accepted ADR file was mutated: `git diff` shows a new file and a new table row, no edits to previously accepted files.
- [ ] The per-change manifest exists at `.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md`, lists the in-force ADRs reviewed, and references every ADR file the change created.
- [ ] If no decision met the bar, the manifest explicitly says "no major durable architectural decision introduced; no new ADR files created."
- [ ] The terms files contain vocabulary only — no implementation details, no specs, no scratch.
- [ ] `ASSUMPTIONS.md` contains the path and not the body.
