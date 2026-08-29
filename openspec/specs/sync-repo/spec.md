# Sync Repo Specification

## Purpose
TBD — describe the `sync-repo` command for local directory installation.

## Requirements

### Requirement: Sync-repo command

The CLI SHALL support `sync-repo PATH` to install from a local directory without network access.

#### Scenario: Sync-repo flag form
- **GIVEN** the user runs `skillgrid --sync-repo /path/to/skillgrid`
- **WHEN** the flag is parsed
- **THEN** the local repo is synced to `~/.skillgrid/repos/skillgrid`

#### Scenario: Sync-repo subcommand form
- **GIVEN** the user runs `skillgrid sync-repo /path/to/skillgrid`
- **WHEN** the subcommand is parsed
- **THEN** the local repo is synced to `~/.skillgrid/repos/skillgrid`

#### Scenario: Sync-repo validates source exists
- **GIVEN** the user runs `skillgrid sync-repo /nonexistent`
- **WHEN** the path is validated
- **THEN** an error is shown and the program exits

#### Scenario: Sync-repo validates source is directory
- **GIVEN** the user runs `skillgrid sync-repo /file.txt`
- **WHEN** the path is validated
- **THEN** an error is shown and the program exits

#### Scenario: Sync-repo overwrites existing
- **GIVEN** `~/.skillgrid/repos/skillgrid` already exists
- **WHEN** sync-repo runs
- **THEN** the existing directory is removed and replaced with the source

#### Scenario: Sync-repo copies .agents
- **GIVEN** the source contains `.agents/`
- **WHEN** sync-repo runs
- **THEN** `.agents/` is copied to `~/.agents/`

#### Scenario: Sync-repo dry-run
- **GIVEN** the user runs `skillgrid --sync-repo /path --dry-run`
- **WHEN** sync-repo runs
- **THEN** planned copies are printed and no changes are written

### Requirement: Destination safety

The sync-repo command SHALL never write outside `~/.skillgrid/repos/` or `~/.agents/`, and SHALL refuse to run when the source and destination collapse into the same tree.

#### Scenario: Source is destination parent
- **GIVEN** the user runs `skillgrid sync-repo ~/.skillgrid` (i.e. the repo is already inside itself)
- **WHEN** the destination is resolved
- **THEN** the command detects the collision, prints a clear error, and exits with a non-zero code before writing anything

#### Scenario: Symlink loops do not hang
- **GIVEN** the source tree contains a symlink that points back into itself
- **WHEN** the copy is executed
- **THEN** the copy does not loop infinitely; it either follows symlinks once or skips them (determined by the copy algorithm) and finishes
