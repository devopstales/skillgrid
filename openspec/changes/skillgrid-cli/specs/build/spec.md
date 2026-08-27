## ADDED Requirements

### Requirement: Taskfile build tasks

The CLI SHALL be buildable via the `Taskfile.yml` in `skillgrid-cli/` with the following tasks: `build` (local), `all` (cross-build), `build-linux`, `build-darwin`, `test`, `fmt`, `clean`, `install`, `version`.

#### Scenario: Local build
- **GIVEN** the developer runs `go build ./cmd/skillgrid`
- **WHEN** `task` finishes
- **THEN** the `dist/skillgrid` binary exists and responds to `--version`

#### Scenario: Cross-build all targets
- **GIVEN** the developer runs `task all`
- **WHEN** the task finishes
- **THEN** four binaries exist in `dist/`: `skillgrid-linux-amd64`, `skillgrid-linux-386`, `skillgrid-darwin-amd64`, and `skillgrid-darwin-arm64`

#### Scenario: Test task
- **GIVEN** the developer runs `task test`
- **WHEN** the task finishes
- **THEN** `go vet ./...` and `go test ./...` succeed

#### Scenario: Version baked at build time
- **GIVEN** the binary is compiled with `task all`
- **WHEN** the resulting binary is run with `--version`
- **THEN** the printed version matches `git describe --tags --always --dirty` (or the `SKILLGRID_VERSION` env override if set)
