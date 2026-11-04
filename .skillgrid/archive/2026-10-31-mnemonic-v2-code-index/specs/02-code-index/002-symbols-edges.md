# 002 — Symbols + Edges (the clean graph schema)

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** `000-overview` (CI-2 single node table, CI-3 sync_id, CI-4 no temporal
window), `001-files-chunks` (`files.id`), memory `00-foundation/002-schema` §4 (the
declared table set).
**Consumed by:** `004-graph-queries` (the walker reads `symbols` + `edges`),
`003-extraction` (writes them), `006-search` (`symbol_fts`), the KB `006-validate`
(`code_resolve` reads `symbols` + `edges`).

---

## 1. Capability

`symbols` is the **single node table** of the graph (CI-2). Every node — a code symbol
(function/method/class/struct/interface), a route, a doc, a config, a SQL table — is a
row in `symbols` with a `kind`. `edges` is the relationship table. This is the v2
rebuild of v1's worst wart: v1's `from_id`/`to_id` pointed at **four** tables (symbols /
doc_nodes / config_nodes / sql_schema_nodes), which forced an FK-drop migration and made
every graph query a union of id spaces. v2 has **one id space**.

## 2. The `symbols` table

```sql
CREATE TABLE symbols (
    id             INTEGER PRIMARY KEY,          -- local (not cross-machine)
    sync_id        TEXT NOT NULL,               -- INV-6 cross-machine identity (ULID)
    file_id        INTEGER NOT NULL,
    name           TEXT NOT NULL,               -- the local name (e.g. capturePayment)
    qualified_name TEXT,                        -- e.g. pkg.CapturePayment (disambiguation key)
    kind           TEXT NOT NULL,               -- the node kind (§3)
    language       TEXT,
    signature      TEXT,                        -- the function/method signature (semantic tier input)
    start_line     INTEGER NOT NULL,
    end_line       INTEGER NOT NULL,
    visibility     TEXT,                        -- public/internal/private (derived, CI below)
    is_exported    INTEGER NOT NULL DEFAULT 0,
    return_type    TEXT,                        -- derived (Go/TS heuristics, not type inference)
    param_types    TEXT,                        -- derived
    content_hash   TEXT NOT NULL,               -- sha256 of the symbol span
    uid            TEXT NOT NULL UNIQUE,        -- local upsert key (stable per file+name+kind)
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);
CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_symbols_qualified ON symbols(qualified_name);
```

**Identity (CI-3):**
- **`uid`** is the **local upsert key** — stable per (file, name, kind, position). It is
  how the extract tx upserts (`ON CONFLICT(uid)`) and how edges reference symbols
  *within* a file at write time.
- **`sync_id`** is the **cross-machine identity** (INV-6) — the ULID the agent/KB sees
  where identity matters. `code_resolve` returns it; the KB's `symbol:` anchor can pin
  on it. The local `id`/`uid` are ephemeral.
- **Disambiguation:** `qualified_name` is the primary disambiguation key. Two symbols
  with the same `name` in different packages/files are distinct `qualified_name`s. The
  KB's `code_resolve` resolves a bare `symbol: Name` by name → if ambiguous (multiple
  `qualified_name`s), it returns the **ranked candidates** (never a silent pick,
  CI-INV-3).

**Derived fields** (`visibility`, `is_exported`, `return_type`, `param_types`):
derived from the source span at extraction (tree-sitter gives name/kind/range; these
are Go/TS heuristics, not full type inference). They are **informational** — the
semantic tier + the `code_orient` read use them; they are not the graph's identity.

**`content_hash`** is the hash of the symbol's source span. It is the change signal for
incremental re-embed (a symbol whose `content_hash` changed is re-embedded; one that did
not is skipped).

## 3. The `kind` vocabulary (all nodes are `symbols` rows)

| `kind` | What it is | Source |
|---|---|---|
| `function` | a top-level function | tree-sitter |
| `method` | a method (receiver-qualified) | tree-sitter |
| `class` | a class/type | tree-sitter |
| `struct` | a struct | tree-sitter |
| `interface` | an interface (a **dispatch kind** — flows stop here, `004` §graph-stops) | tree-sitter |
| `route` | a framework route (path pattern) | the route extractor (`003` §knowledge) |
| `doc` | a markdown doc (title) | the knowledge extractor |
| `config` | a config key (yaml/toml/json) | the knowledge extractor |
| `table` | a SQL table | the SQL extractor |
| `column` | a SQL column | the SQL extractor |

