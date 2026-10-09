# Findings — skill-pi (customized Pi agent for skillgrid)

> **Decision this serves:** Where and how skillgrid ships `skill-pi`, a customized
> Pi coding agent, in this repo — the folder structure, the split between
> committed source and installer-managed machine state, and the mechanism that
> points Pi's two config layers at skill-pi's own paths.
>
> **Type:** technical (which Pi extension points exist, how this repo ships
> agent config) + structural (where skill-pi's source lives and how it is
> installed)
> **Mode:** research (single pass)
> **Date:** 2026-10-09 · **Status:** complete
> **Primary corpus:** `earendil-works/pi` source read directly this run
> (`config.ts`, `settings-manager.ts`, `resource-loader.ts`, `trust-manager.ts`,
> `session-manager.ts`, `footer-data-provider.ts`, `main.ts`, `package.json`);
> this repo's `plugins/`, `agents/`, `git-hooks/`, `.skillgrid/config.yaml`.

## Executive Summary

`skill-pi` is an **opinionated version of `pi-coding-agent`**
(`@earendil-works/pi`) built on the skillgrid standards — not a fork. The
reference implementation of this pattern is `open-gsd/gsd-pi`: an npm package
whose own `package.json` declares `piConfig: { name: "gsd", configDir: ".gsd" }`
and whose `bin` points at a bootstrap script (not the pi entrypoint) that sets
up the environment and launches the pinned pi — the `gsd` CLI **is** pi running
with GSD's identity (`APP_NAME=gsd`, project config in `.gsd/`) [8]. skill-pi
follows the same pattern with the skillgrid identity: `piConfig: { name:
"skill-pi", configDir: ".skill-pi" }`, global state in `~/.skill-pi/`, and the
skillgrid workflow/skills/standards pre-loaded. Pi is extended through five
seams — project config-dir resources (`mcp.json`, `skills/`, `extensions/`,
`prompts/`, `SYSTEM.md`), user-level agent-dir config, `AGENTS.md` context, MCP
servers, and a `pi` key in `package.json` for shipped packages [1][2].

**Core rule — skill-pi owns its own config paths.** All skill-pi config lives
under its own names:

| Layer | skill-pi path | Pi default | Mechanism |
|---|---|---|---|
| Global (user) | **`~/.skill-pi/`** | `~/.pi/agent/` | `PI_CODING_AGENT_DIR=~/.skill-pi` — Pi reads the agent dir from the env var; **the whole global tree moves with one env var** (settings, mcp, auth, models, trust.json, **and the session store**) [1] |
| Project | **`<project-root>/.skill-pi/`** | `<project-root>/.pi/` | Pi resolves the project config dir as `join(cwd, CONFIG_DIR_NAME)`. `CONFIG_DIR_NAME` defaults to `.pi` but is read from the **startup package's** `package.json` `piConfig.configDir` field [1][8]. The launcher execs pi from a per-project directory whose own `package.json` declares `piConfig.configDir = ".skill-pi"` (and `piConfig.name = "skill-pi"` for the identity — same trick as gsd-pi [8]), and symlinks the real project root in — so `join(cwd, ".skill-pi")` lands on `<project-root>/.skill-pi/`. |

Nothing skill-pi reads or writes lands in `~/.pi/` or `<project-root>/.pi/`.
A user's plain `pi` install keeps its own `~/.pi/` untouched (two agents, two
homes).

The repo already ships per-agent customization the same way (opencode plugins +
skills in `plugins/opencode/`, subagent definitions in `agents/opencode/`) [3],
and already has a **Pi extension point reserved**: the Context Orchestrator (CTX)
spec names `pi` in its `wrap:` opt-in agent list — so `mnemonic ctx wrap pi`
is the intended runtime integration [4].

