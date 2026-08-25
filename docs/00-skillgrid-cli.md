# skillgrid cli

A single Go binary that installs and configures your AI agents in one command, from one place: a checked-in `config.d/` where the tools, MCP servers, and rules live as plain YAML.

The point is simple: your agent setup should not live in your head, in shell history, or in a second blog post per IDE. It should be a config, a repo, and one binary that makes your machine match that config.

## Why This Exists

Most AI agent setups accumulate by accident: one MCP server added for a demo, one skill copied from a tutorial, an `AGENTS.md` that drifted out of sync because nobody re-ran the install. When you open a new laptop — or a new agent — you rebuild the whole thing by hand and it never quite matches the old one.

skillgrid solves that by making the whole setup reproducible:

- `config.d/tools.yaml` defines which agent CLIs and tools get installed.
- `config.d/mcp.yaml` defines which MCP servers merge into each agent's config.
- `config.d/skills.yaml` defines which skills get added.
- `config.d/AGENTS.md` defines the rules every agent runs under.

One binary reads those files, does the work, backs up what it touches, and prints the PATH lines you still need. Re-run it any time to reconcile the difference.

## What You Get

- A reproducible agent environment across supported agents (Kilo, OpenCode today; more via `config.d`).
- JSON-aware config merge that preserves your existing keys and comments — no hand-editing `kilo.jsonc`.
- Automatic backups of every agent config before it is modified, pruned to the last 10.
- A dry-run mode that shows exactly what would change without touching anything.
- Config-as-source-of-truth: changing what gets installed means editing a YAML file, not the code.

## Quick Start

```bash
# build
task build

# install into this machine (clone mode)
./bin/skillgrid install

# or point it at a local checkout of this repo (dev loop)
./bin/skillgrid install --sync-repo /path/to/skillgrid-v2

# preview instead of apply
./bin/skillgrid install --dry-run
```

## Documentation Map

| Doc | Topic |
|-----|-------|
| [01-installation](01-installation.md) | Requirements, build, install flow step by step |
| [02-usage](02-usage.md) | Day-to-day usage: flags, dry-run, sync-repo, PATH |
| [03-config-reference](03-config-reference.md) | Every file in `config.d/` and its schema |
| [04-mcp-servers](04-mcp-servers.md) | MCP registry, merge semantics, backups |
| [05-skills](05-skills.md) | Skill installation from `skills.yaml` |
| [06-rules](06-rules.md) | Where `AGENTS.md` comes from and how it lands in agents |
| [07-plugins](07-plugins.md) | superpowers and engram plugins, per-agent registration |
