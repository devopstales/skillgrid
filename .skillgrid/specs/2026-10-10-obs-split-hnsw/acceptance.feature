Feature: Mnemonic obs-split + embedded ANN index

  # Delta spec for 2026-10-10-obs-split-hnsw (briefing.md R1-R5).
  # BDD is always on: every ticket carries SATISFIES -> one scenario here.

  Scenario: obs-parity-after-split
    Given a pre-split project store copy
    When migration 054 rebuilds observations into core + side tables behind a compat view
    Then row-hash parity holds for all rows
    And the 7 bitemporal read sites still filter superseded rows via invalid_at

  Scenario: hot-path-avoids-blobs
    Given the split store
    When mem_context runs its hot-path query
    Then EXPLAIN QUERY PLAN shows no scan of observation_vectors
    And returned rows equal the compat-view rows minus vector columns

  Scenario: migration-rollback
    Given a migrated store with a hash mismatch
    When rollback runs from the pre-migration .dump
    Then the legacy table restores
    And the service opens read-only with no data loss

  Scenario: ivf-recall
    Given 20k synthetic 768-dim vectors
    When IVF probe search runs top-K
    Then recall@10 vs brute force is >= 0.90
    And a corrupt IVF table degrades to brute force without error

  Scenario: hnsw-recall-latency
    Given 100k synthetic 768-dim vectors
    When HNSW search runs top-K
    Then recall@10 vs brute force is >= 0.95
    And cold p95 < 50ms and warm < 10ms

  Scenario: per-model-dim-namespaces
    Given vectors indexed under two models (768-dim and 1024-dim)
    When each model queries its own namespace
    Then both return correct hits
    And a cross-dim query returns an explicit typed error
