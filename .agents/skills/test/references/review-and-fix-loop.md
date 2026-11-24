# Review and Fix Loop

Does the code follow standards and implement the spec, and are the findings fixed with evidence? This area runs the topology-driven code review, triages the findings, and closes them in the fix loop. It is the **standards/spec gate** — complementary to the intent area (the quality gate). They answer different questions: intent verifies the feature exists; review verifies the code is clean and on-spec.

**When NOT to run this area:** T0 changes (no review, per `_shared/planning/rigor-tiers.md`), or an empty diff (stop — there is nothing to review). Record `N/A — <reason>`.

## Pin the diff

```bash
BASE_SHA=$(git merge-base <base-branch> HEAD)   # or the SHA the work started from
HEAD_SHA=$(git rev-parse HEAD)
git diff --stat $BASE_SHA..$HEAD_SHA
```

Capture the changed-line count **and the file types touched**. If the diff is empty, stop.

## Bar guards (the diff must not weaken the bar)

An agent that writes the code and the checks takes the cheapest road to green. This pass watches the diff for the five moves that lower the quality bar instead of fixing the code. It runs on the pinned diff, before the lenses are dispatched, and its findings are **required** findings (they do not go through the triage buckets — a weakened bar is fixed in this pass or the change does not merge):

| # | Move | What to look for in `$BASE_SHA..$HEAD_SHA` |
|---|------|---------------------------------------------|
| 1 | **Threshold moved** | `config.yaml` (or the project's constraints file) edited: a quality threshold lowered, a severity dropped, a gate removed or zeroed. Compare against the branch point. |
| 2 | **Test got easier** | `.skip`/`.only` added, a test file deleted, assertions pulled out of a test that stayed, a fixture changed to match the defect. |
| 3 | **Checker silenced** | New suppression comments (`@ts-ignore`, `eslint-disable`, `# noqa`, `# type: ignore`) — and especially the four that switch off a check rather than a line: `istanbul ignore` (drops code from coverage), mutation-testing disables, `nosemgrep`/`gitleaks:allow` (silence security findings). |
| 4 | **Work unfinished** | A stub that throws `Not implemented`, an empty `catch {}` turning a failure into silence, a `TODO` standing where the implementation should be. |
| 5 | **Exception appeared** | A new waiver/exception row (in the change's `briefing.md` waiver block, a `CONSTRAINTS`-style exceptions table, or a config override) that nobody discussed. |

**Placement:** tightening the bar is silent; loosening it is a finding. A threshold change in the *same* diff as the code that needed it is the tell — the fix and the loosening arrive together. Each finding names the file:line, the before/after value, and the move number.

**Ratchet, not aspiration:** when a threshold has no defensible target number (the project measures "today's value"), the bar is a ratchet — record the current value, refuse to let it fall. A diff that moves a ratchet down is move #1 even without an explicit threshold edit, because the recorded baseline changed.

**The circularity check (once per change, at T2+):** the bar must include at least one check the agent cannot satisfy by writing code — an external scanner (trivy, jscpd, the test runner against a held-out test, a lint config the change does not touch). A bar made entirely of checks the change itself edits is self-referential; flag it as a WARNING in the review signal when no external check is present.

## The Topology Step

Select the fan-out from diff size / risk / rigor tier, never from a choice between skills. Per `_shared/planning/rigor-tiers.md` (review escalation threshold):

- **<50 changed lines, low risk** → the 3-axis pass (Standards + Spec + Security).
- **≥50 changed lines, or high-risk** (auth, data migration, money, concurrency, public API) → the core five (Standards, Spec, Edge cases, Verification gaps, Security), plus conditional Accessibility (diff touches UI) / Performance (diff touches data access / rendering) specialists.
- **Red team always last** (reads the other specialists' merged findings).

The line-count floor gates the *core* five, not the domain specialists: a small diff that touches UI or data access still earns the Accessibility or Performance specialist on top of the 3-axis pass. The high-risk categories and the trivial-skip rule live in `_shared/planning/rigor-tiers.md` (review escalation threshold) — do not restate the numbers here. Say which lenses you ran and why, including any you skipped for lack of surface.

**T3 fresh-model rule:** at T3, at least one reviewer runs on a model different from the one that implemented the change (a model reviewing its own work carries its own blind spots). The fresh-model reviewer is a recommendation the human can decline; the decline is recorded.

## Dispatch

Launch **every selected lens in a single message** (multiple subagent calls) so they run in parallel, each with fresh context and no bias from the others. Use a `general-purpose` subagent per lens. Each lens's prompt is: the lens template, the diff scope (`$BASE_SHA..$HEAD_SHA` passed as a **path**, not inlined text — each lens reads it), and the Blindness Contract.

The lens set (templates in `references/reviewers/`):

| Lens | Focus | Selected when |
|------|-------|---------------|
| Standards | glossary, in-force ADRs, smell baseline | always (3-axis + core five) |
| Spec | per-scenario BDD, missing/partial, scope creep | always (3-axis + core five) |
| Security (persona: `.agents/agents/security-sentinel.md`) | injection, authz, exposure, secrets, dep risk (OWASP 5-step) | always (3-axis + core five) |
| Edge cases | unhandled branches, boundaries, deletion regressions | core five |
| Verification gaps | does verification actually catch a break? | core five |
| Accessibility | keyboard, semantics, forms, ARIA, contrast | diff touches UI |
| Performance | N+1, hot paths, network, caching, rendering | diff touches data access / rendering |
| Red team | what the others missed | always, runs last |

**When the diff touches a REST surface** (a new/changed resource, route, contract, or status code), the Standards and Spec lenses review it against the conventions in [api-design.md](../../execute/references/api-design.md) — resource naming, semantic status codes (200-for-everything, missing `Location` on 201, 500-for-validation, 401-vs-403), the response envelope, and the breaking-vs-non-breaking versioning decision. A breaking change without a new version is a CRITICAL finding, not a suggestion.

### Blindness Contract

The value of a review is its independence from the producer. Every lens is a **fresh subagent context** that sees only the work product — the diff and the sources named for that lens — never the coordinator's session history, rationale, or self-assessment.

- **No leaks.** Do not pass a reviewer your narrative, your "why I did it this way", your confidence, or a prior score/verdict on the same diff. Hand it the diff range plus the sources the template names, and nothing else.
- **Independence grade** (per `verification-calibration.md`):
  - **A** — fresh context, no shared history with the implementer (the normal case for a dispatched subagent).
  - **B** — fresh context but same model family / toolchain as the implementer.
  - **C** — the review happens in the implementer's own context (an inline self-review) or the implementer's prior verdict leaked in.

A **Grade C** review is **diagnostic only**: it can guide a fix but cannot stand as the "independently reviewed" evidence that `ship` or a merge requires. If the runtime cannot isolate a fresh context, record the review as Grade C and say so — never present it as independent. At T3 the fresh-model reviewer reaches a clean Grade A.

## Triage

Do this yourself — the lenses file findings, you render the verdict. They lack the full context to grade severity.

1. **Normalize.** Pull every finding from every lens into one list. Each carries: `location` (file:line), the finding, its **source** (lens name), and the lens's own evidence.
2. **Deduplicate by fingerprint** (`path:line:category`). Findings from two or more lenses pointing at the same defect collapse into one entry, tagged `MULTI-REVIEWER CONFIRMED (a + b)`. Multi-confirmation raises confidence it's real; it does not double the severity.
3. **Group by root cause.** Two findings belong in one entry only when the same defect produced both. The entry's severity is the highest of its members; `source` is the contributing lenses joined with `+`.
4. **Verify each entry at its location.** Read the cited `file:line` and read beyond it — follow callers, upstream guards, the surrounding code — until you can answer: does the bad outcome actually occur? Judge whether the problem is real, not whether the proposed fix is plausible.
5. **Re-grade severity yourself.** A lens's self-assigned severity is dropped — reviewers operate under information asymmetry (they lack the spec context, the project constraints, and the consumer's risk posture) and cannot set the final grade. Assign severity by consequence for the artifact's main consumer (the end user of the program, not the developer): `high` = intolerable, `medium` = tolerable but should be fixed before merge, `low` = cosmetic. Multi-reviewer confirmation raises confidence the finding is real; it never raises the grade.
6. **Render exactly one verdict per entry** from the six-value table below.

### The Verdict Table (six values)

| Verdict | Meaning | Ledger severity (record boundary) |
|---------|---------|-----------------------------------|
| `high` | The bad outcome is real and intolerable (bug, data loss, security break, broken feature) | `CRITICAL` |
| `medium` | Real, tolerable — should be fixed before merge | `WARNING` |
| `low` | Real, cosmetic or negligible | `SUGGESTION` |
| `false` | You checked; the bad outcome does not happen here. Say what disproves it. | — (drop the entry) |
| `maybe-false` | You couldn't tell. Say what would settle it. | — (human look) |
| `could-not-verify` | The check is **not inferable from the diff alone** — it needs a held-out test, a runtime, or state the diff doesn't show. Never a confident pass. Emit the concrete "write a held-out test" action that would settle it. | — (human look / held-out test) |

**Reject** (drop) any entry that is `false`; any `low` whose fix adds more complexity than the harm it removes; or any entry whose only fix is to edit the spec under review. `could-not-verify` entries are **never dropped** — they route to a human look / held-out test.

## The Fix Loop

Route every surviving entry into the fix loop below — triage buckets, fix, validate, repeat (receiving-side protocols: [handling-findings.md](handling-findings.md)).

### 1. Triage (the human's call)

**Route by root cause first, then by bucket.** For each surviving entry, ask where the root cause lives (per `_shared/craft/review-ledger.md` § Root-cause routing):

- **Root cause in the code** → triage into the buckets below.
- **Root cause in the spec** (the briefing / acceptance scenario / blueprint said the wrong thing, or was ambiguous enough to produce the defect) → **spec amendment, not a code patch**: amend the spec artifact, record the amendment in the change's change log (blueprint `## Change Log` or `findings.md`) with the triggering finding + the known-bad state it avoids, and re-derive the affected code (fix-now path). When in doubt between a code fix and a spec amendment, prefer the spec amendment — a patch that works around an underspecified requirement is a defect with a coat of paint.
- **Root cause in human intent** (scope, approach, or acceptance must be renegotiated) → HALT: one question, yield (per `_shared/craft/review-discipline.md`), record the ruling in `briefing.md`. A fix that renegotiates a recorded briefing decision without the human's ruling is scope creep.

Then, for code-root-cause entries, sort findings before touching code. If direction is unclear, surface them grouped and ASK rather than fixing everything by default:

| Bucket | Meaning | Action |
|--------|---------|--------|
| **Fix now** | Real, in-scope, belongs with this change | Fix in this pass |
| **Defer** | Real but later; don't bloat this change | Log as a tracker issue or note in the spec |
| **Human look** | Needs manual inspection or testing before trusting | Flag it, don't silently auto-fix |
| **Noise** | Won't-fix or misread | Say why, drop it |

The mapping from the verdict table: `high`/`medium` in-scope → fix now; `high`/`medium` pre-existing or out of scope → defer; anything you can't confirm → human look; `low`/`maybe-false` you chose to drop → noise. Don't let the reviewer dictate scope — "real, but later" is a valid and common call.

**Per-finding disposition (per `_shared/rules/user-gates.md`):** the coordinator takes a citable stance on **every** entry before the user adjudicates — the disposition column of the report table carries it: **PROPOSE FIX** (the fix to apply, with the exact change), **CONTEST** (the finding is a misread — cite the file:line or spec line that disproves it; "looks intentional" is not evidence), **ESCALATE** (settling it needs knowledge outside the diff — a team convention, a product intent — state what fact is missing; this needs the user's knowledge, not their approval), **DECLINE** (a `low` not worth fixing — record why). When the user answers **interactive breakdown** (item-by-item walkthrough), the per-item options follow the gate: PROPOSE FIX → Accept / Skip / Discuss; CONTEST → Agree / Overrule (apply fix) / Discuss; ESCALATE → Confirm (no change) / Add the fact to the spec / Discuss. A batch "auto accept" applies all PROPOSE FIX entries and skips the rest. The user's per-item rulings are recorded in the review ledger's disposition column.

### 2. Fix the "fix now" set — one at a time

For each: restate what was wrong (verify you agree with the finding) → make the fix → create and run a test that proves it (TDD: the test was RED before the fix) → commit.

### 3. Validate

Run the full verification suite (test runner + lint + build per config). If a fix broke something, the finding was deeper than it looked — re-triage that item.

### 4. Log the rest

- **Deferred:** append a `DW-<n>` entry to `.skillgrid/deferred-work.md` (per `_shared/craft/deferred-work.md` — the `parked` row in the change's `review-ledger.md` and the `DW-` entry are written in the same edit), and, if `ticketing.enabled: true`, also publish a tracker ticket. The `DW-` entry is the cross-change sink: a deferral that lives only in this change's ledger is invisible to the next change.
- **Human look:** flag in the final report with what to inspect.
- **Noise:** state the reason so the reviewer (or a future reviewer) sees the ruling.

### 5. Repeat if findings remain

If validation reveals new issues or a fix was incomplete, re-triage the residuals and loop. Cap per `_shared/planning/rigor-tiers.md` (fix loop cap) — beyond it, surface the remaining items to the human with the rulings you made and why.

### The response pattern (no performative agreement)

When acting on a finding: read it without reacting, restate the requirement, verify against the codebase, evaluate whether it's technically sound for *this* codebase, then respond with a technical acknowledgment or reasoned pushback, then implement one at a time. **Never** "You're absolutely right!", "Great point!", or "Let me implement that now" before verification. If a suggestion seems wrong, push back with technical reasoning; if you can't verify, say so and ask.

## Resilience Rule

A failed lens is `not run` (per `_shared/rules/user-gates.md` honesty tags), logged by name: if a lens fails, times out, or returns empty, record its name and continue with the survivors — a `not run` lens is never "zero findings". If **all** lenses fail or return empty, the review is **"possibly incomplete", never "clean"** — report that it may be incomplete and list which lenses didn't run. A reviewer that finds nothing must say why (an early-exit phrase, e.g. "No spec drift: all scenarios re-traced against the diff") — an empty findings list with no reason is a failed review.

## Output Contract

1. **Report the round.** Present the lens reports side by side under per-lens headings (verbatim or lightly cleaned), each ending with the finding count + worst issue *within* that lens. Never pick a single cross-lens winner. Then the deduplicated/grouped entries with their verdicts and source tags, and what was fixed vs deferred vs needs a human look. If any lens failed to run, say so before the verdict.
2. **Write the audit record.** Fill the review template and write it to `.skillgrid/change-specs/<topic>/review.md`, next to `report.md`. It records the lenses side by side, the "What Important means" threshold, the capped Minor findings, each lens's independence grade, and the floor verdict. **Commit it in the spec zone** before continuing code-zone work — a review that lives only in the transcript is not an audit record.
3. **Record the review signal.** In the change's `tasks.md` `## Review`, record exactly one verdict: **`REVIEW-PASS`** (no unresolved `high`/`medium` entries) or **`BACK-TO-APPLY`** (unresolved entries itemized with `file:line` + evidence), plus the finding counts — `Critical N · Warning N · Suggestion N`. This is the contract `skillgrid:ship` consumes.
4. **Record a ledger round.** Per `_shared/craft/review-ledger.md`: freeze the candidate, then append a round entry to the change's `review-ledger.md` with the findings table (severities normalized to `CRITICAL` / `WARNING` / `SUGGESTION`), dispositions (fixed / parked / waived), and lineage. Every `parked` row writes its matching `DW-<n>` entry to `.skillgrid/deferred-work.md` in the same edit (per `_shared/craft/deferred-work.md`). Commit the ledger and the ledger entry together with the review round.
5. **Update ticket status (if `ticketing.enabled: true`):** set the reviewed ticket → `done`; if all tickets in the epic are `done`, set the epic → `done`.
6. **Update `state.yaml`:** set `changes.<topic>.phase: review`.

## Writing the Report Section

```
## Review
Topology: <3-axis / core five / core five + specialists> — <selection rationale>
Lenses: <ran list> | <failed/empty list, by name>
Independence grade: <A / B / C per lens; C = diagnostic only>
Bar guards: <clean | <N> findings: move #<n> <file:line> <before>→<after>, …>

| # | Verdict | Location | Finding | Source (lens) | Disposition | Ruling |
|---|---|---|---|---|---|---|
| 1 | high | <file:line> | <finding> | security | PROPOSE FIX / CONTEST / ESCALATE / DECLINE | <evidence or reason; user ruling when interactive: accepted / overruled / confirmed> |

Review signal: REVIEW-PASS / BACK-TO-APPLY — Critical N · Warning N · Suggestion N
Fix loop: <rounds>, cap per _shared/planning/rigor-tiers.md
```
