# Skill Anatomy (format contract)

The structural contract every `skillgrid` skill's `SKILL.md` obeys. Cite this
for the *shape*; `skill-creator` owns the *process* (the four phases, the
10-rule audit, description optimization) that produces a file to this shape.

## Directory shape

```
skill-name/
├── SKILL.md              ← required: frontmatter + instructions (< 500 lines)
├── scripts/              ← optional: executable code (no token cost when loaded)
│   └── <name>.mjs
├── references/           ← optional: docs loaded on demand
│   └── <topic>.md
└── assets/               ← optional: files used in output (templates, icons)
```

## Frontmatter (exact field set and order)

```yaml
---
name: skill-name
description: What it does (third person). Use when {trigger}.
license: MIT
metadata:
  author: name
  version: "1.0"
  part-of: skillgrid
---
```

- **`name`** is kebab-case and equals the directory name.
- **`description`** is the only thing the model sees before loading the body:
  third person, trigger-specific, no workflow steps, ≤ 1024 chars, slightly
  pushy (the model under-triggers).
- **`metadata.version`** is quoted semver, bumped on every behavior-changing
  edit. **`based_on`** (when present) records the upstream skills this was
  folded from — a citation, never edited after the fact.
- **`part-of: skillgrid`** is constant across the repo.

## Body rules

- **`SKILL.md` is a routing table, not a manual.** It carries the spine
  (frontmatter, the mode/branch table, shared gates, the step walk) and points
  to `references/` for branch-specific detail. Under ~500 lines.
- **One concern per reference file.** One level deep — every reference links
  directly from `SKILL.md`, no `SKILL.md → a.md → b.md` chains.
- **Scripts cost no tokens.** A 200-line script is cheaper than the prose that
  describes it. Deterministic steps are scripts; judgment steps stay prose
  (see `_shared/craft/deterministic-boundary.md`).
- **The fixed section order is load-bearing.** The agent walks the spine top to
  bottom; moving a section changes when a constraint bites. Keep the order a
  sibling uses — do not invent a new arrangement.

## Cross-reference rules

- Reference by canonical `skillgrid:<name>` form, the name as the skill exists
  *now*. A renamed or folded skill is cited only in `based_on` and the merge
  note, never as a live reference.
- A rename is a whole-skill sweep: every occurrence in `SKILL.md` *and* every
  `references/*.md` changes in the same edit.
- `_shared/` paths must resolve in this repo's `_shared/`. A `_shared/` path
  that resolves only elsewhere is a drift signal — copy the file here or
  reword the pointer.
