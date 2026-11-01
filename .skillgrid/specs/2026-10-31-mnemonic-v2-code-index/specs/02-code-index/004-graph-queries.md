# 004 — Graph Queries (the read API + the two KB contracts)

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** `002-symbols-edges` (the tables it reads), `001-files-chunks`
(`index_freshness` — the staleness check before trusting the graph).
**Consumed by:** the KB `006-validate` (`code_resolve` — the `symbol:`-anchor
contract), the KB `007-lint` (the package map — the coverage-gap contract),
`008-transport` (the `code_*` tools that call these), `006-search` (the structural
leg).

---

## 1. Capability

The graph read API is the set of queries over `symbols` + `edges`. It is the **single
most important surface** of the code-index: the KB's pin-check (`symbol:` anchor), the
KB's module map, and the agent's "what does this change break?" all resolve to these
queries. Every read:
1. **Checks freshness** (`index_freshness`, `001` §4) — a stale graph answers from the
   last index + sets `degraded: stale_index` (CI-INV-4).
2. **Records a trail** (INV-5) — query, route, what was returned.
3. **Honors confidence** — the walker filters by `min_confidence` (the 3/4-tier
   ranking, `002` §6); excluded hops are *counted*, not silently dropped.

## 2. Neighbor views (the orientation)

All are **confidence-labeled** edge views (the v1 `graph.Neighbors`, kept). Each returns
the edges of the requested kind(s) for a symbol, with the endpoints resolved to
`symbols` (name-only edges resolve `to_name` → `symbols.name` at query time; an
ambiguous name-only edge returns `AMBIGUOUS` edges, never a silent drop — CI-INV-3).

| View | Kinds | The answer |
|---|---|---|
| `code_get_callers(symbol)` | reverse `calls` + `imports` + `references` | who calls/uses this (the **pin-check call-site check**, §3) |
| `code_get_callees(symbol)` | forward `calls` + `imports` | what this calls |
| `code_get_dependents(symbol)` | reverse (all dependent kinds) | the transitive "what breaks if this changes" (one hop) |
| `code_get_implementors(symbol)` | reverse `implements` | who implements this interface |
| `code_get_hierarchy(symbol)` | `extends` + `implements` (both directions) | the class/interface hierarchy |
| `code_get_tests_for(symbol)` | `tests_for` / `tested_by` | the tests that cover this |

**Each returns:** the edge (`kind`, `confidence`, `line`, `via` symbol), the target
symbol (`name`, `qualified_name`, `file`, `sync_id`), and — for the pin-check — whether
the call site is in a specific file (the `code_resolve` `called_from`, §3).

## 3. `code_resolve` (the KB `symbol:`-anchor contract) — the load-bearing read

This is the **one tool the KB's `symbol:` anchor calls** (`006-validate` §2,
`001-structure` §5.2). The KB's pin-check needs to know, for a `symbol: Name` anchor
(+ an optional `called_from` file hint): **does the symbol still exist? is it still
called from where the page says?**

```
code_resolve(symbol, called_from?) →
{
  "exists": true,
  "name": "capturePayment",
  "qualified_name": "internal/pay.CapturePayment",
  "file": "internal/pay/pay.go",
  "line": 480,
  "sync_id": "sym-01J…",
  "called_from": [ "internal/billing/enterprise.go" ],   // the files that call it (forward callers)
  "ambiguous": false,
  "candidates": []                                        // set only when ambiguous
}
```

**Semantics:**
- **`exists: false`** → the KB's `symbol:` check is **`unresolved`** (the pinned
  construct may have been deleted/renamed — fail closed, `006-validate` §3).
- **`exists: true` + `called_from` empty** (the symbol exists but nothing calls it) →
  the KB's check is **`stale`** if the page pinned a call-site (the call is gone) or
  **`ok`** if the page pinned only existence.
- **`exists: true` + `called_from` non-empty** → **`ok`** (the construct still exists
  and is still called). If the page's `called_from` hint is a *specific file* and that
  file is **not** in the returned `called_from` list → **`stale`** (the call moved away
  from where the page says).
- **`ambiguous: true`** (multiple `qualified_name`s match the bare `name`) → the result
  carries **`candidates[]`** (the ranked disambiguation, `002` §2). The KB's check is
  **`unresolved`** (fail closed — the pin is ambiguous, never a silent pick).

**The `called_from` is a list of file paths** (CI-O3 — the KB anchor is `code:<path>`;
`sync_id` is internal). It is computed as the **forward callers** of the symbol (the
`from_id`s of reverse-`calls`/`references` edges, resolved to their `files.path`).
**`called_from` is best-effort:** without the LSP tier, a member call site
(`receiver.Callee`) that the static pass could not bind is *not* in `called_from`
(`degraded: no_lsp` if the check needs it) — the static resolution is the floor (CI-6,
CI-7).

**This is the KB contract made checkable:** `006-validate`'s `symbol:` row ("`ok` if
present (+ call-site for call-anchors); `unresolved` if not in the graph") resolves
exactly to this tool's `exists` + `called_from` + `ambiguous` fields.

## 4. The package map (the KB coverage-gap contract)

The KB's `007-lint` coverage-gap check reads the **top-level Go packages** and compares
to the set of pages that cite each. This read is:

```
code_packages() →
[
  { "path": "internal/memory", "files": 12, "symbols": 87 },
  { "path": "internal/codeindex", "files": 9, "symbols": 64 },
  ...
]
```

**Semantics:**
- **The unit is the top-level Go package** (CI: `007-lint` §3, confirmed). A top-level
  package is the first path segment under the project root that contains Go files
  (e.g. `internal/memory`, `cmd/mnemonic`). Sub-packages and files are *not* the unit.
