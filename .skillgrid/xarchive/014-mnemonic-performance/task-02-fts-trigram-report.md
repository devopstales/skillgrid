# Task 02 Report — FTS Trigram/Wildcard Query Support

## Status: COMPLETE (3 sub-tasks, 3 commits)

## Per sub-task

### 02.1 [RED] Trigram mode — `f16eda6`
- `buildFTSQuery` (in `internal/mnemonic/memory/service.go`, NOT retrieve.go) now
  switches on `matchMode`:
  - `trigram` → `trigramFTSQuery`: each term expands to its 3-char substrings
    (terms < 3 chars use the whole term as a single "trigram"); fragments are
    OR-joined per term, and term groups are OR-joined. **Choice: OR across
    groups** — the broadest partial-match net, simplest predictable behavior
    for this step (brief allows OR; mirroring AND was not required).
  - `prefix` → `prefixFTSQuery`: each term becomes a wildcard prefix
    (`"fun"` → `"fun*"`); multi-term OR-joined (prefix is its own mode; OR is
    the simple choice, per brief).
  - Fallback guard: if the trigram/prefix result is empty (e.g. empty query),
    fall back to default phrase behavior (returns `""` for empty query).
  - `""` (default) and `"all"` branches are byte-identical to pre-change.
- Shared helper `quoteFTSTerm` (double-quote + `""` escaping) matches the
  default encoding.
- `TestBuildFTSQueryTrigramMode` (service_test.go) covers: trigram expansion,
  OR-join, default single-term unchanged, short-term trigram, empty-query
  fallback, prefix single/multi-term.

### 02.2 [RED] Phrase queries unchanged — `edc7656`
- `TestFTSPhraseModeUnchanged` (service_test.go): `buildFTSQuery("exact phrase
  match", "")` → `"exact" OR "phrase" OR "match"` (byte-identical to
  pre-change); `"all"` → AND-joined; quote-escaping preserved.
- Additive wiring: `internal/mnemonic/memory/retrieve.go` gained
  `SearchOwnerScopedRetrieveFTS` (the FTS-mode variant); the existing
  `SearchOwnerScopedRetrieve` now delegates to it with `matchMode ""`, so the
  MCP/HTTP read path is byte-identical. `rrfFallbackOwnerScopedFTS` validates
  the mode up front and threads it to `BlendedSearch` (vector leg unaffected).

### 02.3 [AFK] CLI `--mode` flag — `172fe1b`
- `skillgrid mem search` gained an **optional** `--mode` flag
  (`trigram|prefix|phrase|all`; empty = default phrase OR). `phrase` is
  normalized to `""`; invalid values rejected with exit 2 (`invalid match
  mode "..."`) before any store access.
- **Wiring (CLI only, per brief):** `mem.go` →
  `svc.BudgetedRetrievalAsRootFTS` (new additive facade in
  `service/retrieval.go`; the existing `BudgetedRetrievalAsRoot` is unchanged
  and now delegates to the shared `budgetedRetrievalHits` body) →
  `SearchOwnerScopedRetrieveFTS` → `BlendedSearch` → `buildFTSQuery`.
- **MCP `mem_search` tool unchanged** — its `match_mode` parameter is untouched
  (contract shape held; the flag is optional). Documented choice: 02.3 limited
  to the CLI subcommand, as the brief permits.
- `TestMemSearchModeFlag` (cmd/skillgrid/mem_test.go): trigram/prefix/phrase/
  all all return the budgeted search shape; `--mode bogus` → non-zero exit +
  clear error.

## New `buildFTSQuery` (trigram + prefix branches)

```go
func buildFTSQuery(query string, matchMode string) string {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return ""
	}
	switch matchMode {
	case "trigram":
		return trigramFTSQuery(terms)
	case "prefix":
		return prefixFTSQuery(terms)
	}
	escaped := make([]string, len(terms))
	for i, term := range terms {
		term = strings.ReplaceAll(term, `"`, `""`)
		escaped[i] = `"` + term + `"`
	}
	sep := " OR "
	if matchMode == "all" {
		sep = " AND "
	}
	return strings.Join(escaped, sep)
}

func trigramFTSQuery(terms []string) string {
	groups := make([]string, len(terms))
	for i, term := range terms {
		trimmed := strings.TrimSpace(term)
		if len(trimmed) < 3 {
			groups[i] = quoteFTSTerm(trimmed)
			continue
		}
		fragments := make([]string, len(trimmed)-2)
		for j := 0; j+3 <= len(trimmed); j++ {
			fragments[j] = quoteFTSTerm(trimmed[j : j+3])
		}
		groups[i] = strings.Join(fragments, " OR ")
	}
	q := strings.Join(groups, " OR ")
	if strings.TrimSpace(q) == "" {
		return buildFTSQuery(strings.Join(terms, " "), "")
	}
	return q
}

