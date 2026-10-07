# Mnemonic Compaction v2 — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-07)

**Topic:** 2026-10-07-mnemonic-compaction-v2
**Date:** 2026-10-07
**Classification:** medium (new table column + new package + new HTTP routes) — T2
**Build shape:** Journey (each piece is a complete capability path)
**Reference logic:** CLM paper (arXiv:2609.37725), Claude cookbook (session-memory-compaction), compact-adviser (kunchenguid)

## Problem / Intent

Mnemonic's current compaction is thin and reactive: a single `compact` hook fires on `session.idle` and saves one continuity observation (title + ≤5 recent obs titles, 3s budget, fail-open); `CompactionContext` injects a flat title list into the runtime's compaction prompt. It lacks (a) a *timing* decision (when is it safe to compact?), (b) a *structured* prompt that preserves what matters, (c) a *self-managed* editable context file, and (d) a *proactive* background build so compaction is instant. This change adopts three orthogonal source logics — advisory timing (compact-adviser), structured six-section prompt + proactive build (Claude cookbook), and CLM steering (the paper) — to make compaction smart, lossless, and instant.

## Purpose & Success Criteria

- **Purpose:** Enhance mnemonic compaction with four additive, plugin-aligned capabilities.
- **Success criteria (verifiable):**
    1. `GET /compaction/advice` returns `{score, hint, floor, reason}` from a single combined LLM call (JSON `{finished, hands_on}`), fail-open on LLM error.
    2. `CompactionContext` returns six structured sections (User Intent / Actions Succeeded / Errors & Corrections / Active Work / Pending Tasks / Critical Details), recent-weighted.
    3. `hookCompact` appends a `context_revisions` row (rev+1, not blind upsert) with a `steering` column; the prior revision's `steering` is re-injected into the next compaction prompt.
    4. A proactive background build pre-summarizes the structured revision on a 15m interval, skipped when `contextFraction < 0.3`.
    5. `mem_compact_advice` MCP tool returns `{score, hint, floor, reason}`.
    6. `go test ./mnemonic/internal/advice/... ./mnemonic/internal/http/...` passes.
    7. `pnpm test` / `pnpm lint` / `pnpm typecheck` pass.
- **Out of scope:** Replacing the raw-session-history source of truth (the CLM mirror, per ADR-0027/0028, already owns that); TUI panel; model-driven `/ctx-compact`.

## Context

- **Existing flows:** `hookCompact` (`memory/skills.go:567`) saves a `session_summary` observation at `topic_key compaction/<session>`; `CompactionContext` (`memory/service.go:2135`) returns title+summary+≤5 one-line obs; `mnemonic_commit` (`service/compaction.go`) writes L2 + fact extraction + auto-skill.
- **Locked artifacts this extends (not duplicates):**
    - `04-adr-0027-clm-context-language-model.md` — Go owns state/decisions, Node owns the request path; `clm:` block in `config.yaml`; mirror in temp dir; overflow guard + calibration.
    - `04-adr-0028-context-revisions-session-scoped.md` — `context_revisions` table (migration `052_context_revisions.sql`), session-scoped, purged at session end, columns: `id, session_id, project_id, revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, withhold_ids, edit_trace, created_at`.
- **Constraint:** the replace-hooks-with-plugins plan (`.skillgrid/specs/2026-10-07-replace-opencode-hooks-with-plugins/blueprint.md`) is assumed to land first — all plugin wiring targets the 5 new TS plugins, NOT the retired shell hooks.
- **Files:** `mnemonic/internal/{advice(new),config,service,memory,checkpoint,store,http}`, `plugins/opencode/{skillgrid-compaction,skillgrid-events,mnemonic-memory}.ts`.

## Approaches Considered

- **Chosen:** Additive capabilities extending ADR-0027/0028. The advisory gate is a new `internal/advice` package; the structured prompt extends `CompactionContext`; steering is a new column on the existing `context_revisions`; the proactive build reuses the async-tiering goroutine pattern.
- **Rejected:** Two separate LLM calls per advice (2× latency/cost) — the user chose one combined call.
- **Rejected:** Dual-path wiring (new TS plugins + retired shell hooks) — the replace plan lands first, so single-path to the 5 TS plugins.
- **Rejected:** A second context-revisions table — ADR-0028's table already exists; add a `steering` column via a follow-on migration (053) instead.

## Requirements

