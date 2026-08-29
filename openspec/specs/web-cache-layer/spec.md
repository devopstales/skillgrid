# web-cache-layer Specification

## Purpose
TBD - created by archiving change mnemonic. Update Purpose after archive.

## Requirements

### Requirement: Web cache save and lookup

The system SHALL provide `web_cache_save` and `web_cache_lookup` for caching remote research results.

#### Scenario: Save after Context7 call
- **GIVEN** an agent calls Context7 and gets a result
- **WHEN** `web_cache_save` is called with source, library_id, query, content
- **THEN** the snapshot is stored with a content hash and TTL

#### Scenario: Lookup before remote call
- **GIVEN** a cached entry exists for a query
- **WHEN** `web_cache_lookup` is called with the same query
- **THEN** a hit is returned with the cached content and freshness status

#### Scenario: Stale detection
- **GIVEN** a cached entry past its TTL
- **WHEN** `web_cache_lookup` is called
- **THEN** `stale: true` is returned so the agent re-fetches

#### Scenario: Dedup
- **GIVEN** an entry with the same `(project, source, cache_key)` exists
- **WHEN** `web_cache_save` is called again
- **THEN** the existing entry is updated if content_hash differs

#### Scenario: Entry size cap
- **GIVEN** content exceeds 256KB
- **WHEN** `web_cache_save` is called
- **THEN** a structured error is returned (agent should summarize first)

### Requirement: Web cache search

The system SHALL provide `web_cache_search` for FTS5 search over cached research.

#### Scenario: Search cached research
- **GIVEN** cached web snapshots exist
- **WHEN** `web_cache_search` is called with a query
- **THEN** matching cached entries are returned ranked by relevance

#### Scenario: Fresh-only filter
- **GIVEN** `web_cache_search` is called with `fresh_only: true`
- **WHEN** results are returned
- **THEN** expired entries are excluded