**Recommended structure:** commit the skill-pi source under a new top-level
`skill-pi/` directory (source of truth, versioned with the repo); keep
**machine state out of the repo** — the installer syncs `skill-pi/` into
**`~/.skill-pi/`** (skill-pi's own home, NOT under `~/.skillgrid/`). `skill-pi`
is also the **CLI command**: a launcher that (1) points Pi's global config at
`~/.skill-pi` via `PI_CODING_AGENT_DIR`, (2) prepares the per-project cwd shim
so the project layer resolves from `<project-root>/.skill-pi/`, and (3) execs
the pinned `pi-coding-agent` with args forwarded. The launcher does **not**
vendor a copy of the Pi monorepo into this repo — it pins the Pi version
instead.

## Findings

### What Pi is and how it extends

`earendil-works/pi` is an AI agent toolkit (MIT) whose coding surface is
`pi-coding-agent` — an interactive coding-agent CLI — built on `pi-agent-core`
(runtime, tool calling, state) and `pi-ai` (unified multi-provider LLM API),
with `pi-tui` for the terminal UI [1].

Extension seams [2]:

| Seam | Default location | Relocatable? | What it controls |
|------|----------|--------------|------------------|
| Global (user) config | `~/.pi/agent/` | **Yes** — `PI_CODING_AGENT_DIR` env var (`${APP_NAME}_CODING_AGENT_DIR`), or `--session-dir`/`PI_CODING_AGENT_SESSION_DIR` for the session store [1] | `mcp.json`, `settings.json`, `auth.json`, `models.json`, `mcp-auth.json`, `trust.json`, `themes/`, `tools/`, `bin/`, sessions |
| Project config | `<project-root>/.pi/` | **No CLI flag — but the dir name is package-configured**: `join(cwd, CONFIG_DIR_NAME)` where `CONFIG_DIR_NAME = <startup-package.json>.piConfig.configDir \|\| ".pi"` [1] | `mcp.json`, `settings.json`, `SYSTEM.md` / `APPEND_SYSTEM.md`, `extensions/`, `skills/`, `prompts/`, `themes/`; loaded only when project is trusted |
| Context files | `AGENTS.md` / `CLAUDE.md` | Walked upward from the **effective cwd** to the git root; global copy read from the agent dir [1] | Loaded regardless of project trust unless disabled |
| Packaged resources | `package.json` `"pi"` key + `pi install` | n/a | A package can declare extensions/skills/prompts/themes; install from npm/git/local |

Project resources override user resources of the same name; discovery is
tiered (project-local > configured packages > global) and gated by project
trust for project resources [2]. The trust store itself lives at
`<agent-dir>/trust.json` [1].

### Key Pi behaviors (source-verified, main branch) [1]

- `getAgentDir()` returns `process.env[PI_CODING_AGENT_DIR]` when set, else
  `~/.pi/agent`. **One env var relocates the entire global tree** — settings,
  mcp, auth, models, trust, and the session store.
- Sessions live at `<agentDir>/sessions/<encoded-cwd>/` (`session-manager.ts`
  `getDefaultSessionDirPath`). The store is a **child of the agent dir**, so it
  relocates with `PI_CODING_AGENT_DIR` (no separate env var needed). Each
  session records its effective cwd in the header; `--continue`/`--session`
  filter by `sessionCwdMatches()` against the current cwd. Because skill-pi
  always launches pi from the same deterministic shim, resolution stays
  self-consistent entirely inside `~/.skill-pi/`.
- Project config path is `join(resolvePath(cwd), CONFIG_DIR_NAME, …)` in
  `settings-manager.ts` / `resource-loader.ts`. **No CLI flag** changes the
  project dir (`--session-dir` is the only directory flag and covers sessions
  only).
- `CONFIG_DIR_NAME` is computed once at startup:
  `pkg.piConfig?.configDir || ".pi"` where `pkg` is parsed by
  `getPackageJsonPath()`, which walks **upward from `process.argv[1]`** to the
  nearest `package.json`. It is the **startup entrypoint's package**, not the
  cwd's. Pi's own package ships `piConfig: { configDir: ".pi" }`, which is why
  plain pi reads `.pi/`.
- Project resources are discovered at `join(cwd, CONFIG_DIR_NAME)` — i.e.
  **cwd itself is the project root**; there is no `.git`-based project-root
  detection for config loading. `findGitPaths()` (also walks upward) is used
  only for footer git-branch display and worktree context-file shadowing.
- Trust gating: `hasTrustRequiringProjectResources(cwd)` triggers the trust
  prompt when `join(cwd, CONFIG_DIR_NAME)` contains trust-requiring entries, or
  when any ancestor has `.agents/skills`. The trust store
  (`<agent-dir>/trust.json`) is keyed by the **canonicalized (realpath'd)** cwd.
- Git worktrees: `findGitPaths` handles both a `.git` directory and a `.git`
  **file** (`gitdir: …` pointer), so a shim containing a symlinked `.git`
  resolves correctly through the real repo.

### How skill-pi relocates Pi's config

**Global layer** — trivial and fully supported: the launcher sets
`PI_CODING_AGENT_DIR=~/.skill-pi`. Pi then reads everything from `~/.skill-pi/`
(directly, not a nested `agent/` subdirectory — the env var is the agent dir
itself). `~/.pi/` is never touched.

**Project layer** — Pi hardcodes `join(cwd, CONFIG_DIR_NAME)`, but the *name*
is overridable via the startup package's `piConfig.configDir`. The launcher
therefore execs pi from a **cwd shim** whose own `package.json` renames the
config dir:

```
<project-root>/.skill-pi/          # ← pi's project config, read from here
  mcp.json  settings.json
  SYSTEM.md  APPEND_SYSTEM.md
  extensions/  skills/  prompts/  themes/
  overrides/                        # optional: per-user per-project edits
                                    #   (merged over the skill-pi base at launch)

~/.skill-pi/cwd/<project>/         # cwd shim — the directory pi runs from
  package.json                      # {"name":"skill-pi-cwd","version":"0.0.0",
                                    #  "piConfig":{"configDir":".skill-pi"}}
  .git -> <project-root>/.git       # symlink; findGitPaths handles .git files
  .skill-pi -> <project-root>/.skill-pi   # symlink (or real dir, see Q7)
  AGENTS.md -> <project-root>/AGENTS.md   # per-entry symlinks to the real
  src/ README.md go.mod ...             #   project root (rebuilt if drifted)
```

The launcher, per invocation:

1. Resolves the real project root (walk up to `.git`, like pi does).
2. Sets `PI_CODING_AGENT_DIR=~/.skill-pi`.
3. Prepares `~/.skill-pi/cwd/<project>/`: writes its `package.json` (fixed
   content: `piConfig.configDir = ".skill-pi"`), symlinks the real project root
   entries into it, and ensures `<project-root>/.skill-pi/` is materialized
   from the committed `skill-pi/` source (+ optional
   `<project-root>/.skill-pi/overrides/`).
4. `chdir`s into the shim and execs pi with `process.argv` forwarded. Because
   `argv[1]` is `~/.skill-pi/node_modules/.../dist/bundle/cli.js`, pi's
   startup `package.json` lookup lands on the shim's `package.json` (first one
   upward) → `CONFIG_DIR_NAME = ".skill-pi"`, `APP_NAME = "skill-pi"` → project
   config resolves as `join(shim, ".skill-pi")` → the symlink →
   **`<project-root>/.skill-pi/`**. (The gsd-pi precedent: the shim *is* the
   identity carrier; gsd ships the same via its own package `piConfig`.)
5. Trust: the shim's realpath (macOS `/tmp` → `/private/tmp` style) becomes the
   trust key in `~/.skill-pi/trust.json`; the first-run prompt fires once per
   project, then persists.

Consequences:

- Both config layers are exactly where skill-pi wants them: `~/.skill-pi/`
  (global) and `<project-root>/.skill-pi/` (project). Never `~/.pi/` or
  `<project-root>/.pi/`.
- Sessions: `~/.skill-pi/sessions/<encoded-shim-cwd>/`. `--continue` works
  because the shim is deterministic per project. The only observable is the
  cosmetic encoded folder name.
- The shim is the price of pi having no `--project` flag. It is deterministic,
  per-project, and the only machine-local state it adds is under
  `~/.skill-pi/cwd/`.

### How this repo already ships agent customization

| What | Where | Precedent |
|------|-------|-----------|
| Per-harness plugins (TS, package.json) | `plugins/opencode/`, `plugins/cursor/` (mcp.json only), `plugins/kilo/` | opencode gets 6 TS plugins + logo + memory protocol; cursor gets `mcp.json` [3] |
| Subagent definitions | `agents/opencode/*.md`, `agents/cursor/*.md` (debugger, implementer, researcher, reviewer, verifier) | Per-harness agent md files [3] |
| Shared skills + rules | `.agents/skills/` (37 skills), `.skillgrid/config.d/` | Shipped by the Installer, staged to `~/.skillgrid/` (Hub Content) [5] |
| Git hooks | `git-hooks/` + commit-msg zone guard | Conventional-commit enforcement [6] |
| Installer | `skillgrid-cli/` (Go) `install`/`sync-repo`/`setup` | The only writer of agent MCP config + `~/.skillgrid/` [5] |

The installer is the established owner of machine-local agent state: "Writes to
`~/.skillgrid/`, `~/.agents/`, and agent config files; the only writer of agent
MCP config" [5]. skill-pi should follow the same split: **repo = source,
installer = machine state**.

### Runtime integration point already reserved

The CTX spec (ADR-0033…0036, in force) defines `mnemonic ctx wrap <agent>` with
a core-4 (opencode, claude, codex, cursor) and an opt-in set that explicitly
includes `pi`-class agents [4]. Two implications:

1. `skill-pi`'s Pi process is expected to run **through the CTX transparent
   proxy** (`mnemonic ctx wrap pi` → proxy :8787 → base-URL override into Pi's
   LLM endpoint) — so skill-pi's `mcp.json`/`settings.json` must leave the LLM
   base URL overridable rather than hardcoding a provider URL.
2. CTX execution is gated on `2026-10-06-context-harness-clm` +
   `2026-10-07-replace-opencode-hooks-with-plugins` landing [4]. skill-pi does
   not need to wait on that to ship the *structure*; the proxy wiring is a later
   delta.

### Recommended structure

```
skill-pi/                          # NEW top-level dir — committed source of truth
  README.md                        # what skill-pi is, how to install, version pin
  package.json                     # name "skill-pi", piConfig: { name:
                                    #   "skill-pi", configDir: ".skill-pi" }
                                    #   (identity + project config dir, gsd-pi
                                    #   pattern [8]), "pi" key declaring shipped
                                    #   resources (extensions/skills/prompts),
                                    #   bin: { "skill-pi": "./bin/skill-pi.js" },
                                    #   deps @earendil-works/pi-coding-agent
                                    #   (exact-pinned per pi supply-chain rules)
  pi-version.txt                   # pinned pi-coding-agent version (single line,
                                    #   must match package.json)
  bin/
    skill-pi.js                    # THE CLI: launcher — (1) checks the pinned
                                    #   pi-coding-agent in ~/.skill-pi/node_modules
                                    #   matches pi-version.txt, (2) sets
                                    #   PI_CODING_AGENT_DIR=~/.skill-pi, (3)
                                    #   prepares the cwd shim
                                    #   (~/.skill-pi/cwd/<project>/package.json +
                                    #   symlinks to the real root) and ensures
                                    #   <project>/.skill-pi/ is materialized,
                                    #   (4) chdir into the shim and execs pi
                                    #   with process.argv forwarded
  mcp.json                         # MCP servers for skill-pi: mnemonic (stdio)
                                    #   + backlog + fluxcd + codescene + trivy as
                                    #   needed — lands in BOTH the global
                                    #   (~/.skill-pi/) and project
                                    #   (.skill-pi/) layers
  settings.json                    # pi settings (package sources, defaults);
                                    #   model/provider left overridable (CTX proxy)
  system/SYSTEM.md                 # skillgrid system prompt (persona, rules,
                                    #   zone discipline, commit contract)
  system/APPEND_SYSTEM.md          # optional append block
  extensions/                      # TS extensions (jiti-loaded)
    mnemonic-memory.ts             # mem_save / session hooks — ports
                                    #   plugins/opencode/mnemonic-memory.ts
    mnemonic-codeindex.ts          # code_status/code_index hooks — ports
                                    #   plugins/opencode/mnemonic-codeindex.ts
    skillgrid-events.ts            # tool-call event capture — ports
                                    #   plugins/opencode/skillgrid-events.ts
    skillgrid-squad.ts             # task squad — ports
                                    #   plugins/opencode/skillgrid-squad.ts
    skillgrid-compaction.ts        # compaction summary — ports
                                    #   plugins/opencode/skillgrid-compaction.ts
  skills/                          # skillgrid skills packaged for pi discovery
                                    #   (sync of .agents/skills/ subset;
                                    #   pi reads <configDir>/skills/, not
                                    #   .agents/skills/)
  prompts/                         # pi prompt templates (slash commands)
  themes/                          # optional
  install/
    install.md                     # installer instructions (consumed by
                                    #   skillgrid-cli install / manual steps)
    sync.sh                        # sync skill-pi/ -> ~/.skill-pi/, npm ci the
                                    #   pinned pi-coding-agent, build
                                    #   ~/.skill-pi/ + the per-project shim +
                                    #   <project>/.skill-pi/, put the `skill-pi`
                                    #   bin on PATH
```

Machine state (NOT committed, owned by the installer + launcher):

```
~/.skill-pi/                       # skill-pi home — PI_CODING_AGENT_DIR points HERE
  node_modules/                    # pinned @earendil-works/pi-coding-agent
                                    #   (npm ci --ignore-scripts per pi rules)
  pi-version.txt
  bin/skill-pi.js -> linked as `skill-pi` on PATH
  mcp.json  settings.json  models.json  auth.json  mcp-auth.json
  trust.json                       # trust decisions, keyed by shim realpath
  extensions/  themes/  bin/  tmp/
  sessions/                        # <encoded-shim-cwd>/ per project
  cwd/
    <project>/                     # cwd shim per project
      package.json                 # piConfig: { name: "skill-pi",
                                    #   configDir: ".skill-pi" }
      .git -> real-root/.git
      .skill-pi -> real-root/.skill-pi
      <other entries> -> real-root/<entry>

<project-root>/.skill-pi/          # project config — materialized by the
                                    #   installer/launcher, gitignored per repo
  mcp.json  settings.json  SYSTEM.md  APPEND_SYSTEM.md
  extensions/  skills/  prompts/  themes/
  overrides/                       # optional user edits, applied at launch
```

`~/.pi/` and `<project-root>/.pi/` are **never touched** by skill-pi — a
standalone pi install keeps its own paths intact.

**The `skill-pi` command.** One entrypoint, one job: run Pi with skillgrid
already loaded, from skill-pi's own config paths. `bin/skill-pi.js` — (1)
checks `~/.skill-pi/pi-version.txt` matches the installed package, (2) ensures
the project's cwd shim and `<project-root>/.skill-pi/` are synced (re-running
the sync step quietly if drifted), (3) sets `PI_CODING_AGENT_DIR`, chdir's into
the shim, and execs the real pi entrypoint forwarding `process.argv` unchanged
so `skill-pi --help`, `skill-pi mcp list`, etc. behave exactly like `pi`. No
skill-pi-specific flags in v1 — customization is carried by resources, not
flags (matching how the opencode/kilo plugins customize by config, not CLI).

### Design decisions (to lock during brainstorming/interview)

1. **Source location: top-level `skill-pi/`** — matches the existing
   top-level module convention (`mnemonic/`, `skillgrid-cli/`, `skillgrid-ui/`)
   and keeps skill-pi versionable as its own `package.json` unit. Rejected:
   `plugins/pi/` (it is not just plugins — it carries skills, system prompt,
   prompts, install logic, and its own package) and `.skillgrid/skill-pi/`
   (that zone is spec/state, not shipped source).
