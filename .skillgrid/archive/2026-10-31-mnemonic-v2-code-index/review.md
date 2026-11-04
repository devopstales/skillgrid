# Review — Mnemonic v2 Code-Index Function

> Change: `.skillgrid/specs/2026-10-31-mnemonic-v2-code-index/` (moves to `.skillgrid/archive/2026-10-31-mnemonic-v2-code-index/` at ship)
> Generated: 2026-11-03T20:45:00Z (requesting-code-review)
> Diff range: `f5033704~1..771f0ea0`
> Rigor tier: T2 (Beta) — two-axis review (Standards + Spec + Security)
> Independence: Grade B per axis (fresh subagent contexts, same model family as implementer)
> Review budget: 3 rounds (tier T2)
>
> The durable audit record of the code review. **requesting-code-review** writes this into
> the spec folder, next to `report.md` (which owns the QA gate). It is committed in the spec
> zone before code-zone work continues. **ship** archives it with the folder; **reflect**
> cites it for lineage. The three review axes are reported **side by side, never merged** —
> that separation is the point.

## What Important means

> The explicit threshold that separates a finding worth fixing now from one to park.

- **Critical** — must fix before merge. Security, data loss, broken functionality, a
  `SATISFIES` scenario whose RED evidence is missing or whose code path does not match the
  Given/When/Then. Blocks `ship`.
- **Important** — should fix before merge. Architecture drift from a term or an in-force
  ADR, a hard standard breach, a missing or partial requirement, a behavior the spec never
  asked for (scope creep). Unfixed Important issues do not proceed (per the skill's
  Red Flags).
- **Minor** — nice to have. Baseline smells, style, optimization, doc polish. These are the
  nits; they are capped and summarized, never one-per-line below the fold.

## Cap the nits

- **Max 5 Minor findings listed per axis.** Beyond that, collapse the rest into a count.
- A Minor finding that is actually a Critical dressed as polish gets re-labeled, not capped.

## Do not report

- Anything tooling already enforces (lint, typecheck, format).
- Informational context with no required action — `FYI` at most, one line.

## Passes

### Standards

- **Worst issue (within this axis):** The PG code-index store has a compile-time duplicate (stubs in `postgres.go` + real impls in `postgres_code.go`) — the Standards reviewer's Critical #1 was a *pre-existing* conflict in the tree, not introduced by this diff's logic.
- **Findings:** 2 Critical / 4 Important / 5 Minor (capped to 5 listed)
- **Verdict:** met-with-fixes

### Spec

- **Worst issue (within this axis):** The TICKET-08a/b/c deferral (vec0/HNSW + LSP + PG render) is post-hoc — not recorded in the spec zone at T0, and the DoD item "one real Postgres run is on record" is not satisfied.
- **Findings:** 0 Critical / 2 Important / 3 Minor (capped to 5 listed)
- **Verdict:** met-with-fixes

### Security

- **Worst issue (within this axis):** `code_read` (full mode) is an unconfined arbitrary file read on an unauthenticated stdio MCP surface — any client (or prompt-injected agent) can read any file the process can read.
- **Findings:** 0 Critical / 1 High / 2 Medium / 2 Low / 0 Info (capped to 5 listed)
- **Verdict:** secure-with-fixes

## Findings

### Critical

