---
id: decision-010
title: "ADR-0010 Verification scope discriminator: a zero is never a bare zero"
date: "2026-09-24"
status: "accepted"
---

Source: `.skillgrid/artifacts/04-adr-0010-verification-scope-discriminator.md`

# Verification scope discriminator: a zero is never a bare zero

---
status: "accepted"
supersedes: none
date: 2026-09-24
---

## Context and Problem Statement

Our drift guards and the QA gate report *counts and "none" verdicts*:
`DRIFT: none`, an empty changed-file set, `0/0` scenarios, an empty drift
table. A count of zero is the most ambiguous output a verifier can produce,
because two very different situations produce the identical byte sequence:

- **A real answer** — I looked at all of my input and there genuinely was
  nothing.
- **A non-answer** — I couldn't see all of my input (no base ref, no active
  change, an unreadable file), so the zero tells me nothing.

Before this convention, those were output-identical. A `DRIFT: none` from a
guard whose base ref didn't exist (so `git diff` returned an empty set) read
exactly the same as a `DRIFT: none` from a guard that checked a clean tree.
The guard was silently assuming it had seen everything, and a consumer had no
way to tell a clean bill from a check that couldn't run. This is the same
failure class GSD Core's `SCOPE` enum was built to remove, and it matters here
because `state-drift-check.mjs` and `ship-drift-check.mjs` are read-only
verifiers whose whole value is *trust* — a verifier that can't distinguish
"clean" from "couldn't see" erodes the trust it is meant to provide.

## Considered Options

- **A. Convention + script (four-atom scope).** Add a shared convention
  (`_shared/conventions/verification-scope.md`) defining `COMPLETE` /
  `TRUNCATED` / `UNSCOPED` / `UNREADABLE`, and have the two drift scripts emit
  a separate `SCOPE: <atom>` line after their `DRIFT:` verdict. The QA gate
  names the scope of its own derivations in `report.md` and fails closed on
  non-`COMPLETE` scope. No new dependency; the scripts already read the input
  they need.
- **B. Convention only.** Define the vocabulary in a convention and rely on
  skills to name scope by hand, with no script output. Cheaper, but the drift
  scripts (the part that runs unattended) still emit a bare `DRIFT: none` with
  no machine-readable scope — the non-answer stays invisible to a consumer that
  only reads stdout.
- **C. Full Go leaf module.** Add a frozen scope enum to `skillgrid-cli` as a
  cycle-free leaf (GSD's `planning-scope.cjs` analog). Most rigorous, but the
  drift guards are Node (per ADR-0008), so the Go enum would have no consumer
  in this change and would cross the Node/Go boundary for no current payoff.

## Decision Outcome

Chosen option: **A. Convention + script (four-atom scope)**, because the
failure is in the *verifiers*, and verifiers must emit the scope where it is
consumed (stdout, `report.md`), not in a human's head. The four atoms are the
minimum that make "genuine zero" vs. "non-answer" representable, and a frozen
enum beats a free-form message string because a consumer can branch on the
atom rather than pattern-match prose (the same rationale ADR-0008 used for
keeping the drift guard read-only and machine-checkable).

Option B (convention only) was rejected: it fixes the *reporting* skills but
leaves the unattended guards — the ones that run on every commit/ship — emitting
bare verdicts. A scope that only a human remembers is not a scope.

Option C (Go leaf) was rejected as premature: the consumers of scope in this
change are the Node drift scripts and the QA skill, not the Go binary. A Go
enum would be dead surface until a future change gives it a consumer. The
convention is written to be portable to a Go leaf later without rework.

### Implementation

- `_shared/conventions/verification-scope.md` — the four-atom vocabulary, the
  "a zero is never a bare zero" rule, the fail-closed rule, and worst-scope-wins
  composition.
- `scripts/state-drift-check.mjs` — emits `SCOPE: <atom>` (the worst of the
  spec-zone enumeration scope and the active-change-dir scope). Exit codes
  unchanged; scope is an additive output line.
- `scripts/ship-drift-check.mjs` — emits `SCOPE: <atom>` (`COMPLETE` when the
  `git diff` ran, `UNSCOPED` when it failed — e.g. a bad base ref). Exit codes
  unchanged.
- `scripts/test-state-drift.mjs`, `scripts/test-ship-drift-check.mjs` —
  fixtures asserting the scope line on the clean, missing-specs, and bad-base
  paths.
- `.agents/skills/verification/SKILL.md` — Step 9.7 (Verification Scope + Staleness
  Check), a `## Verification Scope` section in `templates/report.md`, and a
  fail-closed rule in the four-state gate (only `COMPLETE` scope satisfies PASS).
- `.agents/skills/qa/templates/report.md` — the `## Verification Scope` table.

### Consequences

- Good, because a verifier that can't see its input now says so: `DRIFT: none,
  SCOPE: UNSCOPED` is a non-answer a consumer can act on, not a silent clean
  bill.
- Good, because the gate fails closed on scope: a `TRUNCATED` / `UNSCOPED` /
  `UNREADABLE` verification routes off PASS, so a "done" can never rest on a
  check that didn't look at everything.
- Good, because exit codes are unchanged — scope is additive, so no existing
  caller of the drift scripts changes behavior; the line is new but the
  contract (0/1/2) is stable.
- Bad, because the scope derivation is deliberately *cheap* (it reflects what
  the script already read: a directory listing or a git-diff exit). It does not
  count enumerated plans against a declared total, so `TRUNCATED` is only
  reachable in the drift scripts' current paths when an input is genuinely
  absent — a stricter derivation (count vs. declared) is a follow-on, named in
  Revisit Criteria.
- Bad, because a human reading the report must now understand four atoms.
  Mitigation: the convention's table names each atom's meaning and the
  "a zero here is a real answer / non-answer" column.

### Revisit criteria

Revisit this ADR (write a superseding ADR) when **any** of:
1. A drift script gains a *partial-read* path (budget-limited enumeration,
   `--since` windowing) — at which point `TRUNCATED` becomes reachable in the
   scripts and the derivation should count seen-vs-expected, not just
   present-vs-absent.
2. The Go CLI (`skillgrid-cli`) takes on scope-bearing verification (e.g. a
   `skillgrid drift` command) — at which point option C (a shared Go leaf enum)
   becomes the right home, and the convention's four atoms port unchanged.
3. The QA gate needs *more* than four atoms to describe a derivation's
   completeness (e.g. a "sampled" scope for statistical checks) — at which
   point the enum grows and the convention is amended.
