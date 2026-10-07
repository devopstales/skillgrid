# Compaction advisory gate + steering column + proactive build

---
status: "accepted"
supersedes: none
date: 2026-10-07
---

## Context and Problem Statement

Mnemonic's compaction is thin and reactive: a single `compact` hook fires on `session.idle` and saves one continuity observation (title + ≤5 recent observation titles, 3s budget, fail-open); `CompactionContext` injects a flat title list into the runtime's compaction prompt. Three capabilities are missing, and each is solved by a distinct source logic:

- **Timing** (when is it safe to compact?) — solved by compact-adviser: a two-classification score (task-finished + hands-off) compared against a floor that tightens as context fills (0.90 empty → 0.50 full).
- **Prompt quality** (what must survive) — solved by the Claude cookbook's six-section structured summary (User Intent / Actions Succeeded / Errors & Corrections / Active Work / Pending Tasks / Critical Details), recent-weighted, corrections captured verbatim.
- **Self-management** (who edits the context) — solved by the CLM paper (ADR-0027): the context is a file the model edits, and the in-context-learning knob is a steering instruction the model can evolve.
- **Latency** (instant, not when-full) — solved by the Claude cookbook's proactive background build, pre-summarizing before the context is full.

The integration question is whether these four compose without colliding with the already-accepted ADR-0027 (CLM context-file) and ADR-0028 (`context_revisions`, migration 052, session-scoped).

## Considered Options

- **Duplicate the CLM layer:** a second context-revisions table + a new steering mechanism.
- **Extend the existing CLM layer:** add a `steering` column to the existing `context_revisions` table (migration 053) and feed the structured prompt into the CLM mirror.
- **No advisory gate, reactive only:** keep the current `compact` hook, only add the structured prompt.

Chosen option: "Extend the existing CLM layer," because ADR-0027/0028 already own the context-file and the revision table. The advisory gate is a new, orthogonal `internal/advice` package (it decides *when*, not *what*). The structured prompt feeds the existing `CompactionContext` → `context_revisions.messages` path. Steering is a single additive column — a natural-language instruction the model can evolve, re-injected into each compaction prompt. The proactive build reuses the existing async-tiering goroutine pattern. This keeps the change additive (no schema rewrite, no new LLM provider, no new dependency).

- The advisory gate makes **one combined LLM call** (JSON `{finished:bool, hands_on:bool}`), not two — the user chose this over the 2× latency/cost of separate calls. The `contextFraction = min(1, contextChars/budgetChars)` drives the floor: `floor = 0.90 − 0.40·contextFraction`; `score = 0.5·finished + 0.5·(1−handsOn)`; `hint = score ≥ floor`. Fail-open on LLM error → `hint:false, reason:"llm unavailable"` (never blocks compaction).
- Config lives in a new `mnemonic.compaction` section: `adviser_enabled` (default false), `budget_chars` (default 60000), `proactive` (default false), `interval` (default 15m), `max_revisions` (default 10). All four new capabilities are **opt-in** — no behavior change on upgrade.
- Plugin wiring targets the 5 new TS plugins (`skillgrid-compaction.ts`, `skillgrid-events.ts`, `mnemonic-memory.ts`) per the replace-hooks plan, NOT the retired shell hooks. Single-path; no dual-path.
- v1 scope: advisory gate + structured prompt + steering column + proactive build. No TUI panel, no model-driven `/ctx-compact`, no per-project budget file beyond the `mnemonic.compaction` block.

### Consequences

- Good, because the four capabilities are orthogonal and compose: the adviser decides *when*, the structured prompt decides *what survives*, steering lets the model *manage its own context*, and the proactive build makes it *instant*.
- Good, because the change is additive and opt-in: no schema rewrite (one new column), no new LLM provider (reuses `internal/llm`), no new dependency, and all four capabilities are off by default.
- Good, because it extends rather than duplicates ADR-0027/0028: the existing `context_revisions` table gains a `steering` column; the structured prompt feeds the existing `CompactionContext` → `messages` path; the CLM mirror (ADR-0027) is untouched.
- Good, because the adviser is fail-open: an LLM error returns `hint:false` and never blocks a compaction — the same philosophy as the existing `compact` hook.
- Bad, because the one combined LLM call couples two classifications in a single JSON response — if the model's JSON is malformed, both `finished` and `hands_on` are lost (falls back to `hint:false`). A two-call design would be more robust but 2× latency/cost; the user accepted the trade-off.
- Bad, because the proactive build adds a background goroutine per active session — if the interval is set too low and `contextFraction` hovers near 0.3, the build can churn. The `0.3` threshold + one-build-per-interval guard mitigate this; a very low interval is a config footgun.
- Bad, because the `steering` column is a prompt-injection surface (like the CLM mirror): a malicious steering can induce the model to rewrite its own constraints in the next compaction. This is the same known limitation ADR-0027 documents for the mirror, now one column deeper.
