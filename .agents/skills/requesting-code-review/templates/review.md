# Review — <topic>

> Change: `.skillgrid/specs/YYYY-MM-DD-<topic>/` (moves to `.skillgrid/archive/YYYY-MM-DD-<topic>/` at ship)
> Generated: <ISO 8601 timestamp> (requesting-code-review)
> Diff range: `{BASE_SHA}..{HEAD_SHA}`
> Rigor tier: T<n> (per `_shared/planning/rigor-tiers.md`)
> Independence: <Grade A/B/C per axis — see `## Independence`>
>
> The durable audit record of the code review. **requesting-code-review** writes this into
> the spec folder, next to `report.md` (which owns the QA gate). It is committed in the spec
> zone before code-zone work continues. **ship** archives it with the folder; **reflect**
> cites it for lineage. The three review axes are reported **side by side, never merged** —
> that separation is the point.

## What Important means

> The explicit threshold that separates a finding worth fixing now from one to park.
> Do not restate the fix-loop cap or escalation numbers here — they live in
> `_shared/planning/rigor-tiers.md`. Name only the bar for "Important."

- **Critical** — must fix before merge. Security, data loss, broken functionality, a
  `SATISFIES` scenario whose RED evidence is missing or whose code path does not match the
  Given/When/Then. Blocks `ship`.
- **Important** — should fix before merge. Architecture drift from a term or an in-force
  ADR, a hard standard breach, a missing or partial requirement, a behavior the spec never
  asked for (scope creep). Unfixed Important issues do not proceed (per the skill's
  Red Flags).
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. These are the
  nits; they are capped and summarized, never one-per-line below the fold.

## Cap the nits

- **Max 5 Minor findings listed per axis.** Beyond that, collapse the rest into a count:
  "N further Minor items (style/smells) — see the raw reviewer report, not this file."
- A Minor finding that is actually a Critical dressed as polish gets re-labeled, not
  capped. Capping is for true nits, not for hiding a real issue behind a count.

## Do not report

- Anything tooling already enforces (lint, typecheck, format). If `gofmt` / the linter
  would catch it, it is not a finding here — it belongs to the QA gate's tooling pass.
- Informational context with no required action — put it in `FYI` at most, one line, and
  it does not count against any cap.

## Passes

> Three axes, reported side by side. Never merge them into one verdict — a change can be
> spec-perfect and violate every convention, beautifully written and implement the wrong
> thing, or conventionally clean and spec-complete and still one SQLi away from a data
> breach. Each axis carries its own worst issue and its own verdict.

### Standards

- **Worst issue (within this axis):** <one line, or "none">
- **Findings:** <n> Critical / <n> Important / <n> Minor (capped to 5 listed)
- **Verdict:** met / met-with-fixes / not met

### Spec

- **Worst issue (within this axis):** <one line, or "none">
- **Findings:** <n> Critical / <n> Important / <n> Minor (capped to 5 listed)
- **Verdict:** met / met-with-fixes / not met

### Security

- **Worst issue (within this axis):** <one line, or "none">
- **Findings:** <n> Critical / <n> High / <n> Medium / <n> Low / <n> Info (capped to 5 listed)
- **Verdict:** secure / secure-with-fixes / not secure

## Findings

> One per line where possible. Every finding names its file + hunk (or scenario) so a
> reader can act without the session. Severity is the reviewer's; the label prefix
> (`Critical:` / `Optional:` / `Consider:` / `FYI`) is the reviewer's action marker.

### Critical

- [axis] <file:hunk or scenario> — <what's wrong, name the smell/standard> — <why it matters> — <how to fix, if not obvious>

### Important

- [axis] <file:hunk or scenario> — <what's wrong> — <why it matters>

### Minor

- [axis] <file:hunk> — <smell>
- (N further Minor items collapsed — see the raw reviewer report)

## Independence

> The no-leaks proof. Each axis records its independence grade so a reader can see whether
> the "independently reviewed" evidence is real. Mirrors the evaluator protocol in
> `_shared/verification/calibration.md`.

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | A / B / C | fresh context / same-model-family / inline-or-leaked |
| Spec | A / B / C | fresh context / same-model-family / inline-or-leaked |
| Security | A / B / C | fresh context / same-model-family / inline-or-leaked |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker). **C** =
inline self-review or the implementer's verdict leaked in — **diagnostic only**, cannot stand
as the independent evidence that `ship` or a merge requires.

## Verdict

> The overall review verdict is the **floor** across the three axes (per
> `_shared/verification/floor.md`): the weaker axis caps the whole review. Never average the
> three axes into a single "overall."

- **Standards:** met / met-with-fixes / not met
- **Spec:** met / met-with-fixes / not met
- **Security:** secure / secure-with-fixes / not secure
- **Floor (decides):** met / met-with-fixes / not met
- **Worst issue (across all three axes):** <one line>
- **Unfixed Important count (must be 0 to proceed):** <n>
