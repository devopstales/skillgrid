# Glossary Reference

| Term | Source Glossary | Context |
| --- | --- | --- |
| Mnemonic | `glossary/technical.md` | The memory engine being extended with a graph |
| Observation | `glossary/technical.md` | Node kind linked by MENTIONS/CITES edges |
| Node | `glossary/technical.md` | Graph entity in separate tables (incl. basic_block, route, tool, shape) |
| Edge | `glossary/technical.md` | Typed relationship between nodes, pass-owned |
| Pass | `glossary/technical.md` | Extraction/unit owning its edges (synthesis, symbols, pdg, contracts, manual) |
| Edge type vocabulary | `glossary/technical.md` | Fixed edge types the proposal enforces (core/verdicts/pdg/contracts groups) |
| Mini-Cypher | `glossary/technical.md` | The limited Cypher subset the `cypher` tool accepts |
| Auto-synthesis | `glossary/technical.md` | Deterministic edges derived from existing rows |
| Verdict | `glossary/technical.md` | `graph_judge` relationship between observations stored as an edge |
| Symbol | `glossary/technical.md` | Derived Go/TS/Python function/method/type node |
| Symbol pipeline | `glossary/technical.md` | Phase 2 multi-language extraction pass |
| PDG | `glossary/technical.md` | Per-function dataflow graph (phase 3) |
| Taint path | `glossary/technical.md` | Precomputed source→sink flow for `explain` |
| API contract | `glossary/technical.md` | Route/tool/shape subgraph for contract tools (phase 4) |
| Rename planner | `glossary/technical.md` | Read-only rename impact report |
