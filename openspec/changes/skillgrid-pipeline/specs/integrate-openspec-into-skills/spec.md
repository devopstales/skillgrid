## ADDED Requirements

### Requirement: OpenSpec operations embedded in skills

The system SHALL embed OpenSpec operations as callable steps within skillgrid skills.

#### Scenario: Brainstorming triggers propose
- **GIVEN** the brainstorming skill reaches the "write proposal" step
- **WHEN** the user approves the design
- **THEN** the skill calls `openspec new change` and scaffolds the proposal

#### Scenario: Apply-change reads tasks
- **GIVEN** the openspec-apply-change skill is active
- **WHEN** it reads context files
- **THEN** it uses `openspec status --change --json` to get task progress

#### Scenario: Verify-change validates
- **GIVEN** the openspec-verify-change skill is active
- **WHEN** it validates implementation
- **THEN** it runs `openspec validate --type change --strict`

#### Scenario: Archive-change finalizes
- **GIVEN** the openspec-archive-change skill is active
- **WHEN** it archives a completed change
- **THEN** it runs `openspec archive` and confirms the move
