# Web UI

Mnemonic ships a small embedded **data viewer** and Swagger UI with `skillgrid serve`.

## Quick path

```bash
skillgrid serve
# open http://127.0.0.1:7438/
```

| URL | What |
|-----|------|
| `/` | Data viewer (sessions, observations, code index, web cache) |
| `/swagger-ui` | Interactive OpenAPI explorer |
| `/openapi.yaml` | OpenAPI spec |
| `/health` | Liveness JSON |

Defaults: bind `127.0.0.1`, port `7438`. Override with `--port` / `--bind` or `SKILLGRID_MNEMONIC_PORT`.

```bash
skillgrid serve --port 7438 --bind 127.0.0.1
```

## What you can do

- Browse projects and session list (titles from `mem_session_start` / `mem_session_set_title`)
- Inspect observations and search memory
- Check code index status and search hits
- Inspect web-cache entries
- Call documented HTTP routes from Swagger

Write routes require `Authorization: Bearer …` when `SKILLGRID_HTTP_TOKEN` is set. Read routes stay open on localhost.

## Who uses it

| Consumer | How |
|----------|-----|
| You | Browser at `/` for debugging memory and index health |
| OpenCode / Kilo plugins | HTTP client to the same server (auto-start on health fail) |
| Agents | Prefer MCP (`skillgrid mcp`); HTTP is secondary |

## Security note

Default bind is loopback only. Do not expose `skillgrid serve` on a public interface without authentication and network controls.

## Next step

Back to [Start here](00-start-here.md) or [Memory and indexing](07-memory-and-indexing.md).
