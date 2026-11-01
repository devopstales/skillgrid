# 007 — Lint (deterministic health)

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `001-structure` (the page format it lints), `004-index` (the `kb_pages`
rows it reads), `002-sources` (the `sources:` citations it checks), `006-validate`
(the pin-check it complements).
**Reference:** pi-llm-wiki deterministic lint (orphans / broken links / dup aliases /
coverage gaps), openwiki `.openwikiignore` + incremental health.

---

## 1. Capability

`kb_lint` is the **deterministic health check** of the vault. Unlike `kb_validate`
(which checks *grounded* pages against *live code*), lint checks the vault's **internal
consistency and completeness** — the structural health that does not depend on the
code. It is fast (no LLM, no graph required for most checks), runs frequently (after
`kb_generate`, in CI), and reports a **findings list** the agent (or a human) can triage.

**Lint is structural; validate is semantic.** Lint says "this page has no backlinks,
this wikilink is broken, this citation is missing." Validate says "this page's pinned
literal no longer holds in the code." Both are needed; lint is the cheap, always-on
one.

## 2. The checks

| Check | What it finds | Severity |
|---|---|---|
| **orphans** | a page with 0–1 incoming backlinks (no one links to it) | warn |
| **broken wikilinks** | a `[[type/slug]]` that resolves to no existing page | error |
| **duplicate aliases** | two pages sharing an `aliases` entry (ambiguous wikilink target) | warn |
| **coverage gaps** | a code module / package with no page (a gap in the brain's map) | info |
| **broken citations** | a `sources:` citation that does not resolve: a `memory:` `sync_id` that is gone, a `code:` path that does not exist, an `external:` citation whose version/URL is unresolvable **or whose `build_fingerprint` no longer matches upstream** (neuledge's staleness gate, run at lint time) | warn |
| **ungrounded grounded-page** | a grounded page (decision/invariant/workflow) with no `anchors:` or with `anchors:` that resolve to nothing | error |
| **weak pin** | a grounded page that pins a span by bare `line:` with no relocating `version:` token (`006-validate` §2.1) — a line-number drift will false-positive | warn |
| **past `stale_after`** | a page whose OKF `stale_after` date has passed (the *content-may-be-outdated* signal, distinct from a pin being stale) | warn |
| **stale index** | a page whose `body_hash` changed but `kb_index` has not re-indexed it (the index lags the files) | warn |
| **missing `sources:`** | a page with an empty `sources:` — **error** on a grounded page (no provenance), **warn** on a contextual page (architecture/gotcha/glossary) | error / warn |
| **contradiction markers** | pages carrying a `## Contradiction` marker (from `003-generate` step 4) that are unresolved | warn |
| **orphan citations** | a `sources:` citation to a fact/page that itself is orphaned (transitive staleness) | info |

**Severity model:** `error` findings are blocking in `--strict` mode (and in CI); `warn`
findings are reported but non-blocking; `info` findings are advisory. The default (non-
strict) `kb_lint` reports all three without failing.

## 3. How each check works

- **orphans / backlinks:** compute the backlink graph on the fly (scan all pages for
  `[[…]]`, build the in-degree per page). A page with in-degree ≤ 1 is an orphan. (A page
  with 0 links is a strong orphan; 1 is a weak orphan — still flagged, lower weight.)
- **broken wikilinks:** for every `[[type/slug]]` in every page, check that
  `<vault>/<type>/<slug>.md` exists (or that `type/slug` is in the page's `aliases`).
  A non-resolving link is broken.
- **duplicate aliases:** collect every `aliases` entry; an entry appearing on >1 page is
  a duplicate (ambiguous target).
- **coverage gaps:** read the code-index graph's **top-level Go packages** (live, or —
  if the graph is absent — a shallow `go list ./...`-style package scan) and compare to
  the set of pages that cite that package (`code:<pkg>/…` in `sources:` or `anchors:`).
  A top-level package with no page is a coverage gap. (This is the "the brain's map is
  incomplete" signal — advisory, not blocking. The unit is the *top-level package*,
  not a sub-package or file.)
- **broken citations:** for each `sources:` entry, resolve it:
  - `memory:<sync_id>` → the memory store has a live (or soft-deleted) row for it. A
    missing `sync_id` is broken (the fact was hard-deleted or never existed).
  - `code:<path>#L<a>-<b>` → the path exists in the working tree (and the line span is in
    range). A missing path is broken.
  - `external:<name>` (+`version`, `source_url`, `source_commit`, `build_fingerprint`)
    → the **neuledge staleness gate** (moved client-side from neuledge's publish-side
    `publicationSkipReason`): re-resolve the cited version against `source_url`. If the
    version/URL no longer resolves → **unresolved** (broken). If it resolves but the
    upstream `source_commit`/`build_fingerprint` no longer matches the cited one (the
    docs moved at the same version) → **stale** (the agent should re-cite at the new
    revision). If it still matches → fresh (no finding). Bounded + time-boxed; skipped
    offline (a skipped check is `info`, not `error`). This replaces the TTL — a citation
    is "stale" when the source moved, not when a clock expired.
- **ungrounded grounded-page:** a page with `type` ∈ {decision, invariant, workflow}
  must have `anchors:` that resolve (at least one). No anchors, or all anchors
  unresolvable, is a finding. (Complements `kb_validate` — lint checks the *presence*
  of a pin; validate checks the *truth* of the pin.)
- **weak pin:** a grounded page that pins a span by bare `line:` with no relocating
  `version:` token (`006-validate` §2.1). A line-number drift will false-positive on
  the next pin-check, so lint flags it `warn` and `--fix` leaves it (adding a token
  requires re-deriving the anchor, a judgment call).
- **past `stale_after`:** a page whose OKF `stale_after` date has passed. This is the
  *content-may-be-outdated* horizon (distinct from a pin being stale — the pin may still
  be `fresh`). Advisory: the agent should re-verify the page. `--fix` does not touch it.
- **stale index:** compare `kb_pages.body_hash` to the live file hash. A mismatch is a
  stale-index finding (the agent should run `kb_index`).
- **missing `sources:`:** a page with empty `sources:` is a provenance gap (INV-1).
  **Severity is type-dependent:** `error` on a grounded page (decision/invariant/
  workflow — a grounded page *must* be grounded), `warn` on a contextual page
  (architecture/gotcha/glossary — it *should* cite but may synthesize from general
  knowledge). This matches `001-structure` §5.1, where `sources:` is required (✅) on
  all six types but the *finding severity* scales with groundedness.
- **contradiction markers:** a `## Contradiction` section that has not been resolved (the
  sources still disagree) is surfaced. The agent must either resolve it (update the page
  / the memory fact) or explicitly mark it resolved.

## 4. `kb_lint --fix` (auto-repair what is safe)

`kb_lint --fix` auto-repairs the **safe, mechanical** findings — only the ones that do
not require judgment:

- **stale index** → runs `kb_index` (rebuild the lagging FTS rows). Safe (derived).
- **broken wikilinks (renamed target)** → if the link's `type/slug` matches a page by
  `aliases` or by a near-match, rewrites the link to the correct target. (Conservative:
  only when the target is unambiguous.)
- **duplicate aliases** → removes the duplicate from the *newer* page (the one with the
  later `updated_at`), keeping the canonical page's alias. (The newer page's alias is
  the likely accident.)

**Not auto-fixed** (require judgment → reported only): orphans, coverage gaps, broken
citations, ungrounded grounded-pages, **weak pin**, **past `stale_after`**, contradiction
markers, missing `sources:`. These need the agent (or a human) to decide the fix — lint
points at them, it does not guess. (Adding a relocating version token to a weak pin
requires re-deriving the anchor from live code; advancing a `stale_after` requires
re-verification — both are judgment calls, not mechanical repairs.)

## 5. Output

`kb_lint` returns a structured findings list:

```json
{
  "checked_pages": 42,
  "findings": [
    { "severity": "error", "check": "broken_wikilink",
      "page": "decisions/auth-model", "line": 18,
      "detail": "[[invariants/token-rotation]] does not resolve" },
    { "severity": "warn", "check": "orphan",
      "page": "gotchas/wal-timeout", "in_degree": 0 },
    { "severity": "warn", "check": "broken_citation",
      "page": "invariants/enterprise-grace",
      "citation": "code:internal/memory/old_path.go#L10",
      "detail": "path does not exist" }
  ],
  "summary": { "error": 1, "warn": 3, "info": 2 },
  "strict_failed": true
}
```

`--strict` (or CI) fails the process when `error` findings exist. The default reports
without failing.

## 6. What lint is NOT

- **Not the pin-check.** Lint does not re-read the code to verify a claim's literal
  (that is `kb_validate`). Lint checks the *structure* (citations resolve, links
  resolve, pins are *present*).
- **Not a linter for prose.** It does not grade the quality of a page's writing. It
  checks the mechanical health (links, citations, pins, coverage).
- **Not blocking by default.** Non-strict `kb_lint` is a report. It fails only under
  `--strict` / CI (and only on `error` findings).

## 7. Invariants

INV-1 (broken/missing citations are findings — provenance is enforced), INV-5 (a lint
run is recorded — the last lint time + findings count feed `kb_status`), KB-INV-2
(grounded pages are grounded — ungrounded grounded-pages are `error`), KB-INV-3 (the
stale-index check protects the derived layer).

## 8. Open items

None. The check set, the severities, and the `--fix` scope are fixed. The coverage-gap
unit is fixed at the **top-level Go package** (`007` §3); the no-graph fallback scan is a
shallow `go list ./...`-style package enumeration.
