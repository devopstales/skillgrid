# Build Specification

## Purpose
TBD — describe the Taskfile build pipeline for the skillgrid-cli binary.

## Requirements

### Requirement: Taskfile build tasks

The CLI SHALL be buildable via the `Taskfile.yml` at the repo root with the
following tasks: `build` (local), `all` (cross-build), `build-linux`,
`build-darwin`, `test`, `fmt`, `clean`, `install`, `version`. All tasks operate
on the Go module in `skillgrid-cli/` and emit binaries into the repo-root
`dist/` directory.

#### Scenario: Local build
- **GIVEN** the developer runs `task build` from the repo root
- **WHEN** the task finishes
- **THEN** the `dist/skillgrid` binary exists at the repo root and responds to
  `--version`

#### Scenario: Cross-build all targets
- **GIVEN** the developer runs `task all` from the repo root
- **WHEN** the task finishes
- **THEN** four binaries exist in `dist/`: `skillgrid-linux-amd64`,
  `skillgrid-linux-386`, `skillgrid-darwin-amd64`, and `skillgrid-darwin-arm64`

#### Scenario: Test task
- **GIVEN** the developer runs `task test` from the repo root
- **WHEN** the task finishes
- **THEN** `go vet ./...` and `go test ./...` succeed in `skillgrid-cli/`

#### Scenario: Version baked at build time
- **GIVEN** the binary is compiled with `task all`
- **WHEN** the resulting binary is run with `--version`
- **THEN** the printed version matches `git describe --tags --always --dirty`
  (or the `SKILLGRID_VERSION` env override if set)

### Requirement: Clean build

The Taskfile SHALL provide a `task clean` target that removes all build
artifacts (`dist/` and local `skillgrid-cli/skillgrid`) without touching any
tracked source files.

#### Scenario: Clean removes dist
- **GIVEN** `dist/skillgrid` exists
- **WHEN** `task clean` runs
- **THEN** `dist/` is empty or removed, no tracked file is modified, and `git status` reports no changes

#### Scenario: Clean is idempotent
- **GIVEN** `dist/` does not exist
- **WHEN** `task clean` runs
- **THEN** no error is raised and the task is a no-op

### Requirement: Local install to `~/.local/bin`

`task build` SHALL place the local build into a directory on the user's PATH
so that the freshly-built binary is immediately usable in tests and plugin
smoke tests without re-installing.

#### Scenario: Build leaves a binary at repo root
- **GIVEN** the developer has a Go toolchain
- **WHEN** `task build` runs
- **THEN** a `dist/skillgrid` binary exists and can be run (`./dist/skillgrid --version`)

#### Scenario: Build is repeatable
- **GIVEN** `dist/skillgrid` already exists
- **WHEN** `task build` runs again
- **THEN** the existing file is replaced (not appended to) and the build succeeds exactly once
