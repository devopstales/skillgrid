# Gryph Integration Design

Date: 2026-08-26
Status: approved (bounded, brainstorming)
Scope decision: **With policy** (hooks for both agents + default policy scaffolded and enabled)

## Summary

Integrate [gryph](https://github.com/safedep/gryph) — a local-only audit-trail and security-policy tool for AI coding agents — into the skillgrid installer. After `skillgrid install`, agents (Kilo + OpenCode) emit every tool call to gryph's local SQLite store, and gryph's YAML security policy (block/warn/guide) applies in real time.

## What gryph is

- Installs lightweight **hooks** into agents. For OpenCode the hook is a JS plugin at `~/.config/opencode/plugins/gryph.js` that bridges to the `gryph` binary (`gryph install --agent opencode` writes it; verified against `agent/opencode/hooks.go` in the gryph repo).
- The plugin invokes `gryph` **by name from PATH** at runtime (`agent/utils/command.go` returns the literal string `gryph`). PATH is therefore a requirement for hooks to work — skillgrid already prints PATH export lines after every install.
- Events land in a local SQLite DB (default retention 90 days). No cloud, no telemetry. Sensitive files are detected; content is never stored for them.
- Query surface: `gryph logs`, `gryph query`, `gryph session(s)`, `gryph stats`, `gryph export`, `gryph diff <event-id>`.
- Policy: YAML policy in gryph's global config dir (`gryph policy init` scaffolds Gryph's embedded example and **refuses to overwrite an existing one** — verified against `cli/policy.go`). Toggled with `gryph config set policy.enabled true`.
- Gryph adapters: claude-code, codex, cursor, gemini, opencode, openclaw, windsurf, pi-agent. **No adapter for Kilo** — that is why Kilo needs the plugin-file copy (the same bridge pattern skillgrid already uses for the engram plugin).
- Distribution: npm package `@safedep/gryph` (v0.7.0 at time of writing) whose postinstall downloads the platform binary. This works inside skillgrid's existing `npm install -g --prefix ~/.skillgrid/npm --dangerously-allow-all-scripts` flow (registry packages run scripts; same as `husky`).

## Current state of the repo (verified)

- `config.d/` is the single source of truth for installs; CLI code reads `tools.yaml` → npm install (registry and git packages split), `mcp.yaml` → JSON merge, `skills.yaml` → skills CLI, `AGENTS.md` → rules copy.
- `skillgrid-cli/cmd/install.go` runs the pipeline: agent selector → clone/sync → node → engram → npm → **gated per-tool steps** (`hasTool(tools.Tools, "agent-browser")` block at lines 88-96) → plugins/skills paths → skills → MCP merge → rules.
- `skillgrid-cli/cmd/steps.go` already contains the exact pattern needed for Kilo: copying an OpenCode-format plugin into `~/.config/kilo/plugins/` (engram bridge, lines 151-201).
- Conventions: no new npm packages outside `config.d/`, dry-run never executes and never writes, install-step failures are `logging.Warn` (never fatal), backup-before-write for agent configs, tests in `cmd/*_test.go` and `internal/*/_test.go`, no new Go deps without cause.
- Existing test style: pure-Go table/functional tests; no exec-based tests yet in `cmd/` (the gryph step needs the first — isolated with `t.Setenv("HOME", t.TempDir())` and a fake `gryph` on PATH).

## Requirements

1. **Install trigger** — presence of `@safedep/gryph` in `config.d/tools.yaml` `tools` list gates the gryph step (same convention as the agent-browser gate). No new config file, no new top-level config concept.
2. **Binary availability** — the npm install of `@safedep/gryph` (existing step 4) places the binary at `~/.skillgrid/npm/bin/gryph`. The gryph step prefers that path, falls back to bare `gryph` (matches the `skills` bin resolution at `steps.go:247-250`).
3. **Hooks for OpenCode** — run `gryph install --agent opencode`. Gryph is idempotent (warns + skips if the plugin already contains the hook marker); re-runs are safe.
4. **Hooks for Kilo** — after step 3, if `~/.config/kilo/plugins/gryph.js` is missing and `~/.config/opencode/plugins/gryph.js` exists, copy it (create `plugins/` dir). If the source is missing, `logging.Warn` and continue. Never overwrite an existing Kilo plugin file (first-write-wins, mirrors the engram bridge).
5. **Policy** — after hooks: `gryph policy init` (scaffold; no-op overwrite for repeated installs), then `gryph policy validate`, then `gryph config set policy.enabled true`. Each failure is logged via `logging.Warn` and does not abort the install.
6. **Dry-run** — `--dry-run` prints every planned gryph action (`[dry-run] gryph install --agent opencode`, `[dry-run] cp <opencode-plugin> <kilo-plugin>`, and each policy command) and performs zero execs/writes.
7. **Agent selection** — the gryph step runs for the agents selected by the existing selector: OpenCode hooks only when `opencode` is selected; Kilo copy only when `kilo` is selected. Selecting neither skips the step entirely except the policy sub-step (policy is machine-global, run when at least one of the two agents is selected).
8. **No uninstall in scope** — `gryph uninstall` exists upstream and is documented for removal; skillgrid v1 does not drive it.
9. **Documentation** — usage doc gets a Gryph section (what was installed, how to log/query/policy); config-reference notes that `@safedep/gryph` is special-cased; NOTE.md Usage Data checklist marks gryph done.

