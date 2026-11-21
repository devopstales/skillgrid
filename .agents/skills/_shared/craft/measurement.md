# Measuring a Skill

The contract for the `## How to measure it` section every skill carries. A
skill that cannot name how it is doing better is not instrumented.

## The shape

Two indicators, one leading and one lagging, in a four-column table:

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | … | … | should rise / fall |
| Lagging | … | … | should rise / fall |

- **Leading** predicts the outcome before it lands (a process you can watch
  weekly). **Lagging** confirms the outcome after the fact (the truth you are
  trying to move). You need both: a leading alone can be gamed, a lagging alone
  arrives too late to act on.
- **Data source is a real artifact**, not an impression: PR metadata, a commit
  timestamp, a git-history query, a ledger count. "Feels faster" is not a data
  source.
- **Direction is explicit** — should rise or should fall. An indicator with no
  direction is a decoration.

## Rules

- **One skill, its own table.** Do not copy a sibling's indicators; each skill
  measures the thing it exists to do.
- **The measurement must be cheap enough to run often.** If it takes an hour to
  compute, it will not be computed. Prefer git metadata and existing ledgers
  over new instrumentation.
- **Restate nothing.** This file is the *how*; the skill's own table is the
  *what*. A skill names its two indicators and cites this file for the shape —
  it does not re-explain leading/lagging.
