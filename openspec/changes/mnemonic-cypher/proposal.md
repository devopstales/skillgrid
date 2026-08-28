## Why

The in-flight `mnemonic-graph` change ships a `cypher` tool scoped as a **bounded mini-Cypher
subset** (single-pattern `MATCH` + `WHERE` on known props + minimal `RETURN`) that rejects
variable-length paths, `WITH`, `UNION`, `MERGE`, and aggregation with a structured "feature
unsupported" error. That is the right first cut — but it leaves out the GitNexus `cypher`
capability (one of its 17 tools), which is raw graph querying of exactly the
variable-length/`WITH`/`MERGE`/aggregate kind agents reach for once the high-level tools
stop fitting the question ("give me every symbol transitively reachable from this file up to
3 hops, grouped by kind").

This change is the "**full Cypher for later**" piece: a full OpenCypher **read-and-write**
query engine that **supersedes and upgrades** the same `cypher` tool — one tool, full
behavior — landing after `mnemonic-graph` so it upgrades that tool rather than adding a
second.

## What Changes

- New `cypher` module (`skillgrid-cli/internal/mnemonic/cypher/`) — a full OpenCypher
  query engine (parser + IR planner/executor) over the existing `graph_nodes` /
  `graph_edges` store from `mnemonic-graph`.
- **Parser**: ANTLR-generated Go parser from the ANTLR Cypher grammar (BSD-2,
  `grammars-v4`), covering the full read-write OpenCypher scope used here — `MATCH` /
  `OPTIONAL MATCH`, variable-length relationships (`*`, `{1..3}`), full `WHERE` expressions,
  `CREATE` / `MERGE` (with `ON CREATE` / `ON MATCH`) / `SET` (incl. `+=`, labels) / `REMOVE`
  / `DELETE` (`DETACH`), `WITH`, `ORDER BY` / `SKIP` / `LIMIT`, `UNION` / `UNION ALL`, `UNWIND`,
  `RETURN` [`DISTINCT`] with aggregation, and parameters.
- **Executor**: IR operators over the SQLite graph store — scan / pattern-match / bounded
  variable-length expand (DFS + cycle guard) / filter / project + aggregate (SQL `GROUP BY`)
  / create / set / remove / delete (transactional DML) / merge (find-then-act) / unwind /
  union — with query caps (default rows, timeout, variable-length hop bounds) in
  `config.d/indexing.yaml` under a `cypher:` block.
- **Supersedes** the mini-Cypher `cypher` in `mnemonic-graph`: the single `cypher` MCP tool
  and `POST /graph/cypher` now accept the full read-write language and return a JSON
  `{columns, rows}` envelope; `mnemonic-graph`'s "feature unsupported" branches become
  implemented clauses. Adds a `POST /graph/query` alias.
- New capability `graph-cypher`; a `graph-transport` delta re-scopes the existing `cypher`
  requirement (mini-subset → full engine).
- Documentation + glossary: full-Cypher surface, IR operators, and the query-limit
  vocabulary; the `mnemonic-graph` non-goal "full Cypher" is retired.

**No breaking changes** to the *graph model* (no new node/edge tables); new deps are
build-time (ANTLR Go runtime + generated parser).

## Capabilities

### New Capabilities

- `graph-cypher`: Full OpenCypher read-and-write query language (parser, IR planner,
  SQLite-backed executor, query limits) as the single `cypher` tool surface

### Modified Capabilities

- `graph-transport`: The `cypher` MCP tool / `POST /graph/cypher` re-scoped from the
  mini-Cypher subset to the full read-write engine (supersedes `mnemonic-graph`)

## Impact

- **Depends on**: `mnemonic-graph` (graph store + `graph_nodes`/`graph_edges` + the
  `cypher` transport) — must be implemented first
- **Code**: `skillgrid-cli/{internal/mnemonic/cypher, internal/mnemonic/http, internal/mnemonic/mcp,
  internal/mnemonic/service, cmd/skillgrid}` + generated ANTLR artifacts
- **Deps**: ANTLR Go runtime (`github.com/antlr/antlr4/runtime/Go/antlr/v4`) + generated
  grammar files; pure-Go (no runtime cgo)
- **Data**: No new tables — reads/writes the existing `graph_nodes` / `graph_edges`
- **Config**: New `cypher:` block in `config.d/indexing.yaml` (rows timeout, hop bounds,
  readonly flag)
