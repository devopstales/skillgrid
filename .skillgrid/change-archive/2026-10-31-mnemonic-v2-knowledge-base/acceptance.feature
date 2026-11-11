# Mnemonic v2 — Knowledge Base (ProjectWiki) Function
#
# Backfilled 2026-11-11 at the QA gate (W004): this change predates the
# acceptance.feature layout; its oracle was blueprint.md Must-Haves
# (T1–T8) + specs/000–013. The scenarios below mirror those truths 1:1
# and name the Go test that covers each (traceability-matched).
# Source: blueprint.md (T1–T8, Artifacts A–C), specs/005-query §6,
# specs/007-lint §2, specs/013-wiki-autolearning §2/§9.

## Requirements

### Requirement: vault bootstrap and absence

The system SHALL scaffold the ProjectWiki vault idempotently, and a
project with no vault/index SHALL report cleanly (empty, thin) rather
than error.

#### Scenario: bootstrap-scaffolds-contextual-types

```gherkin
      Given a project with no .skillgrid/projectwiki/config.json
      When projectwiki_bootstrap runs twice
      Then the vault scaffold exists (config.json, page-type folders, templates/)
      And the second run is a no-op (sentinel listed as skipped)
      And a stage-1 query on the empty index returns thin with fallback mem_lookup
```

#### Scenario: assess-query-thin-on-empty-index

```gherkin
      Given an empty ProjectWiki index
      When a stage-1 query returns 0 hits
      Then the result is thin:true with fallback mem_lookup
      And degraded is empty (nothing indexed, nothing stale)
```

### Requirement: generate no-op proof

A clean projectwiki_generate run SHALL report changed: 0 and write
nothing (PW-INV-4), proven at each of the four proof layers; a dirty
run SHALL create pages through the degraded floor, idempotently.

#### Scenario: generate-proven-noop

```gherkin
      Given an indexed vault with no new facts
      When projectwiki_generate runs
      Then changed is 0 and no page file is written
```

#### Scenario: generate-layer1-blocks-noop

```gherkin
      Given a generate run stopped at proof layer 1
      When the run completes
      Then nothing is written
```

#### Scenario: generate-creates-new-page-via-floor

```gherkin
      Given a new memory fact and no LLM synthesizer
      When projectwiki_generate runs
      Then a page is created via the degraded floor with degraded:true
      And a second run writes nothing further
```

### Requirement: derived index

projectwiki_index SHALL re-tokenize only changed pages;
projectwiki_reindex SHALL rebuild the full derived state (aliases,
counts, claims, in_degree — identical to the incremental path);
removed pages SHALL be soft-deleted.

#### Scenario: index-incremental-only-changed

```gherkin
      Given an indexed vault where one page file changes
      When projectwiki_index runs
      Then only the changed page is re-indexed
```

#### Scenario: reindex-rebuilds-projectwiki-pages

```gherkin
      Given a vault of pages with aliases and wikilinks
      When projectwiki_reindex runs
      Then every page row carries parsed aliases/counts and a recomputed in_degree
```

#### Scenario: index-soft-deletes-removed-pages

```gherkin
      Given an indexed page whose file is removed from the vault
      When projectwiki_index runs
      Then the page is soft-deleted and excluded from query/lint/status
```

### Requirement: wiki-first query with explicit fallback

projectwiki_query SHALL return ranked cited stage-1 hits; a thin
result (< ThinThreshold) SHALL carry fallback mem_lookup; a lagging
index SHALL carry degraded stale_index (spec 005 §6).

#### Scenario: query-returns-ranked-projectwiki-pages

```gherkin
      Given an indexed vault
      When projectwiki_query searches a covered topic
      Then ranked pages with scores are returned
```

#### Scenario: query-no-match-returns-empty

```gherkin
      Given an indexed vault
      When projectwiki_query searches an uncovered topic
      Then the result is empty, not an error
```

#### Scenario: assess-query-degraded-when-index-lags-vault

```gherkin
      Given an indexed page whose vault file changes without re-indexing
      When a stage-1 query is assessed
      Then degraded is stale_index
```

### Requirement: validate pin-check

projectwiki_validate SHALL pass fresh grounded pages and fail stale,
unresolved, or unattested pins/claims.

#### Scenario: validate-fresh-page-passes

```gherkin
      Given a grounded page whose anchors hold in the working tree
      When projectwiki_validate runs
      Then the page is ok (fresh)
```

