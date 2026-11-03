## 02-fts-trigram

### Goal

Trigram/wildcard FTS query support.

### Out of scope / Non-Goals

- Store pooling, parallel search, TTL, extraction, embedder.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/memory/retrieve.go` — MODIFY: trigram/prefix FTS query support

**Interfaces:**
- Consumes: cached store handles from step 01; existing `buildFTSQuery` function
- Produces: `trigram` and `prefix` match modes in `buildFTSQuery`; `--mode trigram` / `--mode prefix` CLI flags

### Tasks

- [ ] 02.1 `[RED]` Trigram mode splits query into 3-character trigrams joined by OR
  - [ ] 02.1.a Write failing test (`TestBuildFTSQueryTrigramMode`): call `buildFTSQuery("hello", "trigram")` and verify the output contains trigrams like `"hel OR elh OR llo"`; verify existing phrase mode (`buildFTSQuery("hello", "phrase")`) still returns `"\"hello\""`
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestBuildFTSQueryTrigramMode'` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — extend `buildFTSQuery` in `retrieve.go` with `matchMode` parameter; for `trigram` mode, split the query into all 3-character substrings and join with OR; for `prefix` mode, append `*` to each space-separated term; default mode remains phrase-only
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestBuildFTSQueryTrigramMode'` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): add trigram and prefix FTS match modes`
- [ ] 02.2 `[RED]` Existing phrase queries unchanged by default
  - [ ] 02.2.a Write failing test (`TestFTSPhraseModeUnchanged`): verify `buildFTSQuery("exact phrase match", "")` (empty mode = default) returns `"\"exact phrase match\""` identical to pre-change behavior; verify no trigram or prefix transformation is applied when mode is empty
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestFTSPhraseModeUnchanged'` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation — ensure default match mode is `""` (phrase-only) and that the trigram/prefix code paths are only entered when explicitly requested; add guard that falls back to phrase matching if trigram produces empty results
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestFTSPhraseModeUnchanged'` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): preserve phrase-mode FTS backward compatibility`
- [ ] 02.3 `[AFK]` CLI `--mode` flag for trigram/prefix search
  - [ ] 02.3.a Write failing test (`TestMemSearchModeFlag`): invoke `mem search --mode trigram "partial"` and verify the search uses trigram mode; invoke `mem search --mode prefix "fun"` and verify prefix matching is used; verify invalid mode returns an error
  - [ ] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemSearchModeFlag'` — Expected: FAIL
  - [ ] 02.3.c Minimal implementation — add `--mode` flag to `mem search` CLI subcommand; pass the mode through to `SearchWithScope` → `buildFTSQuery`; validate mode values against `trigram`, `prefix`, `phrase`
  - [ ] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemSearchModeFlag'` — Expected: PASS
  - [ ] 02.3.e Commit — `feat(mnemonic): add --mode flag to mem search for trigram and prefix`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestBuildFTSQueryTrigramMode\|TestFTSPhraseModeUnchanged'` | PASS | | |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify default mode is unchanged for existing queries | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): FTS5 trigram and wildcard query support via opt-in match modes`

---

