# ASSUMPTIONS

The live, single-file record of what skillgrid *is*, what we have *decided*, and what is *locked*. This is the AI's understanding of the project plus its decisions, consolidated into one file. The terms glossaries (`.skillgrid/artifacts/01-business-terms.md`, `02-technical-terms.md`) are a separate glossary — nothing else lives in this file's place.

**Read order:** VERIFIED (what we know to be true) → INFERRED (hypotheses we are acting on) → LOCKED (decisions + user-locked boundaries) → Open Questions.

**How this file is maintained:**
- **VERIFIED** — facts confirmed against the code, a spike, or a primary source. Brainstorming writes these during the interview as they are confirmed.
- **INFERRED (HYPOTHESIS)** — things we are treating as true but have not confirmed. Mark each with the assumption it rests on. Never let an INFERRED item carry a decision; promote it to VERIFIED (with evidence) or to LOCKED (with a user OK) before it does.
- **LOCKED** — decisions (ADRs) and user-locked constraints. Requires an explicit user OK to add. Supersede by link + table flip; **never delete an ADR entry** (the IRON RULE).
- **Open Questions** — questions that are not yet decided and not yet spiked.

The full PRD, ADR files, and locked-constraints file were folded into this file on 2026-09-29 and archived to `.skillgrid/archive/`.

---

## VERIFIED

Confirmed against the code, a spike, or a primary source. These are the facts the rest of the record stands on.

**Product.** skillgrid is a local-first engine for agent memory + code intelligence, distributed as a single binary exposed over three transports: CLI, MCP (stdio), and HTTP/REST. It persists decisions, observations, and code context outside the chat so AI coding agents (OpenCode, Kilo, Cursor) keep working across sessions, and it indexes a project's code so the agent can orient, search, and assess blast radius before editing. Everything runs on the developer's machine — no cloud, no telemetry.

**Component map (ADR-0004).** Four product components, each with one clear purpose and a testable boundary:

| Component | Responsibility | Boundary |
|-----------|----------------|----------|
| **Installer** | `install` / `sync-repo` / `setup`; wires agents, MCP, Hub Content | Writes to `~/.skillgrid/`, `~/.agents/`, and agent config files; the only writer of agent MCP config |
| **Mnemonic Engine** | Memory (observations, search, provenance, governance, session relay) + Code Intelligence (indexing, orientation, graph, hybrid search, taint) + shared project resolution | Owns the per-project SQLite store; no transport concerns |
| **Distribution Surface** | MCP stdio (`mcp`), HTTP/REST + embedded SPA + OpenAPI/Swagger (`serve`), and the CLI command group | Owns the transports; no data ownership; the one real trust boundary lives here (MCP-spawn model, ADR-0005) |
| **Hub Content** | `.agents/skills/` (30 skills), `hooks/` + `git-hooks/` + `plugins/`, `config.d/` | Shipped by the Installer (staged to `~/.skillgrid/`); not a runtime component of the binary |

**Stack (measured from the repo).** Go 1.26 (module `github.com/devopstales/skillgrid/skillgrid-cli`); pure-Go SQLite (`modernc.org/sqlite`, cgo-free — `skillgrid doctor` reports `cgo: free`); tree-sitter (`odvcencio/gotreesitter`); ONNX inference in-process (`benedoc-inc/onnxer`, nomic-embed-code default); MCP server (`mark3labs/mcp-go`); Leiden community detection (`bluuewhale/loom`); fsnotify watcher; `charmbracelet/huh` for the interactive multiselect. Embedded SPA is Vite + React 19 + TypeScript + Tailwind 4 + TanStack Router, built into `skillgrid-cli/internal/mnemonic/http/ui/dist` and embedded via `//go:embed`.

**Store.** Per-project SQLite (WAL) under `~/.skillgrid/mnemonic/`, pure-Go driver (no CGo), embedded migrations (001→043), store pooling (refcounted `ProjectHandle` by project ID), TTL soft-expiry + distill/dream consolidation bound the store. No unbounded growth.

