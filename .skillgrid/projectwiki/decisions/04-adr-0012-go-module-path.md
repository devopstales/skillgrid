# Go module path: same path as the old module (branch handover)

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

The new `mnemonic` binary builds in `mnemonic/` **in this repo**
([[entities/mnemonic]]). The repo's remote is
`git@github.com:devopstales/skillgrid.git` — the **same remote** as
`../skillgrid` (observed: this checkout is branch `feat/v2-rewrite`, the
sibling checkout is `release/2`). The old memory package
([[sources/skillgrid-mnemonic-v2]]) declares module
`github.com/devopstales/skillgrid/mnemonic` in `../skillgrid/mnemonic/go.mod`.
The programmer chose (2026-11-23 gate) the **same module path** for the new
module.

## Considered Options

- **`…/skillgrid/mnemonic/v3`**: idiomatic major-version rewrite, no import
  collision while both branches' `mnemonic/` directories coexist.
- **`…/skillgrid-skills/mnemonic`**: matches the directory name, but dir ≠
  remote name (the remote is `skillgrid`).
- **`…/skillgrid/mnemonic`** (same as the old module): one path, one
  product — when `feat/v2-rewrite` ships, the path **hands over** from the
  old package to the new one; no import-path migration for any consumer.

## Decision Outcome

Chosen option: **`github.com/devopstales/skillgrid/mnemonic`** for the new
module in this repo's `mnemonic/` directory.

### Consequences

- Good: consumers of the v2 rewrite import the same path they used for v0/v1
  — the path is the product's name, not a versioned artifact.
- Good: consistent with the branch model — two checkouts of one remote;
  the new `mnemonic/` is the successor of the old one, in the same repo.
- Bad: while both branches exist, two codebases share one module path —
  **constraint: no cross-repo/cross-branch imports** of
  `github.com/devopstales/skillgrid/mnemonic/…` while the old package is
  live on `release/2`; the old package stays frozen
  ([[decisions/04-adr-0008-memory-core-only-v1]] declined in-place
  evolution).
- Bad: go tooling resolving the path before the v2 branch ships may fetch
  the old package's code — pin with `replace` in any module that consumes
  the new one pre-release.

## Not adopted

- **`/v3`** — the branch-handover model makes the bare path the cleaner
  story; the v3 suffix would imply a parallel import identity for the same
  product.
- **`skillgrid-skills` path** — dir name ≠ remote name; it would also fork
  the product's import identity from its repo.

## Related

- Supersedes: none.
- Depends on: [[entities/mnemonic]] (the module this names),
  [[decisions/04-adr-0008-memory-core-only-v1]] (the old package stays
  frozen), [[sources/skillgrid-mnemonic-v2]] (the old module declaration).
- Consumed by: [[syntheses/mnemonic-core]], the blueprint's File Structure.
