## ADDED Requirements

### Requirement: Quality gates (L5)

The system SHALL enforce quality gates (coverage, mutation, duplication) before merge.

#### Scenario: Branch coverage gate
- **GIVEN** a project with `branch_coverage_min: 80`
- **WHEN** coverage is measured on touched packages
- **THEN** merge is blocked if coverage is below the threshold

#### Scenario: Mutation testing (optional)
- **GIVEN** a project with mutation testing enabled
- **WHEN** quality gates run
- **THEN** `go-mutesting` (or stack-specific tool) executes on changed modules

#### Scenario: Test duplication check
- **GIVEN** a project with test duplication checking enabled
- **WHEN** quality gates run
- **THEN** `jscpd` checks for duplicated test code
