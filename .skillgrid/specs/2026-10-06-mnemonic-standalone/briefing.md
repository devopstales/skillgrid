# Mnemonic Standalone Module — Design Briefing (requirements & intent)

**Status:** PROPOSED

**Tier:** T2

**Goal:** Extract `mnemonic` from `skillgrid-cli` into an independently buildable Go module (`mnemonic/`) with its own `go.mod`, `cmd/mnemonic` binary, and top-level public facade. `skillgrid-cli` keeps only `install`, `sync-repo`, and thin passthrough wiring. No passthrough to `skillgrid mcp|serve|index|...` — those commands move entirely to the `mnemonic` binary.

**Architecture:** Two-module Go workspace. `skillgrid-cli` (existing) shrinks to an installer + a thin CLI that shells out to the `mnemonic` binary for all mnemonic subcommands. `mnemonic/` (new) contains all 37 mnemonic packages under `internal/`, a `cmd/mnemonic/main.go` with the full subcommand tree (mcp, serve, index, prime, orient, search, mem, memory, skill, etc.), and ~22 top-level facade files exposing the types/functions that `skillgrid-cli/internal/install` needs (setup, config, store). A `go.work` file links both modules for local development.

**Tech Stack:** Go 1.26, go.work (multi-module workspace), existing deps (bubbletea, mcp-go, gotreesitter, modernc.org/sqlite, etc.)

