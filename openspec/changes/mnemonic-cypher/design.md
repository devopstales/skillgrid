## Context

The in-flight `mnemonic-graph` change adds a property graph to Mnemonic
(`graph_nodes` / `graph_edges` in the existing per-project SQLite store) plus a `cypher`
tool scoped as a **bounded mini-Cypher subset** (single-pattern `MATCH` + `WHERE` + minimal
`RETURN`) that deliberately rejects variable-length paths, `WITH`, `UNION`, `MERGE`, and
aggregation with a structured "feature unsupported" error. That split was the right
first cut: get high-level tools shipping while deferring the heavyweight language surface.

This change is the **full-Cypher-for-later** piece. It supersedes that subset with a full
OpenCypher **read-and-write** engine, reusing the same `cypher` tool name (one tool, full
behavior). It is a new change (`mnemonic-cypher`) that depends on `mnemonic-graph` and
lands after it.

Reference points:

- **GitNexus** ships a `cypher` tool that runs raw Cypher over the graph (one of its 17
  tools) — the capability this change restores in Mnemonic, plus the read-write clauses
  GitNexus exposes.
- **OpenCypher** (`opencypher.org`) is the language subset; the **ANTLR Cypher grammar**
  (`grammars-v4`, BSD-2) is a maintained, complete grammar for it that we generate a
  parser from.
- **Mnemonic constraints**: single Go binary, pure-Go runtime (no cgo), SQLite is the
  only engine, embedded static assets for UI, additive to the `mnemonic-graph` schema.

## Goals / Non-Goals

**Goals:**

- Full OpenCypher **read** scope: `MATCH` / `OPTIONAL MATCH`, variable-length
  relationships (`*`, `{1..3}`, `{..}`, `+`), full `WHERE` expression language, `WITH`,
  `UNWIND`, `UNION` / `UNION ALL`, `ORDER BY` / `SKIP` / `LIMIT`, `RETURN` [`DISTINCT`]
  with aggregation (`count`/`sum`/`avg`/`min`/`max`/`collect`), parameters (`$param`),
  functions, list/map/predicate expressions.
- Full OpenCypher **write** scope: `CREATE`, `MERGE` (with `ON CREATE` / `ON MATCH`),
  `SET` (incl. `+=`, label addition), `REMOVE` (props / labels), `DELETE` [DETACH].
- One unified `cypher` tool — **supersedes** `mnemonic-graph`'s mini-subset; no second
  `cypher`-like tool. The tool's read/write surface is the full language.
- Deterministic, capped execution (row bounds, wall-clock timeout, variable-length hop
  bounds) so one agent query cannot hang the store.
- Transactional writes: a `CREATE`/`MERGE`/`SET`/`DELETE` query is atomic — either all
  commits or all rolls back.
- Parser from a **maintained, full grammar** (ANTLR) — not hand-rolled.

**Non-Goals:**

- A full Cypher *semantic* feature matrix (indexes, constraints DDL, `CALL` procedures,
  user-defined functions, temporal types, `LOAD CSV`, full `EXISTS{}` subqueries,
  `FOREACH`, `REDUCE`) — out of scope; the grammar parseable here is the **OpenCypher
  query surface** (read + write clauses), not schema DDL or procedure calls.
- Cross-project (cross-database) queries — single project store, as everywhere in Mnemonic.
- Pushing the engine out to a graph database — SQLite remains the store; we translate to
  SQL/CTE, not to a separate engine.
- A Cypher-specific REPL/CLI beyond the existing `skillgrid mcp` / `serve` transports.

## Decisions

### 1. ANTLR-generated parser + IR plan/execute over SQL (Approach A)

**Decision:** Generate a **Go parser** from the **ANTLR Cypher grammar**
(`antlr/grammars-v4`, BSD-2) — a complete OpenCypher/Neo4j-Cypher grammar with a lexer and
parser (`CypherLexer.g4` + `CypherParser.g4`). A Go **visitor** walks the generated AST
into a small **intermediate representation (IR)**: a linear pipeline of operations
(matching, filtering, aggregation, projection, create/merge/set/delete, unwind, union)
plus a per-op binding scope. A **planner/executor** translates IR ops to **SQL/CTE over
`graph_nodes` / `graph_edges`** (joins for relationship patterns, `GROUP BY` for
aggregation, `IN`/`NOT IN`/`EXISTS` for predicates, transactions for write ops), applying
uniform **query caps** (max rows, wall-clock timeout, variable-length depth bound) from
`config.d/indexing.yaml`.

