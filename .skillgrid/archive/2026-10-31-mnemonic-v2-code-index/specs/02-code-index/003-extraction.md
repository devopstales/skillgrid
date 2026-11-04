# 003 — Extraction (the tree-sitter ladder + the optional LSP tier)

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** `002-symbols-edges` (what it writes), `001-files-chunks` (the `files`
row it upserts, the `unresolved_edges` it writes), memory `00-foundation/003-
intelligence` (the LLM/embedder seam — the extraction itself is **not** LLM, it is
deterministic).
**Consumed by:** `004-graph-queries` (reads what this writes), `006-search` (the
`signature`/`text` it stores), the KB `006-validate` (the graph it builds).

---

## 1. Capability

Extraction is how the working tree becomes `symbols` + `edges` + `chunks`. It is
**deterministic** (no LLM in the default path — the LLM is the *embedder* seam for the
semantic tier, not the extractor). The pipeline is:

```
scan (walk the tree, detect language, read content, hash)
  → per changed file (one tx):
      extract (the tree-sitter ladder) → FileGraph {symbols, edges, unresolved}
      chunk (AST-first) → chunks
      write target-state (files + chunks + symbols + edges + unresolved_edges)
  → post-commit optional passes (separate connection): LSP / PDG / communities / process
```

## 2. The scan (the walk)

`filepath.WalkDir` over the project root (the cwd git root, resolved by the shared
project logic). Per file:
- **Exclude pruning** (directory-level, before reading): the standard excludes
  (`.git`, `node_modules`, `vendor`, build output, `.skillgrid/knowledge` — the KB vault
  is *not* indexed as code) + include/exclude globs (config).
- **Language detection** by extension (`extToLang` — the v1 map, kept). A file whose
  extension is unmapped is skipped (not indexed).
- **Read + hash:** content read, `sha256` (the `files.content_hash`).
- **Size-skip is a first-class outcome** (not silent): a file over `MaxFileSize`
  (default 500 KiB, config) is **not** indexed and is counted in
  `index_freshness.files_oversized` (`001` §2). The skip is reported in `code_status`.

**The incremental guard** (`001` §2): a file is **skipped** (no re-extract) iff
`mtime_ns AND size AND content_hash` are all unchanged vs the stored `files` row.
Otherwise it is extracted. (The `ast_hash` is *not* part of the guard — it is
informational, `001` §2.)

## 3. The extraction ladder (per file)

The ladder is **tree-sitter → self-heal → regex fallback**. It never errors; every file
produces a `FileGraph` (possibly empty).

### 3.1 Tier 1 — tree-sitter (the primary)

**gotreesitter** (the pure-Go binding, no CGo — the v1 choice, kept). Two sub-tiers:

