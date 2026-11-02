# Findings — Trail-Code (`trailhq/Trail-Code`) comparison

> Research digest for the code-index change: a deep-dive of Trail-Code's code
> mapping + storage, and the concrete list of what mnemonic v2's code index
> should adopt from it. Source: `trailhq/Trail-Code` at `main` (accessed
> 2026-11-02), npm `@nanonets/graft` v0.28.0, MIT. This is a **competitor
> comparison**, not a v1-research artifact — the v1 probe lives in
> `findings.md`.
>
> Method: researcher subagent read the actual source (`src/graph/*.ts`,
> `src/ask/ask.ts`, `src/mcp/server.ts`, `src/context/*`), not the README.
> Claims below cite Trail's file paths as read by the subagent; file-level
> spot-checks are recommended before any of this drives an ADR.

## 1. What Trail-Code is

A TypeScript/Node CLI + MCP server that builds a per-repo **"context graph"**
as **plain files on disk — no database, no embeddings**. Two products in one
repo: **graft** (the local code map; CLI `graft` + MCP `graft_*` tools) and
**trail** (team memory/skills via a cloud workspace connector; only the client
`src/mcp/connector.ts` is in this repo).

### 1.1 How it maps the code

**Three-tier extraction over tree-sitter** (`src/graph/build.ts` `tierOf()`,
`src/graph/extract.ts` ~117KB):

| Tier | Mechanism | Languages |
|---|---|---|
| **Depth** | hand-written per-language AST walkers; WASM grammars, pinned `web-tree-sitter` 0.22.2; python/kotlin/swift WASMs committed in `src/graph/grammars/` with provenance in `grammars.json` | TS/TSX/JS, Python(+.pyi), Go, Java, Kotlin, Swift, PHP, R |
| **Breadth** | **one generic extractor** over any tree-sitter grammar + its `tags.scm` queries (`@definition.*` nodes + `@reference.call` bare names; no receiver typing) | rust, c, cpp, ruby, c#, scala, elixir, solidity, ocaml, zig, dart, clojure, nix, lua, gleam |
| **Container** | `.vue` SFCs: grammar locates `<script>`, handed to the depth TS extractor | Vue |

Files are read in **<32KB slices** (tree-sitter WASM 32KB limit workaround);
one `file` node per file, one node per definition, deduped by span with
`~N` ordinal ids.

**Relationships / call graph** (`src/graph/resolve.ts`,
`src/graph/bindings.ts`) — 6 edge types: `contains`, `calls`, `imports`,
`references`, `implements`, `extends`. Edges are emitted as *intents* then
matched against a whole-repo name index with a **confidence ladder**:
`lsp_resolved` > `lsp_dispatch` > `extracted` (same-file match / import
specifier) > `inferred` (unique cross-file name match). **Ambiguous matches
are dropped, never guessed.**

- **Receiver binding** (`bindings.ts`): per-language binding pass
  (variable/field → type, `this`/`self`, Go receivers) + owner-qualified
  index; walks the `extends` chain (depth ≤ 3); PHP traits; Java/Swift
  **arity-based overload narrowing**.
- **Import resolution** is language-aware: Go `go.mod` module→dir, Java
  FQN→path-suffix, PHP PSR-4 suffix, C `#include` suffix, Rust `crate::`
  crate-root, Gleam src-root.
- **Language-family reachability guard** (TS/JS, JVM, C/C++ families only)
  prevents cross-language name collisions.
- **Opt-in LSP tier** (`graft build --lsp`, `src/graph/lsp/`): minimal
  `vscode-jsonrpc` client over rust-analyzer, clangd, gopls,
  pyright-langserver, typescript-language-server; runs call hierarchy and
  adds `lsp_resolved` call edges.

**Incremental vs full** (the core trick):

- **Fingerprint probe** (`src/graph/fingerprint.ts`): **every retrieval
  call** runs `probeDrift` — a walk + `stat(size, mtimeMs)` per file
  (~3ms / 280 files, no reads, no parsing); suspects confirmed by sha256.
  Clean → answer from disk; dirty → structural rebuild before answering
  (`src/graph/refresh.ts` `ensureFreshGraph`).
