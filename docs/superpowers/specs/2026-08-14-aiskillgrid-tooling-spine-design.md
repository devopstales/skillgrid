# aiskillgrid CLI — Tooling Spine Design

Date: 2026-08-14  
Slice: **A — Tooling spine**  
Status: Implemented (extends [2026-08-14-aiskillgrid-cli-design.md](./2026-08-14-aiskillgrid-cli-design.md))

## Goal

On `aiskillgrid install`, provision managed tool runtimes and wire Skillgrid-owned MCP entries into selected agents:

1. Ensure home layout for native bins + managed npm
2. Install tools (**binary where possible, npm where not**)
3. Resolve MCP server configs with absolute managed paths
4. Merge into agent MCP files (`aiskillgrid-` prefix, one-time `.bak`)

## Install order

After layout ensure + optional git sync, before/with agent skill copy:

1. Ensure `~/.aiskillgrid/dependencies/bin/` and `~/.aiskillgrid/npm/` (`bin/`, `cache/`)
2. Ensure system `node` + `npm` (required for npm-only tools). If missing → warn; skip npm installs; continue binary installs + agent skill copy
3. Install native binaries into `dependencies/bin/`
4. `npm install --prefix ~/.aiskillgrid/npm` for npm-only packages (idempotent)
5. Resolve `packs/mcp/servers.json` → substitute placeholders → drop entries that cannot be satisfied → emit warnings
6. Agent install merges the **resolved** server map (not the raw pack with placeholders)

## Install policy: binary first, npm fallback

| Tool | Method | Target |
|------|--------|--------|
| Engram | GitHub release binary | `dependencies/bin/engram` |
| qntx/skill | GitHub release binary | `dependencies/bin/skills` |
| Backlog.md | Prefer GitHub release binary into `dependencies/bin/` when a Skillgrid-supported asset exists; else managed npm `backlog.md` | bin or `npm/bin` |
| GitNexus | npm (no standalone binary) | `npm` prefix → `gitnexus` |
| OpenSpec | npm `@fission-ai/openspec` | `npm` prefix → `openspec` |
| Context7 MCP | npm `@upstash/context7-mcp` | managed npm |
| Playwright MCP | npm `@playwright/mcp` | managed npm |
| DeepWiki MCP | Pin package or HTTP at implement; prefer npm if package exists | managed npm / HTTP |

### Rules

- **No Homebrew (or Nix) for tool installs.** Only: (1) GitHub release binaries into `dependencies/bin/`, or (2) managed `npm install --prefix ~/.aiskillgrid/npm`. Do not shell out to `brew` / `nix`.
- Never pollute the user’s global npm. Always `--prefix ~/.aiskillgrid/npm` (and dedicated cache).
- Prefer absolute paths to managed binaries in MCP `command` fields (avoids `npx -y` cold starts).
- Re-running install is idempotent (re-download only if missing/outdated policy TBD; v1: ensure present, npm install again OK).
- This slice does **not** run `skills add` / upstream pack orchestration (slice B — see [05-skills.md](../../05-skills.md) and `packs/skills/sources.yaml`).
- This slice does **not** run OpenSpec/Backlog project scaffolds or `AGENT.md` generation (slice C / later).
- Playwright browser ensure (`npx playwright install`) is a follow-up; warn once that browsers may be required.

## Home layout

```text
~/.aiskillgrid/
  dependencies/
    bin/          # engram, skills, backlog (if binary), …
  npm/            # isolated npm prefix
    bin/          # gitnexus, openspec, backlog.md, npx, MCP CLIs
    cache/        # npm cache
  tools/          # synced hub repo
  config.yaml
  state.json
  logs/
  …
```

Extend `home.Paths` with `DepsBinDir`, `NpmDir`, `NpmBinDir`, `NpmCacheDir`. `EnsureLayout` creates them.

## MCP pack contract

`packs/mcp/servers.json` holds templates. Placeholders substituted at install:

