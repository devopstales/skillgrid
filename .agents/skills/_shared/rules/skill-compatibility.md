# Skill Compatibility

A **load-time** advisory: before two or more skills are loaded as *active rules in the
same session*, decide whether they can coexist. This is the inverse of merging — the
question is not "can these rule sets be combined into one" but "**should an agent load
these together as active rules, or pick one and treat the rest as reference?**"

This is a standard (like `sdd-structure.md`), not an invokable skill. It is referenced,
never inlined. The matrix is **filled lazily** — a pair is added only when a real
conflict, or a real need to confirm coexistence, is observed. An empty cell means
"no recorded verdict," which is *not* a green light and not a red light.

## Method

For a pair, produce one canonical file per **unordered** pair at
`_shared/rules/compatibility/<earlier>/<later>.md` (alphabetical; `<earlier>` sorts
before `<later>`), with:

- **Verdict** — one of:
  - ✅ **complementary** — safe to load both as active rules; they reinforce each other.
  - 🔁 **overlap** — they cover the same ground; load one as active, treat the other as reference (avoids doubled tokens, no clash).
  - ❌ **conflicting** — they issue incompatible instructions for the same decision; load **only the one the task needs**, note the other as reference, and name the decision they fight over.
- **Evidence** — for each skill, **≥ 2 cited rule locations** (file + line range or
  section) that establish the verdict. No whole-file citations. The citation is what
  makes a re-runner able to re-derive the verdict without re-reading both skills.
- **Known high-risk note** (when applicable) — pairs where the conflict is subtle
  enough to warrant an explicit "check before loading both" reminder.

Scoring rubric (record as a percentage next to the verdict): `Conflict / Overlap /
Complementarity: N%`. A verdict is a judgment constrained by the cited evidence, not a
gut call — boilerplate ("these are both good", "they overlap a bit") without the two
cited locations per side is rejected.

## How to add a pair

1. Observe the pair actually loaded together (or a real question of whether to).
2. Read both skills' operative rules; locate the specific instructions that meet.
3. Write `_shared/rules/compatibility/<earlier>/<later>.md` per the method above.
4. Add the pair's verdict to the matrix below with a link to the file.

Do **not** pre-fill pairs speculatively. The matrix earns its token cost only when a
pair is genuinely in question.

## Matrix (lazy — empty until observed)

| Pair | Verdict | File |
|---|---|---|
| — | (none recorded) | — |

> To query: does `<skill A>` conflict with `<skill B>`? Look for the pair (either
> order) in the matrix. No row → no recorded verdict; check the two skills' operative
> rules directly before loading both as active rules.
