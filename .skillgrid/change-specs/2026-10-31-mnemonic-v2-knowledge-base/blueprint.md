# Knowledge-Base Implementation Blueprint (steps, files, verification)

> Copy this template to `.skillgrid/change-specs/YYYY-MM-DD-<feature-name>/blueprint.md`.
> **Role:** this is the *how* — the implementation plan: concrete steps, files, code, tests, and verification. The companion `briefing.md` is the *what* (requirements & intent). The filename pair is load-bearing: `brainstorming` and `resume` key phase state off which of the two files exist. Do not rename.
> Assume the engineer has zero context for this codebase. Document every file to touch, the code, the tests, and how to verify. DRY. YAGNI. TDD. Frequent commits. No placeholders.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread

**Claim:** The mnemonic-v2 knowledge base can ship as a single end-to-end function — file-truth vault in `.skillgrid/knowledge/` + derived FTS cache in the shared project DB + `kb_*` engine tools + `kb` management skill — without a separate KB database and without new dependencies.

**Right condition:** `kb_bootstrap` scaffolds the vault; `kb_generate` proves a clean run is `changed: 0`; `kb_query` returns cited sections wiki-first; `kb_validate`/`kb_lint` catch drift; `kb_eval` catches retrieval regressions; the `kb` skill is present and cross-references real tool names.

**Wrong condition:** The vault and the index disagree silently, the no-op proof is bypassed, or the skill names tools that do not exist.

**Thinnest MVP:** `kb_bootstrap` + `kb_index`/`kb_reindex` + `kb_query` over an already-populated vault, with the no-op proof and degraded reporting.

**Door check:** Task 1 (bootstrap) / Task 2 (index) — if the shared-store extension or migration runner is missing, stop and resolve before building on top.

**Goal:** Make the markdown vault at `.skillgrid/knowledge/` the single synthesized knowledge surface of mnemonic-v2, with the FTS index as a derived cache in the shared project store, guarded by `kb_*` engine tools and the `kb` management skill.

**Architecture:** The KB keeps markdown files as source of truth (KB-INV-1) and keeps all index state (`kb_pages`, `kb_pages_fts`, `kb_claims`, `kb_run_state`, `kb_page_baselines`, `kb_events`) in the shared project DB — no separate `kb.sqlite` (KB-O2). The write path is eager-per-save: memory `SaveEvent` → `kb_generate` (graduate → target → synthesize → reindex → no-op proof). The read path is wiki-first: `kb_query` returns cited markdown sections, falling back to raw memory when the vault is thin (OPEN-12). The derived FTS uses external-content FTS over page bodies (content-tokenized, not duplicated), with a crash-recovery pending ledger and the trigger-vs-explicit soft-delete discipline. The skill layer is the behavioral contract over the mechanical `kb_*` tools.

**Tech Stack:** Go 1.25.5 module `github.com/devopstales/skillgrid/mnemonic` (standalone module; CLI at `mnemonic/cmd/mnemonic/`); SQLite via `modernc.org/sqlite` + optional PostgreSQL via `pgx` behind the existing dual-mode `Store` abstraction (ADR-0024); Ollama openai-compat embedder (`embeddinggemma:300m`) for the semantic leg (ADR-0035/0037); tree-sitter (ADR-0055) and fsnotify (ADR-0056) for the code-index graph; no new dependencies without ADR-0008.

**Spec:** `.skillgrid/change-specs/2026-10-31-mnemonic-v2-knowledge-base/specs/000-overview.md` (companions: `kb-indexing-plan.md`, `kb-skill-plan.md`).

## Terms

- None beyond the spec's own `KB-*` / `INV-*` / `KQ-*` / `DE-*` vocabulary; no new term minted here.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold when the plan is complete):

- [ ] T1: A project without `.skillgrid/knowledge/config.json` has no KB; `kb_*` tools report that cleanly (not error).
- [ ] T2: A clean `kb_generate` run reports `changed: 0` and writes nothing (KB-INV-4, proven via the 4-layer no-op proof).
- [ ] T3: `kb_index` (incremental) re-tokenizes only changed pages; `kb_reindex` (full) rebuilds from scratch; both idempotent.
- [ ] T4: `kb_query` returns cited markdown sections, wiki-first, with memory fallback when thin; sets `degraded: stale_index` when stale.
- [ ] T5: `kb_validate` fails a turn (or flags, `--strict`) on stale/unresolved pins/claims.
- [ ] T6: `kb_lint` reports orphans / broken wikilinks / dup aliases / coverage gaps / external-staleness; `--fix` repairs the safe subset.
- [ ] T7: `kb_status` reports page counts by type, stale count, last run, pending queue, orphan count.
- [ ] T8: `kb_cite_source` records an external citation with `build_fingerprint`; re-fetch is section-granular.