#### Scenario: validate-stale-claim-fails

```gherkin
      Given a grounded page whose pinned literal no longer holds
      When projectwiki_validate runs
      Then the page is drifted
```

#### Scenario: validate-unresolved-pin-fails

```gherkin
      Given a grounded page whose anchor resolves to nothing
      When projectwiki_validate runs
      Then the page is drifted (unresolved)
```

### Requirement: deterministic lint

projectwiki_lint SHALL report the spec 007 §2 checks with severities
(orphans, broken wikilinks, duplicate aliases, missing sources,
ungrounded grounded pages, weak pins, past stale_after, weak
cross-references, contradictions, broken citations, coverage gaps);
--fix SHALL repair the safe subset; --strict SHALL block only on
error-severity findings.

#### Scenario: lint-structural-checks

```gherkin
      Given pages with a broken wikilink, a shared alias, no anchors,
        a past stale_after, a contradiction, and a dead code: citation
      When projectwiki_lint runs
      Then each defect is reported with its spec 007 severity
      And the cited Go package is not flagged as a coverage gap
```

#### Scenario: lint-weak-pin

```gherkin
      Given a grounded page pinning a span by bare line: with no version:
      When projectwiki_lint runs
      Then a weak_pin warning is reported
```

#### Scenario: lint-missing-title-and-fix

```gherkin
      Given an indexed page with an empty title
      When projectwiki_lint --fix runs
      Then the title is backfilled from the rel_path
```

### Requirement: status breakdown

projectwiki_status SHALL report indexed count, run phase, page counts
by type, stale count, pending generation jobs, and orphan count.

#### Scenario: status-reports-indexed-pages

```gherkin
      Given an indexed vault
      When projectwiki_status runs
      Then the indexed count and a non-empty phase are reported
```

#### Scenario: status-breakdown

```gherkin
      Given pages of two types, one mutated without re-indexing
      When projectwiki_status runs
      Then by-type counts, orphans, pending, and stale: 1 are reported
```

### Requirement: cite-source and eval

projectwiki_cite_source SHALL record an external citation with its
build fingerprint, idempotently; projectwiki_eval SHALL measure
recall@K against the frozen baseline.

#### Scenario: cite-source-returns-content

```gherkin
      Given a vault page
      When projectwiki_cite_source records an external citation twice
      Then the citation exists once, with its build fingerprint
```

#### Scenario: eval-recall-at-one

```gherkin
      Given a frozen eval case set over an indexed vault
      When projectwiki_eval runs
      Then recall@1 is reported against the baseline
```

### Requirement: 013 contextual page types

The four 013 page types (sources, concepts, syntheses, preferences)
SHALL be first-class: taxonomy, index whitelist, path validation,
bootstrap scaffold, and the documented [[type/slug]] link form SHALL
resolve into in_degree (ADR-0060).

#### Scenario: validate-page-path-contextual-types

```gherkin
      Given a page path under each 013 type folder
      When the path is validated
      Then every 013 type is accepted
```

#### Scenario: collect-pages-whitelist-scope

```gherkin
      Given a vault with pages inside and outside the 001 §7 whitelist
      When pages are collected for indexing
      Then only whitelisted folders are collected, including the 013 four
```

#### Scenario: index-in-degree

```gherkin
      Given pages linking by title, alias, slug, rel path, and [[type/slug]]
      When the index recomputes in_degree
      Then every documented link form counts toward the target page
```

### Requirement: transport parity

The 12 projectwiki_* tools SHALL be served identically over CLI and
MCP through the shared ProjectWikiHandler (008 §3/D1 contract), with
the write queue and ownership assertion on generate.

#### Scenario: projectwiki-dispatch-reads

```gherkin
      Given the MCP projectwiki server over a fixture vault
      When projectwiki_index, query, read, search, validate, lint, status run
      Then query carries thin+fallback and status carries the T7 breakdown
```

#### Scenario: projectwiki-tool-contract

```gherkin
      Given the MCP projectwiki server
      When the tool list is requested
      Then exactly the contracted projectwiki_* tools are served
```

#### Scenario: projectwiki-generate-queue-and-ownership

```gherkin
      Given a generate call with a mismatched expected_project
      When projectwiki_generate is dispatched
      Then the call is rejected; a matching call runs on the write queue
```
