# Code Review — Web Admin Dashboard (parallel, 8 specialists)

> Change: `.skillgrid/specs/2026-09-08-web-admin-dashboard/`
> Reviewed range: `6b835e27..HEAD` (the committed dashboard implementation, 138 files / ~25k lines)
> Specialists: Standards · Spec · Edge cases · Verification gaps · Security · Accessibility · Performance · Red team (last)
> Raw findings: ~118 (pre-dedup). Below are the deduplicated, coordinator-verified entries.

**Concurrency note (reviewed 2026-09-19):** the review ran against the committed range (`6b835e27..HEAD`), where the dashboard is intact. A concurrent session on `release/2` has since committed a "visual companion" blueprint + a Mermaid fix and **deleted `skillgrid-ui/` from the working tree** (unstaged) while adding a Hugo `site/`. That concurrent work is mid-refactor; this report reviews the committed dashboard, and the fix loop is paused pending a decision on the working-tree state (see Open Items).

## Triage summary

| Specialist | Findings | Worst (verified) |
|---|---|---|
| Standards | 8 | 404-vs-500 inconsistency on `openHandleFor` (real, low) |
| Spec | 16 | **Prototypes version history MISSING** (a @phase-7 scenario) |
| Edge cases | 23 | SSE `openHandleFor` after headers written (real, medium) |
| Verification gaps | 8 | SSE leak test passes with 1 leaked goroutine (real, medium) |
| Security | 17 | git sha/path flag injection (real, medium); unbounded git output (real, medium) |
| Accessibility | 13 | mobile slide-over no focus mgmt / no Escape (real, low) |
| Performance | 21 | git N+1 `show --numstat` per commit (real, medium); ActivityPage per-event re-render (real, medium) |
| Red team | 12 | **3 different "root" resolution strategies** (docsCwd/sddRoot/embed) (real, medium) |

## Grouped findings (coordinator-verified)

### A — `medium` / fix-now (real, in-scope, worth fixing before archive)

| # | Finding (grouped) | Source | Verdict | Evidence |
|---|---|---|---|---|
| A1 | **Prototypes version history is MISSING** — a @phase-7 scenario ("Prototypes view shows version history with diffs") has no implementation at all. | Spec | `medium` (scenario unmet) | `fd features/prototypes` — no version/history/diff code. Scenario at `acceptance.feature` @phase-7. |
| A2 | **git flag injection** — `sha` (path value) and `path` (query) go to `git show`/`git log` without validating shape; a leading `-` or a `--output=`-shaped value can be parsed as a git option. | Security + Edge | `medium` | `git.go:92` (`git show … sha`), `git.go:174` (`git log … -- path`). |
| A3 | **Unbounded git output** — `gitCmd` uses `CombinedOutput()` with no byte cap and no context timeout; a huge diff/blame/log is held in memory per request (DoS). | Security + Perf | `medium` | `git.go:16` (`cmd.CombinedOutput()`), `git.go:149` (diff), `git.go:214` (blame). |
| A4 | **git N+1 process spawn** — `/git/commits` runs one `git show --numstat` per commit (default 50, max 500) = up to 501 sequential forks per request. | Perf | `medium` | `git.go:73` (per-commit `git show` in the loop). |
| A5 | **SSE poller: `openHandleFor` after headers** — the stream writes `Content-Type: text/event-stream` + `event: ready` before the DB handle is confirmed openable; a handle error then writes a JSON error body into an SSE response. | Edge | `medium` | `activity.go:164-180` (headers/ready, then the poller opens). |
| A6 | **SSE leak test is weak** — `after > before` passes with exactly one leaked poller goroutine (the realistic single-disconnect leak); only ≥2 leaks fail. | Verification | `medium` | `activity_test.go:183-185` (`if after > before`). |
| A7 | **SSE poll query has no LIMIT + per-client poller** — a burst of observations is all fetched/pushed in one tick (no LIMIT); each SSE client spawns its own 800ms poller (N tabs → N pollers on SQLite). | Perf + Security | `medium` | `activity.go:204` (per-request ticker), `activity.go:211-215` (no LIMIT). |
| A8 | **Plans spec sandbox check** — `strings.HasPrefix(relCheck, "..")` without the separator: `..foo` passes. (The sibling `stitchFile` in prototypes does it correctly with the separator — inconsistent.) | Security | `medium` | `plans.go:243` vs `prototypes.go` `stitchFile`. |
| A9 | **Three "root" strategies coexist** — `docsCwd` (fixed at package init), `sddRoot()` (env re-read per request), embedded `ui/dist` (fixed at go build). Setting `SKILLGIT_DOCS_CWD` steers plans/specs/git/prototypes but NOT `/docs`. | Red team | `medium` | `server.go:50` (docsCwd), `plans.go:45` (sddRoot), `docs_markdown.go` (docsCwd), `embed.go:14` (embed). |
| A10 | **ActivityPage re-renders 200 EventCards per SSE event** — `EventCard` is not memoized; each new event re-renders the whole filtered list, and triggers a full `/activity/stats` refetch per event. | Perf | `medium` | `ActivityPage.tsx:55` (per-event stats refetch), `:112` (unmemoized EventCard). |
| A11 | **`/specs` endpoints have no production consumer** — `fetchSpecFiles` in the UI is dead code; spec browsing actually goes through `/docs`. The `/specs` handlers ship with no live caller. | Red team | `medium` (dead surface) | `server.go:188` (`/specs/{path...}`), UI `plans/api.ts` dead fetch. |