**Alternatives considered and *documented for later revisiting*:**

- **Approach B — hand-rolled recursive-descent parser in Go.**
  No grammar tooling, smaller dependency surface, fully owned parser. *Why not now:*
  the read-write Cypher surface (expressions, `MERGE` actions, quantified relationships,
  list/map comprehensions, `UNION`, `WITH` scoping) is large and grammar-adjacent
  correctness is exactly where hand parsers rot; a maintained grammar is the safer source
  of truth for "what is a valid query." *Revisit when:* the ANTLR-generated artifact bloat
  or codegen friction becomes a real cost, or we need a parser with a very small footprint
  (e.g., embedding into a hot path) — a hand parser is the right tool then.
- **Approach C — delegate to a dedicated graph engine (Neo4j / Kuzu / Ladybug).**
  Offload parse + execute to a purpose-built Cypher database; Mnemonic stores/queries
  against it. *Why not now:* it introduces a second runtime dependency and a deployment
  mode (embedded vs. server) that conflicts with the single-binary, pure-Go-SQLite
  runtime story of Mnemonic; and it moves the execution model out of our control (caps,
  transactions, project scoping). *Revisit when:* query plans outgrow what SQLite/CTE can
  express efficiently (e.g., heavy multi-hop analytics at scale), or a single shared
  queryable graph across projects becomes a hard requirement — a dedicated engine is the
  natural fit then.

- **`gopher-cypher` (Go) as the parser/executor** — rejected for the executor (its
  driver targets a Neo4j/bolt server, not our in-store executor), but its *parser/AST*
  is a viable substitute within Approach A if the ANTLR codegen is problematic; it is the
  first fallback, not a separate approach.

**Rationale:** A maintained full grammar is the source of truth for *what* we accept; the
IR keeps *how* it executes decoupled and testable, and SQL/CTE keeps the runtime pure-Go
and single-binary. The two heavier alternatives (B, C) are the documented escalation
paths if A's trade-offs stop being acceptable.

### 2. One `cypher` tool, upgraded (supersede, not add)

**Decision:** The `cypher` MCP tool and `POST /graph/cypher` (plus a `POST /graph/query`
alias) now accept the full read-write language and return `{columns, rows}`. The
`mnemonic-graph` mini-subset branches ("feature unsupported: …") become *implemented*:
the same tool, now larger. A single tool avoids agent confusion between two
"query-the-graph" tools.

**Consequence for the in-flight change:** `mnemonic-graph`'s `graph-transport`
spec `Requirement: cypher` is **superseded** by `mnemonic-cypher`'s `graph-transport`
MODIFIED delta (full read-write). No new tool name. `mnemonic-graph` remains the owner of
the *graph store* and *transport plumbing*; `mnemonic-cypher` owns the *language engine*.

### 3. Read-write is the default surface; opt-in write guard

