## ADDED Requirements

### Requirement: User prompt persistence

The system SHALL provide `mem_save_prompt` and a `user_prompts` store (with FTS) for
recording user prompts so future sessions can recall what was asked.

#### Scenario: Save a prompt
- **GIVEN** a session in active use
- **WHEN** `mem_save_prompt` is called with prompt content
- **THEN** the prompt is stored in `user_prompts` with the project and session, and indexed in `prompts_fts`

#### Scenario: Truncated prompt warning
- **GIVEN** a prompt longer than the configured byte cap
- **WHEN** `mem_save_prompt` stores it
- **THEN** the response indicates truncation and a truncation metadata field is present

#### Scenario: Search prompts
- **GIVEN** prompts exist in the store
- **WHEN** prompt search is called with a query
- **THEN** matching prompts are returned ranked by FTS relevance

### Requirement: Observation review cycle

The system SHALL provide `mem_review` for listing observations due for local review and a
per-observation `review_after` timestamp.

#### Scenario: List observations due
- **GIVEN** observations exist with `review_after` in the past
- **WHEN** `mem_review` is called
- **THEN** those observations are returned, ordered by `review_after`

#### Scenario: Not-yet-due observations excluded
- **GIVEN** an observation with `review_after` in the future
- **WHEN** `mem_review` is called
- **THEN** that observation is not returned

#### Scenario: Mark reviewed resets the cycle
- **GIVEN** an observation returned by `mem_review`
- **WHEN** it is marked reviewed
- **THEN** its `review_after` is advanced so it is not immediately due again

### Requirement: Observation update and delete

The system SHALL provide `mem_update` and `mem_delete` for modifying and removing
observations, with soft-delete as the default.

#### Scenario: Update an observation field
- **GIVEN** an existing observation
- **WHEN** `mem_update` is called changing title or content
- **THEN** the field is updated, `updated_at` is bumped, and the FTS row is re-synced

#### Scenario: Soft delete
- **GIVEN** an existing observation
- **WHEN** `mem_delete` is called without `hard_delete`
- **THEN** the observation is soft-deleted (`deleted_at` set), its FTS row removed, and auto-edges removed
- **THEN** manual edges are preserved

#### Scenario: Hard delete
- **GIVEN** an existing observation
- **WHEN** `mem_delete` is called with `hard_delete=true`
- **THEN** the observation row is permanently removed and its FTS row and all edges are removed

### Requirement: Passive capture and relationship verdicts

The system SHALL provide `mem_capture_passive` for deterministic extraction of a
structured observation from a text block, and `graph_judge` for storing a relationship
verdict between two observations as a typed graph edge.

#### Scenario: Capture passive learning
- **GIVEN** a text block containing a structured learning
- **WHEN** `mem_capture_passive` is called with the content and a session
- **THEN** a structured observation is created from the block without an LLM round-trip

#### Scenario: Judge a conflicting relationship
- **GIVEN** two observations that conflict
- **WHEN** `graph_judge` is called with the two IDs and a `conflicts_with` verdict
- **THEN** a `CONFLICTS_WITH` edge is stored between their graph nodes with `confidence` and reason properties

#### Scenario: Judge a superseding relationship
- **GIVEN** two observations where one supersedes the other
- **WHEN** `graph_judge` is called with the two IDs and a `supersedes` verdict
- **THEN** a `SUPERSEDES` edge is stored between their graph nodes

#### Scenario: Not-a-conflict verdict removes the edge
- **GIVEN** two observations currently linked by a `CONFLICTS_WITH` edge
- **WHEN** `graph_judge` is called with a `not_conflict` verdict
- **THEN** the existing conflict edge is removed

#### Scenario: Verdict vocabulary
- **GIVEN** a `graph_judge` call with a verdict outside `related|compatible|scoped|conflicts_with|supersedes|not_conflict`
- **WHEN** validation runs
- **THEN** the call is rejected with the list of valid verdicts

### Requirement: Memory diagnostics and project context

The system SHALL provide `mem_current_project`, `mem_stats`, and `mem_doctor` for
agent-facing project resolution and store health.

#### Scenario: Current project
- **WHEN** `mem_current_project` is called
- **THEN** the detected project, its source, and the list of available projects are returned

#### Scenario: Memory stats
- **WHEN** `mem_stats` is called for a project
- **THEN** counts by observation type/scope, session counts, web cache totals, and graph size are returned

#### Scenario: Store doctor
- **WHEN** `mem_doctor` is called
- **THEN** schema version, FTS integrity (row counts vs. FTS counts), WAL state, last error, and disk size are reported
