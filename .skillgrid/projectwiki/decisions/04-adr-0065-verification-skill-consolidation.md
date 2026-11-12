---
type: TechnicalDecision
status: accepted
accepted: 2026-11-12
supersedes: none
date: 2026-11-12
---

# ADR-0065 — Verification skill consolidation: the after-code review is one skill (`code-review`) whose topology step selects the fan-out; the review ledger is a shared reference, not a skill

Change: 2026-11-02-egm-goldfish-brainstorm-shootdown

## Context

skillgrid carries three verification skills with overlapping roles:

- `requesting-code-review` (T2): the 3-axis pass — Standards + Spec + Security — writes `review.md`, owns the Blindness Contract + independence grades.
- `parallel-code-review` (T3): up to 8 specialists, six-value verdicts, red team last.
- `review-ledger` (v1.0): dispatches nothing, renders no verdict, and gates nothing — a round-entry format, frozen-candidate + invalidation rule, disposition vocabulary, lineage, and debt lifecycle, invoked from exactly three fixed points (after a review round, at park, at qa time).

The two review skills share the same target (the diff) and the same dispatch mechanism (fresh subagent per lens, template-driven). `parallel-code-review` imports `requesting-code-review`'s two-axis templates, and its "<50 changed lines → Standards + Spec only" mode is exactly the requesting pass — the two-skill choice is redundant. They also duplicate the security reviewer template: `requesting-code-review/references/security-reviewer.md` (230 lines: full OWASP 5-step workflow + four-part triage rubric + report format) and `parallel-code-review/reviewers/security.md` (a 70-line variant). The three-gap-shape verification canon exists twice: in `qa`'s audit and in the parallel specialist's prompt.

The owner approved the consolidation (Amendment A2, 2026-11-11): "we have too many validation skills."

## Decision

The after-code review is a single skill (`code-review`) whose topology step selects the fan-out by diff size / risk / rigor tier; the review ledger is a shared reference (`_shared/craft/review-ledger.md`), not a skill; `qa` (gate) and `receiving-code-review` (fix loop) keep their roles; observable review behavior is unchanged.

- **One review skill** (R39): a single skill at `.agents/skills/code-review/`. Its **topology step** (replaces the two-skill choice) selects the fan-out from diff size / risk / rigor tier:
  - **<50 changed lines, low risk** → the 3-axis pass (Standards + Spec + Security).
  - **≥50 changed lines, or high-risk** (auth, data migration, money, concurrency, public API) → the core five (Standards, Spec, Edge cases, Verification gaps, Security), plus conditional Accessibility (diff touches UI) / Performance (diff touches data access / rendering) specialists.
  - **Red team always last** (reads the other specialists' merged findings); the tier floor per `_shared/planning/rigor-tiers.md` still applies (T3: at least one fresh-model reviewer).

  Selection rules carry over verbatim from both skills — nothing is lost.
- **Shared protocol, exactly once** (R39): the Blindness Contract (no-leaks + independence grade A/B/C, grade-C-diagnostic-only) and the six-value verdict table (`high` / `medium` / `low` / `false` / `maybe-false` / `could-not-verify`, mapped to ledger severities), plus the resilience rule (a failed lens is logged by name; all-lenses-failed = "possibly incomplete", never "clean").
- **Output contract preserved** (R39): the `review.md` audit record (template carried over) + `REVIEW-PASS` / `BACK-TO-APPLY` + Critical/Warn counts (the contract `ship` consumes) + the ledger round entry (R41).
- **Security templates merged** (R40): one `code-review/reviewers/security.md` — the full OWASP 5-step workflow + four-part triage rubric + report format (citing the `owasp-security` skill) — so the 3-axis pass and the full fan-out dispatch the same security specialist. The 8 specialist templates live under `code-review/reviewers/` (`code-reviewer.md`, `spec-reviewer.md` moved from `requesting-code-review/references/`, `security.md` merged, `edge-case-hunter.md`, `verification-gap.md`, `accessibility.md`, `performance.md`, `red-team.md` moved from `parallel-code-review/reviewers/`).
- **Ledger demoted to a shared reference** (R41): `_shared/craft/review-ledger.md` carries the same contract — round format, frozen candidate + invalidation, dispositions fixed / parked / waived, lineage, debt lifecycle open→addressed→closed / expired, the 3-changes-stale flag. The per-change `review-ledger.md` **artifact** is unchanged. Code-review rounds (R39), Goldfish rounds (R10), and drift-mode findings (R33) record ledger rounds per this reference.
- **Debt surfacing moves into `qa`** (R42): `qa`'s process gains a named "surface review debt" step, carried over verbatim from the ledger skill's Step 5 — read prior archived changes' `review-ledger.md` files, surface open `parked` findings (context, not a gate), WARNING on file overlap. R17's Goldfish path rides the same step. `qa`'s routing table and "When NOT to use" cite `code-review` (topology per tier / diff size).
- **Verification-gap canon hoisted** (R42): the three gap shapes + evidence rule move to `_shared/verification/verification-gap.md`, cited from both `qa`'s audit and the code-review specialist.
- **Clean cutover** (R43): ~15 citation sites update to `code-review` / `_shared/craft/review-ledger.md` (including the `AGENTS.md` workflow line, whose phases are unchanged); `_shared/skill-size-budget.json` carries one `code-review` entry at the post-merge ceiling. No alias, no shim, no back-compat dispatch. The three old skill directories (`requesting-code-review/`, `parallel-code-review/`, `review-ledger/`) are deleted.

## Alternatives considered

- **Status quo (two review skills + ledger skill)** — rejected: too many validation skills (owner, A2); the two review skills duplicate the dispatch mechanism, the Blindness Contract text, and the security templates, and the two-skill choice is redundant — `parallel-code-review`'s small-diff mode is exactly the `requesting-code-review` pass.
- **Keep `review-ledger` a standalone skill** — rejected: it dispatches nothing, renders no verdict, and gates nothing; it is a record contract invoked from three fixed points, and a shared reference is its correct home. Its Step 5 (debt surfacing at qa time) moves into `qa`, where the gate already runs.
- **Keep the verification-gap canon in two places** — rejected: the same canon exists in `qa`'s audit and the parallel specialist's prompt; one home under `_shared/verification/` ends the drift risk between the two copies.

The validation-skill roster goes from four invocable skills (`requesting-code-review`, `parallel-code-review`, `review-ledger`, `qa`) to two invocable (`code-review`, `qa`) + one shared reference (`_shared/craft/review-ledger.md`); `receiving-code-review` (fix loop) is unchanged.

## Consequences

- Good, because one skill reproduces both old selection behaviors — the 3-axis pass and the full specialist fan-out — with the selection rules carried over verbatim, so nothing is lost.
- Good, because the Blindness Contract + grades + verdict table appear exactly once, and there is one security template.
- Good, because the ledger contract is unchanged — its home moves from a skill directory to `_shared/craft/`, but the per-change `review-ledger.md` artifact is untouched.
- Bad, because the three old skill directories are deleted only after ~15 citation sites are remapped: the remap and the deletions are a single cutover that must land in the same task sequence (reversible by `git revert` of the change's commits).
- Neutral, because observable review behavior is unchanged — same dispatches, same outputs, same ledger format; `ship`'s review-signal path is unaffected.
- Neutral, because `qa` (gate) and `receiving-code-review` (fix loop) keep their roles; `qa` gains one step (debt surfacing) and loses nothing.