**HTTP surface.** Binds `127.0.0.1:7438` (or `SKILLGRID_MNEMONIC_PORT`); REST routes for memory/code/sessions/web/health/tracker; embedded SPA served at `/`; `/openapi.yaml` + Swagger UI present. HTTP write routes are protected by the `SKILLGRID_HTTP_TOKEN` bearer check (`requireWriteAuth`); read routes stay open.

**MCP surface.** `skillgrid mcp` (stdio) exposes tool registries (mem/code/web/session/teams); an fsnotify auto-sync watcher + staleness gate keep the index within ~5s of file changes under watch; `--no-watch` disables the watcher. The stdio server is trusted by construction (spawned by the agent harness).

**Vector search (measured, ADR-0006 + ADR-0009).** The live `aiskillgrid` store held 3,905 symbol + 18,676 chunk vectors (768-dim, 54.7 MB of BLOBs). The semantic leg is an **in-memory brute-force cosine over a process-global vector cache** (`hybrid/vectorcache.go`), invalidated on re-index / model swap. Measured cost: pre-cache ~220 ms/query, of which ~190 ms (87%) was the SQLite BLOB scan + row decode through the pure-Go reader — not the cosine math (~28 ms). Post-cache the warm semantic leg is ~28 ms. The cgo-free single-binary invariant is preserved. A spike (`.skillgrid/spikes/001-cgo-free-vector-db`) measured the in-SQL `modernc.org/sqlite/vec` path at ~6.9 s median top-K at 100K vectors (~1.4 s at 20K) and `viant/sqlite-vec` as blocked (packaging bug + `ensureIndex` query deadlock).

**Bi-temporal observations (implemented, ADR-0011).** `observations` carries `valid_at`, `invalid_at`, `superseded_by` (migration 043); the save path (`SaveWithAction`) classifies each write as Add/Update/Delete/Noop and produces the supersede chain automatically; `mem_save` returns `action` + `superseded_id`. The 7 primary read sites filter `invalid_at` so superseded observations do not leak into results.

