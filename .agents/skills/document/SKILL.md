---
name: document
description: Write the human-facing record of a change — PR body, changelog entry, release note, or postmortem — from the real diff and commits, never from memory. Use at ship time (via skillgrid:ship) or standalone.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# Document

**Announce at start:** "I'm using the skillgrid:document skill to write the human-facing record of this change."

**Your role:** the technical writer who writes from the record, not from imagination, and for the reader, not the author. Every sentence traces to something that actually happened — a commit, a diff line, a named fact from the change folder. You never invent a timeline entry, a cause, or a change that isn't in the source.

## Overview

Generates the human-facing prose about a finished change: the PR body, the changelog entry, the release note, or the postmortem. It is the tail's voice — `skillgrid:ship` invokes it for the PR body when the user chooses the PR option, and offers the other types where applicable. It writes prose only: never code, tests, specs, or planning artifacts.

**Core principle:** write from the real diff and commits, never from memory. The AI's memory of what it changed is not a source. `git log` + `git diff` + the change folder are.

## When to Use

- At ship time: `skillgrid:ship` invokes this skill for the PR body (type `pr`) when the user chooses "Push and create a Pull Request".
- Standalone, after a change has shipped: "write the changelog entry", "release notes for v1.4.0", "postmortem for the auth outage".
- When a finished change needs its human record and the planning artifacts (briefing, blueprint, qa-report) exist to argue from.

**When NOT to use:** mid-change before the work is verified (the record describes what shipped, not what is in flight). For planning artifacts (briefing, blueprint, tasks) — those are `skillgrid:brainstorming` / `skillgrid:writing-blueprints` / `skillgrid:slicing`. For the terminal retrospective (`report.md`) — that is `skillgrid:reflect`, which owns the close-out document; this skill owns the *human-facing* records that outlive the change folder.

## The Four Types

| Type | Source | Audience | Output |
|---|---|---|---|
| `pr` | branch commits + diff vs base | reviewers | PR title + body (to the forge via `gh`/`glab`, or chat) |
| `changelog` | the merged change | developers | entry appended to `CHANGELOG.md` (root; created on first use) |
| `release-note` | a tag / version range | end users | `.skillgrid/releases/<version>.md` (created on first use) |
| `postmortem` | the incident facts + the debug state file | the team | `.skillgrid/postmortems/<date>-<slug>.md` (created on first use)

**Lazy creation:** `CHANGELOG.md`, `.skillgrid/releases/`, and `.skillgrid/postmortems/` are created by this skill on first use — not at onboarding. A repo that never ships releases never grows the directory.

## The Process

### Step 1: Determine the type

If invoked by `ship` with a type, use it. If invoked standalone with an argument (`pr` | `changelog` | `release-note` | `postmortem`), use it. Otherwise infer where obvious (on a feature branch ahead of base → `pr`; just tagged a version → `release-note`), then confirm with one question. Present the inferred type as recommended (one-line why); never a neutral menu. This is the **only** question this skill asks (plus, for a postmortem, the incident facts it cannot read from git — see Step 2).

### Step 2: Gather the source material

Collect from the real record, in this order:

1. **The change set** (the real diff, never memory):
   ```bash
   # base branch: main if it exists, else master
   git rev-parse --verify main
   git rev-parse --abbrev-ref HEAD
   git log --oneline "BASE..HEAD"
   git diff --name-status "BASE...HEAD"
   ```
2. **The change folder** (the "why"): the change's `briefing.md` (goal + falsifiable requirements), `blueprint.md` (approach + Build shape + Tier + Status), `tasks.md` (what shipped), `qa-report.md` (gate verdict + evidence). If the folder was already archived, read it from `.skillgrid/archive/YYYY-MM-DD-<topic>/`.
3. **The decision record:** the 3 most recently modified ADRs in `.skillgrid/adr/` that this change touched (for the "why this design" a reviewer or future reader needs).
4. **Postmortem only** — the incident facts: the observed vs expected behavior, the reproduction, the root cause. Read the change's debug state file (`.skillgrid/sdd/debug/<date>-<slug>/state.md`) if present; ask the user for any fact it cannot derive (when it was noticed, blast radius, user impact, timeline).

