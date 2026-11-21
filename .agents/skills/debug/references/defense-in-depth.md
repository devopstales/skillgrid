# Defense in Depth

After a root-cause fix, add validation so the same *class* of defect cannot
slip through again. The root-cause fix makes this instance correct; the
validation layers make the next instance fail early, close to the origin.

## Principle

One fix, many tripwires. The origin fix is mandatory. Each added layer is
optional and proportional to the blast radius — not a ritual.

## The Layers (closest to the origin first)

1. **At the origin** — the fix itself: correct transformation, correct
   default, the missing case handled. This is the only layer that makes the
   behavior right.
2. **At the boundary** — validate external input where it enters the system
   (request parser, message consumer, file load, DB read). Fail fast with a
   message that names the field and the constraint. This converts "bad value
   found six layers deep" into "bad value rejected at the door."
3. **At the invariants** — the few checks that must always hold
   (uniqueness, range, referential integrity). Prefer the mechanism that
   owns the invariant (DB constraint, type system, framework validation)
   over hand-rolled checks.
4. **At the consumer of last resort** — a check that turns a would-be silent
   failure into a loud one. Use sparingly: it is a detector, not a fix, and
   it must log or throw with enough context to be useful.

## Proportionality

| Blast radius | Layers to add |
|--------------|---------------|
| Local (one function, one caller) | Origin fix only |
| Module (shared helper, many callers) | Origin + boundary validation |
| System (cross-service, persisted data, user-visible) | Origin + boundary + invariant check + last-resort detector |

## Rules

- **Each layer fails at a different time.** A useful stack of layers means
  the next defect of this class is caught at the *earliest* layer, with an
  error that says what is wrong and where — not the same crash, six frames
  later.
- **The failing test from Phase 4 is layer zero.** It proves this instance;
  the new layers are exercised by their own tests when they are non-trivial.
- **Do not add a layer that duplicates the origin fix.** A guard that repeats
  the origin's correction at the consumer hides the origin when it regresses.
  Guards detect; they do not correct.
- **Record the layers** in the debug report so the next session can see what
  is in place and where the next tripwire should go.
