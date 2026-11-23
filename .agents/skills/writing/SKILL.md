---
name: writing
description: "Write human-facing text with low cognitive load. Use when writing guides, READMEs, RFCs, onboarding, architecture docs, or other review-facing Markdown (doc shape); PR feedback, issue replies, review comments, and chat messages (comment voice); or the human-facing record of a finished change — PR body, changelog, release note, postmortem (record mode, from the real diff, never memory)."
license: MIT
metadata:
  author: devopstales
  version: "1.2"
  part-of: skillgrid
  based_on: gentle-ai:cognitive-doc-design 1.0 + gentle-ai:comment-writer 1.0 + skillgrid:document 1.0
  created_by: human
  related_skills: [blueprint, brainstorm, ship, slicing]
  version: "1.4"
---

## Contents
- When to Use / Pick the Mode / Doc Shape / Quick path / Details
- Checklist / Next step / Comment Voice / Record / Verification / Commands


# Writing

Three modes for the human-readable text the agent produces: **doc shape**
(structure for readers and reviewers), **comment voice** (warm, direct text
for another human), and **record** (the human-facing record of a *finished*
change — PR body, changelog, release note, postmortem — written from the real
diff and commits, never from memory). Pick the mode from the table below; the
sentence level (tone, the 21 rules) is owned by
[../_shared/rules/prose-style.md](../_shared/rules/prose-style.md) in all
modes.

**Announce at start:** "I'm using the skillgrid:writing skill for <doc
shape | comment voice | record>."

## When to Use

- **Doc shape:** creating or editing documentation people need to understand
  quickly, retain, or use during review — especially PR bodies and review
  notes, contributor or maintainer guides, architecture / workflow /
  onboarding docs, and any doc that currently feels long, dense, or hard to
  scan.
- **Comment voice:** any comment another human will read — GitHub PR or
  issue comments, review feedback and requested changes, maintainer replies,
  Slack / Discord / async project updates.
- **Record:** the human-facing record of a *finished, verified* change — the
  PR body, the changelog entry, the release note, or the postmortem — written
  from the real diff and commits. `skillgrid:ship` invokes this mode for the
  PR body / changelog / release note at close-out.

**When NOT to use:**

