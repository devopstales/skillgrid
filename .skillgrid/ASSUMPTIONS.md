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

**Component map.** *Pending re-decision for mnemonic-v2.* The old 4-way decomposition (Installer / Mnemonic Engine / Distribution Surface / Hub Content, ADR-0004) was deleted with the rest of mnemonic-v1 on 2026-10-31; the new product shape lands with the mnemonic-v2 specs (`.skillgrid/specs/2026-10-31-mnemonic-v2-*/`).

**Stack (measured from the repo).** Go 1.26 (module `github.com/devopstales/skillgrid/skillgrid-cli`); pure-Go SQLite (`modernc.org/sqlite`, cgo-free — `skillgrid doctor` reports `cgo: free`); tree-sitter (`odvcencio/gotreesitter`); ONNX inference in-process (`benedoc-inc/onnxer`, nomic-embed-code default); MCP server (`mark3labs/mcp-go`); Leiden community detection (`bluuewhale/loom`); fsnotify watcher; `charmbracelet/huh` for the interactive multiselect. Embedded SPA is Vite + React 19 + TypeScript + Tailwind 4 + TanStack Router, built into `skillgrid-cli/internal/mnemonic/http/ui/dist` and embedded via `//go:embed`.

**Store.** Per-project SQLite (WAL) under `~/.skillgrid/mnemonic/`, pure-Go driver (no CGo), embedded migrations, store pooling (refcounted `ProjectHandle` by project ID), TTL soft-expiry + distill/dream consolidation bound the store. No unbounded growth. *Schema history is being re-baselined by mnemonic-v2; the old migration inventory was retired with ADR-0011.*

**HTTP surface.** Binds `127.0.0.1:7438` (or `SKILLGRID_MNEMONIC_PORT`); REST routes for memory/code/sessions/web/health/tracker; embedded SPA served at `/`; `/openapi.yaml` + Swagger UI present. HTTP write routes are protected by the `SKILLGRID_HTTP_TOKEN` bearer check (`requireWriteAuth`); read routes stay open.

**MCP surface.** `skillgrid mcp` (stdio) exposes tool registries (mem/code/web/session/teams); an fsnotify auto-sync watcher + staleness gate keep the index within ~5s of file changes under watch; `--no-watch` disables the watcher. The stdio server is trusted by construction (spawned by the agent harness).

