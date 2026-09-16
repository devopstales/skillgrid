# Product Requirements Document: skillgrid (Hub Product)

> Product requirements for the `skillgrid` Hub Product — the local-first
> engine for agent memory + code intelligence, distributed as a CLI / MCP /
> HTTP server. Written 2026-09-16 from the brainstorming PRD template.
> Scope and framing decisions are recorded in ADR-0001 and ADR-0002; the v1.0
> line in ADR-0003; the component map in ADR-0004; the trust boundary in
> ADR-0005.

## Product Overview

**Product Vision:** skillgrid is a local-first engine for agent memory + code
intelligence, distributed as a CLI / MCP / HTTP server. It persists decisions,
observations, and code context outside the chat so AI coding agents
(OpenCode, Kilo, Cursor) keep working across sessions, and it indexes a
project's code so the agent can orient, search, and assess blast radius before
it edits. Everything runs on the developer's machine — no cloud, no telemetry.

**Target Users:**
- **Primary — solo developer / small team using AI coding agents.** The
  developer runs an agent in a repo and wants the agent to remember decisions,
  not re-derive the codebase every session, and to produce "done" with
  evidence.
- **Secondary — AI-tooling builder who embeds the capability.** A builder who
  wants a local memory + code-intelligence backend to point an agent at,
  without running a service. This persona falls out of the MCP/HTTP surface;
  it is not a separate product.

**Business Objectives:**
- Give AI agents a persistent, local memory + code-intelligence backend that
  survives sessions and survives compaction.
- Make "done" a claim with evidence: the engine exposes retrieval trails,
  blast radius, and a health/eval harness so the agent (and the developer) can
  verify.
- Distribute the capability as a single binary + a one-command install, so
  adoption is "run one command," not "stand up a service."

**Success Metrics:** (see also Release Planning — MVP)
- **Install reliability:** 100% fresh-machine install success across the three
  agents (OpenCode, Kilo, Cursor); idempotent re-run (0 drift on a second
  `skillgrid install`); `--dry-run` writes nothing.
- **Engine trustworthiness:** `doctor --strict` green on a fresh project and a
  real project; `eval` retrieval harness reports MRR/recall with a recorded
  baseline (advisory, no hard threshold — matching the config's
  advisory-only posture); index freshness: the staleness gate keeps the index
  within ~5s of file changes under watch.
- **Reliability:** no data loss across process restarts (SQLite WAL); `serve`
  recovers from a killed process; `mcp` reopens the store after the ~5s
  auto-reopen window.

## User Personas

### Persona 1: The Solo Dev with Agents (primary)
- **Demographics:** Experienced developer, comfortable with CLIs and git, uses
  an AI coding agent daily in at least one repo.
- **Goals:** The agent remembers project decisions and code orientation
  across sessions; the agent can orient in a large codebase before editing;
  "done" is backed by evidence, not a claim.
- **Pain Points:** Agent context fills up and intent drifts on long-horizon
  work; every new session re-derives the codebase; "done" is a claim without
  evidence; running a memory service means standing up infrastructure.
- **User Journey:** Run `skillgrid install` once (picks agents, wires MCP).
  In the agent, the agent calls Mnemonic tools (save/search/recall) and
  code-intel tools (orient/search/impact) via MCP. When curious, the dev runs
  `skillgrid doctor` / `skillgrid serve` to inspect. The data lives in
  `~/.skillgrid/mnemonic/`.

### Persona 2: The AI-Tooling Builder (secondary)
- **Demographics:** Builder of agent tooling or a custom agent harness;
  comfortable pointing an agent at an MCP server or HTTP API.
- **Goals:** A local memory + code-intelligence backend with no service to
  run; a stable MCP tool surface; an HTTP API for programmatic access.
- **Pain Points:** Existing memory/code-intel options are cloud services or
  require standing up a service; they want local-first with a clean contract.
- **User Journey:** Runs `skillgrid mcp` (stdio) or `skillgrid serve` (HTTP,
  127.0.0.1:7438) and registers the endpoint with their agent. Uses the
  OpenAPI/Swagger to wire the HTTP surface. Treats the binary as a library-
  quality backend, not an installer.

## Feature Requirements

Prioritized MoSCoW. **Must** = v1.0 gate (ADR-0003); **Should** = Phase 2+;
**Could** = later; **Won't** = explicitly out this release.

