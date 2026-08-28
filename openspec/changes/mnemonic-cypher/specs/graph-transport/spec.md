## MODIFIED Requirements

### Requirement: Cypher MCP tool

The system SHALL expose a `cypher` tool that executes **full OpenCypher** (read and write)
queries against the existing `graph_nodes` / `graph_edges` store and returns a JSON
record envelope. This supersedes the bounded mini-Cypher subset defined in `mnemonic-graph`:
variable-length paths, `WITH`, `UNION`, `UNWIND`, `MERGE`, `SET`/`REMOVE`/`DELETE`, `+`,
functions, and aggregation are now fully supported clauses rather than "feature
unsupported" rejections.

#### Scenario: Mini-subset clauses remain valid
- **GIVEN** a single-pattern `MATCH` + `WHERE` + `RETURN` query (previously the supported
  subset)
- **WHEN** `cypher` is called
- **THEN** the matching node/edge set is returned as before, with the same `{columns, rows}`
  envelope

#### Scenario: Previously-rejected features are now implemented
- **GIVEN** a query using a construct that `mnemonic-graph` rejected with
  `cypher feature unsupported: <feature>` (e.g., a variable-length path, `OPTIONAL`-free
  `WITH`, `UNION`, or a `MERGE`)
- **WHEN** `cypher` is called
- **THEN** the feature executes normally and returns the correct result — no "feature
  unsupported" error

#### Scenario: Unsupported surface is still named, not silent
- **GIVEN** a query using a construct outside this change's surface (`CALL` procedures,
  `LOAD CSV`, `FOREACH`, `CREATE CONSTRAINT`, full `EXISTS{…}`)
- **WHEN** `cypher` is called
- **THEN** a JSON error with `kind: unsupported` naming the construct is returned
- **AND** no partial rows are returned and no partial write is applied

#### Scenario: Tool contract unchanged from mnemonic-graph
- **GIVEN** a `cypher` tool call over MCP
- **WHEN** the query is valid
- **THEN** the tool returns raw JSON through the existing `JSONResult` wrapper (OCBI
  convention), with `{columns, rows, truncated?}` — the shape defined by
  `mnemonic-graph`
- **AND** HTTP `POST /graph/cypher` (and alias `POST /graph/query`) accept the same
  `{query, params}` body and return the same envelope, with the same auth rules
