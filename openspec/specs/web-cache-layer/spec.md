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

### Requirement: Per-source TTL defaults

Different research sources SHALL expire on different schedules; the defaults
SHALL be overridable via `config.d/indexing.yaml` under `web_ttl_days`.

| source   | default TTL | override key |
|----------|-------------|--------------|
| `context7` | 30 days | `context7` |
| `exa`      | 7 days  | `exa` |
| `deepwiki` | 14 days | `deepwiki` |
| `fetch`    | 7 days  | `fetch` |
| `manual`   | never   | `manual` (0) |

#### Scenario: context7 snapshot expires after 30 days
- **GIVEN** a `context7` entry was fetched 31 days ago
- **WHEN** `web_cache_lookup` is called for it
- **THEN** the response carries `stale: true` (or a hint prompting re-fetch)

#### Scenario: Manual entry never expires by definition
- **GIVEN** a `manual` entry was saved 100 days ago
- **WHEN** `web_cache_lookup` is called for it
- **THEN** it is still reported fresh (no staleness flag)

#### Scenario: User override wins
- **GIVEN** `config.d/indexing.yaml` sets `web_ttl_days: { context7: 1 }`
- **WHEN** a `context7` entry 2 days old is looked up
- **THEN** it is reported stale (the repo override beats the built-in 30d default)