**Drift guards (implemented, ADR-0008 + ADR-0010).** `scripts/state-drift-check.mjs` and `scripts/ship-drift-check.mjs` are read-only Node verifiers (the `yaml` npm package is the repo's first and only npm dependency). Each emits a `SCOPE: <atom>` line (`COMPLETE`/`TRUNCATED`/`UNSCOPED`/`UNREADABLE`) after its `DRIFT:` verdict; the QA gate fails closed on any non-`COMPLETE` scope.

### Product requirements

The target users, personas, MoSCoW feature table, user flows, non-functional requirements, release line, and competitive position are in `.skillgrid/artifacts/00-prd.md`. Open it when a change touches scope or acceptance. The one-paragraph product statement is the **Product** bullet at the top of this tier.

---

## INFERRED (HYPOTHESIS)

Treated as true but not yet confirmed. Each rests on a named assumption; promote to VERIFIED (with evidence) or LOCKED (with user OK) before it carries a decision.

- **H1 — the secondary persona is real but small.** The "AI-tooling builder who embeds the capability" persona falls out of the MCP/HTTP surface and is not a separate product. Assumed (PRD Question 2): no explicit MCP tool-contract semver in v1.0; the OpenAPI is the de-facto contract for the HTTP surface. *Not confirmed* by external demand.
- **H2 — the advisory-only posture carries into `eval`.** The config's `coverage_min: 0` / no hard thresholds means `eval` reports a baseline, it does not gate (PRD Assumption 1). *Not confirmed* that a future release won't add a hard threshold; that would be a change, not a reframe.
- **H4 — the shipped skills + hooks ship as content, their behavior is out of PRD scope.** Their *shipping* is in scope (Installer section); their *behavior* is covered in the user guide (PRD Assumption 3). This is the assumption that lets the engine PRD and the pipeline skills stay separate documents. (Skill count is 37 as of 2026-09-29; the assumption is about scope, not count.)

---

## LOCKED

The ADR index — the in-force decisions. Adding a row requires an explicit user OK.

### ADR index

The ADR index. One row per decision: a short description and a link to the full decision file. This table is an index, not the body — `status`, `supersedes`, and `date` live in each file's frontmatter; open the linked file for the full Context / Decision / Consequences.

**In force** = the file's `status: accepted` AND no later ADR's `supersedes:` names it. Superseded rows stay in the index (frozen) but are marked `no`. **IRON RULE: never delete an ADR file or its row — supersede by adding a new file that names it and flipping its row here.**

> 2026-10-31: the old-mnemonic-v1 ADR set (0001–0006, 0009, 0011–0012, 0016–0018, 0020, 0022–0030, 0032–0038) was deleted with the rest of mnemonic-v1's spec zone in preparation for mnemonic-v2. Their numbers are retired: **do not reissue them.** The deleted rows no longer exist in this index (exception to the "never delete a row" rule, made explicit by the user for the v1→v2 reset).

| # | Description | In force | File |
|---|-------------|----------|------|
| 0007 | SDD multi-method docs viewer with generic, existence-gated doc roots | yes | `.skillgrid/artifacts/04-adr-0007-sdd-multi-method-docs-viewer.md` |
| 0008 | State drift guard: Node.js + `yaml` package, read-only verifier | yes | `.skillgrid/artifacts/04-adr-0008-state-drift-guard-js-yaml.md` |
| 0010 | Verification scope discriminator: a zero is never a bare zero | yes | `.skillgrid/artifacts/04-adr-0010-verification-scope-discriminator.md` |
| 0013 | Source of truth: the repo is authoritative; Backlog task ID ↔ commit SHA is the linkage | yes | `.skillgrid/artifacts/04-adr-0013-repo-source-of-truth.md` |
| 0014 | (removed 2026-09-30) | no | `.skillgrid/artifacts/04-adr-0014-removed.md` |
| 0015 | (removed 2026-09-30) | no | `.skillgrid/artifacts/04-adr-0015-removed.md` |
| 0019 | Locked decisions are ADR files; ASSUMPTIONS.md holds the path | yes | `.skillgrid/artifacts/04-adr-0019-decisions-are-files.md` |
| 0021 | Pre-tool policy is opt-in, first-match field rules, and fails open | yes | `.skillgrid/artifacts/04-adr-0021-pre-tool-policy-fail-open.md` |
| 0031 | Go 1.22+ minimum to build (raised from a locked constraint to a decision) | yes | `.skillgrid/artifacts/04-adr-0031-go-version-floor.md` |
| 0039 | Code-style group: three axes — (1) agent-style v0.9.0 (CC-BY-4.0) prose standard (generation + review), (2) code-design vocabulary (`codebase-design` from `verification/`), (3) 17 Google language style guides (MIT, TestDino 2026) one skill per language; manual vendoring bumps; no new Go deps | yes | `.skillgrid/artifacts/04-adr-0039-agent-style-code-style-group.md` |
| 0040 | Mnemonic v2 module extraction: fresh `mnemonic-v2/` module (path `github.com/devopstales/skillgrid/mnemonic-v2`), v1 renamed to `mnemonic_back/`, coexistence until v2 ships | yes | `.skillgrid/artifacts/04-adr-0040-mnemonic-v2-module-extraction.md` |
| 0041 | Mnemonic v2 Postgres driver: `lib/pq` (pure-Go, `database/sql`-confined to `postgres.go`) | yes | `.skillgrid/artifacts/04-adr-0041-mnemonic-v2-postgres-driver.md` |
| 0042 | Goldfish plan review: before-code document review is a fresh-context 3-stage subflow of writing-blueprints (zoom-out / comprehension / critic), not a new phase and not part of qa | yes | `.skillgrid/artifacts/04-adr-0042-goldfish-plan-review.md` |
| 0043 | Code-index tree-sitter dependency: `gotreesitter` v0.55.1 (pure-Go, cgo-free) for the rich-6 + map-tier extraction ladder | yes | `.skillgrid/artifacts/04-adr-0043-code-index-tree-sitter-dep.md` |
| 0044 | Code-index fsnotify dependency: `fsnotify` v1.10.1 for the watch loop (trigger-not-a-poll) | yes | `.skillgrid/artifacts/04-adr-0044-code-index-fsnotify-dep.md` |
| 0045 | Code-index conditional ONNX dependency: `onnxer` v0.7.0 only if the 768-dim ONNX embedder is wired (hash-64 is the default, no dep) | yes (conditional) | `.skillgrid/artifacts/04-adr-0045-code-index-onnxer-conditional.md` |
| 0046 | Code-index `Store` interface extension: additive code-index write/read + vector + trail + ExecDDL + Begin/Txn methods (one-way doors D2/D3) | yes | `.skillgrid/artifacts/04-adr-0046-code-index-store-extension.md` |

**Highest sequence in use:** 0046 (next ADR is `04-adr-0047-slug.md`). This index is a pointer — the body of each decision is the linked file (ADR-0019).

Operational rules (the hard limits a change must respect — build toolchain, commit format, dependency policy, spec/code zone order) live in `AGENTS.md`, not here. `AGENTS.md` is the real source for those; the onboarding skill renders them into the agent prompt. A rule that is actually a decision belongs in this index as an ADR instead.

---

## Open Questions

Questions that are not yet decided and not yet prototyped.

1. **MCP tool-contract versioning** — is the secondary persona strong enough to warrant stable, versioned MCP tool contract (semver on the tool surface)? Assumed no in v1.0 (see INFERRED H1).
2. **README alignment** — the README is still installer-first; aligning it to the engine-first PRD is a follow-up ticket, not part of this consolidation.
3. **Knowledge-retrieval direction** — the goal is a fast, cheap knowledge-retrieval surface that makes the agent smarter, sourced from the Mnemonic store (observations + web_cache), plus ingesting local `.skillgrid/` files into that store. The direction question is now owned by the mnemonic-v2 specs (`.skillgrid/specs/2026-10-31-mnemonic-v2-*/`).
