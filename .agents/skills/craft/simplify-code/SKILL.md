---
name: simplify-code
description: "Cleanup pass over recent changes with four parallel reviewers (reuse, quality, efficiency, altitude): aggregate findings, apply by risk tier. Use when asked to simplify, clean up, or review recent changes."
license: MIT
metadata:
  author: Hermes Agent, inspired by Claude Code /simplify (ported; see ADR-0015)
  version: "1.0"
  part-of: skillgrid
---

# Simplify Code — Parallel Review & Cleanup

**Announce at start:** "I'm using the skillgrid:simplify-code skill to clean up these changes."

## Overview

Review recent changes with four focused reviewers running in parallel, aggregate findings, apply what's worth applying. **Cleanup pass, not a bug hunt** — the code already works; remove duplication, flatten needless complexity, cut waste, deepen band-aids. Correctness bugs belong to `skillgrid:requesting-code-review`.

**Core principle:** four narrow reviewers beat one broad reviewer. Each searches the codebase for one problem class — reuse, quality, efficiency, altitude — concurrently, so you pay one review's latency, not four.

Relation to neighbors: `skillgrid:dry-refactoring` is clone-driven (starts from jscpd output); this skill is diff-driven (starts from your recent changes). `skillgrid:ponytail` picks the laziest solution before/instead of writing; this cleans what got written. `skillgrid:parallel-code-review` renders a verdict on a diff; this one applies fixes.

Ported from the Hermes `simplify-code` skill under MIT (see ADR-0015). Behavioral delta from upstream: fan-out via skillgrid subagents (`skillgrid:parallel-execution`, depth ≤ 1) instead of `delegate_task`; Hermes `delegation.max_concurrent_children` dropped.

## When to Use

User asks to "simplify", "clean up", "review my recent changes". Honor modifiers: **focus** (`reuse` / `quality` / `efficiency` / `altitude` — run only that reviewer or weight toward it), **dry run** ("just report" → present, apply nothing), **scope** (last commit / staged / branch / paths — see Phase 1).

**When NOT to use:** After every edit or tacked onto unrelated tasks — four reviewers cost tokens; explicit ask only. For correctness review — `skillgrid:requesting-code-review`. For clone elimination across the repo (not your diff) — `skillgrid:dry-refactoring`. For choosing what to build — `skillgrid:ponytail`.

## The Process

### Phase 1 — Capture the diff

Default order: `git diff` (tracked working tree) → `git diff HEAD` (incl. staged) → scoped variants (`--staged`, `HEAD~1`, `main...HEAD`, `-- <paths>`). Empty everywhere and no named files → stop, say so. Warn past ~2000 changed lines: four reviewers each carrying it is token-heavy; scope down first.

### Phase 2 — Fan out four reviewers

Dispatch via `skillgrid:parallel-execution` (one level only — reviewers don't delegate further). No subagents available (leaf context, disabled)? Work all four angles inline, sequentially, and state that in the summary. Every reviewer gets the **complete diff** plus the repo path and search rights. Each must: search the codebase for evidence (never reason from the diff alone); apply Chesterton's Fence (`git blame` before flagging removal; unknown purpose → `confidence: low`); report `file:line → problem → cost → fix | confidence: high/medium/low | risk: SAFE/CAREFUL/RISKY`; skip nits.

- **Reuse:** duplicates of existing utils/helpers/patterns — name the existing thing + where it lives.
- **Quality:** redundant state, parameter sprawl, copy-paste-with-variation, leaky abstractions, stringly-typed code (check canonical registries first), deep nesting (flatten via guards/early returns/lookup), AI slop (comment restating code, defensive nulls on validated inputs, `as any`, file-inconsistent patterns).
- **Efficiency:** redundant work (rereads, duplicate calls, N+1), missed concurrency, hot-path bloat, TOCTOU pre-checks, memory issues (unbounded growth, leaks, closure capture — prefer small explicit-fields structs), overly broad reads, silent failures (empty catches, swallowed errors — at minimum must log).
- **Altitude:** too-shallow fixes — special cases in generic paths, symptom patches at one call site, stacked workarounds, wrappers avoiding the real fix, flags routing around broken defaults. Name the dodged mechanism + the deeper fix; flag when it's follow-up-sized. Check `git blame` first: compat shims, staged migrations, vendored isolation are deliberate boundaries, not band-aids.

Risk tiers: **SAFE** (no behavior effect — auto-apply) · **CAREFUL** (semantics-preserving — apply with test verification) · **RISKY** (may change behavior/contracts — human review only).

### Phase 3 — Aggregate and apply

Merge + dedupe (same line/mechanism collapses), drop false positives silently, resolve conflicts by **correctness > stated focus > readability/reuse > micro-perf** (touch less code on ties). Apply SAFE → CAREFUL (one file at a time, tests after each, revert breakage) → present RISKY without applying. Dry run: present all three, apply nothing. Verify with the repo's targeted tests + lint/typecheck for touched files only. Summarize applied fixes by reviewer × tier plus deliberately skipped findings and why (and inline-vs-fanout, if relevant).

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "One reviewer with all four angles is cheaper" | Breadth dilutes attention — that's the failure mode. Four narrow passes, one latency. |
| "Split the diff across reviewers to save tokens" | Cross-file duplication and N+1s hide in the gaps. Whole diff to everyone, or scope down first. |
| "A finding without a file pointer is still useful" | "Probably a helper for this" is noise. Require `file:line` evidence or drop it. |
| "While I'm here, refactor the module" | Cleanup is scoped to the diff + minimal surroundings. Deeper fixes get flagged, not unilaterally rebuilt. |
| "This test failure is unrelated, ship anyway" | Revert that one fix and report it. Never bank a red test. |
| "The reviewer found a real bug — fold it into cleanup" | Report correctness bugs as separate "found a bug" notes with different verification standards. |

## Red Flags

- Fan-out wider than 4 (cost + conflicts, not coverage)
- Public-contract renames (exports, routes, columns, config keys) applied instead of tagged RISKY
- Removing "unnecessary" error handling that is intentional (expected-benign errors) — flag, let the human decide
- Reviewer suggestions fighting AGENTS.md/linter house style (fold repo conventions into prompts first)
- Dead-code tool output (`knip`, `depcheck`, Go `unused`) treated as proof — grep the symbol first (dynamic use hides from tools)
- Drifting into bug-hunting instead of cleanup

## Verification

- [ ] Explicit user ask (not auto-run); focus/dry-run/scope modifiers honored
- [ ] Whole diff to every reviewer (or scoped-down with user agreement)
- [ ] Findings carry cost + confidence + risk tier; overlapping ones merged
- [ ] SAFE auto-applied, CAREFUL test-verified per file, RISKY presented unapplied (dry run: nothing applied)
- [ ] Targeted tests + lint/typecheck green on touched files; breakage reverted
- [ ] Summary lists applied fixes by reviewer × tier + skipped findings + why
