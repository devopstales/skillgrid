## ADDED Requirements

### Requirement: Impeccable as primary design skill

The system SHALL register Impeccable as the primary design skill in the default skillgrid bundle.

#### Scenario: Impeccable available after install
- **GIVEN** `skillgrid install` completed
- **WHEN** the agent lists available skills
- **THEN** Impeccable is available with commands: shape, craft, audit, polish, critique

#### Scenario: No npx for design tools
- **GIVEN** Impeccable is installed
- **WHEN** the agent invokes an impeccable command
- **THEN** it runs via the skill/CLI path, not `npx impeccable`

### Requirement: SkillUI as optional bootstrap

The system SHALL support SkillUI as an optional tool for bootstrapping `DESIGN.md` from existing UI.

#### Scenario: SkillUI in tools.yaml
- **GIVEN** `config.d/tools.yaml` lists `skillui`
- **WHEN** `skillgrid install` runs
- **THEN** SkillUI binary is available at `~/.skillgrid/npm/node_modules/.bin/skillui`

#### Scenario: SkillUI extracts design system
- **GIVEN** SkillUI is installed
- **WHEN** `skillui --dir . --mode ultra` runs
- **THEN** a `DESIGN.md` seed is generated from the existing codebase
