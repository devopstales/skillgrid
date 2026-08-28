## ADDED Requirements

### Requirement: Cypher query language

The system SHALL expose a full OpenCypher read-and-write query language as the single
`cypher` tool, superseding the bounded mini-Cypher subset defined by `mnemonic-graph`.
The tool parses, plans, and executes queries against the existing `graph_nodes` /
`graph_edges` store and returns a JSON record envelope.

#### Scenario: Single-pattern read matches
- **GIVEN** a graph with `:Symbol` nodes and `CALLS` edges
- **WHEN** `cypher` is called with `MATCH (a:Symbol)-[c:CALLS]->(b:Symbol) RETURN a.name, b.name`
- **THEN** the matching caller/callee pairs are returned as `{columns, rows}`

#### Scenario: Variable-length path read
- **GIVEN** a graph whose `CALLS` edges form a directed chain of length ≥ 3
- **WHEN** `cypher` is called with `MATCH (a:Symbol)-[*2..3]->(b:Symbol) RETURN a.name, b.name`
- **THEN** node pairs at path length 2 and 3 (and not 1 or 4+) are returned

#### Scenario: Unbounded variable-length is depth-capped
- **GIVEN** a graph with a `CALLS` chain longer than the configured variable-length depth cap
- **WHEN** `cypher` is called with `MATCH (a)-[*]->(b) RETURN a, b`
- **THEN** the traversal is bounded to the configured depth and the result is capped, not unbounded

#### Scenario: Projection, distinct, and aggregation
- **GIVEN** a graph with `:Symbol` nodes labelled by kind
- **WHEN** `cypher` is called with `MATCH (n:Symbol) RETURN n.kind, count(*) AS c ORDER BY c DESC`
- **THEN** per-kind counts are returned in descending order as `{columns, rows}`

#### Scenario: WITH and UNION
- **GIVEN** a graph with two disjoint subsets of nodes
- **WHEN** `cypher` is called with a multi-part `WITH … RETURN` query and a `UNION` / `UNION ALL`
- **THEN** the combined records are returned with the correct deduplication semantics

#### Scenario: UNWIND
- **GIVEN** a list of node names
- **WHEN** `cypher` is called with `UNWIND $names AS name MATCH (n {name: name}) RETURN n`
- **THEN** one row per matching node in the list is returned

#### Scenario: Parameters are typed and bound
- **GIVEN** the query `MATCH (a) WHERE a.x = $v RETURN a`
- **WHEN** `cypher` is called with the JSON param `{"v": 42}`
- **THEN** `$v` is bound as an integer and rows matching `x = 42` are returned
- **AND** a param of the wrong type is rejected at bind time, not at execution

#### Scenario: Syntax errors are reported with position
- **GIVEN** a query with a parse error
- **WHEN** `cypher` is called
- **THEN** the response is a JSON error with `kind: syntax` and line/column position
- **AND** the result is not a partial row set

#### Scenario: Unsupported constructs are named, not silently skipped
- **GIVEN** a query using a construct outside this surface (e.g., `CALL`, `LOAD CSV`, `FOREACH`, `CREATE CONSTRAINT`, `EXISTS{…}`)
- **WHEN** `cypher` is called
- **THEN** the response is a JSON error with `kind: unsupported` naming the construct
- **AND** no rows are returned and no partial write is applied

### Requirement: Cypher write language

The system SHALL support the OpenCypher write clauses — `CREATE`, `MERGE` (with
`ON CREATE` / `ON MATCH`), `SET` (props, `+=`, labels), `REMOVE` (props/labels), and
`DELETE` [DETACH] — executed transactionally.

#### Scenario: Create nodes and relationships
- **GIVEN** an empty graph namespace for a key
- **WHEN** `cypher` is called with `CREATE (a:Symbol {name:"n1"}), (b:Symbol {name:"n2"}), (a)-[:CALLS]->(b)`
- **THEN** the two nodes and the `CALLS` edge exist after the call
- **AND** the response includes write counters under `updates`

#### Scenario: MERGE find-then-act
- **GIVEN** a graph where a `Symbol {name:"n1"}` already exists
- **WHEN** `cypher` is called with `MERGE (n:Symbol {name:"n1"}) ON MATCH SET n.touched = 1 ON CREATE SET n.created = 1 RETURN n`
- **THEN** the existing node is updated with `touched` set, and no duplicate node is created

#### Scenario: MERGE creates when absent
- **GIVEN** a graph where a `Symbol {name:"new"}` does not exist
- **WHEN** `cypher` is called with `MERGE (n:Symbol {name:"new"}) ON CREATE SET n.created = 1 RETURN n`
- **THEN** the node is created with `created` set

#### Scenario: SET and REMOVE
- **GIVEN** an existing node
- **WHEN** `cypher` is called with `MATCH (n) SET n.x = n.x + 1, n:Label RETURN n` then `MATCH (n) REMOVE n:Label RETURN n`
- **THEN** the property is incremented and the label is added then removed

#### Scenario: DETACH DELETE cascades relationships
- **GIVEN** a node incident to relationships
- **WHEN** `cypher` is called with `MATCH (n {name:"victim"}) DETACH DELETE n`
- **THEN** the node and all incident edges are removed; non-incident nodes remain

#### Scenario: Plain DELETE refuses a connected node
- **GIVEN** a node with at least one incident relationship
- **WHEN** `cypher` is called with `MATCH (n {name:"victim"}) DELETE n` (no `DETACH`)
- **THEN** the call fails with a JSON error and the node is not deleted

#### Scenario: Writes are atomic
- **GIVEN** a write query that succeeds at the first clause and raises an error at a later clause
- **WHEN** `cypher` is called
- **THEN** all writes are rolled back: the first clause's created nodes are not present and no partial state is committed

### Requirement: Query cap enforcement

The system SHALL enforce configured caps — max rows, wall-clock timeout, and variable-length
depth — so a single query cannot run unbounded.

#### Scenario: Max rows caps the result
- **GIVEN** a query whose result set exceeds `cypher.max_rows`
- **WHEN** `cypher` is called
- **THEN** at most `max_rows` rows are returned and the envelope sets `truncated: true`

#### Scenario: Wall-clock timeout fails the query
- **GIVEN** a long-running expansion and `cypher.timeout_ms` set below its expected duration
- **WHEN** `cypher` is called
- **THEN** the call returns a JSON timeout error and no row set

#### Scenario: Read-only guard blocks writes
- **GIVEN** `cypher.readonly` is enabled (via config or `SKILLGRID_CYPHER_READONLY=1`)
- **WHEN** `cypher` is called with a write clause (`CREATE`, `MERGE`, `SET`, `REMOVE`, `DELETE`)
- **THEN** the call is rejected with a named "readonly" error
- **AND** read-only clauses (`MATCH`, `WHERE`, `RETURN`) still return rows
