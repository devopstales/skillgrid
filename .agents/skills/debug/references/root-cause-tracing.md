# Root-Cause Tracing

Trace a bad value backward through the call stack until you find where it
originates. The fix belongs at the origin, not where the error surfaces.

## When

The error is deep in the call stack, or the failing value's producer is not
obvious from the crash site. The crash site is usually the *consumer* of the
bad value.

## The Technique

Start at the frame where the bad value is first *observed* (an assertion
failure, an exception, a wrong result) and walk up the stack:

1. **Name the bad value precisely.** What exactly is wrong — the value, its
   type, its absence, its timing? "It's null" is not precise enough;
   "the `region` field is empty when it is a required ISO code" is.
2. **Who passed it?** Find the caller that supplied the value to the current
   frame.
3. **Was it bad there, or produced there?**
   - Bad already (just forwarded) → go up one more frame. Repeat from 2.
   - Produced here → this is the candidate origin.
4. **At the candidate origin, ask how it was produced.** Trace the source of
   every input to that computation: a variable, a function call, a config
   read, a deserialization, an external response, user input.
5. **Stop at the true origin** — the earliest point where the value is
   *created or transformed*, not merely forwarded. Typical origins:
   - a parser / deserializer that drops or mis-maps a field
   - a function with a wrong default or a missing case
   - config / env propagation that never reached this layer
   - an assumption about caller order or state that does not hold
   - a data migration or seed that predates a schema change

## Rules

- **Fix at the origin.** A guard added at the crash site converts a loud
  failure into a silent one and leaves the origin producing bad values for
  everyone else.
- **Every hop is a trail entry.** Each "who passed it?" step is a
  hypothesis about where the value came from — record it in the debug state
  file with the evidence and the verdict, the same as any other Phase 3
  hypothesis.
- **Do not stop at the first "plausible" origin.** Plausible is a hypothesis;
  the origin is confirmed when a minimal test at that point changes the
  value and the symptom disappears (or is produced in isolation).
- **If the trace reaches the program boundary** (input from outside) with no
  internal transformation at fault, the origin is the boundary: the system
  accepts input it does not validate. The fix is validation at the boundary,
  not defensive checks in every consumer.

## Multi-Component Systems

When the value crosses a component boundary (HTTP call, queue message, file
format, database row), the trace splits:

1. **Instrument both sides** of the boundary: log the value entering and
   exiting, with enough identity to match the pair (request id, row id).
2. One instrumented run shows which side corrupts or drops it.
3. Continue the trace on the side that produced the good value and received
   the bad one.

## Common Misreadings

| Misreading | Reality |
|------------|---------|
| "The crash frame is where the bug is" | The crash frame is where the bad value was *noticed*. Trace up. |
| "The value looks fine in the log" | The log may print a formatted/summarized view. Log the raw representation at the origin. |
| "Two frames both could be the origin" | They are two hypotheses. Minimal-test each; the trail keeps the verdicts. |
| "It works in the test, so the origin is the production environment" | The environment difference *is* an origin: config, data, timing, or version. Find which, don't name it vaguely. |
