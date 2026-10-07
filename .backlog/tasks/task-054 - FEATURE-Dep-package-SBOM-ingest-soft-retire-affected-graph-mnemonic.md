---
id: TASK-054
title: '[FEATURE] Dep package: SBOM ingest + soft-retire + affected + graph (mnemonic)'
status: needs-triage
assignee: []
created_date: '2026-10-07 11:31'
updated_date: '2026-10-07 11:34'
labels: []
milestone: m-7
dependencies:
  - TASK-050
references:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/blueprint.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/tasks.md
documentation:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/artifacts/04-adr-0030-scan-findings-structured-store.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: dep package does not exist.\n\nExpected State: dep package: sbom.go ParseSBOM (CycloneDX components[] + dependency edges; SPDX fallback) -> []Pkg + []Edge; (*Service).Ingest(ctx, scanID, sbomPath) -> (added, retired, err); Affected (reverse BFS over dep_edges WHERE to_purl=?); Graph (full edge set); List(retired bool); Get. Fixtures sbom.cyclonedx.json, sbom-noflask.json, sbom.graph.json.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 ingest upserts dependencies by purl (last_seen set, retired cleared)
- [ ] #2 second ingest without flask: the flask row survives with retired=1 (soft-retire, never a hard delete)
- [ ] #3 Affected(werkzeug) on graph app->flask->werkzeug returns flask + app (reverse depends_on BFS, depth-capped at 10)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Ingest runs in one transaction: dependencies upsert + dep_edges delete/reinsert + soft-retire UPDATE
- [ ] #7 dep_edges replaced per-ingest so the graph reflects the latest SBOM
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 5 (tasks.md TICKET-05); work from mnemonic/ module
2. TDD: TestIngestUpsertsAndSoftRetires + TestAffectedReturnsReverseSet in dep/service_test.go RED first; author fixtures sbom.cyclonedx.json, sbom-noflask.json, sbom.graph.json
3. Implement sbom.go ParseSBOM (CycloneDX components[] + dependency edges; SPDX fallback); service.go Ingest (one transaction: upsert dependencies by purl clearing retired + set last_seen, delete+reinsert dep_edges for ingested set, UPDATE retired=1 WHERE purl NOT IN set), Affected (reverse BFS depth-capped 10), Graph, List, Get
4. go test ./internal/mnemonic/dep/ -v + go build ./... -> green
5. Commit: feat(mnemonic): dep package — SBOM ingest, soft-retire, affected, graph + Refs: TASK-054
<!-- SECTION:PLAN:END -->
