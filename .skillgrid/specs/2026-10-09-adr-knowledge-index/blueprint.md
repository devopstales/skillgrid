# Blueprint — ADR-aware knowledge index + retrieval

Two independent, additive pieces. No schema migration, no new dependency.

## Part 1 — `DocInclude` carve-out so `.skillgrid/**/*.md` is indexed

The walk checks `Exclude` before `Include` and `shouldSkipDir` prunes the whole `.skillgrid/`
subtree. A new `DocInclude []string` field lets a file that matches `DocInclude` be indexed even
when it matches `Exclude`, while still honoring `Include` (when non-empty).

### T1.1 `codeindex.Config` + walk filter
- File: `mnemonic/internal/codeindex/indexer.go`
- Add `DocInclude []string` to `Config` (after `Exclude`).
- `Scan`: after the `matchesAny(exclude, rel)` check, short-circuit to index when
  `matchesAny(docInclude, rel)` is true. A file that matches `DocInclude` must also match
  `Include` (when `Include` is non-empty) to be indexed.
- `shouldSkipDir`: do NOT prune a dir when a `DocInclude` glob can still match inside it. Keep
  the prune for pure-`Exclude` dirs; add a `docInclude` parameter so a dir that is an ancestor of
  a `DocInclude` match is not skipped. Simplest correct form: pass `docInclude` and only
  `SkipDir` when `shouldSkipDir(rel, exclude)` is true AND no `DocInclude` pattern has a
  prefix that could descend into `rel`. (Implementation note: `**/.skillgrid/**/*.md` makes the
  `.skillgrid` dir itself an ancestor — the carve-out must not prune it.)
- Test: `indexer_test.go` — a tree with `.skillgrid/artifacts/04-adr-0001-x.md`,
  `.skillgrid/state.yaml`, `src/main.go`. Assert the ADR is scanned, the yaml is not, the Go file
  is.

### T1.2 fingerprint + watch parity
- Files: `mnemonic/internal/codeindex/fingerprint.go`, `watch.go`
- Add `DocInclude` to the fingerprint walk config and `WatchConfig`; apply the same
  exclude-short-circuit in both so the freshness gate and the comfort-layer watcher agree with
  `Scan`.
- Test: a fingerprint of the same tree includes the ADR path and excludes the yaml path.

### T1.3 config plumbing
- File: `mnemonic/internal/config/load.go`
- `Indexing`: add `DocInclude []string`.
- `mnemonicSection`: add `DocInclude []string \`yaml:"doc_include"\``.
- `mergeIndexing`: copy `section.DocInclude` into `out.DocInclude` when non-empty.
- `DefaultIndexing()`: set `DocInclude: []string{"**/.skillgrid/**/*.md"}`.
- Files threading it into the walker: `cmd/mnemonic/mcp.go` (both `codeindex.Config` and
  `WatchConfig`), `internal/service/service.go` (the 4 `cfg.Include` sites),
  `internal/http/server.go` (the `cfg.Include` site). Add `DocInclude: cfg.DocInclude` at each.
- Test: a config load with the default + a repo `.skillgrid/config.d/indexing.yaml` override.

## Part 2 — `adr_search` MCP tool

### T2.1 `search/adr.go` (new)
- File: `mnemonic/internal/search/adr.go`
- `type AdrHit struct { Path, Title, Status, Supersedes, Date, Snippet string; Score float64; SupersededBy string }`
- `func AdrSearch(db *sql.DB, query string, limit int) ([]AdrHit, error)`:
  1. `buildCodeFTSQuery(query)` → `chunks_fts MATCH`, join `chunks`/`files`, restrict to
     `f.path GLOB '**/04-adr-*.md'`, order by `bm25(chunks_fts)`, `LIMIT ?`.
  2. For each row, take the file path; fetch the file's first chunk text (the ADR head) and parse
     frontmatter: the block between the first `---` and the second `---` after the H1, keyed
     `status`, `supersedes`, `date`. Title = the H1 line (`# ...`).
  3. Supersedes chain (one hop): if a hit's `supersedes` names another ADR number that is also in
     the result set (or discoverable by a second query on the same table for that number), set
     the earlier ADR's `SupersededBy` to the superseding path and ensure the superseding ADR is
     present in the result set.
- Frontmatter parse: small, pure, no dependency (split on `---`, split lines on `: `).
- Test: `search/adr_test.go` — open a temp sqlite with the `chunks`/`files`/`chunks_fts` schema,
  insert two ADR files (0006, 0009 with `supersedes: 0006`), run `AdrSearch("vector search")`,
  assert both are returned, 0009 has `SupersededBy` empty and 0006 has `SupersededBy` = 0009 path,
  frontmatter fields are populated.

### T2.2 MCP tool registration
- File: `mnemonic/internal/mcp/tools_code.go`
- Register `adr_search` tool (input: `query` string, `limit` int optional) alongside `code_search`/
  `code_read`. Handler resolves the project store db (same pattern as `code_search`) and calls
  `search.AdrSearch`.
- Test: `mcp` handler test — seed an ADR chunk, call `adr_search`, assert the JSON shape.

## Execution order

Wave 1 (Part 1): T1.1 → T1.2 → T1.3 (T1.3 depends on T1.1/T1.2 field existing).
Wave 2 (Part 2): T2.1 → T2.2 (T2.2 depends on T2.1 function).

Part 1 and Part 2 are independent; Part 2's ADRs only become searchable once Part 1 has indexed
them, but the code for Part 2 does not depend on Part 1's field.
