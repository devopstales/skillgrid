# SDD ledger — plan: .skillgrid/specs/2026-10-03-local-ollama-models/tasks.md

## Commit position (start)
- branch: main
- HEAD: 14b30a9 (docs: publish tracker IDs to backlogmd)

## Pre-flight conflict scan

| # | Pair / Task | Produces → Consumes | Finding | Ruling |
|---|-------------|--------------------|---------|--------|
| 1 | T1 ↔ T2 (provider.go) | T1 adds `parseOllamaVersion`+`ollamaVersionAtLeast` helpers; T2 rewrites `pullMissingModels` to consume them (reads `spec.MinOllamaVersion` + calls `ollamaVersionAtLeast`). | Complementary, sequential (T1 Wave1 → T2 Wave2). No file conflict; T2 consumes T1's helpers by name — both defined per blueprint. | CLEAN — sequence T1 before T2. |
| 2 | T1 ↔ T4 (provider_test.go) | T1 adds `TestOllamaVersionAtLeast`; T4 adds `TestMergePreservesOperatorKeys`. Distinct tests, distinct sections. | No shared test names. | CLEAN. |
| 3 | T3 ↔ T4 (tests) | T3 adds `TestDefaultOllamaModelIsEmbeddinggemma` (embedder pkg); T4 adds `TestMergePreservesOperatorKeys` (install pkg) + updates existing install tests. Different packages. | No file conflict. | CLEAN. |
| 4 | T2 ↔ T5 (provider.go + provider_test.go) | T2 rewrites `pullMissingModels` (impl); T5 updates the existing `TestSetupProviderLocal` expectations (expects catalog) and adds `TestPullMissingModels`/`TestPullMissingModelsGatesHeavyBelowFloor`. T5 consumes T2's catalog + gating behavior. | Complementary; T5's tests target T2's rewritten function. Wave order T2(W2) → T5(W4) holds. T5 also updates `TestSetupProviderLocal` which T2's catalog now feeds. | CLEAN — sequence T2 before T5. |
| 5 | T3 (embedder/ollama.go) vs ADR-0023 (no new SDK, stdlib) | T3 flips one constant `DefaultOllamaModel`. | No SDK change; stdlib-only preserved. ADR-0023 respected. | CLEAN. |
| 6 | All ↔ ADR-0016 (fail-open floors) | Pull failure is non-fatal (existing pattern); T2 keeps the fail-open loop + adds a version warning. | Floors preserved; no new fatal path. | CLEAN. |
| 7 | T3 one-way-door | T3 changes the runtime embedder default for existing local installs. | Flagged in tasks.md as one-way; **controller pauses for human checkpoint before applying T3's change.** | GATED — human checkpoint. |

- Blueprint has a `## Plan Review` section? — Checking now.
- Note: blueprint.md is pure-technical; spec = briefing.md (interview-closed). Spec is the binding authority.

## Status
- (none complete yet)

## Plan Review section
- Blueprint has NO `## Plan Review` section (pre-existing; written before the gate was introduced). Proceeding without the structural gate; the execution review loop catches what it would have.
- Spec = briefing.md (interview-closed). Binding authority confirmed.
- Baseline: `go build ./...` OK; `go test ./internal/install/... ./internal/mnemonic/embedder/...` green.

## Status

