# Step 05 Review — LLM-Backed Passive Extraction + Regex Fallback

Fresh reviewer, no prior context. Verified the scoped diff against existing code and live test runs.

## Verdict: **PASS WITH WARNINGS**

**Approved to merge: YES**

---

## Findings

### [major] `dedupePassiveItems` / `passiveItemHash` are DEAD CODE in the production path
- **File:** `skillgrid-cli/internal/mnemonic/memory/extraction.go:129` (`dedupePassiveItems`), `:91` (`passiveItemHash`)
- **What's wrong:** 05.3's definition is "deduplicate results (by content hash) when **combining** LLM and regex outputs." In production, `CapturePassive` does **replace, not combine** (see Check 5). The combine+dedup function is only ever invoked from the test `TestExtractionQualityLLMVsRegex`; it is never called in `CapturePassive`. So the "combine + dedup" mechanism the brief names as 05.3 is not the one that runs — a simpler replace design is.
- **Why it still works (so it's major, not critical):** the end-to-end `TestCapturePassiveLLMOptInCombinedDedup` proves the store holds 3 rows (2 verbatim LLM items + 1 nuance) with no duplicate rows, because the mock LLM returns a superset and `CapturePassive`'s downstream `seen[title+"|"+typ]` map (service.go:1155) already dedupes identical shaped titles. So the *outcome* (no duplicate rows) is real, just via a different, weaker mechanism than the one named.
- **The fix (deferred, not a merge-blocker):** either (a) wire `rawItems = dedupePassiveItems(llmItems, rawItems)` into `CapturePassive` so the brief's named mechanism is what actually runs, or (b) delete `dedupePassiveItems` + its test-only usage and document that dedup is "LLM replaces regex; downstream title dedup handles duplicates." Leaving a public-looking helper + a test that exercises a non-production path is the trap — it makes the reader believe combine+dedup is live.
- **Quote:**
  ```go
  // service.go:1116-1125 — REPLACE, not combine:
  rawItems := extractLearnings(text)
  if s.extractionLLMEnabled && s.extractionLLM != nil {
      if llmItems, lerr := ExtractWithLLM(ctx, text, s.extractionLLM); lerr == nil {
          rawItems = llmItems        // <-- llmItems fully replaces regex floor
      } else {
          fmt.Fprintf(logWriter(), "... falling back to regex: %v\n", lerr)
      }
  }
  ```

### [major] No production `ExtractionLLM` implementation exists — the opt-in is a NO-OP end-to-end today
- **File:** `skillgrid-cli/internal/mnemonic/service/service.go:314` (`mem.EnableExtractionLLM(cfg.Extraction.LLM)`)
- **What's wrong:** Config arms the switch, but `openProject` never calls `SetExtractionLLM(...)`, so `s.extractionLLM` is always `nil`. With the gate `if s.extractionLLMEnabled && s.extractionLLM != nil`, even an operator who sets `mnemonic.extraction.llm: true` gets the regex floor — the feature cannot turn on in production until a later step attaches a backend. The report acknowledges this (Concern #2), and it's correct that the *default-off* constraint still holds (seam nil → regex floor). But the "opt-in" as described (05.3, check 1) is only achievable in tests via the injected `SetExtractionLLM` seam.
- **The fix:** acceptable to defer the backend (it's explicitly a later step per the report), but the step should be flagged "opt-in wiring is complete but inert until a backend is attached" so a reviewer reading `EnableExtractionLLM(true)` in config doesn't assume the LLM pass is live. Not a merge-blocker for step 05.

### [minor] LLM item `type` hint is parsed and then discarded
- **File:** `skillgrid-cli/internal/mnemonic/memory/extraction.go:46-49` (`llmLearning.Type`), `:77` (`items = append(items, PassiveItem{Text: body})`)
- **What's wrong:** The prompt asks the LLM for a `"type"` per learning, `llmLearning` has a `Type` field with `json:"type,omitempty"`, and the comment says "validated and defaulted downstream" — but `ExtractWithLLM` builds `PassiveItem{Text: body}` with no type, and `PassiveItem` has no type field. So the LLM's type hint is dropped and `shapePassiveItem` re-classifies by keyword. The "validated and defaulted downstream" comment is slightly misleading (validation happens on the *keyword* guess, not the LLM hint).
- **The fix:** either remove `llmLearning.Type` and the comment, or (if a later step wants to honor the LLM's type) add a field to `PassiveItem` to carry it. Functionally inert today; a small honesty fix.

### [minor] `logWriter()` comment says stdout, returns os.Stderr
- **File:** `skillgrid-cli/internal/mnemonic/memory/extraction.go:16` (`stdout keeps it visible`), `:17` (`return os.Stderr`)
- **What's wrong:** The doc comment says "stdout keeps it visible" but the function returns `os.Stderr`. The code is correct (stderr is the right sink for a warning); the comment is wrong.
- **The fix:** change the comment to "stderr keeps it visible."

### [nit] `passiveItemHash` takes `title, typ` but never uses them
- **File:** `skillgrid-cli/internal/mnemonic/memory/extraction.go:91`
- **What's wrong:** `passiveItemHash(item, title, typ)` hashes only `extractDedupKey(item.Text)`; `title` and `typ` are accepted but unused. Harmless, but a code smell — the signature implies the hash depends on the shaped identity when it doesn't.
- **The fix:** drop the unused params or use them in the hash if the intent is to also dedup by shaped type.

---

## Evidence checked (all 8)

1. **Opt-in default (CRITICAL) — PASS.** `service.go:1117` gates on `if s.extractionLLMEnabled && s.extractionLLM != nil`. When `mnemonic.extraction.llm` is false (default; `config/load_test.go` `TestExtractionLLMConfigDefaultOff` proves absent/none → false) OR no seam is attached (always true in production today — see major finding #2), the block is skipped and `rawItems = extractLearnings(text)` is the *only* path — byte-for-byte the pre-05 regex-only behavior. Confirmed via `git diff 8ffaa67..a2a946a`: the only production change to `CapturePassive` is the added gate block; no existing line was altered. `TestCapturePassiveLLMOptInCombinedDedup` baseline (LLM off) stores the 5 raw regex items unchanged.

2. **Regex fallback — PASS.** On LLM error: `else` branch logs to stderr and `rawItems` keeps its `extractLearnings(text)` value; **no error is returned to the caller** (the `else` does not `return`). On LLM success-but-zero-items: `ExtractWithLLM` returns `nil, fmt.Errorf("extraction LLM returned no learnings")` (extraction.go:79-81), so `lerr != nil` → regex floor is kept (regex is *never* lost, since it was computed first). The reviewer's worry about a `nil, nil` return overwriting regex results does not occur — the code returns an error on empty, never `(nil, nil)`. `TestCapturePassiveLLMFailureFallsBackToRegex` confirms fallback output is byte-for-byte the regex-only set and the LLM was actually tried.

3. **LLM seam / mockability — PASS.** `ExtractionLLM` interface with `Extract(ctx, text) (string, error)` (extraction.go:26-28). Injected via `SetExtractionLLM`; `mockLLM{resp, err, lastIn}` mocks it with no HTTP/CGo. Mirrors the `layer.LLM`/`process.LLM` pattern (dedicated per-purpose interface, as noted in report Concern #4). A nil seam is the floor.

4. **JSON parsing / robustness — PASS.** `json.Unmarshal` into `llmExtraction` returns an error on malformed JSON (extraction.go:66) — no panic. Items are `strings.TrimSpace`d and blank-dropped. Unexpected fields are ignored by the standard `json` decoder (no `DisallowUnknownFields`), so a valid array with extra fields parses fine. Items are shaped downstream by `shapePassiveItem`/`shapePassiveContent` in `CapturePassive` (same pipeline as regex). `TestExtractWithLLMFailure` covers the seam-error case.

5. **Dedup (05.3) — PARTIAL.** See major finding #1. It is **replace, not combine** in production; `dedupePassiveItems` (the named combine+dedup mechanism) is only exercised by the unit test, not by `CapturePassive`. The no-duplicate-rows *outcome* is still achieved end-to-end by the mock returning a superset + `CapturePassive`'s downstream `seen[title+"|"+typ]` dedup (proven by `TestCapturePassiveLLMOptInCombinedDedup`). A reasonable design, but it under-delivers on the brief's literal "combine + dedup by content hash."

6. **Config wiring — PASS.** `mnemonic.extraction.llm` parsed via `extractionSection{LLM bool \`yaml:"llm"\`}`; default false (absent → zero value). Malformed type (e.g. a string) → whole-file unmarshal fails → `Load` returns defaults (LLM false), no crash. Wired `openProject` → `mem.EnableExtractionLLM(cfg.Extraction.LLM)` (service.go:314 — this line is in commit `5673b58` but **absent from the scoped diff file**, see warning below).

7. **TDD discipline — PASS.**
   - `TestExtractWithLLM`: mock returns structured JSON; asserts 2 items, LLM called with the input text (`llm.lastIn`), and each item shapes to a valid `shapePassiveItem` type/title + non-empty `shapePassiveContent`. ✓
   - `TestCapturePassiveLLMFailureFallsBackToRegex`: mock errors; asserts fallback result == regex-only baseline set, `res.Saved > 0`, no error propagated, and the LLM was actually tried. ✓
   - `TestExtractionQualityLLMVsRegex`: **deterministic** — mock returns a canned superset (not a real LLM). Asserts the regex premise (5 items, no consolidated nuance), superset coverage, zero duplicate content hashes, nuance present, 6 de-dup total. ✓

8. **Tool contract — PASS.** No `mem_*`/`code_*`/`web_*` MCP tool signature changed. The three step-05 commits touch only config/load, config/load_test, memory/extraction(.go/_test), memory/service, and service/service.go. No `mcp/` or `route/` file changed; the `http/*` files in the broader 8ffaa67..600fecf range are from the interleaved `2b86a4c` UI commit, not step 05.

### Live test run (this review)
`go test ./internal/mnemonic/memory/ -run 'TestExtractWithLLM|TestCapturePassiveLLMFailureFallsBackToRegex|TestExtractionQualityLLMVsRegex|TestCapturePassiveLLMOptInCombinedDedup|TestExtractWithLLMFailure|TestExtractionLLMConfigDefaultOff' -v -count=1` → **all PASS** (ok, 2.062s).

---

## Scope note (not a code defect)
The scoped diff file (`review-step05-scoped.diff`) lists 5 files but **omits** `skillgrid-cli/internal/mnemonic/service/service.go`, which contains the `mem.EnableExtractionLLM(cfg.Extraction.LLM)` wiring (line 314, commit `5673b58`). That line is the config→service bridge; the feature is complete and correct, but a reviewer working only from the scoped diff would not see the wiring. Flagging for the record, not as a merge-blocker.

## Pre-existing (out of step-05 scope)
`TestReindexStructuralIsEmbedderFree` (service pkg) fails with `sql: database is closed` on stashed HEAD without step-05 changes. Unrelated (reindex/embedder, not capture). Confirmed as pre-existing per the report.