**Drift guards (implemented, ADR-0008 + ADR-0010).** `scripts/state-drift-check.mjs` and `scripts/ship-drift-check.mjs` are read-only Node verifiers (the `yaml` npm package is the repo's first and only npm dependency). Each emits a `SCOPE: <atom>` line (`COMPLETE`/`TRUNCATED`/`UNSCOPED`/`UNREADABLE`) after its `DRIFT:` verdict; the QA gate fails closed on any non-`COMPLETE` scope.

**v1.0 line (ADR-0003).** v1.0 is "the engine is trustworthy" — verifiable with the existing tools (`install_test.go`, `doctor --strict`, `eval`, the staleness gate). The embedded dashboard ships as stubs in v1.0; it is Phase 2 (v1.1).

**Known gaps (current state, committed as roadmap).** UI dashboard pages are all `StubPage` (Phase 2, v1.1); the `ui:build` Taskfile task is missing and `go build` requires a pre-built `ui/dist` (v1.1); tracker provider 501s on some `/tracker/*` providers (v1.1); the process-pass LLM labeler is a deterministic stub `processLLMStub` (v1.2); the README is stale (installer-first framing, smaller command surface, "bare `skillgrid` = install") — a follow-up ticket; housekeeping: a stray `internal/mnemonic/mcp/.git` nested repo (remove) and a migration 032 numbering gap (known, no action).

---

## INFERRED (HYPOTHESIS)

Treated as true but not yet confirmed. Each rests on a named assumption; promote to VERIFIED (with evidence) or LOCKED (with user OK) before it carries a decision.

- **H1 — the secondary persona is real but small.** The "AI-tooling builder who embeds the capability" persona falls out of the MCP/HTTP surface and is not a separate product. Assumed (PRD Question 2): no explicit MCP tool-contract semver in v1.0; the OpenAPI is the de-facto contract for the HTTP surface. *Not confirmed* by external demand.
- **H2 — the advisory-only posture carries into `eval`.** The config's `coverage_min: 0` / no hard thresholds means `eval` reports a baseline, it does not gate (PRD Assumption 1). *Not confirmed* that a future release won't add a hard threshold; that would be a change, not a reframe.
- **H3 — the v1.1 dashboard prioritization is undecided.** Memories+Sessions-first vs. Graph+Tracker-first is assumed to be decided at v1.1 scoping, not now (PRD Question 1). *Not confirmed* either way.
- **H4 — the 30 skills + hooks ship as content, their behavior is out of PRD scope.** Their *shipping* is in scope (Installer section); their *behavior* is covered in the user guide (PRD Assumption 3). This is the assumption that lets the engine PRD and the pipeline skills stay separate documents.
- **H5 — the `Distribution Surface` term is stable enough to use in the component map.** The code does not use the term (it has `internal/mnemonic/mcp`, `internal/mnemonic/http`, and the CLI command group as separate packages). The term is in the glossary; if it drifts, ADR-0004's component map needs a re-word, not a re-decomposition.

---

## LOCKED

Decisions (ADRs) and user-locked constraints. Adding here requires an explicit user OK.

### In-force set

Single source for what is **currently in force**. **In force** = `status: accepted` AND no later ADR's `supersedes` names it. Superseded / deprecated entries stay in the table (frozen) but are marked out of force. **IRON RULE: never delete an ADR entry — supersede by adding a new entry that names it and flipping its row here.**

| # | Title | Status | Supersedes | Amends | Date | In force |
|---|-------|--------|------------|--------|------|----------|
| 0001 | PRD scope is the whole Hub Product, not the binary alone | accepted | — | — | 2026-09-16 | yes |
| 0002 | PRD framing is engine-first, not installer-first | accepted | — | — | 2026-09-16 | yes |
| 0003 | v1.0 line is "the engine is trustworthy"; UI is Phase 2 | accepted | — | — | 2026-09-16 | yes |
| 0004 | PRD component map is a 4-way decomposition | accepted | — | — | 2026-09-16 | yes |
| 0005 | Trust boundary: MCP-spawned process with project read access | accepted | — | — | 2026-09-16 | yes |
| 0006 | Vector search: in-memory brute-force cosine, no SQLite vector extension | accepted | — | — | 2026-09-16 | yes (amended by 0009) |
| 0007 | SDD multi-method docs viewer with generic, existence-gated doc roots | accepted | — | — | 2026-09-17 | yes |
| 0008 | State drift guard: Node.js + `yaml` package, read-only verifier | accepted | — | — | 2026-09-24 | yes |
| 0009 | Vector search: in-SQL sqlite-vec latency corrected; viant deferred; revisit path confirmed | accepted | — | 0006 | 2026-09-24 | yes |
| 0010 | Verification scope discriminator: a zero is never a bare zero | accepted | — | — | 2026-09-24 | yes |
| 0011 | Observations are bi-temporal; the save path classifies each write as Add/Update/Delete/Noop | accepted | — | — | 2026-09-24 | yes |
| 0012 | ASSUMPTIONS consolidation: PRD + ADRs + locked constraints fold into one root file | accepted | — | — | 2026-09-29 | yes |
| 0013 | Source of truth: the repo is authoritative; Backlog task ID ↔ commit SHA is the linkage | accepted | — | — | 2026-09-29 | yes |

**Highest sequence in use:** 0013 (next ADR is `### ADR-0014` below).

### ADR-0001 — PRD scope is the whole Hub Product, not the binary alone

**Decision.** The PRD is scoped to the whole Hub Product (CLI binary + Mnemonic Engine + Distribution Surface + Hub Content), one product document — not the binary alone, not the installer alone.
**Why.** The README describes skillgrid as "an installer that sets up AI-agent tooling," but the binary now ships 20+ subcommands across install, sync, setup, doctor, serve/mcp, index/search/code-intel, mem/session/trail/eval. A PRD scoped to "the `skillgrid-cli/` directory" or "the installer" under-describes the product the code actually is. The PRD is a product description, not a module description: the user installs one thing (`skillgrid`) and gets the engine + surface + content together. Hub Content (skills/hooks) is repo content shipped *by* the installer, so it is covered in the Installer section, not as a peer component.
**Consequences.** Good: the PRD matches what a user receives; the 4-way component map is a clean reference for future changes. Bad: the PRD is larger than a binary-only doc and must stay honest about current state — a "Known gaps" section is mandatory, not optional.

### ADR-0002 — PRD framing is engine-first, not installer-first

**Decision.** The product is framed as "a local-first engine for agent memory + code intelligence, distributed as a CLI/MCP/HTTP server" — engine-first, not installer-first and not "opinionated pipeline"-first.
**Why.** The README's one-paragraph framing is installer-first. The code has drifted engine-first and the PRD should name the product for what it is, not for what the README says. Leading with "opinionated pipeline" makes the PRD sound like a methodology rather than a product. Engine-first keeps the user's primary pain (agent context doesn't survive sessions; "done" is a claim without evidence) as the through-line.
**Consequences.** Good: product vision, personas, and success metrics all derive from a single through-line, making the PRD internally consistent; the secondary persona falls out naturally from the MCP/HTTP surface. Bad: the README is now stale by construction; aligning it is a follow-up ticket (ADR-0001 scope).

### ADR-0003 — v1.0 line is "the engine is trustworthy"; UI is Phase 2

**Decision.** v1.0 is "core engine + installer solid": install idempotent on 3 agents, `doctor --strict` green, `mcp`/`serve` stable, index/search/code-intel reliable, mem/session/trail working; the UI is still stubs (explicitly "Phase 2").
**Why.** The PRD commits to a target state. The embedded dashboard is a shell with every page a `StubPage`; the tracker has 501s; the process-pass LLM labeler is a deterministic stub. Shipping those as "v1.0" would make the target-state claim false. "The engine is trustworthy" is verifiable with the tools that already exist; a v1.0 gate on UI or LLM labeling would either expand the PRD beyond "engine trustworthy" or lower the claim to "engine works, UI is decorative."
**Consequences.** Good: v1.0 is testable — each claim maps to an existing test or a fresh-machine install run; the roadmap is a clean "Phase 2+" list. Bad: users installing v1.0 see stub dashboard pages and must trust the engine underneath is the real product.

### ADR-0004 — PRD component map is a 4-way decomposition

**Decision.** The component map is 4 components: Installer, Mnemonic Engine (memory + code-intel + search), Distribution Surface (MCP stdio + HTTP/REST + embedded UI + CLI), Hub Content (skills + hooks + config.d, shipped as repo).
**Why.** The code has ~25 `internal/` packages, but those are implementation units, not product components. The Distribution Surface is a distinct concern (three transports, each with different auth, embedding, and lifecycle implications). Folding the UI into the engine (3-way) hides that the embedded SPA + OpenAPI/Swagger are a serving concern, not a data concern. Splitting memory from code-intel (5-way) over-decomposes: they share the same per-project SQLite store and the same project-resolution layer, and "memory + code intelligence" is one user experience.
**Consequences.** Good: each component has one clear purpose and a testable boundary; the trust-boundary section maps cleanly (the one real boundary lives between Surface and Engine). Bad: "Distribution Surface" is a term the code does not use; a glossary entry is required to keep it from drifting.

### ADR-0005 — Trust boundary: MCP-spawned process with project read access

**Decision.** The trust section states local-first (data stays in `~/.skillgrid/mnemonic/`, HTTP binds 127.0.0.1, no telemetry, no network except install-time npm/git and an optional external embedder) **plus** names the one real boundary: the agent process spawns the binary (`skillgrid mcp`), which then has full read access to the project it indexes.
**Why.** The MCP-spawn model is the one boundary that is (a) surprising without context (a local CLI that "just indexes code" is actually a process the agent spawns with project read access) and (b) hard to reverse (changing who spawns whom is an interface change across the Surface). The `doctor --strict` redaction guarantee is a `doctor` detail that belongs in the command reference, not the trust section.
**Consequences.** Good: the PRD names the one real boundary in one sentence, grounding the threat-matrix rows; the local-first posture is stated as a positive claim (what the product does NOT do). Bad: the redaction guarantee is in the command reference, not the trust section.

### ADR-0006 — Vector search: in-memory brute-force cosine, no SQLite vector extension

**Decision.** The semantic leg of hybrid code search ranks stored embeddings by cosine over an **in-memory vector cache** (`hybrid/vectorcache.go`), not a SQLite vector extension. No driver change, no new dependency, exact top-K.
**Why.** At the current ~20K-vector scale the bottleneck is BLOB I/O, not the search algorithm — 87% of the 220 ms is re-reading 55 MB of vectors out of SQLite per query. Caching the decoded vectors in RAM eliminates that cost with zero driver risk, zero new dependencies, and exact (not approximate) top-K ranking, which the RRF fusion in `hybrid/rank.go` is built around. The cgo-free invariant is preserved (no driver swap).
**Consequences.** Good: the semantic leg drops ~220 ms → ~28 ms cosine + ~55 ms FTS/signal; the cgo-free invariant is untouched; invalidation keys on the embedding model (already tracked in `embed_meta`). Bad: the cache is process-global (one slot) — correct for one project per MCP session, would need per-store keying for multi-project concurrent semantic search; the cold-cache build is a one-time ~7 s decode of 55 MB.
**Revisit (amended by ADR-0009).** When a single store crosses ~100K vectors, or cross-project (`all_projects`) semantic search lands, or the ncruces driver migration is done for an unrelated reason. Default revisit path is **option G** (`modernc.org/sqlite/vec`): same-module, cgo-free, a real `go.mod` dependency, transactional, exact top-K — only a `modernc.org/sqlite` bump to ≥ v1.59.0, not a driver swap. B′ (ncruces) is the fallback. sqlite-vss (E) and vectorlite (F) were evaluated and rejected.

### ADR-0007 — SDD multi-method docs viewer with generic, existence-gated doc roots

**Decision.** The Web Admin Dashboard Docs view declares its doc roots as `map[string][]string` (selector → candidate repo-relative paths) and walks only the directories that exist in the current repo. One binary serves Skillgrid, OpenSpec, SpecKit, Superpowers, and Backlog.md artifacts from whatever repo it runs in, with zero config.
**Why.** `skillgrid serve` is a single Go binary that must work from any repo it is started in. Each SDD method lives at a different canonical on-disk location, and not every repo uses every method. A single hard-coded root broke the moment the dashboard was pointed at a SpecKit or Superpowers repo; a per-repo config file degrades "works from anywhere" to "works once configured."
**Consequences.** Good: one binary serves all five methods with zero config; ADRs and PRDs get first-class schema-aware rendering (status lifecycle, Context/Decision/Consequences) with no backend schema migration; empty/absent methods degrade to a clean empty state. Tension: the root vocabulary is fixed (a novel method needs one line added to `mdRoots`); existence-gating means a repo mid-initialization shows nothing for that method until real files land. Reads stay sandboxed (`..`/absolute → 400, unknown → 404, read-only, rendered never executed).

### ADR-0008 — State drift guard: Node.js + `yaml` package, read-only verifier

**Decision.** The drift guard is `scripts/state-drift-check.mjs`, a read-only Node script that uses the `yaml` npm package (the repo's first npm dependency) to compare `state.yaml` against the spec zone and report a named drift verdict. It owns the verification, not the write (pipeline skills keep writing `state.yaml` directly).
**Why.** Nothing verifies that `state.yaml` (the resume pointer hand-edited by ten pipeline skills) agrees with the spec-zone artifacts it summarizes. A skill that forgets to update it produces a silent pointer that misleads resume. The repo had no YAML library; a real parser beats regex for a schema we own and will extend (`sdd-structure.md` is the source of truth and grows). The `yaml` package is zero-transitive, MIT, and works in both ESM and CJS. Option B (regex) was rejected as the lazy path; option C (Python) rejected as a larger footprint than one npm package.
**Consequences.** Good: robust to schema evolution; zero-transitive dependency. Bad: the repo now has its first npm dependency (`npm install` is a prerequisite; `node_modules` gitignored) — mitigated by the guard being invoked from `skillgrid:qa` and `skillgrid:resume`, both Node-equipped.
**Revisit.** When the repo adopts a Node test runner (port the fixtures), when the `state.yaml` schema grows beyond a flat read (revisit the derivation logic, not the parser), or when `yaml` ships a breaking major.

### ADR-0009 — Vector search: in-SQL sqlite-vec latency corrected; viant deferred; revisit path confirmed

**Decision (amends ADR-0006).** ADR-0006's current decision (in-memory option A at ~20K) is unchanged. The ~40 ms-at-100K figure in ADR-0006's revisit analysis is corrected to **~6.9 s median for the in-SQL G path** (the ~40 ms was in-memory Go cosine; the in-SQL `ORDER BY vec_distance_cosine` is ~170× slower). **viant is deferred, not rejected** (packaging bug + `ensureIndex` query deadlock). G remains the default revisit path.
**Why.** A comparison spike (`.skillgrid/spikes/001-cgo-free-vector-db`) tested the two cgo-free non-memory candidates head-to-head at 768-dim / 100K vectors. The spike produced two findings: (1) G's in-SQL top-K at 100K is ~6.9 s median, not ~40 ms; (2) all four released viant versions and `main` fail to build, and viant's lazy-index query path deadlocks (`ensureIndex` re-enters `db.Exec` from inside the vtab `Filter` callback, blocked by `SetMaxOpenConns(1)`).
**Consequences.** Good: the revisit path is grounded in a measured latency rather than an extrapolation; viant's deferral is evidence-based. Bad: G's in-SQL latency (~6.9 s at 100K) is far higher than the ~40 ms originally cited; the modernc v1.45.0 → v1.59.0 bump has not yet been validated against the existing 39 migrations, WAL-retry, and store pooling (a prerequisite for the real build). The design question at the threshold is *which path serves which query* (in-memory = hot path, in-SQL = durable path), not a single latency number.

### ADR-0010 — Verification scope discriminator: a zero is never a bare zero

**Decision.** A shared convention (`_shared/conventions/verification-scope.md`) defines `COMPLETE` / `TRUNCATED` / `UNSCOPED` / `UNREADABLE`; the two drift scripts emit a separate `SCOPE: <atom>` line after their `DRIFT:` verdict (worst-scope-wins); the QA gate names the scope of its own derivations in `report.md` and **fails closed on non-`COMPLETE` scope**.
**Why.** A count of zero is the most ambiguous output a verifier can produce: a real answer ("I looked at all of my input and there genuinely was nothing") and a non-answer ("I couldn't see all of my input, so the zero tells me nothing") are byte-identical. `DRIFT: none` from a guard whose base ref didn't exist read exactly the same as `DRIFT: none` from a guard that checked a clean tree. Verifiers must emit the scope where it is consumed (stdout, `report.md`), not in a human's head. A frozen enum beats a free-form message string because a consumer can branch on the atom.
**Consequences.** Good: a verifier that can't see its input now says so (`DRIFT: none, SCOPE: UNSCOPED` is a non-answer a consumer can act on); the gate fails closed on scope, so "done" can never rest on a check that didn't look at everything; exit codes are unchanged (scope is additive). Bad: the scope derivation is deliberately cheap (reflects what the script already read); a human reading the report must understand four atoms.
**Revisit.** When a drift script gains a partial-read path (then `TRUNCATED` becomes reachable and the derivation counts seen-vs-expected), when the Go CLI takes on scope-bearing verification (then a shared Go leaf enum is the right home), or when the gate needs more than four atoms.

### ADR-0011 — Observations are bi-temporal; the save path classifies each write as Add/Update/Delete/Noop

**Decision.** Add `valid_at`, `invalid_at`, `superseded_by` columns to `observations` (migration 043); extend the `DedupLLM` seam from a binary `(bool, int64)` to a 4-way `DedupDecision`; the `SaveWithAction` save path routes noop → `BumpDuplicate`, add → INSERT, update → topic-key upsert, delete → `MarkSuperseded` + INSERT new; the `mem_save` response gains `action` + `superseded_id`. The 7 primary read sites filter `invalid_at`.
**Why.** A memory system that only appends facts never answers "what was true when?" The codebase had the lifecycle flag (`status = 'superseded'`) and the forward pointer (`supersedes` edge), but neither was temporal, and there was no back-pointer or auto-supersede on save — supersession was manual. Two reference systems (mnemonic-ai/Rust, Mnemon/Go) both converged on the same fix. The bi-temporal columns make the supersede chain *queryable* (time-travel, "what was true at T"); the AUDN classifier makes it *automatic* (the save path produces it, the agent doesn't have to remember).
**Consequences.** Good: the agent can answer "what was true at time T?"; the save path produces the supersede chain automatically; the `action` field tells the agent what happened. Bad: the `DedupLLM` interface changes (binary → 4-way; existing test mocks break — mitigated by keeping `Dedup` as a deprecated wrapper); 7 read sites need the filter (a missed site is a test failure, caught by the "superseded observations do not appear in search results" acceptance test); 4 inline FTS SELECTs duplicate the column list (a missed one is a runtime column-count mismatch — the migration task lists all 5 SELECT sites).

### ADR-0012 — ASSUMPTIONS consolidation: PRD + ADRs + locked constraints fold into one root file

**Decision.** Consolidate `.skillgrid/artifacts/00-prd.md`, the 11 ADR files (`04-adr-*.md`), `03-adr-index.md`, and `05-locked-constraints.md` into a single live root file, `.skillgrid/ASSUMPTIONS.md`, with the VERIFIED / INFERRED / LOCKED / Open-Questions tiers. Architecture becomes a standalone live root `.skillgrid/ARCHITECTURE.md`. Spikes live in `.skillgrid/spikes/NNN-name/`. Blueprints become pure-technical (the product why lives in the spec `briefing.md` + this file). The 14 source files are `git mv`'d to `.skillgrid/archive/`.
**Why.** The PRD, the ADR set, and the locked constraints were four separate files that together answered "what is this project and what have we decided." They were read as a set, updated as a set, and their cross-references (index → records → constraints → AGENTS.md `### Rules`) added a generation of staleness every time one moved. One file with an in-force table and an IRON RULE (never delete an ADR entry; supersede by link + table flip) removes that seam and gives the AI one authoritative record of its own understanding plus its decisions. ADR-0012 records this consolidation itself, so the trail that "the ADRs used to be separate files" is preserved here.
**Consequences.** Good: one root file is the single source for the in-force ADR set, the locked constraints, and the verified/assumed product facts; the load-bearing re-points are `AGENTS.md` `### Rules` (rendered from `### Locked constraints`), `state.yaml constraints_ref`, and the skill re-points. Bad: the file is large; it is a consolidation, not a loss — the full ADR records (Context/Options/Consequences for the load-bearing ones) are inlined here, and the originals are archived, not deleted.

### ADR-0013 — Source of truth: the repo is authoritative; Backlog task ID ↔ commit SHA is the linkage

**Decision.** The **repo is the single source of truth** for project state: `.skillgrid/ASSUMPTIONS.md` (facts + decisions + locked constraints), the spec-zone artifacts under `.skillgrid/specs/`, the committed `state.yaml`, and the git history. When an external tracker, a session memory, or a dashboard disagrees with a committed artifact, **the committed artifact wins and the other source is reconciled to it.** Backlog.md tasks live under `.backlog/` in the repo, so the tracker is already in-repo — it is not an external system to defer to. The required linkage is **Backlog task ID ↔ commit SHA**: every commit that closes a task names that task ID in the conventional-commit subject or body (`[skillgrid-context]` block), so the audit chain `task → commit → diff` is recoverable from git alone.

**Why.** The Claude Academy AI-native SDLC playbook names three coexistence modes for projects with legacy trackers: repo-as-truth, legacy-as-truth, and linkage-as-minimum-bar. skillgrid was running repo-as-truth *by accident* — nothing had ever locked it, so a future reader could not tell whether Backlog.md or an in-session memory was authoritative, and the commit chain was the audit trail without anyone having said so. Locking it as an ADR makes the assumption explicit and gives the task↔commit linkage a named home, so the "is this done?" question is answered by reading git, not by remembering a conversation.

**Consequences.** Good: any reviewer, agent, or future session can reconstruct the full `task → decision → commit → diff` chain from the repo alone; the ADR set, the specs, and the commit history are mutually reconcilable; resume and reflect read from the same authoritative record. Bad: the in-session memory (`mem_*`) is explicitly *subordinate* — a saved observation that contradicts a committed artifact is stale, not new information, and must be updated, not treated as a correction. The commit-message linkage is now a load-bearing invariant: a merge that closes a task without naming its ID breaks the chain (the QA gate's commit-audit is where that would be caught).

**Revisit.** When a hosted Backlog/sync lands that would re-introduce a second authoritative tracker (then this becomes linkage-as-minimum-bar, not repo-as-truth), or when the `[skillgrid-context]` block gains a machine-checkable task-reference field (then the linkage moves from convention to schema).

### Locked constraints

These override per-change decisions and are the hard limits a change must respect. A constraint is locked only when the user says so — inferred limits belong in VERIFIED or an ADR entry, not here. The `### Rules` section of `AGENTS.md` is rendered from this list, one bullet per constraint. `state.yaml constraints_ref` points at this file.

- Go 1.22+ minimum to build.
- No new dependencies without an ADR.
- Trivy is advisory-only — findings are reported, never blocking the QA gate.
- Serial development: one change at a time, no parallel branches.
- Conventional commits only; no AI-attribution trailers (commit-msg hook enforces).
- Spec-zone changes commit before code-zone changes (pre-commit zone guard enforces).
- The repo is the source of truth (ADR-0013): when an external tracker, session memory, or dashboard disagrees with a committed artifact, the committed artifact wins. A commit that closes a Backlog task names that task ID in the `[skillgrid-context]` block so the `task → commit → diff` chain is recoverable from git alone.
- Session-inject uses a two-layer mechanism: (1) auto-prepend a slim token-capped L1 summary only on resume (not fresh sessions), and (2) an on-demand `mem_inject_session` tool for deeper BM25/semantic retrieval. See `.skillgrid/artifacts/07-mnemonic-tool-surface.md`.
- Session-inject privacy: tag-by-default — secrets, full local paths, and `private`-tagged content are auto-excluded from injection; project-relative paths, commit SHAs, tool names, and task IDs are included by default.
- Session-inject scope: project-scoped by default; cross-project injection only when the agent explicitly passes `all_projects: true` (reuses the existing `mem_search` flag).
- Session-inject selection is hybrid (BM25 + semantic, RRF-fused) by default. The vector leg degrades to BM25-only when no embedder is active (reuses the existing degrade-to-Null design); BM25-only is a degraded state, not the target model.

*Unlocked (historical):* (none yet). A constraint that is later unlocked gets moved here with the date — it is not deleted, so the boundary history stays readable.

### Locked assumptions

- The config's advisory-only posture (coverage_min 0, no hard thresholds) carries into `eval`: it reports a baseline, it does not gate. *(see INFERRED H2 — treat as an assumption, not a confirmed constraint, until a release changes it.)*
- "Local-first" means no network except install-time npm/git and an optional external embedder; a future hosted mode is a v2.0 exploration, not a v1.x commitment.
- The 30 skills + hooks are repo content shipped by the Installer; their *behavior* is out of scope for the PRD (covered in the user guide), but their *shipping* is in scope. *(see INFERRED H4.)*

---

## Open Questions

Questions that are not yet decided and not yet spiked.

1. **v1.1 dashboard prioritization** — Memories+Sessions-first vs. Graph+Tracker-first. Decided at v1.1 scoping, not now (see INFERRED H3).
2. **MCP tool-contract versioning** — is the secondary persona strong enough to warrant stable, versioned MCP tool contract (semver on the tool surface)? Assumed no in v1.0 (see INFERRED H1).
3. **README alignment** — the README is still installer-first; aligning it to the engine-first PRD is a follow-up ticket, not part of this consolidation.
4. **Housekeeping** — remove the stray `internal/mnemonic/mcp/.git` nested repo; the migration 032 numbering gap is a known issue with no action.