| Feature | Description | User Stories | Priority | Acceptance Criteria | Dependencies |
|---------|-------------|--------------|----------|---------------------|--------------|
| **Installer** | `skillgrid install` sets up the hub on a machine across the three agents | As a solo dev, I run one command and my agent gets the engine + skills + hooks wired | Must | Fresh-machine install succeeds for opencode, kilo, cursor; re-run is idempotent (0 drift); `--dry-run` writes nothing; `--skip-clone`/`--skip-tools`/`--skip-agents` each skip their step; timestamped backups of agent configs before modification | Node/npm on PATH at install time; git available |
| **Repo sync** | `skillgrid sync-repo PATH` copies a local clone into the hub home | As a dev iterating on the hub, I sync a local path instead of cloning | Must | Copies repo into `~/.skillgrid/repos/skillgrid` + `.agents/` → `~/.agents/` without npm/git | — |
| **Agent setup** | `skillgrid setup <agent>` standalone agent plugin install | As a dev, I install the agent plugin for one agent without a full install | Must | `--agent`, `--repo-root`, `--dry-run` honored; idempotent | Installer registry (`config.d/tools.yaml`, `config.d/mcp.yaml`) |
| **Mnemonic Engine — Memory** | Observation lifecycle, hybrid/semantic search, provenance, governance, session relay | As an agent, I persist a decision and recall it in a later session; as a dev, I inspect what was recalled and why | Must | `mem save/search/context/timeline/layers/governance/share` work; `session handoff/resume/status` work; retrieval trail is persisted and inspectable via `trail` | Per-project SQLite store |
| **Mnemonic Engine — Code Intelligence** | Incremental indexing, orientation, graph queries, hybrid search, blast radius, taint | As an agent, I orient in a codebase, search it, and assess blast radius before editing | Must | `index` + `index status` work; `orient`/`grep`/`callers`/`callees`/`dependents`/`implementors`/`hierarchy`/`tests-for`/`path`/`explain`/`impact`/`explore` return confidence-labeled results; `search` (FTS+signals+semantic) returns per-signal provenance; `search affected` returns blast radius of a changed set | tree-sitter; ONNX embedder (or off) |
| **Distribution Surface — MCP stdio** | `skillgrid mcp` exposes the engine as a stdio MCP server to an agent | As an agent, I call the engine's tools over MCP | Must | Server starts, registers tool registries (mem/code/web/session/teams), fsnotify auto-sync watcher + staleness gate active; `--no-watch` disables watcher | Engine |
| **Distribution Surface — HTTP/REST** | `skillgrid serve` exposes the engine over HTTP + embedded SPA + OpenAPI/Swagger | As a dev, I inspect the engine's state and API in a browser | Must | Binds 127.0.0.1:7438 (or `SKILLGRID_MNEMONIC_PORT`); REST routes for memory/code/sessions/web/health/tracker; embedded SPA served at `/`; `/openapi.yaml` + Swagger UI present | Engine; built `ui/dist` |
| **Distribution Surface — CLI** | The command group is itself a surface (index/search/mem/session/trail/doctor/eval) | As a dev, I drive the engine directly from the shell | Must | Every command in the command reference works against a real project; `--json` where applicable | Engine |
| **Health check** | `doctor` (and `--strict`) functional health check | As a dev, I verify the engine is healthy before relying on it | Must | `doctor` runs embed round-trip + capability checks; `--strict` adds redaction + freshness checks for CI; exit code reflects health | Engine |
| **Retrieval eval** | `eval` retrieval-eval ablation harness | As a dev, I measure retrieval quality with a leak-free corpus | Must | `--corpus self \| name=path`; leak-free git-history queries; bootstrap CIs + permutation tests; baseline recorded | git; engine |
| **UI dashboard pages** | Real implementations of the embedded SPA pages (Memories/Sessions/Graph/Tracker/Docs/…) | As a dev, I inspect the engine's state in a working dashboard | Should | Memories + Sessions pages render real data; tracker functional (no 501s); per-widget error isolation; show-numbers table twin | `serve` surface; `ui:build` task |
| **Process-pass LLM labeling** | Replace the deterministic stub with a live LLM for process-flow labels | As an agent, I get meaningful process-flow labels | Could | Live LLM configured → labels from LLM; not configured → deterministic stub (current behavior) | engine; LLM endpoint |
| **`ui:build` task** | Taskfile task to build the UI before `go build` (currently missing) | As a dev, `task all` builds the UI then the binary | Should | `ui/dist` produced by the task; `go build` no longer requires a pre-built dist | Taskfile; Vite |
| **Tracker provider support** | Resolve the 501s on unsupported `/tracker/*` providers | As a dev, every supported tracker provider returns data, not 501 | Should | No 501 for a provider listed as supported in the OpenAPI | engine; provider backends |
| **README alignment** | Update README to match this PRD (kill the stale installer-first framing) | As a new user, the README describes the product this PRD defines | Should | README product paragraph + command surface match the PRD; no stale "bare `skillgrid` = install" claim | PRD approval |
| **Dashboard as v1.0 gate** | Shipping stub dashboard pages as part of v1.0 | — | Won't | — | ADR-0003 |
| **Cloud / hosted service** | A hosted skillgrid service | — | Won't | — | local-first posture (ADR-0005) |
| **Telemetry** | Any telemetry / analytics out of the box | — | Won't | — | local-first posture (ADR-0005) |