**The knowledge nodes (route/doc/config/table/column) are `symbols` rows, not separate
tables** (the v1 rebuild). Their edges live in `edges` (kinds `references`,
`configures`, `reads`, `writes`, `navigates` — §4). This is what removes v1's
polymorphic-id wart: a `route` node and a `function` node are the same table, so an edge
from a `route` to a `function` is a normal `edges` row with two `symbols.id` endpoints.

## 4. The `edges` table

```sql
CREATE TABLE edges (
    id          INTEGER PRIMARY KEY,
    kind        TEXT NOT NULL,                 -- the edge kind (§5)
    from_id     INTEGER NOT NULL,              -- symbols.id (single id space)
    to_id       INTEGER,                       -- symbols.id, or NULL = name-only (unresolved)
    to_name     TEXT,                          -- the literal target name (always set for name-only)
    file_id     INTEGER NOT NULL,              -- the owner file (target-state prune key)
    confidence  TEXT NOT NULL DEFAULT 'EXTRACTED',  -- §6
    line        INTEGER,
    context     TEXT NOT NULL DEFAULT '',      -- e.g. 'route', 'doc_mention'
    UNIQUE(kind, from_id, file_id, to_id, to_name, line)
);
CREATE INDEX idx_edges_from ON edges(from_id);
CREATE INDEX idx_edges_to ON edges(to_id) WHERE to_id IS NOT NULL;
CREATE INDEX idx_edges_to_name ON edges(to_name) WHERE to_id IS NULL;
```

**No temporal window** (CI-4): v1's `valid_from`/`valid_to` are gone. An edge is in the
graph or it is not (CI-INV-2).

**The target-state prune key is `file_id`** (the v1 mechanics, kept): on a re-extract of
a file, `DELETE FROM edges WHERE file_id = ?` then re-upsert the file's fresh edge set.
This makes the write idempotent (a re-extract of an unchanged file is a no-op).

**`to_id NULL + to_name` is a deliberate state** (CI-INV-3, drop-not-guess): an edge the
extractor saw but could not resolve (the target symbol does not exist, or the resolution
was ambiguous and *dropped*). A name-only edge is **never a fabricated resolved edge** —
it is the explicit "we saw this, we could not bind it" record. The walker
(`004-graph-queries`) handles name-only edges by matching `to_name` against `symbols.name`
*at query time* (an edge that was name-only at write time may become resolvable after a
re-extract adds the target).

## 5. The edge-kind vocabulary

| `kind` | Meaning | Confidence range |
|---|---|---|
| `calls` | call site → callee | EXTRACTED / AMBIGUOUS |
| `imports` | file/module import (`to_name` = the imported path/symbol) | EXTRACTED |
| `extends` | heritage (class extends) | EXTRACTED |
| `implements` | heritage (implements an interface) | EXTRACTED |
| `references` | a typed reference (route→handler, doc→doc, config→doc) | EXTRACTED / INFERRED / AMBIGUOUS |
| `configures` | a config key → the code symbol it configures | INFERRED / AMBIGUOUS |
| `navigates` | screen→screen navigation (`to_name` = route path, `to_id` often NULL) | INFERRED |
| `reads` | code symbol → SQL table (a SELECT) | EXTRACTED |
| `writes` | code symbol → SQL table (an INSERT/UPDATE/DELETE) | EXTRACTED |
| `tests_for` | a test symbol → the code it tests | INFERRED |
| `tested_by` | code symbol → its test (the inverse of `tests_for`) | INFERRED |

**The walker's kind lists** (`004-graph-queries`):
- **caller/dependent kinds** (reverse BFS for impact): `calls`, `imports`, `references`,
  `extends`, `implements`, `navigates`, `tests_for`, `tested_by`.
- **hierarchy kinds**: `extends`, `implements`.
- **test kinds**: `tests_for`, `tested_by`.

**Import cycles** are **not a stored edge kind** (v1's `import_cycles` table is dropped,
`000-overview` §drop) — a cycle is re-derived from `kind='imports'` edges on demand by
the `code_circles` query (admin, `008-transport`).

## 6. Confidence (3-tier default, 4th when LSP is on)

| Label | Score | Meaning | Set when |
|---|---|---|---|
| `EXTRACTED` | 1.0 | explicit syntax / fully resolved | a call site statically bound; an import; a heritage edge; a same-file route→handler |
| `INFERRED` | 0.85 | convention-derived | a wikilink in a doc; a conventional config ref; a `tests_for` by naming convention; a `navigates` from markup |
| `AMBIGUOUS` | 0.5 | resolved-by-name-only guess, or an unresolved call boundary | a unique-global name guess for a route→handler; a name-only edge the walker matched at query time |
| `LSP_RESOLVED` | 0.95 | resolved by a language server | **only when the LSP tier is on** (`--lsp`, CI-6). A member call site (`receiver.Callee`) the static pass could not bind, bound by gopls/pyright/etc. |

