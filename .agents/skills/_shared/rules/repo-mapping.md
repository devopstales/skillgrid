# Repo Mapping — the architecture/ map of the repo

The recipe for producing and refreshing the `projectwiki` `architecture/`
pages that map a repository subtree. Home: `architecture/` (projectwiki
page type `Architecture`). Consumed by `execute` Step 5 (scoped refresh,
automatic) and by `projectwiki` Maintain (full map, on demand).

Scouts run under [rules/scout-contract.md](scout-contract.md), the `scout`
persona (`.agents/agents/scout.md`). Multi-scout runs are gated by
[planning/fan-out-gate.md](../planning/fan-out-gate.md).
The deterministic scope computation is
[scripts/map-scope.mjs](../scripts/map-scope.mjs) — the agent decides
*whether* to map; the script owns *what* to map.

## Page shapes (wiki, not READMEs)

- **Component page** — a directory with direct source code →
  `architecture/<dir-path>.md`, frontmatter `type: Architecture`,
  `sources: [code:<path>]`:
  - `## Interface` — public symbols, each `[file:symbol]` cited
  - `## Invariants` — each cited; `None` if absent (a valid value)
  - `## Side Effects` — each cited; `None` if absent
  - `## Verification` — test command, valid target syntax per build system
- **Blueprint page** — a directory with no direct source but with mapped
  children → same type:
  - `## Subdirectories & Boundaries` — one line per child:
    `[[architecture/<child>]]` + one-sentence boundary summary
  - `## Global Invariants` / `## Global Side Effects` — cross-child
    constraints only
- Roll up bottom-up: leaves first, parents from children's compiled JSON,
  root last (root is always a blueprint; root-level source goes in an extra
  `## Local Interface`).

## Refresh (the execute hook — scoped)

Triggered by `execute` Step 5 whenever the run changed code:

1. `node .agents/skills/_shared/scripts/map-scope.mjs --base <MERGE_BASE> --mode refresh`
   → JSON: `affected` dirs, `pages[]` in post-order, `gate`.
2. `gate: "ask"` → apply the fan-out gate: state the cost verbatim
   ("N scout units, cap 5 concurrent, ~1,500 source lines per scout") and
   wait. The user may answer **defer** → record `map: deferred (<N> units)`
   in the execution ledger and continue; the next run's refresh folds in the
   deferred dirs.
3. Re-scout each `component` page in `pages[]` (scout contract: ~1,500-line
   chunks, ≤5 concurrent, post-order). A directory already in the manifest
   gets the correction policy: read existing page → scout fresh → merge
   extracted symbols with scout symbols → edit in place, keep `## Overview`
   and human notes.
4. Re-render each `blueprint` page from the (new + cached) child JSON.
5. `node map-scope.mjs --merge scout-returns.json` → manifest `head` moves
   to HEAD.
6. `projectwiki index` + one `log.md` line (`mapping | <topic> | N pages`)
   — the projectwiki same-edit rule.

A failed scout leaves its page untouched and one line
`> ⚠️ mapping failed <date> — section pending` in the page; the manifest
keeps the old `head` for that dir, so the next refresh re-scouts it.
Partial state is a feature.

## Full map (cold start — projectwiki on demand)

1. `map-scope.mjs --mode full` (no base needed on a fresh manifest; for a
   populated manifest, pass `--base <oldest mapped commit>`).
2. Same steps 2–6. Partitioning is the script's; fan-out per partition,
   gated.

## What a map is NOT

- Not a test suite. No assertions live in these pages; verification
  commands point at the real ones.
- Not the source of truth for behavior. Pages summarize; the code is the
  truth. A claim a scout cannot cite with `[file:symbol]` does not go in.
- Not per-file documentation. One page per directory; file-level detail
  stays in the code and in `invariants/` claims.