- [Standards] `postgres.go:1601-1703` + `postgres_code.go` — **PG code-index store has a compile-time duplicate.** The `pgCodeIndexNotImplemented` stubs in `postgres.go` (25 methods) duplicate the real PG implementations in `postgres_code.go`. This is a *pre-existing* conflict in the tree (not introduced by this diff's logic — the stubs were supposed to be removed when `postgres_code.go` landed). **Fix:** remove the stub block from `postgres.go` (the real impls in `postgres_code.go` are authoritative). *(Verified: the stubs are already gone in the current tree — this was a stale-build-cache artifact. The compile conflict resolves on a clean build.)*
- [Standards] `extract/adapter.go:73-79` → `ladder.go:43` → `types.go:133-141` — **Circuit breaker never trips in production.** `DefaultExtractFn` calls `ExtractFile` with no `ExtractOption`, so `applyOptions` builds a **fresh `Pool` per file**. Each file's panic records its death in a pool that is immediately discarded — the breaker's `deaths[lang] >= TripAfter` never accumulates across files. The documented "4th go file goes straight to regex" decision point never fires. **Fix:** hold one `Pool` for the lifetime of the index run (construct in `CodeHandler`/driver, pass via `WithPool`).

### Important

- [Standards] `graph/walker.go:108,115,152`; `neighbors.go:140`; `path.go:99,106` — **`string(rune(symID))` trail queries are lossy and can be empty.** A symbol's numeric rowid cast to a Unicode code point: for rowids ≥ 128 → multi-byte/unprintable; for rowid 0 → `""`. The audit answer to "which symbol did `code_impact` read?" is an opaque, often-empty string. **Fix:** pass the symbol's **name** (already fetched at `walker.go:99` as `curSym.Name`) into `RecordTrail`.
- [Standards] `handler_code.go:302,475-495,540+,692` — **`CodeHandler` reads `h.st`, `h.emb`, `h.det` without the mutex it uses for `h.root`/`h.watcher`.** The three are set in `NewCodeHandler` and never mutated (effectively immutable), but the invariant is implicit — one future `SetEmbedder` away from a data race. **Fix:** document `st`/`emb`/`det` as immutable-after-construction on the struct.
- [Standards] `graph/affected.go:124-126` vs `walker.go:106-110` — **`code_affected` BFS swallows edge-read errors while `Impact` returns them.** `Affected` silently skips a node whose reverse edges can't be read (the affected-test set can be *incomplete with no signal*), while `Impact` treats the same failure as degraded-but-reported. **Fix:** on `EdgesByTo` error in `Affected`, accumulate the error (or a degraded marker) and surface it on `AffectedResult`, mirroring `Impact`.
- [Standards] `search/hybrid.go:334-344` — **`search.itoa` is a hand-rolled int→string that mishandles negatives.** The loop `for n > 0` yields `""` for `n < 0`. `StartLine` is `≥0` so negatives don't occur today, but the divergence from `strconv.Itoa` is gratuitous. **Fix:** delete `search.itoa`, use `strconv.Itoa`.
- [Spec] TICKET-08a/b/c deferral (vec0/HNSW + LSP + PG render) — **post-hoc, not recorded in the spec zone at T0.** The blueprint's Task 8a/b/c still list the named tests as gate tests (`TestSemanticStoreWired3Index`, `TestLSPAllOrNothingPerCallSite`, `TestPostgresCodeDialectRender` — none exist). The DoD item "one real Postgres run is on record" (tasks.md line 183) is not satisfied and was not explicitly waived. The deferral is REAL in the code (`code_status` reports `lsp:false`, `--lsp` returns `degraded: lsp_not_built`, the semantic leg is BLOB brute-force), but the spec zone and the deferral disagree on the version bucket (spec 005 §2 lists LSP as **v2**, G8 re-point calls it **v2.1-deferred**). **Fix:** amend the blueprint + ADR-0045/0046 + 000-overview INV-2 to record the deferral in the spec zone, or land the missing files.
- [Spec] `graph/affected.go:11-17` + `cmd/mnemonic/index.go:323-353` + `tools_code.go:105` — **`code_affected` base-ref mode (T8 must-have) is missing.** `AffectedOpts` has no `Base` field, the CLI has no `--base` flag, and the MCP tool's description promises a CLI flag that does not exist ("base reports the CLI command (`mnemonic affected --base <ref>`)"). This is a contract lie, not just a gap. **Fix:** add `--base` to the CLI + `Base` to `AffectedOpts` + git merge-base/diff, or remove the `base` arg from the MCP tool schema.
- [Security] `search/read.go:73,89` — **`code_read` (full mode) is an unconfined arbitrary file read.** `path` is the agent-supplied arg used verbatim in `os.ReadFile(path)` — no `filepath.Rel`/`..`/absolute check, no confinement to the index root. A prompt-injected agent can read any file the process can read (e.g. `/etc/passwd`, `~/.aws/credentials`). **CWE-22 · OWASP A01:2025 · LLM02 / ASI02.** **Fix:** confine `path` to the index root (reject `..`, absolute paths, and symlinks that escape the root).
- [Security] `graph/walker.go:66` + `graph/path.go:33` — **`code_impact` `max_depth` and `code_path` BFS are unbounded (DoS).** A positive `max_depth` arg is used with no upper bound; `code_path` runs BFS with no node cap. On a dense graph, an agent can walk the full reverse-dependency subgraph (one query per node). **CWE-400 · OWASP A04:2025 · LLM06.** **Fix:** cap `max_depth` (e.g. ≤ 10) and `code_path` node visits (e.g. ≤ 1000).

### Minor

- [Standards] `affected.go:225-231`, `orient.go:262-269`, `packages.go:86-91`, `semantic.go:106-110`, `path.go:157-162` — **Five copies of a hand-rolled insertion sort** instead of `sort.Slice`. *(Duplicated Code.)*
- [Standards] `graph/trail.go:39-50` — **`RecordTrail` uses a local `stringsJoin`** to avoid a `strings` import; the import cost is trivial. *(Consider importing `strings`.)*
- [Standards] `watch.WatchConfig.Debounced` — **field name is `Debounced`** but the constant/method are `DefaultDebounce`/`Debounce` — trailing `d` inconsistent. *(Mysterious Name, minor.)*
- [Standards] `handler_code.go:380-385` — **`newRunID` uses an unsynchronized package-level `runCounter`** (`runCounter++` under no lock) while `h.mu` guards the run state. Benign (the run_id is opaque), but an unlocked global mutation. *(Consider `atomic`.)*
- [Standards] `index.go:200-221` — **M10 delete does N+1:** `SELECT path FROM files` (all rows) then `DeleteFile` per row, outside any single tx. Works (FK cascades are per-statement), but a `DELETE FROM files WHERE path NOT IN (seen…)` inside the run's tx would be one statement. *(FYI — performance, not a bug.)*
- [Spec] `report.md:22,81` — **`TestSingleNodeTable` citation is a hallucination.** The coverage is real but inline (in `TestDoorCheckCodeIndexWritesAndSearches` + `TestMigrationParityAllVersions`), not in a named test. The "21/21 scenarios covered" claim stands.
- [Spec] `cmd/mnemonic/index.go:363,371` — **`code_grep --path` flag is parsed but unused.** `_ = path` — the CLI surface does not match the spec table. No scenario covers `--path`.
- [Spec] `handler_code.go` (`code_circles`) — **returns `cycles: []` without running any cycle algorithm.** No scenario covers `code_circles` (admin tool). Note-level.
- [Security] `server.go:436` — **MCP dispatch ignores the type-assert failure** (`args, _ := req.Params.Arguments.(map[string]any)`). A wrong-typed argument silently becomes nil → empty map. No direct injection (sinks are parameterized), but it weakens the tool-contract guarantee. *(Low, defense-in-depth.)*
- [Security] `search/read.go:50` — **Secret redaction is read-time-only and partial.** `secretRe` matches only a fixed keyword list; raw unredacted content is stored in SQLite `chunks.text` and `symbols.signature`. The primary exposure is the DB file/FTS, not the redaction path. *(Low, defense-in-depth.)*

## Independence

> The no-leaks proof. Each axis records its independence grade.

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | fresh subagent context, same model family as implementer (independent, weaker) |
| Spec | B | fresh subagent context, same model family as implementer (independent, weaker) |
| Security | B | fresh subagent context, same model family as implementer (independent, weaker) |

**Reading the grade:** **A** = fresh subagent, no shared history (independent). **B** = fresh
context but same model family / toolchain as the implementer (independent, weaker). **C** =
inline self-review or the implementer's verdict leaked in — **diagnostic only**, cannot stand
as the independent evidence that `ship` or a merge requires.

## Verdict

> The overall review verdict is the **floor** across the three axes (per
> `_shared/verification/floor.md`): the weaker axis caps the whole review.

- **Standards:** met-with-fixes
- **Spec:** met-with-fixes
- **Security:** secure-with-fixes
- **Floor (decides):** met-with-fixes
- **Worst issue (across all three axes):** `code_read` (full mode) is an unconfined arbitrary file read on an unauthenticated stdio MCP surface (Security, High) — any client can read any file the process can read.
- **Unfixed Important count (must be 0 to proceed):** 8 (2 Standards + 2 Spec + 2 Security High/Medium + 2 Standards Important)
