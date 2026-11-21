# Goldfish Review — Sharpening Loop (Track B)

The loop that iterates the 5-stage pipeline in `goldfish-review.md`.
R9's rules, verbatim semantics. Each round's findings resolve per
`goldfish-resolve.md`; the ledger round entry per
`_shared/craft/review-ledger.md`.

## Multi-round

Up to 5 rounds per stage — that is the cap. Each round: spawn reviewers
→ collect findings → resolve (categorize + fix) → frozen-candidate
re-check.

## Convergence

Convergence is 2 consecutive clean rounds. A round is clean = no new
Blocker or Major finding that is not settled. One clean round is not
enough — a fresh reviewer can find a different gap in the same text.

## Settled findings

Contested, or accepted without fix. Kept across rounds. If a new
reviewer raises a settled finding again — same scope, same issue — do
NOT re-open it.

## Fixed findings

An applied fix does NOT settle a finding. If a later reviewer raises the
same issue at the same scope, the fix failed — re-open.

## Churn

A new finding is at the scope of an applied fix, and reverting that fix
would resolve it. On churn, ask the user: **accept and proceed /
continue past the cap / return to drafting**.

## Turn-taking

After each round's resolution the loop goes idle and waits for the user
to confirm "run another round" — the agent does not auto-loop. The chat
turn ends after each round's resolution; the next round starts only on
the user's go-ahead.

## Frozen-candidate re-check

Replaces the original line-range re-scan (A1): before each round,
re-read `blueprint.md` — line numbers shift after edits — and re-hash
it. If the hash changed since the round started, the round is
invalidated per the `review-ledger` frozen-candidate rule
(`_shared/craft/review-ledger.md`) and re-dispatched.
