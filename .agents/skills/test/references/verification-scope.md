# Verification Scope Convention (shared across all Skillgrid skills)

Single source of truth for the **scope** of a verification or enumeration
result. The rule it exists to enforce: **a zero count is never a bare zero —
it carries its scope.** A `0` that means "I looked at everything and there was
nothing" is a real answer; a `0` that means "I couldn't see all of my input" is
a non-answer. Before this convention, those two cases were output-identical,
which is the failure class this convention removes.

## The Scope Vocabulary

Four atoms. Every derivation that *counts or enumerates* something (plans,
scenarios, changed files, drift fields, gate checks) MUST be able to say which
of these four it produced. The report names it verbatim on a `SCOPE:` line; a
skill or script that cannot determine scope must not emit a count at all.

| Scope | Meaning | A zero here is |
|---|---|---|
| `COMPLETE` | I saw all of my input. The enumeration was total. | a **real answer** — there genuinely is nothing |
| `TRUNCATED` | I saw part of my input and stopped (budget, limit, partial read). | a **non-answer** — there may be more I didn't see |
| `UNSCOPED` | I could not determine the boundary of my input (no scope context — e.g. no active change, no base ref, no milestone window). | a **non-answer** — I don't know what the full set is |
| `UNREADABLE` | My input was present but I could not read it (missing/unparseable file, permission, I/O error). | a **non-answer** — the set exists but I couldn't see it |

**Severity ordering** (for combining independently-scoped results about the
same target): `UNREADABLE` > `UNSCOPED` > `TRUNCATED` > `COMPLETE`. When a
result is composed of several scoped sub-answers, the composite carries the
**worst** scope. `COMPLETE` is the identity — combining with it changes
nothing.

## The Rule

1. **A zero is never bare.** Any reported count of zero (or an empty
   enumeration: empty drift table, `0/0` scenarios, empty changed-file set)
   MUST be paired with its scope. `DRIFT: none` with `SCOPE: COMPLETE` means
   "no drift, and I checked everything." `DRIFT: none` with `SCOPE: UNSCOPED`
   means "no drift *that I could see*, but I didn't have a boundary to check
   against" — and that is not a clean bill.
2. **Fail closed on non-COMPLETE.** A verification whose scope is
   `TRUNCATED` / `UNSCOPED` / `UNREADABLE` does **not** satisfy a "done" or
   "pass" gate. Only `COMPLETE` does. A stale, unscoped, or unreadable
   verification routes **away** from PASS — it is not a silent pass and not a
   silent fail; it is a named non-answer that the human (or a re-run) must
   resolve.
3. **Name it, don't imply it.** The scope is written out on its own line —
   `SCOPE: UNSCOPED` — not folded into prose. A consumer branches on the named
   atom, never on a guess about whether the check "probably" saw everything.
4. **No scope, no count.** If a derivation cannot assign one of the four
   atoms, it reports that it cannot verify (scope `UNREADABLE` with the reason,
   or `UNSCOPED` when the boundary is genuinely undeterminable) rather than
   emitting an unattributed number.

## Where It Applies

- **Drift scripts** (`.agents/skills/qa/scripts/state-drift-check.mjs`,
  `.agents/skills/qa/scripts/ship-drift-check.mjs`): after the `DRIFT:` verdict line, emit a
  separate `SCOPE: <atom>` line describing the scope of the enumeration that
  produced the verdict. Exit codes are unchanged — scope is additive.
- **QA gate** (`skillgrid:qa`): the `report.md` carries a `## Verification
  Scope` field naming the scope of the goal-backward and traceability
  derivations. The four-state gate treats non-`COMPLETE` scope as routing away
  from PASS (see `skillgrid:qa` fail-closed rule).
- **Any skill that reports a count or "none"** (resume, progress, review): if
  the number is a zero or the result is "none", say the scope.

## Relationship to the Other Conventions

- This is the **scope** axis. `verification-ladder.md` is the **depth** axis
  (L1–L4). A check can be at L2 (full suite) *and* `UNSCOPED` (couldn't see
  all the plans) — the ladder says how hard it looked, scope says how much of
  the input it looked at. Both must be `COMPLETE`/satisfied for a clean
  verdict.
- It complements `sdd-structure.md`'s "the spec zone is the deeper truth" —
  scope is how a guard reports *whether it actually saw* that deeper truth,
  rather than assuming it did.

## Red Flags

- A `DRIFT: none` with no `SCOPE:` line (or a `SCOPE: COMPLETE` the check
  never actually established).
- Treating an empty changed-file set or an empty drift table as a clean bill
  when the base ref was missing (`UNSCOPED`) or the read failed (`UNREADABLE`).
- A PASS verdict resting on a `TRUNCATED` or `UNREADABLE` verification.
- Combining scoped results by taking the best scope instead of the worst.
