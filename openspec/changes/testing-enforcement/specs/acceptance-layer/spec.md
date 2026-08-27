## ADDED Requirements

### Requirement: Acceptance layer (L3) BDD

The system SHALL require acceptance tests (macro BDD) from Gherkin scenarios when BDD is enabled.

#### Scenario: RED before apply
- **GIVEN** a BDD-enabled feature
- **WHEN** the agent starts implementation
- **THEN** acceptance scenarios are extracted and run RED before code is written

#### Scenario: GREEN before promote
- **GIVEN** implementation is complete
- **WHEN** the agent requests promote
- **THEN** acceptance tests must be GREEN

#### Scenario: Gherkin lint
- **GIVEN** extracted `.feature` files
- **WHEN** acceptance tests run
- **THEN** `gherkin-lint` validates the Gherkin syntax

#### Scenario: Acceptance runner
- **GIVEN** a project with BDD enabled
- **WHEN** acceptance tests run
- **THEN** `cucumber-js` (or stack-specific runner) executes the scenarios
