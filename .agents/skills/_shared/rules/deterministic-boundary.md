# Deterministic Boundary (the script/prose split)

The single most expensive mistake in skill authoring: encoding deterministic
logic in prose that a non-deterministic agent re-interprets on every invocation.
This file is the rule. The process for applying it lives in the
`skillgrid:skill-write` skill.

## The Rule

**If a step produces the same output every time it runs with identical input,
it is a script, not a skill line.** Skills decide *what* to do; tools do *it*.

A skill line is a **decision boundary** — a point where the agent must judge,
classify, or choose. A script line is a **deterministic execution** — a point
where the correct action is fixed by the input.

## The Classification Test

For every step in a draft SKILL.md, ask:

> "If I ran this step twice with identical input, would the correct output be
> identical both times?"

- **Yes** → script candidate. Extract it to `scripts/<name>.mjs` (or bash for
  git plumbing). The skill line becomes: "Run `node scripts/<name>.mjs`; report
  the output."
- **No** → judgment. Keep it in the skill. The agent must see the input and
  decide (classify a diff, choose an approach, write a review comment).

## Why It Matters (token economics)

Every skill line is read by the agent on every invocation. A 50-line prose
description of "how to check for state drift" costs ~2000 input tokens per
invocation, and the agent must then *interpret* those instructions, which costs
output tokens and carries a risk of misinterpretation. A one-line "run the
script, report the verdict" costs ~30 input tokens, and the script's output is
deterministic — no interpretation risk.

This is the same insight as caching: generate the logic once (the script),
serve it many times (the script's output), instead of regenerating it on every
request (the agent re-reading and re-interpreting the prose).

## Examples

### Extract (deterministic → script)

```markdown
# BAD — prose describing a deterministic check
If the state file has `current_phase: spec` but the spec directory contains
`blueprint.md`, there is a drift. Report it as a WARNING. If the state file
is missing or unparseable, report the parse error as a WARNING.

# GOOD — one line calling the script
Run `node .agents/skills/qa/scripts/state-drift-check.mjs` from the project
root. Exit 0 = no drift. Exit 1 = drift (report the table it prints).
Exit 2 = parse error (report the error it prints).
```

### Keep (judgment → prose)

```markdown
# KEEP — the agent must judge
Does this diff look atomic? A single concern, independently revertable?
If it bundles two unrelated changes, split it.

# KEEP — the agent must choose
Which test layer fits this behavior? Highest available layer that exercises
the behavior end-to-end, degrading if the layer is unavailable.
```

### Boundary (mixed → split)

```markdown
# MIXED — split into script + judgment
# The script part (deterministic): parse the Trivy JSON, filter by severity,
# count findings per category.
# The judgment part (agent): classify a "suspicious but unconfirmed" OWASP
# pattern — is it CRITICAL or SUGGESTION? That needs human context.

Run `node scripts/trivy-parse.mjs` → JSON summary (counts by severity).
For any finding marked "needs-review" in the JSON, classify it:
CRITICAL (secret committed, CVE with fix) or SUGGESTION (pattern needs
human confirmation). Record your classification in the report.
```

## Where Scripts Live

Per [sdd-structure.md](sdd-structure.md) and [skill-anatomy.md](skill-anatomy.md):

- Skill-specific scripts → `.agents/skills/<skill>/scripts/`
- Project-level CI/drift guards → `scripts/` (repo root)
- Data files (budgets, thresholds) → alongside the script that consumes them,
  or in `_shared/` when multiple scripts read the same file

## Citing This Rule

Other skills reference this convention by name, not by restating it:

```markdown
Per `_shared/conventions/deterministic-boundary.md`: deterministic logic
belongs in a script, not in skill prose.
```

`skillgrid:skill-write` is the invocable process that applies this rule when
creating or refactoring a skill. `skillgrid:requesting-code-review` flags
violations during review. The `skill-size-budget` ratchet measures the
consequence (bloat) but does not detect the cause (prose-that-should-be-script).
