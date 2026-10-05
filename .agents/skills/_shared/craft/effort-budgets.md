# Effort Budgets Convention (shared across all Skillgrid skills)

Two related ideas, both borrowed from the honest-overhead posture of GSD Core:
a framework that tells you **when not to use itself** is a framework you can
trust, and a skill that declares **how much context it will spend** keeps the
orchestrator's window predictable.

## 1. The effort signal (`effort:` frontmatter)

A skill MAY declare its expected context spend in its frontmatter:

```yaml
effort: low        # reads minimal state, returns concise output
effort: standard   # a normal skill — the default when the field is absent
effort: max        # a heavy orchestrator: spawns subagents, reads many artifacts
```

- **`effort:` is advisory, not a gate.** It is a *budget signal* for the
  orchestrator and for humans choosing a path, never a verification criterion
  and never a gate the QA skill enforces. It does not appear in the four-state
  gate.
- **When to set it:**
  - `max` — orchestrator skills that spawn subagents and carry the main
    session's context (`using-skillgrid`, `subagent-execution`,
    `parallel-execution`). The cost is real and the signal is honest.
  - `low` — status/read-only skills that read minimal state and return
    concise output (`resume`, `progress`, `qa`'s quick-status reads). Setting
    `low` is a *promise of a small footprint*, not an excuse to under-deliver.
  - absent — `standard`. Most skills. Do not add the field to a skill just to
    have it.
- **Do not fork the orchestrator to save context.** An `effort: max` skill
  that *spawns* subagents gets its context isolation from the subagents, not
  from forking itself (forking would strip it of the `Agent` tool). The signal
  names the cost; it does not change the mechanism.

## 2. When NOT to use the pipeline

The full pipeline is the phase order in [sdd-structure.md](../rules/sdd-structure.md). It is the *maximal* path. It is not the default for every
change. Match effort to risk:

| Situation | Path |
|---|---|
| A task that could be fully specified in **one short prompt** and completed in **one agent turn** with no clarification | Skip the pipeline. Do it directly (or `fast-track` `trivial` if it must leave a trace). |
| A single-file, no-new-capability change | `fast-track` (`small`/`trivial`) — see `fast-track.md`. The full pipeline is overkill. |
| A multi-file feature, a cross-cutting change, or work that spans **hours or sessions** | The full pipeline. This is where context rot becomes a real risk and the pipeline pays for itself. |
| A change you are not sure about, with unsettled decisions | At least brainstorming + blueprints, before execution. The phase loop protects you. |

**The rule of thumb:** if the task is specifiable in one short prompt and
finishable in one turn, the pipeline is ceremony. Reach for the lighter
primitive. The pipeline earns its overhead only when the work is complex enough
that context rot, unverified assumptions, or a late design mistake are real
risks.

**This is the honest-overhead principle.** A workflow that pretends it is always
the right path trains the operator to skip it when it is — and then they skip it
when it *is* the right path. Stating when NOT to use it is what makes the
"when to use it" trustworthy.

## Red Flags

- An `effort: max` on a status/read-only skill (inflating its footprint signal).
- Running the full eight-phase pipeline for a one-line config change.
- Skipping the pipeline for a multi-session, cross-cutting refactor "because
  the phases are slow" — that is exactly the case the pipeline exists for.
- Adding `effort:` to every skill as a habit instead of as a signal.
