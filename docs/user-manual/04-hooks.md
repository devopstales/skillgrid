# Hooks

Agent lifecycle hooks that run on session events (start, prompt, tool use, compaction).

## Current status

**No agent hooks are configured in the current Skillgrid release.**

Install copies skills, rules, MCP config, and Mnemonic plugins. Hook directories are reserved for a later release.

## Planned targets

| Agent | Hook location (planned) |
|-------|-------------------------|
| OpenCode | `~/.config/opencode/hooks/` |
| Kilo | `~/.config/kilo/hooks/` |
| Cursor | `~/.cursor/hooks/` |

Repo-local git hooks for this project (contributor hygiene) live under `git-hooks/` and are separate from agent runtime hooks.

## What hooks will cover (intent)

When shipped, expect hooks aligned with Mnemonic + SDD discipline:

- Session start → memory protocol / `mem_session_start`
- Compaction → recover context from Mnemonic
- Stale code index nudge → `code_index`
- Optional write guards for curated SDD artifacts

Until then, plugins and AGENTS rules carry that behaviour for OpenCode/Kilo/Cursor — see [Plugins](09-plugins.md).

## Next step

[MCP servers](05-mcp-servers.md)
