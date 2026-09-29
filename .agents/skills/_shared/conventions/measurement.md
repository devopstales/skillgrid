# Measurement Convention (shared across all Skillgrid skills)

Single source of truth for the **`## How to measure it`** section that each
pipeline skill carries. The rule it exists to enforce: **a skill that cannot
say how it will know it worked is not a controlled process — it is a ritual.**
Every stage of the AI-native SDLC ends by writing an artifact, and the value of
that stage is only visible through a number someone can read later, long after
the session is gone.

## The Rule

1. **Two indicators, no more, no less.** Every `## How to measure it` section
   has exactly two rows: a **Leading** indicator (fast signal, observable during
   or immediately after the stage) and a **Lagging** indicator (slow signal,
   observable only after downstream stages have consumed this stage's output).
   A skill with only a leading indicator can't detect the failure that surfaces
   later; a skill with only a lagging indicator is blind while it runs.
2. **Every indicator names its data source.** The source must be a
   machine-readable record the pipeline already produces: Git history, PR
   metadata, CI logs, `state.yaml`, `report.md`, or `review.md`. If the number
   has to come from a person's memory or a gut feel, it is not an indicator —
   it is an opinion, and it goes in the prose, not the table.
3. **Every indicator names its direction.** "Should fall" or "should rise."
   An indicator with no direction is a dashboard, not a control. A falling
   leading indicator that is *supposed* to rise is a signal, not noise.
4. **The indicator is read, not computed, at the gate.** The gate (QA, review,
   ship) does not branch on these numbers in real time — they are the
   after-the-fact health signal a human or a periodic report reads to decide
   whether the skill needs tuning. Confusing the gate (which branches on
   evidence) with the health signal (which branches on trends) is the one
   error this section must not make.
5. **Read from the artifact chain, never from the session.** A metric that
   requires re-opening the conversation to compute is not durable. The commit
   chain, the PR, and the committed artifacts are the only admissible sources,
   because they survive the session that produced them.

## The Section Format

Each pipeline skill appends, before `## Common Rationalizations`:

```
## How to measure it

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | <one line> | <Git / PR / CI / state.yaml / report.md> | <should fall / should rise> |
| Lagging | <one line> | <Git / PR / CI / incident> | <should fall / should rise> |
```

One row each. One line each. No narrative in the cells — the cell is a
quantity plus its source plus its direction, nothing else.

## Where It Applies

- **Every pipeline skill** that ends by writing or consuming an artifact in the
  spec zone: `brainstorming`, `writing-blueprints`, `slicing`, `qa`,
  `requesting-code-review`, `ship`.
- Not applied to utility skills with no artifact chain (`resume`, `reflect`
  reads rather than writes a stage artifact; `reflect`'s own health is the
  acceptance verdict already in `report.md`).

## Red Flags

- A `## How to measure it` row whose data source is "the team feels" or "the
  reviewer thinks."
- Three or more indicators (it is now a dashboard, not a control).
- An indicator with no direction, so a change in the number tells you nothing.
- A metric that the gate branches on in real time (that belongs in the gate's
  own evidence, not in the health signal).
- A skill that writes an artifact but has no `## How to measure it` section —
  it produced a thing and has no way to say whether the thing was worth it.
