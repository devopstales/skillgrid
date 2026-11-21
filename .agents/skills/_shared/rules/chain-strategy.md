# PR Chain Strategy

The single home for the three delivery-chain definitions. `chain.strategy` in
`.skillgrid/config.yaml` is a **project-level** decision, set at onboarding — not
a per-change one. Slicing, onboarding, execute, and ship all read from the same
definitions; none of them restates them.

## The three strategies

- **`stacked-to-main`** — each PR merges to main in order. Fast, fix on the go.
  Best for independent slices.
- **`feature-branch-chain`** — PR #1 targets the feature branch; later PRs
  target the previous PR branch so each child diff stays focused; only the
  tracker merges to main. Best for rollback control.
- **`size:exception`** — keep as one PR with maintainer approval. Best for
  generated code, migrations, vendor diffs.

## Base boundaries (feature-branch-chain)

For `feature-branch-chain`, name the intended base boundary per work unit:
PR #1 base = main (or the tracker branch); PR #2 base = PR #1 branch; PR #3 base
= PR #2 branch. The per-unit boundaries live in `tasks.md` → `## Delivery
Strategy`; the project-level choice lives in `config.yaml` `chain.strategy`.

## Who decides what

- **Onboarding** detects an existing `chain.strategy`, validates it, or asks the
  programmer to pick one.
- **Slicing** records the choice in `tasks.md` (or defers with
  `Chain strategy: pending` when it is a team decision, not the slicer's).
- **Execute** refuses to dispatch until the strategy is resolved (not
  `pending`), or a `size:exception` is accepted.
- **Ship** resolves the merge base from `chain.strategy` + the work-unit table;
  if it is missing or `pending`, it asks before merging.
