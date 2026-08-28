## ADDED Requirements

### Requirement: Generic policy setup

The system SHALL provide a generic mechanism to scaffold and enable security policies.

#### Scenario: Policy scaffolded
- **GIVEN** a policy configuration is provided
- **WHEN** the policy setup runs
- **THEN** a policy file is created with the configured rules

#### Scenario: Policy validated
- **GIVEN** a policy file exists
- **WHEN** validation runs
- **THEN** syntax and semantic errors are reported

#### Scenario: Policy enforcement active
- **GIVEN** a valid policy is configured
- **WHEN** enforcement is enabled
- **THEN** agent tool calls violating the policy are blocked or flagged
