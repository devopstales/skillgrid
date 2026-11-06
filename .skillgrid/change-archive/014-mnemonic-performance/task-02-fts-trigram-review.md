# Task 02 Review — FTS Trigram/Wildcard Query Support + `--mode` flag

## Verdict: `PASS WITH WARNINGS`

Reviewed `d7541f7..172fe1b` (3 commits: `f16eda6`, `edc7656`, `172fe1b`; 6 files, +261/-14) against the step 02 brief, the implementer report, and the pre-change code at `d7541f7`.

---

## Findings

### F1 — major — `internal/mnemonic/memory/retrieve.go:290` (`rrfFallbackOwnerScopedFTS`)
**The "backward-compat byte-identical" claim holds for `buildFTSQuery` but NOT for the owner-scoped retrieve path it protects.**
The pre-change code called `s.BlendedSearch(ctx, query, "any", ...)` — the literal `"any"` — whereas the new code passes the `matchMode` through:
```go
// pre-change
hits, err := s.BlendedSearch(ctx, query, "any", "", Vector{}, limit*3)
// post-change
if err := ValidateMatchMode(matchMode); err != nil { return nil, err }
...
hits, err := s.BlendedSearch(ctx, query, matchMode, "", Vector{}, limit*3)
```
`SearchOwnerScopedRetrieve` (the MCP/CLI read seam) delegates with `matchMode == ""`, so the new code feeds `""` to `BlendedSearch` where the old code fed `"any"`. The claim that "" and "any" are semantically identical (both → `" OR "` join in `buildFTSQuery`) is **true**, but this is a *semantic* identity, not the *byte-identical* identity the report and Global Constraint assert for the default path. For every existing caller the query string ends up byte-identical — but the mechanism changed from a hardcoded literal to a passed-through variable.
**Fix (or at minimum, document):** either keep the `"any"` literal default (`matchMode := matchMode; if matchMode == "" { matchMode = "any" }` is not needed if you accept the semantic identity, but then the report should say "semantically identical" not "byte-identical"), or add a unit test that proves `BlendedSearch(..., "", ...)` and `BlendedSearch(..., "any", ...)` return identical `[]Observation` for a representative query, so the equivalence is *pinned by a test* rather than inferred from reading `buildFTSQuery`.

### F2 — major — `cmd/skillgrid/mem_test.go:171` (`TestMemSearchModeFlag`)
**The CLI flag test is shape-only — it does not prove trigram/prefix are actually used.**
All five positive scenarios assert `strings.Contains(out, `"observations"`)` — the same JSON shape returned by *any* successful `mem search`, regardless of mode. The test would pass if `--mode trigram` were silently ignored and routed through default phrase OR. The report even says "routes trigram/prefix through the FTS match-mode read path" but the test has no assertion that distinguishes trigram from default.
**Fix:** assert a mode-sensitive outcome. E.g. seed a store with a content word whose trigrams don't appear as whole tokens, then assert `--mode trigram` finds it while `--mode phrase` (default) does not — or at minimum assert the search returns the seeded observation in trigram mode and not in default mode. Without a mode-sensitive assertion, 02.3 is unverified at the CLI layer.

