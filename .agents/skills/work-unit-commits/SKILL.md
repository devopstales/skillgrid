---
name: work-unit-commits
description: Plan reviewable commits during SDD apply (and general implementation) — one deliverable behavior per commit, with tests and docs kept with the code. Use when applying tasks, splitting commits, or keeping reviewer load healthy.
license: MIT
metadata:
  author: skillgrid
  version: "1.0"
  part-of: skillgrid
---

# work-unit-commits

Commit by **work unit**, not by file type. Typical caller: `sdd-apply`.

## Rules

| Rule | Requirement |
|------|-------------|
| One purpose | A commit = one deliverable behavior, fix, or migration. |
| Tests with code | Tests that verify the behavior land in the same commit. |
| Docs with change | User-visible docs land with the feature they explain. |
| Reviewable alone | After this commit alone, the repo should still make sense. |
| Message tells why | Conventional Commits; outcome, not a file list. |

Shared contract: [`_shared/conventions/commits.md`](../_shared/conventions/commits.md).

## Before each commit

1. State the unit's purpose in one sentence.
2. Include focused tests (or note N/A with reason).
3. `git diff` / `git diff --cached` — drop unrelated files.
4. Commit with a Conventional Commits subject.

## Weak vs better

| Weak | Better |
|------|--------|
| `add models` then `add tests` | `feat(auth): add token validation and tests` |
| `update docs` alone for a feature | Docs in the same commit as the feature |

## Gotchas

- Do not layer `models` → `services` → `tests` when none works alone.
- A red or half-done commit is not a checkpoint — finish the unit first.
