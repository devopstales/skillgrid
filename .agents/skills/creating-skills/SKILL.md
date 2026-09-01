---
name: creating-skills
description: Create a new agent skill (a SKILL.md plus optional scripts/references) from a real task, domain expertise, or user request. Use when the user asks to make, write, scaffold, or refine a skill; when a repeating workflow, project convention, or repeated correction should become a skill; or when validating an existing skill's structure.
---

# Creating skills

A skill is a directory with a `SKILL.md` (YAML frontmatter + Markdown instructions), optionally plus `scripts/`, `references/`, and `assets/`. Ground every skill in real expertise: steps that actually worked, corrections that were needed, project-specific facts. A skill generated from general knowledge alone is vague and worthless.

## Workflow

Progress:
- [ ] Step 1: Gather expertise — find/collect the real material (see below)
- [ ] Step 2: Name and scope the skill
- [ ] Step 3: Write frontmatter (check name rules in `references/anatomy.md`)
- [ ] Step 4: Write the body — concise, calibrated, with working examples
- [ ] Step 5: Split into reference files if the body grows large
- [ ] Step 6: Validate — run `scripts/validate_skill.sh <skill-dir>` and fix all failures
- [ ] Step 7: Test against a real task; add every correction the agent needed to a Gotchas section

### Step 1: Gather expertise

If the user is describing a task they recently did, or one that repeats, extract:

- Steps that worked (the successful sequence)
- Corrections they made ("use X, not Y", "always check Z first")
- Concrete formats in and out
- Project-specific facts, conventions, edge cases

Otherwise synthesize from project artifacts: runbooks, config files, code review comments, commit history of fixes, failure reports. Ask the user which material to draw from rather than inventing content.

### Step 2: Name and scope

A skill is a coherent unit of work — like a function. Query + format results: one skill. Query + database administration: two skills. One word of the skill's purpose in the name works well: `creating-skills`, `data-pipeline-operations`, `deploy-staging`.

Default location: the project's `.agents/skills/` directory (or global `~/.agents/skills/` if it should travel to every project). Always check how OTHER skills in that directory are named and structured first — match those conventions.

### Step 3: Frontmatter

Minimal required frontmatter:

```markdown
---
name: <dir-name>
description: <what it does> + "Use when <trigger conditions and keywords>."
---
```

`name` must match the directory name exactly and obey the rules in `references/anatomy.md`. `description` must state both what the skill does and when to use it, and include keywords a user would actually type. One-line test: if the description is "helps with X", it fails.

### Step 4: Body

Rules of thumb (details and patterns in `references/patterns.md`):

- Add what the agent lacks, omit what it knows. No "HTTP is a protocol..." — straight to `use requests, set timeout=30`.
- Provide a default with an escape hatch, not a menu of equal options.
- Favor procedures over one-off answers.
- Calibrate per-section: prescriptive for fragile/sequential steps ("run exactly this command"), permissive for judgment work ("explain why, not how").
- Include a Gotchas section listing concrete corrections, e.g. "table X uses soft deletes — always `WHERE deleted_at IS NULL`".
- Include a working example (input → output) for non-obvious output formats.

Keep the body under ~500 lines / ~5000 tokens of actual instructions.

### Step 5: Progressive disclosure

If the body wants reference material > 50 lines (full API details, long templates, edge-case catalog), move it to `references/<topic>.md` and tell the agent WHEN to load it — e.g. "Read the API-errors reference if the API returns a non-200."

Bundled code goes in `scripts/` (run it, don't explain it), static templates/data in `assets/`. Keep references one level deep from `SKILL.md`.

### Step 6: Validate

Run `scripts/validate_skill.sh <path-to-skill-dir>`. It checks frontmatter presence, name format/directory match, description length, and line budget. If it exits non-zero, fix everything it reports and re-run until clean.

### Step 7: Test and iterate

Ask the user (or run yourself) a real task that should trigger this skill. Read the ACTUAL execution, not just the final output. For every mistake the agent makes, add the correction to the Gotchas section — this is the single most direct iteration loop. One round of execute-then-revise materially improves quality; complex domains need more rounds. Cut any instruction that the agent was going to follow anyway.

## Gotchas

- `name` must match the directory name exactly — the validator fails on mismatch, and `name: My-Skill` (uppercase) is an invalid skill entirely.
- Description that lacks "when to use" triggers the skill on the wrong prompts or not at all. "Use when…" is load-bearing, not decorative.
- Don't create a skill where none is needed. If the agent handles the task well without one, the skill adds context cost, not value.
- Overly comprehensive bodies make agents lose focus on irrelevant instructions. If you're covering every edge case, most belong in the agent's own judgment.
- Skills in `scripts/` must be self-contained (document dependencies in the skill body) — the agent won't know your local setup.
- Never commit secrets into a skill's code examples or reference files.
