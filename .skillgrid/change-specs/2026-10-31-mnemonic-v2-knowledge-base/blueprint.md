# ProjectWiki Implementation Blueprint (steps, files, verification)

> Copy this template to `.skillgrid/change-specs/YYYY-MM-DD-<feature-name>/blueprint.md`.
> **Role:** this is the *how* — the implementation plan: concrete steps, files, code, tests, and verification. The companion `briefing.md` is the *what* (requirements & intent). The filename pair is load-bearing: `brainstorming` and `resume` key phase state off which of the two files exist. Do not rename.
> Assume the engineer has zero context for this codebase. Document every file to touch, the code, the tests, and how to verify. DRY. YAGNI. TDD. Frequent commits. No placeholders.

**Status:** PROPOSED — amended at QA (2026-11-11, W005): the Key Link below is corrected to the shipped spec-008 design (agent-proactive `projectwiki_generate`, not an eager `SaveEvent` subscription); truths T1–T8 were verified against the shipped engine at the QA gate (see `report.md`). Ratification of this blueprint as a whole remains the human's call at the ship gate.

**Tier:** T2

**Build shape:** Tracer thread

**Claim:** The mnemonic-v2 project wiki can ship as a single end-to-end function — file-truth vault in `\.skillgrid/projectwiki/` + derived FTS cache in the shared project DB + `projectwiki_*` engine tools + `projectwiki` management skill — without a separate ProjectWiki database and without new dependencies.

**Right condition:** `projectwiki_bootstrap` scaffolds the vault; `projectwiki_generate` proves a clean run is `changed: 0`; `projectwiki_query` returns cited sections wiki-first; `projectwiki_validate`/`projectwiki_lint` catch drift; `projectwiki_eval` catches retrieval regressions; the `projectwiki` skill is present and cross-references real tool names.

**Wrong condition:** The vault and the index disagree silently, the no-op proof is bypassed, or the skill names tools that do not exist.

**Thinnest MVP:** `projectwiki_bootstrap` + `projectwiki_index`/`projectwiki_reindex` + `projectwiki_query` over an already-populated vault, with the no-op proof and degraded reporting.

**Door check:** Task 1 (bootstrap) / Task 2 (index) — if the shared-store extension or migration runner is missing, stop and resolve before building on top.

**Goal:** Make the markdown vault at `\.skillgrid/projectwiki/` the single synthesized knowledge surface of mnemonic-v2, with the FTS index as a derived cache in the shared project store, guarded by `projectwiki_*` engine tools and the `projectwiki` management skill.

**Architecture:** The ProjectWiki keeps markdown files as source of truth (PW-INV-1) and keeps all index state (`projectwiki_pages`, `projectwiki_pages_fts`, `projectwiki_claims`, `projectwiki_run_state`, `projectwiki_page_baselines`, `projectwiki_events`) in the shared project DB — no separate `projectwiki.sqlite` (KB-O2). The write path is eager-per-save: memory `SaveEvent` → `projectwiki_generate` (graduate → target → synthesize → reindex → no-op proof). The read path is wiki-first: `projectwiki_query` returns cited markdown sections, falling back to raw memory when the vault is thin (OPEN-12). The derived FTS uses external-content FTS over page bodies (content-tokenized, not duplicated), with a crash-recovery pending ledger and the trigger-vs-explicit soft-delete discipline. The skill layer is the behavioral contract over the mechanical `projectwiki_*` tools.

**Tech Stack:** Go 1.25.5 module `github.com/devopstales/skillgrid/mnemonic` (standalone module; CLI at `mnemonic/cmd/mnemonic/`); SQLite via `modernc.org/sqlite` + optional PostgreSQL via `pgx` behind the existing dual-mode `Store` abstraction (ADR-0024); Ollama openai-compat embedder (`embeddinggemma:300m`) for the semantic leg (ADR-0035/0037); tree-sitter (ADR-0055) and fsnotify (ADR-0056) for the code-index graph; no new dependencies without ADR-0008.

**Spec:** `.skillgrid/change-specs/2026-10-31-mnemonic-v2-knowledge-base/specs/000-overview.md` (companions: `projectwiki-indexing-plan.md`, `projectwiki-skill-plan.md`).

## Terms

- None beyond the spec's own `KB-*` / `INV-*` / `KQ-*` / `DE-*` vocabulary; no new term minted here.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold when the plan is complete):

- [ ] T1: A project without `\.skillgrid/projectwiki/config.json` has no ProjectWiki; `projectwiki_*` tools report that cleanly (not error).
- [ ] T2: A clean `projectwiki_generate` run reports `changed: 0` and writes nothing (PW-INV-4, proven via the 4-layer no-op proof).
- [ ] T3: `projectwiki_index` (incremental) re-tokenizes only changed pages; `projectwiki_reindex` (full) rebuilds from scratch; both idempotent.
- [ ] T4: `projectwiki_query` returns cited markdown sections, wiki-first, with memory fallback when thin; sets `degraded: stale_index` when stale.
- [ ] T5: `projectwiki_validate` fails a turn (or flags, `--strict`) on stale/unresolved pins/claims.
- [ ] T6: `projectwiki_lint` reports orphans / broken wikilinks / dup aliases / coverage gaps / external-staleness; `--fix` repairs the safe subset.
- [ ] T7: `projectwiki_status` reports page counts by type, stale count, last run, pending queue, orphan count.
- [ ] T8: `projectwiki_cite_source` records an external citation with `build_fingerprint`; re-fetch is section-granular.