- **It is derived from `files` + `symbols`** (the `files.language = 'go'` rows, grouped
  by top-level package path). **It does not require the graph to be "built"** in the
  LSP/PDG sense — it needs only the core graph (the `files` table). This is why the KB's
  coverage-gap check has a **no-graph fallback** (`007-lint` §3): if the code-index is
  absent, the KB does a shallow `go list ./...`-style package scan *itself*. The
  `code_packages` read is the *preferred* path (it uses the indexed `files`), the
  `go list` scan is the *fallback* (the code-index is an optional dependency, CI-7).
- **The KB compares** this list to the pages that cite each package (`code:<pkg>/…` in
  `sources:` or `anchors:`). A top-level package with **no page** is a coverage gap
  (advisory, not blocking — `007-lint`).

## 5. Impact (blast radius) — `code_impact`

The reverse-BFS "what breaks if this changes?" (the v1 `graph.Impact`, kept). From a
single symbol:
- **Honor `min_confidence`** (rank EXTRACTED > LSP_RESOLVED > INFERRED > AMBIGUOUS;
  excluded hops *counted*, not dropped).
- **Cap depth** (default 5, `max_depth` config).
- **Risk-tier by depth:** depth-1 direct dependents → **`WILL BREAK`**; deeper →
  **`LIKELY AFFECTED`**.
- **Traverses only stored edges** (a dropped reference — a name-only edge that
  resolved to nothing — has no row, so ambiguity can never inflate the radius,
  CI-INV-3).
- **Returns:** the two sorted edge lists (WILL BREAK / LIKELY AFFECTED) + the excluded
  count + a `summary()` (the tier counts + the direct will-break names, used by
  `code_explain`/`code_explore`).

## 6. `code_affected` (the PR question: "which tests do I run?")

The v1 `affected.Affected`, kept. Answers "which tests do I run after this change?":
- **Inputs** (mutually exclusive): `changed[]` (explicit file list) | `stdin` (a
  `git diff --name-only` stream) | `base` (a git ref → `git merge-base <ref> HEAD` +
  diff + git-history owners).
- **Seeds** = every symbol defined in the changed files.
- **Depth-capped reverse BFS** (default 5) over the dependent kinds (`002` §5). Every hop
  lands on a `symbols` row; hops landing on a **test path** (the language's test-file
  conventions, `003` §8) are collected as **affected test files**.
- **Returns:** the affected test files + the relationship chain (from, to, kind,
  confidence, depth, via) + the git-history owners (for the `base` mode).
- **Traverses only stored edges** (the drop-not-guess property, as in `code_impact`).

## 7. Path + "where the graph stops" — `code_path`

The v1 `graph.Path`, kept. BFS shortest path between two symbols (confidence ignored
for connectivity). **If no path exists**, the answer is **not empty and not
fabricated** — `buildGraphStops` inspects the *visited frontier*: every forward edge
whose target is unresolved (`to_id NULL`) is a **refused hop**. The answer reports:
- the **dispatch kind** that ended the flow (an `interface`/`bus`/`callback` — the
  flow stops at a dispatch boundary, `002` §3),
- the **line** + the **refused name-only matches** with their confidence.

A `PathResult` is always *something* (a path, or a "the graph stops here because
…" answer). This is the "where the graph stops" answer (the v1 crown feature, kept).

## 8. Explain — `code_explain` / `code_orient`

- **`code_explain(symbol)`:** the node + its degree + **all connections ranked by the
  neighbor's degree** (hubs first, the v1 `graph.Explain`). The "what is this symbol,
  and who is it most connected to?" answer.
- **`code_orient(symbol)`:** the symbol metadata + signature + **the file TOC** (every
  symbol in the containing file, line order) + the symbol list. The "where am I in this
  file?" orientation (the v1 `code_orient`/`code_signature`/`code_file_toc` combined).

## 9. Fair coverage (the health signal)

**`coverage_fair`** (stored in `index_freshness`, `001` §4): per language, the fraction
of symbol-bearing files that have **≥ 1 resolved cross-file dependent** (a symbol in
the file is the `to_id` of an edge whose `from` is in *another* file). Measured from
`edges`, not asserted. Reported by `code_status` (`008-transport`). A low fair coverage
is the "the graph is under-connected here (the extractor is weak for this language)"
signal.

## 10. The read conventions (shared with the KB)

- **Staleness check first** (every read): `index_freshness` → fresh (proceed) or stale
  (answer from last index + `degraded: stale_index`, CI-INV-4).
- **Trail recorded** (INV-5): query, route, what was returned.
- **Confidence honored** (the `min_confidence` filter; excluded hops counted).
- **No-graph degradation** (CI-7): the KB calls these reads; if the code-index is
  *absent* (not just stale), the KB's `symbol:` check degrades to a **ripgrep
  occurrence check** (`degraded: no_graph`) and the coverage-gap check falls back to a
  `go list` scan. The code-index being absent is a *specified* state, not an error.

## 11. Invariants

INV-2 (dual-mode — the walker is dialect-agnostic), INV-3 (`degraded` on a stale graph /
no-LSP / no-graph read), INV-5 (trails), CI-INV-2 (current-state — the walker has no
temporal filter), CI-INV-3 (drop-not-guess — name-only edges are explicit, never
fabricated; a `PathResult` is never empty/fabricated).

## 12. Open items

- **CI-O3 (`code_resolve` `called_from` shape):** file paths (not `sync_id`s). Confirmed.
- **The `code_resolve` ambiguity threshold:** a bare `name` matching > 1 `qualified_name`
  is ambiguous (return `candidates[]`). A `qualified_name` (or a `name` + a file hint
  that disambiguates to 1) is not ambiguous. This is the disambiguation rule (`002` §2).
- **`code_affected` depth default:** 5 (the v1 default). Configurable.
