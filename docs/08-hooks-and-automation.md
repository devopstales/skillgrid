# Hooks And Automation

This document defines hook locations, shared scripts, and automation policy across supported agent surfaces.

## Canonical Shared Hook Scripts

- Shared refresh hook: `.agents/hooks/refresh-indexes.sh`
- SDD gate script: `.skillgrid/scripts/sdd-gate.sh`
- Hook installation: `install.sh` or `skillgrid install` (not a standalone hook script)

Full script inventory: [`.skillgrid/scripts/README.md`](../.skillgrid/scripts/README.md).

### SDD Gate Hooks

- Purpose: programmatic enforcement of SDD gates (label validation, artifact checks, phase routing)
- Trigger: pre-commit (staged `openspec/changes/*`), pre-push (only changes touched in the push)
- Behavior:
  - Pre-commit infers phase from staged paths per change (`propose` → `spec` → `design` → `tasks` → `apply` → `verify-report` → `reviews`)
  - `sdd-gate.sh review` blocks until **sdd-verify** evidence exists; `sdd-gate.sh archive` also requires **sdd-review** APPROVED
  - Pre-push runs `sdd-gate.sh verify` only for changes with `tasks.md` that appear in the outgoing commit range
  - Exit code 1 = blocked (no commit/push proceeds)

Installation (automatic when the target is a git repo; skip with `--no-sdd-hooks`):

```bash
./install.sh -p /path/to/project
skillgrid install -p /path/to/project
```

## Surface Wiring

- Cursor:
  - `.cursor/hooks.json`
  - `.cursor/hooks/refresh-indexes.sh` (wrapper delegating to shared script)
- GitHub agents:
  - `.github/hooks/refresh-indexes.json`
- Kilo:
  - `.kilo/hook/hooks.md`
- OpenCode:
  - `.opencode/hook/hooks.md`

## Trigger Policy

Refresh automation is intended for shell commands matching:

- `git merge`
- `gh pr merge`
- `git pull`
- `ccc init`

## Operational Policy

1. Keep hook logic deduplicated under `.agents/hooks/`.
2. Surface-specific files should be thin wrappers/config only.
3. Hook failures should not block user workflow unless explicitly set to fail-closed.
4. Any hook behavior change must be reflected in this file and in `17-ide-configs.md`.

## Related Documents

- `01-installation.md`
- `02-workflow-usage.md`
- `12-memory-and-indexing.md`
- `17-ide-configs.md`
