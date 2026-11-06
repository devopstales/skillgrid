# 007 — Watch (live incremental auto-index)

**Status:** DRAFT · **Owner:** code-index change
**Depends on:** `001-files-chunks` (`index_freshness` — the re-check), `003-extraction`
(the extract tx the watch triggers).
**Consumed by:** `008-transport` (the `code_watch` command, admin).

---

## 1. Capability

Watch mode keeps the graph **live** without the agent manually running `code_index`
after every edit. It is the v1 `watch.go` (fsnotify-based), kept: an fsnotify watcher
on the project source tree, with **debounce** + **burst-collapse**, that triggers an
incremental `code_index` when files change.

**The model:** the watch is a *trigger*, not a *gate*. It does not block the agent; it
re-indexes in the background, and the query path's freshness check (`001` §4) is what
tells the agent the graph is fresh/stale. A query *during* a re-index sees the last
committed index (+ `degraded: stale_index` if the tree moved) — it never blocks on the
re-index (CI-INV-4).

## 2. The watcher

- **fsnotify** on the project source root (the same root the scan walks, `003` §2).
- **Debounce** (the v1 mechanics, kept): a configurable debounce window (default 2s,
  clamped 100 ms–60 s). Events within the window are **burst-collapsed** (a single
  re-index for a burst of edits, not one per file).
- **The re-check is the fingerprint** (`001` §4): on a debounced change, the watcher
  runs the **metadata-only stat-walk** (size + mtime vs `files` + the extractor stamp)
  to find the *actually-changed* files. No real change (a `git checkout` that touched
  mtime only) → no re-index (a no-op).
- **The re-index is incremental** (`003` §7): only the changed files are re-extracted
  (the target-state tx per file). The optional passes (LSP/PDG/communities/process)
  re-run only if their flag is on *and* the changed files affect them.

## 3. The auto-index loop

```
fsnotify event(s)
  → debounce (2s, burst-collapsed)
  → stat-walk re-check (size/mtime vs files + extractor stamp)
  → changed set (added/modified/deleted)
  → if empty: no-op (no re-index)
  → else: incremental code_index (the changed files)
  → update index_freshness
  → (advisory) re-run enabled optional passes
```

**Failure isolation:** a re-index failure (a parse error, a store lock) is logged + the
watch continues (the next event re-checks). A failure never stops the watcher. The
`index_freshness.last_indexed_at` reflects the last *successful* index.

**WAL retry** (the shared store's WAL retry, memory `00-foundation/001-storage`): a
flaky store (a WAL lock) retries the re-index (the store's retry policy, not the
watcher's).

## 4. The watch command

`mnemonic watch` (admin, `008-transport`): starts the watcher for the current project.
`code_status` reports whether the watch is active + the last re-index time. The watch is
**per-process** (a `mnemonic watch` process owns the watcher); it is not a daemon
(v2 has no `serve` for the code-index — the plugins + CLI drive it, the watch is an
opt-in background process the agent/user starts).

## 5. The relationship to the KB

The KB's **eager-per-save** write path (the memory `SaveEvent`) is *not* the watch — the
KB regenerates on a *memory save*, not on a *code change*. The watch re-indexes the
*code graph* on a *code change*; the KB's pin-check (`kb_validate`) is what *detects*
that the code moved (a stale pin). The two are independent: the watch keeps the *graph*
fresh; the pin-check keeps the *knowledge* honest. (The KB does not subscribe to the
watch — it reads the graph live, `002-sources` §2.2.)

## 6. Invariants

INV-2 (dual-mode — the re-check + re-index are dialect-agnostic), INV-3 (`degraded:
stale_index` during a re-index), INV-4 (local is truth — the watch reads the local
tree), CI-INV-1 (the graph is derived — the watch just re-derives it), CI-INV-4 (stale
is a signal, not a failure — a query during a re-index never blocks).

## 7. Open items

- **The debounce default:** 2 s (clamped 100 ms–60 s). Configurable. Not a spec constant.
- **Watch scope:** the project source root (the scan root, `003` §2). The KB vault
  (`.skillgrid/knowledge`) is *not* watched by the code-index watch (it is not code; the
  KB has its own `kb_index` for its FTS).
- **Watch + remote PG:** the watch re-indexes the *local* SQLite (source of truth,
  INV-4); the sync to remote PG is the memory function's concern (the code-index's
  remote replication is the shared store's sync, not the watch's).
