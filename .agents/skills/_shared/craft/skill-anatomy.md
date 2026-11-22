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
├── templates/            ← optional: fill-in skeletons copied into the
│   └── <type>-template.md   user's workspace (one per output type)
└── assets/               ← optional: static files used in output but not
                            copied (icons, reference data)
```

**`templates/` vs `assets/`:** a file that is *copied* into the user's
workspace (a fill-in skeleton the skill fills and writes out) goes in
`templates/`, named `<type>-template.md`. A file that is *read* by the skill
or *referenced* in output but not copied (icons, reference data,
pre-built fragments) goes in `assets/`. If a file is neither copied nor
read but is documentation the agent loads on demand, it belongs in
`references/`, not `assets/`.

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
- **`metadata.version`** is quoted `major.minor`, bumped on every
  behavior-changing edit. (Repo convention is 2-part; the validator enforces
  `MAJOR.MINOR`.) **`based_on`** (when present) records the upstream skills
  this was folded from — a citation, never edited after the fact.
- **`part-of: skillgrid`** is constant across the repo.
- **`_shared/` is exempt** from `name` == directory (it is a non-skill bundle)
  and from `license`; it still carries a quoted `metadata.version`.

## Body rules

- **`SKILL.md` is a routing table, not a manual** — progressive disclosure: the
  model sees only the description before loading, then walks the spine
  (frontmatter, the mode/branch table, shared gates, the step walk) and follows
  one-line pointers to `references/` for branch-specific detail, on demand.
  Under ~500 lines. The load order is what the routing table buys: single-branch
  material never pays the token cost on branches that don't use it.
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
- **MCP tool references** use fully-qualified `ServerName:tool_name`
  form (e.g. `Mnemonic:mem_save`, `GitHub:create_issue`). A bare
  `tool_name` is a drift signal when the skill may run with multiple
  MCP servers loaded.
