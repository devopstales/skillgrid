# Review Ledger (shared across all Skillgrid skills)

Own the persistent review record. Record every review round's findings, dispositions, and fix-commit lineage in a per-change ledger file. The ledger is the audit trail: a shipped bug can be traced back to the review round that caught it (or missed it).

The `review-ledger` skill is demoted to this reference (R41); the old invocation form no longer exists — the fixed points below *cite* this reference.

**Core principle:** A finding without a disposition is open. An open finding without a resolving commit is debt. The ledger makes both visible.

**Config:** Read `.skillgrid/config.yaml` before recording a round.
- `conventions.specs_root` (default `.skillgrid/change-specs/`) — the change's artifacts

## When to Use

**Mandatory — three fixed points cite this reference:**
- After every review round completes (after the `code-review` skill returns findings)
- When a review finding is parked (dispositioned as "parked" with a ruling)
- At `qa` time, to surface open findings from prior changes

**When NOT to cite:**
- T0 changes (no review)
- A review round that found zero findings (record the round as "clean" with no findings table)

## Round entry format

Append a round entry to `.skillgrid/change-specs/<topic>/review-ledger.md`. Create the file if it does not exist.

**Round entry format:**

```markdown
## Round <N> — <YYYY-MM-DD>

**Frozen candidate:** base <hash> → head <hash>, <N> files, hash <hash>
**Reviewer:** <code-review: 3-axis pass | core-five fan-out>
**Independence grade:** <A | B | C>
**Budget:** round <N>/<total> (tier T<n>)

### Findings

| ID | Severity | File:Line | Finding | Disposition | Fix Commit | Lineage |
|---|---|---|---|---|---|---|
| R<N>-F<NNN> | CRITICAL / WARNING / SUGGESTION | <file:line> | <one-line description> | fixed / parked / waived | <commit hash or "—"> | R<N>-F<NNN> |

### Dispositions

- **R<N>-F001** — fixed in `<commit hash>`. <one-line note on the fix.>
- **R<N>-F002** — parked. Ruling: <why it's parked>. <named human>.
- **R<N>-F003** — waived. Risk accepted: <one line>. <named human>.

### Round Verdict

<clean | findings-dispositioned | invalidated | converged>
```

**Finding ID format:** `R<round>-F<NNN>` (e.g. `R1-F001`, `R2-F003`). Sequential within a round.

Goldfish-review rounds are ledger rounds: the `Reviewer` column accepts the Goldfish review, and the debt-surfacing step (at `qa` time) covers parked goldfish findings.

## Frozen candidate + invalidation

Before recording findings, snapshot the exact set of files being reviewed. This is the **frozen candidate** — the review is against this set, not the live working tree.

```
Frozen candidate:
  Base: <commit hash of the review's base ref>
  Head: <commit hash of the reviewed tip>
  Files: <N> files
  Hash: <git hash-object of the file list, sorted>
```

**Rule:** if any file in the frozen candidate changes after the review starts (a new commit, a manual edit), the round is **invalidated** — the findings may not apply to the new state. Invalidate the round, record `invalidated: <reason>` in the ledger, and re-dispatch the reviewer on the new candidate.

A round is invalidated when:
- A new commit lands between the frozen-candidate snapshot and the round's disposition.
- A file in the frozen candidate is edited manually (not through a commit).
- The reviewer's independence grade degrades (e.g. the reviewer's context was contaminated by the implementer's narrative).

On invalidation:
1. Record `invalidated: <reason>` in the round entry.
2. The round's findings are **not** carried forward — they applied to a candidate that no longer exists.
3. Re-dispatch the reviewer on the new candidate (a new round).
4. The invalidated round stays in the ledger for audit — it is not deleted.

## Dispositions

**Disposition values:**
- `fixed` — a fix commit exists. The `Fix Commit` column carries the hash. The finding is closed.
- `parked` — deferred with a ruling. The finding is open. It surfaces in the next change's QA report.
- `waived` — explicitly waived by a named human with an accepted risk. The finding is closed.

## Lineage

**Lineage column:** the finding's own ID. When a fix commit resolves multiple findings, each finding's lineage row names the commit. This is the forward trace: finding → commit.

## Debt lifecycle

A parked finding has a lifecycle:

| State | Meaning | Transition |
|---|---|---|
| `open` | Parked, not yet addressed | → `addressed` when a fix commit resolves it |
| `addressed` | A fix commit exists that resolves the finding | → `closed` when the fix is shipped |
| `closed` | The fix is shipped and the finding is resolved | terminal |
| `expired` | The finding no longer applies (code changed, feature removed) | terminal — record why |

**Who transitions:**
- `open` → `addressed`: the implementer who fixes it records the fix commit in the finding's `Fix Commit` column.
- `addressed` → `closed`: `ship` moves the change to archive; the finding is closed.
- `open` → `expired`: a human explicitly marks it expired with a reason (e.g. "feature removed in <change>").

### Debt surfacing (at QA time)

When `skillgrid:qa` runs for a **new** change, it reads `review-ledger.md` files from **prior archived changes** (`.skillgrid/change-archive/*/review-ledger.md`) and surfaces any findings with disposition `parked`:

```
Review debt from prior changes:
- R1-F002 (2026-09-15-auth-migration): parked — "needs DBA sign-off". Open.
- R2-F001 (2026-09-20-api-rate-limit): parked — "deferred to Q4". Open.
```

This is **context, not a gate**. Open parked findings do not block the new change's QA verdict. They are surfaced so the human sees the accumulated debt and can decide to address it.

