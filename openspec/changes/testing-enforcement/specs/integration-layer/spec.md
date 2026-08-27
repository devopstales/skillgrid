## ADDED Requirements

### Requirement: Integration test execution

The system SHALL run integration tests when the design crosses module boundaries.

#### Scenario: Go integration tests
- **GIVEN** a Go project with `testing-capabilities.yaml` declaring L2 available
- **WHEN** integration tests are required
- **THEN** `go test ./... -tags=integration` is executed

#### Scenario: Integration failure blocks promote
- **GIVEN** integration tests fail
- **WHEN** the results are evaluated
- **THEN** promote/merge is blocked until tests pass
