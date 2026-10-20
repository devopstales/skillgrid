Feature: Second-Brain Learnings (growth, health, knowledge-class, embed gate, capture gate, fused recall, promotion, onboarding)
  As a Mnemonic operator
  I want growth timeline, actionable health, durability class, centrality-gated embeddings, confidence-gated capture, fused web recall, reflect-time promotion, and onboarding recall with bootstrap
  So that the second brain is observable, maintainable, and cheap, and project knowledge compounds across sessions

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
    Given migration 070_add_knowledge_class applied with column knowledge_class and index on knowledge_class and project
    When I save a type decision observation without knowledge_class
    Then it reads back knowledge_class convention
    And mem_search with knowledge_class convention returns only convention rows, each labeled
    And omitting knowledge_class on mem_save infers decision to convention and session_log to episodic and never fails
    And prior mem_search results are unchanged when no knowledge_class filter is passed
    #### Gates
    # G3: CHECK: cd mnemonic && go test ./internal/memory/... -run TestSave_KnowledgeClassInfer -v
    # G3: EXPECT: PASS, additive nullable column, heuristic backfill re-runnable, zero behavior change unfiltered
    # G4: CHECK: migration file mnemonic/internal/store/migrations/070_add_knowledge_class.sql exists with index
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
    And a strong correction ("don't do that") is suggested while weak alone ("no") is not and weak with directive ("no, use pnpm") is
    And a negative phrase ("no worries") suppresses the suggestion
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

  @step-07 @fused-recall
  Scenario: fused recall labeled
    Given one observation and one unexpired web_cache row and one expired web_cache row sharing a term
    When I call mem_search for that term
    Then the observation hit is present
    And the unexpired web_cache hit is present labeled corpus web_cache with source, url, fetched_at, and expires_at
    And the retrieve pointer resolves via web_cache_get
    And the expired web_cache row is absent
    And observations-only behavior is otherwise unchanged
    #### Gates
    # G8: CHECK: cd mnemonic && go test ./internal/memory/... -run TestMemSearch_FusedWebCache -v
    # G8: EXPECT: PASS, web hit labeled, expired excluded, vector leg observations-only

  @step-08 @promotion-candidates
  Scenario: promotion candidates proposed
    Given the reflect skill with the promotion step
    When a change closes with web_cache entries from its sessions
    Then report.md carries a Web-cache promotion candidates section with one card per entry
    And every card cites its cache id and url with a proposed concept home
    And no concept file is written by the promotion itself
    #### Gates
    # G9: CHECK: grep -c 'promotion candidates' .agents/skills/lifecycle/reflect/SKILL.md
    # G9: EXPECT: count at least 2 (step plus verification or template pointer)

  @step-09 @onboarding-recall-bootstrap
  Scenario: onboarding recall and bootstrap
    Given the onboarding skill with recall-before-detect and bootstrap steps
    When onboarding re-runs on a project with recorded knowledge
    Then index.md, the log tail, and memory are consulted before re-detecting facts
    And brownfield enroll yields draft candidates with anchors for one-at-a-time confirmation
    And unconfirmed drafts are dropped and fresh projects skip recall with a note
    #### Gates
    # G10: CHECK: grep -c -e 'recall-before-detect' -e 'Step 1.5b' .agents/skills/lifecycle/onboarding/SKILL.md .agents/skills/lifecycle/onboarding/references/process.md
    # G10: EXPECT: count at least 3 across both files (rule plus procedure plus verification)

  @step-10 @knowledge-graph-view
  Scenario: knowledge concept graph view
    Given a fixture bundle with 4 concepts, one orphan, one broken link, one deprecated page, plus index.md and log.md
    When I derive the knowledge graph
    Then 4 nodes carry their frontmatter fields with edges only for resolving links
    And the orphan is flagged and the broken link is tolerated without error
    And index.md and log.md are excluded from nodes
    And an empty bundle yields an empty state, not an error
    #### Gates
    # G11: CHECK: cd mnemonic && go test ./internal/knowledge/... -v
    # G11: EXPECT: PASS, derive per call over current Markdown, no store, read-only route

  @step-11 @explicit-search-scope
  Scenario: explicit search scope
    Given two non-empty project buckets sharing a term
    When I call mem_search with all_projects true
    Then the response carries a scope block naming both buckets with counts ahead of merged ranks
    And a default scoped search is byte-identical in shape to today with no scope block
    #### Gates
    # G12: CHECK: cd mnemonic && go test ./internal/mcp/... -run TestMemSearch_ScopeBlock -v
    # G12: EXPECT: PASS, additive scope block, existing federated tests green unmodified

  @step-12 @okf-lint
  Scenario: knowledge lint script verdicts
    Given a clean fixture bundle and a drifted fixture bundle (broken link, missing index entry, broken claim, stable page missing generated)
    When I run okf-lint.mjs over each fixture
    Then the clean bundle exits 0 with DRIFT clean and SCOPE COMPLETE
    And the drifted bundle exits 1 naming the broken link, the index gap, the broken claim, and the missing provenance
    And neither run writes to the fixture directories
    #### Gates
    # G13: CHECK: node .agents/skills/verification/qa/scripts/okf-lint.mjs test/fixtures/okf-clean && node .agents/skills/verification/qa/scripts/okf-lint.mjs test/fixtures/okf-drift; test $? -eq 1
    # G13: EXPECT: clean exits 0 with DRIFT clean, drifted exits 1 with all four finding classes, read-only both

  @step-13 @teaching-graph-page
  Scenario: teaching graph page
    Given a fixture repo with modules, calls, imports, an orphan, a broken concept link, and a deprecated page
    When I browse tree to canvas and canvas to tree
    Then directory selection filters the canvas and node selection reveals its file
    And the inspector shows identity with lines, connections grouped by edge kind with source lines, and a code preview
    And search centers a named symbol and the tour completes at least 3 stops
    And locals expand ephemerally with an empty store diff
    And orphan, broken-link, deprecated, empty, and degraded states all render
    #### Gates
    # G14: CHECK: cd mnemonic && go test ./internal/... -run 'TestGraphEnrich_Lines|TestLocalsResolver_Ephemeral' -v
    # G14: EXPECT: PASS, lines served, locals ephemeral with store untouched, no new endpoint

  @step-14 @contextualized-retrieval
  Scenario: contextualized retrieval
    Given a fixture with colliding code names across files
    When I index with deterministic path-symbol-scope prefixes
    Then embedded text carries the prefix while display text stays raw
    And prefixed retrieval beats raw on the fixture's code-name queries
    And the MMR-vs-RRF ablation numbers are recorded with the ranker unchanged
    #### Gates
    # G15: CHECK: cd mnemonic && go test ./internal/codeindex/... -run TestChunkPrefix_Context -v
    # G15: EXPECT: PASS, prefix present, raw display unchanged, ablation recorded either way

  @step-15 @knowledge-moves
  Scenario: knowledge moves land intact
    Given the post-R6 layout with dissolved artifacts and xarchive
    When I run okf-lint over the repo
    Then it exits 0 with DRIFT clean and SCOPE COMPLETE
    And every LOCKED row resolves and git log follow shows history through both moves
    And code_search for a relocated ADR title returns the decisions path
    #### Gates
    # G16: CHECK: node .agents/skills/verification/qa/scripts/okf-lint.mjs .
    # G16: EXPECT: exit 0, DRIFT clean, SCOPE COMPLETE, artifacts and archive gone, history intact