## Non-requirements (YAGNI)

- No per-project/repo gryph policy management (gryph's repo-local policy layer is out of scope; the global policy from `policy init` is the shipped default).
- No MCP entry for gryph (it is a hook/audit tool, not an MCP provider).
- No Kilo-native hook adapter, no OpenCode hooks.json manipulation — the plugin file is the whole integration surface.
- No Windows-specific path logic beyond what the existing `mustExpandHomePath` already does (installNPM already no-ops on Windows; the gryph step will run only after npm install succeeds, consistent with current behavior).
- No retention/DB tuning — gryph defaults apply.

## Architecture / data flow

```
config.d/tools.yaml (tools: [..., "@safedep/gryph"])
        |
        v
skillgrid install
  step 4 : npm install -g --prefix ~/.skillgrid/npm  -->  ~/.skillgrid/npm/bin/gryph
  step 4c: installGryph(agents, dryRun)
           |   [opencode selected]  gryph install --agent opencode
           |                        -> ~/.config/opencode/plugins/gryph.js
           |   [kilo selected]      cp opencode plugin -> ~/.config/kilo/plugins/gryph.js
           |                        (only if missing; warn if source missing)
           v
           gryph policy init
           gryph policy validate
           gryph config set policy.enabled true

At agent runtime:
  Kilo plugin (.mjs)  / OpenCode plugin (gryph.js)
        |   for each tool call (pre/post)
        v
  exec `gryph` (PATH)  ->  policy check (block/warn/guide) + SQLite event log
```

## Error handling

| Failure | Behavior |
|---|---|
| `gryph` binary missing (npm install failed earlier) | bare-`gryph` PATH fallback already covers user-rc PATH; if unresolved, `exec.Command` error → `logging.Warn("gryph install failed: ...")`; install continues |
| Kilo plugin source missing | `logging.Warn` with the missing path; no crash; Kilo simply has no gryph hooks |
| `gryph policy init` fails (file exists + non-empty conflict is not possible without `--force`, and we do not pass it) | `logging.Warn`, continue to validate |
| `gryph policy validate` fails | `logging.Warn`; `config set` still attempted (enablement independent) |
| `gryph config set policy.enabled true` fails | `logging.Warn`; hooks remain installed (audit still works, enforcement off) |
| Any exec step in dry-run | never executed; only logged |

## Testing strategy

- `cmd/gryph_test.go` (new):
  - dry-run emits the four expected log lines and never reports a real install;
  - real path (temp `HOME` + fake `gryph` shell script on PATH appending its args to a log file) verifies the exact argument sequences: `install --agent opencode`, `policy init`, `policy validate`, `config set policy.enabled true`, and that the Kilo copy actually writes the file with the source content;
  - Kilo copy source-missing case logs a `WARN` and does not panic;
  - agent-selection gating: `opencode`-only selection skips the Kilo copy; `kilo`-only selection still copies if the source exists.
- Regression bar: `go build ./... && go test ./...` green in `skillgrid-cli/`.

## Interfaces

```go
// cmd/steps.go — new, called from cmd/install.go step 4c
func installGryph(baseDir string, agents []string, dryRun bool)
```

No changes to signatures of existing functions. No new packages. No new imports beyond `os`, `os/exec`, `path/filepath`, `strings` (all already imported in the touched files).

## Files touched

| File | Change |
|---|---|
| `config.d/tools.yaml` | +1 line: `@safedep/gryph` |
| `skillgrid-cli/cmd/install.go` | +gated call after the agent-browser block (step 4b) |
| `skillgrid-cli/cmd/steps.go` | +`installGryph` |
| `skillgrid-cli/cmd/gryph_test.go` | new test file |
| `docs/02-usage.md` | +Gryph section |
| `docs/03-config-reference.md` | tools.yaml semantics note |
| `docs/NOTE.md` | Usage Data: gryph `[ ]` → `[X]` |

## Migration / rollout

- First run on a machine: npm fetch adds one small package; hooks land for both agents; policy scaffolded once.
- Re-runs: idempotent (gryph plugin marker check + first-wins copy + non-force `policy init`).
- Rollback: `gryph uninstall --restore-backup` per agent + remove `~/.config/kilo/plugins/gryph.js` + remove the `tools.yaml` line.
- User-visible new dependency: one more process on PATH + one SQLite DB under gryph's data dir.

## Open decisions (resolved)

1. Install channel → npm (matches existing installer) over brew/install.sh.
2. Kilo integration → plugin copy (no adapter upstream) over writing Kilo hooks JSON manually.
3. Policy scope → global default from `policy init`, not a skillgrid-managed policy file (keeps `config.d/` honest: it declares installs, not gryph's policy semantics).