1. **Advisory Timing Gate** — `internal/advice` (NEW).
    - **Current:** No timing decision; the `compact` hook fires unconditionally on `session.idle`.
    - **Target:** `Adviser.Advise(ctx, in) (Advice, error)`; `in = {sessionID, recentDigest, contextChars, budgetChars}`. `contextFraction = min(1, contextChars/budgetChars)`. One LLM call → JSON `{finished:bool, hands_on:bool}`. `score = 0.5*finished + 0.5*(1-handsOn)`; `floor = 0.90 - 0.40*contextFraction`; `hint = score >= floor`. Fail-open on LLM error → `{Hint:false, Reason:"llm unavailable"}`.
    - **Acceptance:** `GET /compaction/advice` returns `{score, hint, floor, reason}`; LLM-down returns `hint:false, reason:"llm unavailable"`.
    - **Acceptance scenario:** `happy path advisory hint curve` → `acceptance.feature`.

2. **Structured Six-Section Prompt** — `CompactionContext` + `checkpoint/prompt.go`.
    - **Current:** Flat list of ≤5 one-line observation titles.
    - **Target:** `CompactionContext.Sections map[string]string` (recent-weighted): `User Intent` / `Actions Succeeded` / `Errors & Corrections` / `Active Work` / `Pending Tasks` / `Critical Details`. `checkpoint/prompt.go` renders the six-section compression instruction. Preserve-order: `corrections > errors > active work > completed work`.
    - **Acceptance:** A session with a user correction + an error + an in-progress task produces a `CompactionContext` whose `Sections` contains all three, with the correction present verbatim.
    - **Acceptance scenario:** `happy path structured sections preserve corrections` → `acceptance.feature`.

3. **CLM Steering** — `context_revisions.steering` column + `hookCompact`.
    - **Current:** `hookCompact` upserts a single `session_summary` observation (blind overwrite on `topic_key compaction/<session>`).
    - **Target:** `hookCompact` appends a `context_revisions` row (rev+1). New `steering` column (migration `053_context_revisions_steering.sql`). The prior revision's `steering` is re-injected into the next compaction prompt; the agent may evolve it. `CompactionContext` returns the latest revision's `steering`.
    - **Acceptance:** After two `hookCompact` runs, `context_revisions` has rev 1 and rev 2; the rev-2 prompt input contains rev-1's `steering` text.
    - **Acceptance scenario:** `happy path steering re-injection` → `acceptance.feature`.

4. **Proactive Instant Compaction** — `service/compaction.go` background build.
    - **Current:** Compaction is reactive (fires when the context is full).
    - **Target:** A background goroutine (mirroring the async-tiering pattern in `MnemonicCommit`) pre-builds the structured revision on a 15m interval, riding the existing `/sessions` end HTTP path. Guard: one build per interval; skip if `contextFraction < 0.3`.
    - **Acceptance:** With `proactive:true` and `interval:15m`, a session at `contextFraction 0.4` produces a pre-built revision within one interval without a `session.idle` event.
    - **Acceptance scenario:** `happy path proactive pre-build` → `acceptance.feature`.

5. **Config + Service Wiring** — `internal/config` + `internal/service`.
    - **Current:** No `mnemonic.compaction` section.
    - **Target:** `Compaction` struct (`AdviserEnabled, BudgetChars, Proactive, Interval, MaxRevisions`) + `DefaultCompaction()` (adviser/proactive false; budget 60000, 15m, 10) + `compactionSection` (pointer fields) + `mergeCompaction` (mirror `mergeCheckpoint`). `Service.SetCompaction(cfg)` + LLM seam wired at the config-load site (mirror `SetBudget`).
    - **Acceptance:** `mnemonic.compaction.adviser_enabled: true` in config → `SetCompaction` receives `AdviserEnabled:true`; absent section → defaults.
    - **Acceptance scenario:** `happy path compaction config merge` → `acceptance.feature`.

6. **HTTP Routes + Plugin Wiring** — `internal/http` + `plugins/opencode/*.ts`.
    - **Current:** No `/compaction/advice` route.
    - **Target:** `internal/http/compaction.go` — `registerCompactionRoutes()`: `GET /compaction/advice` (run adviser), `POST /compaction/advice` (submit structured context + steering). Wire into `registerRoutes()` (mirror `teams.go`). Plugins: `skillgrid-compaction.ts` (on `experimental.session.compacting`/`session.compacted`: GET hint + POST sections), `skillgrid-events.ts` (on `tool.execute.after`: POST context-char count), `mnemonic-memory.ts` (`mem_compact_advice` tool).
    - **Acceptance:** `GET /compaction/advice?session_id=...` returns a JSON advice; `POST /compaction/advice` with a structured body persists a revision and returns the new rev.
    - **Acceptance scenario:** `happy path compaction advice routes` → `acceptance.feature`.

## Implementation Decisions

