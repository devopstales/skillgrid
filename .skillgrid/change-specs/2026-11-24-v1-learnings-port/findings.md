# Audit: v1 `skillgrid-skills` (branch `feature/skill`) → v2 rewrite

**Date:** 2026-11-23 (scouts) · 2026-11-24 (execution + follow-ups)
**Sources read:** `/data/git/AI/skillgrid-skills` (v1, read-only) — `qa/`, `requesting-code-review/`, `parallel-code-review/`, `receiving-code-review/`, `test-driven-development/`, `test-driven-verification/`, `reflect/`, `sketch/`, `docs/user-guide/`, `scripts/test-hooks.sh`; v2 repo (`feat/v2-rewrite`) — full skill tree, `sdd-structure.md`, `git/hooks/`, `omp/hooks/`, `.skillgrid/`.
**Method:** 3 read-only scout agents (content gaps `SkillContentGaps-2`, hook/CI gaps `HookAndCiGaps`, accrued learnings `AccruedLearnings`), results consolidated into the P0–P3 report the user approved for execution.

## Framing

v2 inherited v1's *structure* (phase chain, gates, spec zone, resume protocol)
but not all of its *substance*. Three gap classes, in priority order:

1. **Content v2 lacks entirely** — the TDD canon, the review-lens set, the
   findings/rigor protocols, the concepts doc. Porting was mechanical but
   required home decisions (below).
2. **Contracts v2 references but never implemented** — the three
   drift/lock scripts `sdd-structure.md` specifies by path and exit code,
   and the guard-hook test layer v1 proved with `scripts/test-hooks.sh`
   (20 cases) and v2 had not.
3. **Accrued mechanics v1 carried and v2 dropped** — the ship GO/NO-GO
   decision, the session-start router injection, the code-index
   orientation ladder + TTL bars.

**Home decisions that shaped the port** (the non-obvious calls):

- **Review home = the `test` review area, not a new `code-review` skill.**
  `ship/SKILL.md` already states test "wrote … plus `review.md`";
  `adversarial-authoring` is the cross-model *council* (writes
  `<basename>.council.md`) — a separate chain stage, not a replacement.
  The 8 lens templates therefore live in `test/references/reviewers/`.
- **`sketch` = a mechanic, not a skill.** v2's own statements say "sketch
  still inline in blueprint" — ported as
  `blueprint/references/sketch-phase.md` (variant mechanic, viewport
  toolbar, labeled synthesis, winner marking, findings contract).
- **TDD canon split three ways:** the skill `test-driven-development/`
  (procedure), `_shared/references/strict-tdd.md` (evidence table /
  assertion rules), `_shared/references/test-driven-verification.md`
  (completion-claim canon — a reference, not a skill, cited by debug,
  execute, and the gate-stop hook).

## P0 — content ports (executed)

| Ported | v1 source | v2 home |
|---|---|---|
| TDD skill (Iron Law, RED/VERIFY/GREEN/REFACTOR, triangulation) + writing-good-tests | `test-driven-development/` | `.agents/skills/test-driven-development/` |
| strict-TDD canon | qa + test-driven-verification | `_shared/references/strict-tdd.md` |
| 8 review lens templates (standards, spec, security, edge-cases, verification-gap, accessibility, performance, red-team) — zero checklist items dropped | `requesting-code-review/references/`, `parallel-code-review/reviewers/` | `test/references/reviewers/` |
| handling-findings / test-rigor / verification-gap + test-driven-verification canon | `receiving-code-review/`, `qa/references/`, `test-driven-verification/` | `test/references/` + `_shared/references/` |

The 4th verification-gap shape (**missing-oracle**) lived in v1
`qa/SKILL.md:116`'s verdict table, not in the reference — ported verbatim.

## P0 — dangling-reference redirects (executed)