- **Per-file parse memo** (`src/graph/extract-cache.ts`): `extract.json`
  caches each file's parsed `{nodes, rawEdges}` keyed by content hash; the
  cache is keyed by an **extractor stamp** = sha256 over every
  `src/graph/*.js` module + package version, so any extractor change
  invalidates it automatically.
- Downstream (resolution, graph write) runs over the full node set, with a
  pinned invariant: **incremental output is byte-identical to a cold build**.
- Walk uses `git ls-files --cached --others --exclude-standard` (exact git
  ignore semantics); skips `node_modules/dist/build/...` + dot-dirs + files
  >1MB (`src/ingest/fs.ts`); optional submodule following in
  `.graft/config.json`.
- **Monorepo scopes** (`src/graph/scopes.ts`): sub-projects discovered via
  marker files (`package.json`, `go.mod`, `pyproject.toml`, `setup.py`,
  `Cargo.toml`, `composer.json`, `pom.xml`, `build.gradle[.kts]`); stored in
  `meta.scopes`; per-scope ranking.

### 1.2 What it stores (and where) — no database

Everything is `node:fs` + JSON under `<root>/graft/` (or
`<root>/.trail/cache/`; overridable via `--dir`/`GRAFT_DIR`), gitignored by
`graft build` but re-admitted to ripgrep via a root `.ignore`
(`!graft/`) so the markdown cards stay greppable:

```
graft/
├── .graph/
│   └── wiring.json              ← THE graph: all nodes + edges (machine-only, sorted, deterministic)
├── .cache/                      ← gitignored, regenerate-anytime
│   ├── ask-index.json           ← BM25 sidecar: per-node token bags + corpus df + avgBodyLen
│   ├── extract.<stamp>.json     ← per-file parse memo (nodes+rawEdges+hash per file)
│   ├── fingerprint.<stamp>.json ← per-file [size, mtimeMs, hash] prints for the drift probe
│   ├── summaries.json           ← LLM concept-summary + synthesis batch cache
│   ├── stats.json               ← {nodeCount, edgeCount, languages, readyCount, staleCount, dirty, syncing, …}
│   └── .sync.lock               ← exclusive-create lock (stale after 5 min)
├── workspace.json               ← workspace parents only: {version:1, children:[…]}
├── INDEX.md                     ← human/agent roster
├── <mirrored source tree>/*.md  ← one "wiring card" per source file
├── _root/                       ← root-level cards colliding with concept slugs
└── <concept-slug>.md            ← LLM-synthesized concept nodes
```

**`wiring.json` schema** (v1, `src/graph/types.ts`):

```ts
GraphV1 {
  meta: { version: 1, nodeCount, edgeCount, languages: string[], scopes?: ScopeV1[] }
  nodes: NodeV1[]   // id "src/cache.ts#Cache.get", name, kind (file|class|function|method|
                    //   interface|type|enum|struct|trait|module|constant|variable),
                    //   owner?, path, span "L165-L222", signature, exported,
                    //   origin "ast"|"generic", body_hash (sha256), chars?, body_text? (≤5000 chars,
                    //   whitespace-collapsed; stripped from the serialized file — lives in ask-index),
                    //   arity?, variadic?, summary_state "pending"|"ready"|"stale", summary,
                    //   crux {code, span}
  edges: EdgeV1[]   // { source, target, relation, confidence }
                    //   target = node id, or an unresolved module string for imports
}
```

`crux: {code, span}` is the LLM-chosen ≤8-line "business-logic core" of a
definition. Writes are atomic (temp+rename, pid-suffixed) and sorted so
unchanged repos produce byte-identical files. Concept nodes (opt-in `--deep`
LLM pass) are markdown with YAML frontmatter + a human-preserved `## Notes`
region; `manifest.json` holds the O(files) staleness ground truth.

**No vector/index files**: retrieval is BM25 (`ask-index.json`) + IDF +
personalized PageRank, explicitly "deterministic and $0 … no LLM, no
embeddings" (`src/ask/ask.ts`). The only DB-like thing in the product is the
cloud workspace (server-side, not in this repo), reached over MCP Streamable
HTTP with a `trl_` token.

