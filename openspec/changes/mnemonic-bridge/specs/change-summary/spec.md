## ADDED Requirements

### Requirement: `change_summary` MCP tool

The system SHALL provide a `change_summary` MCP tool that returns a
git-diff digest for the current working tree (or an explicit `repo` path)
without requiring GitNexus to be installed.

#### Scenario: Basic diff summary

- **GIVEN** the working tree has uncommitted changes in 3 files
- **WHEN** the agent calls `change_summary {}`
- **THEN** the response contains `files` (array of relative paths),
  `symbols` (array of added/removed symbol names), and `summary`
  (one-line natural language, e.g. "added 2 handlers in `internal/auth/`,
  touched 3 callers in `src/api/`")

#### Scenario: Explicit repo path

- **GIVEN** the agent is not standing in the target repository
- **WHEN** it calls `change_summary { repo: "/abs/path/my-app" }`
- **THEN** the response reflects the diff in that repository, not cwd

#### Scenario: Clean working tree

- **GIVEN** the working tree matches HEAD
- **WHEN** the agent calls `change_summary {}`
- **THEN** the response contains empty `files` and `symbols` arrays and
  `summary: "working tree clean"`

#### Scenario: No git repository

- **GIVEN** the cwd is not inside a git repository
- **WHEN** the agent calls `change_summary {}`
- **THEN** the tool returns a clear error message (not a stack trace)

#### Scenario: Agent uses output in session summary

- **GIVEN** the agent called `change_summary` and received a non-empty result
- **WHEN** it calls `mem_session_summary` with the diff block appended
- **THEN** the next session can read the summary and see both what was
  decided and what was changed