Task TICKET-01 (TICKET-01/TASK-042): in_progress
- Implementer (inline): b67500f feat(install): add Ollama version-floor gate (parse + compare + at-least)
- Review round 1: APPROVED 0 blockers (1 minor Atoi-overflow parked as out-of-scope; 1 nit doc-trim applied c64d8fb)
- Commits: b67500f, c64d8fb
Task TICKET-01: complete
Task TICKET-03 (TICKET-03/TASK-044, ONE-WAY-DOOR): in_progress
- Human checkpoint: APPLIED (user chose "Apply and continue")
- Implementer (inline): 38b0bced feat(embedder): default Ollama embed model to embeddinggemma:300m
- Review round 1: APPROVED 0 blockers (2 nits out-of-scope/pre-existing)
- Commits: 38b0bced
Task TICKET-03: complete
Task TICKET-02 (TICKET-02/TASK-043): in_progress
- Implementer (inline): 442f47ba feat(install): six-model Ollama catalog with version-floored pulls
- Catalog: 6 models (llama3.2:1b chat, embeddinggemma:300m embedder, qwen2.5:1.5b system-one, tev1:0.8b + gemma2:2b + clef-flash research); heavy gated behind 0.35.1 floor via version seam (fail-open, ADR-0016).
- Gate tests added: TestCatalogPullList, TestFloorGatesHeavyModels, TestAtFloorPullsAll, TestSmokeProbeNonFatal — all pass.
- Legacy tests updated in-ticket (tasks.md lists them under T2 deliverables): TestSetupProviderLocal (now asserts 6-model catalog, floor seam at/above), TestSetupProviderEnsureSkipsPullWhenPresent (all 6 listed present, floor seam).
- install + embedder packages green. Pre-existing failures (NOT this diff): cmd/skillgrid init/arch/eval, internal/mnemonic/project resolve_test.go x3.
- Review round 1: APPROVED, 1 blocker + 2 in-scope nits.
- Review fixes: 1553b346 — blocker: TestSmokeProbeNonFatal was vacuous; wired smokeProbe into pullMissingModels as a non-fatal post-pull health check (warn+record, never abort, ADR-0016), exposed as package-var seam, rewrote test to stub a failing probe and assert setupProvider returns nil + config merge runs. Nits: heavyModels() now used by the floor gate (name→floor map, was dead code); stale package doc refreshed.
- Commits: 442f47ba, 1553b346
Task TICKET-02: complete
Task TICKET-04 (TICKET-04/TASK-045): in_progress
- Test-only: added TestConfigMergeWritesNewModels — deterministic, server-free gate calling mergeHomeProviderConfig directly; asserts llm.model=llama3.2:1b + embedder.model=embeddinggemma:300m + base_url=ollama/v1, preserves profile + embedder.dimension.
- Review round 1: APPROVED, 0 blockers (1 cosmetic nit: comment on intentional 127.0.0.1-vs-localhost host divergence, applied).
- Commits: 3dc1dff, 5c15b65
Task TICKET-04: complete
Task TICKET-05 (TICKET-05/TASK-046): in_progress
- Legacy test updates (TestSetupProviderLocal, TestSetupProviderEnsureSkipsPullWhenPresent → six-model catalog + floor seam) already landed in TICKET-02 (442f47ba); this ticket verifies + records evidence.
- All G1-G6 acceptance gates PASS: G1 TestCatalogPullList, G2 TestFloorGatesHeavyModels, G3 TestAtFloorPullsAll, G4 TestConfigMergeWritesNewModels, G5 TestDefaultOllamaModelIsEmbeddinggemma, G6 TestCatalogPullList (glm excluded).
- go build ./... OK; go test ./internal/install/... ./internal/mnemonic/embedder/... -count=1 all ok.
- Recorded G1-G6 EVIDENCE in acceptance.feature (replaced "pending").
- Commits: 17d43abb
Task TICKET-05: complete

## QA gate
- skillgrid:qa run (efde48c): verdict **CONCERNS**. 0 CRITICAL.
- 7/7 blueprint truths VERIFIED by named passing tests; traceability 6/6 COMPLIANT; P0 100% / P1 100%; go build/vet/test exit 0.
- W1 WARNING (CLOSED 2026-10-06, human override "Fix W1 now"): implemented real per-role smoke dispatch (chat→/v1/chat/completions, embed→/api/embed + dimension recording, systemone→/v1/systemone, research no-op), embed dimension → mnemonic.embedder.dimension, mismatch warning (both lengths, model still switches), mnemonic.ollama.models list. New tests G7-G10 all PASS. acceptance.feature gained the 3 scenarios. Verdict re-run → PASS (0 CRITICAL, 0 WARNING, S1 still open).
- S1 SUGGESTION: state.yaml line 10 malformed YAML (pre-existing) breaks state-drift check. Still open.
- WINDOWS.md: W016 (W1) closed, W017 (S1) open. state.yaml phase -> qa.
- ROUTE: PASS → requesting-code-review.

## Change complete
All 5 tickets done, all G1-G6 gates green, full install+embedder suite green, go build clean.
Commits (main, local): b67500f, c64d8fb (T1), 38b0bced (T3 one-way-door), 442f47ba, 1553b346 (T2), 3dc1dff, 5c15b65 (T4), 17d43abb (T5 evidence), efde48c (QA gate).
Pre-existing failures (NOT this diff): cmd/skillgrid init/arch/eval, internal/mnemonic/project resolve_test.go x3.
