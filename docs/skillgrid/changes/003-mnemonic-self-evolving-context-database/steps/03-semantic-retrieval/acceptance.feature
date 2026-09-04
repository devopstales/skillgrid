# 03-semantic-retrieval acceptance
# Source: intent.md + plan.md (docs/skillgrid/changes/003-mnemonic-self-evolving-context-database/)
# Threat: Mnemonic tool surface — L1-default search; L2 only via load_full_details

Feature: Overview-first semantic retrieval
  As an agent
  I want ranked overviews by default
  So that I load full detail only when needed

  @happy @p0 @security
  Scenario: Semantic search returns ranked overviews only
    Given tiered content with abstracts and overviews
    When an agent runs semantic search
    Then results are ranked overviews with abstracts
    And full-detail bodies are absent

  @happy @p0 @security
  Scenario: Explicit load returns full markdown
    Given a path from a semantic search result
    When the agent loads full details for that path
    Then the full markdown is returned

  @edge
  Scenario: Embeddings off falls back with trail
    Given embeddings are disabled or empty
    When an agent runs semantic search
    Then ranking uses title or abstract fallback
    And a retrieval trail is recorded

  @failure @p1
  Scenario: Unknown path rejects full-detail load
    Given an unregistered path
    When the agent loads full details for it
    Then a clear not-found error is returned
