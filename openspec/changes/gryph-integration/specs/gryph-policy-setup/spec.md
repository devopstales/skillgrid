## ADDED Requirements

### Requirement: Policy scaffolded and enabled

After hooks are installed, the gryph step SHALL scaffold, validate, and enable the gryph security policy.

#### Scenario: Policy init runs
- **GIVEN** hooks are installed successfully
- **WHEN** the policy sub-step executes
- **THEN** `gryph policy init` scaffolds the default policy

#### Scenario: Policy validate runs
- **GIVEN** `gryph policy init` succeeded
- **WHEN** the policy sub-step continues
- **THEN** `gryph policy validate` is invoked

#### Scenario: Policy enabled
- **GIVEN** validation succeeded
- **WHEN** the policy sub-step completes
- **THEN** `gryph config set policy.enabled true` is invoked

#### Scenario: Policy failure is non-fatal
- **GIVEN** any policy command fails
- **WHEN** the failure occurs
- **THEN** a warning is logged and the install continues
