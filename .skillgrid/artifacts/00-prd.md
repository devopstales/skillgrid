# Product Requirements: skillgrid (Hub Product)

Reference for the `skillgrid` Hub Product. The product statement, verified facts, and the in-force decision index are in `.skillgrid/ASSUMPTIONS.md`; this file holds the longer requirements. Scope and framing are ADR-0001 and ADR-0002; the v1.0 line is ADR-0003; the component map is ADR-0004; the trust boundary is ADR-0005. Where this file and `ASSUMPTIONS.md` § VERIFIED disagree, VERIFIED wins (it carries the later verification date). The 2026-09-16 draft this was rewritten from is at `.skillgrid/archive/00-prd.md`.

**Target users.** Primary: a solo developer or small team using AI coding agents, who wants the agent to remember decisions, skip re-deriving the codebase, and produce "done" with evidence. Secondary: an AI-tooling builder who points an agent at a local memory + code-intelligence backend. The secondary persona falls out of the MCP/HTTP surface; it is not a separate product (INFERRED H1).

**Business objectives.**
- Give AI agents a persistent, local memory + code-intelligence backend that survives sessions and compaction.
- Make "done" a claim with evidence: retrieval trails, blast radius, and a health/eval harness.
- Distribute the capability as a single binary plus a one-command install.

**Success metrics.**
- **Install reliability:** 100% fresh-machine install success across OpenCode, Kilo, and Cursor; idempotent re-run (0 drift on a second `skillgrid install`); `--dry-run` writes nothing.
- **Engine trustworthiness:** `doctor --strict` green on a fresh project and a real project; `eval` reports MRR/recall with a recorded baseline (advisory, no hard threshold); the staleness gate keeps the index within ~5s of file changes under watch.
- **Reliability:** no data loss across process restarts (SQLite WAL); `serve` recovers from a killed process; `mcp` reopens the store after the ~5s auto-reopen window.

**Personas.**

- **Solo dev with agents (primary).** Comfortable with CLIs and git; uses an agent daily in at least one repo. Goals: decisions and code orientation survive sessions; the agent orients before editing; "done" is backed by evidence. Pain: context fills up, every session re-derives the codebase, "done" is a claim, a memory service means standing up infrastructure. Journey: `skillgrid install` once; the agent calls Mnemonic and code-intel tools over MCP; `skillgrid doctor` / `skillgrid serve` to inspect. Data lives in `~/.skillgrid/mnemonic/`.
- **AI-tooling builder (secondary).** Points a harness at MCP or HTTP. Goals: no service to run, a stable MCP tool surface, an HTTP API. Pain: existing options are cloud services or require standing up a service. Journey: `skillgrid mcp` (stdio) or `skillgrid serve` (HTTP, 127.0.0.1:7438); OpenAPI/Swagger for the HTTP surface. The binary is a backend, not an installer.

**Feature requirements.** MoSCoW. **Must** = v1.0 gate (ADR-0003). **Should** = Phase 2+. **Could** = later. **Won't** = out of this release.

| Feature | Description | Priority | Acceptance |
|---------|-------------|----------|------------|
| Installer | `skillgrid install` wires the hub across the three agents | Must | Fresh-machine install succeeds for opencode, kilo, cursor; re-run is idempotent (0 drift); `--dry-run` writes nothing; `--skip-clone` / `--skip-tools` / `--skip-agents` each skip their step; timestamped backups of agent configs before modification |
| Repo sync | `skillgrid sync-repo PATH` copies a local clone into the hub home | Must | Copies the repo into `~/.skillgrid/repos/skillgrid` and `.agents/` → `~/.agents/` without npm/git |
| Agent setup | `skillgrid setup <agent>` installs one agent plugin | Must | `--agent`, `--repo-root`, `--dry-run` honored; idempotent |
| Mnemonic Engine — Memory | Observation lifecycle, hybrid/semantic search, provenance, governance, session relay | Must | `mem save/search/context/timeline/layers/governance/share` work; session handoff/resume/status work; retrieval trail is inspectable via `trail` |
| Mnemonic Engine — Code Intelligence | Incremental indexing, orientation, graph queries, hybrid search, blast radius, taint | Must | `index` + `index status`; `orient` / `grep` / `callers` / `callees` / `dependents` / `implementors` / `hierarchy` / `tests-for` / `path` / `explain` / `impact` / `explore` return confidence-labeled results; `search` returns per-signal provenance; `search affected` returns blast radius |
| Distribution Surface — MCP stdio | `skillgrid mcp` | Must | Registers mem/code/web/session/teams; fsnotify watcher + staleness gate; `--no-watch` disables the watcher |
| Distribution Surface — HTTP/REST | `skillgrid serve` plus embedded SPA and OpenAPI/Swagger | Must | Binds 127.0.0.1:7438 (or `SKILLGRID_MNEMONIC_PORT`); REST for memory/code/sessions/web/health/tracker; SPA at `/`; `/openapi.yaml` + Swagger UI |
| Distribution Surface — CLI | index/search/mem/session/trail/doctor/eval | Must | Commands work against a real project; `--json` where applicable |
| Health check | `doctor` and `--strict` | Must | Embed round-trip + capability checks; `--strict` adds redaction + freshness; exit code reflects health |
| Retrieval eval | `eval` ablation harness | Must | `--corpus self \| name=path`; leak-free git-history queries; bootstrap CIs + permutation tests; baseline recorded |
| UI dashboard pages | Embedded SPA pages | Should | 10 feature modules render real data (2026-09-29); `SettingsPage` still a stub; tracker without 501s on supported providers; per-widget error isolation; show-numbers table twin |
| `ui:build` task | Build the UI before the binary | Should | Landed: the Taskfile task exists and `build:all` depends on it. Residual: `go build` alone still requires a pre-built `ui/dist` |
| Tracker provider support | `/tracker/*` providers | Should | No 501 for a provider listed as supported in the OpenAPI |
| Process-pass LLM labeling | Live LLM labels instead of `processLLMStub` | Could | Live LLM configured → labels from the LLM; not configured → deterministic stub |
| README alignment | README matches the engine-first product | Should | Product paragraph and command surface match this file and `ASSUMPTIONS.md`; no "bare `skillgrid` = install" claim |
| Dashboard as a v1.0 gate | Shipping stub pages as the v1.0 claim | Won't | ADR-0003 |
| Cloud / hosted service | A hosted skillgrid | Won't | ADR-0005 |
| Telemetry | Analytics out of the box | Won't | ADR-0005 |