| Placeholder | Meaning |
|-------------|---------|
| `{{AISKILLGRID_NPX}}` | Absolute managed `npx` (if needed) |
| `{{AISKILLGRID_NPM}}` | `~/.aiskillgrid/npm` |
| `{{AISKILLGRID_NPM_CACHE}}` | `~/.aiskillgrid/npm/cache` |
| `{{AISKILLGRID_BIN}}` | `~/.aiskillgrid/dependencies/bin` |
| `{{AISKILLGRID_ENGRAM}}` | Absolute path to managed `engram` |
| `{{AISKILLGRID_GITNEXUS}}` | Absolute path to managed `gitnexus` |
| `{{AISKILLGRID_BACKLOG}}` | Absolute path to managed backlog binary |

Optional entry metadata (stripped before merge):

- `"requires": "binary:engram"` | `"npm:gitnexus"` — skip + warn if not installed
- Always include Context7 / Playwright / DeepWiki when their npm packages installed; else warn as a group if npm unavailable

Merge rules unchanged: overwrite only keys with prefix `aiskillgrid-`; one-time `.bak`.

Example shapes (commands pinned at implement):

```json
{
  "mcpServers": {
    "aiskillgrid-engram": {
      "command": "{{AISKILLGRID_ENGRAM}}",
      "args": ["mcp"],
      "requires": "binary:engram"
    },
    "aiskillgrid-gitnexus": {
      "command": "{{AISKILLGRID_GITNEXUS}}",
      "args": ["mcp"],
      "requires": "npm:gitnexus"
    },
    "aiskillgrid-context7": {
      "command": "{{AISKILLGRID_NPX}}",
      "args": ["-y", "@upstash/context7-mcp"],
      "env": {
        "npm_config_prefix": "{{AISKILLGRID_NPM}}",
        "npm_config_cache": "{{AISKILLGRID_NPM_CACHE}}"
      },
      "requires": "npm:@upstash/context7-mcp"
    }
  }
}
```

Prefer bin absolute path over `npx` when the package exposes a bin after `npm install --prefix`.

## Go packages

| Package | Responsibility |
|---------|----------------|
| `home` | New path fields + layout |
| `tools` | Ensure binaries, npm install set, resolve MCP map, warnings |
| `install` | Orchestrate tools phase; pass resolved servers into agent merge |
| `agents` / `mcpmerge` | Accept resolved server map (inject) so agents don’t re-resolve |

Suggested `tools` surface:

- `EnsureManagedNPM(paths) error` — layout + verify `node`/`npm`
- `InstallNPMPackages(paths, pkgs []string) error`
- `EnsureReleaseBinary(name, repo, destDir) error` — Engram, skills
- `ResolveMCPServers(packPath, paths) (servers map, warnings []string, err error)`

## Status / logging

- Log each binary/npm install result under `logs/`
- `aiskillgrid status` should show whether managed npm exists and which managed bins are present (lightweight; can be incremental)

## Tests

- Placeholder substitution
- `requires` skip/include
- Layout directories created under temp `AISKILLGRID_HOME`
- MCP merge still prefix-scoped
- Default unit tests: **no network** (mock downloader / fake PATH bins)
- Optional network tests behind build tag for real release fetch

## Out of scope (this slice)

- Upstream skill orchestration via `skills add` (Superpowers, mattpocock, OpenSpec, Engram, gentle-ai — [05-skills.md](../../05-skills.md))
- OpenSpec `openspec init` / Backlog project scaffold
- `AGENT.md` / `CLAUDE.md` / `GEMINI.md` generation
- Portable Node download (system Node required for npm tools)
- Homebrew formula / Nix flake for aiskillgrid itself
- Auto-install of Engram via npm (impossible — Go binary only)

## Doc updates after implement

- Flip relevant checkboxes in [TODO.md](../../TODO.md)
- Adjust “detect + warn” language in [04-tools.md](../../04-tools.md) to “install binary/npm into managed home + wire MCP”
