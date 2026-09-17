# Research Log

Chronological, append-only record of research decisions and actions.

| # | Date | Type | Summary |
|---|------|------|---------|
|  |      |      |         |

<!-- Entry types:
  bootstrap    — initial scoping, literature search, hypothesis formation
  inner-loop   — experiment run and result
  outer-loop   — synthesis, reflection, direction decision
  pivot        — change of research direction
  report       — progress presentation generated
  conclude     — decision to finalize

Example entries:
| 1  | 2026-09-17 | bootstrap  | Searched Exa + arXiv + Context7 for SQLite vector search. Found 12 relevant sources. Gap: no systematic recall/latency comparison of sqlite-vec vs vectorlite at 20K vectors. Formed 3 hypotheses. Baseline: in-memory cosine over 18.7K chunk vectors, 28ms/query. |
| 2  | 2026-09-17 | inner-loop | H1 run_001: inserted 3000 768-dim vectors into vec0 table. Measured KNN latency: 1.2ms/query (brute force within SQLite). Recall: 100% (exact). Confirms sqlite-vec v0.1.9 is brute-force, not ANN. |
| 3  | 2026-09-17 | inner-loop | H1 run_002: same 3000 vectors, HNSW via vectorlite (ef=50). Latency: 0.35ms. Recall: 86.5%. ANN speedup real, recall cost measurable. |
| 4  | 2026-09-17 | outer-loop | Reviewed 2 runs. Pattern: at 3K vectors, ANN's latency advantage (3.4x) is offset by recall loss (13.5pp). Crossover likely at 50K+. Direction: DEEPEN — scale to 20K and 100K to find the actual crossover. |
| 5  | 2026-09-18 | report     | Generated progress-001.html with recall-vs-scale chart. | -->