### 1.3 How the data is consumed

- **Pre-query freshness**: every retrieval call first runs
  `ensureFreshGraph` (`src/graph/refresh.ts`) — drift probe, optional
  worktree **seeding** (copy parent checkout's graph in — `seed.ts`), and a
  `$0`, `graphOnly` structural rebuild if dirty (never LLM, never fatal,
  lock-serialized with the Claude Code Stop-hook background build).
- **MCP server** (`src/mcp/server.ts`): hand-rolled NDJSON-RPC 2.0 over
  stdio (no SDK), protocol 2024-11-05/2025-06-18; 6 tools, advertised only
  when a graph exists:

| Tool | Behavior |
|---|---|
| `graft_find_code` | `ask.ts`: "what calls X" → edge walk; else **lexical rank** = BM25 over name/path/body token bags + **personalized PageRank** (α=0.25, 25 iters, undirected over WALK_RELATIONS) seeded by lexical scores, blended; multi-scope RRF (K=60) fusion. Returns hits with exact `file:line` + inlined ≤8-line crux. |
| `graft_trace_calls` | BFS blast-radius over `calls`/`references`/`imports`/`implements`/`extends` edges; `contains` deliberately excluded from walks. |
| `graft_find_all` | regex over indexed files, hits grouped by innermost enclosing symbol, ranked by incoming-edge count. |
| `graft_file_api` | every definition's signature + span for one file. |
| `graft_repo_map` | directory clusters (split if one dir >60%), per-dir hubs + global hotspots by in-degree. |
| `graft_check_freshness` | re-extract, diff by id/body_hash → `added/removed/changed/stale/pending` (CI gate). |

- **Claude Code hooks**: prompt-hook injects `graft ask` packs; Stop-hook
  rebuilds the passive markdown surface; PreToolUse guards.
- **Passive channel**: the markdown cards + `INDEX.md` are directly greppable
  — `grep <symbol>` lands on a ~150-token card instead of the whole file.
- **Workspace federation**: parent dir with ≥2 git children holds
  `workspace.json`; `ask` fuses children by RRF; grep/map/check/callers run
  per-child and merge with `<child>/` labels.
- **CLI + engine**: `src/cli.ts` (~143KB) and the `Graft` engine class
  (`src/engine.ts`: `init/check/graph/ask`) are the same functions the MCP
  tools wrap — `mcp/tools.ts` is "pure functions over the existing engine."

## 2. Overlap with the v2 spec (already aligned — do not re-derive)

Both designs share:

- tree-sitter extraction as the deterministic core (no LLM in the extract
  path);
- confidence tiers — v2 `EXTRACTED / INFERRED / AMBIGUOUS / LSP_RESOLVED`
  (`specs/02-code-index/002-symbols-edges.md` §6) ≈ Trail `extracted /
  inferred / lsp_resolved`;
- **drop-not-guess** — v2 `to_id NULL` + `unresolved_edges` ledger
  (`001` §5, CI-INV-3) ≈ Trail "ambiguous matches are dropped, never
  guessed";
- **extractor-stamp** invalidation (v2 `index_freshness.extractor_stamp`,
  `001` §4) ≈ Trail's cache keyed by sha256 of extractor source;
- **stat-walk freshness** (v2 `001` §4 step 1: size+mtime vs `files` rows,
  hash-confirm) ≈ Trail `probeDrift`;
- **RRF fusion K=60** (v2 `006` §1, inherited from v1) = Trail's
  multi-scope fusion constant;
- optional **LSP tier** as an additive 4th confidence tier (v2 CI-6, `003`
  §4) ≈ Trail `--lsp` over gopls/pyright/typescript-language-server/
  rust-analyzer/clangd.

The learnable deltas are below.

## 3. What v2 should learn (ranked by value/effort)

### L1 — Generic breadth tier via `tags.scm` (biggest gap)

**v2 today** (`003` §3.1): 6 rich one-pass extractors (go/python/java/js/ts/
tsx) + **24 map-based extractors** (node-type → kind data tables) where
"calls are best-effort (often name-only → `unresolved_edges`)". Result: in
Rust/C++/Ruby/C#/Scala/etc. the graph has almost no call edges and
`unresolved_edges` stays permanently high — the "extractor is weak here"
signal never recovers.

**Trail's answer**: one **generic extractor** over any tree-sitter grammar +
its `tags.scm` (`@definition.*` + `@reference.call`) covering 16 long-tail
languages in a single code path (`src/graph/generic.ts`), with vendored
`.scm` overrides only where needed.

**Action**: replace the 24 map-based extractors with a `tags.scm`-driven
extractor. Same data-driven principle v2 already chose ("mapping as data,
not 1481 lines of code"), but the data source is the grammars' own
tree-sitter queries, which upstream grammars maintain. Keeps the self-healing
pool + regex floor as tiers below it. Expect a large share of today's
name-only edges to become real `references`/`calls` edges across the long
tail. (Note: generic tier gives bare-name references only — no receiver
typing; pair with L2.)

### L2 — Receiver binding pass (the difference between "calls" and "member calls")

**v2 today**: a statically-bound call = "the callee is a known symbol in
scope" (`003` §3.1); anything else is `to_id NULL`.

**Trail's answer** (`src/graph/bindings.ts`): per-language **binding pass** —
variable/field → type, `this`/`self`, Go receivers — feeding an
owner-qualified name index; walks the `extends` chain (depth ≤ 3); PHP
traits; Java/Swift **arity-based overload narrowing**.

**Action**: add a lightweight binding tier (at minimum `this`/`self` +
one-level receiver inference + owner-qualified lookup). This is the cheapest
item on the list and moves a big share of `unresolved_edges` into
`EXTRACTED`/`INFERRED` without an LSP. It is the piece that makes
`self.foo()` (Python) and `obj.Bar()` (Go) resolvable statically — the most
frequent failure mode of a no-LSP index.

### L3 — Probe-on-read freshness policy + byte-identical invariant

**v2 today** (`001` §4): a stale query "answers from the last index + sets
`degraded: stale_index`" (CI-INV-4).

**Trail's answer**: every retrieval call runs `ensureFreshGraph` first —
stat-walk probe, sha256-confirm suspects, and if dirty a **$0 structural
rebuild before answering**. Answers are never stale; they are sometimes
slower. The `degraded: stale_index` state does not exist in their contract.

**Action**: change the query-path policy to probe-on-read + cheap structural
rebuild when dirty (the stat-walk machinery in v2's spec is already
identical — only the policy changes). Consequences: (a) CI-INV-4 simplifies
— staleness becomes a latency property, not a result-quality state;
(b) add the pinned test **"incremental output is byte-identical to a cold
build"** — v2's target-state upserts (`002` §9) already make this
achievable, and it is the strongest guarantee that the incremental path
cannot drift from the cold path.

### L4 — `ast_hash` skip-the-write hook

**v2 today** (`001` §2): `ast_hash` (FNV-1a over extracted structure) is
"informational … a future skip-re-embed hook" — *not* part of the
incremental guard.

**Trail's answer**: the parse memo caches each file's parsed
`{nodes, rawEdges}` keyed by content hash; a re-extract that produces
identical structure is a no-op end-to-end (byte-identical files).

**Action**: use `ast_hash` to **skip the write too** — when re-extract
produces an unchanged structure (comment-only edit, whitespace change), do
not touch `chunks`/`symbols`/`edges`/FTS at all. This avoids FTS churn on
structure-preserving edits and makes the incremental path's write volume
proportional to real graph change. (Content-hash guard still forces the
re-extract; `ast_hash` just makes it a no-op write — the spec already
documents exactly this shape, it is unhooked.)

### L5 — Deterministic graph-rank leg (personalized PageRank)

**v2 today** (`006` §1): hybrid = FTS + structural + **semantic
(embeddings)**. There is no *structural* re-ranking signal; when the
embedder is down, "find the important code that mentions X" degrades to
plain BM25.

**Trail's answer**: `graft_find_code` blends BM25 with **personalized
PageRank** (α=0.25, 25 iterations, undirected walk over call/import edges,
seeded by the lexical scores) — $0, deterministic, and it is the leg that
keeps their "find a feature" useful without any model.

**Action**: add a PageRank leg over the existing `edges` table (either as a
4th RRF leg or as a re-weight inside the FTS leg). Seeds = lexical scores;
walk over `calls`/`imports`/`references`/`extends`/`implements` (mirror the
walker's kind lists, `004`). Degrades to nothing when the graph is empty —
no new degraded state. This is the missing structural signal in v2's
degradation floor.

### L6 — Greppable markdown cards (passive channel)

**v2 today**: export (Obsidian markdown) exists but is a manual, on-demand
export; orientation is MCP-only.

**Trail's answer**: the source tree is **mirrored as ~150-token wiring
cards** (`INDEX.md` + per-file `<path>.md`, re-admitted to ripgrep via root
`.ignore`), so `grep <symbol>` lands on a card, not a whole file, with no
MCP call at all.

**Action**: maintain a card surface in-repo (`.skillgrid/` — **not** the KB
vault, which is excluded from code indexing) generated by `code_index` from
`symbols` (signature + span per definition). Gives agents a zero-dependency
fallback orientation path when the MCP/daemon is down, and it is free for
`code_map` to emit. Keep cards derived (CI-INV-1: rebuildable from the
tree).

### L7 — Language-family reachability guard

**v2 today**: "unique-global name guess" → `AMBIGUOUS` edge, otherwise
dropped (`002` §6, `003` §5 route policy).

**Trail's answer**: cross-file name matching is restricted to the same
**language family** (TS/JS, JVM, C/C++) before the guess/drop decision.

**Action**: add a family filter before the unique-global guess. Cheap, and
it directly reduces false `AMBIGUOUS` edges in polyglot repos (e.g. a Go
`Serve` and a JS `Serve` should never be candidates for each other).

### L8 — `crux` (≤8-line business-logic core per symbol)

**v2 today**: `signature` + chunk text; no per-symbol "core" concept.

**Trail's answer**: `crux: {code, span}` — an LLM-chosen ≤8-line core,
inlined into search hits and MCP results (`body_text` ≤5000 chars
whitespace-collapsed lives in the ask-index sidecar).

**Action (optional)**: add a `crux` column to `symbols`. Two options:
(a) **deterministic** — first N non-comment lines of the body (keeps the
spec's "no LLM in the extract path"); (b) **LLM pass** — opt-in, mirrors
Trail's `--deep`. Option (a) is cheap and makes `code_read`/search results
denser; option (b) needs the same opt-in LLM seam the spec already has for
embeddings.

### Not-to-copy (deliberate)

- **Files-on-disk JSON instead of SQLite.** Trail's "no database, no
  vectors" bet trades away the semantic leg, the memory/code unification in
  one store, and PG duality (INV-2). v2's dual-dialect design is a feature
  Trail does not have.
- **BM25 token-bag sidecar (`ask-index.json`).** v2's FTS5 trigram /
  Postgres tsvector already do this natively.
- **32KB slice workaround.** A tree-sitter WASM limitation; gotreesitter
  (pure-Go binding) does not need it.
- **Worktree seeding / Claude Code Stop-hook build** — Trail is
  Claude-Code-specific glue; v2 has its own watcher + hooks story.

## 4. Ranked summary

| # | Item | v2 spec anchor | Value | Effort |
|---|---|---|---|---|
| 1 | `tags.scm` generic tier (replaces 24 map extractors) | `003` §3.1 | high | medium |
| 2 | Receiver binding pass (`this`/`self`/receiver → owner-qualified resolve) | `003` §3.1, `002` §6 | high | medium |
| 3 | Probe-on-read freshness policy + byte-identical-to-cold-build test | `001` §4, `002` §9 | medium | low |
| 4 | `ast_hash` skip-the-write hook | `001` §2 | medium | low |
| 5 | Personalized PageRank re-rank leg (over `edges`) | `006` §1 | medium | low–medium |
| 6 | Greppable markdown cards in `.skillgrid/` (not KB vault) | `008`/transport | medium | low |
| 7 | Language-family guard before AMBIGUOUS guess | `002` §6, `003` §5 | low–medium | low |
| 8 | `crux` per symbol (deterministic first, LLM opt-in) | `002` §2 | low–medium | low |
