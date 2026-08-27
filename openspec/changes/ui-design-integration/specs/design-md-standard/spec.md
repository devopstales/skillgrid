## ADDED Requirements

### Requirement: DESIGN.md standard

The system SHALL define `DESIGN.md` as the visual system document standard, distinct from IDD technical specs.

#### Scenario: DESIGN.md created on project init
- **GIVEN** a UI-capable project runs `skillgrid install`
- **WHEN** the init completes
- **THEN** a `DESIGN.md` (or `docs/design/DESIGN.md`) is scaffolded from the template

#### Scenario: Required sections present
- **GIVEN** a `DESIGN.md` file
- **WHEN** it is validated
- **THEN** it includes: Product context, Brand & voice, Color, Typography, Spacing & layout, Motion, Components, Accessibility, Implementation stack, Verification

#### Scenario: Cross-link to IDD spec
- **GIVEN** a UI feature IDD `-design.md`
- **WHEN** the file is authored
- **THEN** it links to `DESIGN.md` and does NOT duplicate token tables

#### Scenario: Zone separation
- **GIVEN** a `DESIGN.md` at repo root
- **WHEN** it is committed
- **THEN** it is committed in a docs-only commit (zone-guard) or placed under `docs/design/`

### Requirement: DESIGN.md template

The system SHALL provide a canonical `config.d/templates/DESIGN.md` template.

#### Scenario: Template copied on init
- **GIVEN** `config.d/templates/DESIGN.md` exists
- **WHEN** a UI project is initialized
- **THEN** the template is copied to the project's `DESIGN.md` location
