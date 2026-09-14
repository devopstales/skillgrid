# based on skillgrid-v2:_shared/conventions/mnemonic-code-indexing.md

# Mnemonic Code-Indexing Convention

NOTE: This is the shared convention for the Mnemonic **code index** and retrieval surface (`code_*` MCP tools, `skillgrid search` / `export` / `serve`). It is supplementary — the entry point is the **`mnemonic` skill** (`.agents/skills/mnemonic/SKILL.md`, Step 5; former `mnemonic-code-index` skill now folded there). Sub-agents do NOT need to read the whole convention to follow the Orientation Ladder; read it when a tool returns an unexpected shape or you need CLI/operator detail.

Applies to every Skillgrid skill that explores, reads, or changes source code. The code index lives in the same per-project SQLite store as the memory layer and web cache (`~/.skillgrid/mnemonic/<project>.sqlite`) — same project scope, same session lifecycle as `mem_*`.

## Orientation Ladder (canonical order)

Primary repo navigation. Prefer this over listing whole directory trees into chat.

```
1. code_status      → health + Index Freshness (fresh|lag|empty|unknown)
2. code_map         → structural overview from the index (not a prose dump)
3. symbols / outline → code_search_symbols / code_outline (narrow the unit)
4. code_related     → neighbors via Edges
5. code_read        → exact slice for a path + range already narrowed
```

When freshness is `lag`, `empty`, or `unknown` (or compat `stale=true` for empty-only), run `code_index` before trusting search/map results. `stale=true` remains **empty-index compat only** — lag is `freshness=lag`, not silent "fine".

Rules:
- Do **not** dump the whole tree as primary orientation. Use the Orientation Ladder; the narrative Codebase Map under `.skillgrid/codebase/` is secondary context only.
- `code_read` is **only** for a path + line range already narrowed by map/symbols/related/search. Never read a whole file speculatively.
- Prefer index tools over `rg`/`grep` when exploring an **unfamiliar** large repo. Use `rg`/`grep` for exact-identifier lookups and when the index is a poor fit.

## Index Freshness

`code_status` reports:

| Field | Meaning |
|---|---|
| `freshness` | `fresh` \| `lag` \| `empty` \| `unknown` |
| `stale` | **compat:** `true` only when the index is empty — not a lag signal |
| `lagged_paths` | optional paths when `freshness=lag` |

- Unreadable mtime/hash → `unknown` (never false-clean).
- After clone, branch switch, large edits: check status; reindex when not `fresh`.

## Search Intent Router (daily path vs advanced)

Retrieval is **not** "six equal corpora" as the only model. The **Search Intent Router** picks a **daily path**; advanced corpora are an escape hatch (`skillgrid search --corpus …`).

| Intent signal | Daily path |
|---|---|
| Identifier-shaped query | `symbols` |
| Structural / `func $NAME` / Matcher dialect | `grep` |
| Decision / remember / past work | `mem` |
| Unclassified | `code` (`route=default`) |

Advanced modes (`--corpus`): `code`, `grep`, `mem`, `symbols`, `hybrid`, `semantic`. Multi-corpus results must keep **provenance** (mem vs code never silently fused unlabeled).

### Explicit semantic degrade

The `semantic` corpus must never pretend full quality when the Local Code Embedder is unavailable. Expect `degraded=true` plus reason + fallback. **Never silent** degrade to FTS-as-semantic.

## Extractors and tree-sitter Python

- Default Go/TS extractors stay CGo-free.
- Python tree-sitter Extractor Adapter compiles only with **`-tags treesitter`**. Without the tag, Python uses stub/regex fallback (`warn+continue`).
- Tree-sitter/CGo stay **inside** Extractor Adapters — Matchers and MCP handlers remain CGo-free at the process edge where feasible.

## Matcher dialect v2

Structural `code_grep` / `--corpus grep` patterns (invalid pattern → abort):

- v1: `func $NAME($ARGS)`, `$FUNC($ARGS)`, `class $NAME`, `type $NAME`, `interface $NAME`
- v2: `struct $NAME`, `method $NAME($ARGS)`, `const $NAME`, `enum $NAME`

## Impact Analysis

`code_impact` is **additive** blast-radius over Edges (callers, callees, dependents, … with Confidence Labels). It does **not** replace `code_get_*`.

## Memory export and serve visualization

- `skillgrid export --project ID --out DIR` — read-only Obsidian Markdown + viz JSON under an allowed root (path outside root → abort).
- **Memory Visualization** lives in the **`skillgrid serve`** dashboard webui (read-only observation browser). Viz clients must not mutate/delete observations.
- **Code Graph dashboard** (CodeGraph-shaped, same serve UI): 3-pane callers | source | callees, blast-radius, symbol search, entry points, flow/path — read-only over Edges (`/graph/*`; mutate → 405).

## Tool quick-reference

| tool | role |
|---|---|
| `code_status` | Index Freshness + health |
| `code_index` | Incremental index when not fresh |
| `code_map` | Orientation overview |
| `code_outline` / `code_related` | Symbol outline / neighbors |
| `code_search` / `code_search_symbols` | Chunk / symbol FTS |
| `code_grep` | Structural Matcher (dialect v1+v2) |
| `code_impact` | Additive Impact Analysis |
| `code_read` | Narrowed slice |
| `code_get_*` | Symbol/signature getters (unchanged) |

## Configuration

Config file `config.d/indexing.yaml`, searched up from the indexed dir; repo-local and `~/.skillgrid/config.d/indexing.yaml` override defaults.

| field | default |
|---|---|
| `chunk_lines` | `80` |
| `chunk_overlap` | `10` |
| `max_file_size_kb` | `512` (hard cap; larger files skipped) |

## Recording findings

After indexing or searching, `mem_save` what you learned following [memory.md](memory.md).

## CLI fallback

```bash
skillgrid index [--dir <path>] [--project <id>]
skillgrid search <query>                 # Search Intent Router daily path
skillgrid search --corpus semantic …   # advanced; watch degraded flag
skillgrid export --project ID --out DIR
skillgrid serve                          # dashboard: Memory Visualization + Code Graph
```

## Gotchas

- `code_index` indexes the **git root**, not the cwd.
- Semantic path without embedder → **explicit** `degraded` metadata, never silent.
- `stale=true` ≠ lag; use `freshness`.
- Codebase Map prose under `.skillgrid/codebase/` does **not** replace the Orientation Ladder.
- The index is **per-project** (one SQLite file). No cross-project search.

## Why This Convention

- Orientation Ladder first → agents stop tree-dumping as primary nav.
- Search Intent Router → daily path vs advanced corpora, not flat six-peer-only UX.
- Honest freshness + explicit semantic degrade → agents reindex and trust labels.
- Pointers, not copies → heavy schemas stay in the `mnemonic` skill + this convention.
