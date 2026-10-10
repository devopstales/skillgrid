Feature: Second-Brain Learnings (growth, health, knowledge-class, embed gate, capture gate)
  As a Mnemonic operator
  I want growth timeline, actionable health, durability class, centrality-gated embeddings, and confidence-gated capture
  So that the second brain is observable, maintainable, and cheap

  Background:
    Given the mnemonic Go module at mnemonic/
    And a clean per-test SQLite store with live observations only

  @step-01 @growth-timeline
  Scenario: happy path growth timeline cumulative
    Given synthetic observations across Jan, Feb, Mar periods
    And one soft-deleted March row
    When I call GrowthTimeline with granularity month
    Then period Jan returns added a1 and total a1
    And period Feb returns added a2 and total a1+a2
    And period Mar returns added a3 excluding the soft-deleted row with total a1+a2+a3
    And an empty store returns an empty list, not null and not an error
    #### Gates
    # G1: CHECK: cd mnemonic && go test ./internal/secondbrain/... -run TestGrowthTimeline_Monthly -v
    # G1: EXPECT: PASS, cumulative totals verified, soft-deleted excluded, empty store returns []

  @step-02 @health-recommendations
  Scenario: happy path health recommendations actionable
    Given a fixture with high duplicate density
    And a fixture with 0 embedding coverage and an active embedder
    And a fixture with more than half of rows older than 90 days and total greater than 10
    When I compute healthRecommendations directly as a pure function
    Then a HIGH dedup recommendation is present for the dense fixture
    And an embeddings MEDIUM recommendation is present when the embedder is on and absent when off
    And a stale MEDIUM recommendation is present for the old fixture
    And all rules off yields an empty Recommendations list
    #### Gates
    # G2: CHECK: cd mnemonic && go test ./internal/secondbrain/... -run TestHealthRecommendations -v
    # G2: EXPECT: PASS, 5 rules (dedup/embeddings/stale/cleanup/review) ordered and stable, no new DB queries

  @step-03 @knowledge-class
  Scenario: happy path knowledge class infer and search
    Given migration 051_add_knowledge_class applied with column knowledge_class and index on project and knowledge_class
    When I save a type decision observation without knowledge_class
    Then it reads back knowledge_class convention
    And mem_search with knowledge_class convention returns only convention rows, each labeled
    And omitting knowledge_class on mem_save infers decision to convention and session_log to episodic and never fails
    And prior mem_search results are unchanged when no knowledge_class filter is passed
    #### Gates
    # G3: CHECK: cd mnemonic && go test ./internal/memory/... -run TestSave_KnowledgeClassInfer -v
    # G3: EXPECT: PASS, additive nullable column, heuristic backfill re-runnable, zero behavior change unfiltered
    # G4: CHECK: migration file mnemonic/internal/store/migrations/051_add_knowledge_class.sql exists with index
    # G4: EXPECT: file present, applies once via migration tracker, no down-migration required

  @step-04 @pagerank-embed
  Scenario: happy path pagerank embed gate
    Given a code fixture with N symbols plus doc-bearing chunks and recently-changed files
    When I index with embed-top 50
    Then at most ceil(N/2) centrality-selected code symbols are embedded
    And all doc-bearing chunks and recently-changed files are always embedded
    And index output reports embedded and skipped counts
    And gate 100 reproduces the identical vector set as ungated indexing
    And a skipped symbol has a NULL vector and degrades, not errors
    #### Gates
    # G5: CHECK: cd mnemonic && go test ./internal/codeindex/... -run TestIndex_EmbedTop -v
    # G5: EXPECT: PASS, carve-out honored, counts reported, gate-100 identical, lexical retrieval unaffected

  @step-05 @confidence-capture
  Scenario: happy path confidence gated capture
    When I call mem_capture_suggest with "We're switching from Bootstrap to Tailwind"
    Then trigger switching is returned with confidence at least 0.75 and suggested true with a draft
    And no observation row is created by the suggest call
    When I call mem_capture_suggest with "hey what's up"
    Then suggested false is returned and no row is created
    And after the agent confirms with mem_save exactly one observation exists with the suggested knowledge_class
    And repeated suggest calls return the same result and never write
    #### Gates
    # G6: CHECK: cd mnemonic && go test ./internal/secondbrain/... -run TestCaptureSuggest -v
    # G6: EXPECT: PASS, suggest never persists, threshold default 0.75, deterministic no-LLM classifier

  @step-06 @regression
  Scenario: full module regression green
    Given all five features implemented behind advisory fail-open legs
    When I run the full mnemonic test suite
    Then no prior mem_search, code_index, or lifecycle behavior regresses
    #### Gates
    # G7: CHECK: cd mnemonic && go test ./internal/... -v
    # G7: EXPECT: PASS, whole module green