**Artifacts** (files that must exist with real implementation, not stubs):

- [ ] A. `kb_*` engine over the shared store: `kb_pages` / `kb_pages_fts` / `kb_claims` / `kb_run_state` / `kb_page_baselines` / `kb_events` tables + additive migration (user_version / BEGIN IMMEDIATE / legacy-reconcile / trigger-aware split); `kb_bootstrap`, `kb_status`, `kb_generate`, `kb_index`, `kb_reindex`, `kb_query`, `kb_validate`, `kb_lint`, `kb_cite_source`, `kb_eval` handlers.
- [ ] B. `kb` management skill files: `SKILL.md`, `CONVENTIONS.md`, `PAGES.md`, `CLAIMS.md`, `MERMAID.md`.
- [ ] C. Acceptance scenarios in `acceptance.feature`.

**Key links** (critical connections between artifacts that must work together):

- [ ] The memory `SaveEvent` → `kb_generate` (eager-per-save) → `kb_index` (final step) → no-op proof (KB-INV-4).
- [ ] `kb_pages` (source of truth) ↔ `kb_pages_fts`/`kb_claims` (derived) in the shared DB; `index/last-update.json` is the file projection of `kb_run_state`.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):

- [ ] O1: Adding `kb_*` tables to the shared project DB (additive migration) — rides the memory change's migration runner; requires the four-property migration discipline. **Needs user approval before the migration lands.**
- [ ] O2: Semantic-leg injection point / hybrid weight blend ownership (single blend place) — design decision already fixed in `005-query` §2.1.
- [ ] O3: `kb_eval` gating policy (advisory vs gating) — default: gating for changes touching ranking / `thin` / semantic fallback.

## Global Constraints

- Go 1.22+ minimum to build (ADR-0031); no new dependencies without an ADR (ADR-0008).
- Spec zone (`.skillgrid/change-specs/**`) commits before code zone; one commit per zone.
- Conventional commits only; no AI-attribution trailers.
- TDD always on (failing test before code); BDD always on (every requirement has Given/When/Then).
- Agent authors `title`/`type`/`project`/`aliases`/`anchors`/`claims`/`sources`; engine owns `status`/`stale_after`/`generated`/`verified`.

## File Structure

- `mnemonic/internal/...` — store (kb tables + migration), generate, index, query, validate, lint, status, cite, eval, bootstrap, transport.
- `mnemonic-v2/skills/kb/` — `SKILL.md`, `CONVENTIONS.md`, `PAGES.md`, `CLAIMS.md`, `MERMAID.md`.
- `.skillgrid/knowledge/` — vault scaffold (`config.json`, `KNOWLEDGE_SCHEMA.md`, `templates/`, page folders, `.knowledgeignore`).

## Tasks

See `tasks.md` in this change folder. Execution waves: engine core → generate → index/query → validate/lint/status → skill + eval.

---

## Task 1: `kb_bootstrap` — vault scaffold

**Files:**
- Create: `mnemonic/internal/kb/bootstrap.go`
- Test: `mnemonic/internal/kb/bootstrap_test.go`

**Interfaces:**
- Consumes: none (reads/writes `.skillgrid/knowledge/`).
- Produces: `Bootstrap(ctx, dir) error` — idempotent scaffold.
- Seam: filesystem boundary.
- Deletion test: pass-through.

**SATISFIES:** KB vault bootstrap scenario.

- [ ] Step 1: Write the failing test
- [ ] Step 2: Run test to verify it fails
- [ ] Step 3: Write minimal implementation
- [ ] Step 4: Run test to verify it passes
- [ ] Step 5: Commit

## Task 2: `kb_index` / `kb_reindex` — derived FTS rebuild

**Files:**
- Create: `mnemonic/internal/kb/index.go`
- Test: `mnemonic/internal/kb/index_test.go`

**Interfaces:**
- Consumes: vault markdown (source of truth).
- Produces: `Index(ctx, dir) error`, `Reindex(ctx, dir) error`.
- Seam: store FTS boundary.

**SATISFIES:** KB index scenarios.

- [ ] Step 1: Write the failing test
- [ ] Step 2: Run test to verify it fails
- [ ] Step 3: Write minimal implementation
- [ ] Step 4: Run test to verify it passes
- [ ] Step 5: Commit
