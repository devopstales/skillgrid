---
name: creating-skills
description: Create or improve an agent skill (SKILL.md plus optional scripts/references) from real expertise. Use when the user asks to make, write, scaffold, refine, or audit a skill; escape skill hell; apply the writing-great-skills checklist (trigger, structure, steering, pruning); or validate skill structure.
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
---

# Creating skills

A skill is a directory with a `SKILL.md` (YAML frontmatter + Markdown), optionally plus `scripts/`, `references/`, and `assets/`. Ground every skill in real expertise — steps that worked, corrections needed, project facts. General knowledge alone produces vague, worthless skills.

Rubric for great skills (trigger → structure → steering → pruning). Use it when writing **or** auditing. Details and examples: [`references/checklist.md`](references/checklist.md). Frontmatter rules: [`references/anatomy.md`](references/anatomy.md). Content patterns: [`references/patterns.md`](references/patterns.md).

## Workflow

Progress:
- [ ] Step 1: Gather expertise
- [ ] Step 2: Decide the trigger (user-invoked vs model-invoked)
- [ ] Step 3: Name, scope, and location
- [ ] Step 4: Structure as steps + reference; keep SKILL.md minimal
- [ ] Step 5: Steer — leading words and enough leg work per step
- [ ] Step 6: Write frontmatter and body
- [ ] Step 7: Prune — DRY, sediment, no-ops, deletion tests
- [ ] Step 8: Validate — `scripts/validate_skill.sh <skill-dir>`
- [ ] Step 9: Test on a real task; fold every correction into Gotchas

### Step 1: Gather expertise

Extract from a recent or repeating task:

- Steps that worked (successful sequence)
- Corrections ("use X, not Y", "always check Z first")
- Concrete formats in and out
- Project-specific facts, conventions, edge cases

Otherwise synthesize from runbooks, config, review comments, fix commits, failure reports. Ask which material to draw from — do not invent.

### Step 2: Decide the trigger

| Mode | Mechanism | Cost | Prefer when |
|------|-----------|------|-------------|
| **Model-invoked** | `description` stays in agent context as a context pointer | Context load every request + unpredictability (agent may skip) | Agent (or another skill) must discover/fire it |
| **User-invoked** | `disable-model-invocation: true` — description is human-facing only | Cognitive load on the user (must remember to invoke) | You want deterministic control; avoid evals for "did it fire?" |

Model-invoked is not "better" — it is more flexible and more expensive. Prefer user-invoked when predictability matters more than autonomous discovery. See checklist for harness notes.

### Step 3: Name, scope, and location

One coherent unit of work — like a function. Query + format results: one skill. Query + database admin: two. Prefer a purpose word in the name: `creating-skills`, `deploy-staging`.

Default: project's `.agents/skills/` (or `~/.agents/skills/` if it should travel). Match naming/structure of sibling skills first.

### Step 4: Structure — steps + reference; minimal SKILL.md

Compose from two units:

1. **Steps** — the procedure the agent walks
2. **Reference** — supporting material those steps need (definitions, templates)

Skills may be steps-only, reference-only, or both. Keep **SKILL.md as small as possible** (maintainability + tokens).

**Branches:** if reference is only needed on one path, hide it behind a **context pointer** ("If updating the glossary, read the matching file under `references/`"). Always-needed reference for a single-branch skill can stay in SKILL.md. Bundled code → `scripts/` (run it); templates/data → `assets/`. References one level deep from SKILL.md.

### Step 5: Steer — leading words and leg work

**Leading words** pack meaning into a short phrase the agent will echo in thinking and output (e.g. `vertical slice`, not a long "don't code layer by layer…" essay). Repeat the phrase consistently through the skill. If the agent ignores you, strengthen or replace the leading word — watch reasoning traces for adoption.

**Leg work:** when a step under-invests because a later goal is visible (classic: skim clarifying questions, rush the plan), split into a focused skill so the agent only sees the current phase. Not always required — use when you need extra depth on one step.

### Step 6: Write frontmatter and body

Minimal frontmatter (full rules in anatomy):

```markdown
---
name: <dir-name>
description: <what it does> + "Use when <triggers and keywords>."
# user-invoked only:
# disable-model-invocation: true
---
```

Body rules of thumb (patterns for templates/checklists):

- Add what the agent lacks; omit what it knows
- Default + escape hatch, not a menu of equal options
- Procedures over one-off answers; calibrate control per section
- Gotchas = concrete corrections; working example for non-obvious formats
- Target a tight SKILL.md — move branch-only detail out

### Step 7: Prune

Before shipping, run a pruning pass:

- **DRY / single source of truth** — no duplicated reference or restated steps
- **Sediment** — stale or drive-by additions; delete or move to the right branch
- **No-ops** — instructions the agent would follow anyway; deletion-test each paragraph
- **Massive body** — usually a symptom of the above, not a goal

### Step 8: Validate

Run `scripts/validate_skill.sh <path-to-skill-dir>`. Fix every failure; re-run until clean.

### Step 9: Test and iterate

Run a real task that should use the skill. Read the actual execution (and reasoning), not only the final output. Add every mistake to Gotchas. Cut instructions the agent followed without help. One execute→revise loop helps a lot; hard domains need more.

## Gotchas

- `name` must match the directory exactly — uppercase/`My-Skill` is invalid.
- Description without "when to use" mis-triggers or never triggers. For model-invoked skills the description is the always-loaded context pointer — keep it sharp.
- Don't create a skill the agent already handles well — context cost with no gain.
- Every model-invoked skill adds permanent context load; pile-up is skill hell.
- Branch-only templates left in SKILL.md bloat every invocation — context-pointer them out.
- Scripts must be self-contained (document deps) — the agent won't know your local setup.
- Never commit secrets into examples or references.
