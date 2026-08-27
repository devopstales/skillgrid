## ADDED Requirements

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
