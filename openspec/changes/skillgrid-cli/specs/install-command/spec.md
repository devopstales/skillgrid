## ADDED Requirements

### Requirement: Default install command

The CLI SHALL run the install pipeline when invoked without a subcommand (or with `install`).

#### Scenario: Default invocation runs install
- **GIVEN** the user runs `skillgrid`
- **WHEN** no subcommand is provided
- **THEN** the install pipeline executes

#### Scenario: Explicit install subcommand
- **GIVEN** the user runs `skillgrid install`
- **WHEN** the `install` subcommand is provided
- **THEN** the install pipeline executes

#### Scenario: Version flag
- **GIVEN** the user runs `skillgrid --version`
- **WHEN** the flag is parsed
- **THEN** the version is printed and the program exits without running install

#### Scenario: Help flag
- **GIVEN** the user runs `skillgrid --help`
- **WHEN** the flag is parsed
- **THEN** usage information is printed and the program exits

### Requirement: Install pipeline ordering

The install pipeline SHALL execute steps in the following order:
1. Create `~/.skillgrid` directory structure
2. Sync the Skillgrid repository (clone or pull)
3. Verify node + npm are on PATH
4. Select agents (interactive or preset)
5. Install selected agents via npm
6. Install global tools (skills, openspec)
7. Copy repo `.agents/` → `~/.agents/`

#### Scenario: Directory structure created first
- **GIVEN** `~/.skillgrid` does not exist
- **WHEN** the install pipeline runs
- **THEN** the directory structure is created before any other step

#### Scenario: Repo synced before agent install
- **GIVEN** the repo is not yet cloned
- **WHEN** the install pipeline runs
- **THEN** the repo is cloned before agents are installed

#### Scenario: Node check before npm install
- **GIVEN** node or npm is missing from PATH
- **WHEN** the check step runs
- **THEN** a descriptive error is shown with the install script path and the run stops

#### Scenario: Agents installed before global tools
- **GIVEN** agents are selected
- **WHEN** the install pipeline runs
- **THEN** agents are installed before global tools

### Requirement: Dry-run mode

The CLI SHALL support `--dry-run` (`-n`) to print planned changes without writing.

#### Scenario: Dry-run prints all steps
- **GIVEN** the user runs `skillgrid --dry-run`
- **WHEN** the install pipeline runs
- **THEN** each planned action is prefixed with `[dry-run]` and no changes are written

#### Scenario: Dry-run reports no changes
- **GIVEN** the install pipeline completes in dry-run mode
- **WHEN** the run finishes
- **THEN** a "no changes were written (dry run)" message is displayed

### Requirement: Skip flags

The CLI SHALL support flags to skip optional pipeline steps.

#### Scenario: Skip clone
- **GIVEN** the user runs `skillgrid --skip-clone`
- **WHEN** the pipeline runs
- **THEN** the repo sync step is skipped (repo must already exist)

#### Scenario: Skip tools
- **GIVEN** the user runs `skillgrid --skip-tools`
- **WHEN** the pipeline runs
- **THEN** the global tool install step is skipped

#### Scenario: Skip agents copy
- **GIVEN** the user runs `skillgrid --skip-agents`
- **WHEN** the pipeline runs
- **THEN** the `.agents/` copy step is skipped

### Requirement: Verbose mode

The CLI SHALL support `--verbose` (`-l`) to print detailed output.

#### Scenario: Verbose shows node version
- **GIVEN** the user runs `skillgrid --verbose`
- **WHEN** the node check step runs
- **THEN** the node and npm versions are printed

#### Scenario: Verbose shows command output
- **GIVEN** the user runs `skillgrid --verbose`
- **WHEN** a command succeeds
- **THEN** the command output is printed

### Requirement: Repository configuration

The CLI SHALL support `--repo-url` and `--branch` to configure the source repository.

#### Scenario: Custom repo URL
- **GIVEN** the user runs `skillgrid --repo-url https://github.com/fork/skillgrid`
- **WHEN** the repo sync step runs
- **THEN** the specified URL is cloned

#### Scenario: Custom branch
- **GIVEN** the user runs `skillgrid --branch main`
- **WHEN** the repo sync step runs
- **THEN** the specified branch is checked out
