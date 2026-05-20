# IDEs

* Embeddid
  * Google Antigravity
  * Cursor
* Plugins
  * Github Copilot
  * Kilo Code
  * OpenChamber
* Agents
  * Gemini
  * cursor cli
  * Copilot CLI
  * kilo cli
  * Opencode
  * Claude Code
  * Codex
  * pi
  * factory droid

# IDE Configs

## Embedded

- Google Antigravity
- Cursor

| Config Area | Google Antigravity | Cursor |
| ----------- | ------------------ | ------ |
| Commands | `.agents/workflows` | `.cursor/commands/`? |
| Skills | `.agents/skills/` | `.agents/skills/`, `.cursor/skills/` |
| Rules | `.agents/rules/`, `AGENTS.md` | `.cursor/rules/`, `AGENTS.md` |
| Agents | `.agents/agents.md`, `.agent/agents/` | `.cursor/agents/` |
| Hooks | `.agents/hooks/`, `.agents/settings.json` (Gemini/Antigravity) | `.cursor/hooks.json`, `.cursor/hooks/` |
| MCP | `~/.gemini/antigravity/mcp_config.json`, `.agent/mcp_config.json` | `.cursor/mcp.json` |
| Plugins | X | `.cursor/plugins/` |

Hub workflow rules (modular Cursor-style `.mdc`, scope via `description` / `globs` / `alwaysApply`) live under **`.agents/rules/`**, for example:

