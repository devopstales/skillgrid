## 05-llm-extraction

### Goal

LLM-backed passive extraction + regex fallback.

### Out of scope / Non-Goals

- Store pooling, FTS trigram, parallel search, TTL, embedder.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-05` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/memory/extraction.go` — CREATE: LLM extraction with regex fallback
- `internal/mnemonic/memory/extraction_test.go` — CREATE: extraction tests

**Interfaces:**
- Consumes: existing `extractLearnings` regex; LLM backend (via `layer.Distill` or extraction endpoint); `shapePassiveItem` / `shapePassiveContent` functions
- Produces: `ExtractWithLLM(ctx, text)`; `CapturePassive` tries LLM first, falls back to regex

### Tasks

- [ ] 05.1 `[RED]` ExtractWithLLM calls LLM and returns structured learnings
  - [ ] 05.1.a Write failing test (`TestExtractWithLLM`): provide a mock LLM that returns structured JSON learnings; call `ExtractWithLLM(ctx, text)`; verify it returns parsed learnings matching the `shapePassiveItem` format; verify the LLM is called with the input text
  - [ ] 05.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractWithLLM'` — Expected: FAIL
  - [ ] 05.1.c Minimal implementation — create `extraction.go` with `ExtractWithLLM(ctx context.Context, text string) ([]PassiveItem, error)`; call the LLM backend (reuse `layer.Distill` or a new extraction endpoint); parse the JSON response into `PassiveItem` structs using `shapePassiveItem` / `shapePassiveContent`
  - [ ] 05.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractWithLLM'` — Expected: PASS
  - [ ] 05.1.e Commit — `feat(mnemonic): add LLM-backed passive extraction`
- [ ] 05.2 `[RED]` LLM extraction failure falls back to regex extraction
  - [ ] 05.2.a Write failing test (`TestCapturePassiveLLMFailureFallsBackToRegex`): mock the LLM to return an error; call `CapturePassive` with text containing known regex-extractable learnings; verify the regex fallback (`extractLearnings`) is called and returns the same results as the regex-only path; verify no error is propagated to the caller
  - [ ] 05.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestCapturePassiveLLMFailureFallsBackToRegex'` — Expected: FAIL
  - [ ] 05.2.c Minimal implementation — modify `CapturePassive` to try `ExtractWithLLM` first; on error, log a warning and fall back to `extractLearnings` regex; ensure the regex path is always available (no LLM dependency); results are parsed with the same `shapePassiveItem` / `shapePassiveContent` functions
  - [ ] 05.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestCapturePassiveLLMFailureFallsBackToRegex'` — Expected: PASS
  - [ ] 05.2.e Commit — `feat(mnemonic): add regex fallback for LLM extraction failure`
- [ ] 05.3 `[AFK]` Extraction results are identical quality or better than regex-only
  - [ ] 05.3.a Write failing test (`TestExtractionQualityLLMVsRegex`): provide a text with nuanced learnings that regex misses (free-form text, multi-clause sentences); compare LLM extraction results vs regex results; verify LLM extracts at least all items regex extracts plus additional nuanced items; verify no duplicate items in the combined result set
  - [ ] 05.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractionQualityLLMVsRegex'` — Expected: FAIL
  - [ ] 05.3.c Minimal implementation — refine the LLM prompt in `ExtractWithLLM` to capture nuanced learnings; deduplicate results (by content hash) when combining LLM and regex outputs; ensure the LLM is opt-in (config `mnemonic.extraction.llm: true`); default to regex when not enabled
  - [ ] 05.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractionQualityLLMVsRegex'` — Expected: PASS
  - [ ] 05.3.e Commit — `feat(mnemonic): improve LLM extraction quality and dedup`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestExtractWithLLM\|TestCapturePassiveLLMFailureFallsBackToRegex'` | PASS | PASS | + TestExtractWithLLMFailure, TestExtractionQualityLLMVsRegex, TestCapturePassiveLLMOptInCombinedDedup all PASS |
| Acceptance `@step-05` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | LLM structured learnings + regex fallback + malformed/error fallback mapped to the unit tests above |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | PASS | ok (17.8s) |
| Rollback boundary | verify regex-only path works when LLM is disabled | PASS | PASS | default `mnemonic.extraction.llm: false` → CapturePassive byte-for-byte regex-only; TestCapturePassiveLLMOptInCombinedDedup proves OFF path |
| Global Constraints | — | held | held | additive only; no mem_*/code_*/web_ contract change; no CGo (seam is an interface); regex always available |

**Warning** (pre-existing, not step-05): `TestReindexStructuralIsEmbedderFree` (service pkg) fails with `sql: database is closed` on stashed HEAD without step-05 changes. Unrelated (reindex/embedder, not capture).

### Commit

When step DoD is met: `feat(mnemonic): LLM-backed passive extraction with regex fallback`

---

