# Skills and Plugins

aiskillgrid installs two classes of extensions on top of the base agent CLIs:

- **Skills** — per-agent behavior definitions installed via the `skills` CLI
- **Plugins** — richer packages (currently `superpowers` and `engram`) installed and registered per agent

## Skills

Skills are the unit of behavior that agents can load per task. aiskillgrid installs them from `config.d/skills.yaml` (schema in [03-config-reference](03-config-reference.md)).

The CLI invokes the `skills` CLI — not `npx`, because that pulls in a version at runtime that may drift:

```bash
~/.aiskillgrid/node_modules/.bin/skills add <repo> --agent <agent> -g -s <skill> -y
```

Installed by the default `skills.yaml`:

| Skill | Source | Target |
|-------|--------|--------|
| Superpowers skill set | `obra/superpowers` | all (`*`) |
| engram-memory | `gentleman-programming/engram` | `amp` (default) |
| engram-memory-protocol | `gentleman-programming/engram` | `amp` |
| engram-testing-coverage | `gentleman-programming/engram` | `amp` |

Note: `skills.yaml` entries take an optional `agent` field. If unset, the default (`amp`) is used. Change them if you want a skill targeted at a different agent.

## Plugins

### superpowers

Per selected agent, the CLI runs:

```bash
npm install superpowers@git+https://github.com/obra/superpowers.git \
    --prefix "$HOME/.config/<agent>"
```

Then registers the plugin path under the `plugin` key of that agent's config:

```json
{
  "plugin": ["~/.config/kilo/node_modules/superpowers"]
}
```

The append is idempotent — re-running does not duplicate the entry.

### engram

The `engram` binary is installed in step 3 (`~/.aiskillgrid/bin/engram`). For the opencode agent, the CLI additionally runs:

```bash
~/.aiskillgrid/bin/engram setup opencode
```

If a kilo config also exists, the `engram.ts` plugin file that opencode wrote is copied to kilo:

```
~/.config/opencode/plugins/engram.ts  ->  ~/.config/kilo/plugins/engram.ts
```

This gives both agents the same engram plugin without double-running `engram setup`.

## Uninstall / Update

aiskillgrid is idempotent but does not yet track "skills I installed" for removal. To update to the latest skill, bump the entry in `config.d/skills.yaml` and re-run `install` — the `skills` CLI handles add/replace semantics for its own registry.

To remove a plugin, uninstall it directly in the target agent's own mechanism (`kilo` / `opencode` settings, or `npm uninstall` the prefix).

## What "agent" Mean in aiskillgrid

`config.d/skills.yaml` uses the same identifier as the `skills` CLI's `--agent` option: `amp` (a generic target), `kilo`, `opencode`, etc. This is the same `agent` the selector in step 0 asks about — but the selector only decides which *config files to merge*, not which *skill target* to use. Set `agent:` per entry to route skills to the right tool.