- **Modules to build/modify:**
    - `mnemonic/internal/advice/advice.go` (NEW): `Adviser`, `Advice`, `Advise`, floor curve.
    - `mnemonic/internal/config/load.go`: `Compaction` struct + default + `compactionSection` + `mergeCompaction`; wire into `Load`.
    - `mnemonic/internal/service/service.go`: `compactionCfg` + `SetCompaction` + LLM seam.
    - `mnemonic/internal/service/compaction.go`: proactive build goroutine + revision write.
    - `mnemonic/internal/memory/skills.go`: `hookCompact` → append `context_revisions` row + steering.
    - `mnemonic/internal/memory/service.go`: `CompactionContext.Sections` + `steering` read.
    - `mnemonic/internal/checkpoint/prompt.go`: six-section structured prompt.
    - `mnemonic/internal/store/migrations/053_context_revisions_steering.sql`: add `steering` column.
    - `mnemonic/internal/http/compaction.go` (NEW) + `server.go`: routes.
    - `plugins/opencode/{skillgrid-compaction,skillgrid-events,mnemonic-memory}.ts`: wiring.
- **Interfaces:**
    - `func (a *Adviser) Advise(ctx context.Context, in AdviceInput) (Advice, error)`
    - `type Advice struct { Score, Floor float64; Hint bool; Reason string }`
    - `func (s *Service) SetCompaction(cfg config.Compaction)`
    - `CompactionContext` gains `Sections map[string]string` and `Steering string`.
- **Data flow:**
    - `tool.execute.after` (plugin) → POST context-chars → `GET /compaction/advice` → adviser LLM call → `{score, hint, floor, reason}` → hint line injected.
    - `experimental.session.compacting` (plugin) → `CompactionContext` (structured) → `POST /compaction/advice` → `hookCompact` appends `context_revisions` (rev+1, steering).
    - Proactive: timer → structured revision → `context_revisions` (no LLM needed if structured build is deterministic; LLM only for the advisory gate).
- **Error handling:**
    - Adviser fails open (LLM error → `hint:false`, never blocks compaction).
    - Proactive build is async (goroutine, warn on stderr, never fails the request).
    - `POST /compaction/advice` on invalid body → 400; on store error → 500.

## Testing Decisions

- **What makes a good test:** Only external behavior — the advice JSON, the `CompactionContext.Sections`, the `context_revisions` rev sequence, the proactive build timing.
- **Modules to test:** `advice`, `http`, `memory` (hookCompact), `config`, `service`.
- **Prior art:** `compact_hook_test.go`, `facts_test.go`, `mergeCheckpoint` tests.
- **Edge cases:** LLM down, empty session, `contextFraction < 0.3` skip, two compactions in a row (rev+1), steering re-injection, config absent → defaults.

## Impact on Global Docs

- `.skillgrid/artifacts/04-adr-index.md`: add ADR-0029.
- `.skillgrid/ARCHITECTURE.md`: note the `steering` column on `context_revisions` and the new `/compaction/advice` route.
- `.skillgrid/artifacts/07-mnemonic-tool-surface.md`: add `mem_compact_advice`.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.95  | 0.75 | OK     | Four distinct capabilities.        |
| Boundary Clarity    | 0.85  | 0.70 | OK     | Extends ADR-0027/0028, no overlap. |
| Constraint Clarity  | 0.85  | 0.65 | OK     | One combined LLM call, replace-first. |
| Acceptance Criteria | 0.90  | 0.70 | OK     | Specific behaviors per feature.    |
| **Clarity**         | 0.05  | ≤0.20| OK     |                                    |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | Scope + LLM strategy     | All four pieces, one combined LLM call. |
| 2     | Wiring target            | Assume replace-hooks plan lands first; wire to 5 TS plugins. |
| 3     | Proactive interval       | 15m default.                       |

## Open Questions & Assumptions

- **Assumption:** The one combined LLM call's JSON `{finished, hands_on}` parses cleanly; if parsing proves fragile, split to two calls (noted in the blueprint).
- **Assumption:** The `steering` column is a non-load-bearing enhancement to the existing `context_revisions` (ADR-0028) — migration 053 is additive.

## Decisions (ADR)

- `04-adr-0029-compaction-advisory-steering.md` (Planned): Document the advisory timing gate + steering column + proactive build.

## Terms

- **Advisory Gate:** The compact-adviser logic — a 0–1 score that decides whether it is safe to compact now.
- **Steering:** A natural-language instruction the model can evolve, re-injected into each compaction prompt (the CLM in-context-learning knob).
- **Proactive Build:** A background pre-summarization so the real compaction event is instant.
- **Second Brain:** See `.skillgrid/artifacts/01-business-terms.md`.
