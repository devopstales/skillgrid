# Concurrent Leaves (Concurrent Coupled Build)

This reference covers the case that is neither a batch of independent fixes nor a
sequential build: **2+ implementation tasks that must run at once AND touch
overlapping files or shared interfaces**, so they can only be safely concurrent
if they stay disjoint and are verified at the parent. That is the "concurrent
coupled build."

The base `parallel-execution` pattern (one agent per independent domain, no
shared state) does not cover this — it says "shared state → don't parallelize."
This reference is the narrow exception: shared files are allowed *only* when each
leaf's write set is provably disjoint and a parent re-verifies the join.

## When this applies (all must be true)

- 2+ implementation tasks, not investigations.
- The tasks touch the same file(s) or a shared interface.
- Each task's *write* set can be made **disjoint** from the others'.
- You want them in flight at the same time (wall-clock win), not sequential.

If any of these fail — especially "write sets can be made disjoint" — do not
parallelize. Fall back to `skillgrid:subagent-execution` (sequential,
dependency-gated, with its Depth Tree). That is always the safe path; this
reference is the optimization.

## The contract (per leaf)

Every concurrent leaf carries the same contract as a `subagent-execution`
implementer, plus two coordination lines:

- **`Owns:`** — the exact repository-relative globs this leaf may **write**.
  `Owns:` is coordination metadata, not a filesystem sandbox: the leaf may read
  outside its `Owns`, but it may not write to another in-flight leaf's `Owns`.
  If two leaves' `Owns:` overlap and cannot be split into disjoint sets, the
  build is not concurrent-safe — go sequential.
- **`Needs:`** — the tasks or interfaces this leaf builds on. A leaf with an
  in-flight `Needs:` is not dispatchable; it must wait for that dependency to be
  verified complete.

Record both in the dispatch brief. The brief also carries the leaf's `#### Gates`
oracles (see `skillgrid:test-driven-verification`) and the 4-pass refinement loop
(see `skillgrid:subagent-execution` → implementer template).

## Dispatch

- Dispatch all leaves of a **wave** in one response (multiple dispatch calls =
  concurrent). A wave is a set of leaves whose `Needs:` are all satisfied.
- Keep waves small. This is a lightweight reference: it does not track lease
  state, wave state, or a `dispatch.json`. That machinery only pays off at 4+
  leaves in flight; below that, the prose contract above is enough. If you find
  yourself juggling lease release by hand across many leaves, that is the signal
  to build the full orchestrator, not to stretch this reference.
- Never dispatch a leaf whose `Owns:` intersects another in-flight leaf's
  `Owns:`. If you cannot prove disjointness, hold the leaf and run it in the next
  wave (sequential behind the other).

## Branch integration gate

If the leaves in a wave share an interface or file (the reason they are coupled),
one named leaf **integrates** them. That leaf carries an extra `G<n>` gate: the
other leaves' outputs compose, and the shared interface holds end-to-end. This
gate is independent of the child leaves — it verifies the branch, not just the
parts. If it cannot be made runnable, it is a `manual` gate you settle at the
integration review, not a silent skip.

## Parent re-verify (the non-negotiable)

A leaf's self-pass is a claim, not verification. Before marking any leaf
complete:

1. Re-run its `#### Gates` oracles yourself — the runnable `CHECK:` + `EXPECT:`
   lines, freshly run (per `skillgrid:test-driven-verification`).
2. For a shared file or interface, re-verify the integration gate from the
   integrated leaf, not from the individual leaves.
3. Confirm no leaf wrote outside its `Owns:` — a stray write into a sibling's
   `Owns:` is a conflict even if both suites pass.

Only after the parent re-verify passes does a leaf count as done. Then the
integration step (base pattern step 4) runs the full suite across all leaves
together.

## Fall back, don't force

The moment you hit any of these, stop parallelizing and switch to
`skillgrid:subagent-execution` (sequential) for the affected leaves:

- `Owns:` cannot be made disjoint.
- A leaf's `Needs:` is in flight.
- You cannot re-verify a leaf's oracle at the parent.
- A leaf reports a write outside its `Owns:`.

Parallel is a wall-clock optimization. Correctness does not depend on it — the
sequential path is always available, and it is the one that is hard to get wrong.
