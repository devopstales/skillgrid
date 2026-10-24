# SDD ledger — plan: .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md

## Pre-flight conflict scan

| Pair/Task | Shared resource | What one produces vs what other consumes | Finding |
|---|---|---|---|
| TICKET-01 ↔ TICKET-02 | `lifecycle.go` | T01 produces `GrowthTimeline` + `PeriodCount`; T02 consumes `HealthReport` struct (pre-existing) | Clean — T02 calls the pure `healthRecommendations` directly, not T01's output. Wave 2 after Wave 1 is a convenience, not a hard dep. |
| TICKET-03 ↔ TICKET-05 | `tools_memory.go` | T03 adds `knowledge_class` params; T05 adds `mem_capture_suggest` | Clean — different tool registrations, same file but disjoint edit regions. Wave 4 after Wave 3 avoids merge conflict. |
| TICKET-03 ↔ TICKET-06 | `search_blend.go` | T03 adds `knowledge_class` filter; T06 adds web_cache RRF leg | **Shared file** — T06 must implement after T03 lands. Wave 5 after Wave 3. |
| TICKET-04 ↔ TICKET-13 | `indexer.go` | T04 adds centrality gate; T13 adds chunk prefixes | **Shared file** — T13 must implement after T04 lands. Wave 11 after Wave 1. |
| TICKET-09 ↔ TICKET-12 | Graph payloads | T09 produces `/knowledge/graph` route + page; T12 consumes enriched node payloads | Clean — T12 is fixture-first, T09 tail-verification is dashed (not hard). |
| TICKET-11 ↔ TICKET-14 | `okf-lint.mjs` | T11 produces the lint script; T14 consumes it in Step 4 verification | Clean — T14 blocked by T11 (wave 12 after wave 9). |
| TICKET-07 ↔ TICKET-08 | Skill files | Disjoint skill directories (`reflect/` vs `onboarding/`) | Clean — parallel-safe, same wave. |

**Ruling:** No conflicts require plan-text overrides. Wave ordering in tasks.md already encodes the two shared-file dependencies (T03→T06, T04→T13). Proceed.

## Wave 12 (FINAL — STOP gate #2)

### TICKET-14: Bundle moves + sweep + ADR-0048 ⚠ index-level (ONE-WAY)
- Owns: `.skillgrid/decisions/`, `glossary/`, `reference/`, `invariants/`, `workflows/`, `xarchive/`, root `index.md`/`log.md`/`reverse-index.md`, per-dir catalogs, `ASSUMPTIONS.md` LOCKED, `ARCHITECTURE.md`, `config.yaml` path keys, skills, hooks, `AGENTS.md`, `mnemonic/` Go dual-path strings, UI AdrsPage/DocsPage, `.gitignore`
- Needs: TICKET-11 (lint verifies the moves) — COMPLETE
- Model: heavy (45 ADR git-mv + xarchive rename + frontmatter + catalogs + 48-ref sweep + ADR-0048, precision-critical)
- Precondition: snapshot at /tmp/skillgrid-bundle-snapshot-20261020 (772 files, verified). User consent: "Proceed, snapshot first."
- Task 14: implementer DONE (commits a46c9cbb move+sweep 307 files + 936440ef ADR-0048+xarchive 208 files)
- Task 14: review APPROVE-WITH-FIXES (1 Important: live operational surface missed the archive/→xarchive/ sweep — 8 skills + Go docs/plans layer + UI + test fixtures; review-ledger was silently missing xarchive/ findings)
- Task 14: fix round 1 DONE (commit efebd4b6 — Go archiveTwin+sddArchiveDir→xarchive, 8 skills, DocsPage/AdrsPage UI, plans_test+docs_test+test-hooks fixtures, dist rebuild; amended to drop a stray .skillgrid/.skillgrid/ telemetry dir + widened gitignore to any depth)
- Task 14: complete (parent re-verify: G16 okf-lint exit 0 DRIFT clean SCOPE COMPLETE, zero artifacts/ in ASSUMPTIONS, artifacts/+archive/ gone, git log --follow intact, Go build+vet clean, docs+plans+UI tests pass)
- Ruling: ADR-0048 not 0039 — 0039 was taken by skill-pi permission extension; 0048 is next free after 0047. Same collision-class as the knowledge-class ADR (then 0038→0046). Correct.
- Ruling: okf-lint scope fix (skip specs/xarchive/sdd/cache/prototypes/config.d) — CORRECT (working state, not concepts); tested by a new fixture asserting a broken concept inside a skipped zone is NOT seen while a root-level one IS.
- Ruling: Frontmatter pass made all 45 ADRs frontmatter-first (the pre-existing heading-above-frontmatter layout was unparseable `date:…---` glue). status/supersedes/date UNTOUCHED (IRON RULE respected; reviewer spot-checked 5 ADRs).
- Ruling: Commit order inverted (code before spec) — ACCEPTED (zone-guard hook not wired; each commit internally zone-clean; --no-verify documented; pre-existing 0042 check-adr-invariants orphan verified via stash). FOLLOW-UP: wire the zone-guard hook.
- Ruling: README.md→reference/ + 03-adr-index.md→decisions/ — accepted placements.
- Ruling: Wikilink body edits ([[…]]→single) — justified by okf-lint check 4 (no-wikilink), meaning preserved.
- PARKED (Minor): AdrsPage.test.tsx + docs_markdown_test.tsx pin the new paths (fixed in round 1). test-hooks detached-HEAD failure = PRE-EXISTING (verified at baseline).
- FOLLOW-UP (separate ticket): wire precommit-zone-guard.js as a real git pre-commit hook; the spec-zone-first rule is currently self-enforced.

