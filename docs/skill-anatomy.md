# Skill anatomy

The contract every skill under `.agents/skills/` follows. Skills are not prose — they are behavior-shaping code. A change to a skill's structure or voice is a change to how the agent acts, so it earns the same care as a code change: make it for a reason, and prefer the smallest change that fixes the behavior.

## Frontmatter

Every invokable skill opens with a YAML frontmatter block:

```yaml
---
name: skill-name          # must match the directory name
description: "..."        # see Description rules below
---
# based on <source>        # provenance comment (optional but conventional)
```

`name` must equal the directory name. The `# based on <source>` comment records where the skill came from (e.g. `superpowers:test-driven-development`, `BMAD:bmad-deep-recon`, `skillgrid-v2:mnemonic`). Original skillgrid work may omit it. **Do not put provenance inside the `description:` field** — it is a separate comment after the frontmatter close.

## Description rules

The `description` is what the harness sees to decide *when* to load the skill. It is always in context, so it costs tokens on every session.

- **Trigger-based.** Start with **what the skill does**, then one or more clear **"Use when"** trigger conditions.
- **No process steps.** The description must not summarize the workflow. If a reader could follow the description and skip the body, it has leaked process into the frontmatter — move it into the body.
- **Length.** Keep it short (a sentence or two, well under 1024 chars). The longest descriptions in the pack should be the ones doing the most routing.
- **Third person, present tense.** "Investigates a question…" not "You should use this…".

Good: `Use when a change has passed qa and review and you need to integrate it to its base branch and close its change folder.`
Bad: `…verify tests on the integrated tree, merge / open a PR / keep the branch, then mechanically move the change folder to .skillgrid/archive/. No release mechanics, no docs check.` (that's the body, not the trigger).

## Body structure

The conventional section order. Not every skill needs every section, but when present they appear in this order:

1. **Announcement** — a line the agent says at start: `"I'm using the skillgrid:<name> skill to <intent>."`
2. **Overview / When to use** — what the skill owns, and the boundaries (what it does *not* do).
3. **Config** — for skills that read `.skillgrid/config.yaml`, which keys and their defaults.
4. **The process** — the steps, in order. Gate-driven skills mark hard gates explicitly (e.g. an **Iron law** or a `## Gates` block with verdict states).
5. **`## Common Rationalizations`** — a two-column table: the excuse an agent will make, and the reality that defeats it. This is the anti-rationalization layer; it is what makes a skill resist "just this once."
6. **`## Red Flags`** — a short **Never:** list of the failure modes that mean "start over."
7. **`## References`** — links to the skill's own `templates/`, `references/`, and to other skills.

### Phase-boundary contract (tail skills only)

Skills that are a stage in the pipeline declare, near the top:

```
prev-phase: [x] · next-phase: [y] · artifact: <name>
```

and end with a **Return Envelope** — a fixed Markdown template the skill's final output must match. This makes phase boundaries machine-checkable: the next phase knows exactly what artifact to read.

## Cross-referencing

- **Reference shared material, never duplicate it.** Conventions, the TDD cycle, and the threat matrix live in `_shared/`. A skill links to them; it does not copy them.
- **Reference other skills by relative path** when a link helps navigation: `[`../qa/SKILL.md`](../qa/SKILL.md)`. In prose, reference by skill name (`skillgrid:qa`).
- **Stages load general skills.** A workflow stage does not re-implement a capability — it names the general skill that owns it.
- **Every relative link must resolve.** A dead `../` link is a broken contract; verify before committing.

## File layout

```
<skill-name>/
├── SKILL.md            # the skill (the only required file)
├── templates/          # artifact scaffolds the skill fills in (optional)
├── references/         # in-skill reference docs (optional)
└── scripts/            # helper scripts the skill invokes (optional)
```

Keep authoring residue out of the directory: creation logs and validation/pressure-test prompts belong in a PR or a dev branch, not the shipped skill. `_shared/` is the one exception to "everything under a skill dir" — it is a non-invokable library consumed by the others.

## Sizing

- A skill is the *smallest* unit that changes agent behavior for one job. If a skill covers two unrelated jobs, split it.
- If a skill is growing past ~250 lines, ask whether part of it is a `references/` doc that should be loaded on demand rather than always in context.
- Token-conscious: every section must justify its inclusion. If removing it would not change agent behavior, remove it.
