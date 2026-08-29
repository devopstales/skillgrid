## ADDED Requirements

### Requirement: `session_diff_summary` MCP tool

The system SHALL provide a `session_diff_summary` MCP tool that produces
a Markdown diff block the agent can append to a `mem_session_summary` body.
The tool is read-only — it does not write the summary itself.

#### Scenario: Basic diff block

- **GIVEN** the working tree has changes in 3 files
- **WHEN** the agent calls `session_diff_summary { session_id: "<id>" }`
- **THEN** the response is a Markdown block:

```
## Diff Summary
- Changed files: 3
  - src/auth/login.ts
  - src/auth/middleware.ts
  - src/api/routes.ts
- Added symbols: 2 (authenticateUser, TokenRefresh)
- Removed symbols: 1 (validateUser)
- Risk: MEDIUM
```

#### Scenario: Explicit repo path

- **GIVEN** the agent is not standing in the target repository
- **WHEN** it calls `session_diff_summary { session_id: "<id>", repo: "/abs/path/my-app" }`
- **THEN** the diff block reflects that repository

#### Scenario: Agent folds block into session summary

- **GIVEN** the agent received a non-empty diff block
- **WHEN** it calls `mem_session_summary(session_id, "<existing body>\n\n<diff block>")`
- **THEN** the summary is persisted with the diff appended

#### Scenario: Clean working tree

- **GIVEN** the working tree matches HEAD
- **WHEN** the agent calls `session_diff_summary { session_id: "<id>" }`
- **THEN** the response block reads "working tree clean — no diff"

#### Scenario: No session found

- **GIVEN** the supplied `session_id` does not exist
- **WHEN** the agent calls `session_diff_summary { session_id: "bogus" }`
- **THEN** the tool returns a clear error (not a stack trace)
