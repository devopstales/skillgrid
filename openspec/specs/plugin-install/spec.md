# Plugin Install Specification

## Purpose
TBD — describe editor plugin installation behavior.

## Requirements

### Requirement: Plugin installation

The CLI SHALL install agent plugins via `skillgrid setup <agent>`; `--dry-run`
print-only and `--skip-<step>` flags are not supported on `setup` (use
`skillgrid install --skip-agents` to skip agent/plugin wiring).

#### Scenario: Plugin installed for editor
- **GIVEN** `skillgrid setup opencode` runs against a populated repo
- **WHEN** the setup completes
- **THEN** the OpenCode plugin is copied to `~/.config/opencode/plugins/mnemonic.ts`

#### Scenario: Plugin first-write-wins
- **GIVEN** `~/.config/opencode/plugins/mnemonic.ts` already exists
- **WHEN** `skillgrid setup opencode` runs again
- **THEN** the existing plugin is preserved (first-write-wins) and the setup reports the file as unchanged

#### Scenario: Plugin target validated
- **GIVEN** `skillgrid setup unknown` runs
- **WHEN** the command validates the target
- **THEN** a clear error is returned listing the supported targets (`opencode`, `kilocode` / `kilo`, `cursor`)

### Requirement: Supported plugin set and install targets

The supported plugin set is closed and version-controlled in the repo under
`plugins/`, and each target maps to a single well-known install path.

| target     | source in repo               | install target                                |
|------------|------------------------------|-----------------------------------------------|
| `opencode` | `plugins/opencode/mnemonic.ts`   | `~/.config/opencode/plugins/mnemonic.ts`       |
| `kilocode` \| `kilo` | `plugins/kilo/mnemonic.ts` | `~/.config/kilo/plugins/mnemonic.ts`     |
| `cursor`   | `plugins/cursor/mnemonic.mdc`    | `~/.cursor/rules/mnemonic.mdc`            |

`vscode` is reserved for a future change and is not in the supported set this
release.

#### Scenario: Copy source exists before install
- **GIVEN** `skillgrid setup opencode` runs
- **WHEN** the source is resolved
- **THEN** `plugins/opencode/mnemonic.ts` exists in the resolved repo and is non-empty; otherwise the command reports a missing-plugin error and exits non-zero

#### Scenario: Kilo and OpenCode plugins are identical
- **GIVEN** the repo is checked out
- **WHEN** `plugins/opencode/mnemonic.ts` and `plugins/kilo/mnemonic.ts` are diffed
- **THEN** the files are identical (kilo is a verbatim copy of opencode) — a divergence is an install error
