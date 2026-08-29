## ADDED Requirements

### Requirement: `code_status` response includes `graph_stale`

The `code_status` MCP tool response SHALL include a `graph_stale: bool`
field indicating whether the GitNexus index for the same repository is
stale relative to the last Mnemonic `code_index` run.

#### Scenario: Both indexes fresh

- **GIVEN** Mnemonic `code_index` ran within the staleness window
  AND GitNexus `analyze` ran within its staleness window
- **WHEN** the agent calls `code_status {}`
- **THEN** the response contains `"stale": false` and `"graph_stale": false`

#### Scenario: Mnemonic fresh, GitNexus stale

- **GIVEN** Mnemonic `code_index` is fresh
  AND `.gitnexus/` mtime is older than the last `code_index` run
- **WHEN** the agent calls `code_status {}`
- **THEN** the response contains `"stale": false` and `"graph_stale": true`

#### Scenario: GitNexus absent

- **GIVEN** the repository has no `.gitnexus/` directory and no
  GitNexus registry entry
- **WHEN** the agent calls `code_status {}`
- **THEN** the response contains `"graph_stale": false` (absence is not staleness)

#### Scenario: Agent pairs the two signals

- **GIVEN** the agent reads `code_status` and sees `"stale": false,
  graph_stale: true`
- **WHEN** it reports index health to the user
- **THEN** the report mentions both indexes explicitly and recommends
  `node .gitnexus/run.cjs analyze` for the graph side

#### Scenario: `code_status` response schema is backward-compatible

- **GIVEN** an existing agent parses `code_status` by field name
- **WHEN** the response includes the new `graph_stale` field
- **THEN** the agent ignores the unknown field and continues to function
  (no required-field breakage)