## User Flows

### Flow 1: First install (solo dev)
1. Dev runs `skillgrid install` (or `skillgrid` with no args → usage; `install` to run).
2. Installer creates `~/.skillgrid/`, syncs the hub repo (`git clone --branch release/2`, or `--skip-clone`).
3. Verifies `node` + `npm` on PATH; hard fail otherwise (points to `scripts/install_node.sh`).
4. Selects agents: interactive multiselect, `--agents`, or `--yes` (default opencode,kilo); non-TTY falls back to a letter list.
5. For each selected agent with an npm package: `npm install -g` (skip if binary already on PATH).
6. Installs the MCP packages from `config.d/tools.yaml` once (not per agent).
7. **Memory setup** (sole writer of agent MCP config, from `config.d/mcp.yaml`): upserts each MCP entry into the agent config (opencode.jsonc / kilo.jsonc / cursor mcp.json), copies the agent plugin, renders the memory-protocol marker block. Timestamped backups to `~/.skillgrid/backup/<agent>/` before each modification.
8. **Harness config:** TUI logo/theme, plugin[] appends, per-agent personas (skip with `--skip-agents`).
9. Installs remaining global tools (`skills`, `@cucumber/cucumber`, `backlog.md`).
10. Copies the repo `.agents/` → `~/.agents/` (recursive overwrite).
    - **Error state:** any hard-fail (no node/npm, npm install failure, config write failure) stops the flow with a non-zero exit; already-completed steps are left in place (re-run is idempotent).

### Flow 2: Agent uses the engine (steady state)
1. Agent starts in a project; the MCP server (`skillgrid mcp`) is spawned by the agent harness.
2. Agent calls memory tools (`mem_save`, `mem_search`, `mem_context`) — the engine resolves the project from CWD, opens the per-project store, and returns.
3. Agent calls code-intel tools (`orient`, `search`, `impact`, `explore`) — the engine reads the index (or triggers an incremental index) and returns confidence-labeled results.
4. The fsnotify watcher keeps the index fresh under watch; the staleness gate flags results whose indexed bytes no longer match the working tree.
    - **Error state:** ambiguous project (CWD parents multiple repos) → `AvailableProjects` returned, not a silent fallback; embedder not ready → semantic leg degrades to FTS + signals, never a hard fail.

### Flow 3: Dev inspects the engine
1. Dev runs `skillgrid doctor` (or `--strict` for CI) — health, embed round-trip, capabilities; redaction checked under `--strict`.
2. Dev runs `skillgrid serve` — opens the embedded dashboard at 127.0.0.1:7438 (stub pages in v1.0; real pages in Phase 2) or hits the OpenAPI/Swagger.
3. Dev runs `skillgrid eval --corpus self` — measures retrieval quality with a leak-free git-history corpus.
    - **Error state:** port in use → clear error with the port; store locked by another process → WAL-lock retry backoff, then a clear error.

## Non-Functional Requirements

### Performance
- **Index freshness:** under watch, the staleness gate keeps indexed bytes within ~5s of a file change (the auto-reopen window).
- **Startup:** `skillgrid mcp` starts and serves the first tool call without a network dependency (embedder may be warming; semantic leg degrades, not fails).
- **Storage:** per-project SQLite (WAL) under `~/.skillgrid/mnemonic/`; pure-Go driver (no CGo); no unbounded growth — TTL soft-expiry + distill/dream consolidation bound the store.
- **Concurrency:** store pooling (refcounted `ProjectHandle` by project ID) eliminates N+1 SQLite opens across concurrent MCP/HTTP requests.

