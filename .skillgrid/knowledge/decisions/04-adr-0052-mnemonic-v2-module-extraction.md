---
type: TechnicalDecision
status: accepted
supersedes: none (v1 module structure retired, not superseded)
date: 2026-11-01
---

# ADR-0040: Mnemonic v2 module extraction


## Context

Mnemonic v1 lives in `mnemonic/` (module `github.com/devopstales/skillgrid/mnemonic`). The v2 spec zone (`2026-10-31-mnemonic-v2-*`) defines a total rewrite with a different module layout (`mnemonic-v2/`), a fresh schema, a dialect-agnostic `Store` interface, and LLM-first seams. The v1 module is too coupled to its own schema and invariants to rewrite in place without a migration path that the spec explicitly disclaims (spec `002-schema` §6: "v1 store files are *not* auto-migrated into v2").

During v2 development, both modules must coexist: v1 keeps serving the running install while v2 is built and validated.

## Decision

Create a fresh Go module `mnemonic-v2/` (module path `github.com/devopstales/skillgrid/mnemonic-v2`, `go 1.26.0`) as a sibling of `mnemonic/`. The v1 module is renamed to `mnemonic_back/` (git mv + go.work + Taskfile) to make the name collision explicit and to prevent accidental imports from v1's packages. The two modules coexist until v2 ships, at which point `mnemonic_back/` is archived and `mnemonic-v2/` is renamed to `mnemonic/`.

The `go.work` `use` block lists both modules during the coexistence period.

## Consequences

- **Positive:** clean v2 build with no v1 import coupling; v1 stays live during development; the cutover is a rename, not a migration.
- **Negative:** two `go.mod` files to maintain during the overlap; the Taskfile must build both; the install script must know which module produces the binary.
- **One-way door:** the module path `github.com/devopstales/skillgrid/mnemonic-v2` is referenced by the plugins (opencode-plugin, pi-plugin) as their import target. Renaming `mnemonic-v2/` to `mnemonic/` at ship-time requires updating the plugins' imports — a mechanical change but a contract break if missed.
