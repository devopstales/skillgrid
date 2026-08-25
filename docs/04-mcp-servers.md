# MCP Servers

MCP servers are defined in `config.d/mcp.yaml` and merged by the installer into each selected agent's config under the top-level `mcp` key.

The merge is JSON-aware (uses `tidwall/gjson` + `tidwall/sjson`), so it preserves existing keys, comments, and formatting; it never clobbers an MCP server that is not in `mcp.yaml`. This is the difference between re-running the installer with confidence and hand-editing a JSON file while holding your breath.

## Default Registry

What ships in `config.d/mcp.yaml` today:

| ID | Type | Source |
|-----|------|--------|
| `context7` | remote | `https://mcp.context7.com/mcp` |
| `deepwiki` | remote | `https://mcp.deepwiki.com/mcp` |
| `exa` | remote | `https://mcp.exa.ai/mcp` |
| `engram` | local | `engram mcp` (binary from step 3) |
| `ccc` | local | `ccc mcp` |
| `gitnexus` | local | `npx -y gitnexus@1.3.11 mcp` |
| `trivy` | local | `trivy mcp` |

Local tools whose binary is not on PATH do not fail the install — the merge still happens and the agent will surface a "server failed to start" error at runtime if the binary is missing. Add the binary to your PATH later and the config still works.

## Merge Semantics

For each selected agent, the CLI:

1. Backs up the existing config file to `~/.skillgrid/backups/` (keep last 10 per file).
2. Reads the JSON/JSONC.
3. For each server in `mcp.yaml`, sets `mcp.<name>` to the canonical entry for that server.
4. Appends `mcp.<name>-<old-type>` with the old entry if the entry existed under the same name before with a *different* `type` — the CLI never deletes a user override silently.
5. Writes the file back; dry-run skips step 5 and reports the change instead.

Result: the managed block in `kilo.jsonc` / `opencode.jsonc` looks the same every run.

```json
{
  "mcp": {
    "context7": { "type": "remote", "url": "https://mcp.context7.com/mcp", "enabled": true },
    "engram":   { "type": "local",  "command": ["engram", "mcp"], "enabled": true }
  }
}
```

This matches both Kilo Code's and OpenCode's documented MCP schema — `type` + (`url` for remote, `command` for local) + `enabled`. (Older CLI revisions emitted `"Type": "remote"` with capitalized keys; that shape was invalid to both agents and has been fixed.)

## Verify a Merge

```bash
# dry run shows what would change
./bin/skillgrid install --dry-run --sync-repo $(pwd) --verbose

# verify the file is still valid JSON after
node -e 'JSON.parse(require("fs").readFileSync(process.env.HOME+"/.config/kilo/kilo.jsonc","utf8")); console.log("kilo OK")'
node -e 'JSON.parse(require("fs").readFileSync(process.env.HOME+"/.config/opencode/opencode.jsonc","utf8")); console.log("opencode OK")'
```

## Backup and Rollback

Every non-dry-run run creates a backup before each edit:

```bash
ls ~/.skillgrid/backups/
# kilo.jsonc.20260825-154345.bak
# kilo.jsonc.20260825-154345.bak.2
# opencode.jsonc.20260825-154345.bak
```

Rolling back to a specific moment is a `cp`:

```bash
cp ~/.skillgrid/backups/kilo.jsonc.20260825-154345.bak ~/.config/kilo/kilo.jsonc
```

## Adding a New MCP Server

Edit `config.d/mcp.yaml`, then re-run:

```yaml
servers:
  context7:
    type: remote
    url: https://mcp.context7.com/mcp
  # add a new entry
  my-server:
    type: local
    command:
      - my-server
      - serve
      - --mcp
```

The installer will add it to Kilo and OpenCode configs. Existing entries you manage by hand that are not in `mcp.yaml` remain untouched.