If the new change's scope overlaps a parked finding's file, the QA report adds a WARNING: "This change modifies <file>, which has an open parked finding R<N>-F<NNN> from <change>. Verify the finding still applies or close it."

## Stale flag

A parked finding that is `open` for more than 3 archived changes is flagged as **stale** in the QA report: "Finding R<N>-F<NNN> has been open for 3+ changes. Address or expire."

## Loop Termination

A review loop over a frozen candidate runs rounds until it is **decided**
converged — not until a round returns zero findings. Fresh reviewers sample
differently each round, so a zero-finding round is not proof the text is
stable.

- **Clean round:** no CRITICAL or WARNING finding is raised that is not already
  **settled**. SUGGESTIONs (nits) do not block a clean round. A reviewer
  recorded as *not run* (timeout, no report, or no status in the synthesizer
  summary) makes the round NOT clean — a missing reviewer is not a
  zero-finding reviewer.
- **Settled finding:** contested, or recorded as an accepted risk (waived /
  parked with a ruling). The settled list persists across rounds. A settled
  finding raised again at the same scope by a new reviewer is NOT re-opened.
- **Fixed finding:** an applied fix does NOT settle a finding. If a later
  reviewer re-raises the same issue at the same scope, the fix failed or a
  later edit reintroduced it — re-open it.
- **Converged:** two consecutive clean rounds. One clean round is not enough —
  a fresh reviewer can find a different gap in the same text.
- **Churn:** a new finding at the scope of an applied fix, where reverting that
  fix would resolve it. On churn, stop and ask the user.
- **Cap:** 5 rounds. At the cap, or on churn, list the open findings by
  severity and ask the user one question: accept and proceed, continue past
  the cap, or return to drafting.

Record the round's clean/converged status in the `### Round Verdict` line
(`clean | findings-dispositioned | invalidated | converged`).

**Turn boundary:** after a round's resolve phase, the loop driver yields —
zero tool calls, go idle — and re-enters only when the user confirms the next
round. A loop that auto-loops across human turns has not respected the gate.
Per `_shared/craft/review-discipline.md` (zero-tool turn-yield).
## Validation

After appending the round entry, check:

1. **Every finding has a disposition.** A finding with no disposition is a bug in this step.
2. **Every `fixed` finding has a fix commit hash.** The hash resolves in git history.
3. **Every `parked` finding has a ruling and a named human.** "Park it" without a ruling is not a disposition.
4. **The frozen candidate hash matches the files actually reviewed.** If the reviewer saw files not in the frozen candidate, the round is invalid.
5. **The round number is sequential.** No gaps, no duplicates.

If any check fails, fix the ledger entry before committing.

## Commit protocol

Commit `review-ledger.md` with the review round. Use the `work-unit-commits` protocol: conventional subject, `[skillgrid-context]` block.

**Commit message pattern:** `review: record round <N> (<clean | <N> findings, <M> fixed, <K> parked)`)

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | % of review findings dispositioned in the same round (not parked) | `review-ledger.md` | should rise |
| Leading | % of review loops that converge within the 5-round cap (no overrun) | `review-ledger.md` | should rise |
| Lagging | Review debt: count of open parked findings across all archived changes | `.skillgrid/change-archive/*/review-ledger.md` | should fall |

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll record the findings in the review.md instead" | `review.md` is the review's output. The ledger is the **persistent record across rounds and changes**. A finding in `review.md` disappears when the change archives; a finding in the ledger survives. |
| "Parking a finding is the same as waiving it" | No. Parked = "we'll fix it, here's the ruling." Waived = "we accept the risk, here's the human." Different dispositions, different lifecycles. A parked finding is debt; a waived finding is a decision. |
| "The frozen candidate is overkill — just review the current state" | If the state changes mid-review, the findings may not apply. The frozen candidate is the evidence that the review was against a specific, named set of files. Without it, "the reviewer said X" is unanchored. |
| "I'll close the parked finding when I get to it" | "When I get to it" is how findings become permanent. The ledger's stale-detection (3+ changes open) forces the decision: address or expire. |
| "The review debt surfacing is noisy" | It's 2-3 lines in the QA report. The alternative is findings that accumulate invisibly until they become incidents. |

## Red Flags

- A review round with findings but no ledger entry
- A finding with no disposition
- A `fixed` finding with no fix commit hash
- A `parked` finding with no ruling or no named human
- A round that is not invalidated after a commit lands mid-review
- Review debt that grows monotonically across changes (no findings being addressed)
- A finding that is `open` for 3+ changes without a stale flag

## Verification

- [ ] `review-ledger.md` exists in the change dir (created on the first review round)
- [ ] Every review round has a ledger entry with a frozen candidate, findings table, and dispositions
- [ ] Every finding has a disposition (fixed / parked / waived)
- [ ] Every `fixed` finding has a fix commit hash that resolves in git history
- [ ] Every `parked` finding has a ruling and a named human
- [ ] The frozen candidate hash matches the files the reviewer actually inspected
- [ ] The ledger is committed with the review round
- [ ] At QA time, prior-change review debt was surfaced (or "no open findings" recorded)
- [ ] The loop stopped on two consecutive clean rounds, the 5-round cap, or a user decision — not on a single zero-finding round (see `## Loop Termination`)

## Final Rule

```
Review round → frozen candidate → findings → dispositions → ledger entry
Fixed finding → fix commit → lineage binding
Parked finding → ruling + human → review debt → surfaces in next QA
Invalidated round → new candidate → new round (old round stays for audit)
→ The ledger is the audit trail: finding → disposition → commit → ship
```
