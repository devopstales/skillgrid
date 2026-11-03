# Research Findings

<!-- This file serves two purposes: the research narrative for humans AND the
     accumulated knowledge base for the agent. Read it at the start of every
     session, loop tick, or heartbeat to remember what has been learned. -->

## Research Question

<!-- One clear sentence. What are we trying to discover? -->

## Current Understanding

<!-- Updated after each outer-loop cycle. What do we know so far?
     What patterns explain our results? What is the mechanism?
     This section should read like the core argument of a paper. -->

## Key Results

<!-- Significant findings with metrics, comparisons, and brief interpretation.
     Link to experiment directories for full details. -->

## Patterns and Insights

<!-- What emerges across multiple experiments? What types of changes
     consistently work or fail? Why? -->

## Lessons and Constraints

<!-- Specific actionable learnings that should guide future experiments.
     Things tried that didn't work AND WHY, so they are not repeated.
     Constraints discovered about the problem space.

     Examples:
     - HNSW recall at 20K vectors collapses below 60% at ef=10; ef>=50 needed
     - sqlite-vec v0.1.9 is brute-force only; no ANN index yet
     - modernc.org/sqlite /vec subpackage first ships in v1.59.0
     - FTS5 porter tokenizer drops code identifiers; use unicode61 instead
     - Baseline only reproduces with batch_size=64, not 32 -->

## Open Questions

<!-- What remains unanswered? What would strengthen or challenge
     our current understanding? What would answer it? -->

## Optimization Trajectory

<!-- Summary of inner-loop progress. How has the metric evolved?
     Note inflection points and what caused them. -->