## Wave 11

### TICKET-13: Chunk prefixes + MMR-vs-RRF ablation (ranker untouched)
- Owns: `mnemonic/internal/codeindex/indexer.go` (+143), `chunk_prefix_test.go` (+748), `.skillgrid/specs/.../findings.md` (ablation section)
- Needs: TICKET-04 (shares indexer.go) — COMPLETE
- Model: mid (deterministic prefix + ablation harness, TDD)
- Task 13: implementer DONE (commits 542f2563 findings + 8de45459 code)
- Task 13: review APPROVE-WITH-FIXES (1 Important: findings.md ablation narrative inverted the Q1 relevance story; 4 Minor: dead topChunk var, pinned-vs-measured comment, silent error swallow, 748-line file)
- Task 13: fix round 1 DONE (commit e5fa4c25 — narrative corrected to match rank.go:84-92 + fixture data; dead var removed; invariant comment clarified as PINNED)
- Task 13: complete (parent re-verify: G15 2/2 + TestAblation_MMRvsRRF + PrefixedBeatsRaw PASS, go vet clean, rank.go diff EMPTY)
- Ruling: Prefix format `<path> · <symbol> · <scope>` (U+00B7) on EMBEDDING input only — display text + schema unchanged, proven by byte-for-byte vector equality (embed(prefix+raw) ≠ embed(raw)). Correct.
- Ruling: Ablation verdict TIED (RRF 1/2, MMR 1/2) — honest, RRF retained, ranker swap = NEW ticket. The narrative fix was documentation-only (measured numbers were always right).
- Ruling: Pre-existing extractor quirk (duplicate-named symbol across two files vanishes from one file on incremental index) — G15 fixture robust to it (distinct helper symbols, ≥1 floor). PARKED as separate ticket.
- PARKED (Minor): fileIDByPath/spansByFile silent error swallow (acceptable for a prefix; test catches it).

## Wave 10