**Spec:** `../briefing.md` (this file)

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `cd mnemonic && go build ./cmd/mnemonic` produces a working `mnemonic` binary
- `mnemonic mcp` starts the MCP stdio server (same behavior as today's `skillgrid mcp`)
- `mnemonic serve` starts the HTTP API on :7438 (same behavior as today's `skillgrid serve`)
- `mnemonic index --dir .` runs incremental code indexing (same behavior)
- `skillgrid install` still works end-to-end (clones, configures agents, writes MCP entries pointing to `mnemonic` binary)
- `skillgrid sync-repo /path` still works
- `skillgrid mcp`, `skillgrid serve`, `skillgrid index` etc. are **removed** from the CLI (clean break, no passthrough)
- `cd mnemonic && go test ./...` passes
- `cd skillgrid-cli && go test ./...` passes
- `go.work` at repo root allows `go build ./...` from root to build both modules
- External config files (`config.d/mcp.yaml`, `plugins/cursor/mcp.json`, hooks, git-hooks) reference the `mnemonic` binary, not `skillgrid`
- The 61 MB committed binary `skillgrid-cli/skillgrid` is deleted
- `scripts/sync-mnemonic-rule.sh` no longer references the stale `plugins/_shared/` path

**Artifacts** (files that must exist with real implementation, not stubs):
- [`mnemonic/go.mod`] — module `github.com/devopstales/skillgrid/mnemonic`, go 1.26, with all mnemonic deps
- [`mnemonic/cmd/mnemonic/main.go`] — full subcommand tree (mcp, serve, init, index, prime, orient, grep, callers, callees, etc.)
- [`mnemonic/internal/`] — all 37 mnemonic packages moved from `skillgrid-cli/internal/mnemonic/`
- [`mnemonic/` facade files] — ~22 top-level `.go` files exposing types/functions needed by `skillgrid-cli`
- [`go.work`] — multi-module workspace linking both modules
- [`skillgrid-cli/go.mod`] — updated: loses mnemonic deps, gains `replace` directive or direct require on `github.com/devopstales/skillgrid/mnemonic`
- [`skillgrid-cli/cmd/skillgrid/main.go`] — trimmed: only install, sync-repo, help remain

**Key links** (critical connections between artifacts that must work together):
- `skillgrid-cli/internal/install` imports `mnemonic/setup` (for agent plugin config) via the facade, not via `internal/` path
- `config.d/mcp.yaml` command is `[mnemonic, mcp]` (was `[skillgrid, mcp]`)
- `plugins/cursor/mcp.json` command is `mnemonic` (was `skillgrid`)
- `hooks/cursor-session-start.sh` and `hooks/opencode-session-start.sh` invoke `mnemonic serve` / `mnemonic prime` (was `skillgrid serve` / `skillgrid prime`)
- `git-hooks/post-commit` and `git-hooks/post-checkout` invoke `mnemonic index` / `mnemonic pages` (was `skillgrid index` / `skillgrid pages`)
- `skillgrid-ui/vite.config.ts` outputs to `../mnemonic/internal/http/ui/dist` (was `../skillgrid-cli/internal/mnemonic/http/ui/dist`)

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- **Clean break: `skillgrid mcp|serve|index|prime|orient|search|mem|memory|skill|logs|sessions|session|stats|policy|embedding-status|doctor|eval|migrate|trail|diff-impact|pages|compact` are removed from `skillgrid` CLI.** All existing installations that call `skillgrid mcp` or `skillgrid serve` will break until the `mnemonic` binary is installed. Mitigation: `skillgrid install` installs the `mnemonic` binary and writes all config to reference it. But any hand-edited configs or older machines will need manual update.
- **Module path `github.com/devopstales/skillgrid/mnemonic`** — once published, the import path is permanent. Changing it later requires a full import-path migration across all dependents.

## Global Constraints

- Go 1.26 (both modules)
- No new external dependencies beyond what `skillgrid-cli/go.mod` already has
- All mnemonic packages move as-is (no refactoring of internal package structure during the move)
- The `mnemonic` binary must be independently installable: `go install github.com/devopstales/skillgrid/mnemonic/cmd/mnemonic@latest` (or build from source)
- `skillgrid install` must install the `mnemonic` binary (copy to `~/.skillgrid/bin/mnemonic` or `~/.local/bin/mnemonic`)
- Both modules must pass `go vet ./...` and `go test ./...` independently
- The `go.work` file is for local dev convenience; each module must be independently buildable without the workspace

## File Structure

### New: `mnemonic/` module
```
mnemonic/
  go.mod                        # module github.com/devopstales/skillgrid/mnemonic
  go.sum
  cmd/
    mnemonic/
      main.go                   # full subcommand tree (moved from skillgrid-cli/cmd/skillgrid/)
      mcp.go                    # runMCP, runServe
      mem.go                    # runMem, runMemory
      search.go                 # runSearch, runSearchEmbeddingStatus
      code_intel.go             # runCodeIntel (orient, grep, callers, etc.)
      loop_cmd.go               # runPrime, runCompact
      index_progress.go         # runIndex
      init_arch.go              # runProjectInit
      init_boot.go
      init_preamble.go
      init_ingest.go
      doctor.go                 # runDoctor
      doctor_strict.go
      migrate.go                # runMigrate
      policy_cmd.go             # runPolicy
      session.go                # runSession, runSessions
      skill.go                  # runSkill
      trail.go                  # runTrail
      logs.go                   # runLogs
      memory.go                 # runMemory (fact memory)
      code_eval.go              # runEval
      search_affected_cli.go
      search_pdg_cli.go
      search_rename_cli.go
      search_taint_cli.go
      handoff_gone.go
      reorder.go
      memhelpers.go
      # + all corresponding *_test.go files
  internal/
    affected/                   # moved from skillgrid-cli/internal/mnemonic/affected/
    checkpoint/
    codeindex/
    community/
    config/
    embedder/
    eval/
    extract/
    facts/
    files/
    graph/
    http/                       # contains ui/dist (SPA embed)
    hybrid/
    integration/
    knowledge/
    llm/
    loop/
    mcp/
    memfs/
    memory/
    pdg/
    policy/
    process/
    project/
    route/
    search/
    secondbrain/
    service/
    session_inject/
    setup/                      # inlined logging (no more import of skillgrid-cli/internal/logging)
    skills/
    store/
    tiered/
    vectorstore/
    webcache/
  # Facade files (top-level, package mnemonic)
  setup.go                      # re-exports setup.Config, setup.Run, etc.
  config.go                     # re-exports config types
  store.go                      # re-exports store types
  # ... ~22 facade files total
```

### Modified: `skillgrid-cli/`
```
skillgrid-cli/
  go.mod                        # loses: onnxer, loom, bubbletea, huh, mcp-go, gotreesitter, gjson, sjson, uuid, fsnotify, yaml, modernc.org/sqlite
                                # gains: require github.com/devopstales/skillgrid/mnemonic v0.0.0 (replace directive)
  cmd/
    skillgrid/
      main.go                   # trimmed: only install, sync-repo, help
      # removed: mcp.go, mem.go, memhelpers.go, search.go, code_intel.go, loop_cmd.go,
      #          index_progress.go, init_*.go, doctor*.go, migrate.go, policy_cmd.go,
      #          session.go, skill.go, trail.go, logs.go, memory.go, code_eval.go,
      #          search_*.go, handoff_gone.go, reorder.go, + all their tests
  internal/
    install/                    # stays; import of mnemonic/setup → facade
    ui/                         # stays (pure stdlib)
    # removed: logging/ (inlined into mnemonic/setup)
    # removed: mnemonic/ (moved to mnemonic/internal/)
```

### New: repo root
```
go.work                         # go 1.26; use ./mnemonic ./skillgrid-cli
```

### Modified: external config
```
config.d/mcp.yaml               # command: [mnemonic, mcp]
plugins/cursor/mcp.json         # command: "mnemonic"
hooks/cursor-session-start.sh   # skillgrid serve → mnemonic serve; skillgrid prime → mnemonic prime
hooks/opencode-session-start.sh # skillgrid serve → mnemonic serve; skillgrid prime → mnemonic prime
git-hooks/post-commit           # skillgrid index → mnemonic index; skillgrid pages → mnemonic pages
git-hooks/post-checkout         # skillgrid index → mnemonic index; skillgrid pages → mnemonic pages
skillgrid-ui/vite.config.ts     # outDir: '../mnemonic/internal/http/ui/dist'
scripts/sync-mnemonic-rule.sh   # fix stale plugins/_shared/ path
Taskfile.yml                    # build both modules, test both, cross-build both
```

### Deleted
```
skillgrid-cli/skillgrid         # 61 MB committed binary
skillgrid-cli/internal/mnemonic/mcp/.git   # stray nested git repo
```

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Module boundary (go.work) | Applicable: two modules with a replace directive | `go.work` for local dev; `replace` in skillgrid-cli/go.mod for independent builds | `go build ./...` from both module roots independently |
| Binary replacement (skillgrid → mnemonic) | Applicable: all external configs reference the binary | `skillgrid install` writes `mnemonic` binary + updates all configs; hooks check `command -v mnemonic` | `skillgrid install --dry-run` output shows mnemonic binary path |
| Internal visibility (Go internal/ rule) | Applicable: CLI cannot import mnemonic/internal/ | Facade files at mnemonic/ top level (package mnemonic) re-export needed symbols | `go build` in skillgrid-cli compiles with facade imports |
| SPA embed path | Applicable: vite.config.ts output path changes | Update outDir to `../mnemonic/internal/http/ui/dist`; `go:embed` in mnemonic/http picks it up | `go build -tags ui` in mnemonic/ succeeds with embedded SPA |

## Build Shape

**Smallest usable whole** — the thinnest version a person could actually use is: `mnemonic mcp` works as a standalone binary. This is the first task. Once the MCP server runs standalone, everything else (install wiring, config updates, cross-build) thickens around it.

## Terms

- **Facade** — top-level `.go` files in `mnemonic/` (package `mnemonic`) that re-export types and functions from `mnemonic/internal/` packages, making them importable by `skillgrid-cli`.
- **Clean break** — `skillgrid` CLI no longer has mnemonic subcommands; there is no `skillgrid mcp` passthrough. Users must use the `mnemonic` binary directly.
