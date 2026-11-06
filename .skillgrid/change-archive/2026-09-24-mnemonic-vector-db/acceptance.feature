# Acceptance criteria — 2026-09-24-mnemonic-vector-db
#
# BDD is always on. Each scenario below is the SATISFIES target for the
# blueprint task that names it. G<n> gates are the runnable checks the
# execution skill runs before declaring the task green.

Feature: Mnemonic vector DB (durable in-SQL vector path, option G)

  Background:
    Given the store opens with the modernc v1.59.0 driver
    And the BLOB embedding tables (embeddings, chunk_embeddings) are the source of truth

  # ---- Task 1: modernc bump validation (door check / prerequisite) ----

  Scenario: modernc-bump-full-suite-green
    Given the modernc driver is bumped from v1.45.0 to v1.59.0
    When the full test suite is run (go test ./...)
    Then the suite exits 0
    And the 39 squashed migrations still apply
    And isWALBusy still classifies a concurrent-writer busy error
    And store.Open still returns a pooled handle for repeated opens of one project

  #### Gates
  G1:
    CHECK: go build ./... && go test ./...
    EXPECT: exit 0, all tests pass

  # ---- Task 2: vec0 virtual tables (migration 042) ----

  Scenario: vec0-tables-exist
    Given a freshly opened store (migration 042 has run)
    When I query sqlite_master for vec_symbols and vec_chunks
    Then both virtual tables are present

  Scenario: vec0-migration-idempotent
    Given a store opened once (migration 042 applied)
    When the same store is opened a second time
    Then no error is returned
    And each vec0 table appears exactly once in sqlite_master

  #### Gates
  G2:
    CHECK: go test ./internal/mnemonic/store/ -run 'TestVec0' -v
    EXPECT: 2 tests pass (TestVec0TablesCreatedAfterOpen, TestVec0MigrationIdempotent)

  # ---- Task 3: vectorstore package (search + upsert + delete) ----

  Scenario: vectorstore-top-k
    Given vec_symbols holds 3 orthogonal 768-d basis vectors at symbol_id 1, 2, 3
    When SearchSymbols is called with the basis-1 query vector and limit 1
    Then it returns symbol_id 2 (the matching basis vector)
    And with limit 3 the matching basis id is ranked first

  Scenario: vectorstore-mirrors-blob
    Given vec_symbols holds 3 rows
    When DeleteSymbols is called in a transaction and committed
    Then Count(vec_symbols) is 0

  Scenario: maxopenconns-one
    Given the store uses SetMaxOpenConns(1) (the WAL single-writer invariant)
    When a vector is upserted and SearchSymbols is called
    Then no deadlock occurs and the top-1 id is returned

  #### Gates
  G3:
    CHECK: go test ./internal/mnemonic/vectorstore/ -v
    EXPECT: all vectorstore tests pass (TableExists, UpsertAndSearch, Delete, MaxOpenConnsOne)

  # ---- Task 4: indexer dual-write ----

  Scenario: indexer-dual-write
    Given a fresh store with the embedder active and seeded symbols + chunks
    When embedPass runs and commits
    Then vec_symbols row count equals embeddings row count
    And vec_chunks row count equals chunk_embeddings row count
    And for each id the vec vector is byte-identical to the BLOB vector

  #### Gates
  G4:
    CHECK: go test ./internal/mnemonic/codeindex/ -run 'TestEmbedPassDualWritesVecTables' -v
    EXPECT: the dual-write row-count mirror test passes

  # ---- Task 5: exactness check (brute-force ground truth) ----

  Scenario: exactness-check-agreement
    Given the vec table and the BLOB table hold the same 3 orthogonal vectors
    When ExactnessCheck runs for a basis query with limit 3
    Then it reports Agree = true (no disagreements)

  Scenario: exactness-check-disagreement
    Given the vec table holds basis-0 at id 1 but the BLOB table holds basis-1 at id 1
    When ExactnessCheck runs for a basis-1 query with limit 1
    Then it reports Agree = false
    And it reports at least one disagreement (the drifted id)

  #### Gates
  G5:
    CHECK: go test ./internal/mnemonic/vectorstore/ -run 'TestExactnessCheck' -v
    EXPECT: 2 tests pass (Agreement, Disagreement)

  # ---- Task 6: durable path behind the flag ----

  Scenario: flag-off-unchanged
    Given MNEMONIC_VECTOR_DB is unset (the default)
    When the semantic leg runs a query
    Then it uses the in-memory cosine cache exactly as before
    And no vec table is read

  Scenario: durable-path-equivalence
    Given MNEMONIC_VECTOR_DB=1 and the vec table is populated with the same vectors as the BLOB table
    When the durable semantic leg runs a query
    Then its top-K ids equal the in-memory leg's top-K ids for the same query

  Scenario: empty-table-degrade
    Given MNEMONIC_VECTOR_DB=1 but the vec table is empty
    When the durable semantic leg runs a query
    Then it degrades to the in-memory path (no error, no panic)

  #### Gates
  G6:
    CHECK: go test ./internal/mnemonic/hybrid/ -run 'TestDurable' -v
    EXPECT: 3 tests pass (FlagRouting, PathEquivalence, EmptyTableDegrade)
