Feature: ADR-aware knowledge index + retrieval

  Background:
    Given a mnemonic project store with the default indexing config
    And a repository containing `.skillgrid/artifacts/04-adr-0006-vector-search.md`
    And a repository containing `.skillgrid/artifacts/04-adr-0009-embedding-model.md`

  @step-01
  Scenario: ADR files are indexed after code_index
    When code_index runs against the repository
    Then the file `.skillgrid/artifacts/04-adr-0006-vector-search.md` is present in the code index
    And the file `.skillgrid/artifacts/04-adr-0009-embedding-model.md` is present in the code index

  @step-02
  Scenario: adr_search returns structured ADR results
    When adr_search is called with query "vector search"
    Then the result contains an ADR with path `.skillgrid/artifacts/04-adr-0006-vector-search.md`
    And the result includes fields status, supersedes, date, title, snippet, and score

  @step-03
  Scenario: adr_search surfaces the superseding ADR
    Given ADR 0009 frontmatter `supersedes: 0006`
    When adr_search is called with query "vector search"
    Then the result for ADR 0009 is marked superseded
    And the result includes the superseding ADR 0009

  @step-04
  Scenario: only DocInclude-matched .skillgrid files are indexed
    Given a repository containing `.skillgrid/artifacts/04-adr-0006-vector-search.md`
    And a repository containing `.skillgrid/state.yaml`
    When code_index runs against the repository
    Then the file `.skillgrid/artifacts/04-adr-0006-vector-search.md` is present in the code index
    And the file `.skillgrid/state.yaml` is not present in the code index

  @step-05
  Scenario: adr_search on an index with no ADR chunks returns empty
    Given a code index with no `04-adr-*.md` files
    When adr_search is called with query "vector search"
    Then the result is an empty list
    And no error is returned
