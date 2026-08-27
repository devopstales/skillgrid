## ADDED Requirements

### Requirement: Kilo plugin copy from OpenCode

When `kilo` is selected and the OpenCode gryph plugin exists but the Kilo plugin does not, the gryph step SHALL copy the plugin file.

#### Scenario: Kilo plugin copied when missing
- **GIVEN** `~/.config/opencode/plugins/gryph.js` exists and `~/.config/kilo/plugins/gryph.js` does not
- **WHEN** the Kilo copy sub-step executes
- **THEN** the file is copied to `~/.config/kilo/plugins/gryph.js`

#### Scenario: Existing Kilo plugin not overwritten
- **GIVEN** `~/.config/kilo/plugins/gryph.js` already exists
- **WHEN** the Kilo copy sub-step executes
- **THEN** the existing file is preserved (first-write-wins)

#### Scenario: Source missing
- **GIVEN** `~/.config/opencode/plugins/gryph.js` does not exist
- **WHEN** the Kilo copy sub-step executes
- **THEN** a warning is logged and no crash occurs

### Requirement: Dry-run support

The gryph step SHALL print all planned actions in dry-run mode without executing any commands or writing any files.

#### Scenario: Dry-run prints actions
- **GIVEN** `--dry-run` is passed to `skillgrid install`
- **WHEN** the gryph step executes
- **THEN** it prints `[dry-run] gryph install --agent opencode`, `[dry-run] cp ...`, and each policy command
