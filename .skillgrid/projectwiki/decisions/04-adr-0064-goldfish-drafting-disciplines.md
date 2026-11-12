---
type: TechnicalDecision
status: accepted
accepted: 2026-11-12
supersedes: none
date: 2026-11-12
---

# ADR-0064 — Goldfish drafting disciplines: section-by-section drafting with per-section approval gates, owner-decides-architecture, blueprint as the single source of truth; the drift report is a mode of qa

Change: 2026-11-02-egm-goldfish-brainstorm-shootdown

## Context

skillgrid's `writing-blueprints` skill writes the blueprint in one shot: the agent drafts the entire plan, then runs Self-Review + Critic + User Review. The user sees the finished product only after all sections are written, so a wrong interface in Task 1 propagates through Tasks 2–15 before the user sees it. EGM's `egm-draft` writes the design document section-by-section with per-section approval gates, along with a set of drafting disciplines (explicit ingestion, spec reconciliation, owner-decides-architecture, alternatives-from-briefing, anti-sycophancy, artifact-first delivery, plain-English) that catch quietly changed decisions and ungrounded alternatives.

A second gap: after implementation, nothing detects that the codebase has diverged from its design documents. Quick point-fixes, untracked refactors, and agent edits leave `briefing.md`, `blueprint.md`, and ADRs outdated; a later session that consults an outdated spec makes flawed assumptions. EGM's `egm-drift-report` audits design docs against source code in four divergence categories. The question was where to put the ported drift report.

## Decision

Blueprint drafting is section-by-section with per-section approval gates (subflow of writing-blueprints); architectural decisions are presented to the owner without ranking or default; the blueprint is the single source of truth (no `design_draft.md` scratch file); the drift report is a mode of `qa` (periodic, advisory), not a new top-level skill (A1).

- **Ingestion step** (R18): before any section is written, `writing-blueprints` reads `briefing.md` (in full), `findings.md` and `adr.md` if present, presents a 2–3 sentence summary of the briefing's key decisions, and waits for user confirmation.
- **Section-by-section drafting** (R19): the section order is Header → Hypothesis → Must-Haves → Threat Matrix (if applicable) → File Structure → Global Constraints → Tasks 1..N → Self-Review → Commit. Each section is written to `blueprint.md`, then the agent presents a chat summary (section, 2–3 key decisions, path) and a single-select approval gate (Approve / Request revisions) before the next section. The user can revisit any earlier section at any point.
- **Per-task gate** (R20): each task in the Tasks section gets its own approval gate — the highest-value addition: it catches a wrong interface, a missing file, or a bad test strategy before 15 more tasks are written. The task template (Files / Interfaces / SATISFIES / Steps with code) is unchanged; only the approval cadence changes.
- **Spec reconciliation** (R21): Self-Review gains a check that re-reads `briefing.md` and reports contradictions (briefing decided X, blueprint decides Y — quoted from both) and significant additions (blueprint decides something the briefing did not raise that changes architecture, interface, or dependency). The agent does not silently change the blueprint to match the briefing — drafting reads code the brainstorm did not. "No Goldfish can do this check, because no Goldfish reads the brief."
- **Owner-decides-architecture** (R22): the Owed-Decision Gate presents each load-bearing option (data model, component boundaries, storage, protocol) with its mechanism, trade-offs, and the conditions under which it fails — no ranking, no default marker, no agent pick. If the owner says "just pick one," the agent restates the trade-offs and re-presents the same gate. Mechanical choices (naming, formatting) are decided by the agent.
- **Alternatives from the briefing** (R23): alternatives come from the briefing's `Approaches Considered` section, not from memory; rejected alternatives are noted one line each; an absent section is noted as "No alternatives recorded in briefing."
- **Tone & discipline** (R24–R25): the drafting conversation cites `_shared/craft/review-discipline.md` (stress-test by default, critique first, no glazing, no banned adjectives, no apologies, plain-English); the agent writes to `blueprint.md` and gives the path — it never dumps the blueprint in chat.
- **Single source of truth**: the blueprint is written directly, section by section — no `design_draft.md` working file (a second source of truth), and no "no code" rule (the blueprint is the *how*; its task steps carry actual code by design).
- **Drift report as a mode of `qa`** (R26–R35, A1): `qa/references/drift-report.md` + `divergence-categories.md` + `reconciliation.md`; a thin Drift mode section (<40 lines) in `qa/SKILL.md`. It compares spec artifacts (`briefing.md`, `blueprint.md`, `adr.md`, `acceptance.feature` per-change, or all spec dirs + `AGENTS.md` + in-force ADRs + projectwiki pages project-wide) against the codebase in four categories (Missing Features, Incomplete Models, Logic Shortcuts, Architectural Divergence), processes spec dirs sequentially with code-search grounding, reports only actionable differences ("No drift detected" when clean), writes a committed `drift-report.md`, and reconciles two-way (code was intentional → update doc; design was violated → file task / fix code) via the Goldfish resolution workflow (R10). It is a periodic ritual, never the four-state gate, and not a phase in `sdd-structure.md`; `reflect`'s checklist gains an advisory drift check.

## Alternatives considered

- **One-shot drafting (the current model)** — replaced: the user sees the finished plan only at the end, so a wrong interface propagates through the remaining tasks uncaught.
- **Port EGM's `design_draft.md` working file** — rejected: skillgrid writes `blueprint.md` directly; a separate draft file would be a second source of truth.
- **Port EGM's "no code" rule** — rejected: it applies to the *design doc* (the *what*); skillgrid splits that into `briefing.md`, and the blueprint's task steps need actual code to be buildable.
- **The agent recommends architectural options (ranked, with a default)** — replaced by owner-decides-architecture: the owner picks from mechanism + trade-offs + failure conditions.
- **A new top-level `drift-report` skill** (the original draft) — rejected by Amendment A1: `qa` already owns code-vs-spec verification, and the drift report is a periodic ritual, not a per-change gate.
- **Make the drift report a separate spec** — rejected: it shares the resolution workflow (R10) and the persona files (R11) with the plan review; one spec keeps the dependencies explicit.

## Consequences

- Good, because a wrong interface, missing file, or bad test strategy is caught at the task gate, not after 15 more tasks are written.
- Good, because spec reconciliation catches decisions the blueprint quietly changed; the briefing's decisions are re-checked against the plan at Self-Review time.
- Good, because the owner keeps ownership of load-bearing choices — the agent presents options and failure conditions instead of recommending.
- Good, because the drift mode reuses `qa`'s existing home (code-vs-spec verification) and the Goldfish resolution workflow, so drift debt surfaces at qa via the ledger like every other parked finding.
- Bad, because drafting many interactive gates (per section + per task) makes blueprint writing a longer, more interactive session with the user.
- Bad, because the drift mode is advisory: it names divergence but does not block anything; cadence (suggested weekly gardener rotation or per-ship) is not enforced.
- Neutral, because the task template is unchanged — the change is the approval cadence and the drafting conversation, not the blueprint's structure.
- Neutral, because the drift report is a mode, not a phase: `sdd-structure.md` gains no phase or resume marker, and the four-state gate semantics are untouched.
