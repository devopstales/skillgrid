## ADDED Requirements

### Requirement: Unit layer (L1) strict TDD

The system SHALL enforce strict TDD (red-before-green) for unit tests during apply.

#### Scenario: Failing test written first
- **GIVEN** a new behavior is being implemented
- **WHEN** the agent starts implementation
- **THEN** a failing unit test is written before production code

#### Scenario: Failure for missing feature
- **GIVEN** a test is written
- **WHEN** it fails
- **THEN** the failure is for a missing feature (not a typo or error)

#### Scenario: Minimal code to green
- **GIVEN** a failing test exists
- **WHEN** the agent writes production code
- **THEN** only the minimal code to make the test green is written

#### Scenario: Full suite still green
- **GIVEN** a test passes after implementation
- **WHEN** the full test suite runs
- **THEN** all existing tests still pass

### Requirement: Integration layer (L2)

The system SHALL require integration tests when design crosses module boundaries.

#### Scenario: Boundary crossing triggers L2
- **GIVEN** a design that crosses module boundaries
- **WHEN** the agent plans testing
- **THEN** integration tests are required