2. **No Pi monorepo vendoring.** Pin the version (`pi-version.txt`), reference
   the published `@earendil-works/pi-coding-agent` package. Pi ships its own
   supply-chain hardening (pinned deps, shrinkwrap) [1]; re-vendoring would
   fight it.
3. **Port, don't duplicate, the opencode plugins.** `plugins/opencode/*.ts`
   (mnemonic-memory, mnemonic-codeindex, skillgrid-events, skillgrid-squad,
   skillgrid-compaction) are the behavior baseline. skill-pi extensions port
   that behavior to Pi's `ExtensionAPI`. The opencode copies stay the
   opencode-harness source of truth; if a port drifts, the drift is reviewed
   (mirrors how the opencode/kilo plugin sets already diverge per harness).
4. **Skills: sync, don't copy.** Pi discovers skills from
   `<configDir>/skills/` (project) and global locations [2]; skillgrid skills
   live in `.agents/skills/`. The installer's sync step copies/symlinks the
   `.agents/skills/` subset that applies to pi into the skill-pi skills
   location. Keeping `.agents/skills/` as the single skill source avoids a
   second catalog to drift (the GSD finding warned against a 70+-skill eager
   catalog [7]).
5. **CTX proxy compatibility.** `settings.json`/`mcp.json` must not hardcode a
   provider base URL; `mnemonic ctx wrap pi` (opt-in, post-CTX-landing) injects
   the loopback :8787 base URL [4].
6. **Config paths: `~/.skill-pi/` (global) + `<project>/.skill-pi/`
   (project).** skill-pi never reads or writes `~/.pi/` or `<project>/.pi/`.
   Global layer: `PI_CODING_AGENT_DIR=~/.skill-pi` (pi's supported relocation
   mechanism [1]). Project layer: cwd shim carrying a `package.json` with
   `piConfig.configDir = ".skill-pi"`, so pi's `join(cwd, CONFIG_DIR_NAME)`
   resolves through a symlink to `<project>/.skill-pi/`. This is a **correction
   to the earlier drafts**, which (a) had the installer merge into
   `~/.pi/agent/`, (b) generated `<project>/.pi/` in the repo, and (c) used a
   `~/.skill-pi/cwd/<project>/.pi/` layout — all rejected: the project config
   must sit at the project root under its own name.
7. **Installer ownership + CLI.** Extend `skillgrid-cli` install/sync to
   (a) copy `skill-pi/` → `~/.skill-pi/` and `npm ci` the pinned
   `pi-coding-agent` there, (b) build `~/.skill-pi/` from the committed source,
   (c) materialize `<project-root>/.skill-pi/` + the per-project cwd shim
   lazily (the launcher does this on first run per project; the installer only
   needs to do it for the current repo at install time), (d) put the `skill-pi`
   bin on PATH. No `~/.pi/` state, no `.pi/` written into repos.
8. **Trust model.** Pi loads project resources only when trusted [2]; the
   trust store is `~/.skill-pi/trust.json` keyed by the **shim's canonicalized
   path** (realpath, not the real project path) [1]. skill-pi's `install.md`
   documents the trust prompt the first run shows, and that re-trusting happens
   per shim.

### Constraints that must hold

- **Serial development / one change at a time** — skill-pi structure lands as
  one change; the CTX proxy wiring is a separate later change [6][4].
- **No new dependencies without an ADR** — the `@earendil-works/pi-*` npm
  packages are a new dependency surface; they need an ADR row before the
  `package.json` devDeps land (the `yaml` npm package is the precedent [6]).
- **Spec-zone commits before code-zone** — this `findings.md` + the eventual
  `briefing.md`/`tasks.md` commit before `skill-pi/` source commits [6].
- **Conventional commits, no AI attribution** [6].
- **Pi runs with the launching user's permissions** (no built-in permission
  system) [1] — same trust posture as the other agents skillgrid wires; if a
  sandbox is wanted it is a containerization pattern from Pi's docs [1], not a
  skillgrid-owned seam in v1. **Updated by ADR-0037:** skill-pi v1 includes a
  permission extension (3-tier config, Tree-sitter bash matching, tool hiding,
  diff preview, hybrid approvals) — the "no built-in permission system"
  constraint is a description of Pi's core, not a lock on skill-pi's
  extension surface.

## Design — Permission Extension (ADR-0037)

### Reference comparison

Two reference implementations were evaluated:

| Dimension | JayGhiya/pi-opencode-permissions-and-preview | gotgenes/pi-permission-system |
|-----------|----------------------------------------------|-------------------------------|
| **Config model** | `allow`/`deny`/`ask` rules, 3-tier (global + project + local) | `allow`/`ask`/`deny` rules, centralized deterministic gates |
| **Bash matching** | Tree-sitter AST extraction (pipes, subshells, command substitution) | Wildcard patterns |
| **Tool visibility** | Block at call time (`tool_call` → `{ block: true }`) | Hide pre-start (`exposure: 'hidden'`) |
| **Edit preview** | `@pierre/diffs` (compact preview + review mode with `Ctrl+F`, `PgUp`/`PgDn`) | No preview; plain confirm |
| **Approval flow** | In-prompt: allow once / allow always (persisted) / allow for session / reject / reject with feedback | Pre-configured only; no in-prompt one-off |
| **Rule precedence** | `deny → persisted → session → ask → allow → defaultMode` | Centralized deterministic (deny wins) |
| **Sensitive paths** | Via rule patterns | Explicit sensitive-path rules, symlink-resolved |
| **Subagent prompts** | Not covered | Forwards subagent prompts to parent's permission context |
| **Events** | Not covered | Broadcasts `permissions:ui_prompt` / `permissions:decision` |
| **Fail mode** | Fails open (no rule → allow) | Fails closed (no rule → deny) |

### skill-pi design

skill-pi takes the best of both:

**Config: 3-tier (JayGhiya model)**

| Tier | Path | Scope | Git |
|------|------|-------|-----|
| Global | `~/.skill-pi/permission.settings.json` | Per-user defaults across all projects | Not in repo |
| Project | `<project>/.skill-pi/permission.settings.json` | Per-project rules | Committed |
| Local | `<project>/.skill-pi/permission.settings.local.json` | Per-user, per-project overrides | Gitignored |

Precedence: local overrides project overrides global. Within a tier, rule precedence is `deny → ask → allow` (first match wins). The local tier is essential: a developer on a shared machine needs per-user overrides without touching committed files.

**Bash matching: Tree-sitter AST (JayGhiya model)**

Commands are parsed with Tree-sitter's bash grammar before matching. This handles:
- Pipes: `git status | grep error` → matches `git status` AND `grep error`
- Subshells: `$(rm -rf /tmp/build)` → matches `rm -rf /tmp/build`
- Command substitution: ``echo `date` `` → matches `date`
- Compound commands: `npm install && npm test` → matches both

On parse failure (malformed command), the extension falls back to wildcard matching (fail open: treat the whole command as a single token). The Tree-sitter dependency is already in the project's dependency surface (the mnemonic code index's `Extractor` term in `02-technical-terms.md`); this ADR records it as a new dependency for the skill-pi package specifically.

**Tool visibility: hide pre-start (gotgenes model)**

At session start, the extension evaluates all rules against the tool registry:
- Tools matching a `deny` rule → `exposure: 'hidden'` (agent never sees them)
- Tools matching an `ask` rule → `exposure: 'hidden'` (agent never sees them; when the agent needs one, the user can re-enable it via the approval flow)
- Tools matching an `allow` rule → `exposure: 'visible'` (default)

This saves LLM turns: the agent doesn't waste calls probing tools it would always be denied. The `tool_call` block remains as a fallback for mid-session rule changes (a tool that was visible at start but whose rule changed).

**Edit preview: diff UI (JayGhiya model)**

For `edit` and `write` tool calls that are not pre-allowed:
1. The extension intercepts the `tool_call`
2. It renders a unified diff (via `@pierre/diffs`) showing the change
3. The user approves or rejects

For new-file writes (no existing content), the preview shows the full content as additions. The `@pierre/diffs` dependency is recorded in ADR-0037.

**Approvals: hybrid (both models)**

Pre-configured rules handle the 80% case. In-prompt approvals handle the long tail:

When a tool call matches an `ask` rule (or no rule at all, depending on `defaultMode`):

```
┌─────────────────────────────────────────────────────────┐
│ skill-pi wants to run:                                  │
│   npm install                                           │
│                                                         │
│   [Allow once]  [Allow always]  [Allow for session]     │
│   [Reject]  [Reject with feedback...]                   │
└─────────────────────────────────────────────────────────┘
```

- **Allow once**: in-memory, one call only
- **Allow always**: persisted to the local tier config file (`.skill-pi/permission.settings.local.json`)
- **Allow for session**: in-memory, survives until session end
- **Reject**: tool call blocked, agent sees "Permission denied"
- **Reject with feedback**: tool call blocked, user types a reason, agent sees "Permission denied: <reason>" — the LLM can adjust its approach

**Default mode: fail closed**

When no rule matches a tool call, the default is `ask` (prompt the user). This is the gotgenes model — safer than JayGhiya's fail-open for a coding agent that makes many file edits. The user can set `defaultMode: "allow"` in the global tier if they want a looser posture.

**Config schema**

```json
{
  "defaultMode": "ask",
  "rules": [
    { "tool": "read", "effect": "allow" },
    { "tool": "write", "effect": "ask" },
    { "tool": "edit", "effect": "ask" },
    { "tool": "bash", "pattern": "git *", "effect": "allow" },
    { "tool": "bash", "pattern": "npm install *", "effect": "allow" },
    { "tool": "bash", "pattern": "rm *", "effect": "ask" },
    { "tool": "bash", "pattern": "rm -rf *", "effect": "deny" },
    { "tool": "bash", "pattern": "curl *", "effect": "ask" },
    { "tool": "mcp__*", "effect": "ask" }
  ]
}
```

