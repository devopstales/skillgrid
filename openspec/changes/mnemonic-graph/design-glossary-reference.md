# Glossary Reference

| Term | Source Glossary | Context |
| --- | --- | --- |
| Mnemonic | `glossary/technical.md` | System the graph layer extends |
| Observation | `glossary/technical.md` | Node kind and MENTIONS/CITES source |
| Node | `glossary/technical.md` | Graph entity model (incl. basic_block, route, tool, shape kinds) |
| Edge | `glossary/technical.md` | Graph relationship model, pass-owned |
| Pass | `glossary/technical.md` | Extraction/unit that owns its edges (synthesis, symbols, pdg, contracts, manual) |
| Edge type vocabulary | `glossary/technical.md` | Accepted edge types across tools, UI, and storage (core/verdicts/pdg/contracts groups) |
| Mini-Cypher | `glossary/technical.md` | The limited Cypher subset accepted by the `cypher` tool |
| Auto-synthesis | `glossary/technical.md` | Deterministic edge derivation from existing tables |
| Manual edge | `glossary/technical.md` | Agent-authored links that survive rebuilds |
| Verdict | `glossary/technical.md` | `graph_judge` relationship judgment stored as a typed edge |
| Symbol | `glossary/technical.md` | Derived Go/TS/Python function/method/type node |
| Symbol pipeline | `glossary/technical.md` | Phase 2 multi-language symbol extraction within codeindex |
| PDG | `glossary/technical.md` | Per-function basic-block + CFG/CDG/reaching-def dataflow |
| Taint path | `glossary/technical.md` | Precomputed source→sink flow (`explain`) |
| API contract | `glossary/technical.md` | Route/tool/shape subgraph read by `route_map`/`tool_map`/`shape_check`/`api_impact` |
| Rename planner | `glossary/technical.md` | Read-only affected-locations report (`rename`) |
| Blast radius | `glossary/technical.md` | `graph_impact` operation semantics |
| Trace | `glossary/technical.md` | `graph_trace` shortest-path operation |
| Subgraph extraction | `glossary/technical.md` | Single-payload fetch for the visualizer |
| OCBI convention | `glossary/technical.md` | Raw-JSON-only MCP tool output |
| Truncated traversal | `glossary/technical.md` | Cap-exceeded traversal result flag |
