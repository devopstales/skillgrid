## ADDED Requirements

### Requirement: Property graph storage

The system SHALL store a property graph in separate tables (`graph_nodes`, `graph_edges`)
within the project SQLite store, without modifying existing tables.

#### Scenario: Node upsert idempotency
- **GIVEN** a node is inserted with (project, kind=observation, ref_table=observations, ref_id=4, name="Fixed N+1 query")
- **WHEN** the same node is upserted again
- **THEN** exactly one graph_nodes row exists for that identity

#### Scenario: Edge type validation
- **GIVEN** the edge type vocabulary (CONTAINS, DEFINES, CALLS, IMPORTS, EXTENDS, MENTIONS, CITES, RELATED_TO, FOLLOWS_FROM)
- **WHEN** an edge is created with a type outside the vocabulary
- **THEN** the write is rejected with an error naming the accepted types

#### Scenario: Edge upsert idempotency
- **GIVEN** an edge (source, target, edge_type) exists
- **WHEN** the same edge is created again
- **THEN** exactly one graph_edges row exists and `properties_json` is updated in place

#### Scenario: Node deletion cascades to edges
- **GIVEN** node N has outgoing and incoming edges
- **WHEN** node N is deleted
- **THEN** all edges referencing N are removed

#### Scenario: Node name search
- **GIVEN** graph nodes exist with searchable names
- **WHEN** a node search query matches node names via FTS
- **THEN** matching nodes are returned with their kind and graph degree

### Requirement: Graph traversal

The system SHALL support depth-capped traversal over the graph using bounded queries.

#### Scenario: Neighbors within depth
- **GIVEN** a node with outgoing and incoming edges
- **WHEN** neighborhood is requested for depth 1
- **THEN** directly connected nodes and the connecting edges are returned

#### Scenario: Neighbor depth cap
- **GIVEN** a path of more than 4 reachable nodes
- **WHEN** neighborhood is requested with default depth
- **THEN** results are limited to the default depth and not all reachable nodes are returned

#### Scenario: Impact downstream
- **GIVEN** a symbol S that other symbols CALL
- **WHEN** downstream impact is requested for S
- **THEN** the symbols that call S are returned with hop distance and edge types

#### Scenario: Impact upstream
- **GIVEN** a symbol S that calls other symbols
- **WHEN** upstream impact is requested for S
- **THEN** the symbols S calls are returned with hop distance and edge types

#### Scenario: Truncated traversal
- **GIVEN** a graph exceeding the traversal node cap
- **WHEN** any traversal is requested
- **THEN** the result includes a `truncated` flag set to true and at most the cap of nodes

#### Scenario: Shortest path
- **GIVEN** two nodes A and B connected by a directed path over edge type CALLS
- **WHEN** trace is requested from A to B restricted to CALLS
- **THEN** the shortest directed path from A to B is returned with intermediate nodes and edges

#### Scenario: Traversal cycle safety
- **GIVEN** a graph containing a directed cycle
- **WHEN** neighborhood or impact traversal is requested
- **THEN** the traversal terminates and does not return repeated node paths