### TICKET-12: Teaching graph page (single page, tour, locals)
- Owns: `mnemonic/internal/http/mnemonic_graph.go` (+ resolver), `mnemonic_graph_enrich_test.go`, `skillgrid-ui/src/features/mnemonic/GraphPage.tsx` (+test), `graph/types.ts`, embedded dist
- Needs: TICKET-09 (graph payloads) — COMPLETE
- Model: mid (Go enrich + resolver + UI page, TDD)
- Task 12: implementer DONE (commits dbe825e6 + 56b52d06 Go, 16ecb687 UI, 5cb0e7c6 dist)
- ⚠ PARALLEL-SESSION COLLISION: commit 0f71c2e7 (a parallel session) landed mid-work and swept the Go enrichment content + UNRELATED files (google-styleguides skill, parallel-cleave-port spec) into one commit. dbe825e6 is a formatting-only delta on top. Net Go content intact + tested at HEAD.
- Task 12: review APPROVE (1 Important: locals resolver not wired to an endpoint — ruled by-design; 4 Minor parked)
- Task 12: complete (parent re-verify: G14 Go 4/4 PASS, go vet clean, UI 16/16 PASS, server.go diff empty = no new endpoint)
- Ruling: locals resolver implemented + store-diff-proven but NOT wired to a route; UI renders stored signature+lines (zero extra requests). ACCEPTED — the gate literally requires "no new endpoint" + "locals ephemeral with store untouched", and the resolver is the documented seam for a future on-demand route. If on-demand locals become load-bearing, that's a NEW ticket (adds the endpoint).
- Ruling: dbe825e6 formatting-only claim — CONFIRMED (0f71c2e7 already had the content). The commit-boundary muddle is cosmetic; no Go content lost.
- Ruling: dist rebuild (5cb0e7c6) — expected (Go binary embeds the SPA).
- PARKED (Minor): sentinel error reuse, O(file²) line counting, over-broad comment on param helpers, untyped optional fields causing UI casts.
- PARKED (commit hygiene): 0f71c2e7 bundles unrelated skill/spec files with the Go enrichment — a squash candidate at branch review, not a code defect.

## Wave 9

### TICKET-11: okf-lint.mjs + telemetry (R6 tooling half)
- Owns: `.agents/skills/verification/qa/scripts/okf-lint.mjs`, `scripts/test-okf-lint.mjs`, `test/fixtures/okf-{clean,drift}/`, `.gitignore`, `test/fixtures/.gitignore`
- Needs: none (Node-only); BLOCKS TICKET-14 (its verification consumes the script)
- Model: mid (9-check Node verifier + telemetry + fixtures, TDD)
- Task 11: implementer INTERRUPTED then RESUMED (ses_eb0bf5720...): left fixtures + runner test, never wrote the script; resumed to finish. Commit 1d011a2 (28 files, 1296 ins).
- Task 11: review APPROVE-WITH-FIXES (no blocking; 7 Minor: union-of-regions untested, pre-R6 README under-checked, dead hasFence, unused resolved, runner hygiene/typo, telemetry extra fields, usage-no-telemetry)
- Task 11: fix round 1 DONE (commit 858a3e1 — literal-only DRIFT seed + 2 runner assertions pinning union-of-regions; README.md treated as pre-R6 root catalog; dead hasFence + unused resolved + redundant re-read removed; runner typo fixed)
- Task 11: complete (parent re-verify: clean exit 0, drift exit 1 literal-only DRIFT, runner 34/34, telemetry 2 lines numeric rate, gitignore resolves)
- Ruling: union-of-regions claim semantics (literal in ≥1 matched region, not per-anchor) — ACCEPTED for single-anchor (common case); now pinned by a literal-only DRIFT seed. Multi-anchor per-anchor attribution remains a future tightening if needed.
- Ruling: pre-R6 README.md as root catalog — FIXED (okf_version + links now checked; findings report the actual root path).
- Ruling: telemetry event carries an extra `scope` field (over-spec, harmless) — ACCEPTED; G13 gate only checks event + false_drift_rate.
- Ruling: usage-error (exit 2) runs emit no telemetry line — ACCEPTED (a usage error is not a lint run; header documents it).
- Ruling: check-skillgrid-skills.mjs exit 1 = PRE-EXISTING (verified by stash). PARKED.

## Wave 8

