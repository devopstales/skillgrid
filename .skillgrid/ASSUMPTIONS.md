# ASSUMPTIONS

The live record of what skillgrid *is*, which decisions are in force, and what is *locked*. Decision bodies live in `.skillgrid/artifacts/04-adr-NNNN-slug.md`; this file stores the path. The terms glossaries (`.skillgrid/artifacts/01-business-terms.md`, `02-technical-terms.md`) are a separate glossary.

**Read order:** VERIFIED (what we know to be true) → INFERRED (hypotheses we are acting on) → LOCKED (the ADR index — in-force decisions) → Open Questions.

**How this file is maintained:**
- **VERIFIED** — facts confirmed against the code, a prototype, or a primary source. Brainstorming writes these during the interview as they are confirmed.
- **INFERRED (HYPOTHESIS)** — things we are treating as true but have not confirmed. Mark each with the assumption it rests on. Never let an INFERRED item carry a decision; promote it to VERIFIED (with evidence) or to LOCKED (with a user OK) before it does.
- **LOCKED** — the ADR index: one pointer row per in-force decision (path only). Requires an explicit user OK to add a row. Supersede by a new file + table flip; **never delete an ADR file or its row** (the IRON RULE). Operational rules that are *not* decisions live in `AGENTS.md`, not here.
- **Open Questions** — questions that are not yet decided and not yet prototyped.

Product requirements live in `.skillgrid/artifacts/00-prd.md` (reference). Locked decisions are files under `.skillgrid/artifacts/04-adr-NNNN-slug.md`. This file's in-force table stores the path only (ADR-0019).

---

## VERIFIED

Confirmed against the code, a prototype, or a primary source. These are the facts the rest of the record stands on.

**Product.** skillgrid is a local-first engine for agent memory + code intelligence, distributed as a single binary exposed over three transports: CLI, MCP (stdio), and HTTP/REST. It persists decisions, observations, and code context outside the chat so AI coding agents (OpenCode, Kilo, Cursor) keep working across sessions, and it indexes a project's code so the agent can orient, search, and assess blast radius before editing. Everything runs on the developer's machine — no cloud, no telemetry.

**Component map (ADR-0004).** Four product components, each with one clear purpose and a testable boundary:

| Component | Responsibility | Boundary |
|-----------|----------------|----------|
| **Installer** | `install` / `sync-repo` / `setup`; wires agents, MCP, Hub Content | Writes to `~/.skillgrid/`, `~/.agents/`, and agent config files; the only writer of agent MCP config |
| **Mnemonic Engine** | Memory (observations, search, provenance, governance, session relay) + Code Intelligence (indexing, orientation, graph, hybrid search, taint) + shared project resolution | Owns the per-project SQLite store; no transport concerns |
| **Distribution Surface** | MCP stdio (`mcp`), HTTP/REST + embedded SPA + OpenAPI/Swagger (`serve`), and the CLI command group | Owns the transports; no data ownership; the one real trust boundary lives here (MCP-spawn model, ADR-0005) |
| **Hub Content** | `.agents/skills/` (37 skills as of 2026-09-29), `hooks/` + `git-hooks/` + `plugins/`, `config.d/` | Shipped by the Installer (staged to `~/.skillgrid/`); not a runtime component of the binary |

**Stack (measured from the repo).** Go 1.26 (module `github.com/devopstales/skillgrid/skillgrid-cli`); pure-Go SQLite (`modernc.org/sqlite`, cgo-free — `skillgrid doctor` reports `cgo: free`); tree-sitter (`odvcencio/gotreesitter`); ONNX inference in-process (`benedoc-inc/onnxer`, nomic-embed-code default); MCP server (`mark3labs/mcp-go`); Leiden community detection (`bluuewhale/loom`); fsnotify watcher; `charmbracelet/huh` for the interactive multiselect. Embedded SPA is Vite + React 19 + TypeScript + Tailwind 4 + TanStack Router, built into `skillgrid-cli/internal/mnemonic/http/ui/dist` and embedded via `//go:embed`.

