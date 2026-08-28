## ADDED Requirements

### Requirement: Standard skillgrid paths

The system SHALL define and document standard path conventions for skillgrid skills.

#### Scenario: Config path resolved
- **GIVEN** a skillgrid skill needs to reference configuration
- **WHEN** the skill resolves a config path
- **THEN** it uses `config.d/` as the single source of truth

#### Scenario: Plugins path resolved
- **GIVEN** a skillgrid skill needs to reference editor plugins
- **WHEN** the skill resolves a plugin path
- **THEN** it uses `plugins/<editor>/` (e.g., `plugins/opencode/`, `plugins/kilo/`, `plugins/cursor/`)

#### Scenario: Skills path resolved
- **GIVEN** a skillgrid skill needs to reference other skills
- **WHEN** the skill resolves a skill path
- **THEN** it uses `.agents/skills/` as the canonical skills directory

#### Scenario: Worktrees path resolved
- **GIVEN** a skillgrid skill needs to reference git worktrees
- **WHEN** the skill resolves a worktree path
- **THEN** it uses the worktree root for the active change

### Requirement: OpenSpec workflow in skills

The system SHALL integrate OpenSpec workflow operations into skillgrid skills.

#### Scenario: Propose workflow
- **GIVEN** a user wants to start a new change
- **WHEN** they invoke the propose skill
- **THEN** it runs `openspec new change` and guides through proposal creation

#### Scenario: Apply workflow
- **GIVEN** a change has tasks ready
- **WHEN** they invoke the apply skill
- **THEN** it runs `openspec status --change` and implements tasks

#### Scenario: Verify workflow
- **GIVEN** implementation is complete
- **WHEN** they invoke the verify skill
- **THEN** it runs `openspec validate` and checks acceptance criteria

#### Scenario: Archive workflow
- **GIVEN** verification passed
- **WHEN** they invoke the archive skill
- **THEN** it runs `openspec archive` and moves the change to archive
