## ADDED Requirements

### Requirement: E2E layer (L4)

The system SHALL require E2E tests only for UI-heavy features.

#### Scenario: E2E for UI features
- **GIVEN** a UI-heavy feature with `testing-capabilities.yaml` declaring L4 available
- **WHEN** E2E tests are required
- **THEN** Playwright runs the browser-level scenarios

#### Scenario: E2E not required for non-UI
- **GIVEN** a non-UI feature
- **WHEN** the agent plans testing
- **THEN** E2E tests are not required