### F3 — minor — `internal/mnemonic/memory/service.go` (trigram/prefix helpers)
**The "fallback guard" in `trigramFTSQuery` / `prefixFTSQuery` is dead code.**
```go
q := strings.Join(groups, " OR ")
if strings.TrimSpace(q) == "" {
    return buildFTSQuery(strings.Join(terms, " "), "")
}
```
`terms` comes from `strings.Fields(query)` — every element is non-empty after trim, so `groups[i]` is always a non-empty quoted term and `q` can never be empty. The guard is unreachable. Not a bug (it's defensive), but the report presents it as a real behavior ("if the trigram/prefix result is empty, fall back to default phrase behavior") — in practice it only ever fires for the empty-query case, which is already handled two lines up in `buildFTSQuery` (`if len(terms) == 0 { return "" }`).
**Fix:** remove the dead guard, or add a test that exercises it (impossible with the current input contract — so removal is the honest move).

### F4 — minor — brief vs. implementation file placement
The brief says the change goes in `internal/mnemonic/memory/retrieve.go` (and even names the helper there), but `buildFTSQuery` already lives in `service.go` — the implementer correctly extended it there (right call). `retrieve.go` was touched for the `SearchOwnerScopedRetrieveFTS` / `rrfFallbackOwnerScopedFTS` threading, which is a *different* seam from `buildFTSQuery`. The brief's "Files:" line is therefore slightly off, but the implementer's choice is the only one that works. No code change needed; just note the brief was imprecise.

### F5 — nit — `cmd/skillgrid/mem.go` (usage string)
The usage line is now longer and wraps differently:
```
search <query> [--mode trigram|prefix|phrase|all] [--limit N] [--reader-owner X]
               [--item N] [--char N] [--timeout 3s]
```
Cosmetic only. Fine as-is.

### F6 — nit — `internal/mnemonic/memory/service.go` (ValidateMatchMode error string)
```go
return fmt.Errorf("invalid match mode %q (valid: , any, all, trigram, prefix)", matchMode)
```
The leading empty-string entry in the "valid:" list reads oddly (``valid: , any``). The CLI test asserts `strings.Contains(out, "invalid match mode")` so it's not test-breaking, but the human-facing message is a bit awkward. Could list `"" (default)` explicitly.

---

## Evidence checked (7/7)

1. **Backward compatibility** — `buildFTSQuery` default (`""`) and `"all"` branches are **byte-identical** to pre-change (verified against `d7541f7`). For `"exact phrase match", ""` → `"exact" OR "phrase" OR "match"` (identical); `"all"` → AND-joined (identical); quote-escaping preserved (`say "hi"` → `"say" OR """hi"""`). **However** the owner-scoped *retrieve* path that the report claims is byte-identical changed from a hardcoded `"any"` to a passed-through `matchMode` (F1) — semantically identical, not byte-identical.
2. **Trigram correctness** — `buildFTSQuery("hello", "trigram")` → `"hel" OR "ell" OR "llo"` (correct 3-char substrings, OR-joined, each quoted). Short-term `<3` chars → whole term quoted (`"hi"`). Empty query → `""` (handled before reaching the helper). Multi-term ORs groups. Each trigram is quoted via `quoteFTSTerm` (double-quote + `""` escaping). ✓
3. **Prefix correctness** — `buildFTSQuery("fun", "prefix")` → `"fun*"` (correct). Multi-term → `"fun*" OR "bar*"`. Quoting preserved (wildcard inside the quotes). ✓
4. **`--mode` flag** — optional (default `""`); validated by `NormalizeMatchMode` (maps `phrase`→`""`) + `ValidateMatchMode` (rejects invalid with exit 2, before any store access). MCP `mem_search` tool signature **untouched** in the diff — `tools_memory.go` is not in the changed file list; its `match_mode` parameter and `GetString("match_mode", "any")` default are unchanged. `BlendedSearch` signature unchanged. Global constraint (no `mem_*`/`code_*`/`web_*` contract shape change) **held**. ✓ (but see F2 — flag *use* is untested at the CLI layer)
5. **Test isolation + TDD** — `TestBuildFTSQueryTrigramMode` and `TestFTSPhraseModeUnchanged` assert the **exact** expected FTS string (`got != want`, not just contains). `TestFTSPhraseModeUnchanged` compares against the literal pre-change output, so it does prove the default is byte-identical at the `buildFTSQuery` level. The CLI flag test covers valid + invalid modes **but** the valid-mode assertions are shape-only (F2). New tests added; no existing test *modified* (diff of the two test files shows zero removed lines). ✓
6. **Additive-ness** — no existing exported signature changed. `SearchOwnerScopedRetrieve`, `rrfFallbackOwnerScoped`, `BudgetedRetrievalAsRoot` keep their signatures and delegate to new `...FTS` variants with `matchMode ""`. `buildFTSQuery` already had the `matchMode` param pre-change (it was extended, not reshaped). `retrieve.go` was touched to thread the mode (correct seam, even though the brief mislabeled the file — F4). ✓
7. **Tokenizer concern (implementer's #1)** — **Accurate and honestly documented.** Verified in `001_initial.sql`: `observations_fts` uses `tokenize='porter'` (line 39) and `chunks_fts` uses `tokenize='trigram'` (line 83). Porter-stemmed FTS5 does not support substring MATCH, so trigram/prefix modes are indeed query-shaping primitives that won't yield substring hits on the porter table until re-tokenized. The report states this in "Concerns" (not silently broken), notes the out-of-scope schema change is the natural next step, and cites `chunks_fts` as the existing trigram pattern. Accurate. ✓

---

## Global Constraints — held
- **Additive only**: no existing signature changed; delegation preserves behavior. ✓
- **`mem_*`/`code_*`/`web_*` contract shape unchanged**: MCP `mem_search` untouched. ✓
- **Default match mode `""` byte-identical**: at the `buildFTSQuery` level, yes (F1 notes the retrieve seam is *semantically* not *byte-identical*). ✓ (with F1 caveat)
- **modernc stays**: no driver change. ✓

## Approved to merge: **yes** (with F1 + F2 to address before the change is considered fully verified — neither blocks merge; both are test/claim-accuracy gaps, not correctness regressions)

**Top risks to close in verify (step 03 or a follow-up):**
- F2: add a mode-sensitive CLI assertion so trigram/prefix *use* is proven, not just shape.
- F1: pin the ""-vs-"any" BlendedSearch equivalence with a test, or reword the report from "byte-identical" to "semantically identical."