**Store.** Per-project SQLite (WAL) under `~/.skillgrid/mnemonic/`, pure-Go driver (no CGo), embedded migrations (verified 2026-09-29: `001_schema.sql` with 002–039 squashed in, plus `040_session_events.sql`, `041_drop_handoff_tables.sql`, `042_vec0_tables.sql` — migration 043 (bi-temporal, ADR-0011) is pending apply), store pooling (refcounted `ProjectHandle` by project ID), TTL soft-expiry + distill/dream consolidation bound the store. No unbounded growth.

**HTTP surface.** Binds `127.0.0.1:7438` (or `SKILLGRID_MNEMONIC_PORT`); REST routes for memory/code/sessions/web/health/tracker; embedded SPA served at `/`; `/openapi.yaml` + Swagger UI present. HTTP write routes are protected by the `SKILLGRID_HTTP_TOKEN` bearer check (`requireWriteAuth`); read routes stay open.

**MCP surface.** `skillgrid mcp` (stdio) exposes tool registries (mem/code/web/session/teams); an fsnotify auto-sync watcher + staleness gate keep the index within ~5s of file changes under watch; `--no-watch` disables the watcher. The stdio server is trusted by construction (spawned by the agent harness).

**Vector search (measured, ADR-0006 + ADR-0009).** The live `aiskillgrid` store held 3,905 symbol + 18,676 chunk vectors (768-dim, 54.7 MB of BLOBs). The semantic leg is an **in-memory brute-force cosine over a process-global vector cache** (`hybrid/vectorcache.go`), invalidated on re-index / model swap. Measured cost: pre-cache ~220 ms/query, of which ~190 ms (87%) was the SQLite BLOB scan + row decode through the pure-Go reader — not the cosine math (~28 ms). Post-cache the warm semantic leg is ~28 ms. The cgo-free single-binary invariant is preserved. A prototype (`.skillgrid/prototypes/001-cgo-free-vector-db`) measured the in-SQL `modernc.org/sqlite/vec` path at ~6.9 s median top-K at 100K vectors (~1.4 s at 20K) and `viant/sqlite-vec` as blocked (packaging bug + `ensureIndex` query deadlock).

**Bi-temporal observations (ADR-0011 — decision committed, implementation not in HEAD).** The ADR specifies that `observations` carries `valid_at`, `invalid_at`, `superseded_by` (planned migration 043); that the save path (`SaveWithAction`) classifies each write as Add/Update/Delete/Noop and produces the supersede chain automatically; that `mem_save` returns `action` + `superseded_id`; and that the 7 primary read sites filter `invalid_at` so superseded observations do not leak into results. The ADR + spec are committed, but at the 2026-09-29 verification the code tree shows none of it (no columns, no migration 043, no `SaveWithAction`) — the apply is pending. Promote this bullet back to "implemented" when the code lands.