10 retired v1 skill names (qa, code-review, reflect, resume,
using-skillgrid, writing-blueprints, receiving-code-review,
work-unit-commits, test-driven-verification, environment-retro). The
scout list was **truncated**: the real surface was ~70 occurrences across
~35 files (3× the initial list). Substitution table applied
contextually; `based_on:` citations untouched (they are provenance, not
live refs); explicit v1→v2 mapping lines kept in `docs/06-concepts.md`.
Also fixed: 3 stale `qa/scripts/` paths (the drift-check references and
the size-budget script's self-reference).

## P1/P2 — mechanisms (executed)

- **Ship GO/NO-GO** (`ship/SKILL.md` v2.5): review-waived non-trivial
  changes get a fresh review fan-out *before* the verdict (Trivial-skip
  threshold from `rigor-tiers.md`), 4 ordered verdict rules, rendered
  before the Step-6 integration menu, recorded in the Step-8 envelope.
- **Session-start hook** (`omp/hooks/hooks/session-start.js`): injects the
  router as a priority-IMPORTANT SessionStart payload; resolves via
  `SKILLGRID_ROUTER` → flat → categorized install paths; degrades to
  INFO, always exit 0. Installed copy patched.
- **Guard proof layer**: `scripts/test-hooks.mjs` (44 hermetic cases over
  every blocking hook, incl. 2 end-to-end `git commit` orchestrator
  cases), `scripts/check-lens-contract.mjs` (8/8 templates),
  `.github/workflows/hooks-ci.yml` (no path filters — hook drift lands in
  any diff).

## P3 — docs & governance (executed)

`docs/06-concepts.md` (every named invariant v2-ified + closing v1→v2
mapping table), `blueprint/references/sketch-phase.md`, `CONTRIBUTING.md`
(v1 quality bar on v2 machinery), `docs/00-start-here.md` reading order,
`docs/01-skills.md` inventory.

## Findings made during execution (2026-11-24)

1. **The three drift/lock scripts exist nowhere** — not in v1 as code, not
   in v2, not in git history. `sdd-structure.md` is the sole spec (three
   checks, exit codes 0/1/2, advisory lock, 2h stale threshold). →
   Implemented: `test/scripts/state-drift-check.mjs`,
   `test/scripts/ship-drift-check.mjs`, `scripts/state-lock.mjs` + 113
   hermetic test cases + `test` Step-3 wiring (`report.md` gained
   `## State Drift` / `## Structure Drift`; test v2.5).
2. **First real guard run found 4 genuine `state.yaml` drifts**:
   mnemonic-code phase `intake` vs derived `spec`; mnemonic-core and
   pipeline-envelope missing from the resume map; pi-worktree-plugin entry
   surviving its own archive move. All fixed; guard now `DRIFT: clean`.
3. **Dead detached-HEAD branch in `precommit-guard.js`**: modern git's
   `rev-parse --abbrev-ref HEAD` reports `"HEAD"` while detached, so the
   unborn-branch allowance swallowed the case before the FATAL branch.
   v1's bash guard blocked it via `symbolic-ref`. → Guard now consults
   `symbolic-ref --quiet HEAD` first; the pinned test (#44, previously
   rc 0) flipped to rc 1 + `/detached/`; full suite 44/44.
4. **Cross-ecosystem references with no v2 home**: `debug` →
   `skillgrid:regression-dog`, `refactor` → `skillgrid:codebase-design` /
   `codebase-inspection`. Not v1 folds, not v2 skills — third ecosystem.
   Left as-is (user call, Open below).
5. **Pre-existing defects surfaced by the ports**: `research` committed
   22B over its size ceiling (verified at HEAD); duplicate `version:`
   frontmatter keys in ship, test, and research. All fixed.
6. **mnemonic-code spec converged on v1's contracts** (separate change):
   new R14 agent orientation contract — v1's Orientation Ladder adapted
   to the 7-tool surface (`code_status → code_search → code_context →
   code_explain`), the `fresh|lag|empty|unknown` freshness enum, trust
   gating, no-speculative-read rule; the web-cache TTL bars
   (context7 720h / exa 168h / deepwiki 336h / fetch 168h) named
   out-of-scope with their ownership (memory/core store contract).

## Verification floor (all green 2026-11-24)

`test-hooks.mjs` 44/44 · `check-lens-contract.mjs` 8/8 · drift/lock
suites 113/113 · `state-drift-check.mjs` `DRIFT: clean` ·
`check-frontmatter.mjs` 22/22 · `skill-size-budget.mjs` 22/22 ·
acceptance grep (10 retired names) zero live matches.

## Open (user calls — the change is in_progress at this gate)

1. **Activate the router live**: run the v2 `skillgrid install` CLI (it
   mirrors this tree into `~/.skillgrid/`, replacing the v1 install) or
   set `SKILLGRID_ROUTER` — the installed tree is the v1 categorized
   install, so session-start currently emits its INFO fallback.
2. **Cross-ecosystem refs** (finding 4): keep as-is or fold.
