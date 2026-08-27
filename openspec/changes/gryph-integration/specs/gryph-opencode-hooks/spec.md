## ADDED Requirements

### Requirement: OpenCode hooks installed via gryph CLI

The gryph step SHALL run `gryph install --agent opencode` to install audit hooks into OpenCode.

#### Scenario: gryph install runs for opencode
- **GIVEN** `opencode` is selected and the gryph step runs
- **WHEN** the OpenCode hooks sub-step executes
- **THEN** `gryph install --agent opencode` is invoked

#### Scenario: gryph install is idempotent
- **GIVEN** the OpenCode plugin already contains the gryph hook marker
- **WHEN** `gryph install --agent opencode` runs again
- **THEN** it warns and skips without overwriting

#### Scenario: gryph binary missing
- **GIVEN** neither npm-installed nor PATH `gryph` is available
- **WHEN** the OpenCode hooks sub-step executes
- **THEN** a warning is logged and the install continues
