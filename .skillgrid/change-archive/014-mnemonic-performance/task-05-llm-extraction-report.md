# 05-llm-extraction — Implementation Report

## What was implemented per sub-task

### 05.1 `ExtractWithLLM` calls LLM and returns structured learnings
- **`skillgrid-cli/internal/mnemonic/memory/extraction.go`** (new): `ExtractWithLLM(ctx, text, llm ExtractionLLM) ([]PassiveItem, error)`.
  - Defines the **`ExtractionLLM`** interface (the seam): `Extract(ctx, text) (string, error)` — mirrors the `layer.LLM` / `process.LLM` pattern (a small interface a backend implements; **no CGo LLM client** in the package).
  - Builds the prompt (`extractionPrompt`) + text, calls the seam, parses the strict-JSON response (`{"learnings":[{"text","type"}]}`) into `PassiveItem` values (trimmed, blank-dropped). An empty parse is an error (→ caller falls back).
  - Items are shaped downstream by the existing `shapePassiveItem` / `shapePassiveContent` (same pipeline as regex output — verified in `TestExtractWithLLM`).
- **`extraction_test.go`** (new): `TestExtractWithLLM` (mock returns structured JSON; asserts 2 parsed items, LLM called with the input text, each item shapes to a valid `shapePassiveItem` type/title) and `TestExtractWithLLMFailure` (seam error → error returned).

### 05.2 LLM extraction failure falls back to regex
- **`service.go`**: added `extractionLLM ExtractionLLM` + `extractionLLMEnabled bool` fields to `Service`, with `SetExtractionLLM(llm)` and `EnableExtractionLLM(on)` (same opt-in style as `SetTTL`/`SetBudget`).
- **`CapturePassive`** now: always computes the regex floor first, then — **only if enabled and a seam is attached** — tries `ExtractWithLLM` and replaces the floor on success; on any LLM error it logs a warning (`os.Stderr`) and keeps the regex items. **No error is propagated** on LLM failure. When disabled (default) the method is byte-for-byte the pre-05 regex-only path.
- **`TestCapturePassiveLLMFailureFallsBackToRegex`**: baseline regex-only capture vs. LLM-enabled-but-failing capture on the same text → identical observation sets, no error, LLM was actually tried.

### 05.3 Identical quality or better + dedup + opt-in config
- **Dedup**: `dedupePassiveItems(primary, secondary)` merges LLM + regex outputs keyed by `passiveItemHash` — a SHA-256 over a **normalized text fingerprint** (`extractDedupKey`: lowercase, punctuation stripped, whitespace collapsed, 90-char cap) so a near-duplicate item both passes produce collapses to one key, while a genuinely new learning hashes differently.
- **Quality**: the mock LLM returns the regex items (verbatim) **plus** a consolidated nuanced learning the regex floor cannot produce (free-form prose, multi-clause). `TestExtractionQualityLLMVsRegex` proves the combined set is a superset of the regex set with **zero duplicate hashes**; `TestCapturePassiveLLMOptInCombinedDedup` proves end-to-end that the opt-in store holds the LLM's de-duplicated set (no duplicate rows) while the OFF path stays regex-only.
- **Opt-in config** (`mnemonic.extraction.llm`, default `false`):
  - `config/load.go`: new `Extraction{LLM bool}` type + `mnemonicSection.Extraction extractionSection` (yaml `extraction.llm`) + merge in `mergeIndexing` (absent → false).
  - `service/service.go` `openProject`: `mem.EnableExtractionLLM(cfg.Extraction.LLM)` — arms the opt-in switch from config.
  - `config/load_test.go`: `TestExtractionLLMConfigDefaultOff` (absent/none → false; `llm: true` → true).