### Step 3: Write the document

Read the matching template from this skill's `templates/` directory and fill it from the gathered material. Rules:

- **Every claim traces to a source.** A timeline entry names its commit. A "fixed" bullet names the file. A cause names the evidence. If you cannot name the source, do not write the claim.
- **Audience pitch.** `pr` is for the person who must approve the diff — lead with what changed and why, then the evidence. `changelog` is for developers integrating the next version — bullet the user-visible changes, group by Added/Changed/Fixed/Breaking. `release-note` is for end users — plain language, no internal names, lead with what they can now do. `postmortem` is for the team that must not repeat this — lead with the impact, then the timeline, then the root cause, then the actions.
- **No marketing.** No "excited to announce", no "seamless", no "robust". State what is true.
- **The gate is named, not implied.** For `pr` and `release-note`, the `qa-report.md` verdict (PASS / WAIVED / CONCERNS + override) is part of the record.

### Step 4: Deliver

- `pr` → if the user is at the ship PR option, hand the body to the forge CLI (`gh pr create --body` / `glab mr create -d`) or fill the repo's PR template if one exists; otherwise present it in chat.
- `changelog` → append the entry to `CHANGELOG.md` under the correct version heading (create the file with a `# Changelog` header on first use; Keep a Changelog format).
- `release-note` → write `.skillgrid/releases/<version>.md`.
- `postmortem` → write `.skillgrid/postmortems/<date>-<slug>.md`.

Commit the written file with `skillgrid:work-unit-commits` (docs zone; the `docs:` prefix for changelog/release-note/postmortem commits).

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I remember what I changed, I'll write it from memory" | Memory is not a source. The AI's account of its own work is exactly the thing that drifts. `git log` + `git diff` + the change folder are the record. |
| "The changelog can be generic — 'bug fixes and improvements'" | A changelog that names nothing helps no one integrating the version. Every bullet names what changed and where. |
| "The postmortem can skip the timeline, the root cause is what matters" | The timeline is how the team sees what detection was slow and what response was fast. Skip it and you lose the two lessons that are not about the bug. |
| "I'll add a little marketing polish to the release note" | End users read "seamless" and learn nothing. Plain language, what they can now do, is the whole job. |
| "The PR body can be short, the diff speaks for itself" | Reviewers do not all read every diff line. Lead with what changed and why, name the gate verdict, point at the evidence. The diff is the appendix, not the cover. |

## Red Flags

- A claim in the document that names no commit, file, or change-folder source — an invented line.
- A changelog entry with "misc improvements" or "various fixes" and nothing named.
- A release note that uses internal names (ticket IDs, module names) for an end-user audience.
- A postmortem without a timeline or without the root cause stated with its evidence.
- The PR body omitting the `qa-report.md` verdict — the gate result is part of the record.
- Writing the document before the change folder (or its archived copy) has been read.

## Verification

- [ ] The document type was determined (from the invocation, an argument, or one confirmed question).
- [ ] The source material was gathered from git (`git log` / `git diff`) and the change folder (briefing, blueprint, tasks.md, qa-report.md) — not from memory.
- [ ] Every claim in the document traces to a named commit, file, or change-folder fact (spot-check at least three claims against their source).
- [ ] The document was written to the correct output path per the type table (forge PR body / `CHANGELOG.md` / `.skillgrid/releases/<version>.md` / `.skillgrid/postmortems/<date>-<slug>.md`).
- [ ] The `qa-report.md` verdict is named in the record (pr / release-note).
- [ ] The file was committed with a `docs:` conventional subject (changelog / release-note / postmortem).
