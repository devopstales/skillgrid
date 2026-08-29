# Start here

**skillgrid** is a Go CLI that installs an AI-assisted development hub onto a machine. It wires **OpenCode**, **Kilo**, and **Cursor** with persistent memory, code search, and web research caching.

## What you get

1. **Managed home** at `~/.skillgrid/` (`SKILLGRID_MNEMONIC_DATA_DIR` overrides the data dir). Binaries, cloned hub, config, and state live there.
2. **Tools** — `skills` CLI (npm), `openspec` CLI (npm), and the `skillgrid` binary itself.
3. **Skills** — installed through the managed `skills` CLI, from configured sources.
4. **MCP** — `skillgrid mcp` stdio server exposes `mem_*`, `code_*`, and `web_*` tools to agents.
5. **Rules** — files from the hub `.agents/` copied into `~/.agents/`.

## Requirements

- `git` on `PATH` (for `sync-repo` and the default `install` path)
- `node` and `npm` on `PATH` (`install` hard-fails without them)
- Go 1.22+ and [Task](https://taskfile.dev/) to build from source

skillgrid does **not** install Node or third-party API keys.

Install the CLI first ([Installation](01-installation.md)). Then run `skillgrid install`.

## Next

1. [Installation](01-installation.md)
2. [Usage](02-usage.md)
3. [Clients](03-clients.md)
4. [Tools](04-tools.md) · [Skills](05-skills.md) · [Rules](06-rules.md) · [MCP](08-mcp.md) · [Plugins](09-plugins.md)
