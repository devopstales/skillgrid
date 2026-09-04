# Tasks: 002 — Step 01-identity-binding

> Goal: Establish clone-private identity binding, child auto-promote, ambiguity, bounded config walk, and seed aliases.
> Depends on: none

## Execution

- [ ] 01.1 Rewrite project resolution to bind the project to its clone; never derive the id from mutable git state.
- [ ] 01.2 Single child repo auto-promotes; more than one returns ambiguity with the candidate list.
- [ ] 01.3 Bound the config walk to the enclosing repo root.
- [ ] 01.4 Seed aliases so prior keys route to the new canonical id.
- [ ] 01.5 Update the store to open idempotently under the new identity.