### B — `low` / fix-now (real, cheap, in-scope)

| # | Finding (grouped) | Source | Verdict | Evidence |
|---|---|---|---|---|
| B1 | `openHandleFor` failure → 404 in `activity.go` but 500 in `mnemonic_graph.go`/`server.go` for the same error class. | Standards | `low` | `activity.go:50` vs `mnemonic_graph.go:263`. |
| B2 | `apiPrefixes` (the 404-JSON catch-all list) omits `activity/plans/git/prototypes/specs` — an unknown sub-path under those returns the SPA shell (HTML) instead of a JSON 404. | Standards + Red | `low` | `embed.go:28` (list). |
| B3 | `SKILLGIT_DOCS_CWD` typo in UI comments (actual env is `SKILLGRID_DOCS_CWD`). | Standards | `low` | `plans/api.ts:3`, `git/api.ts:2`. |
| B4 | Mobile slide-over: no focus management, no Escape-to-close, no visible close button; hamburger lacks `aria-expanded`/`aria-controls`. | Accessibility | `low` | `AppLayout.tsx:29,37,76`. |
| B5 | Density toggle + prototypes/git toggle groups lack `aria-pressed`/tab semantics; activity feed lacks `aria-live`; stream-in not `prefers-reduced-motion` gated; FileExplorer input has no label. | Accessibility | `low` | `AppLayout.tsx:110`, `ActivityPage.tsx:112`, `index.css:41`, `FileExplorer.tsx:92`, `PrototypesPage.tsx:104`. |
| B6 | Prototype HTML served without a CSP; SPA shell served without `X-Content-Type-Options: nosniff`. | Security | `low` | `prototypes.go:83`, `embed.go:74`. |
| B7 | Duplicated `api.ts` fetch helper + `resolveProject` across 5 features (each caches a different project → cross-view desync). | Standards + Red | `low` (smell) | `activity/api.ts`, `mnemonic/api.ts`, `graph/api.ts`. |

### C — `defer` (real, but out of scope for this change / lower priority)