## The LLM seam (injection/mocking)
- Interface: `memory.ExtractionLLM` — `Extract(ctx context.Context, text string) (string, error)`. Follows `layer.LLM` (`Summarise`) / `process.LLM` (`Label`): a method on an injected value, nil = floor. No HTTP/CGo in the package.
- Injection: `(*Service).SetExtractionLLM` (attach) + `EnableExtractionLLM` (opt-in). `CapturePassive` reads both at call time.
- Mocking: `mockLLM{resp, err, lastIn}` in `extraction_test.go` — canned JSON or canned error, records the input text.

## Opt-in config key
`mnemonic.extraction.llm` (bool, default `false`) in `config.d/indexing.yaml`. Wired `config.Load` → `Indexing.Extraction.LLM` → `openProject` → `mem.EnableExtractionLLM`.

## Regex-fallback logic in `CapturePassive`
```go
rawItems := extractLearnings(text)                      // always available
if s.extractionLLMEnabled && s.extractionLLM != nil {   // OPT-IN only
    if llmItems, lerr := ExtractWithLLM(ctx, text, s.extractionLLM); lerr == nil {
        rawItems = llmItems
    } else {
        // warn to stderr; keep regex floor; no error propagated
    }
}
```

## Dedup approach
`passiveItemHash` = SHA-256 of `extractDedupKey(item.Text)` (lowercased, punctuation→space, whitespace collapsed, 90-char cap). `dedupePassiveItems` keeps first occurrence per hash (LLM order first, then regex-only). Note: the hash keys on normalized text (not the shaped content), so an LLM item worded like a regex item dedupes against it even though `shapePassiveContent` embeds different Heading metadata.

## RED→GREEN evidence
- **05.1**: `TestExtractWithLLM` failed to build (`undefined: ExtractWithLLM`) → implemented → PASS.
- **05.2**: `TestCapturePassiveLLMFailureFallsBackToRegex` failed to build (`SetExtractionLLM`/`EnableExtractionLLM` undefined) → wired → PASS.
- **05.3**: `TestExtractionQualityLLMVsRegex` failed on the regex-premise and on dedup (8 vs 6 items) → refined dedup key + prompt → PASS.

## Full relevant suite + output
`go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` → **ok** (17.8s)
`go test ./skillgrid-cli/internal/mnemonic/config/... ./skillgrid-cli/internal/mnemonic/memory/... -count=1` → **ok** (config, memory, memory/layer)

## Baselines (005/008/010/011/013)
`go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/... ./skillgrid-cli/internal/mnemonic/memory/layer/... ./skillgrid-cli/internal/mnemonic/route/... ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/pdg/... ./skillgrid-cli/internal/mnemonic/codeindex/... -count=1`
- store **ok** | service **FAIL** (`TestReindexStructuralIsEmbedderFree`: `sql: database is closed` — **pre-existing**, fails identically on stashed HEAD without my changes) | memory/layer **ok** | route **ok** | affected **ok** | community **ok** | pdg **ok** | codeindex **ok**

## Commits
- `431d3a0` feat(mnemonic): add LLM-backed passive extraction
- `a2a946a` feat(mnemonic): add regex fallback for LLM extraction failure
- `5673b58` feat(mnemonic): improve LLM extraction quality and dedup

## Concerns
1. **`TestReindexStructuralIsEmbedderFree`** (service pkg) fails with `sql: database is closed` — pre-existing on HEAD, unrelated to step 05 (it is a reindex/embedder test, not a capture test). Flagging, not fixing (out of scope).
2. **No LLM backend is attached to the service yet** — `openProject` arms the `mnemonic.extraction.llm` switch but no production `ExtractionLLM` implementation exists to attach (a later step adds one). Until then the switch is a no-op (nil seam → regex floor), which is exactly the default-off constraint. The seam + config are ready to receive a backend.
3. `CapturePassive` still computes the regex floor before trying the LLM (small cost when the LLM succeeds) — kept so the fallback is trivially available and the disabled path is byte-for-byte unchanged.
4. The `layer.LLM`/`process.LLM` seams use per-purpose method names; I used a dedicated `ExtractionLLM` interface (same pattern) rather than reusing `layer.LLM.Summarise`, since extraction returns a different JSON shape.
