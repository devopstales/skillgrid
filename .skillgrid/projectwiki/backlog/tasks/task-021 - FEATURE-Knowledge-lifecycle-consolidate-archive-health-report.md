---
id: TASK-021
title: '[FEATURE] Knowledge lifecycle: consolidate + archive + health report'
status: needs-triage
assignee: []
created_date: '2026-09-30 09:45'
labels:
  - mnemonic
  - second-brain
dependencies: []
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add a knowledge lifecycle to the second-brain store: consolidate near-duplicate observations, archive stale ones, and surface a health report — with an inline staleness warning appended to search results so the brain degrades gracefully instead of silently going stale.

Reference: brobertsaz/claude-os v2.4 "Knowledge lifecycle engine (dedup, consolidate, archive, health)" and v2.5 "Inline health checks" (stale > 24h → HIGH/CRITICAL warning appended to search result, cached so it never slows/breaks search). Mnemonic already dedups by hash within 24h and upserts by topic_key; consolidate/archive/health do not exist.

Scope:
- Consolidate: detect near-duplicate observations (same topic_key family / high FTS+semantic overlap) and merge into one canonical observation, preserving version history (append-only, prior content recoverable).
- Archive: retire stale/superseded observations (soft-archive, excluded from search by default, recoverable).
- Health: a report scoring the store (dupes, stale, orphaned graph refs, FTS drift). Reuse/extend mem_doctor.
- Inline: when a search returns, if the top observation's health is stale, append a lightweight staleness warning. Cache the check (e.g. 24h) so search latency is unaffected. Never block search on health.

Acceptance:
- Two near-duplicate observations can be consolidated into one; prior content is recoverable.
- An archived observation is excluded from mem_search by default and recoverable.
- mem_health (or extended mem_doctor) returns a store report.
- A search over a stale observation appends a staleness warning; a fresh one does not; a cached health check does not measurably slow search.
- No new dependency (locked constraint).

Source: .skillgrid/artifacts/08-second-brain-roadmap.md (change D, P1). Reference project: https://github.com/brobertsaz/claude-os (v2.4/v2.5).
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
