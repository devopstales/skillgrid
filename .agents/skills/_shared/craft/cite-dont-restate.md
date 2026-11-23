# Cite, Don't Restate

The rule that keeps a skill a routing table instead of a copy. When the answer
lives in another file, point at it — do not reproduce it.

## The rule

If a fact, rule, list, or contract has a home elsewhere, reference it by path
and name: `per `_shared/planning/rigor-tiers.md``, `per
`AGENTS.md § Rules``, `per the ADR Review Manifest`. Do not copy the content
next to the pointer.

## Why

A restated fact rots twice: when the source changes, the copy is wrong and the
copy's readers never find out. The pointer stays true. Citing is also cheaper —
the reader pulls the detail only when it needs it, instead of paying for it on
every load.

## What counts as restating

- Quoting a banned-word list, a config-key table, a state-machine, or a rule
  body that already has a file.
- Re-explaining a sibling skill's protocol in your own words ("the review loop
  caps at N rounds" when `rigor-tiers.md` owns N).
- Inlining a template's structure instead of naming the template file.

## What is NOT restating

- The one-line *name* of what the pointer is ("the fix-loop cap", "the user-gate
  contract") — that is routing, not restating.
- A value the reader cannot get elsewhere (a number unique to this file).
- A citation line in an artifact (`Spec:`, `per NNNN-…`) — the artifact
  cites its sources by design.

## The test

Could you delete the restated sentence and replace it with `per <path>` and
still have the reader get the same fact, at the same point in the flow? If yes,
it was restating.