**User flows.**

1. **First install.** `skillgrid install`. The installer creates `~/.skillgrid/`, syncs the hub repo (`git clone --branch release/2`, or `--skip-clone`), requires `node` and `npm` on PATH, selects agents (interactive multiselect, `--agents`, or `--yes`; non-TTY falls back to a letter list), installs each agent's npm package when one exists, installs MCP packages from `config.d/tools.yaml` once, and is the sole writer of agent MCP config from `config.d/mcp.yaml` (upsert, plugin copy, memory-protocol marker, timestamped backups under `~/.skillgrid/backup/<agent>/`). Harness config (TUI logo/theme, plugin appends, personas) can be skipped with `--skip-agents`. Remaining global tools install, then `.agents/` copies to `~/.agents/`. A hard fail stops the flow with a non-zero exit; completed steps stay; re-run is idempotent.
2. **Agent uses the engine.** The harness spawns `skillgrid mcp`. Memory tools resolve the project from CWD and open the per-project store. Code-intel tools read the index or trigger an incremental index and return confidence-labeled results. The watcher keeps the index fresh; the staleness gate flags results whose indexed bytes no longer match the tree. An ambiguous project returns `AvailableProjects`. A missing embedder degrades the semantic leg to FTS + signals.
3. **Dev inspects the engine.** `skillgrid doctor` (or `--strict`) checks health, the embed round-trip, and capabilities. `skillgrid serve` opens the dashboard at 127.0.0.1:7438 (10 real modules; `SettingsPage` still a stub) or the OpenAPI/Swagger. `skillgrid eval --corpus self` measures retrieval on a leak-free git-history corpus. Port in use is a clear error. A store locked by another process retries on the WAL lock, then errors clearly.

**Non-functional requirements.**
- **Performance.** Under watch, indexed bytes stay within ~5s of a file change. `skillgrid mcp` serves the first tool call with no network dependency; a warming embedder degrades the semantic leg. Store growth is bounded by TTL soft-expiry and distill/dream consolidation. Concurrent MCP/HTTP requests share a refcounted `ProjectHandle` per project ID.
- **Security.** HTTP writes require `SKILLGRID_HTTP_TOKEN` (`requireWriteAuth`); reads stay open. MCP stdio is trusted because the harness spawned it. Local single-user; no multi-tenant auth. Data stays in `~/.skillgrid/mnemonic/`. No telemetry. Network only for install-time npm/git and an optional external embedder. `doctor --strict` checks that health output does not leak secrets.
- **Compatibility.** linux amd64+386, darwin amd64+arm64 (`task all`). Install needs Node, npm, and git. Runtime is the binary (pure-Go SQLite, in-process ONNX). Agents: OpenCode (`opencode-ai`), Kilo (`@kilocode/cli`), Cursor (app-side, no npm).
- **Accessibility.** The dashboard targets WCAG 2.1 AA. The CLI is TTY-first, with a letter-list fallback for non-TTY agent selection.

**Embedded dashboard.** Vite + React 19 + TypeScript + Tailwind 4 + TanStack Router (`skillgrid-ui/`), built into `skillgrid-cli/internal/mnemonic/http/ui/dist` and embedded with `//go:embed`. Patterns: per-widget error isolation; a show-numbers table twin whose raw JSON/table is the source of truth. Page status is the known-gaps bullet in `.skillgrid/ASSUMPTIONS.md` § VERIFIED.

**Local metrics (not telemetry).** Retrieval quality (`eval`), index freshness (staleness gate), store health (`doctor`), retrieval-trail volume (`trail`). Trails record the query, directories traversed, files read, and result path. Session handoffs are recorded. Alerting is `doctor --strict` as the CI health signal.

**Release line.**
- **v1.0 — the engine is trustworthy (ADR-0003).** Installer, repo sync, agent setup, Mnemonic Engine, Distribution Surface, `doctor`/`--strict`, `eval`. The dashboard is not a v1.0 gate. Success criteria are the success metrics in this file. No ship date is set here.
- **v1.1.** Finish the dashboard: `SettingsPage`, tracker without 501s on supported providers, per-widget error isolation and the show-numbers twin. Prioritization of remaining pages is Open Question 1.
- **v1.2.** Process-pass LLM labeling (stub → live LLM).
- **v2.0.** README alignment is a follow-up, not a version of its own (Open Question 3). A hosted mode is an exploration, not a v1.x commitment.

**Competitive position.** Cloud agent-memory services (Mem0, Zep, Letta) lead on managed recall. skillgrid's edge is local-first plus code intelligence on the same store. Local code-intel tools lead on orientation. skillgrid's edge is persistent memory plus MCP/CLI/HTTP in one binary. Installer-only tools set up an environment. skillgrid's installer is how the engine gets onto the machine. The through-line pain, from the user guide and the code inventory, is that agent context does not survive sessions and "done" is a claim without evidence (ADR-0002).
