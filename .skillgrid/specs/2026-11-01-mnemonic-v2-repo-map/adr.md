# ADR Review — mnemonic-v2 repo-map

> In-force ADR review + the new ADR for this change. The in-force set comes from the
> `## LOCKED` ADR index in `.skillgrid/ASSUMPTIONS.md`. A blueprint that contradicts an
> in-force ADR must either follow it or record a new superseding ADR.

## In-force ADRs this change relies on (substrate)

| ADR | Decision | How this change uses it |
|---|---|---|
| ADR-0031 | Go 1.22+ minimum to build | `mnemonic-v2/go.mod` declares `go 1.26.0` (the go.work floor); the repo-map adds no new Go-version requirement. |
| ADR-0040 | mnemonic-v2 module extraction | The repo-map lives in the same `mnemonic-v2/` module the memory + code-index changes created; no new module. The new domain package is `internal/repomap/`. |
| ADR-0041 | Postgres driver (`lib/pq`) | The repo-map's reads are dialect-agnostic (the `Store` seams); the Postgres dialect re-uses the same seams. No new driver. |
| ADR-0008 | The `yaml` package dependency | The repo-map's config (`hub_threshold`, `inject_tokens`, `repo_map.commit`, the auto-inject flag) re-uses the memory change's `yaml.v3` config load; no new config dep. |
| ADR-0045 (code-index) | The `Store` interface extension (code-index methods) | The repo-map **reads** the code-index's `Store` methods + seams but **does not extend** the `Store` (ADR-0047 below). The ADR-0045 one-way door is not touched. |
| ADR-0043/0044/0046 (code-index) | tree-sitter / fsnotify / onnxer deps | Substrate only — the repo-map adds no new dependency (the PageRank is hand-rolled). |

> **Note on ADR numbering:** the in-force index's highest sequence is 0042 (the goldfish
> plan review). The code-index change (queued, not yet shipped) claims 0043–0046 for its
> tree-sitter / fsnotify / onnxer / Store-extension decisions. This change's new ADR
> starts at **0047**, confirmed against ASSUMPTIONS.md § LOCKED at T0 (after the
> code-index change's ADRs land). If the code-index change's claim shifts, renumber —
> the decision content is unchanged.

## New ADR introduced by this change (0047)

### ADR-0047 — the repo-map read surface is a view, not a store extension

- **Status:** proposed → in-force at T0
- **Context:** the repo-map (FUNCTION 3) is a read-only composition over two existing
  legs: the code-index graph (`files`/`symbols`/`edges`/`index_freshness` + the
  `internal/codeindex/graph` package) and the memory store (`observations`/
  `observations_fts`/`sessions`/`session_trails`). The design question is whether the
  repo-map needs its own storage surface (a cache table, a `Store` extension) or whether
  it can read through what already exists.
- **Decision:** the repo-map read surface is a **view, not a store extension** — **no
  new tables, no new migration, no `Store` interface method**. The read goes through the
  existing `RawQuery`/`Probe`/`ProbeInt`/`RawExec` seams (the memory change's
  `SQLiteStore` read surface) + the existing `internal/codeindex/graph` methods
  (`Orient`/`Impact`/`Callees`/`Callers`). The only writes are the retrieval-trail row
  (INV-5, the same seam the memory change's `trails.go` uses) and the emitted doc
  (a file, not a table).
- **Alternatives:**
  - (a) **a `repo_map_nodes` cache table with invalidation** — rejected: a second cache
    adds a second source of truth + an invalidation bug class the index already owns.
    The code-index **is** the cache (the openclaude fold: "we do NOT need a second
    cache — the code-index IS the cache"). A cache table would need to track both the
    code-index freshness AND the memory schema version to stay valid — the invalidation
    surface is the whole repo-map's read surface, for no gain over a pure read.
  - (b) **extending the `Store` interface with repo-map methods** — rejected: the
    `Store` is a contract three functions share (memory, code-index, and now repo-map);
    reshaping it for a read the seams already serve touches the ADR-0045 one-way door.
    The `RawQuery`/`Probe`/`ProbeInt` seams are a general read surface (the memory
    change's conflict pre-filter + the replay ledger already use them); the repo-map's
    reads are exactly their shape (multi-row multi-column reads → `RawQuery`; single
    value → `Probe`/`ProbeInt`).
- **Consequences:** the repo-map is a **pure read** over the two legs — it cannot
  corrupt either leg, and a broken leg degrades out + is labeled (the coverage
  contract) without poisoning the other (T-CI-2 posture). The cost is that the repo-map
  reads are a few SQL queries per call (the PageRank builds an in-memory adjacency from
  one `edges` read) — but the re-rank is <100ms (no re-parse) and the cold build is the
  `code_index` cost (already a code-index acceptance). **If a future read is too slow
  for the seams** (a repo-map-specific aggregation the `RawQuery` shape cannot carry),
  the escape is a **`Store` method** (additive, the ADR-0045 pattern) — not a cache
  table. That escape is a new ADR (a supersession of this one's "no `Store` method"
  clause), recorded when it happens.

## Supersessions

None. The repo-map supersedes no in-force ADR. It is a consumer of the code-index
substrate (ADR-0043…0046) + the memory substrate (the `Store` + the seams), and it adds
one ADR (0047) that constrains its own read surface.
