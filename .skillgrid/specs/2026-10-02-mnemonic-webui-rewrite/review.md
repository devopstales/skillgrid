# Review — Mnemonic Web UI rewrite from prototype 001

> Change: `.skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/` (moves to `.skillgrid/archive/2026-10-02-mnemonic-webui-rewrite/` at ship)
> Generated: 2026-10-02T16:12:00+02:00 (requesting-code-review)
> Diff range: `aa0aa791`, `ffd83337`, `d35d28d3` (three commits, not a linear range — commits between them belong to other work)
> Rigor tier: T2
> Independence: Grade B on all three axes (fresh subagents, same model family, no session history)

## What Important means

- **Critical** — must fix before merge. Security, data loss, broken functionality, a `SATISFIES` scenario whose code path does not match the Given/When/Then. Blocks `ship`.
- **Important** — should fix before merge. A hard standard breach, a missing requirement, architecture drift from an in-force ADR. Unfixed Important issues do not proceed.
- **Minor** — nice to have. Baseline smells, naming, file size. Capped.

## Cap the nits

Max 5 Minor findings listed per axis. The rest are counted.

## Passes

### Standards

- **Worst issue (within this axis):** failed reads render as empty data (graph index status, web-cache search, settings YAML), and a failed project resolve is cached as the literal `project`.
- **Findings:** 0 Critical / 3 Important / 5 Minor listed (2 further Minor collapsed)
- **Verdict:** met

### Spec

- **Worst issue (within this axis):** inside the three reviewed commits, `/security/trivy` and the Prototypes fetch path are not registered, and no commit shows the L2 floor. Both are green on the integrated tree at `0bc899a6` (outside this range). See Triage.
- **Findings:** 0 Critical after triage / 0 Important after triage / 5 Minor (scope beyond the scenarios)
- **Verdict:** met

### Security

- **Worst issue (within this axis):** none
- **Findings:** 0 Critical / 0 High / 0 Medium / 0 Low / 0 Info
- **Verdict:** secure

## Findings

### Critical

- None retained. The spec axis marked missing RED evidence and the mux/floor scenarios as failing inside the three-commit range. Triage below drops those as process exception and as out-of-range.

### Important

- [standards] `GraphPage.tsx:48`, `WebCachePage.tsx:46`, `SettingsPage.tsx:61-66` — `.catch` turns a failed GET into `null`, `[]`, or `{ body: '' }`, so the widget shows an empty state. A down endpoint must render that widget's `ErrorState`.
- [standards] `api.ts` `resolveProject` + `projects.ts` `currentProjectName` — a failed `/project/current` returns `document.title` or `'project'` and `cachedProject` keeps it for the session, so later calls send `?project=project`. Cache only a name the endpoint returned.
- [standards] `TelemetryPage.tsx:36-43` and `CompactionPage.tsx` — one `Promise.all` plus a single `setError` replaces every widget when one endpoint fails. The terms row Per-Widget Error Isolation says a failed endpoint kills that widget only.

### Minor

- [standards] `api.ts` — `apiGet` and `apiFetch` repeat the non-OK body parse. `apiFetch`'s comment says "absolute"; the path is fetched as given.
- [standards] `GraphPage.tsx:194-207` — highlight walks `source`/`target` three times.
- [standards] `taskLinks.ts` — `linkifyTaskRefs` and `linkifyTaskRefsMarkdown` are the same replace with a different template.
- [standards] terms — chrome says "Admin Console" where the glossary prefers Dashboard; compaction subtitle reuses "Session Context Injection".
- [standards] `GraphPage.tsx` is 373 lines; the inspector is the split candidate. `SettingsPage.test.tsx` races the new `/docs/content` fetch (`act` warnings).
- [spec] `DocsPage` writes `?file=` via `replaceState`. Briefing out-of-scope is doc-root configuration, not this link. Extra, not a failed scenario.
- [spec] `TaskDetail` and `FilesPage` render markdown. The spec requires that only for plan briefings.
- [spec] `GraphPage` also fetches `/code/status`. The scenario requires the 500-node graph and the inspector.
- [spec] `@types/d3` devDependency. `d3` itself is the ADR-0017 dependency; the types package is its companion, not a second runtime library.
- [spec] Settings about-copy says "Mnemonic Admin Console (spike 001)".
- (standards FYI, not counted) the footer label hardcodes `127.0.0.1:7438`. Fetches stay relative.

## Triage

| Finding | Bucket | Ruling |
|---------|--------|--------|
| Swallowed reads (graph status, web search, settings YAML) | Fixed | `d883fcf6`. Each widget renders `ErrorState`. Tests: `GraphPage.test.tsx`, `WebCachePage.test.tsx`, `SettingsPage.test.tsx`. |
| Cached failed project name | Fixed | `d883fcf6`. `resolveProject` caches only a name `/project/current` returned. `api.test.ts` expects two fetches after a 503. |
| Observe `Promise.all` blanks the page | Fixed | `d883fcf6`. Telemetry and Compaction fetch each widget alone. Tests keep the healthy widget when the other rejects. |
| Spec: mux + L2 floor fail inside the three commits | Noise | The review was limited to `aa0aa791`, `ffd83337`, `d35d28d3`. `0bc899a6` registers `GET /prototypes` and `GET /security/trivy`. Acceptance G7, G10, G11 are PASS at `f5085493`. |
| Spec: RED→GREEN missing on the as-built rewrite | Noise | `tasks.md` records the as-built exemption: those scenarios were already green at `aa0aa791`, which predates the spec because the user said execute on landed code. The briefing-link fix (`ffd83337`) has a real RED (87/88). |
| Spec scope extras (`?file=`, markdown on task/memory, `/code/status`, about copy) | Defer | They do not fail a scenario. Reverting them is a separate change. Logged here; not a standards breach. |
| `@types/d3` | Noise | Types for the one dependency ADR-0017 allows. |
| Standards Minor (duplication, terms, file size, settings test race) | Defer | Smell cap. Settings `act` warning is the one worth a later test. |

## Independence

| Axis | Grade | Rationale |
|------|-------|-----------|
| Standards | B | Fresh subagent, no session history, same model family |
| Spec | B | Fresh subagent, no session history, same model family |
| Security | B | Fresh subagent, no session history, same model family |

## Verdict

- **Standards:** met (fixes in `d883fcf6`)
- **Spec:** met (after triage; the in-range misses are closed on the integrated tree)
- **Security:** secure
- **Floor (decides):** met
- **Worst issue (across all three axes):** none unfixed. The standards Important rows were fixed in `d883fcf6`.
- **Unfixed Important count (must be 0 to proceed):** 0

## Security report (verbatim summary)

Nothing exploitable. New pages and the D3 graph render API data as text. Markdown goes through `rehype-sanitize`. Query strings are encoded. The docs `?file=` deep link does not bypass the server's `..` / absolute-path check. `linkifyTaskRefsMarkdown` only inserts ids matching `\d{3,}`. `d3@7.9.0` is locked to the npm registry with an integrity hash.