### Security
- **Authentication:** HTTP write routes are protected by the `SKILLGRID_HTTP_TOKEN` bearer check (`requireWriteAuth`); read routes stay open. The MCP stdio server is trusted by construction (spawned by the agent harness).
- **Authorization:** local single-user model; no multi-tenant auth.
- **Data protection:** local-first — all data stays in `~/.skillgrid/mnemonic/` (SQLite, WAL). No telemetry. No network calls except install-time npm/git and an optional external embedder endpoint. `doctor --strict` checks that health output does not leak secrets.

### Compatibility
- **OS/arch:** linux amd64+386, darwin amd64+arm64 (cross-compile matrix via `task all`).
- **Runtime deps at install:** Node + npm on PATH; git available. Runtime deps: none beyond the binary (SQLite is pure-Go; ONNX inference is in-process).
- **Agents:** OpenCode (`opencode-ai`), Kilo (`@kilocode/cli`), Cursor (app-side, no npm).

### Accessibility
- The embedded dashboard (Phase 2) targets WCAG 2.1 AA; the CLI is TTY-first with a plain letter-list fallback for non-TTY agent selection.

## Technical Specifications

### Component map (ADR-0004)

Four product components:

| Component | Responsibility | Boundary |
|-----------|----------------|----------|
| **Installer** | `install` / `sync-repo` / `setup`; wires agents, MCP, Hub Content | Writes to `~/.skillgrid/`, `~/.agents/`, and agent config files; the only writer of agent MCP config |
| **Mnemonic Engine** | Memory (observations, search, provenance, governance, session relay) + Code Intelligence (indexing, orientation, graph, hybrid search, taint) + shared project resolution | Owns the per-project SQLite store; no transport concerns |
| **Distribution Surface** | MCP stdio (`mcp`), HTTP/REST + embedded SPA + OpenAPI/Swagger (`serve`), and the CLI command group | Owns the transports; no data ownership; the one real trust boundary lives here (MCP-spawn model, ADR-0005) |
| **Hub Content** | `.agents/skills/` (30 skills), `.agents/hooks/` + `.agents/git-hooks/`, `config.d/` | Shipped by the Installer; not a runtime component of the binary |

### Frontend (embedded dashboard — Phase 2)
- **Technology Stack:** Vite + React 19 + TypeScript + Tailwind 4 + TanStack Router (`skillgrid-ui/`).
- **Build:** built into `skillgrid-cli/internal/mnemonic/http/ui/dist`, embedded via `//go:embed` (the `ui:build` Taskfile task is a Phase 2 item — currently missing).
- **Pages:** Memories, Sessions, Graph, Tracker, Docs, Activity, Plans, Settings, Git, Prototypes — all stubs in v1.0.
- **Patterns:** per-widget error isolation; show-numbers table twin (raw JSON/table is the source of truth).

### Backend
- **Technology Stack:** Go 1.26 (module `github.com/devopstales/skillgrid/skillgrid-cli`); pure-Go SQLite (`modernc.org/sqlite`); tree-sitter (`odvcencio/gotreesitter`); ONNX inference (`benedoc-inc/onnxer`, nomic-embed-code default); MCP server (`mark3labs/mcp-go`); Leiden community detection (`bluuewhale/loom`); fsnotify watcher; `charmbracelet/huh` for the interactive multiselect.
- **API Requirements:** HTTP/REST with OpenAPI + Swagger; MCP stdio tool registries (mem/code/web/session/teams); CLI command group.
- **Database:** per-project SQLite (WAL), embedded migrations (001→038), store pooling, TTL soft-expiry, distill/dream consolidation.

### Infrastructure
- **Hosting:** local-first; no hosting requirement. The binary is the deployment unit.
- **Scaling:** single-machine, single-user; store pooling handles concurrent MCP/HTTP requests.
- **CI/CD:** cross-compile via `task all` (linux amd64+386, darwin amd64+arm64); version via `-ldflags`; `hub-sync-check` CI verifies IDE-asset sync; `doctor --strict` as a CI health gate. Upgrade story: re-run `skillgit install` = `git pull --ff-only` + re-wire (idempotent).

## Analytics & Monitoring

- **Key Metrics (local, not telemetry):** retrieval quality (MRR/recall from `eval`), index freshness (staleness gate), store health (`doctor`), retrieval trail volume (`trail`).
- **Events:** retrieval trails (query, directories traversed, files read, result path) are persisted and inspectable; session handoffs are recorded.
- **Dashboards:** the embedded dashboard (Phase 2) renders memory/sessions/code/tracker state; in v1.0 the CLI + OpenAPI are the inspection surfaces.
- **Alerting:** none (local-first, no telemetry); `doctor --strict` is the CI health signal.

## Release Planning

