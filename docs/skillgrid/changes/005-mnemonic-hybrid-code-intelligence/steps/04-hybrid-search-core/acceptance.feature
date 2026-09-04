# 04-hybrid-search-core acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/)
# Threat: Mnemonic tool surface — code_hybrid_search ≠ memory semantic_search; bad args rejected

Feature: Offline hybrid code search with pluggable embeddings
  As a coding agent
  I want ranked hybrid hits with provenance
  So that search works without embeddings online

  @happy @p0
  Scenario: Hybrid search ranks offline with provenance
    Given an indexed project with embeddings disabled
    When an agent runs code_hybrid_search and checks semantic and embedding status
    Then ranked hits include per-signal FTS and signal provenance
    And code_semantic_search and code_embedding_status are available

  @edge
  Scenario: Down embedder degrades to FTS and signals
    Given hybrid search available and embedder down
    When an agent runs code_hybrid_search
    Then ranking uses FTS and signals without hard-failing for missing embeddings

  @failure @p1 @security
  Scenario: Hybrid tool is distinct and rejects bad args
    Given the Mnemonic tool surface is registered
    When an agent compares code_hybrid_search to semantic_search and sends bad args
    Then code_hybrid_search is distinct, code_search is unchanged, and bad args are rejected
