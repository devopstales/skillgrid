# Serve — the web dashboard

`skillgrid serve` starts the Mnemonic HTTP API **and** a single-page web
dashboard that reads the local Mnemonic store, the repo's code graph, the
SDD/plans ledger, and git. It is the human-readable companion to the MCP
tools: everything you can ask an agent for, you can also look at in a browser.

## Quick path

```bash
skillgrid serve                 # http://127.0.0.1:7438  (default data dir)
skillgrid serve --port 9000     # choose a port
skillgrid serve --dir ~/.skillgrid/mnemonic   # explicit data dir
# env overrides: SKILLGRID_MNEMONIC_PORT, SKILLGRID_MNEMONIC_DATA_DIR
```

Open the URL, then pick a **project** from the top-right selector — every
Mnemonic view is scoped to the selected project (stored in the browser's
`localStorage`, so your choice persists across reloads).

Flags:

| Flag | Default | Env override | Meaning |
|------|---------|--------------|---------|
| `--port` | `7438` | `SKILLGRID_MNEMONIC_PORT` | listen port (bind is `127.0.0.1`) |
| `--dir` | `~/.skillgraph/mnemonic` → `~/.skillgrid/mnemonic` | `SKILLGRID_MNEMONIC_DATA_DIR` | the Mnemonic data directory |

The API and the dashboard are served from the same process; the dashboard is
the SPA shell (`text/html` for a browser, JSON for an API client) and all data
comes from read-only endpoints. **No writes to the store from the UI**, except
the few memory governance actions (pin/share/status) that map 1:1 to MCP tools.

---

## The views

The sidebar groups the views. Most are read-only; each maps to the same
endpoint an agent would call over MCP/HTTP.

### Tracker

The Kanban board of the active tracker provider's tasks (columns by status,
drag to move). This is the **provider-dependent** view — see
[Tracker provider CLI](#tracker-provider-cli) for install/auth and the
degraded state when a provider CLI is missing.

### Mnemonic

- **Graph** — the code-graph view (Sigma.js webgl): nodes are code symbols,
  edges are references. Progressive ForceAtlas2 layout (with a Stop button),
  zoom/fit controls, a file-tree explorer (left) that filters/dims by file or
  owning agent, search (name/path/semantic hybrid), and a node panel
  (callers/callees, source, impact, tests-for).
- **Files** — the indexed-file tree with content read.
- **Memories** — the observation list; supports pin, share, and status actions
  (the only writes).
- **Sessions** — the session list with summaries.
- **Search** — cross-store search (memory + code + web cache).

### Docs

Browse and render the repo's documentation (the shared core document tree).
Mermaid diagrams and KaTeX formulas render inline.

### Plans

The SDD/plans ledger (`.skillgrid/specs/` + `.skillgrid/sdd/`): one card per
change with status + task-progress bar, a dependency pipeline, a detail view
(SDD ledger steps, file inventory), and a spec viewer (renders `briefing.md` /
`tasks.md` / `findings.md`). Read-only — it reflects what the SDD skills wrote.

### Activity

The live agent-activity feed (observations as events): a stats bar, a
filterable feed (type/source/severity/actor), agent-health liveness, and
dismissible high-severity alerts. Live via SSE (`/activity/stream`) — new
events animate in from the top.

### Git

A read-only git bridge: the commit log (with +/- stats), commit detail +
unified diff, per-file history, and per-line blame. Read-only (`git` CLI, no
writes). Returns `503` when the served directory is not a git worktree.

### Prototypes

A sandboxed gallery of `.stitch/` design prototypes: a gallery list, a
sandboxed iframe preview (device viewport: desktop/tablet/mobile), a code
viewer (copy / download the standalone HTML), and export. Prototypes are
sandboxed to `.stitch/` (path-traversal-guarded) and the iframe runs with a
`sandbox` attribute so prototype scripts can't touch the parent.

### Settings

Global UI settings — the **density** toggle (comfortable / compact) is
persisted in the browser and changes list/row spacing across the app.

---

## Tracker provider CLI

The **Tracker** view is the only view that depends on an *external* CLI.
Skillgrid does not ship a tracker backend; it shells out to a per-provider
CLI to list/move tasks. The provider is configured in
`.skillgrid/config.yaml` (`tracker.provider`).

### Install + auth

| Provider | CLI | Install | Auth |
|----------|-----|---------|------|
| `backlog` | local (built-in) | none (ships with `skillgrid`) | none — reads the local `Backlog.md` |
| `github` | `gh` | `brew install gh` / `go install gh` | `gh auth login` |
| `linear` | `linear` (or API) | provider-documented | provider token in env |

When `tracker.provider` is `backlog` (the default), no external CLI is needed —
the board reads the local `Backlog.md` directly.

### Degraded state

If the configured provider's CLI is **missing or unauthenticated**, the
Tracker view degrades gracefully — it does not break the rest of the dashboard:

- The board shows an **amber banner**: the provider name, what's missing
  (CLI not on `PATH` / not authenticated), and the exact command to fix it
  (e.g. `gh auth login`).
- The other views (Graph, Memories, Sessions, Docs, Plans, Activity, Git,
  Prototypes) are **unaffected** — they don't touch the tracker CLI.
- `GET /tracker/providers` reports each provider's availability so the UI can
  render the banner; `GET /tracker/config` reports the active provider.

Fix the dependency (install/auth the CLI), then reload — the board populates
from the provider.

---

## API + OpenAPI

- `GET /openapi.yaml` — the full OpenAPI spec (every route, with examples).
- `GET /swagger-ui/` — interactive Swagger UI to exercise each endpoint.
- `GET /health` — liveness probe.

The spec is embedded in the binary and re-served at runtime; the
`TestPhase*_OpenAPI` tests assert the routes stay documented.

---

## Security notes

- Binds `127.0.0.1` (loopback) by default — not exposed to the network.
- All Mnemonic/code/git/prototypes reads are **read-only**; the only writes are
  the explicit memory-governance actions (pin/share/status), which map 1:1 to
  MCP tools and respect the same ownership/visibility rules.
- `.stitch/` prototype serving is sandboxed (path-traversal guard + nosniff);
  the preview iframe uses the `sandbox` attribute (no `allow-same-origin`).
- No CDN: the dashboard is a self-contained bundle (content-hashed assets,
  per-feature code-splitting); the bundle-size budget is gated in CI
  (`npm run build:check`).

---

## Next step

- [Memory and indexing](./05-memory-and-indexing.md) — the store the dashboard reads.
- [Ticketing](./07-ticketing.md) — the tracker providers in depth.
