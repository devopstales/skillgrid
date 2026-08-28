# Glossary Reference

| Term | Source Glossary | Context |
| --- | --- | --- |
| Full Cypher | `glossary/technical.md` | The scope this design implements (read + write) |
| Mini-Cypher | `glossary/technical.md` | The superseded subset (design decision 2 supersede rationale) |
| IR (intermediate representation) | `glossary/technical.md` | Core of the Approach A parser→IR→SQL architecture |
| Query caps | `glossary/technical.md` | Decision 4 execution caps (rows/timeout/depth) + readonly guard |
| Node | `glossary/technical.md` | The stored entity Cypher patterns match against |
| Edge | `glossary/technical.md` | The relationship Cypher patterns expand along |
| MERGE / DETACH DELETE semantics | `glossary/technical.md` | Write-transaction + graph identity (decision 4) |
| OCBI convention | `glossary/technical.md` | The `{columns, rows, truncated?}` raw-JSON envelope |