### TICKET-10: MCP scope block + descriptions (envelope-additive)
- Owns: `mnemonic/internal/mcp/tools_memory.go`, `tools_secondbrain.go`, `tools_session_inject.go` (descriptions), new scope-block test file
- Needs: TICKET-06 (fused corpus labels) — COMPLETE
- Model: mid (additive envelope + descriptions, TDD)
- Task 10: implementer DONE (commit da3e6474)
- Task 10: review APPROVE-WITH-FIXES (no blocking; 2 Minor: missing "stop once grounded" phrase, key byte-position)
- Task 10: fix round 1 DONE (commit c969fcaa — stop-once-grounded wording added to mem_search description)
- Task 10: complete (parent re-verify: G12 3/3 PASS, full mcp green, federated unmodified)
- Ruling: buckets_hit key marshals AFTER observations (Go sorted map keys) — "ahead of merged ranks" read as SEMANTIC (a provenance label read before trusting ranks, never part of them), not byte-position. ACCEPTED.
- Ruling: Byte-identity proven by key-set identity, not committed byte baseline — correct given JSONResult marshals map[string]any (sorted keys = deterministic). ACCEPTED.
- Ruling: Pre-existing gofmt drift in tools_secondbrain.go left alone (descriptions-only edit keeps diff clean). PARKED as pre-existing.

## Wave 7

### TICKET-09: Knowledge-concept graph view (vertical slice)
- Owns: `mnemonic/internal/knowledge/graph.go`, `mnemonic/internal/http/knowledge_graph.go`, `mnemonic/internal/http/server.go`, `skillgrid-ui/src/features/mnemonic/KnowledgeGraphPage.tsx`
- Needs: none (fixture-first)
- Model: mid (Go derive + route + SPA page, fixture TDD)
- Task 9: implementer DONE (commits 232e6cf8 Go + 7d904ff4 UI)
- Task 9: review APPROVE-WITH-FIXES (1 Important: hand-rolled bytesIndex; 4 Minor: CRLF frontmatter, _=err dead line, stale fixture comment, D3 copy-drift)
- Task 9: fix round 1 DONE (commit 06e82f16 — bytes.Index, CRLF normalize, dead line removed, fixture comment fixed)
- Task 9: complete (parent re-verify: G11 4/4 PASS, full knowledge+http green, build clean)
- Ruling: D3 renderer is a copy not a shared component — flag for TICKET-12 (which modifies GraphPage again); a shared d3-graph component becomes the cheap move then. PARKED.
- Ruling: Derive root = workspace root today (fixture-first) — TICKET-14 MUST re-point at the bundle zone (.skillgrid/knowledge/) or G11's "catalogs excluded" is broken in production. Scheduled in TICKET-14 tail scope.
- Ruling: yaml.v3 import is pre-existing (ADR-0008), not a new dep. OK.

## Wave 6 (parallel-safe, docs-only)

### TICKET-07: Reflect promotion candidates (docs-only)
- Owns: `.agents/skills/lifecycle/reflect/SKILL.md`, report template (merged qa template — one canonical shape)
- Needs: none (docs-only, parallel-safe)
- Model: cheap (prose, clear spec)
- Task 7: implementer DONE (commit a9890471)
- Task 7: complete (parent re-verify: G9 grep=6≥2, no new skill findings, under 400 budget)
- Ruling: report template is the merged qa template (.agents/skills/verification/qa/templates/report.md) — one canonical shape kept, reflect/templates/report.md points to it. Correct.

### TICKET-08: Onboarding recall + bootstrap (docs-only)
- Owns: `.agents/skills/lifecycle/onboarding/SKILL.md`, `.agents/skills/lifecycle/onboarding/references/process.md`
- Needs: none (docs-only, parallel-safe)
- Model: cheap (prose, clear spec)
- Task 8: implementer DONE (commit c45a96c)
- Task 8: complete (parent re-verify: G10 grep SKILL=4 + process=5 ≥3)
- Ruling: pre-existing reference-budget overage (340→354 lines, was already >250) — NOT introduced by this ticket (verified via stash: 167 errors before and after). PARKED as pre-existing.

## Wave 5