- `skillgrid-context-contract.mdc` — `.skillgrid/project/CONTEXT.md`, glossary, conflicts
- `skillgrid-gentle-persona.mdc` — senior-architect tone, workspace rules, skill triggers
- `skillgrid-gentle-orchestrator-extended.mdc` — init guard, registry resolution, artifact keys
- `skillgrid-strict-tdd-mode.mdc` — strict TDD expectation flag
- `skillgrid-karpathy-coding.mdc` — think / simplicity / surgical / goal-driven habits (always on; [upstream CLAUDE.md](https://raw.githubusercontent.com/forrestchang/andrej-karpathy-skills/refs/heads/main/CLAUDE.md))
- `skillgrid-coding-discipline.mdc` — deprecated alias → `skillgrid-karpathy-coding.mdc`
- `skillgrid-sdd-workflow.mdc` — SDD phases and handoff paths
- `skillgrid-sdd-enforcement.mdc` — gates, labels, two-stage review
- `skillgrid-sdd-orchestrator.mdc` — coordinator-only delegation and subagent CONTEXT protocol
- `skillgrid-sdd-execution.mdc` — apply/loop/TDD/evidence expectations
- Phase-bound persona dispatch — `sdd-<phase>/SKILL.md` + `sdd-persona-delegation.md` (see `docs/10-subagent-personas.md`)
- `skillgrid-multiagent-handoff.mdc` — subagent parallelism and merge duty
- `skillgrid-engram-memory.mdc` — Engram retrieval, proactive saves, session close
- `skillgrid-communication-discipline.mdc` — honesty, pause, commits (pairs with persona rule)
- `skillgrid-workflow-rule-index.mdc` — cross-links hub workflow rules (local paths)
- `skillgrid-planning-artifact-parity.mdc` — durable planning paths → `.skillgrid` / `openspec`
- `skillgrid-branch-finish-target.mdc` — after verify: explicit merge/PR/keep/discard → cleanup (manual target)
- `skillgrid-gitnexus-nonnegotiables.mdc` — GitNexus graph safety (impact before edit, `gitnexus_rename`, `gitnexus_detect_changes`, embeddings-aware analyze; adapted from upstream [GitNexus `.cursor/index.mdc`](https://github.com/abhigyanpatwari/GitNexus/blob/main/.cursor/index.mdc))
- `.cursor/rules/skillgrid-gitnexus-index.mdc` — short Cursor-local entry pointing at the canonical GitNexus rule above (mirrors GitNexus [`.cursor` layout](https://github.com/abhigyanpatwari/GitNexus/tree/main/.cursor))

**`.configs/AGENTS.md`** is the short **index** into these rules and key docs (no duplicate policy text).

## Plugins

- Github Copilot
- Kilo Code
- OpenChamber

| Config Area | Github Copilot | Kilo Code | OpenChamber |
| ----------- | -------------- | --------- | ----------- |
| Commands | `.github/prompts/` | `.kilo/commands/` | `.opencode/commands/*.md` |
| Skills | `.agents/skills/`, `.claude/skills/`, `.github/skills`, `.copilot/skills` | `.agents/skills/`, `.claude/skills/`, `.kilo/skills/` | `.agents/skills/`, `.claude/skills/`, `.opencode/skills/` |
| Rules | `.github/instructions/*.instructions.md`, `.github/copilot-instructions.md`, `AGENTS.md` | `.kilo/rules/`, `AGENTS.md` | `.opencode/rules/`, `AGENTS.md` |
| Agents | `.github/agents/` | `.kilo/agents/` | `.opencode/agents/` |
| Hooks | `.github/hooks/*.json` (e.g. `context-mode.json`) | `.kilo/kilo.jsonc` plugin + `.kilo/hook/hooks.md` | `.opencode/opencode.json` plugin + `.opencode/hook/hooks.md` |
| MCP | `.vscode/mcp.json` | `.kilo/kilo.jsonc` | `.opencode/opencode.jsonc` |
| Plugins | X | X | `.opencode/plugins/` |

## Agents

- kilo cli
- Opencode
- Gemini
- Claude Code
- Codex
- pi

| Config Area | kilo cli | Opencode | Gemini | Claude Code | Codex | pi |
| ----------- | -------- | -------- | ------ | ----------- | ----- | -- |
| Commands | `.kilo/commands/` | `.opencode/commands/*.md` | `.gemini/commands/*.toml` | ? | ? | ? |
| Skills | `.agents/skills/`, `.claude/skills/`, `.kilo/skills/` | `.agents/skills/`, `.claude/skills/`, `.opencode/skills/` | `.agents/skills/` | `.agents/skills/`, `.claude/skills/` | `.agents/skills/` | ? |
| Rules | `.kilo/rules/`, `AGENTS.md` | `.opencode/rules/`, `AGENTS.md` | ? | `AGENTS.md` | `AGENTS.md` | ? |
| Agents | `.kilo/agents/` | `.opencode/agents/` | ? | ? | ? | ? |
| Hooks |  `.kilo/hook/hooks.md` | `.opencode/hook/hooks.md` | `.gemini/settings.json` | ? | ? | ? |
| MCP | `.kilo/kilo.jsonc` | `.opencode/opencode.jsonc` | `.gemini/settings.json` | ? | ? | ? |
| Plugins | X | `.opencode/plugins/` | ? | ? | ? | ? |

## Norse Persona Config Set

Norse persona theming and routing are centralized in shared config files:

- `.agents/skills/_shared/sdd-persona-delegation.md` and `sdd-<phase>/SKILL.md` (per-phase persona invocations)
- `.configs/ide-model-mapping.json` (surface model mapping by `fast|balanced|deep`)
- `.configs/ide-persona-capabilities.json` (surface capability tiers and fallback policy)

Render cross-IDE persona manifests with:

```bash
node scripts/render-multi-ide-personas.mjs
```

This writes `norse-personas.json` files into each surface (`.cursor`, `.kilo`, `.opencode`, `.github`, `.gemini`, `.codex`, `.pi`, `.antigravity`, `.vscode`, `.claude`) so agent prompt metadata stays consistent across toolchains. Persona **dispatch** is defined in `sdd-<phase>/SKILL.md`, not in those JSON files.

### Google Antigravity

### Cursor

```json
.cursor/hooks.json
{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "command": "context-mode hook cursor pretooluse",
        "matcher": "Shell|Read|Grep|WebFetch|Task|MCP:ctx_execute|
        MCP:ctx_execute_file|MCP:ctx_batch_execute"
      }
    ],
    "postToolUse": [
      {
        "command": "context-mode hook cursor posttooluse"
      }
    ]
  }
}
```

Hub ships [context-mode](https://github.com/mksglu/context-mode) hooks and routing rules in-repo (synced to targets via `install.sh` / `skillgrid install`):

- `.cursor/hooks.json` — `preToolUse`, `postToolUse`, `stop`
- `.cursor/rules/context-mode.mdc` — routing rules (`alwaysApply: true`)

Requires `context-mode` on PATH (`npm install -g context-mode`). Verify: `ctx stats` in agent chat.

```json
.cursor/mcp.json
{
  "mcpServers": {
    "context7": {
      "args": [
        "-y",
        "@upstash/context7-mcp"
      ],
      "command": "npx"
    },
    "engram": {
      "args": [
        "mcp",
        "--tools=agent"
      ],
      "command": "engram"
    }
  }
}
```


### VSCode / Github Copilot

```json
.vscode/mcp.json
{
  "servers": {
    "context7": {
      "type": "http",
      "url": "https://mcp.context7.com/mcp"
    },
    "engram": {
      "args": [
        "mcp",
        "--tools=agent"
      ],
      "command": "engram"
    }
  }
}
```

### Kilo Code / CLI


### Opencode

```json
.opencode/opencode.json
{
  "$schema": "https://opencode.ai/config.json",
  "instructions": ["CONTRIBUTING.md", "docs/guidelines.md", ".opencode/rules/*.md"],
  "plugin": [
    "./plugins"
  ]
}
```

```json
.opencode/opencode.json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "context7": {
      "enabled": true,
      "type": "remote",
      "url": "https://mcp.context7.com/mcp"
    },
    "engram": {
      "command": [
        "engram",
        "mcp",
        "--tools=agent"
      ],
      "enabled": true,
      "type": "local"
    }
  }
}
```

### pi

```bash
npm install -g @mariozechner/pi-coding-agent
```
