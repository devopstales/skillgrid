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