**Artifacts** (files that must exist with real implementation, not stubs):

- [ ] A. `projectwiki_*` engine over the shared store: `projectwiki_pages` / `projectwiki_pages_fts` / `projectwiki_claims` / `projectwiki_run_state` / `projectwiki_page_baselines` / `projectwiki_events` tables + additive migration (user_version / BEGIN IMMEDIATE / legacy-reconcile / trigger-aware split); `projectwiki_bootstrap`, `projectwiki_status`, `projectwiki_generate`, `projectwiki_index`, `projectwiki_reindex`, `projectwiki_query`, `projectwiki_validate`, `projectwiki_lint`, `projectwiki_cite_source`, `projectwiki_eval` handlers.
- [ ] B. `projectwiki` management skill files: `SKILL.md`, `CONVENTIONS.md`, `PAGES.md`, `CLAIMS.md`, `MERMAID.md`.
- [ ] C. Acceptance scenarios in `acceptance.feature`.

**Key links** (critical connections between artifacts that must work together):

- [x] The agent-proactive save loop (spec 008 §3, as shipped): significant `mem_save` → agent calls `projectwiki_generate` (write queue, ownership-asserted) → `projectwiki_index` (final step) → no-op proof (PW-INV-4). *(Amended 2026-11-11: the originally sketched eager `SaveEvent` subscription was not built; the proactive protocol is the recorded design in 008-transport + the AGENTS.md vault block + the management skill.)*
- [ ] `projectwiki_pages` (source of truth) ↔ `projectwiki_pages_fts`/`projectwiki_claims` (derived) in the shared DB; `index/last-update.json` is the file projection of `projectwiki_run_state`.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):

- [ ] O1: Adding `projectwiki_*` tables to the shared project DB (additive migration) — rides the memory change's migration runner; requires the four-property migration discipline. **Needs user approval before the migration lands.**
- [ ] O2: Semantic-leg injection point / hybrid weight blend ownership (single blend place) — design decision already fixed in `005-query` §2.1.
- [ ] O3: `projectwiki_eval` gating policy (advisory vs gating) — default: gating for changes touching ranking / `thin` / semantic fallback.

## Global Constraints

- Go 1.22+ minimum to build (ADR-0031); no new dependencies without an ADR (ADR-0008).
- Spec zone (`.skillgrid/change-specs/**`) commits before code zone; one commit per zone.
- Conventional commits only; no AI-attribution trailers.
- TDD always on (failing test before code); BDD always on (every requirement has Given/When/Then).
- Agent authors `title`/`type`/`project`/`aliases`/`anchors`/`claims`/`sources`; engine owns `status`/`stale_after`/`generated`/`verified`.

## File Structure

- `mnemonic/internal/...` — store (projectwiki tables + migration), generate, index, query, validate, lint, status, cite, eval, bootstrap, transport.
- `mnemonic-v2/skills/projectwiki/` — `SKILL.md`, `CONVENTIONS.md`, `PAGES.md`, `CLAIMS.md`, `MERMAID.md`.
- `\.skillgrid/projectwiki/` — vault scaffold (`config.json`, `PROJECTWIKI_SCHEMA.md`, `templates/`, page folders, `.ignore`).

## Tasks

See `tasks.md` in this change folder. Execution waves: engine core → generate → index/query → validate/lint/status → skill + eval.

---

## Task 1: `projectwiki_bootstrap` — vault scaffold

**Files:**
- Create: `mnemonic/internal/projectwiki/bootstrap.go`
- Test: `mnemonic/internal/projectwiki/bootstrap_test.go`

**Interfaces:**
- Consumes: none (reads/writes `\.skillgrid/projectwiki/`).
- Produces: `Bootstrap(ctx, dir) error` — idempotent scaffold.
- Seam: filesystem boundary.
- Deletion test: pass-through.

**SATISFIES:** ProjectWiki vault bootstrap scenario.

- [ ] Step 1: Write the failing test
- [ ] Step 2: Run test to verify it fails
- [ ] Step 3: Write minimal implementation
- [ ] Step 4: Run test to verify it passes
- [ ] Step 5: Commit

## Task 2: `projectwiki_index` / `projectwiki_reindex` — derived FTS rebuild

**Files:**
- Create: `mnemonic/internal/projectwiki/index.go`
- Test: `mnemonic/internal/projectwiki/index_test.go`

**Interfaces:**
- Consumes: vault markdown (source of truth).
- Produces: `Index(ctx, dir) error`, `Reindex(ctx, dir) error`.
- Seam: store FTS boundary.

**SATISFIES:** ProjectWiki index scenarios.

- [ ] Step 1: Write the failing test
- [ ] Step 2: Run test to verify it fails
- [ ] Step 3: Write minimal implementation
- [ ] Step 4: Run test to verify it passes
- [ ] Step 5: Commit
