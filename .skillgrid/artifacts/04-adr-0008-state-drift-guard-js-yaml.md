# State drift guard: Node.js + `yaml` package, read-only verifier

---
status: "accepted"
supersedes: none
date: 2026-09-24
---

## Context and Problem Statement

`.skillgrid/state.yaml` is the dynamic project state — `pipeline.current_phase`,
`pipeline.current_change`, `progress.completed_changes` — hand-edited by ten
pipeline skills (brainstorming through reflect) at every phase transition. It
is the resume pointer that `skillgrid:resume` and `skillgrid:using-skillgrid`
read first. But nothing verifies that it agrees with the spec-zone artifacts
it summarizes. A skill that forgets to update it (or updates it wrong)
produces a silent pointer that misleads resume, and `resume` already admits
it must "trust the spec-zone artifacts (the deeper truth) and note the drift"
when they disagree.

We need a **drift guard**: a read-only checker that compares `state.yaml`
against the spec zone and reports a named drift verdict. It does not own the
write (pipeline skills keep writing `state.yaml` directly); it owns the
verification.

The repo currently has **no YAML library** — `package.json` has zero
dependencies, no `node_modules`, and nothing in `scripts/` parses YAML today.
The guard is the first Node script that must parse the committed
`state.yaml`.

## Considered Options

- **A. Node.js + `yaml` package** — add the `yaml` npm package (modern,
  zero-dep, MIT, ESM+CJS); write the guard as `scripts/state-drift-check.mjs`.
  A real YAML parser handles the schema correctly (quotes, comments, nested
  keys) and is robust to the small schema extensions `sdd-structure.md` will
  accumulate. Adds one dependency.
- **B. Node.js + regex** — parse the flat `state.yaml` with a small regex
  (the schema is fixed and small: `pipeline.`, `progress.`,
  `constraints_ref`, `notes`). No dependency. Fragile to schema evolution
  (nested keys, quoted values, comments) and to the one-line edits the
  pipeline skills make.
- **C. Bash + `python3 -c "import yaml"`** — shell out to Python for the
  parse. No npm dependency, but adds a Python + PyYAML runtime requirement
  to a repo whose tooling is Node + Go.

## Decision Outcome

Chosen option: **A. Node.js + `yaml` package**, because the guard is the
first Node script that parses a committed YAML artifact, and a real parser
beats regex for a schema we own and will extend. The `yaml` package is
zero-dep (no transitive dependencies), MIT-licensed, and works in both ESM
and CJS — it fits the repo's existing `type: commonjs` package.json without
forcing an ESM conversion. The cost is one new dependency, which this ADR
satisfies for the "no new dependencies without an ADR" rule.

Option B (regex) was the lazy path and was rejected: the schema is small
*today*, but `sdd-structure.md` is the single source of truth for it and
grows (workstreams, milestones, per-phase overrides will add nested keys).
A regex parser that silently mis-parses a quoted value or a comment line
produces false drift reports, which erode trust in the guard.

Option C (Python) was rejected: the repo's scripting surface is Node
(`scripts/*.mjs` for the new guard, `package.json` for the dep) and Go
(`skillgrid-cli`); adding a Python + PyYAML runtime requirement to a
read-only checker is a larger footprint than the one npm package.

### Implementation

- `package.json` — adds `"yaml": "^2.x"` to `dependencies`.
- `scripts/state-drift-check.mjs` — the guard (read-only; `node
  scripts/state-drift-check.mjs [project-root]`).
- `scripts/test-state-drift.mjs` — fixture-driven tests (temp dirs with spec
  zones + state.yaml combos; asserts exit codes + drift output).

### Consequences

- Good, because the guard is robust to the schema's evolution — the `yaml`
  parser handles nested keys, quoted values, and comments that regex would
  miss.
- Good, because the dependency is zero-transitive and MIT — it does not pull
  a runtime tree into the repo.
- Bad, because the repo now has its first npm dependency, which means
  `node_modules` is populated (gitignored) and `npm install` is a
  prerequisite for running the guard. Mitigation: the guard is invoked by
  `skillgrid:qa` (which already runs in a Node-equipped environment) and by
  `skillgrid:resume` (same).
- Bad, because the `yaml` package is a moving target — a major version
  bump could change parse semantics. Mitigation: pin to `^2.x` (the current
  major); the schema is simple enough that a 3.x migration is a one-line
  version bump.

### Revisit criteria

Revisit this ADR (write a superseding ADR) when **any** of:
1. The repo adopts a Node test runner (jest/vitest) — at which point the
   fixture-driven `test-state-drift.mjs` can be ported to the runner and the
   `yaml` dependency's test-coverage story improves.
2. The `state.yaml` schema grows beyond what a flat read covers (workstreams,
   per-phase state) — at which point the guard's derivation logic (not the
   parser) is the thing to revisit, not the `yaml` package itself.
3. The `yaml` package changes its major version with breaking parse
   semantics — at which point re-evaluate option B (regex) for the
   simplified schema, or option C (Python) if the Node dep becomes
   undesirable.