### MVP (v1.0) — "the engine is trustworthy" (ADR-0003)
- **Features:** Installer (idempotent, 3 agents), Repo sync, Agent setup, Mnemonic Engine (Memory + Code Intelligence), Distribution Surface (MCP stdio + HTTP/REST + CLI), Health check (`doctor`/`--strict`), Retrieval eval (`eval`). The UI dashboard is explicitly **not** a v1.0 gate — it ships as stubs.
- **Timeline:** the current code is at or near the v1.0 line; v1.0 is "the engine is trustworthy, verified by the existing test harness + a fresh-machine install run." No new date invented.
- **Success Criteria:** Install reliability (100% fresh-machine success across 3 agents, idempotent re-run, `--dry-run` safe); Engine trustworthiness (`doctor --strict` green, `eval` baseline recorded, index freshness under watch); Reliability (no data loss across restarts, `serve` recovers, `mcp` reopens).

### Future Releases
- **v1.1 (Phase 2 — the dashboard):** real Memories + Sessions pages, tracker functional (no 501s), `ui:build` task wired, per-widget error isolation + show-numbers table twin.
- **v1.2:** process-pass LLM labeling (deterministic stub → live LLM); Graph + Docs + Activity pages real.
- **v2.0:** README alignment as a first-class artifact; possible multi-tenant / hosted exploration (currently Won't).

### Known gaps (current state, committed as roadmap)
- **UI dashboard pages** are all `StubPage` ("Coming in Phase N") — Phase 2 (v1.1).
- **`ui:build` Taskfile task** is missing; `go build` requires a pre-built `ui/dist` — Phase 2 (v1.1).
- **Tracker provider 501s** on some `/tracker/*` providers — Phase 2 (v1.1).
- **Process-pass LLM labeler** is a deterministic stub (`processLLMStub`) — Phase 2 (v1.2).
- **README** is stale (installer-first framing, smaller command surface, "bare `skillgrid` = install") — follow-up ticket after PRD approval.
- **Housekeeping:** stray `internal/mnemonic/mcp/.git` nested repo (remove); migration 032 numbering gap (known issue, no action).

## Open Questions & Assumptions

- **Question 1:** Should the v1.1 dashboard prioritization (Memories+Sessions first vs. Graph+Tracker first) be fixed now or decided when the dashboard work starts? — Assumed: decided at v1.1 scoping, not in this PRD.
- **Question 2:** Is the secondary persona (AI-tooling builder) strong enough to warrant a stable, versioned MCP tool contract (semver on the tool surface)? — Assumed: no explicit contract versioning in v1.0; the OpenAPI is the de-facto contract for the HTTP surface.
- **Assumption 1:** The config's advisory-only posture (coverage_min 0, no hard thresholds) carries into the PRD: `eval` reports a baseline, it does not gate.
- **Assumption 2:** "Local-first" means no network except install-time npm/git and an optional external embedder; a future hosted mode is a v2.0 exploration, not a v1.x commitment.
- **Assumption 3:** The 30 skills + hooks are repo content shipped by the Installer; their *behavior* is out of scope for this PRD (covered in the user guide), but their *shipping* is in scope (the Installer section).

## Appendix

### Competitive Analysis
- **Cloud agent-memory services (e.g., Mem0, Zep, Letta):** Strong recall + managed infrastructure; weak on local-first, no cloud dependency, no code-intelligence coupling. skillgrid's edge is local-first + the code-intel/memory coupling on one store.
- **Local-first code-intel tools (e.g., codegraph, CocoIndex-Code, GitNexus):** Strong code orientation; weak on persistent memory + the agent-tooling distribution (MCP/CLI/HTTP as one binary). skillgrid's edge is the unified engine + one-command install.
- **Installer-only tools (e.g., devbox, mise):** Strong environment setup; not a memory/code-intel product at all. skillgrid's installer is a means to the engine, not the product.

### User Research Findings
- **Finding 1 (from the user guide + config context):** the through-line pain is "agent context doesn't survive sessions, and 'done' is a claim without evidence." This PRD's product vision, personas, and success metrics all derive from it (ADR-0002).
- **Finding 2 (from the code inventory):** the code has drifted engine-first past the README's installer-first framing; the PRD must name the product for what it is, and the README alignment is a follow-up (ADR-0001, ADR-0002).

### Glossary
- See `.skillgrid/glossary/business.md` (Hub Product, v1.0 Line) and
  `.skillgrid/glossary/technical.md` (Mnemonic Engine, Distribution Surface,
  Hub Content) for the terms this PRD uses in a specific sense.