**Drift guards (implemented, ADR-0008 + ADR-0010).** `scripts/state-drift-check.mjs` and `scripts/ship-drift-check.mjs` are read-only Node verifiers (the `yaml` npm package is the repo's first and only npm dependency). Each emits a `SCOPE: <atom>` line (`COMPLETE`/`TRUNCATED`/`UNSCOPED`/`UNREADABLE`) after its `DRIFT:` verdict; the QA gate fails closed on any non-`COMPLETE` scope.

**v1.0 line (ADR-0003).** v1.0 is "the engine is trustworthy" — verifiable with the existing tools (`install_test.go`, `doctor --strict`, `eval`, the staleness gate). The embedded dashboard ships as stubs in v1.0; it is Phase 2 (v1.1).

**Known gaps (current state, verified against the tree 2026-09-29).** UI dashboard: 10 feature modules are real (decisions, docs, git, kanban, mnemonic, plans, prototypes, sessions, tracker) but `SettingsPage` is still `StubPage` (Phase 2, v1.1); the `ui:build` Taskfile task now exists and `build:all` depends on it — `go build` alone still requires a pre-built `ui/dist`; tracker providers 501 on unknown `/tracker/*` providers (v1.1); the process-pass LLM labeler is a deterministic stub `processLLMStub` (v1.2); the ADR-0011 bi-temporal save path is decided + specified but not yet in the code (see above); the README is stale (installer-first framing, smaller command surface, "bare `skillgrid` = install") — a follow-up ticket; housekeeping: a stray `internal/mnemonic/mcp/.git` nested repo (still present; remove) and a migration numbering gap (002–039 squashed into `001_schema.sql`, so 032 was never a file — known, no action).

### Product requirements

The target users, personas, MoSCoW feature table, user flows, non-functional requirements, release line, and competitive position are in `.skillgrid/artifacts/00-prd.md`. Open it when a change touches scope or acceptance. The one-paragraph product statement is the **Product** bullet at the top of this tier.

---

## INFERRED (HYPOTHESIS)

Treated as true but not yet confirmed. Each rests on a named assumption; promote to VERIFIED (with evidence) or LOCKED (with user OK) before it carries a decision.

- **H1 — the secondary persona is real but small.** The "AI-tooling builder who embeds the capability" persona falls out of the MCP/HTTP surface and is not a separate product. Assumed (PRD Question 2): no explicit MCP tool-contract semver in v1.0; the OpenAPI is the de-facto contract for the HTTP surface. *Not confirmed* by external demand.
- **H2 — the advisory-only posture carries into `eval`.** The config's `coverage_min: 0` / no hard thresholds means `eval` reports a baseline, it does not gate (PRD Assumption 1). *Not confirmed* that a future release won't add a hard threshold; that would be a change, not a reframe.
- **H3 — the v1.1 dashboard prioritization is undecided.** Memories+Sessions-first vs. Graph+Tracker-first was assumed to be decided at v1.1 scoping (PRD Question 1). *Partially overtaken*: as of 2026-09-29 the dashboard ships both families (mnemonic/memories/sessions/graph + tracker) as real feature modules — only `SettingsPage` is a `StubPage`. Revisit whether H3 still carries a decision.
- **H4 — the shipped skills + hooks ship as content, their behavior is out of PRD scope.** Their *shipping* is in scope (Installer section); their *behavior* is covered in the user guide (PRD Assumption 3). This is the assumption that lets the engine PRD and the pipeline skills stay separate documents. (Skill count is 37 as of 2026-09-29; the assumption is about scope, not count.)
- **H5 — the `Distribution Surface` term is stable enough to use in the component map.** The code does not use the term (it has `internal/mnemonic/mcp`, `internal/mnemonic/http`, and the CLI command group as separate packages). The term is in the glossary; if it drifts, ADR-0004's component map needs a re-word, not a re-decomposition.

---

## LOCKED

The ADR index — the in-force decisions. Adding a row requires an explicit user OK.

### ADR index

The ADR index. One row per decision: a short description and a link to the full decision file. This table is an index, not the body — `status`, `supersedes`, and `date` live in each file's frontmatter; open the linked file for the full Context / Decision / Consequences.

**In force** = the file's `status: accepted` AND no later ADR's `supersedes:` names it. Superseded rows stay in the index (frozen) but are marked `no`. **IRON RULE: never delete an ADR file or its row — supersede by adding a new file that names it and flipping its row here.**

| # | Description | In force | File |
|---|-------------|----------|------|
| 0001 | PRD scope is the whole Hub Product, not the binary alone | yes | `.skillgrid/artifacts/04-adr-0001-prd-scope-whole-hub.md` |
| 0002 | PRD framing is engine-first, not installer-first | yes | `.skillgrid/artifacts/04-adr-0002-prd-engine-first-framing.md` |
| 0003 | v1.0 line is "the engine is trustworthy"; UI is Phase 2 | yes | `.skillgrid/artifacts/04-adr-0003-v1.0-line-engine-trustworthy.md` |
| 0004 | PRD component map is a 4-way decomposition | yes | `.skillgrid/artifacts/04-adr-0004-four-component-decomposition.md` |
| 0005 | Trust boundary: MCP-spawned process with project read access | yes | `.skillgrid/artifacts/04-adr-0005-trust-boundary-mcp-spawn-model.md` |
| 0006 | Vector search: in-memory brute-force cosine, no SQLite vector extension (amended by 0009) | yes | `.skillgrid/artifacts/04-adr-0006-vector-search-in-memory-brute-force.md` |
| 0007 | SDD multi-method docs viewer with generic, existence-gated doc roots | yes | `.skillgrid/artifacts/04-adr-0007-sdd-multi-method-docs-viewer.md` |
| 0008 | State drift guard: Node.js + `yaml` package, read-only verifier | yes | `.skillgrid/artifacts/04-adr-0008-state-drift-guard-js-yaml.md` |
| 0009 | Vector search: in-SQL sqlite-vec latency corrected; viant deferred; revisit path confirmed | yes | `.skillgrid/artifacts/04-adr-0009-vector-search-in-sql-latency-viant-deferred.md` |
| 0010 | Verification scope discriminator: a zero is never a bare zero | yes | `.skillgrid/artifacts/04-adr-0010-verification-scope-discriminator.md` |
| 0011 | Observations are bi-temporal; the save path classifies each write as Add/Update/Delete/Noop | yes | `.skillgrid/artifacts/04-adr-0011-observations-are-bitemporal.md` |
| 0012 | SQLite as the second-brain store; llm-wiki markdown files rejected | yes | `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md` |
| 0013 | Source of truth: the repo is authoritative; Backlog task ID ↔ commit SHA is the linkage | yes | `.skillgrid/artifacts/04-adr-0013-repo-source-of-truth.md` |
| 0014 | (removed 2026-09-30) | no | `.skillgrid/artifacts/04-adr-0014-removed.md` |
| 0015 | (removed 2026-09-30) | no | `.skillgrid/artifacts/04-adr-0015-removed.md` |
| 0016 | Mnemonic second-brain capability layer | yes | `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` |
| 0017 | Embedded SPA code graph uses D3 force layout (replaces Sigma) | yes | `.skillgrid/artifacts/04-adr-0017-d3-force-graph.md` |
| 0018 | mem_search returns additive per-signal scores; RRF runs only on the owner-scoped path | yes | `.skillgrid/artifacts/04-adr-0018-mem-search-additive-signals.md` |
| 0019 | Locked decisions are ADR files; ASSUMPTIONS.md holds the path | yes | `.skillgrid/artifacts/04-adr-0019-decisions-are-files.md` |
| 0020 | Execution coordination lives in the SDD ledger; Mnemonic stores an index | yes | `.skillgrid/artifacts/04-adr-0020-sdd-ledger-owns-execution.md` |
| 0021 | Pre-tool policy is opt-in, first-match field rules, and fails open | yes | `.skillgrid/artifacts/04-adr-0021-pre-tool-policy-fail-open.md` |
| 0022 | The host agent is the memory observer; checkpoints are server-gated | yes | `.skillgrid/artifacts/04-adr-0022-host-agent-memory-checkpoint.md` |
| 0023 | Mnemonic chat LLM is OpenAI-compatible HTTP only; one client attaches all seams | yes | `.skillgrid/artifacts/04-adr-0023-openai-compatible-llm-provider.md` |
| 0024 | Team-shared memory state: local SQLite + outbox journal against a VM PostgreSQL (LWW + tombstones, tokens-as-roles, schema-per-project) | yes | `.skillgrid/artifacts/04-adr-0024-central-pg-shared-memory-state.md` |
| 0025 | Context Harness owns the session context lifecycle (absorbs `session_inject`; capture, `ctx_query`, routing, `ctx` CLI) | yes | `.skillgrid/artifacts/04-adr-0025-context-harness-owner.md` |
| 0026 | `indexed_files` reuses the observations schema shape; `ctx_search` fuses sandbox + indexed via RRF | yes | `.skillgrid/artifacts/04-adr-0026-indexed-files-reuses-observations-schema.md` |
| 0027 | Context Harness implements the Context Language Model (mirror + overflow guard + calibration; Go owns state, Node owns request path; opt-in via `clm:` config) | yes | `.skillgrid/artifacts/04-adr-0027-clm-context-language-model.md` |
| 0028 | `context_revisions` is session-scoped audit (purged at session end; raw history + mirror are the source of truth) | yes | `.skillgrid/artifacts/04-adr-0028-context-revisions-session-scoped.md` |
| 0029 | Compaction advisory gate (one combined LLM call, fail-open) + `steering` column on `context_revisions` + proactive build; extends ADR-0027/0028; opt-in via `mnemonic.compaction` config | yes | `.skillgrid/artifacts/04-adr-0029-compaction-advisory-steering.md` |
| 0030 | Scan findings store: structured `scan_*`/`dep_*` tables + FTS, not blobs or per-tool tables | yes | `.skillgrid/artifacts/04-adr-0030-scan-findings-structured-store.md` |
| 0031 | Go 1.22+ minimum to build (raised from a locked constraint to a decision) | yes | `.skillgrid/artifacts/04-adr-0031-go-version-floor.md` |
| 0032 | opencode + Kilo native TS plugin architecture: 5 plugins from `plugins/opencode/`, Kilo copy-at-install, shell hooks retired, facts HTTP routes | yes | `.skillgrid/artifacts/04-adr-0032-opencode-kilo-native-plugins.md` |
| 0033 | Three-module Go workspace: `ctx/` is a library module importing `mnemonic`, no own DB handle | yes | `.skillgrid/artifacts/04-adr-0033-three-module-workspace-ctx.md` |
| 0034 | `ctx` is a `mnemonic` subcommand group (`mnemonic ctx …`), not a standalone binary; `mnemonic stats` unchanged | yes | `.skillgrid/artifacts/04-adr-0034-ctx-cli-mnemonic-subcommand.md` |
| 0035 | CTX judgment model is local Ollama (default `tev1:0.8b` via `/v1/systemone`), fail-open to pre-filter | yes | `.skillgrid/artifacts/04-adr-0035-local-ollama-judgment-model.md` |
| 0036 | CTX config lives at `config.d/ctx.yaml` (machine-local default + repo-local override), reuses the `config.d` loader | yes | `.skillgrid/artifacts/04-adr-0036-ctx-config-configd.md` |

**Highest sequence in use:** 0036 (next ADR is `04-adr-0037-slug.md`). This index is a pointer — the body of each decision is the linked file (ADR-0019). Number 0012 is the SQLite store record; the 2026-09-29 consolidation text that was inlined under that number is preserved in ADR-0019 and is not in force.

Operational rules (the hard limits a change must respect — build toolchain, commit format, dependency policy, spec/code zone order) live in `AGENTS.md`, not here. `AGENTS.md` is the real source for those; the onboarding skill renders them into the agent prompt. A rule that is actually a decision belongs in this index as an ADR instead.

---

## Open Questions

Questions that are not yet decided and not yet prototyped.

1. **v1.1 dashboard prioritization** — Memories+Sessions-first vs. Graph+Tracker-first. Decided at v1.1 scoping, not now (see INFERRED H3). (Partially answered in practice: the dashboard already ships 10 feature modules, including both Memories+Sessions and Graph; what remains is settings + polish.)
2. **MCP tool-contract versioning** — is the secondary persona strong enough to warrant stable, versioned MCP tool contract (semver on the tool surface)? Assumed no in v1.0 (see INFERRED H1).
3. **README alignment** — the README is still installer-first; aligning it to the engine-first PRD is a follow-up ticket, not part of this consolidation.
4. **Housekeeping** — remove the stray `internal/mnemonic/mcp/.git` nested repo (still present as of 2026-09-29); the migration numbering gap (002–039 squashed into `001_schema.sql`) is a known issue with no action.
5. **Knowledge-retrieval direction** — the goal is a fast, cheap knowledge-retrieval surface that makes the agent smarter, sourced from the Mnemonic SQLite store (observations + web_cache), plus ingesting local `.skillgrid/` files into that store. Needs a fresh decision/ADR before any re-attempt.
