---
type: BusinessWorkflow
title: "SDD Pipeline"
---

# SDD Pipeline

The skillgrid spec-driven development cycle, as a concept: the phases, the
zone rule, and the terminal close. This page is the durable *what and why*;
the per-phase mechanics live in the pipeline skills (`.agents/skills/`) and
the machine layout in `_shared/rules/sdd-structure.md`.

## The pipeline

```
brainstorming → writing-blueprints → slicing → ticketing
  → execution (subagent-execution | simple-execution)
  → qa → requesting-code-review → receiving-code-review
  → ship → reflect
```

- **brainstorming** — interview the design tree; write `briefing.md`, the
  per-change ADR Review Manifest, VERIFIED/INFERRED facts to
  `ASSUMPTIONS.md`, and terms/ADRs as decisions crystallize.
- **writing-blueprints** — pure-technical `blueprint.md` (the *how*);
  cites the spec and in-force ADRs, never restates them.
- **slicing** — vertical tracer-bullet tickets in `tasks.md` with
  dependency edges and execution waves.
- **ticketing** — publish tickets to the configured tracker
  (`backlog` type reads `.skillgrid/config.yaml`), track status through
  execution, close at done.
- **execution** — one implementer subagent per ticket (or a simple
  execution session with review checkpoints); TDD + BDD always on —
  each ticket names the `SATISFIES` scenario it drives RED→GREEN.
- **qa** — the quality gate: test suite, verification-gap and
  TDD-evidence audits, OKF lint, four-state verdict
  (PASS / CONCERNS / FAIL / WAIVED).
- **requesting-code-review** → **receiving-code-review** — the two-axis
  review (spec compliance, then code quality) and the evidence-based
  reception of findings; the review ledger records the round.
- **ship** — integrate to the base branch, then the mechanical folder
  move `specs/<topic>/` → `xarchive/<topic>/` (`git mv`, `diff -r`
  readback; content never passes through the model), then close the
  change folder.
- **reflect** — the terminal phase: complete `report.md`'s retro half
  IN PLACE in the archive, persist to Mnemonic, close the session.
  Reflect never re-opens the change.

## The zone rule

Spec-zone changes commit **before** code-zone changes (BDD is always on).
Spec zone: `.skillgrid/specs/**` and `.skillgrid/xarchive/**` (the archive
is the terminal state of spec artifacts). The `pre-commit` zone guard
fails a commit that stages both in one shot; recovery is always the two
commits (spec first, code second).

## The knowledge move (R6, ADR-0048)

The durable knowledge zone was dissolved into the `.skillgrid/` bundle on
2026-10-20: `artifacts/` no longer exists — ADRs live in `decisions/`,
locked constraints in `invariants/`, terms in `glossary/`, PRD/research/
roadmaps in `reference/`, and this workflow lives in `workflows/`.
Closed changes live in `xarchive/` (renamed from `archive/`). The hybrid
source of truth: SQLite (mnemonic) for observations + code intelligence,
the Markdown bundle for the knowledge base (ADR-0048 amending ADR-0012).

## Participating sites

- Phase mechanics: `.agents/skills/planning/`, `.agents/skills/execution/`,
  `.agents/skills/verification/`, `.agents/skills/lifecycle/`
- Layout + machine state: `_shared/rules/sdd-structure.md`
- Zone enforcement: `hooks/precommit-zone-guard.js`
- Drift guards: `scripts/state-drift-check.mjs`,
  `scripts/check-adr-invariants.mjs`, `okf-lint.mjs`
