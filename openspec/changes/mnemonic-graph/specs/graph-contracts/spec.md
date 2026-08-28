## ADDED Requirements

### Requirement: API route extraction

The system SHALL extract API routes (method + path + handler) from framework
route-registration patterns during code indexing, when contract extraction is enabled,
for the enabled frameworks and languages.

#### Scenario: Extract an HTTP route (Go)
- **GIVEN** a Go file registering an HTTP route with a handler function
- **WHEN** code indexing runs with `contracts.enabled=true` and the `http` framework is enabled
- **THEN** a `route` node exists with `method`, `path`, `framework`, `handler_symbol`, `file_path` properties
- **THEN** a `DEFINES_ROUTE` edge connects the route node to its handler symbol
- **THEN** an `ENTRY_POINT_OF` edge or equivalent links the route to its entry symbol

#### Scenario: Extract an HTTP route (TypeScript/Python)
- **GIVEN** a TypeScript or Python file declaring a route handler for an HTTP framework
- **WHEN** code indexing runs with `contracts.enabled=true` and the `http` framework is enabled
- **THEN** a `route` node exists for each declared route

#### Scenario: Route extraction disabled
- **GIVEN** `contracts.enabled=false` in config
- **WHEN** code indexing runs
- **THEN** no `route`, `tool`, or `shape` nodes or contract edges are created

### Requirement: MCP/RPC tool extraction

The system SHALL extract MCP/RPC tool definitions (name + framework + handler) during
code indexing, when contract extraction is enabled with the `mcp` framework.

#### Scenario: Extract an MCP tool
- **GIVEN** a Go, TypeScript, or Python file defining an MCP or RPC tool with a handler
- **WHEN** code indexing runs with `contracts.enabled=true` and the `mcp` framework is enabled
- **THEN** a `tool` node exists with `name`, `framework`, `handler_symbol`, `file_path` properties
- **THEN** a `HANDLES_TOOL` edge connects the provider/server to the tool

### Requirement: Response shape extraction and consumer mapping

The system SHALL extract provider response shapes (field sets) per route and map
consumer property accesses to them, so `shape_check` can report a field diff per route.

#### Scenario: Extract a response shape
- **GIVEN** a route handler whose return expression has an identifiable field set (struct literal, map keys, or JSON tags)
- **WHEN** code indexing runs with `contracts.enabled=true`
- **THEN** a `shape` node exists for the route with a `fields` property (array of field names)

#### Scenario: Consumer fetch and property access
- **GIVEN** a component that fetches a route and reads one of its response fields
- **WHEN** code indexing runs with `contracts.enabled=true`
- **THEN** a `FETCHES` edge connects the consumer to the route
- **THEN** a `CONSUMES_PROP` edge or `shape` property records which field the consumer reads

### Requirement: Contract subgraph reads

The system SHALL expose the contract subgraph through the `route_map`, `tool_map`,
`shape_check`, and `api_impact` tools as reads over the persisted graph.

#### Scenario: Route map
- **WHEN** `route_map` is called for a project
- **THEN** routes → handlers → fetchers are returned grouped by framework, with entry points

#### Scenario: Tool map
- **WHEN** `tool_map` is called for a project
- **THEN** MCP/RPC tools → definitions → handlers → callers are returned

#### Scenario: Shape check reports a diff, never a silent pass
- **GIVEN** a route whose shape is missing a field a consumer reads
- **WHEN** `shape_check` is called for that route
- **THEN** the response reports matched, missing, and extra fields per consumer

#### Scenario: API impact over the contract subgraph
- **GIVEN** a route handler that feeds one or more consumers
- **WHEN** `api_impact` is called for that route
- **THEN** the downstream consumers/tools/routes it feeds are returned with hop depths

#### Scenario: Incremental contract regeneration
- **GIVEN** a file with persisted route/tool/shape nodes
- **WHEN** the file changes and code indexing runs
- **THEN** pass-scoped (`pass='contracts'`) nodes and edges are regenerated; other passes are preserved