- **The 6 rich one-pass extractors** (CI-O1, kept): **go, python, java, javascript,
  typescript, tsx** use gotreesitter's one-pass APIs (`ExtractDefinitionSpans` /
  `ExtractCalls` / `ExtractHeritage` / `ExtractImports`). These produce symbols +
  calls + imports + heritage in one parse. **Go is the primary target** (the skillgrid
  project is Go; the KB's coverage-gap lint is Go-package-scoped).
- **The 24 map-based extractors** (the rest of the 30 supported languages): per-language
  node-type maps (the v1 `languages.go`, kept as a data table, not 1481 lines of code —
  v2 stores the node-type → symbol-kind mapping as data). These produce symbols +
  imports; calls are best-effort (often name-only → `unresolved_edges`).

  **Counting rule (settled 2026-11-02, T0 revision):** 24 map-based extractors + 6 rich
  one-pass = 30 language *maps*; v1's `c` → `cpp` alias means the data table has 25 rows
  (the alias row shares the cpp map). "24" counts maps, "25" counts rows — tests assert
  the counting rule, not a bare number.

**Confidence assignment** (`002` §6): a statically-bound call (the callee is a known
symbol in scope) → `EXTRACTED`; a name-only call (the callee is not resolvable) → a
`to_id NULL` edge + an `unresolved_edges` row (CI-INV-3). Heritage (extends/implements)
→ `EXTRACTED` (explicit syntax).

### 3.2 The self-healing pool (the v1 mechanism, kept)

A panicking grammar **quarantines** the file to the regex fallback, respawns the grammar
(up to 3), and trips a **per-language circuit breaker** after 3 deaths (subsequent files
of that language go straight to the regex fallback). This keeps one malformed file from
taking down the whole language.

### 3.3 Tier 2 — regex fallback (the floor)

A battery of declaration/import/call regexes (the v1 `fallback.go`, kept): emits
`function`/`class` symbols + `imports` (target-only) + `calls` (name-only) edges. A file
with **zero** symbols from tree-sitter also falls back (the empty-extract guard). The
regex tier is the **deterministic floor** (INV-3): when tree-sitter is absent (a
language with no grammar), the regex tier is the path, and `code_status` reports the
extractor tier per language so the degradation is *visible* (never a silent floor).

## 4. The LSP tier (optional, CI-6)

**Default off** (`--lsp`). When on, it runs **after** the core graph tx commits
(separate connection, advisory — it never rolls back the committed graph). It resolves
**member call sites** (`receiver.Callee`) that the static pass could not bind:

- **Servers spawned on PATH** (the v1 set, kept): gopls (go), pyright-langserver
  (python), typescript-language-server (ts/js/tsx), rust-analyzer (rust), clangd
  (c/cpp).
- **Per-call timeout** (30s default, config). A failing/timing-out server → warn +
  continue, **never a partial edge set** (the LSP tier is all-or-nothing per call site).
- **Best-effort:** a resolved member call → an `edges` row with `confidence =
  LSP_RESOLVED` (the 4th tier, `002` §6). An unresolved one → the static-pass
  name-only edge stands (the LSP tier does not *remove* a static edge; it only *adds*
  a resolved one when it can).
- **The KB's `symbol:` anchor works either way** (CI-7): the pin-check reads the graph;
  with LSP off, a `symbol:` call-site check is best-effort (static resolution only);
  with LSP on, it is tighter. The degradation is explicit (`degraded: no_lsp`) when LSP
  is off and a call-site check is requested.

## 5. The knowledge nodes (route/doc/config/sql — written in the core tx)

v1 had separate tables + a separate post-commit pass for these. **v2 writes them in the
core extract tx** as `symbols` rows with a kind (`002` §3) + their `edges`
(`references`/`configures`/`reads`/`writes`/`navigates`). The extractors:

- **route:** framework routing extractors (the v1 `route/extract.go`, kept as data-driven
  patterns). Route nodes are `kind='route'` rows. The **route→handler resolution policy**
  (drop-not-guess, `002` §6): same-file match → `EXTRACTED` edge; unique-global-name
  match → `AMBIGUOUS` edge; multiple/zero → **dropped** (no edge) + an
  `unresolved_edges` row. A `navigates` edge (screen→screen) is `to_name` = route path,
  `to_id` often NULL, `INFERRED`.
- **doc:** markdown docs → `kind='doc'` rows + `references` edges (EXTRACTED markdown
  link / INFERRED wikilink).
- **config:** yaml/toml/json configs → `kind='config'` rows + `configures` edges to code
  (INFERRED by convention / AMBIGUOUS if unresolvable).
- **sql:** DDL → `kind='table'`/`kind='column'` rows; DML → `reads`/`writes` edges from
  code symbols (EXTRACTED).

**These are not optional passes** (they are part of the core graph). The *optional*
passes are PDG/communities/process/LSP (`005-advanced-passes`).

## 6. The chunker (AST-first, CI-O5)

One chunk per function/class/method (≤ 1000 chars target, `semanticChunkTargetChars`);
tiny leading segments merged to ≥ 250 chars; gaps between symbols become line segments;
over-long symbols line-windowed (80/10 overlap). No symbols (a config file) → plain
80-line chunks (`kind='lines'`). The chunker is deterministic (same input → same
chunks) so the FTS + semantic tiers are reproducible.

## 7. The extract tx (the write)

Per changed file, **one transaction** (`001` §6):
1. Upsert the `files` row (by `path`; the `sync_id`/`content_hash`/`ast_hash` update).
2. `DELETE FROM chunks WHERE file_id = ?` + insert the fresh chunks.
3. Upsert `symbols` `ON CONFLICT(uid)` (the file's fresh symbols).
4. `DELETE FROM edges WHERE file_id = ?` + upsert the file's fresh edges.
5. `DELETE FROM unresolved_edges WHERE file_id = ?` + insert the file's fresh
   `unresolved_edges` (the drop-not-guess ledger, `001` §5).

After all files: **orphan prune** (delete `symbols` whose `uid` is not in the target
set, `002` §9) + update `index_freshness` (`001` §4). The core tx commits. Then the
**optional passes** run (separate connection, advisory).

## 8. Language support

**30 languages in scope** (the v1 list, kept): go, typescript, tsx, javascript, python,
rust, java, c, cpp, c_sharp, php, ruby, kotlin, swift, scala, dart, lua, r, matlab,
perl, elixir, haskell, clojure, zig, nim, groovy, objective_c, bash, sql, css.
**6 get the rich one-pass extractors** (go, python, java, js, ts, tsx); the rest use the
node-type map; anything unmapped/malformed/panicking drops to the regex fallback.
**LSP tier (optional):** go, python, ts/js/tsx, rust, c/cpp. **Test-file detection:**
Go/Python/Ruby/JS/TS conventions (`004` §affected).

## 9. Determinism + invariants

**Determinism (INV-3 floor is deterministic):** the extraction is pure (same tree +
same extractor stamp → same `FileGraph`). The only non-determinism is the LSP tier
(external servers), which is advisory + all-or-nothing per call site. The **extractor
stamp** (`index_freshness.extractor_stamp`, `001` §4) is the version key: a change to
the extractor (a new version) invalidates the stored stamp → a full re-walk on the next
query.

**Invariants:** INV-1 (the `files`/`symbols` rows carry origin — `indexed_at`, the
extractor stamp), INV-3 (the regex floor is explicit, not silent), INV-6 (`sync_id`),
CI-INV-1 (derived — the whole graph is rebuildable from the tree), CI-INV-3
(drop-not-guess — a name-only edge is never a fabricated resolved edge).

## 10. Open items

- **CI-O1 (the rich-6):** go, python, java, js, ts, tsx (Go primary). Confirmed.
- **The node-type maps as data:** v1's `languages.go` (1481 lines) becomes a data table
  (per-language node-type → symbol-kind). This is an implementation detail, not a spec
  contract — the spec fixes *what* is extracted (symbols + edges + the confidence), not
  the per-language node-type list.
- **`MaxFileSize` + include/exclude globs:** config defaults (not spec constants). The
  size-skip is a first-class outcome (`files_oversized`).