**The 3 tiers are the default** (no LSP). `LSP_RESOLVED` is a 4th value that only
appears when the LSP tier is active. The walker's confidence filtering
(`code_impact` `min_confidence`) ranks `EXTRACTED` > `LSP_RESOLVED` > `INFERRED` >
`AMBIGUOUS`.

**Where it is set:** the extractor assigns it per edge (`003-extraction`); the route
resolution policy (same-file → EXTRACTED, unique-global-name → AMBIGUOUS, else
*dropped* with a name-only edge + an `unresolved_edges` row) is in `003-extraction`
§knowledge; the LSP tier writes `LSP_RESOLVED` (`005-advanced-passes` §LSP).

## 7. The `dep_edges` table (the SBOM dependency graph)

The SBOM (CycloneDX) dependency graph is kept **thin** (CI-O6, in the declared schema).
It is a **separate** small graph (purl-keyed, not symbol-keyed) — the only non-symbol
graph in v2:

```sql
CREATE TABLE dependencies (
    purl       TEXT PRIMARY KEY,           -- the package URL (the identity)
    name       TEXT NOT NULL,
    version    TEXT NOT NULL,
    ecosystem  TEXT NOT NULL,              -- go / npm / pypi / ...
    manifest   TEXT,                       -- the manifest file it came from
    retired    INTEGER NOT NULL DEFAULT 0,
    last_seen  TEXT NOT NULL
);
CREATE TABLE dep_edges (
    from_purl    TEXT NOT NULL,
    to_purl      TEXT NOT NULL,
    version_range TEXT,
    PRIMARY KEY (from_purl, to_purl),
    FOREIGN KEY (from_purl) REFERENCES dependencies(purl) ON DELETE CASCADE,
    FOREIGN KEY (to_purl)   REFERENCES dependencies(purl)   ON DELETE CASCADE
);
```

**The SBOM pass** parses the project's SBOM (if present) and upserts `dependencies` +
`dep_edges`. It is an **optional pass** (run when an SBOM is present; absent SBOM → the
tables are empty, `code_deps` reports "no SBOM"). It is **not** part of the core graph
tx (it does not depend on `files`/`symbols`). `code_deps` / `code_dep_impact` (admin,
`008-transport`) read it.

## 8. The symbol FTS (the search leg)

`symbol_fts` is an external-content FTS over `(name, qualified_name, signature, kind,
language)` — the symbol-level leg of `code_search` (`006-search`). It is **derived**
(CI-INV-1): rebuilt from `symbols` by `code_index`. (The dual-mode rendering is the same
as `chunks_fts`, `001` §3.)

## 9. The write path (target-state, idempotent)

The extract tx (`003-extraction`) writes `symbols` + `edges` **target-state** per file:
upsert `symbols` `ON CONFLICT(uid)`, then `DELETE FROM edges WHERE file_id = ?` +
re-upsert the file's fresh edges. A re-extract of an unchanged file (same
`content_hash`) produces an identical graph (the upsert is idempotent). A deleted file
cascades: its `symbols` rows delete → their `edges` delete (the `ON DELETE CASCADE`),
and its `unresolved_edges` delete (`001` §5).

**Orphan prune:** after a run, any `symbols` row whose `uid` is not in the declared
target set (the new files' + the skipped files' UIDs) is deleted. This is the
"the file changed, so its old symbols that no longer exist are gone" cleanup.

## 10. Invariants

INV-1 (provenance: `files`/`symbols` carry origin), INV-2 (dual-mode — every table both
dialects), INV-3 (the `degraded` envelope on a degraded extract), INV-6 (`sync_id`).
CI-INV-1 (derived — `symbols`/`edges`/`symbol_fts` rebuildable from the tree),
CI-INV-2 (current-state), CI-INV-3 (drop-not-guess — name-only edges, never fabricated).

## 11. Open items

- **CI-O3 (`code_resolve` return shape):** the read returns `called_from` as **file
  paths** (the KB anchor is `code:<path>`; `sync_id` is internal). Confirmed in
  `004-graph-queries` §`code_resolve`.
- **The `qualified_name` for non-Go languages:** for languages without a package
  concept (Python), `qualified_name` is the module path + name (best-effort). A symbol
  whose `qualified_name` is NULL disambiguates by (file, name). Not a spec constant —
  an extraction detail (`003-extraction`).
