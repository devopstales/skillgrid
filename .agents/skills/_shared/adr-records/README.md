# ADR Records (shared templates + contract)

The single source of truth for ADR authoring and the terms glossary, and the
home of the ADR discipline formerly carried by the
`architectural-decision-records` skill (deleted; its contract lives here).
Consumed by `brainstorm` (interview: terms + ADR offers + per-change
manifest), `blueprint` (ADR rules for decisions the blueprint locks in,
ratification of ASSUMED blueprints), and `skillgrid` onboarding (seeds
`adr_style`). No consumer restates this contract; they cite it.

## Template selection (the only place the mapping lives)

Read `adr_style` from `.skillgrid/config.yaml`. Default is `madr-minimal` when
the key is absent. Onboarding asks the user and stores the choice; consumers
never re-ask once it is set.

| `adr_style` | Template | Shape |
|---|---|---|
| `madr-full` | [madr-full.md](madr-full.md) | Detailed tradeoff record |
| `madr-minimal` | [madr-minimal.md](madr-minimal.md) | Context / Considered Options / Decision Outcome / Consequences |
| `nygard` | [nygard.md](nygard.md) | Classic: Context / Decision / Consequences |
| `y-statement` | [y-statement.md](y-statement.md) | One-sentence decision |
| `custom` | [custom.md](custom.md) | Project-specific — fill in the note in the template |

Glossary templates (always these two, one row per term):

| File | Template |
|---|---|
| `.skillgrid/projectwiki/glossary/01-business-terms.md` | `blueprint/references/adr-business-template.md` — domain, product, workflow terms |
| `.skillgrid/projectwiki/glossary/02-technical-terms.md` | `blueprint/references/adr-technical-template.md` — architecture, platform, protocol terms |

Per-change ADR Review Manifest (always `brainstorm/references/adr-manifest.md`):

`<specs_root>/YYYY-MM-DD-<topic>/adr.md` — pointers only, never an ADR body.

## Fixed heading (all styles)

Every ADR file carries machine-readable status fields regardless of body
shape:

- `# <title>` — the file title
- frontmatter `status`: `proposed | accepted | deprecated | superseded`
- frontmatter `date`: `YYYY-MM-DD`
- frontmatter `supersedes`: `ADR-NNNN`, only when this decision replaces a
  prior in-force ADR (otherwise `none`)

## The IRON RULE — never delete an ADR file

An accepted ADR file is a frozen historical record. To change a decision,
author a NEW `decisions/NNNN-slug.md` whose `status` is `accepted` and
whose `supersedes` names the prior one, explain in its Context *why* the prior
decision is being revisited, and flip the prior row in the `## LOCKED` ADR
index to "In force: no". Consumers derive what is **currently in force** from
that index: an ADR is in force when its status is `accepted` **and** no later
ADR's `supersedes` points at it. Superseded files stay on disk. The index row
is flipped. The file is never removed.

## Numbering

Sequential 4-digit: `0001`, `0002`, etc. Scan the `## LOCKED` ADR index (its
"Highest sequence in use" line) for the highest existing number and increment
by one; the sequence is monotonic and never reused. In a multi-context repo,
prefix the slug with the context (`0003-<context>-….md`). Every new or
superseding ADR gets a pointer row in the `## LOCKED` ADR index in the same
edit — a short description + the path; `status`/`supersedes`/`date` live in
the file's frontmatter, not the row.

## ADR-sparingly test

Only record an ADR when all three are true:

1. **Hard to reverse** — the cost of changing your mind later is meaningful
2. **Surprising without context** — a future reader will wonder "why did they
   do it this way?"
3. **The result of a real trade-off** — there were genuine alternatives and
   one was picked for specific reasons

If any of the three is missing, skip the ADR and say which test failed. One
decision per ADR — split bundles.

## Terms rules

Prefer existing project terms (README, domain docs, code names) over new
jargon. The terms files are a glossary and nothing else — no implementation
details, no specs, no scratch. Update them inline as terms resolve; don't
batch.

## Backlog.md copy

When `.skillgrid/config.yaml` has `ticketing.type: backlogmd`, every ADR file
has a tracker copy under `<backlog_directory>/decisions/` (resolved from
`backlog.config.yml`), written in the same edit as the artifact. Write the
decision file yourself so the Backlog id stays aligned with the ADR number
(`ADR-0024` → `decision-024`). Do **not** use `backlog decision create` for
these copies: it assigns the next free id and the pair drifts.

```
<backlog_directory>/decisions/decision-NNN - ADR-NNNN-slug.md
```

`NNN` is the ADR number zero-padded to the project's `zero_padded_ids`. Quote
`title` in the decision frontmatter — titles often contain colons.

```markdown
---
id: decision-NNN
title: "ADR-NNNN <title from the artifact H1>"
date: 'YYYY-MM-DD'
status: accepted
---

Source: `.skillgrid/projectwiki/decisions/NNNN-slug.md`

<verbatim artifact file>
```

- The artifact is the record. If the copy disagrees, recopy from the artifact.
- A new ADR → a new decision file. A superseding ADR → a new decision file
  **and** set the prior copy's `status` to `superseded`. Do not delete a copy.
- Onboarding copies any `*.md` that has no matching `decision-NNN`
  file.
- Other trackers (`gh` / `glab` / `jira`) do not get a copy.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll edit the ADR to match what we actually did" | The IRON RULE: an accepted ADR is a frozen historical record. Write a NEW ADR with `supersedes: ADR-NNNN` and explain why. |
| "The decision is obvious, skip the ADR" | Obvious *now* is not obvious to the future reader. If it's hard to reverse and surprising without context, it's an ADR. |
| "I'll write the ADRs after the fact in bulk" | The manifest runs at the end of the interview, *before* the blueprint. Bulk-writes miss the in-force set and the `supersedes` trail. |
| "I'll just bump the status on the old ADR" | Status is immutable on an accepted ADR file. A superseding file sets its own `status: accepted`; the prior file is untouched. |
| "I'll add it to the terms instead — it's faster" | The terms files are a vocabulary, not a decision record. Decisions live as `decisions/NNNN-slug.md` with a pointer row in the `## LOCKED` ADR index. |
| "I'll paste the body into ASSUMPTIONS.md" | `ASSUMPTIONS.md` stores the path. The body is the file. |