func prefixFTSQuery(terms []string) string {
	prefixed := make([]string, len(terms))
	for i, term := range terms {
		prefixed[i] = quoteFTSTerm(strings.TrimSpace(term) + "*")
	}
	q := strings.Join(prefixed, " OR ")
	if strings.TrimSpace(q) == "" {
		return buildFTSQuery(strings.Join(terms, " "), "")
	}
	return q
}
```

Also added (service.go): `ValidateMatchMode` (accepts `""`, `any`, `all`,
`trigram`, `prefix`) and `NormalizeMatchMode` (maps the CLI alias `phrase`
→ `""`).

## RED → GREEN evidence

| Sub-task | Test | RED | GREEN |
|----------|------|-----|-------|
| 02.1 | `TestBuildFTSQueryTrigramMode` | FAIL: `trigram mode missing trigram "hel" in "\"hello\""` (phrase output only); prefix `= "\"fun\"", want "\"fun*\""` | PASS (all sub-assertions) |
| 02.2 | `TestFTSPhraseModeUnchanged` | (written same cycle; passes structurally since the default branch is untouched) | PASS |
| 02.3 | `TestMemSearchModeFlag` | [AFK] — flag absent pre-impl | PASS (11.8s; 5 scenarios) |

## Test output

### Focused suite
```
$ go test ./internal/mnemonic/memory/ -run 'TestBuildFTSQueryTrigramMode|TestFTSPhraseModeUnchanged' -v
=== RUN   TestBuildFTSQueryTrigramMode
--- PASS: TestBuildFTSQueryTrigramMode (0.00s)
=== RUN   TestFTSPhraseModeUnchanged
--- PASS: TestFTSPhraseModeUnchanged (0.00s)
PASS
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory
```

### Full relevant suite
```
$ go test ./internal/mnemonic/memory/... -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory        12.031s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer  2.829s

$ go test ./cmd/skillgrid/ -run 'TestMemSearchModeFlag' -v -count=1
--- PASS: TestMemSearchModeFlag (11.81s)
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid

$ go test ./cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  50.594s
```

### Baselines (005/008/010/011/013)
```
$ go test ./internal/mnemonic/store/... ./internal/mnemonic/memory/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store     4.735s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory   12.031s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer 2.829s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route     2.386s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected 11.036s

$ go test ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community 0.856s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg       1.595s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex 11.713s

$ go test ./internal/mnemonic/service/... -count=1   (changed package)
FAIL github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service 23.060s
    --- FAIL: TestReindexStructuralIsEmbedderFree: count gamma: sql: database is closed

$ go test ./internal/mnemonic/mcp/... ./internal/mnemonic/http/... -count=1   (consumers, unchanged contracts)
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp   35.890s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http  9.073s
```

**Pre-existing failure (not caused by this change):**
`TestReindexStructuralIsEmbedderFree` fails identically on the unmodified base
commit `d7541f7` (verified via `git stash` → run → `git stash pop`; also
`-count=3` on the clean tree fails). It is a codeindex/pooling test, not a
memory/FTS test, and none of the files it exercises were touched by this step.
Carried over from step 01 (store-pooling) as an open concern.

## Commits

| SHA | Subject |
|-----|---------|
| `f16eda6` | feat(mnemonic): add trigram and prefix FTS match modes |
| `edc7656` | feat(mnemonic): preserve phrase-mode FTS backward compatibility |
| `172fe1b` | feat(mnemonic): add --mode flag to mem search for trigram and prefix |

## Global Constraints — held

- Additive only: `buildFTSQuery` default (`""`) and `"all"` branches
  byte-identical; no schema/migration touched; `SearchOwnerScopedRetrieve`
  and `BudgetedRetrievalAsRoot` keep their signatures (delegation preserves
  behavior).
- `mem_*`/`code_*`/`web_*` MCP tool contracts unchanged (`mem_search`
  `match_mode` parameter untouched).
- `modernc.org/sqlite` driver untouched; no CGO.
- Default match mode remains `""` (phrase-only OR).

## Concerns

1. **Tokenizer caveat (functional, not a blocker):** `observations_fts` is
   `tokenize='porter'` (001_initial.sql). Porter-stemmed FTS5 does not support
   substring MATCH, so trigram/prefix modes are query-shaping primitives (the
   unit-level contract of this step) that will not yield substring hits on the
   porter table until the FTS table is re-tokenized (`trigram` tokenizer). That
   schema change is additive-only and out of scope for 02 (02.1–02.3); it is
   the natural next step. `chunks_fts` already uses `tokenize='trigram'`, so
   the pattern exists in this codebase.
2. **Pre-existing failure:** `TestReindexStructuralIsEmbedderFree`
   (`service` package) fails on the clean base commit — carried over from step
   01, unrelated to this step's files.
3. **CLI-only flag:** `--mode` is wired to `mem search` only (brief permits
   this). The MCP `mem_search` tool and the HTTP `match_mode` param could be
   extended to `trigram`/`prefix` later via `ValidateMatchMode` (already
   exported) without further signature changes.
