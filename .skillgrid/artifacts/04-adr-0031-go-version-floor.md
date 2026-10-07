# Go 1.22+ minimum to build

---
status: "accepted"
supersedes: none
date: 2026-10-07
---

**Deciders:** paladm
**Related:** ADR-0012 (SQLite as second brain), ADR-0019 (locked decisions are files)

## Context and Problem Statement

The repo builds with a Go toolchain and ships a single binary. The `go.mod` declares a minimum language/toolchain version, and contributors on older toolchains need a stable floor to build against. Previously this floor sat inline as a "locked constraint" bullet in `.skillgrid/ASSUMPTIONS.md` § `Locked constraints` — but a minimum build version is a *decision* (we chose 1.22 over a lower or higher value), not an inferred operational limit. Under the ADR-home rule (ADR-0019) decisions live in `artifacts/04-adr-NNNN-slug.md` with a pointer-only index row; operational rules that are not decisions live in `AGENTS.md`. This record promotes the Go floor to its proper home.

## Considered Options

- Keep the Go floor as an inline locked-constraint bullet in `ASSUMPTIONS.md`
- Promote it to an ADR file (decision) indexed in `## LOCKED`
- Promote it to an ADR and also state the floor in `AGENTS.md` as the build contract

## Decision Outcome

Chosen option: "Promote it to an ADR file (decision) indexed in `## LOCKED`", because a minimum toolchain version is a settled engineering decision, and the ADR is where such decisions are recorded and superseded.

**Go 1.22+ is the minimum toolchain required to build skillgrid.** A `go.mod` `go` directive below 1.22 is a drift signal; a build that fails only on an older toolchain is not a product bug. The exact floor in `go.mod` is the source of truth for the number; this ADR locks the *policy* (there is a floor, and raising it is a decision recorded as a new ADR that supersedes this one).

### Consequences

- Good, because the Go floor is now a first-class decision: raising or lowering it is an explicit, recorded, superseding act rather than an edit to a constraints bullet.
- Good, because `ASSUMPTIONS.md` § `LOCKED` is now purely the ADR index, and non-decision operational rules live in `AGENTS.md` where the onboarding skill renders them.
- Bad, because a future reader must open this file (or its successor) to see the current floor rather than finding it inline next to the other constraints.