- `tool`: exact tool name or glob pattern (`mcp__*` matches all MCP tools)
- `pattern`: (bash only) command pattern — wildcard for the fallback path, Tree-sitter AST for the primary path
- `effect`: `allow` | `ask` | `deny`

### Extension implementation notes

The permission extension is a Pi extension loaded via `skill-pi`'s project config-dir (`<project>/.skill-pi/extensions/`):

- **`pi.on("tool_call", handler)`**: the main gate. Evaluates rules, hides/shows tools, renders diffs, prompts for approval. Returns `{ block: true, reason }` on denial.
- **`pi.on("tool_result", handler)`**: logs decisions, updates session-scoped approval state.
- **Pre-start tool hiding**: done via the extension's initialization phase (before the first model request), setting `exposure: 'hidden'` on deny/ask tools.
- **Config loading**: reads all 3 tiers at startup, merges with local > project > global precedence.
- **Subagent prompts**: forwards subagent tool calls to the parent's permission context (gotgenes pattern) — a subagent's `bash` call is gated by the same rules as the parent's.

### Dependencies (ADR-0037)

| Dependency | Purpose | Status |
|------------|---------|--------|
| `tree-sitter` + `tree-sitter-bash` | AST bash command parsing | Already in project (mnemonic code index); new for skill-pi package |
| `@pierre/diffs` | Diff preview UI for edits | New for skill-pi package |

### What this does NOT do

- **No OS-level sandboxing.** The permission layer is a rule engine, not a container. A `bash` tool call that is `allow`-ed runs with the launching user's full permissions. If sandboxing is wanted, it is a containerization pattern from Pi's docs [1], separate from this extension.
- **No network permission model.** v1 does not gate `curl`/`wget` by destination. The `curl *` pattern in the example config is a manual rule, not a built-in network-access check.
- **No path-based rules in v1.** The gotgenes extension has explicit sensitive-path rules (e.g., `~/.ssh/`, `/etc/`). skill-pi v1 relies on tool-level rules (`write` to a path is gated by the `write` rule, not by the specific path). Path-based rules are a v2 candidate.

## Omegon-Pi Learnings (skillgrid lens: mnemonic memory integration)