Cypher read-write is on by default (per the user's scope choice). A config/env guard,
`SKILLGRID_CYPHER_READONLY=1` (or `cypher.readonly: true` in `indexing.yaml`), rejects
`CREATE`/`MERGE`/`SET`/`REMOVE`/`DELETE` clauses, mirroring GitNexus's read-only MCP
mode. Agents in read-only agent modes get a hard boundary without a separate tool.

### 4. Execution model: IR → SQL plan, with uniform caps

- **Planner** validates clause ordering (read → optional write → `RETURN`), single-part
  vs. multi-part (`WITH`) queries, and binding scopes; it *rejects* unsupported
  constructs (procedure `CALL`, `LOAD CSV`, full `EXISTS{}` subqueries) with a structured
  error naming the construct — so "not in this surface" is explicit, not a silent no-op.
- **Executor** streams rows; **caps** applied at the boundary:
  - `cypher.max_rows` (default example: `10000`) — `LIMIT` applied, `truncated:true` set
  - `cypher.timeout_ms` (default example: `5000`) — `context.WithTimeout` on the whole
    query
  - `cypher.max_varlen_depth` (default example: `5`) — bounds `*`/`{n..m}` expansions
- **Writes** run inside a single transaction; any mid-query error rolls back all writes
  (SQLite `BEGIN`/`ROLLBACK`).
- **`DETACH DELETE`** issues `DELETE` and leaves edges (they cascade via FK
  `ON DELETE CASCADE` on `graph_edges.source_id`/`target_id`); plain `DELETE` refuses
  (400) any node still incident to edges — matches OpenCypher semantics.

### 5. Query envelope & parameters

- MCP `cypher(query, params?)` → `{columns: [...], rows: [[...], ...], truncated?: bool}`
  (raw JSON, OCBI convention). Write tools return the same envelope with an added
  `updates: {created, deleted, set, merged}` counter summary.
- `params` is a JSON object; the parser binds `$name` references, the executor substitutes
  typed values (no string interpolation — injection-safe by construction).
- `POST /graph/cypher` body is the same `{query, params}`; auth per the existing
  write-route rules when `SKILLGRID_HTTP_TOKEN` set (writes only).

### 6. Schema & dependency notes

- **No new tables.** `mnemonic-cypher` reads and writes `graph_nodes` / `graph_edges`
  (and their FTS) owned by `mnemonic-graph`. This change adds no migrations.
- **Config** `cypher:` block in `config.d/indexing.yaml` (max_rows, timeout_ms,
  max_varlen_depth, readonly). Values are ceilings, not behavior toggles.
- **Build:** `//go:generate antlr4 -Dlanguage=Go …` to materialize the parser/lexer Go
  sources under `internal/mnemonic/cypher/generated/`. Commit the generated sources
  (deterministic build; no ANTLR needed at build time after generate). An ANTLR Java or
  Go `antlr4` generator is a *build-time* tool dependency only.
- **Runtime deps:** `github.com/antlr/antlr4/runtime/Go/antlr/v4` (pure Go).

## Architecture

### Component (inside `internal/mnemonic`)

```mermaid
flowchart TB
  subgraph cypher module
    GRAM[ANTLR Cypher grammar<br/>CypherLexer.g4 + CypherParser.g4]
    GEN[generated Go parser/lexer<br/>//go:generate antlr4]
    VIS[visitor: AST -> IR ops + scopes]
    PLAN[planner: validate clauses, ordering,<br/>scopes; pick SQL strategy]
    EXEC[executor: IR -> SQL/CTE over store<br/>caps + transactions]
    GRAM --> GEN --> VIS --> PLAN --> EXEC
  end
  SVC[service.Service facade]
  MCP[graph-transport cypher tool<br/>(upgraded, full read-write)]
  HTTP[POST /graph/cypher + alias /graph/query]
  SVC --> MCP
  SVC --> HTTP
  MCP --> EXEC
  HTTP --> EXEC
  EXEC -->|SQL / CTE / DML tx| STORE[(graph_nodes<br/>graph_edges)]
```

### Query lifecycle (single `cypher` call)

1. **Parse** (generated ANTLR parser + lexer) — lexical/syntactic errors surfaced with
   line/column (OCBI JSON error).
2. **Bind** `$params` → typed bindings.
3. **IR-build** (visitor) → ordered list of ops with explicit binding scopes.
4. **Plan & validate** — clause ordering, single/multi-part, scoping; reject
   out-of-surface constructs (`CALL`, `LOAD CSV`, `EXISTS{}`) with a named error.
5. **Execute** — translate to SQL/CTE; apply caps (rows/timeout/depth); for write
   clauses, a single transaction; return `{columns, rows, truncated?}` (plus update
   counters for writes).

## API surface

### MCP tool (upgraded, single `cypher`)

| Tool | Behavior |
|---|---|
| `cypher(query, params?)` | Execute a full OpenCypher **read or write** query against the graph store; params are JSON-bound `$name` placeholders; returns `{columns, rows, truncated?}` (read) or the same plus `updates: {created, deleted, set, merged}` (write) |

**Read scope** (all supported): `MATCH`, `OPTIONAL MATCH`, `WHERE` (full expressions,
`IN`/`CONTAINS`/`STARTS WITH`/`ENDS WITH`, `IS NULL`, `EXISTS` prop/rel, list/map/predicate
expressions, `IN` on lists, function calls `count`/`size`/`type`/`labels`/`keys`, `CASE`),
`CREATE`, `MERGE` (`ON CREATE`/`ON MATCH`), `SET` (props, `+=`, labels), `REMOVE` (props,
labels), `DELETE` [DETACH], `WITH` (incl. aggregation before it), `ORDER BY` / `SKIP` /
`LIMIT`, `UNION` / `UNION ALL`, `UNWIND`, `RETURN` [`DISTINCT`], `*` in projections/returns,
parameters `$p`.

**Write semantics** (all supported): transactional `CREATE`/`MERGE`/`SET`/`REMOVE`/
`DELETE`; `MERGE` find-then-create/update with `ON CREATE`/`ON MATCH` actions.

**Out of this surface** (parse succeeds only if the construct is in the grammar *query*
scope; the planner rejects these with a named error rather than silently no-op):
standalone `CALL` / procedures, `LOAD CSV`, `FOREACH`, full `EXISTS {…}` subquery,
`CREATE CONSTRAINT` / `CREATE INDEX` DDL, `FOREACH` and other DDL.

### HTTP routes

| Route | Purpose |
|---|---|
| `POST /graph/cypher` | `{query, params}` → `{columns, rows, truncated?, updates?}` (auth per write-route rules when token set; read also allowed) |
| `POST /graph/query` | Alias of `POST /graph/cypher` (discoverability for agents) |

## Error handling

- **Syntax errors** → `{"error": {"kind":"syntax","line":N,"column":M,"message":"…"}}`
  (line/column from the ANTLR parse exception).
- **Semantics errors** (unknown relationship type on write, `MERGE` pattern not a
  single node, `DETACH` on a non-node, write clause after a read clause, etc.) →
  `{"error": {"kind":"semantics","message":"…"}}`.
- **Out-of-surface** constructs → `{"error": {"kind":"unsupported","construct":"CALL"…}}`.
- **Cap exceeded** → `truncated: true` (rows) or `timeout` error (not a row set);
  `max_varlen_depth` exceeded → `semantic: "variable-length depth limit exceeded"`.
- **Write failure** mid-transaction → rollback; `{"error": {"kind":"transaction", "message":"…"}}`.
- **Param type mismatch** → `{"error": {"kind":"param","name":"…","expected":"…"}}`.

## Testing

Per existing conventions (httptest + seeded stores, no mocks; the *store* is real
SQLite, the *engine* under test is isolated and testable against a fixture DB):

1. **Grammar coverage** — parse every supported clause in at least one query; assert
   parse *success*; assert each out-of-surface construct (`CALL`, `LOAD CSV`, `FOREACH`,
   `CREATE CONSTRAINT`) parses then fails at plan with the named "unsupported" error.
2. **Read semantics** — fixture graph (≥ 3 node kinds, ≥ 2 rel types, ≥ 1 cycle):
   variable-length path (`(a)-[*1..3]->(b)`), bounded (`[*2]`); `MERGE`-free `WHERE`
   combinations (IN / CONTAINS / CASE / IS NULL / function calls); `UNION` / `UNION ALL`;
   `UNWIND`; `WITH` + `RETURN DISTINCT`; full aggregation (`count`/`sum`/`avg`/`min`/
   `max`/`collect`) — assert exact row sets and column order.
3. **Write semantics** — `CREATE` node + rel; `CREATE` with prop map; `MERGE` (create
   new, merge existing, `ON CREATE` vs `ON MATCH`); `SET` (assign, `+=`, add label);
   `REMOVE` (prop, label); `DETACH DELETE` (removes incident edges, cascades); plain
   `DELETE` (refuses a node with edges); `DELETE` (removes only, leaves orphaned edges);
   multi-op write in a single query commits atomically, mid-query error rolls back
   (assert: prior `CREATE` gone, no partial state).
4. **Parameters** — `MATCH (a) WHERE a.x = $p RETURN a` with `$p` as int, string,
   bool, float, list, map, `null`; wrong-type assertion fails at *bind* (not exec).
5. **Caps** — `max_rows` set below result → `truncated:true` + exactly the cap count;
   `timeout_ms` set low on a large expand → timeout error; `max_varlen_depth` = 2 on
   a 4-hop path → bounded at 2.
6. **Read-only guard** — `cypher.readonly: true` (or env) → `CREATE`/`MERGE`/`SET` /
   `REMOVE`/`DELETE` return the named "readonly" error; reads still return rows.
7. **Transport** — `POST /graph/cypher` + `POST /graph/query` both return the same
   envelope; auth on `POST` when `SKILLGRID_HTTP_TOKEN` set; 400 on missing/malformed
   body; line/column syntax error shape on a bad query.
8. **Build gate** — `go build && go vet && go test ./...` (ANTLR runtime is pure-Go;
   generated sources committed; a `task build` still passes with no toolchain beyond
   Go + the optional `antlr4` generator at *generate* time).

## Risks / Trade-offs

- **ANTLR Go codegen size + generator toolchain** → generated sources are committed
  (deterministic); the `antlr4` generator is a build-time tool only (like a linter or
  bundler, not a runtime dep); the ANTLR *runtime* is pure-Go (no cgo, no Java at
  runtime). If the generated-code bloat becomes a real cost, **Approach B** (hand parser)
  is the documented fallback.
- **SQL/CTE translation of multi-hop patterns** → bounded depth + fan-out caps (per node
  in the pattern, global row cap) + a "no cross-file recursive CTE" rule (each `WITH`
  rebinds scope to its own query, so no unbounded recursion); if scale demands, **Approach
  C** (dedicated engine) is the documented escalation.
- **`MERGE` uniqueness** → "uniqueness" for `MERGE (a:L {k: v})` is (label + all props
  in the map) equality over `graph_nodes` (the graph's identity is the node *itself*,
  not a property constraint); no separate constraint table — matches our "no new schema"
  posture. Documented in the tool description so agents don't assume DB-level unique indexes.
- **Read-only default vs. write-on-by-default** → user chose read-write; we keep the
  opt-out `cypher.readonly` guard so agent modes can pin writes off without a separate
  tool — mirrors GitNexus's read-only MCP mode.
- **`DETACH DELETE` cascade ordering** → `graph_edges.source_id`/`target_id` are FK
  `ON DELETE CASCADE` (from `mnemonic-graph`), so `DELETE a` with `DETACH` removes the
  node and all incident edges in one statement; plain `DELETE a` must check incident
  edges first (400 if any exist).
- **`UNWIND` on an empty list** → zero rows (standard); tests cover it.

## Migration Plan

1. `mnemonic-cypher` **requires `mnemonic-graph`** to be implemented first (graph store +
   `graph_nodes`/`graph_edges` + the `cypher` transport plumbing) — this change upgrades
   the `cypher` in place.
2. **No schema migration** — this change adds no tables; it reads/writes the existing
   graph tables.
3. **Rollback** — drop the `cypher` module, restore `mnemonic-graph`'s mini-subset
   `cypher` handler (or remove `cypher` entirely); the graph tables are unchanged by
   this change. The `mnemonic-graph` non-goal "full Cypher" is retired the moment this
   change ships.
4. **Phasing suggestion** (each independently shippable):
   - **C1 — Parser + read IR + SQL read executor + `cypher` tool (upgrade)** — the
     first "full Cypher is live" moment; write clauses rejected with "not yet."
   - **C2 — Write IR + transactional executor (`CREATE`/`MERGE`/`SET`/`REMOVE`/`DELETE`)**
   - **C3 — `WITH` / `UNION` / `UNWIND` / full aggregation + `DISTINCT` / `ORDER BY` /
     `SKIP` / `LIMIT`** — the analytics surface
   - **C4 — Caps (rows/timeout/depth) + read-only guard + `POST /graph/query` alias +
     error-shape tests**
   Phases C1–C4 are additive; the `mnemonic-graph` mini-subset is the "before"
   state, this change's full engine is the "after."

## Open Questions

- **`CALL { … }` subqueries** (Cypher's scoping subqueries) — *out of scope* for now;
  the ANTLR grammar parses them, but our planner rejects with a named "unsupported"
  error. Revisit if agent workflows need inline subqueries rather than multi-`WITH`.
- **Cross-project Cypher** (query spanning two project `.sqlite` files) — *out of scope*;
  matches Mnemonic's single-project posture everywhere else.
- **`EXISTS {…}` full subquery** — *out of scope*; `EXISTS(prop)` / `EXISTS(rel)` on a
  bound node are supported; the `{query}` form is rejected with a named error.
- **Cypher-specific DDL** (`CREATE CONSTRAINT`, `CREATE INDEX`) — *out of scope*; graph
  identity is the node, and SQLite indexes are managed by the store, not Cypher.
- **Taint/PDG-specific Cypher helpers** (e.g., Cypher-native `taint(a→b)`) — *out of
  scope*; the high-level `explain` / `pdg_query` tools are the surface.

