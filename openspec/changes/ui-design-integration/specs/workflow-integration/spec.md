## ADDED Requirements

### Requirement: UI feature workflow

The system SHALL define a UI feature workflow that integrates design into the IDD/BDD/TDD pipeline.

#### Scenario: Brainstorm before design
- **GIVEN** a UI feature request
- **WHEN** the agent starts work
- **THEN** `brainstorming` is invoked first to establish scope

#### Scenario: Impeccable shape before IDD spec
- **GIVEN** brainstorming is complete
- **WHEN** the agent proceeds
- **THEN** `/impeccable shape` or `craft` is run for UI direction (human approves before continuing)

#### Scenario: DESIGN.md updated before IDD spec
- **GIVEN** new tokens or components are introduced
- **WHEN** the design is approved
- **THEN** `DESIGN.md` is updated before the IDD spec is written

#### Scenario: Audit and polish after TDD apply
- **GIVEN** implementation is complete
- **WHEN** the agent finalizes
- **THEN** `/impeccable audit` and `/impeccable polish` are run on the code

#### Scenario: Playwright verify after polish
- **GIVEN** impeccable polish is complete
- **WHEN** the agent verifies
- **THEN** Playwright runs visual smoke tests on changed routes

### Scenario: Full UI workflow order
- **GIVEN** a UI feature
- **WHEN** the agent executes the workflow
- **THEN** the order is: brainstorm → shape/craft → update DESIGN.md → IDD spec → BDD (optional) → TDD apply → audit/polish → Playwright verify → verification-before-completion