- Commit messages (the work-unit commit contract governs).
- Code comments (the repo's language convention governs).
- Mid-change before the work is verified — the record describes what
  shipped, not what is in flight (record mode).
- The change-spec zone (`.skillgrid/change-specs/` — artifact shapes are
  owned by `skillgrid:blueprint` and the spec-zone rules).
- The change's own `report.md` retrospective — that is `skillgrid:ship`
  Step 2 (Reflect), a different artifact; this skill owns the *human-facing*
  records that outlive the change folder.
- Planning artifacts (briefing, blueprint, tasks) — `skillgrid:brainstorm` /
  `skillgrid:blueprint` / `skillgrid:slicing` govern them.
- Marketing landing pages or brand copy (a different design surface).

## Pick the Mode

| Condition | Mode |
|---|---|
| Guide / README / RFC / onboarding / architecture doc | **Doc shape** |
| A finished change needs its PR body / changelog / release note / postmortem | **Record** |
| PR feedback / issue reply / review comment / Slack or Discord message | **Comment voice** |
| PR body *and* review comments in the same task | **Record** for the body, **Comment voice** for each comment |

## Doc Shape

### Critical Patterns

| Pattern | Rule |
|---------|------|
| Lead with the answer | Put the decision, action, or outcome first. Context comes after. |
| Progressive disclosure | Start with the happy path, then add details, edge cases, and references. |
| Chunking | Group related information into small sections. Keep flat lists short. |
| Signposting | Use headings, labels, callouts, and summaries so readers know where they are. |
| Recognition over recall | Prefer tables, checklists, examples, and templates over prose that must be remembered. |
| Review empathy | Design docs so reviewers can verify intent without reconstructing the whole story. |

### Documentation Shape

Use this default structure unless the repo already provides a stronger
template:

```markdown
# <Outcome-oriented title>

<One paragraph: what changed, who it helps, and why it matters.>

## Quick path

1. <First action>
2. <Second action>
3. <Verification or expected result>

## Details

| Topic | Decision |
|-------|----------|
| <area> | <concise explanation> |

## Checklist

- [ ] <Reader can confirm this>
- [ ] <Reader can confirm that>

## Next step

<Link or action that continues the workflow.>
```

### PR and Review Docs

When documenting a PR, reduce reviewer burnout by making the review path
explicit:

- State what to review first.
- State what is intentionally out of scope.
- Link the previous and next PR when work is chained (the chain context in
  [../_shared/work-unit-commits/README.md](../_shared/work-unit-commits/README.md)
  § Chain strategy decision).
- Keep each section focused on one decision or unit of work.
- Use checklists for acceptance criteria and verification.

## Comment Voice

### Voice Rules

| Rule | Requirement |
|------|-------------|
| Be useful fast | Start with the actionable point. Do not recap the whole PR before feedback. |
| Be warm and direct | Sound like a thoughtful teammate, not a corporate bot. |
| Keep it short | Prefer 1 to 3 short paragraphs or a tight bullet list. |
| Explain why | Give the technical reason when asking for a change. |
| Avoid pile-ons | Comment on the highest-value issue, not every tiny preference. |
| Match target context language | Write in the target context language by default: Spanish issue/thread → Spanish comment, English thread → English comment, mixed context → the target message language. If the user explicitly requests a language or tone, follow that request. |
| No em dashes | Use commas, periods, or parentheses instead. |

### Comment Formula

```text
<Direct observation or request>

<Why it matters, only if needed>

<Concrete next action>
```

### Examples

**Request change:**

```markdown
Good approach overall. I'd split this into a separate commit because it mixes validation logic with UI wiring.

That keeps the reviewer's focus narrower and makes rollback cleaner if the integration fails.
```

**Approve with a note:**

```markdown
Approved. The scope is clear and the change is well-contained.

For the next PR, add links to the previous and following PRs so the chain stays navigable.
```

**Ask for split:**

```markdown
This PR is large for one review (about 600 changed lines). Could we split it, or follow the repo's size policy if it allows one PR?

Suggested order: foundation + tests first, then integration, then docs. That gives each review a clear start and end.
```

## Record (change records)

The human-facing prose about a *finished* change. Core principle: **write
from the real diff and commits, never from memory.** The AI's account of its
own work is exactly the thing that drifts — `git log` + `git diff` + the
change folder are the sources.

### The Four Types

| Type | Source | Audience | Output |
|---|---|---|---|
| `pr` | branch commits + diff vs base | reviewers | PR title + body (to the forge via `gh`/`glab`, or chat) |
| `changelog` | the merged change | developers | entry appended to `CHANGELOG.md` (root; created on first use) |
| `release-note` | a tag / version range | end users | `.skillgrid/releases/<version>.md` (created on first use) |
| `postmortem` | the incident facts + the debug state file | the team | `.skillgrid/postmortems/<date>-<slug>.md` (created on first use) |

**Lazy creation:** `CHANGELOG.md`, `.skillgrid/releases/`, and
`.skillgrid/postmortems/` are created on first use — not at onboarding. A
repo that never ships releases never grows the directory.

**Determining the type:** if invoked by `ship` with a type, or standalone
with an argument, use it. Otherwise infer where obvious (on a feature branch
ahead of base → `pr`; just tagged a version → `release-note`), then confirm
with one question — the inferred type, presented as recommended with a
one-line why, never a neutral menu.

### Gather the Source

Collect from the real record, in this order:

```bash
# base branch: main if it exists, else master
git rev-parse --verify main
git rev-parse --abbrev-ref HEAD
git log --oneline "BASE..HEAD"
git diff --name-status "BASE...HEAD"
```

1. **The change set** — the real diff, never memory (the commands above).
2. **The change folder** (the "why"): `briefing.md` (goal), `blueprint.md`
   (approach), `tasks.md` (what shipped), `report.md` → `## Gate Decision`
   (verdict + evidence). If already archived, read it from
   `.skillgrid/change-archive/YYYY-MM-DD-<topic>/`.
3. **The decision record:** the ADR files under
   `.skillgrid/projectwiki/decisions/*.md` this change touched (the
   in-force table in `ASSUMPTIONS.md` is the index). For the product why,
   read `.skillgrid/projectwiki/reference/00-prd.md` when the change touches
   scope.
4. **Postmortem only** — the incident facts: observed vs expected, the
   reproduction, the root cause. Read the change's debug state file
   (`.skillgrid/work/debug/<date>-<slug>/state.md`) if present; ask the user
   for any fact it cannot derive (when it was noticed, blast radius, user
   impact, timeline).

### Write + Deliver

Read the matching template from [templates/](templates/) and fill it from the
gathered material. The rules, all types:

- **Every claim traces to a source.** A timeline entry names its commit. A
  "fixed" bullet names the file. A cause names the evidence. If you cannot
  name the source, do not write the claim.
- **Audience pitch.** `pr` is for the person who must approve the diff — lead
  with what changed and why, then the evidence. `changelog` is for developers
  integrating the next version — user-visible changes only, grouped by
  Added/Changed/Fixed/Breaking. `release-note` is for end users — plain
  language, no internal names, lead with what they can now do. `postmortem`
  is for the team that must not repeat this — blameless, lead with the impact,
  then the timeline, then the root cause with its evidence, then the actions.
- **No marketing.** No "excited to announce", no "seamless", no "robust".
  State what is true.
- **The gate is named, not implied.** For `pr` and `release-note`, the
  `report.md` → `## Gate Decision` verdict is part of the record.
- **Merge danger is explicit, for `pr`.** The body carries a `## Merge
  Danger` section: one-way vs two-way door plus a one-word blast radius.

Deliver per the type: `pr` → the forge CLI (`gh pr create --body` /
`glab mr create -d`) or the repo's PR template if one exists, else chat;
`changelog` → append under the correct version heading (Keep a Changelog
format); `release-note` → `.skillgrid/releases/<version>.md`; `postmortem` →
`.skillgrid/postmortems/<date>-<slug>.md`. Commit the written file per
[../_shared/work-unit-commits/README.md](../_shared/work-unit-commits/README.md)
(`docs:` prefix for changelog / release-note / postmortem).

### Anti-Patterns (record mode)

| Anti-pattern | Symptom | Fix |
|---|---|---|
| Written from memory | A timeline entry or "fixed" bullet that names no commit, file, or change-folder source | `git log` + `git diff` + the change folder are the record — name the source or cut the claim. |
| Generic changelog | "misc improvements" / "various fixes" with nothing named | Every bullet names what changed and where, with the change topic or ticket ID as source. |
| Marketing release note | "seamless", "robust", or internal names for an end-user audience | Plain language, what the user can now do, is the whole job. |
| Postmortem without a timeline | Only the root cause, no noticed/resolved gap | The timeline is how the team sees what detection was slow — every line names its source. |
| PR body omits the gate | No `report.md` verdict in the record | The gate result is part of the record — name it. |
| Written before the record was read | Document drafted before the change folder (or its archived copy) was read | Read the source first — a document written before the record is an invention. |

## Verification

**Doc shape:**

- [ ] The first section states the outcome, not the background.
- [ ] A reviewer can find "what to review first" and "what is out of scope" without reading the whole doc.
- [ ] No section exceeds a page of dense prose without a table, list, or heading break.
- [ ] Every claim that is a closure statement names its evidence (per [../_shared/rules/evidence-claims.md](../_shared/rules/evidence-claims.md)).

**Comment voice:**

- [ ] The first sentence is the actionable point, not a summary.
- [ ] Every requested change carries its technical reason.
- [ ] The comment fits 1–3 short paragraphs or a tight bullet list.
- [ ] Language matches the target thread unless the user overrode it.

**Record:**

- [ ] The document type was determined (from the invocation, an argument, or one confirmed question).
- [ ] The source material was gathered from git (`git log` / `git diff`) and the change folder — not from memory.
- [ ] Every claim traces to a named commit, file, or change-folder fact (spot-check at least three claims against their source).
- [ ] The document was written to the correct output path per the type table.
- [ ] The `report.md` → `## Gate Decision` verdict is named in the record (pr / release-note).
- [ ] The `## Merge Danger` section is present in the PR body.
- [ ] The file was committed with a `docs:` conventional subject (changelog / release-note / postmortem).

## Commands

```bash
# Check markdown files changed in the current branch
git diff --name-only -- '*.md'

# Inspect PR changed-line count for cognitive load
gh pr view <PR_NUMBER> --json additions,deletions,changedFiles

# Inspect a PR before writing review feedback
gh pr view <PR_NUMBER> --json title,body,additions,deletions,changedFiles

# Record mode — gather the change set (base: main if it exists, else master)
git rev-parse --verify main
git rev-parse --abbrev-ref HEAD
git log --oneline "BASE..HEAD"
git diff --name-status "BASE...HEAD"
```

## References

- [../skill-creator/references/rationalizations.md](../skill-creator/references/rationalizations.md) — the authoring rationalizations (script extraction, size, spine sections); load when you or the user push back on this skill's shape