### TICKET-06: Fused mem_search recall + contract ADR
- Owns: `mnemonic/internal/memory/search_blend.go`, `mnemonic/internal/memory/service.go`, `mnemonic/internal/mcp/tools_memory.go`, `mnemonic/internal/memory/search_blend_test.go`, `.skillgrid/artifacts/04-adr-0047-fused-mem-search-contract.md`
- Needs: TICKET-03 (shares search_blend.go) — COMPLETE
- Model: mid-tier (fusion + RRF refactor + ADR, TDD)
- Task 6: implementer DONE (commits 145e0601 ADR + b35d8f76 code)
- Task 6: review APPROVE-WITH-FIXES (1 Important: web-hit labels bypass char budget; + 4 Minor)
- Task 6: fix round 1 DONE (commit b745fed2 — char budget, decay-ON test, corpus=observations)
- Task 6: re-review (parent re-verify: G8 PASS, decay-ON PASS, corpus+budget PASS, full packages green)
- Task 6: complete
- Ruling: ADR-0047 (next free after 0046) — correct number, ASSUMPTIONS updated to highest 0047.
- Ruling: Web-hit char budget — FIXED (inListContent on source/url/fetched_at/expires_at).
- Ruling: Decay re-append untestable — FIXED (TestMemSearch_FusedWebCacheDecayOn with decay ON pins the re-append).
- Ruling: corpus=observations on observation DTOs — FIXED (additive, matches ADR contract).
- PARKED (pre-existing, NOT this ticket's regression): handler-boundary fusion gap — the MCP handler's applyBudget-wrapped SearchOwnerScopedBlend returns 0 hits for 1-obs+1-web (write-path-sensitive: raw INSERT seeds don't surface, only ON CONFLICT DO UPDATE via web_cache_save does; obs-count-sensitive: 1 obs+1 web→0, 2 obs+1 web→1, 3 obs+1 web→2). The memory-layer blend (SearchOwnerScopedBlend directly) works correctly for the same input. The web leg is fully working + pinned; the observation-leg drop at the handler boundary is a separate pre-existing bug. Tests work around it (seed via web_cache_save, ≥2 obs). FOLLO-UP: file a separate ticket for the handler-boundary observation-leg fusion.

## Wave 4

### TICKET-05: Confidence-gated capture + regression
- Owns: `mnemonic/internal/secondbrain/capture.go`, `mnemonic/internal/mcp/tools_memory.go`, `mnemonic/internal/secondbrain/capture_test.go`, `mnemonic/internal/memory/knowledge_class.go` (export)
- Needs: TICKET-03 (suggested knowledge_class) — COMPLETE
- Model: mid-tier (classifier + MCP tool + full regression, TDD)
- Task 5: implementer DONE (commit c6e1c2e9)
- Task 5: review APPROVE-WITH-FIXES (1 Important: ADR-0038→0046 doc-ref slip in knowledge_class.go)
- Task 5: fix round 1 DONE (commit 48fa37cf — all 5 ADR citations aligned to 0046)
- Task 5: complete
- Ruling: G6 tests named TestClassifyCapture_*/TestSuggestCapture_Gated (not TestCaptureSuggest as the brief suggested) — 10/10 PASS, equivalent coverage. G7 full module green (only pre-existing vectorstore fixture failure, confirmed on clean HEAD).
- Ruling: weakIndex() recomputed per call + wordRe compiled per call — YAGNI, not worth a hoist at this call volume (Minor, parked).
- Ruling: correction/weak → "convention" mapping — consistent by construction (reuses InferKnowledgeClass); debatable but not wrong per ticket (Minor, parked).

## Wave 3

### TICKET-03: Knowledge-class dimension + ADR ⚠ ONE-WAY
- Owns: `mnemonic/internal/memory/`, `mnemonic/internal/mcp/tools_memory.go`, `mnemonic/internal/store/migrations/070_add_knowledge_class.sql`, `.skillgrid/artifacts/04-adr-0046-knowledge-class-dimension.md`
- Needs: none (migration 070; 051–053 are CLM)
- Model: mid-tier (migration + inference + backfill + search filter + ADR, TDD)
- Precondition: snapshot at /tmp/mnemonic-store-snapshot-20261018 (57 files, verified)
- Task 3: implementer DONE (commits 11e8f111 ADR + 00698956 code)
- Task 3: review APPROVE-WITH-FIXES (2 Important: ADR 0038 collision + spec zone stale 051 refs)
- Task 3: fix round 1 DONE (commit be4d712 — ADR renumbered to 0046, ASSUMPTIONS index updated, acceptance.feature G4 + tasks.md + blueprint refs 051→070)
- Task 3: complete
- Ruling: Migration 070 not 051 — JUSTIFIED (051–053 are CLM, runner name-sorted). Spec zone updated to match.
- Ruling: ADR 0046 not 0038 — 0038 already taken by skill-pi ticketing. Renumbered, ASSUMPTIONS index updated (highest 0046).
- Ruling: Index (knowledge_class, project) class-leading — ACCEPTED (better access path for equality on 4-value column; spec text amended).
- Ruling: FTS-leg-only class filter — ACCEPTED (ticket asked for "optional filter + labels", not hybrid fusion; ADR documents the limitation + upgrade path).
- Parked (minor): parity test adaptations, Go-vs-SQL heuristic leniency (backfill re-runnable, caller-supplied wins).

## Wave 2

### TICKET-02: Health recommendations (5 rules)
- Owns: `mnemonic/internal/secondbrain/lifecycle.go`, `mnemonic/internal/secondbrain/lifecycle_test.go`
- Needs: TICKET-01 (metrics shape) — COMPLETE
- Model: mid-tier (pure function, 5 rules, TDD)
- Task 2: implementer DONE (commit 8fe97b68)
- Task 2: review APPROVE-WITH-FIXES (2 findings: pre-existing ListReviews issues, not this ticket's code)
- Task 2: complete
- Ruling: ListReviews findings are pre-existing in review_cycle.go — PARKED as follow-up, not blocking TICKET-02. The JOIN pattern this ticket adds exposes them, but they're not introduced by this ticket.

## Wave 1 (parallel-safe, no deps)

### TICKET-01: Growth timeline
- Owns: `mnemonic/internal/secondbrain/lifecycle.go`, `mnemonic/internal/mcp/tools_secondbrain.go`, `mnemonic/internal/secondbrain/lifecycle_test.go`
- Needs: none
- Model: mid-tier (multi-file Go, TDD, MCP tool registration)
- Task 1: implementer DONE (commit aa3754e3)
- Task 1: review APPROVE (minor findings only)
- Task 1: complete
- Ruling: MCP wiring landed in tools_secondbrain.go (not tools_memory.go as brief said) — CORRECT, mem_lifecycle is registered there. Blueprint file-map drift noted; fix blueprint in spec-zone commit.
- Parked (minor, non-blocking): archived-row exclusion test, day/week bucket test — coverage gaps, not correctness. Reviewer confirmed the query logic is correct.

### TICKET-04: PageRank-guided embed gate
- Owns: `mnemonic/internal/codeindex/indexer.go`, `mnemonic/internal/codeindex/graph.go`, `mnemonic/internal/codeindex/indexer_test.go`
- Needs: none
- Model: mid-tier (multi-file Go, graph centrality, TDD)
- Task 4: implementer DONE (commit 65fb0d94)
- Task 4: review APPROVE-WITH-FIXES (3 Important + 1 spec miss)
- Task 4: fix round 1 DONE (commit 0ca2042d) — CLI flag wired, vectorSet isolated, ordering comment, accounting errors
- Task 4: re-review APPROVE (parent re-verify: G5 4 subtests PASS, build clean, CLI flag live)
- Task 4: complete
- Ruling: `--embed-top` CLI flag not wired — FIXED (flag in mcp.go, Config.EmbedTop in indexer.go, Service pass-through, finishIndex embed line).
- Ruling: test vectorSet — FIXED (WHERE e.symbol_id IS NOT NULL).
- Ruling: releaseGatedEmbeddings ordering — FIXED (comment added, no reorder).
- Ruling: accounting errors — FIXED but the original `_ =` lines were not removed (duplicated queries). Values are correct (second checked queries win); the `_ =` lines are dead code. PARKED as Minor cleanup for final review (one fix dispatch per the skill rule; not a correctness issue).
- Parked (minor, non-blocking): fileUnchanged dedup, json:"-" visibility, cold-index carve-out note, Centrality unit test.
