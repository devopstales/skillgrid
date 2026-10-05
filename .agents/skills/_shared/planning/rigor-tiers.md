# Rigor Tiers (shared convention)

Match the verification effort to the risk. A throwaway prototype self-checks; a payment system runs the full gauntlet. The tier is a **rigor dial per change**, not a track you're locked onto — every step it drives is still overridable by the human, and an override is recorded, never silent.

## The tiers

| Tier | Name | Blueprint | Acceptance | QA floor | Review |
|---|---|---|---|---|---|
| **T0** | Prototype | none (direct execution from briefing) | skipped (record the skip) | self-check evidence only (L1, no gates) | none |
| **T1** | Alpha | light (header + Must-Haves, no threat matrix unless applicable) | required | L1 + runtime verification (verify the behavior runs) | none |
| **T2** | Beta | full | required | L2 + L3 (full suite, build, coverage, lint/typecheck) | two-axis (`skillgrid:requesting-code-review`) |
| **T3** | GA | full | required | L4 (mutation + security) + all thresholds | `skillgrid:parallel-code-review`, at least one reviewer on a **fresh model** (different from the implementer's) |

The QA floor references [verification-ladder.md](verification-ladder.md) L1–L4. T2 is the project default; T3's fresh-model rule exists because a model reviewing its own work carries its own blind spots.

## Precedence

1. **Per-change override** — `briefing.md` carries a `Tier: T<n>` line. It wins over the project default.
2. **Project default** — `rules.tiers.default` in `.skillgrid/config.yaml` (set at onboarding, recommended from the stated product's risk/size; never re-asked once set).
3. **Inference** — if neither records a tier, infer from the change classification below and note the assumption in the briefing.

## Relationship to fast-track and change classification

Three dials, three different questions:

- **Fast-track** (trivial / small) — micro-changes that skip the *planning* chain (brainstorming, blueprint). See [fast-track.md](fast-track.md). A fast-track change never carries a tier; the waiver record is its policy.
- **Change classification** (trivial → high-risk) — a property of the change's shape (files, migrations, trust boundaries). It sets the *verification floor* independently of the tier.
- **Rigor tier** (T0–T3) — a property of what the *project* wants verified for this change. It selects which of the pipeline's tail steps (acceptance, QA gates, review topology) run.

**The floor is the max of the two.** A T0 prototype that happens to be a 6-file refactor with a migration still gets the migration's L3 floor — the tier lowers the *default* verification, it never lowers the floor the change classification demands. When the classification floor exceeds the tier's floor, the classification wins and the report notes which gates the tier would have skipped.

## What each skill does with the tier

- `skillgrid:writing-blueprints` — records `Tier: T<n>` in the blueprint header; selects which blueprint sections are mandatory (T1: light header, no threat matrix unless applicable; T2/T3: full).
- `skillgrid:slicing` — reads the tier from the blueprint/briefing; T0 skips slicing (direct execution), T1 may use a light `tasks.md`.
- `skillgrid:qa` — the tier selects the verification floor and which audits run (T0: self-check evidence only; T3: mutation + security mandatory). The change-classification floor still applies as the minimum.
- `skillgrid:requesting-code-review` / `skillgrid:parallel-code-review` — the tier selects the review topology (T2: two-axis; T3: parallel fan-out with a fresh-model reviewer).

## Review escalation threshold

The single source for when a two-axis review escalates to fan-out. Do not restate the numbers elsewhere — reference this section.

- **Escalate to `skillgrid:parallel-code-review` fan-out** when the diff has **50+ changed lines OR is high-risk** (auth, data migration, money, concurrency, public API).
- **Trivial skip** — a change that is **≤ 2 files AND < 50 lines AND touches no auth / payments / data migration / config** skips fan-out; the decision rests on the QA gate plus a one-line rollback.

`skillgrid:parallel-code-review`'s own specialist-selection rule ("< 50 changed lines → Standards + Spec only; ≥ 50 or high-risk → the core five") applies the same threshold at dispatch time.

## Fix loop cap

The single source for fix/review loop caps. Do not restate the round counts elsewhere — reference this section.

- **Gate fix loop** (QA gate FAIL, or a review fix loop: `skillgrid:qa`, `skillgrid:requesting-code-review`, `skillgrid:receiving-code-review`, `skillgrid:parallel-code-review`, `skillgrid:simple-execution`): **3 rounds, then escalate to the human**. Past the cap the failure is structural (architectural or scope), not a test gap.
- **Task-level fix loop** (`skillgrid:subagent-execution`, per task): **5 rounds, then the breaker adjudicates** each open finding (park with a ruling, or rule on the smallest change that unblocks dependent work). Rounds 1-3 resume the original implementer; rounds 4-5 dispatch a fresh implementer on a more capable model.

## Rules

- A tier is recorded, never re-derived downstream: the briefing (or blueprint) line is the single source of truth.
- T0's skipped steps are **recorded as skipped** in the briefing, never silently absent.
- The tier never weakens the gates that do run — a T3 change gets every T2 gate plus the T3 additions.
- T3's fresh-model reviewer is a **recommendation the human can decline**; the decline is recorded. The model-blind-spot rule is the reason, not a preference.
