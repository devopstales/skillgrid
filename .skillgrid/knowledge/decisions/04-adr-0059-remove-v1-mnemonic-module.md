---
type: TechnicalDecision
status: in-force
supersedes: none
date: 2026-11-07
---

# ADR-0059 — Remove the v1 mnemonic engine module

Change: — (direct instruction, no change folder; the `task → commit` chain is in the commit's `[skillgrid-context]` block)

## Context

- ADR-0052 extracted the v2 engine into a fresh `mnemonic-v2/` module with v1 (`mnemonic/`, module `github.com/devopstales/skillgrid/mnemonic`) **coexisting until v2 ships**. The memory, code-index, and knowledge-base workstreams have all landed in v2; v2 is the sole active engine.
- The only remaining v1 consumer was `skillgrid-cli/internal/install`, which used six functions of the v1 root package (`Interactive`, `Option`, `MultiSelect`, `FindRepoRoot`, `LoadMCPConfig`, `RunSetup` — plus the `KanbanExec` test seam) through thin facade re-exports backed by `mnemonic/internal/{setup,ui,config}`.
- The Taskfile was already building only v2 + the CLI: its `MEM1_DIR` pointed at `mnemonic_back/`, a directory that did not exist in the tree (stale since the v1 rename).
- The v1-only surfaces — HTTP `:7438` + embedded SPA dashboard, fact memory (`POST /facts*`, `mem_fact_*`), the Context Harness / CLM (`ctx_*`), the executable-skill registry (`skillgrid skill`), the `mnemonic init` preamble writer, and the all-stores seed integration tests — have no consumer in the v2 flow. v2's `serve` command is already a no-op that prints that v2 dropped the HTTP dashboard.

## Decision

- **Delete the `mnemonic/` module.** `go.work` is now `mnemonic-v2/` + `skillgrid-cli/` only.
- **Move the three packages the CLI needed into `skillgrid-cli/internal/`:** `setup/` (per-agent MCP setup: cursor / opencode / kilocode / pi / hermes), `ui/` (huh multiselect + interactivity), and `config` → `harnessconfig/` (config.d loader; renamed to avoid a name collision with `install.Config`).
- **`skillgrid-cli/go.mod`:** drop the v1 `require` + `replace => ../mnemonic` (the replace applied workspace-wide and would have broken every module's build once the directory vanished); `charmbracelet/huh` is promoted indirect → direct. No new module-level dependencies.
- **Taskfile:** drop `MEM1_DIR` and its vet/test/gofmt lines; drop the `seed-test` task (its `TestSeed*` integration tests lived in v1's `internal/integration/` and v2 has no equivalent).
- **Docs:** `ARCHITECTURE.md` module map rewritten to the v2 layout; sections describing v1-only surfaces carry a "Removed with v1 (ADR-0059)" marker instead of a full re-derivation. The stale `mnemonic_back/mnemonic` `.gitignore` line is removed.

## Consequences

- Builds, tests, and the QA gate cover two modules; workspace-mode `replace` resolution is no longer affected by the installer's go.mod.
- `skillgrid-cli` is self-contained — the installer builds without the engine module (no cross-module `replace` anywhere in the workspace).
- v1's surfaces leave the product with it: HTTP `:7438` + embedded SPA, fact memory, Context Harness / CLM, `skillgrid skill`, the `init` preamble writer, seed integration tests. Re-adding any of them is a new ADR + change (the K6 handler seam is already shaped for a later HTTP mirror, per spec `004-transport-mcp` §4).
- The `mnemonic` binary is built from `mnemonic-v2/cmd/mnemonic`; the `skillgrid` binary from `skillgrid-cli/cmd/skillgrid`. No `mnemonic` v1 binary is built or shipped.
- ADR-0052's module-extraction decision stays in force; its "v1 coexists until v2 ships" clause ends here. The Taskfile dual-module build obligation (referenced as ADR-0040 in Taskfile comments; no ADR-0040 file exists in the index) no longer applies.
