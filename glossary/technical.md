# Technical Glossary

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| **Mnemonic** | skillgrid's local-first memory engine: memory + code index + web cache + graph in one SQLite store and single binary | Referring to the `internal/mnemonic` subsystem, `skillgrid mcp`/`serve`/`index` | "MemIndex" (legacy name), "Engram" (the protocol it is compatible with) |
| **Observation** | A curated memory record (What/Why/Where/Learned) stored in the `observations` table | Any memory content stored or searched by agents | "Note", "record" (unqualified) |
| **Node** | A graph entity (kind: session, observation, file, symbol, web_entry) stored in `graph_nodes` | Graph schema, tools, and visualizer | "Vertex", "entity" (in graph context) |
| **Edge** | A typed, directed relationship between two graph nodes stored in `graph_edges` | Graph schema, tools, and visualizer | "Link", "relation" (use MENTIONS/CITES/RELATED_TO edge types instead of generic "relation") |
| **Edge type vocabulary** | The fixed set of allowed edge types: CONTAINS, DEFINES, CALLS, IMPORTS, EXTENDS, MENTIONS, CITES, RELATED_TO, FOLLOWS_FROM | Writing or validating edges, UI legend | Inventing new edge types without updating the vocabulary in one place |
| **Auto-synthesis** | Deterministic regeneration of edges derived from existing tables (CONTAINS/MENTIONS/CITES/FOLLOWS_FROM) | Graph rebuild behavior, "rebuild" discussions | "Event sourcing" (it is not) |
| **Manual edge** | A graph edge created by an agent via `graph_slink`/`POST /graph/link` with `source=manual`, surviving auto-synthesis rebuilds | Edge lifecycle, `graph_slink` docs | "Pinned edge" |
| **Symbol** | A derived graph-owned node for a function, method, or type extracted from a Go source file | Symbol pipeline, `graph_context` on symbols | "Function" (unqualified — may also mean a node of kind symbol) |
| **Symbol pipeline** | The opt-in tree-sitter extraction pass inside `codeindex` that produces symbol nodes and DEFINES/CALLS/IMPORTS/EXTENDS edges | Phase 2 tasks, config `symbols.enabled` | "GitNexus indexer" |
| **Blast radius** | The set of nodes reachable from one node along edge directions (GitNexus "impact"), returned with hop distances | `graph_impact` tool, impact discussions | "Downstream impact" (use direction parameter instead) |
| **Trace** | Shortest directed path between two nodes over an allowed edge-type set (GitNexus "trace") | `graph_trace` tool | "Path" (use for the result, trace for the operation) |
| **Subgraph extraction** | One round-trip fetch of `{nodes, edges}` JSON for the visualizer, anchored optionally at a node | `/graph/subgraph`, Sigma rendering | "Graph snapshot", "export" |
| **OCBI convention** | MCP tool output is raw JSON only, no leading prose (validated by `ValidateRawJSON`) | Writing MCP handlers and tool output tests | "JSON responses" (unqualified) |
| **Truncated traversal** | Traversal result flag set when depth or node caps cut the walk short | Traversal error semantics, cap tuning | "Timeout" |
