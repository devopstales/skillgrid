## ADDED Requirements

### Requirement: Static layer (L0) enforcement

Every project SHALL run static analysis (lint, format, typecheck) on every commit and in CI.

#### Scenario: Go static checks
- **GIVEN** a Go project with `testing-capabilities.yaml` declaring L0
- **WHEN** a commit is made or CI runs
- **THEN** `go vet`, `staticcheck`, and `gofmt -l` are executed

#### Scenario: Node static checks
- **GIVEN** a Node project with `testing-capabilities.yaml` declaring L0
- **WHEN** a commit is made or CI runs
- **THEN** `eslint`, `tsc`, and `prettier --check` are executed

#### Scenario: L0 failure blocks commit
- **GIVEN** static analysis finds issues
- **WHEN** the results are evaluated
- **THEN** the commit/CI is blocked until issues are resolved

### Requirement: Test layer matrix

The system SHALL define which test layers are required for each change type.

#### Scenario: Bug fix requires L0 + L1
- **GIVEN** a bug fix change type
- **WHEN** the agent plans testing
- **THEN** L0 (static) and L1 (unit repro test) are required

#### Scenario: IDD + BDD feature requires L0 + L1 + L3
- **GIVEN** an IDD + BDD feature change type
- **WHEN** the agent plans testing
- **THEN** L0, L1, and L3 (acceptance) are required

#### Scenario: Spike requires no tests
- **GIVEN** a spike change type
- **WHEN** the agent plans testing
- **THEN** no test layers are required
