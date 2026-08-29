# Configuration System Specification

## Purpose
TBD — describe the CLI configuration system, flags, and output handling.

## Requirements

### Requirement: Configuration system

The CLI SHALL use a Config struct to hold all runtime state.

#### Scenario: Config holds all flags
- **GIVEN** the CLI is invoked with flags
- **WHEN** flags are parsed
- **THEN** all values are stored in the Config struct

#### Scenario: Default repo URL
- **GIVEN** no `--repo-url` is provided
- **WHEN** Config is created
- **THEN** the default URL is `https://github.com/devopstales/skillgrid`

#### Scenario: Default branch
- **GIVEN** no `--branch` is provided
- **WHEN** Config is created
- **THEN** the default branch is `release/2`

### Requirement: Output handling

The CLI SHALL write install log lines to stderr to keep stdout clean for scripts.

#### Scenario: Progress messages on stderr
- **GIVEN** the install pipeline runs
- **WHEN** a progress message is printed
- **THEN** it is written to stderr

#### Scenario: Command failure output
- **GIVEN** a command fails during install
- **WHEN** the error is reported
- **THEN** the command output is printed to stderr

### Requirement: Error handling

The CLI SHALL stop on hard failures and continue on non-fatal issues.

#### Scenario: Missing home directory
- **GIVEN** the home directory cannot be resolved
- **WHEN** Config is created
- **THEN** an error is shown and the program exits with code 1

#### Scenario: Missing node/npm
- **GIVEN** node or npm is not on PATH
- **WHEN** the check step runs
- **THEN** a descriptive error with the install script path is shown and the program exits

#### Scenario: Unknown subcommand
- **GIVEN** the user runs `skillgrid unknown`
- **WHEN** the subcommand is parsed
- **THEN** an error is shown and the program exits with code 2

### Requirement: Environment overrides

The CLI SHALL honour a small set of environment-variable overrides so
CI/scripts can configure behaviour without long flag strings, and the
overrides SHALL be documented in `--help` output (or a `skillgrid env`
subcommand).

| environment variable           | default                         |
|--------------------------------|---------------------------------|
| `SKILLGRID_REPO_URL`           | `https://github.com/devopstales/skillgrid` |
| `SKILLGRID_BRANCH`             | `release/2`                     |
| `SKILLGRID_VERSION`            | `git describe --tags --always --dirty` |
| `SKILLGRID_HTTP_TOKEN`         | (unset)                         |

#### Scenario: Repository URL overridden by env
- **GIVEN** `SKILLGRID_REPO_URL=https://github.com/fork/skillgrid` is set
- **WHEN** the repo sync step runs without `--repo-url`
- **THEN** the fork URL is cloned

#### Scenario: Explicit flag wins over env
- **GIVEN** `SKILLGRID_BRANCH=release/2` is set
- **WHEN** the user runs `skillgrid install --branch main`
- **THEN** `main` is used (the flag overrides the env var)

#### Scenario: VERSION drives baked version
- **GIVEN** `SKILLGRID_VERSION=1.2.3-rc1` is set
- **WHEN** `task build` or `task all` runs
- **THEN** the produced binary reports `1.2.3-rc1` for `--version`
