# Web UI

Mnemonic ships an embedded **admin dashboard** and Swagger UI with `skillgrid serve`. The dashboard is a six-entry menu over this machine's Mnemonic data:

| Entry | What it shows |
|-------|---------------|
| Welcome | Roadmap + links to the API docs |
| Tracker | Issues/tickets from the configured issue tracker (see the CLI note below) |
| Docs | SDD change list + per-change detail (change.md, tasks.md) |
| Memory | Observations: search, detail pane, 013 governance (pin, share, status, edit), web cache |
| Code | Code-index health, BM25 search, source view, re-index |
| Sessions | Workspace session list (title, started_at, status), recent context, per-session summaries |

## Quick path

```bash
skillgrid serve
# open http://127.0.0.1:7438/
```

| URL | What |
|-----|------|
| `/` | Dashboard shell (Welcome entry) |
| `/tracker` · `/docs` · `/memory` · `/code` · `/sessions` | Path-routed menu entries (same shell) |
| `/swagger-ui` | Interactive OpenAPI explorer |
| `/openapi.yaml` | OpenAPI spec |
| `/health` | Liveness JSON |

Defaults: bind `127.0.0.1`, port `7438`. Override with `--port` / `--bind` or `SKILLGRID_MNEMONIC_PORT`.

```bash
skillgrid serve --port 7438 --bind 127.0.0.1
```

## What you can do

- Browse projects and pick the store the data entries read from (project selector, top bar)
- Track issues in the Tracker entry (Backlog.md, Jira, GitLab, GitHub)
- Read SDD changes and tasks in the Docs entry
- Search observations, open the detail pane, and run the governance controls in Memory
- Check code-index status, search hits, and open source in Code
- List workspace sessions, jump from recent context, and read session summaries in Sessions
- Call documented HTTP routes from Swagger

Write routes require `Authorization: Bearer …` when `SKILLGRID_HTTP_TOKEN` is set (store the token in the dashboard Settings). Read routes stay open on localhost.

### Tracker entry CLI dependency

The Tracker entry is a bridge: it shells out to the provider's CLI to fetch and create issues. Install the CLI for your tracker before using the entry:

| Tracker | CLI required |
|---------|--------------|
| Backlog.md | `backlog` (`brew install backlog` / `npm i -g backlog.md`) |
| GitHub | `gh` |
| GitLab | `glab` |
| Jira | `jira` (Atlassian CLI) |

If the CLI is missing, the Tracker entry renders an in-pane error naming the missing binary instead of a blank board.

## Who uses it

| Consumer | How |
|----------|-----|
| You | Browser at `/` for the dashboard entries |
| OpenCode / Kilo plugins | HTTP client to the same server (auto-start on health fail) |
| Agents | Prefer MCP (`skillgrid mcp`); HTTP is secondary |

## Security note

Default bind is loopback only. Do not expose `skillgrid serve` on a public interface without authentication and network controls.

## Next step

Back to [Start here](00-start-here.md) or [Memory and indexing](07-memory-and-indexing.md).
