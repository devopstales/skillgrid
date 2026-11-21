# Test Report — <change-dir-name>

> Filled by `skillgrid:test`. The four area sections are written in order
> (intent → functional → code quality → security); the verdict is rendered
> from the four-state gate and the floor. Commit this file (spec zone) after
> the verdict is rendered.

## Header

- **Change:** <change-dir-name>
- **Target URL:** <url> / (none)
- **Effective tier:** T<n> (source: briefing `Tier:` line / blueprint `Tier:` / `rules.tiers.default` / inferred)
- **Applied floor:** L<n> (per `_shared/planning/rigor-tiers.md`; classification floor if it exceeds the tier)
- **Date:** <YYYY-MM-DD>

## Test Plan

| Behavior / Truth | Source (briefing / scenario) | Layer | Test | Cadence |
|---|---|---|---|---|
| <behavior> | <briefing truth / scenario name> | <unit/integration/e2e> | <test name> | <pr/nightly> |

Out of scope: <named behaviors NOT tested, and why>

## Intent

<!-- references/intent-testing.md -->
Tier: T<n> (floor: L<n>)

| Truth (briefing) | Level | Named test | Status |
|---|---|---|---|
| <falsifiable claim> | <Truth/Artifact/Key Link/Data Flow> | <test name> | VERIFIED / PRESENT_BEHAVIOR_UNVERIFIED / UNVERIFIED |

Scenarios: <N> total, <M> covered+passed, <K> gaps (<names>)

## Functional

<!-- references/functional-testing.md -->
Suite: <command> → <exit code>, <N> tests, <K> failing
P0: <pass %>/<floor>   P1: <pass %>/<floor>

| Behavior | Layer | Test | Ran? | Result |
|---|---|---|---|---|
| <behavior> | <layer> | <test name> | yes/no | pass/fail |

Browser pass: <ran / N/A — no reachable URL> — <findings>
Regression: <N> behavioral deltas, <K> without a covering test (<names>)

## Code Quality

<!-- references/code-quality.md -->
jscpd: <pass(es) run> on <target> → <dup%> (<exact N, renamed N, near-miss N>)
Hotspots: <top file/folder, dup%>
Health: <score> (dup <n>, dead <n|n/a>, cx <n>)
Dead code: <N> findings (<top>, confidence)

| Gate | Command | Threshold | Result | Status |
|---|---|---|---|---|
| Coverage | <cmd> | <n> | <pct> | PASS / FAIL / N/A |
| Mutation | <cmd> | <n> | <pct> | PASS / FAIL / N/A |
| Lint | <cmd> | — | <n errors> | PASS / FAIL |
| Typecheck | <cmd> | — | <n errors> | PASS / FAIL |
| P0 pass rate | <cmd> | <n> | <pct> | PASS / FAIL |
| P1 pass rate | <cmd> | <n> | <pct> | PASS / FAIL |

## Security

<!-- references/security.md -->
trivy: <command> → <N> findings (<C> critical, <H> high, <S> secrets)
owasp: <N> findings (<severity breakdown>)
nuclei: <ran / N/A — no target> → <N> leads triaged, <K> confirmed
wapiti: <ran / N/A> → <N> hits triaged, <K> confirmed
akca:   <ran / N/A> → <N> verified findings

| # | Sev | Title | Tool | Location | Confidence |
|---|---|---|---|---|---|
| 1 | <Sev> | <title> | <tool> | <loc> | Confirmed / Likely / Needs verification |

## Findings

<!-- one block per finding, highest severity first; the format is the
     "finding block" in references/security.md and the status definitions in
     references/intent-testing.md -->

### CRITICAL
<finding blocks, or "none">

### WARNING
<finding blocks, or "none">

### SUGGESTION
<finding blocks, or "none">

## Gate Decision

- **Verdict:** PASS / CONCERNS / FAIL / WAIVED
- **Floor (weakest dimension):** <area> → <its verdict>
- **Criteria met / unmet:** <list>
- **Human decision (if any):** <override / waiver, the criterion, the accepted risk>

## Project Wiki

<!-- references: skillgrid:projectwiki — file only durable, non-obvious knowledge;
     the four-state verdict itself is NOT a wiki page -->
- **Filed:** <path — one line each, or> nothing filed — no durable, non-obvious knowledge surfaced
- **Index:** <projectwiki index / validate / lint ran; orphan count not growing>

## Human Override

- **Override:** <the human's decision, or "none">
- **Waiver (if WAIVED):** <criterion waived, the accepted risk, recorded by>
