# Plugins

`skillgrid setup <agent>` installs Mnemonic integration for one of `opencode`,
`kilocode`, or `cursor` (`kilo` is accepted as an alias for `kilocode`). It copies
plugin files from `plugins/<agent>/` in the repo into the agent's config
directory and registers `skillgrid mcp` as the MCP server.

## OpenCode / Kilo

- Copies `plugins/opencode/mnemonic.ts` → `~/.config/opencode/plugins/mnemonic.ts`
- Copies `plugins/kilo/mnemonic.ts` → `~/.config/opencode/shared/http-client.ts`
- Upserts `mcp.skillgrid-mnemonic` in `~/.config/opencode/opencode.jsonc` (or `kilo.jsonc` for Kilo)
- For Kilo: writes the Memory Protocol (from `plugins/kilo/memory-protocol.md`)
  into `~/.config/kilo/AGENTS.md` between managed markers, and bridges shared files from
  OpenCode's config.
- The OpenCode plugin reads `SKILLGRID_MNEMONIC_HTTP_URL` (default `http://127.0.0.1:7438`)
  and `SKILLGRID_MNEMONIC_HTTP_TOKEN` for requests to `skillgrid serve`.
- Both plugins auto-start `skillgrid serve` if `GET /health` fails, create a session
  on `session.created`, inject the Memory Protocol on every chat turn, recover context
  on compaction, and nudge on a stale code index.

## Cursor

- Registers `skillgrid mcp` in `~/.cursor/mcp.json` under `mcpServers.skillgrid-mnemonic`
- Writes `~/.cursor/rules/mnemonic.mdc` from the template at
  `plugins/cursor/mnemonic.mdc`, injecting the Memory Protocol
  in place of `{{MEMORY_PROTOCOL}}`.
