# Vault is source of truth; DB is derived index

---
status: "accepted"
date: 2026-11-24
supersedes: none
---

## Context and Problem Statement

The Phase 3 subsystem reads and indexes wiki content. Someone must own the content: the git-tracked vault in the repo, or the project DB.

## Considered Options

- Files-first: the vault (`.skillgrid/projectwiki/`) is truth; the DB holds only derived state (body mirror + hashes, FTS, graph, vectors), always rebuildable
- DB-first: DB is truth, markdown exported for humans

## Decision Outcome

Chosen option: "Files-first", because it is the v2 model and the skillgrid convention ("the agent owns the page markdown; the engine owns derived state — always rebuildable, never hand-edited"), and because the vault is diffable, versionable, reviewable, and survives binary absence. Index rebuild is the recovery path.

### Consequences

- Good, because content changes are reviewable in git and survive binary and schema changes
- Good, because a lost/corrupt index is a rebuild, not a migration
- Bad, because the index must stay in sync with the vault (reindex on change); DB-local state (run state, baselines) is not reviewable in the vault
