---
id: TASK-054
title: '[FEATURE] Dep package: SBOM ingest + soft-retire + affected + graph (mnemonic)'
status: done
assignee: []
created_date: '2026-10-07 11:31'
updated_date: '2026-10-07 13:16'
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

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
dep package: SBOM ingest + soft-retire + affected + graph.

## What
- `sbom.go`: `ParseSBOM` (CycloneDX `components[]` → []Pkg by purl, `dependencies[]` → []Edge; SPDX stub returns a clear "not yet supported" error). purl from `bom-ref`, else `purl`, else synthesized `pkg:generic/<name>@<version>`; ecosystem derived from the purl type segment.
- `service.go`: `Ingest` (one tx: `INSERT ... ON CONFLICT(purl) DO UPDATE SET retired=0, last_seen` per purl, then `DELETE FROM dep_edges WHERE from_purl IN (<set>)` + reinsert edges, then `UPDATE dependencies SET retired=1 WHERE purl NOT IN (<set>)` — soft-retire, never delete); `Affected` (reverse `depends_on` BFS over dep_edges WHERE to_purl=?, depth-capped 10); `Graph`, `List(retired)`, `Get`.
- Fixtures sbom.cyclonedx.json (app→flask→werkzeug), sbom-noflask.json (flask absent), sbom.graph.json + tests.

## Why
TASK-054 — dependency graph ingest + reverse-impact query. SATISFIES `happy path dep ingest upserts by purl and soft-retires absent` + `happy path dep_affected returns reverse dependency set`.

## Where
- mnemonic/internal/dep/{service.go, sbom.go, dep_test.go} + fixtures/*.json

## Verified
- `go test ./internal/dep/ -v -count=1` → 8/8 PASS (upsert-by-purl, soft-retire leaves flask retired=1 with row surviving, Affected(werkzeug)={flask,app} transitive, Graph, List, Get, ParseSBOM CycloneDX, unsupported formats).
- `go build ./...` + `go vet ./internal/dep/` clean.

## Key Learnings
- Store DB handle: `st.DB *sql.DB` public field; `store.Open(dataDir, projectID)` is the exact two-arg signature; no testStore helper → tests use store.Open(t.TempDir(), "deptest") + t.Cleanup(st.Close).
- 050 DDL column surprise: `dependencies` columns are `purl, name, version, ecosystem, manifest, retired, last_seen` — NOT group/package_type as the brief suggested.
- Soft-retire SQL (the crux): upsert clears retired, then `UPDATE ... SET retired=1 WHERE purl NOT IN (<ingested>)`; retired rows survive, last_seen preserved.
- Bug caught by TDD: `List` first written as `WHERE retired = 1-retiredBool(retired)` (inverted); TestList caught it → `WHERE retired = retiredBool(retired)`.
<!-- SECTION:FINAL_SUMMARY:END -->