> **Reference:** [styrene-lab/omegon-pi](https://github.com/styrene-lab/omegon-pi) — an
> opinionated pi distribution (21 extensions, 12 skills, 4 prompts). Deprecated in
> favor of a Rust rewrite; the TS surface is the source, not the destination.
> **How to read this section:** the primary lens is **skillgrid's mnemonic memory
> integration** — how each omegon mechanism maps onto the mnemonic MCP surface
> (`mem_*`, `code_*`, `web_*`, `session_events`, `mem_inject_session`) that the
> skill-pi extension ports wire into pi. Secondary mappings cover launcher,
> bootstrap, skills, SDD, and parallel work. "Steal" = adopt for skill-pi v1.
> "Note" = record the seam for a later change. "Skip" = skillgrid's existing
> approach is correct.

### Mnemonic memory integration (primary)

> skill-pi's memory is not a pi extension — it is the **mnemonic MCP server**
> (SQLite + FTS5 + optional ONNX embeddings) exposed through three extension
> ports: `mnemonic-memory.ts` (observation lifecycle), `mnemonic-codeindex.ts`
> (code intelligence), and `skillgrid-events.ts` (session event capture).
> Omegon's `project-memory` extension is the closest analog: a single pi
> extension that owns the full memory lifecycle. The table below maps each
> omegon memory mechanism to the **mnemonic tool or extension port** that
> already covers it, and flags the seams that need building.

| # | Omegon mechanism | Mnemonic equivalent | Extension port | Verdict |
|---|------------------|---------------------|----------------|---------|
| M1 | **11 memory tools** (`memory_store`, `memory_recall`, `memory_query`, `memory_supersede`, `memory_archive`, `memory_connect`, `memory_compact`, `memory_episodes`, `memory_focus`, `memory_release`, `memory_search_archive`). | `mem_save`, `mem_search`, `mem_get_observation`, `mem_update`, `mem_delete`, `mem_compare`, `mem_lifecycle`, `mem_session_summary`, `mem_pin`/`mem_unpin`. The MCP surface is richer (18+ tools vs omegon's 11). | `mnemonic-memory.ts` registers these in pi's tool registry. | **Already covered** — no new tools. The MCP surface supersedes omegon's. |
| M2 | **Semantic retrieval** (Ollama `qwen3-embedding` → FTS5 fallback). | `mnemonic` hybrid search: `mem_search` / `mem_inject_session` / `mnemonic_hybrid_search` fuse vector (ONNX `all-MiniLM-L6-v2` or external) + FTS5 via RRF. Code side: `code_search` (BM25) + `code_semantic_search` (vector) + `code_hybrid_search`. | `mnemonic-memory.ts` (observation search), `mnemonic-codeindex.ts` (code search). | **Already covered** — same architecture, different embedder. No Ollama. |
| M3 | **Background extraction** (auto-discovers facts from tool output). | `mem_save` with `infer: true` (auto-inferred `type` + `topic_key`). `session_events` table captures every tool call; `mem_inject_session` retrieves relevant observations by query. The `mnemonic-second-brain` skill drives the agent to save on triggers. | `skillgrid-events.ts` (event capture), `mnemonic-memory.ts` (save with infer). | **Already covered** — events + `mem_save.infer`. No new extension. |
| M4 | **Episodic memory** (session narratives at shutdown). | `mem_session_summary` — mandated by the Engram protocol (AGENTS.md § SESSION CLOSE PROTOCOL). The summary is structured: Goal, Discoveries, Accomplished, Next Steps, Relevant Files. | `skillgrid-compaction.ts` (triggers summary on pi's compaction event). | **Already covered** — Engram protocol + compaction extension. |
| M5 | **Directive minds** (zero-copy fork with parent-chain inheritance; subagent reads parent's memory, writes its own, archives discoveries back). | `owner` field on observations + `mem_search` scoping by `project` + `reader_owner` + `mem_inject_session` by `query`. A subagent gets a scoped `session_id`; its `mem_save` calls carry that owner. **Gap:** zero-copy fork with parent-chain inheritance (subagent sees own writes + parent's reads, not parent's writes) is not yet in mnemonic. | `skillgrid-squad.ts` (sets subagent `owner` to task ID). | **Partial** — ownership + scoping exists. **Note** v2: add `mem_fork` (scoped view inheriting parent's read set). v1: subagent shares parent's `session_id` (shared view). |
| M6 | **Global knowledge base** (cross-project facts at `~/.pi/memory/global.db`). | `scope: global` on `mem_save`; `mem_search` with `all_projects: true` spans every store. `~/.skillgrid/` data dir = the global store. | `mnemonic-memory.ts` (saves with `scope: global`). | **Already covered** — `scope: global` + `all_projects: true`. |
| M7 | **Git sync** (exports memory to `facts.jsonl`; `.gitattributes` `merge=union`; `*.db` gitignored, JSONL committed). | `mnemonic` has no built-in JSONL export. The equivalent: a `/memory-sync` command that dumps `~/.skillgrid/` observations to `<project>/.skill-pi/memory/facts.jsonl`. The SQLite DB stays gitignored; the JSONL is the versioned, diff-able artifact in PRs. | **New:** `skill-pi /memory-sync` prompt template + a small export script. `skillgrid-events.ts` can trigger it on session end. | **Steal** — add `/memory-sync` + `facts.jsonl` with `merge=union`. Gives skill-pi a commit-able memory snapshot that survives machine migration. |
| M8 | **Auto-compact** (context pressure → automatic compaction). | Pi's native compaction + `skillgrid-compaction.ts` wiring `mem_session_summary` into the compaction event. | `skillgrid-compaction.ts`. | **Already covered** — compaction extension port. |
| M9 | **Session log** (append-only structured tracking). | `session_events` table — append-only, structured, queryable via `mem_query_events` / `mem_export_events`. | `skillgrid-events.ts`. | **Already covered** — events extension. |
| M10 | **Web research cache** (omegon caches Context7/Exa/DeepWiki results to avoid re-fetching). | `mnemonic_web_cache_save` / `mnemonic_web_cache_lookup` / `mnemonic_web_cache_search` — the exact equivalent: cache remote MCP research results with TTL, dedup by source+query, FTS5 search over cached snapshots. | `mnemonic-memory.ts` (or `mcp-bridge.ts` if the web tools are MCP-registered). | **Already covered** — `web_cache_*` tools. No Ollama or separate cache extension needed. |

**Mnemonic integration summary for skill-pi v1:**

- **3 extension ports** wire mnemonic into pi: `mnemonic-memory.ts` (observation CRUD + search + web cache), `mnemonic-codeindex.ts` (code search + index + explore), `skillgrid-events.ts` (session event capture + compaction trigger).
- **1 new command:** `/memory-sync` (M7) — export observations to committed `facts.jsonl`.
- **1 v2 gap:** `mem_fork` (M5) — zero-copy scoped view for subagent directive minds.
- **0 new dependencies** — mnemonic is the existing skillgrid MCP server; skill-pi just registers its tools in pi's registry via the extension ports.

### Launcher

| # | Omegon mechanism | skillgrid mapping | Verdict |
|---|------------------|-------------------|---------|
| L1 | **Spawn + restart loop** (`bin/omegon-pi.mjs`): spawns pi as a child, intercepts exit code 75 → re-spawn. `/update` and `/restart` signal restart via `process.exit(75)`. | skill-pi's `bin/skill-pi.js` currently `chdir` + `exec`s pi. Spawn + restart-loop gives `/update` + `/restart` for free. | **Steal** — change launcher to spawn+restart. Add `skill-pi /update` and `skill-pi /restart`. |
| L2 | **`PI_SKIP_VERSION_CHECK=1`** + quiet-startup `settings.json`. | skill-pi is branded — pi's version banner is wrong on every launch. | **Steal** — set `PI_SKIP_VERSION_CHECK=1` + quiet settings before spawn. |
| L3 | **`--no-skills --no-prompt-templates --no-themes`** + `--extension <root>`. Suppresses pi auto-discovery; distribution is sole resource authority. | skill-pi's resources come from committed `skill-pi/` source. Without `--no-*`, pi scans `~/.pi/agent/skills/` + other packages → contamination. | **Steal** — inject `--no-skills --no-prompt-templates --no-themes --extension <skill-pi-root>`. |
| L4 | **`--where` diagnostic flag**. | skill-pi's shim + config-dir resolution is the hardest part to debug. | **Steal** — `skill-pi --where` prints `{ skillPiRoot, piCli, agentDir, stateDir, shimDir, projectRoot, configDir, executable }`. |
| L5 | **`bundleDependencies`** (pi packages inside the npm tarball). | skill-pi currently `npm ci`s into `~/.skill-pi/node_modules`. `bundleDependencies` makes `npm install -g skill-pi` self-contained. | **Steal** — switch to `bundleDependencies` in `skill-pi/package.json`. |
| L6 | **Self-referential package purge**. | Filter `settings.json.packages` for `skill-pi` / `skillgrid` patterns. | **Steal** — one-line filter. |

### Bootstrap / first-run

| # | Omegon mechanism | skillgrid mapping | Verdict |
|---|------------------|-------------------|---------|
| B1 | **Bootstrap with capability matrix** (Ready / Partial / Not-available per feature). | `install/sync.sh` should check: hard prereqs (node 20+, npm, git), skillgrid checks (mnemonic MCP reachable, `skillgrid-cli` on PATH, `.skillgrid/config.yaml` present), optional (CTX proxy, Ollama for semantic memory). Matrix rows: **Mnemonic Memory, Code Index, CTX Proxy, Permission Extension, Skills Sync**. | **Steal** — upgrade `sync.sh` to bootstrap with capability matrix. |
| B2 | **Secrets in env vars AND `secrets.json`** (shell-command recipes, `!`-prefixed). | skill-pi's MCP servers (mnemonic, backlog, fluxcd, codescene, trivy) should read from `~/.skill-pi/secrets.json` recipes. Bootstrap checks both: env var first, then `grep -qE` in secrets.json. | **Steal** — add secrets.json recipe support. **New seam.** |
| B3 | **Legacy state migration** (`~/.pi/agent/` → agent dir). | A user with `~/.pi/agent/` may want to migrate auth/sessions to `~/.skill-pi/`. | **Note** — v2 `/migrate` command. |

### Skills / prompts / extensions

| # | Omegon mechanism | skillgrid mapping | Verdict |
|---|------------------|-------------------|---------|
| S1 | **12 skills** (loaded on-demand). | skillgrid has **37 skills** in `.agents/skills/` — a superset. Sync the relevant subset. | **Already covered** — sync script is the mechanism. |
| S2 | **4 prompt templates** (`new-repo`, `init`, `status`, `oci-login`). | `skill-pi/prompts/`: `init` → `resume` skill, `status` → `using-skillgrid`, `new-repo` → `onboarding`. | **Note** — author 3-4 prompt templates wrapping skillgrid skills. |
| S3 | **Content-hash guard** (prevents clobbering user edits on re-sync). | Launcher's sync step: hash-compare user-edited files; skip if user edited, warn if drifted. | **Steal** — answers Q7 (shim rebuild cadence). |
| S4 | **`mcp-bridge`**: registers external MCP servers as native pi tools (`mcp__` prefix in tool registry). | Prerequisite for ADR-0037's `mcp__*` permission glob. The `mnemonic-memory.ts` extension (or dedicated `mcp-bridge.ts`) must register MCP tools in pi's registry. | **Steal** — **prerequisite for ADR-0037**. |

### Spec-driven development

| # | Omegon mechanism | skillgrid mapping | Verdict |
|---|------------------|-------------------|---------|
| D1 | **OpenSpec** (proposal → specs → design → tasks; GWT scenarios are source of truth). | skillgrid's 9-phase workflow (`brainstorming` → … → `reflect`) is the equivalent, richer. | **Already covered**. |
| D2 | **OpenAPI 3.1 derivation from GWT**. | `acceptance-test-authoring` skill derives BDD tests; OpenAPI derivation is a v2 enhancement. | **Note** — v2. |
| D3 | **Design Tree** (persistent design docs, `ready`/`blocked` actions). | `.skillgrid/specs/` + `state.yaml` + `backlog` MCP (`ready: true` filter). | **Already covered**. |

### Parallel task decomposition

| # | Omegon mechanism | skillgrid mapping | Verdict |
|---|------------------|-------------------|---------|
| P1 | **Cleave** (parallel decomposition, dependency-ordered waves, isolated worktrees). | `parallel-execution` + `isolated-workspace` + `backlog` task dependencies. `skillgrid-squad.ts` dispatches. | **Already covered**. |
| P2 | **Skill-aware dispatch** (file-scope skill matching for children). | Skills are already file-context-matched. Enhancement: squad matches task file paths → skill triggers. | **Note** — v2 enhancement to `skillgrid-squad.ts`. |
| P3 | **Model tier routing** (7 tiers, per-child model selection). | CTX proxy routes to configured model. Tier routing = cost optimization. | **Note** — v2. Requires CTX per-task model override. |
| P4 | **Review loop** (severity-gated fix iterations). | `requesting-code-review` + `receiving-code-review` + `qa` skills. | **Already covered**. |

### Inference / models

| # | Omegon mechanism | skillgrid mapping | Verdict |
|---|------------------|-------------------|---------|
| I1 | **Effort tiers** (7 tiers, `/effort` switch). | Pi's native model selection + CTX proxy base-URL override. | **Note** — v2. |
| I2 | **Local inference via Ollama**. | skillgrid uses ONNX embeddings for memory; no Ollama. | **Skip**. |
| I3 | **Offline driver** (cloud → local on connectivity loss). | CTX proxy config, not a skill-pi extension. | **Note** — CTX proxy feature. |

### Quality of life

| # | Omegon mechanism | skillgrid mapping | Verdict |
|---|------------------|-------------------|---------|
| Q1 | **`chronos`** (authoritative date/time injection). | One-line extension. | **Steal** — add `chronos.ts`. |
| Q2 | **`tool-profile`** (named tool profiles, `/profile` switch). | Permission extension already hides tools. Named profiles = v2 enhancement. | **Note** — v2. |
| Q3 | **`view`** (inline image/PDF viewer). | Pi's native file-read is sufficient for v1. | **Skip**. |
| Q4 | **`secrets`** (env vars, shell commands, keychains). | Covered by B2. | **Already covered**. |
| Q5 | **`01-auth`** (multi-platform auth diagnostics). | v2 prompt template. | **Note**. |
| Q6 | **`ambiance`** (themed loading messages). | Cosmetic. | **Skip**. |
| Q7 | **Dashboard / Web UI**. | `skillgrid-ui/` is the full solution. | **Skip**. |
| Q8 | **Render** (image gen, D2, Excalidraw). | Out of scope. | **Skip**. |
| Q9 | **Web Search** (multi-provider with dedup). | `exa_web_search_exa` MCP tool already in the agent's tool set. | **Already covered**. |

### What NOT to copy

- **21 extensions is too many.** skill-pi v1 ships **7 extensions**: `mnemonic-memory.ts`, `mnemonic-codeindex.ts`, `skillgrid-events.ts`, `skillgrid-squad.ts`, `skillgrid-compaction.ts`, `permission.ts` (ADR-0037), `mcp-bridge.ts`.
- **Ollama dependency.** skillgrid uses its own mnemonic (SQLite + FTS5 + ONNX). Don't import Ollama.
- **Shared `~/.pi/agent/` state.** Omegon does NOT relocate. skill-pi's `~/.skill-pi/` relocation is correct (two agents, two homes).
- **`.pi/` committed at repo root (no shim).** skill-pi's shim renames `.pi` → `.skill-pi` via `piConfig.configDir`. **Simplification to verify** (Q8).

## Open Questions

- **Q1 (scope):** Is skill-pi v1 *just* the structure + opencode-behavior port
  (memory hooks, events, squad, compaction) with plain provider config, or
  does v1 also include the CTX proxy base-URL override (blocked on CTX
  landing)?
- **Q2 (skills subset):** Which of the 37 `.agents/skills/` does pi need?
  All, the same subset opencode gets, or a pi-curator pass? (Affects the sync
  script.)
- **Q3 (packaging):** Ship skill-pi as a `pi install`-able package
  (`package.json` `"pi"` key, npm/git/local install) or purely as
  installer-synced local resources? The `"pi"` key makes it distributable;
  the sync-only path keeps one distribution channel (the installer).
- **Q4 (system prompt):** Does `system/SYSTEM.md` reuse the AGENTS.md persona
  block (`gentle-ai:persona`/`engram-protocol`) verbatim, or is it a
  pi-tuned rewrite (Pi already reads AGENTS.md, so `SYSTEM.md` may only need
  the delta)?
- **Q5 (CLI shape):** `skill-pi` is a passthrough exec of pi (args forwarded
  verbatim, confirmed above) — or does v1 want `skill-pi init` / `skill-pi
  sync` / `skill-pi version` as real subcommands owned by the launcher before
  it hands off to pi?
- **Q6 (name/id):** Confirm the package/agent name is `skill-pi` (vs
  `skillgrid-pi`) and the directory `skill-pi/` (vs `agents/pi/`).
- **Q7 (shim + project-dir mechanics):** The shim is the load-bearing piece for
  the project layer. Open: (a) symlink rebuild cadence — on every launch vs
  mtime/content check of the real root? (per-entry symlinks vs a recursive
  bind-mount-free copy), (b) `<project>/.skill-pi/` — real directory
  (gitignored) or a symlink to a per-project copy under `~/.skill-pi/`?
  A real dir makes `overrides/` natural but adds a gitignore line per repo;
  a symlink keeps everything under `~/.skill-pi/` but the trust prompt's
  "install missing project packages" and any user file edits target a path
  outside the repo, (c) where do per-project overrides live —
  `<project>/.skill-pi/overrides/` (in-repo, committable) or
  `~/.skill-pi/cwd/<project>/overrides/` (per-user, not committable)?,
  (d) session cwd: pi stores sessions under
  the effective cwd — sessions will be attributed to the shim path, is that
  acceptable for `--continue`/`--resume` UX? [RESOLVED: sessions live under
  `~/.skill-pi/sessions/<encoded-shim-cwd>/` (session-manager.ts) and
  `--continue` resolves against the same shim path, so behavior is
  self-consistent; the only observable is the cosmetic folder name]
- **Q8 (pi version floor):** Does `piConfig.configDir` exist in the pinned pi
  version? Verified present in `main` (package.json + config.ts, v1.1.0) [1];
  if the pin is older than this field, the shim package.json trick needs a
  minimum-version constraint documented in `pi-version.txt`.

## References

1. earendil-works/pi — README (packages table, extension/package model,
   supply-chain hardening, containerization) and source, read directly this
   run (main branch):
   `packages/coding-agent/package.json` (`piConfig: { configDir: ".pi" }`,
   bin, exports),
   `src/config.ts` (`getPackageJsonPath` walks up from `process.argv[1]`;
   `CONFIG_DIR_NAME = pkg.piConfig?.configDir || ".pi"`;
   `PI_CODING_AGENT_DIR` / `PI_CODING_AGENT_SESSION_DIR`; `getAgentDir`),
   `src/core/settings-manager.ts` (project settings path =
   join(resolvePath(cwd), CONFIG_DIR_NAME, "settings.json")),
   `src/core/resource-loader.ts` (project mcp/SYSTEM.md/extensions/skills/
   prompts/themes at join(cwd, CONFIG_DIR_NAME); `findShadowedContextFile`),
   `src/core/footer-data-provider.ts` (`findGitPaths` — `.git` dir or file,
   upward walk),
   `src/core/trust-manager.ts` (`hasTrustRequiringProjectResources` checks
   join(cwd, CONFIG_DIR_NAME); trust store at `<agentDir>/trust.json` keyed by
   canonicalized cwd),
   `src/core/session-manager.ts` (default session dir = `<agentDir>/sessions/
   <encoded-cwd>/`; session cwd metadata + `sessionCwdMatches` filtering),
   `src/core/session-cwd.ts` (missing-cwd prompt on resume),
   `src/main.ts` (`process.cwd()` at startup),
   `src/cli/args.ts` (no project-dir flag; only `--session-dir`).
   https://github.com/earendil-works/pi
2. DeepWiki: earendil-works/pi — "Extension Loading, Packages, and Resource
   Discovery", "MCP Integration", "Session Management and Compaction",
   config locations (`~/.pi/agent/`, `.pi/`), `ExtensionAPI` surface,
   `AGENTS.md` loading, project trust.
   https://deepwiki.com/earendil-works/pi
3. Repo: `plugins/opencode/`, `plugins/cursor/`, `plugins/kilo/`,
   `agents/opencode/`, `agents/cursor/`, `git-hooks/`, `.agents/skills/`,
   `go.work`, `package.json` (read this run).
4. `.skillgrid/specs/2026-10-08-context-orchestrator/briefing.md` — CTX
   `wrap` opt-in agent list, proxy :8787, ADR-0033…0036.
5. `.skillgrid/ASSUMPTIONS.md` — component map (ADR-0004): Installer is the
   only writer of agent MCP config / `~/.skillgrid/`.
6. `AGENTS.md` — operational rules (Go 1.22+, no new deps without ADR, serial
   development, spec-before-code, conventional commits).
7. `.skillgrid/specs/2026-10-02-skillgrid-workflow-updates/findings.md` — GSD
   two-stage-router / catalog-drift findings (why skills stay a single source).
 8. `open-gsd/gsd-pi` — the precedent for an opinionated pi distribution: npm
    package `@opengsd/gsd-pi` with `piConfig: { name: "gsd", configDir: ".gsd" }`,
    `bin` → bootstrap scripts (not the pi entrypoint), pinned pi launched with
    GSD identity/workflow pre-loaded; global state in `~/.gsd/`, project state in
    `.gsd/`. https://github.com/open-gsd/gsd-pi (package.json + README read this
    run).
 9. `Gentleman-Programming/gentle-shell` — Pi-native harness with modular TUI
    architecture. Source read this run: `lib/shell-sidebar-layout.ts`
    (per-section digest cache, layout engine), `lib/shell-prompt.ts`
    (petal state machine, ODD phase labels), `lib/shell-bar.ts`
    (segment drop order, context gauge), `lib/shell-changes-view.ts`
    (two-pane accordion, event-driven capture), `lib/agents-view.ts` +
    `lib/agents-widget.ts` (task card, column degradation),
    `extensions/startup-banner.ts` (system stats, intro modes, animation
    policy), `@earendil-works/pi-tui` primitives (`Component`, `VStack`,
    `HStack`, `ScrollView`, `LayoutNode`, `TuiAltScreen`, `matchesKey`,
    `visibleWidth`).

## TUI Architecture (gentle-shell reference)

> **Reference:** `Gentleman-Programming/gentle-shell` — a Pi-native harness with
> a modular extension architecture. The TUI is the highest-leverage
> differentiator for skill-pi. All patterns below are portable; the only
> Pi-specific coupling is the `ExtensionAPI` lifecycle hooks
> (`session_start`, `tool_call`, `before_agent_start`), which we'd replace with
> our own transport hooks.

### TUI library: `@earendil-works/pi-tui`

| Primitive | Responsibility |
|-----------|---------------|
| `Component` | `render(width) → string[]`, `invalidate()`, optional `handleMouse()` |
| `VStack` / `HStack` | Layout primitives with `gap` and `align` |
| `ScrollView` | Wraps a component, `follow` mode, `overscroll`, `scrollbar: auto` |
| `LayoutNode` | Symbolic tree, `[NODE]` symbol method for custom layout |
| `TuiAltScreen` | Terminal management, input dispatch |
| `matchesKey` | Keyboard event matching |
| `visibleWidth` | ANSI-aware string width calculation |

### Layout model

- **Responsive breakpoint at 140 columns**: full layout (header + transcript + right rail) vs compact (bottom bar + agents widget)
- **Right rail**: fixed 50-column `ScrollView`, scrolls independently, contains Status → Changes → TODO cards
- **Header**: single always-visible row, identity left, live counters right
- **Bottom bar**: replaces Pi's 3-line footer with one dense segment rail

### Key TUI files (their structure)

| File | Responsibility |
|------|---------------|
| `lib/shell-bar.ts` | Status bar model + rendering |
| `lib/shell-sidebar.ts` | Sidebar part registration |
| `lib/shell-sidebar-layout.ts` | Layout engine, per-section cache |
| `lib/shell-prompt.ts` | Prompt editor frame + petal state machine |
| `lib/shell-changes-view.ts` | Two-pane changes viewer |
| `lib/agents-view.ts` | Full-terminal agents overlay |
| `lib/agents-widget.ts` | Persistent card above prompt |
| `extensions/startup-banner.ts` | Intro + runtime panel |

### Patterns worth porting directly

**1. Per-section digest cache.** Each rail card computes a `digest()` (component
state hash). The sidebar only re-renders a section when its digest changes.
Prevents a status counter update from redrawing the Changes or TODO cards.
Directly applicable to any multi-panel TUI.

**2. Petal state machine.** Three states: `IDLE` (static), `WORKING`
(animating through glyph sequence `✿ → ❀ → ❁ → ✾` with cycling tone), `QUEUED`
(warning tone). Plus **ODD phase labels** — the working label can display
`exploring…`, `implementing…` etc. for sub-step transparency. Cheap, high-signal
way to show what the agent is doing without a separate status line.

**3. Context gauge with tone escalation.** `▰▰▰▱▱▱` at accent (<80%), warning
(80%), error (95%). Simple, scannable, no tooltip needed.

**4. Cost formatting.** Two decimals if ≥$1, three decimals otherwise. Appends
`sub` for subscription models. Trivial but the "don't waste space" principle
applies everywhere.

**5. Segment drop order.** When width decreases, segments are dropped in a fixed
priority order: profile → effort → cwd/branch/dirty → ... with brand and
counters being last to go. Deterministic, testable, no "what fits" calculations.

**6. Theme system: `vars` + semantic roles.**
```json
{
  "vars": { "accent": "#F095C8", "border": "#444444" },
  "colors": { "accent": "accent", "border": "border" }
}
```
Themes define hex values in `vars`, then map semantic roles (`accent`,
`success`, `error`, `warning`, `muted`, `dim`, `text`, `selectedBg`,
`userMessageBg`, `toolDiffAdded`, `syntaxKeyword`, `thinkingLow`...) to those
vars. ~40 design tokens covering UI, messaging, tooling, markdown, diffs,
syntax, and thinking levels. Separation lets you ship new themes by changing
only hex values.

**7. Card-based rendering.** All structured content uses `renderCard` with
explicit tone (`CARD_TONE`). Consistent visual language across Status, Changes,
TODO, and notice cards.

**8. Startup banner with system stats.** Shows git branch, cwd, MCP server
count, background agent count, package count, extension count, skills loaded,
custom tool count, and version. Skipped for CLI subcommands. Three intro modes:
`full`, `minimal`, `skip` based on terminal dimensions. Animation policy:
`quality` / `performance` / `potato`.

**9. Changes viewer: two-pane, accordion, event-driven.**
- Left: worktree groups (accordion), per-file line counts
- Right: captured diff for selected file
- `j/k` navigate, `Enter/Space/Right` expand, `Left` collapse, `o` open in `$EDITOR`
- Mouse: click to select, wheel to scroll, hover to highlight
- **No git scan, no file reload** — reads only from the captured session model
- 64 KiB / 2000 line capture limit; binary or oversized shows "unavailable"

**10. Agents widget: fixed column card.** Above the prompt, shows active +
recently finished (60s TTL, 200 cap) subagent tasks. Right-aligned columns:
model · effort, tokens, cost, elapsed. Degrades gracefully by dropping columns
(task text → model → tokens → cost → elapsed last). Reads only `TaskRecord`
fields, no thread iteration per frame.

### What we'd need for skill-pi

Since skill-pi is our own opinionated Pi, we'd either:

1. **Use `@earendil-works/pi-tui`** if it's publicly available (it's not in the
   repo — it's a Pi ecosystem package), or
2. **Build our own minimal TUI layer** with the same primitives: `Component`,
   `VStack`, `HStack`, `ScrollView`, theme resolution, input dispatch

The key design decisions to make before starting:

- **Breakpoint**: where do we split full vs compact? (140 cols is their choice
  — worth testing)
- **Rail width**: 50 cols is their choice
- **Theme tokens**: we'd define our own ~40 semantic roles
- **State ownership**: they store `SidebarState` on the TUI terminal object;
  we'd need equivalent for our Go binary + embedded SPA architecture
- **Event-driven change capture**: requires hooking into the write/edit tool
  call layer, which we control in skillgrid

### TUI-specific file inventory (gentle-shell)

| File | Key exports |
|------|-------------|
| `lib/shell-sidebar-layout.ts` | `SidebarLayoutState`, `renderSidebarSection`, `computeSectionDigest` |
| `lib/shell-prompt.ts` | `PromptEditorFrame`, `PetalState`, `PetalPhase` |
| `lib/shell-bar.ts` | `StatusBarSegment`, `renderStatusBar`, `dropOrder` |
| `lib/shell-changes-view.ts` | `ChangesViewer`, `WorktreeGroup`, `FileDiff` |
| `lib/agents-view.ts` | `AgentsOverlay`, `TaskRecord` |
| `lib/agents-widget.ts` | `AgentsWidget`, `renderTaskCard` |
| `extensions/startup-banner.ts` | `StartupBanner`, `IntroMode`, `AnimationPolicy` |

## Context Extension: compact-adviser + pi-context-view + mnemonic

### Goal

Add a `skill-pi` context extension that combines:

1. **Compaction advising** from `kunchenguid/compact-adviser` — decide when to
   hint at or auto-run `/compact` based on a Jev two-question score.
2. **Context visualization** from `dimk90/pi-context-view` — inspect the system
   prompt, tool definitions, skills, and extension-injected messages.
3. **Mnemonic context injection** from skillgrid — use `mem_inject_session`,
   `ctx_query`, and the `/context` / `/context/compaction` HTTP endpoints to
   surface persistent project memory and session context.

### Reference 1: compact-adviser

`compact-adviser` is a Pi extension that advises when to compact. Core
mechanics:

- Hooks into Pi lifecycle events:
  - `agent_settled`, `turn_end`, `session_start`, `session_before_compact`,
    `session_compact`, etc.
- Maintains a score using a two-question Jev heuristic:
  1. Is the context window becoming crowded?
  2. Is the current session likely to keep growing?
- When the score crosses a threshold, it emits a hint or triggers `/compact`
  automatically (behavior configurable).

Relevant files:
- `packages/pi-extension/src/adviser.ts` — main entry, event wiring, score.

Takeaways for `skill-pi`:
- Reuse the same Pi extension event surface.
- Port the Jev score, but add a third signal: **mnemonic context pressure**
  (how much token budget `mem_inject_session` is consuming).

### Reference 2: pi-context-view

`pi-context-view` visualizes context usage and injections without adding
instructions to the model context. Key modules:

- `src/index.ts` — extension entry; registers the `/context` command.
- `src/capture.ts` — `InitialCaptureState`, `SilentProbeState`,
  `CompactionState`; captures system prompt, active tools, and injected
  messages at session start or via a silent probe.
- `src/measure.ts` — `analyzeSystemPrompt()` splits the prompt into
  base prompt, appended prompt, context files, skills, and tool contributions.
- `src/usage.ts` — `computeUsage()` classifies live messages into categories
  (user, agent text/thinking/tool calls, tool output, extensions, compacted
  data) and merges with the frozen snapshot.
- `src/model.ts` — data types: `InjectionItem`, `InitialSnapshot`,
  `ContextUsageSnapshot`, `UsageCategory`, `UsagePreviewEntry`.
- `src/ui/usage-view.ts`, `src/ui/injections-view.ts` — TUI renderers.

Key commands:
- `/context` or `/context usage` — open context usage visualization.
- `/context injections` — show hidden parts (system prompt, tool definitions,
  extension injections).

Takeaways for `skill-pi`:
- Port the usage-map + injections model.
- Add a **mnemonic layer** to the usage map: show token cost of
  `mem_inject_session` blocks and the `/context` endpoint response.
- Keep the extension passive — no extra instructions in the model context.

### Reference 3: skillgrid mnemonic context

Existing skillgrid context functions that the extension should expose:

- **MCP tools**
  - `mem_inject_session(query, max_tokens=2000, project?, all_projects?)` —
    hybrid retrieval of injectable observations, token-cost-annotated.
  - `mem_context(query?)` — recent session summaries.
  - `mem_save`, `mem_search`, `mem_get_observation`, etc.
- **HTTP endpoints**
  - `/context` — on-demand context injection for the current session.
  - `/context/compaction` — pre-compaction context summary / handoff.
- **CTX orchestrator** (`.skillgrid/specs/2026-10-08-context-orchestrator/`)
  - `wrap` opt-in agents, proxy `:8787`, ADR-0033…0036.

Integration points:
- The extension can call `mem_inject_session` (via MCP) or `GET /context`
  (via HTTP) to pull project memory into the view.
- `/context/compaction` can be used to seed a compact summary just before
  auto-compaction.

### Proposed `skill-pi` design

1. **Extension package**: `skill-pi-context` (or folded into the main
   `skill-pi` package) implementing a Pi extension.
2. **Commands**
   - `/skillgrid context` — open a unified context view.
   - `/skillgrid context usage` — usage map with mnemonic layer.
   - `/skillgrid context injections` — hidden context parts.
   - `/skillgrid context compact` — advise on / trigger compaction.
3. **Events**
   - `session_start` — capture system prompt/tools; optionally call
     `GET /context` for baseline mnemonic context.
   - `agent_settled` / `turn_end` — update Jev score; update usage map.
   - `session_before_compact` — optionally call `GET /context/compaction` and
     inject the result before compaction.
4. **Scoring**
   - Combine Jev two-question score with mnemonic token pressure:
     - `memory_pressure = mem_inject_session_tokens / max_tokens`
     - threshold crossed → hint/auto-compact.
5. **UI**
   - Reuse `pi-context-view` TUI patterns (`usage-view`, `injections-view`).
   - Add a "Mnemonic" category in the usage map:
     - `mem_inject_session` cost
     - `ctx_query` / `/context` endpoint cost
     - `/context/compaction` summary cost

### Open questions

- Should the extension be a separate npm package or part of `skill-pi`?
- Does `skill-pi` already expose an MCP bridge to `mem_*` tools, or does the
  extension need to call skillgrid HTTP directly?
- Is the Jev score the right compaction heuristic, or should it be replaced
  by a simpler token-threshold + mnemonic-pressure rule?
- Should `/skillgrid context` be the only new command, or should we also
  register `/context` (shadowing `pi-context-view`)?

### Next steps

1. Read `dimk90/pi-context-view` TUI code (`src/ui/usage-view.ts`,
   `src/ui/injections-view.ts`) to extract the view model.
2. Read `kunchenguid/compact-adviser` `adviser.ts` to port the Jev score.
3. Define the `skill-pi-context` package layout and extension entry.
4. Spike: wire `mem_inject_session` output into the usage map.

## Ticketing Extension: pi-tk-tickets (Backlog.md adapter)

### Reference: pi-tk-tickets

`ProbabilityEngineer/pi-tk-tickets` (MIT, v0.2.0) is a compact Pi extension
that wraps the `wedow/ticket` (tk) CLI into one model-visible `ticket` tool
plus a `/tickets` slash command. Design principles worth stealing:

- **Low prompt overhead.** The tool injects only a 1-line `promptSnippet` +
  2 `promptGuidelines` entries. Ticket lists are never auto-injected into the
  prompt; the agent calls the tool on demand.
- **Single compact tool.** One `ticket` tool with 9 actions (`ready`, `list`,
  `show`, `create`, `start`, `close`, `note`, `status`, `update`) instead of
  one tool per action. Reduces tool-registry bloat and LLM tool-selection cost.
- **Bundled dependency.** The `tk` bash script (42 KB, MIT, pinned to upstream
  commit `194b71a8`) is vendored in `vendor/wedow-ticket/`. No separate `tk`
  install required. Resolver: `TK_BIN` env → bundled file → `PATH` lookup.
- **Cross-repo support.** A `cwd` parameter on the tool (and `--cwd` / `-C` on
  the slash command) lets the agent manage tickets in another repository
  without `chdir`-ing the whole Pi session.
- **Native list/show without tk spawn.** `list` and `show` parse the markdown
  frontmatter directly (`findTicketsDir` + regex extract of `id`, `status`,
  `priority`, `title`). This avoids a `tk ls` / `tk show` process spawn for
  the most common read actions — faster, no bash dependency for reads.
- **Direct file mutation for `update`.** The `update` action reads the `.md`
  file, patches frontmatter fields (`status`, `type`, `priority`, `assignee`,
  `external-ref`, `tags`), rewrites title/description sections, and writes
  back. This bypasses `tk` entirely for field-level edits, giving the agent
  fine-grained control without a full `tk` re-create.
- **Output capping.** `MAX_OUTPUT = 40_000` chars. `ready` results are
  line-clamped to `limit` (default 20, max 100). Prevents a large backlog
  from blowing the tool-result context.

### Tk data model (wedow/ticket)

Tickets are markdown files with YAML frontmatter in `<repo>/.tickets/`:

```
.tickets/
  nw-5c46.md          # filename = ticket ID
```

Frontmatter fields: `id`, `status` (open|in_progress|closed), `priority`
(0–4, 0=highest), `type` (bug|feature|task|epic|chore), `assignee`,
`created`, `updated`, `tags`, `external-ref`, `parent`, `deps`. Body: `# Title`
+ description + `## Design` + `## Acceptance Criteria` + `## Notes`
(timestamped, appended by `tk add-note`).

Key tk commands used by the extension: `ready`, `create`, `start`, `close`,
`status`, `add-note`, `show`, `ls`. The extension does NOT use `dep` /
`dep tree` / `blocked` / `link` / `query` / `migrate-beads` — those remain
available via the bare `tk` CLI if needed but are not surfaced in the tool.

### skillgrid Backlog.md vs tk format comparison

| Dimension | Backlog.md (skillgrid) | tk (wedow/ticket) |
|-----------|------------------------|-------------------|
| Storage | `.backlog/tasks/*.md` (per-repo) | `.tickets/*.md` (per-repo) |
| ID format | `task-001` (zero-padded prefix) | `<prefix>-<hex>` (e.g. `nw-5c46`) |
| Statuses | 9: Draft, needs-triage, needs-info, ready-for-agent, ready-for-human, in-progress, done, blocked, wontfix | 3: open, in_progress, closed |
| Priority | high, medium, low | 0–4 numeric (0=highest) |
| Type | feature, bug, enhancement, refactor, docs, chore | bug, feature, task, epic, chore |
| Acceptance | `## Acceptance Criteria` (checkbox list) | `## Acceptance Criteria` (free text) |
| Dependencies | `[task-002]` in frontmatter | `deps: [nw-abc, nw-def]` |
| Milestone | frontmatter `milestone:` | Not supported (tags only) |
| Assignee | `assignee:` (array) | `assignee:` (single string, defaults to git user.name) |
| Notes | `## Notes` section | `## Notes` (timestamped via `add-note`) |
| Parent | `parentTaskId:` | `parent:` (ticket ID) |
| Modified files | `modifiedFiles:` (array) | Not supported |
| References | `references:` + `documentation:` (arrays) | `external-ref` (single) |

**Backlog.md is a strict superset of tk's data model.** Every tk field has a
Backlog.md equivalent (status maps 3→9, priority maps numeric→ordinal, type
maps 5→6 with near 1:1 overlap). The reverse is not true: Backlog.md has
milestones, modified-files, references, documentation, and 6 extra statuses
that tk lacks.

### Design decision: Backlog.md adapter, not tk

skillgrid's existing ticketing is Backlog.md (`.skillgrid/config.yaml`
`ticketing.type: backlogmd`, `backlogmd.dir: .backlog`). The `backlog` MCP
tools (`backlog_task_create`, `backlog_task_list`, `backlog_task_view`,
`backlog_task_edit`, `backlog_task_complete`, etc.) are already wired into
opencode, cursor, and kilo. skill-pi should **not** introduce a second ticket
format (tk's `.tickets/`) that would require a migration story and a parallel
set of MCP tools.

Instead, the skill-pi ticketing extension is a **Backlog.md adapter** that
provides the same compact single-tool + slash-command interface as
pi-tk-tickets, but backed by the `backlog` MCP tools (or direct file
operations on `.backlog/tasks/`).

**Why not just use `pi install npm:pi-tk-tickets`?** It would work for tk-native
projects, but skillgrid projects use Backlog.md. Running both would create
two ticket directories (`.tickets/` + `.backlog/`) in the same repo with no
sync between them. The adapter keeps one source of truth.

### skill-pi ticketing extension design

**File:** `skill-pi/extensions/tickets.ts`

**Interface (same shape as pi-tk-tickets, adapted for Backlog.md):**

- One `ticket` tool with 10 actions:
  - `ready` — list ready-for-agent tasks (all deps satisfied)
  - `list` — list tasks, optional `status` filter
  - `show` — full task detail (view + metadata)
  - `create` — create a task (title, description, status, priority, type,
    assignee, dependencies, milestone, acceptance criteria)
  - `start` — set status to `in-progress`
  - `complete` — set status to `done` (moves to completed folder)
  - `archive` — archive a task
  - `note` — append to task notes
  - `status` — set an arbitrary status (Draft, needs-triage, needs-info,
    ready-for-agent, ready-for-human, in-progress, done, blocked, wontfix)
  - `update` — field-level edit (title, description, status, priority, type,
    milestone, dependencies, labels, acceptance criteria, modified files)

- One `/tickets` slash command with the same action surface (for
  user-facing, non-LLM interaction):
  ```
  /tickets ready
  /tickets list [status]
  /tickets show <id>
  /tickets create <title>
  /tickets start <id>
  /tickets complete <id>
  /tickets status <id> <status>
  /tickets note <id> <text>
  ```

**Backed by `backlog` MCP tools.** The extension calls the MCP server
(stdio, already in skill-pi's `mcp.json`) rather than shelling out to a
bash script. Action → MCP tool mapping:

| Action | MCP tool |
|--------|----------|
| `ready` | `backlog_task_list` (`ready: true`) |
| `list` | `backlog_task_list` (`status` filter) |
| `show` | `backlog_task_view` |
| `create` | `backlog_task_create` |
| `start` | `backlog_task_edit` (`status: "in-progress"`) |
| `complete` | `backlog_task_complete` |
| `archive` | `backlog_task_archive` |
| `note` | `backlog_task_edit` (`notesAppend: [text]`) |
| `status` | `backlog_task_edit` (`status: <value>`) |
| `update` | `backlog_task_edit` (field-specific params) |

**Prompt overhead (same minimal pattern as pi-tk-tickets):**

```
promptSnippet: "Ticket routing: for non-trivial work, use ticket
  ready/show/create/start/complete to track backlog tasks."
promptGuidelines: [
  "For non-trivial work, check ready tickets or create/start/complete a task
   when useful.",
  "The ticket tool manages .backlog/tasks/ — it should not touch code unless
   the task requires it."
]
```

**Output capping:** same `MAX_OUTPUT = 40_000` char cap. `ready` results
line-clamped to `limit` (default 20, max 100).

**Cross-repo support:** the `cwd` parameter is less critical for skillgrid
than for tk (Backlog.md is per-project-root, and skillgrid's installer
already wires the `backlog` MCP server to the correct `.backlog/` dir).
Keep the parameter for consistency with the pi-tk-tickets interface, but it
resolves to the project root (walk up to `.git`) rather than an arbitrary
directory.

**Milestone support:** unlike tk, the `create` and `update` actions carry a
`milestone` parameter that maps directly to Backlog.md's `milestone:` field.
This is a skillgrid-specific extension point — `backlog_milestone_list` /
`backlog_milestone_add` MCP tools are available for the agent to manage
milestones if needed, but v1 does not surface them in the ticket tool (keep
the tool surface compact).

**Status mapping (Backlog.md 9 → tool surface):**

The tool exposes all 9 Backlog.md statuses in the `status` action. The
`ready` action uses Backlog.md's `ready: true` filter (all dependencies
satisfied/completed), not a status filter. `start` is sugar for
`status: "in-progress"`. `complete` is sugar for `backlog_task_complete`
(which also moves the file to the completed folder — a Backlog.md-specific
behavior tk lacks).

### What this does NOT do

- **No tk dependency.** The extension does not vendor or require the `tk`
  bash script. Backlog.md is the storage format; the `backlog` MCP server is
  the access layer. A project that wants tk can still `pi install
  npm:pi-tk-tickets` in parallel, but skill-pi does not ship it.
- **No auto-injection.** Ticket lists are never injected into the system
  prompt. The agent calls `ticket` when it needs ticket context (same as
  pi-tk-tickets).
- **No `dep tree` / `blocked` / `link` / `query` in v1.** Backlog.md has
  dependencies but no tree visualization. `backlog_task_list` with
  `ready: true` and `backlog_task_search` cover the common queries. These
  are v2 candidates.
- **No `migrate-tk` command.** If a project transitions from tk to
  Backlog.md, that is a one-time manual migration, not a skill-pi feature.

### Dependencies

| Dependency | Purpose | Status |
|------------|---------|--------|
| `backlog` MCP server (stdio) | Backlog.md CRUD | Already in skill-pi `mcp.json` (same server opencode/cursor/kilo use) |
| `typebox` (Type.Object) | Tool parameter schema | Already in skill-pi dependency surface (pi-coding-agent peer dep) |

### Open questions

Resolved 2026-10-09 (user confirmed both leans; recorded as ADR-0038):

- **Q9 (MCP vs direct file):** **MCP path.** The extension calls the
  `backlog` MCP tools over stdio — single source of truth for parsing; the
  MCP server is already a required part of skill-pi's tool set.
- **Q10 (tool naming):** **`ticket`.** Matches pi-tk-tickets, shorter, and
  universal — the tool manages tickets regardless of the underlying format.