| # | Finding (grouped) | Source | Verdict | Why deferred |
|---|---|---|---|---|
| C1 | **Git view is not a commit graph** (no branch lanes / merge nodes / avatars) — a @phase-7 scenario wants a graph; the impl is a flat commit list. | Spec | `medium` → defer | Larger UI feature; the flat list + detail + diff + blame meet the core read-only goal. A follow-up change. |
| C2 | **Plans "DAG" is a status pipeline, not a dependency DAG** (no dependency edges surfaced). | Spec | `low` → defer | The backend surfaces no dependency edges; a real DAG needs a data-model change. |
| C3 | **Activity stats counters differ from the scenario** (impl: total/byType/activeSessions; scenario: active-tasks/memories/searches/errors); no time-range filter; no infinite scroll; `relatedIds` always empty. | Spec | `low` → defer | The scenario's counter set implies new metrics that aren't tracked in the store; a follow-up. |
| C4 | **Kanban hard-codes 4 providers** (shows all, marks unconnected) rather than tabs driven by connected providers. | Spec | `low` → defer | Provider connectivity is a tracker-registry concern; acceptable degraded state. |
| C5 | **Prototypes list lacks title/thumbnail/updatedAt**; no `/prototypes/{id}/html` route (only `/{id}` returning HTML); no CSS/JS side-by-side. | Spec | `low` → defer | The list is a flat path array (UI derives title by filename); the single-HTML-file assumption means no CSS/JS split. |
| C6 | **User manual lists `linear` (no adapter) and omits `glab`/`jira`.** | Spec | `low` → defer | Doc correction; fix when the provider set is finalized. |
| C7 | **OpenAPI Phase 6/7 routes lack request/response examples** (docs routes have them; the new ones don't). | Spec | `low` → defer | The paths + schemas are documented + tested; examples are a polish item. |
| C8 | **Project selector is read-only** (no localStorage persistence / no switching) — a scenario wanted a persistent selector. | Spec + Red | `low` → defer | Intentional: the dashboard is bound to the served project (server-derived). A switcher is a follow-up. |
| C9 | **MarkdownView sanitize schema not UI-tested** (only the server side is tested); **SandboxPreview `sandbox` attr not asserted**; **swagger JS bundle not asserted**; **audit `chain_valid` always true / hash not recomputed**. | Verification | `low` → defer | The server-side behavior is tested; the client-side sanitize + sandbox are belt-and-suspenders. Adding these UI assertions is a follow-up. |
| C10 | **graph full rebuild per request** (`loadGraph` scans all symbols/edges/files even for a depth-limited neighborhood) + **community `contentKey` re-hash per request** + **docs/tree + docs/search O(all docs) per request** + **plans per-request full file reads** + **backlog `Get` full-dir re-read + `SetStatus` 2×`Get`**. | Perf | `low` → defer | Per-request I/O without caching; these scale with the repo but are acceptable at current size. A caching pass is a follow-up change. |
| C11 | **`treeLayout`/`filterGraphByDepth` use `queue.shift()` (O(n²) BFS)** + **`forceLayoutAnimated` per-frame Map alloc** + **no wall-clock floor on the force sim** + **`MermaidBlock` `svgCache` unbounded** + **`DiffViewer` no memo/virtualization**. | Perf | `low` → defer | UI perf at ~5k-node scale; the layout already settles + stops. Optimization pass is a follow-up. |
| C12 | **`app.tsx` lazy chunk 404 has no error boundary** (rejected import → blank page); **`__404__` route reachability unverified**; **StrictMode double-mount can open two EventSources**; **`shellOrJSON` shell branch never Go-tested** (all tests send `Accept: application/json`). | Edge + Verification + Red | `low` → defer | Real but low-frequency (chunk 404, double-mount in dev); the shell branch is browser-verified. A hardening pass is a follow-up. |
| C13 | **Read routes unauthenticated when `SKILLGIT_HTTP_TOKEN` unset** (data leak to any client on the port); **bearer token in `localStorage`** (XSS can read it); **`sddRoot` trusts env per request**; **docs `LIKE` scope `%`/`_` unescaped**; **MarkdownView `img src` `javascript:`/`data:` not constrained**; **L2 content unbounded**. | Security | `low` → defer (C13a token is `medium`) | The server binds `127.0.0.1` (loopback) by default, so the blast radius is local; the token gate exists for writes. A hardening change. |
| C14 | **Pre-existing `service` test failures** (`TestOpenFor*` — global conventional-commits git hook rejects `git commit -m init` in temp repos). | QA | `defer` | Different package, environment issue, confirmed unrelated (QA report). |

### D — `false` / rejected (checked, not a real issue)

| # | Finding | Verdict | Disproves it |
|---|---|---|---|
| D1 | "EventSource auto-reconnect stacks streams" — the client is a single `EventSource` per mount; the browser's auto-reconnect is to the same endpoint, not a second connection. (The StrictMode double-mount in C12 is the real, narrower variant.) | `false` (as filed) / `low` (narrowed) | `activity/api.ts` opens one `EventSource`; reconnect is browser-native to the same URL. |
| D2 | "shellOrJSON: a browser sending both `text/html` AND `application/json` gets JSON" — real browsers send `text/html` first; the condition is `?project=` OR `application/json`, and a bare browser nav has no `?project=` and its Accept starts with `text/html`. The data-leak framing (Security) is the more accurate C13/Red finding. | `maybe-false` | `mnemonic_files.go:386-394` (condition). |

## Verdict

**No `high` (blocker) findings.** The worst are `medium` — concentrated in the git bridge (A2/A3/A4), the SSE poller (A5/A6/A7), the plans sandbox (A8), and the root-resolution inconsistency (A9). These are fixable in this change. Several `medium` Spec gaps (A1 version history, C1 commit graph) are deferred as follow-ups because they'd be larger UI features, not regressions.

**Gate:** the QA gate was PASS. This review adds `medium` findings that should be fixed before archive (A1–A11, B1–B7) or explicitly deferred (C-group). The fix loop should address the in-scope `medium`+`low` set, then re-verify.

## Open items (→ next change / follow-up)

- **Working-tree concurrency:** `skillgrid-ui/` is deleted (unstaged) by a concurrent session + a Hugo `site/` + "visual companion" blueprint are in progress. The fix loop is paused; decide whether to (a) commit the review fixes against HEAD then let the concurrent work proceed, or (b) wait for the concurrent refactor to settle first.
- Prototypes version history (A1) + git commit graph (C1) + real plans DAG (C2) + activity metrics (C3) — the larger Spec gaps, as a follow-up change.
- A caching pass (C10) + a UI-perf pass (C11) + a hardening pass (C12/C13).
