## ADDED Requirements

### Requirement: Memory observation save and search

The system SHALL provide `mem_save` and `mem_search` tools for curated observations with FTS5 search.

#### Scenario: Save observation
- **GIVEN** an agent calls `mem_save` with type, title, content
- **WHEN** the tool executes
- **THEN** the observation is stored in SQLite and indexed for FTS search

#### Scenario: Search observations
- **GIVEN** observations exist in the store
- **WHEN** an agent calls `mem_search` with a query
- **THEN** relevant observations are returned ranked by FTS relevance

#### Scenario: Topic key upsert
- **GIVEN** an observation with `topic_key` already exists
- **WHEN** `mem_save` is called with the same `topic_key`
- **THEN** the existing observation is updated (not duplicated)

#### Scenario: Session lifecycle
- **GIVEN** `mem_session_start` is called
- **WHEN** a new session begins
- **THEN** a session record is created and returned

#### Scenario: Session summary
- **GIVEN** `mem_session_summary` is called at session end
- **WHEN** the session ends
- **THEN** the summary is stored with the session record

### Requirement: Memory taxonomy

Observations SHALL use `type` + `scope` fields for consistent routing.

#### Scenario: Type scope validation
- **GIVEN** `mem_save` is called with an invalid `type`
- **WHEN** validation runs
- **THEN** the observation is rejected with a clear error

### Requirement: Bi-temporal evolution (v1.1)

The system SHALL support `mem_replace` and `mem_history` for observation evolution.

#### Scenario: Replace preserves history
- **GIVEN** an observation exists with `topic_key`
- **WHEN** `mem_replace` is called on that key
- **THEN** the prior content moves to `observation_history` and the new content becomes active
