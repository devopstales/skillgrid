# Report — Mnemonic Web UI rewrite from prototype 001

> Change: `.skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/` (moves to `.skillgrid/archive/2026-10-02-mnemonic-webui-rewrite/` at ship)
> Generated: 2026-10-02T15:45:00+02:00 (qa); re-verified 2026-10-02T15:56:00+02:00
> Gate: PASS
> Effective tier: T2 (briefing `Tier:` line) · Classification: standard → applied floor L2 (full UI suite + build:check + Go embed build). T2 selects L2+L3; L3 (a browser walk against `skillgrid serve`) was not run. The L2 floor is met on the integrated tree.
>
> Two phases, one file: **qa** writes the QA half (the sections above `## Final-State Facts`,
> through `## Gate Decision` + `## Human Override`) into the spec folder. **ship** reads the
> `## Gate Decision` verdict PRE-MOVE, then moves the folder. **reflect** completes the retro
> half (from `## Final-State Facts` onward) IN PLACE in the archive folder.

Evidence base: clean detached worktree `/tmp/sg-webui-verify` at `ffd83337` (HEAD of `release/2`), `node_modules` symlinked from the main tree. Pre-rename baseline `d5e01985` where noted. Ledger: `.skillgrid/sdd/2026-10-02-mnemonic-webui-rewrite/progress.md`.

## Test Plan

### Risk Ranking

Most likely production break: a panel whose fetch path the Go mux never registers renders `ErrorState` forever (Security, Prototypes) — the UI is embedded in the binary, so there is no separate deploy to catch it. Second: the bundle crossing the 400 kB budget after the d3 swap. Third: the markdown sanitiser silently dropping links (the regression TICKET-05 fixed).

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Nav renders the six mockup groups in order | `AppLayout` DOM | unit | `src/components/layout/AppLayout.test.tsx` | P1 | pr | covered (4 passed) |
| 2 | Demoted routes still exist | route tree | unit | gate G2 (`rg -c` on `app.tsx` = 5) | P1 | pr | covered |
| 3 | Indigo tokens, no green accent | `styles/index.css` | unit | gate G3 | P1 | pr | covered |
| 4 | Graph requests `limit=500`, D3 force, inspector present | `GraphPage` | unit | gate G4 (3 lines) | P0 | pr | covered |
| 5 | Sigma stack removed, d3 present | `package.json` | unit | gate G5 (0 / 1) | P1 | pr | covered |
| 6 | Bundle under 400 kB with `vendor-d3` chunk | `scripts/check-bundle-size.mjs` | integration | `npm run build:check` | P0 | pr | covered (298.7 kB at `0bc899a6`) |
| 7 | Observe panels hit registered routes | Go mux ↔ `apiGet` paths | integration | gate G6 (= 4) + `src/features/settings`, `src/features/swagger` suites | P0 | pr | covered (4; 9 passed) |
| 8 | Security + Prototypes panels hit registered routes | Go mux ↔ `apiGet` paths | integration | gate G7 | P0 | pr | covered (1 / 1 at `0bc899a6`) |
| 9 | API origin relative in prod | `lib/apiBase.ts` | unit | gate G8 | P1 | pr | covered |
| 10 | Briefing `#NNN` becomes `/tracker?task=NNN` under MarkdownView | `PlansPage` DOM | unit | `src/features/plans/PlansPage.linkedTasks.test.tsx`, `taskLinks.test.ts::emits a markdown link…` | P1 | pr | covered (11 passed) |
| 11 | Full UI suite green | vitest | unit | `npm test` | P0 | pr | covered (122 passed after `d35d28d3`) |
| 12 | Go builds with and without `-tags ui` | `go build` | integration | gate G11 | P0 | pr | covered (both exit 0 at `0bc899a6`) |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| d3-code-graph | graph > 500 nodes | request capped at 500 | G4 (`fetchGraph({ limit: 500 })`) | covered (static) |
| briefing-task-refs-linkified | `#012` inside markdown text | markdown link, not raw `<a>` | `taskLinks.test.ts::emits a markdown link for text that a markdown renderer will display` | covered |
| panels-fetch-live-endpoints | endpoint unregistered | panel shows `ErrorState` | none (asserted by reading `SecurityPage.tsx` only) | pending |
| relative-api-urls | `import.meta.env.DEV` | absolute `127.0.0.1:7438` only in dev | G8 | covered |
| verification-floor | bundle at budget edge | "exceeds budget" exits non-zero | `check-bundle-size.mjs` | covered at `d5e01985` |

### Out of Scope

- Runtime (L3) browser walk of `/mnemonic/graph` against `skillgrid serve` — the binary does not build at HEAD; re-run after the external fix lands.
- Backend behaviour of `/security/trivy` and the prototypes listing — owned by the uncommitted `docs/security.go` / `docs_prototypes.go`; reviewed there, not here.
- Auth, pagination, mobile — out of scope per briefing.

## Goal-Backward Verification

**Stated goal** (from briefing.md): Rewrite the embedded SPA to match prototype 001's IA, theme and panels, with the D3 graph and live-endpoint panels, keeping `npm test` + `build:check` + Go embed green.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | Nav matches mockup groups | `AppLayout.test.tsx` 4 passed; G2 = 5 | VERIFIED |
| Truth | D3 graph loads ≤500 nodes with inspector | `GraphPage.test.tsx` (`d35d28d3`): `fetchGraph({ limit: 500 })`, "Node Inspector", svg appended | VERIFIED |
| Truth | New panels fetch live endpoints | G6 = 4; G7 = 1 then 1 at `0bc899a6`; `go test` docs + http packages ok | VERIFIED |
| Truth | `npm test` + `build:check` pass; Go embed builds | clean worktree `0bc899a6`: 110 passed, budget OK 298.7 kB, both `go build` variants exit 0; main tree after `d35d28d3`: 122 passed | VERIFIED |
| Artifact | `GraphPage.tsx`, `apiBase.ts`, `index.css`, `AppLayout.tsx` | present and imported from `app.tsx` | VERIFIED |
| Key Link | `PlanDetail` → `linkifyTaskRefsMarkdown` → `MarkdownView` | `PlansPage.linkedTasks.test.tsx` renders the anchor | VERIFIED |
| Data Flow | `apiGet` injects `project` and uses relative origin | `api.test.ts` (`/prototypes?project=skillgrid`, opt-out) and `apiBase.test.ts` (`''` outside DEV) in `d35d28d3` | VERIFIED |

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| nav groups render in mockup order | `AppLayout.test.tsx` (#1) | yes | pass |
| demoted routes remain reachable | G2 (#2) | yes | pass |
| index.css carries the indigo surface tokens | G3 (#3) | yes | pass |
| graph page requests at most 500 nodes | G4 (#4) | yes | pass |
| Sigma stack removed | G5 (#5) | yes | pass |
| Observe panels hit registered routes | G6 + settings/swagger suites (#7) | yes | pass |
| Security and Prototypes panels hit registered routes | G7 (#8) | yes | pass (1 / 1 at `0bc899a6`) |
| apiBase is relative in production | G8 (#9) | yes | pass |
| briefing #012 becomes a tracker link | #10 | yes | pass |
| full UI suite and bundle budget pass | #11 + #6 | yes | pass (110 at `0bc899a6`; 122 after `d35d28d3`; budget OK) |
| Go embed builds | #12 | yes | pass (`go build` and `-tags ui` at `0bc899a6`) |

**Coverage:** 11/11 scenarios covered by a check that ran and passed at `0bc899a6` (UI suite re-run 122/122 after `d35d28d3`). The first render of this report (8/11, FAIL) recorded HEAD `ffd83337`, which did not build.

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| (closed) HEAD did not compile | `server.go`, `app.tsx` | regression (external) | closed by `0bc899a6`; both builds exit 0 | — |
| (closed) endpoints unregistered | `server.go` | missing-adoption | closed by `0bc899a6`; G7 = 1 / 1 | — |
| (closed) GraphPage unmounted | `GraphPage.tsx` | broken-verification | closed by `GraphPage.test.tsx` in `d35d28d3` | — |

No open gap. The first render of this audit (three open rows) is what the FAIL verdict rested on.

## TDD Evidence Audit

| Task (from tasks.md) | SATISFIES Scenario | RED Evidence | GREEN Evidence | Verdict |
|----------------------|--------------------|--------------|----------------|---------|
| TICKET-01 | nav-matches-mockup, theme-tokens, relative-api-urls | pre-`aa0aa791` state, not re-run (as-built, declared in tasks.md) | G1-G3, G8 at `d5e01985` | OK (as-built exemption recorded) |
| TICKET-02 | d3-code-graph | same | G4, G5, build:check at `d5e01985` | OK (as-built exemption recorded) |
| TICKET-03 | panels-fetch-live-endpoints (Observe) | same | G6 = 4, 9 passed | OK (as-built exemption recorded) |
| TICKET-04 | panels-fetch-live-endpoints (Security, Prototypes) | G7 = 0 / 0 at `ffd83337` | G7 = 1 / 1 at `0bc899a6`; docs + http tests ok | OK |
| TICKET-05 | briefing-task-refs-linkified | `npm test` 87/88 at `d5e01985` | 11 passed at `ffd83337` | OK |
| TICKET-06 | verification-floor | G10/G11 FAIL at `ffd83337` | both builds + suite + budget green at `0bc899a6` | OK |

## Test Quality Audit

No open contract violation. `GraphPage.test.tsx` now renders the page (the earlier G4 grep stands as a supplement). `PlansPage.linkedTasks.test.tsx` renders real `PlanDetail`.

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| Nav / tokens / task links | unit | no | no | pure render + DOM assertions |
| Endpoint registration | integration | no | no | needs the Go mux + UI path in one check (G6/G7) |
| Bundle budget / Go embed | integration | no | no | build artefacts |

**Layer distribution:** 7 unit, 5 integration, 0 e2e. Reasonable; e2e not in `testing.layers`.

## Security Audit

**Mode:** Trivy

### Trivy Findings (Mode A)

**Command:** `trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW` (worktree at `ffd83337`, `skillgrid-ui/node_modules` skipped)

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| vuln | GHSA-p98j-92pf-mc4p (dompurify IN_PLACE afterSanitize hook) | LOW | `skillgrid-ui/package-lock.json` dompurify 3.4.15 (prod dependency) | 3.4.16 | SUGGESTION — `src/` uses neither `IN_PLACE` nor `addHook`/`afterSanitize` |

`skillgrid-cli/go.mod`: 0. `.skillgrid/prototypes/001-cgo-free-vector-db/go.mod`: 0.

**Trivy gate:** `fail_on: ""` → N/A (advisory-only, per locked constraint)

### OWASP Spot-Check

| File | OWASP Item | Pattern | Evidence |
|------|------------|---------|----------|
| `src/features/docs/MarkdownView.tsx:101` | A03 | `react-markdown` + `rehype-sanitize` (`defaultSchema` extended for GFM) | sanitiser in place; TICKET-05 emits markdown links rather than raw HTML, so the sanitiser stays the single sink |
| `src/features/docs/MermaidBlock.tsx:61` | A03 | `DOMPurify.sanitize(svg, …)` on rendered mermaid SVG | the only DOMPurify call site; no `IN_PLACE` / `addHook`, so GHSA-p98j-92pf-mc4p is not reachable |
| `src/features/swagger/SwaggerPage.tsx` | A05 | iframe `src={apiUrl('/swagger/')}` | same-origin, no user-controlled src |

**Security verdict:** 1 finding — CRITICAL: 0, WARNING: 0, SUGGESTION: 1

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage | not configured | 0 | n/a | N/A |
| Mutation | not configured | 0 | n/a | N/A |
| Lint | not configured (`go vet` inline) | errors only | `go vet` prints one pre-existing note (`memory/budget.go:114`, W005); `http` builds | N/A |
| Typecheck | `tsc -b` via `build:check`; Go compilation | errors only | 0 (`build:check` and both `go build` variants at `0bc899a6`) | PASS |
| P0 pass rate | plan rows #4,#6,#7,#8,#11,#12 | 100 | 6/6 = 100% | PASS |
| P1 pass rate | plan rows #1,#2,#3,#5,#9,#10 | 95 | 6/6 = 100% | PASS |
| Trivy security | `trivy fs .` | report only | 0 ≥ fail_on | N/A |

**Quality config status:** configured

## State Drift

**Verdict:** DRIFT: none (`node .agents/skills/verification/qa/scripts/state-drift-check.mjs`)

**Scope (from the guard's `SCOPE:` line):** COMPLETE

## Verification Scope

| Derivation | Scope | Stale? |
|---|---|---|
| Traceability matrix | COMPLETE (11/11 scenarios enumerated from acceptance.feature) | none |
| Verification-gap audit | COMPLETE (changed-file set of `aa0aa791` + `ffd83337`) | none |
| State Drift (9.5) | COMPLETE | n/a |
| Structure Drift (9.6) | not run (no structure script in this skill version) | n/a |

**Stale-verification:** `STALE: none` for the re-verification. Counts that name `ffd83337` are the first-render baseline; the gate rests on `0bc899a6` plus `d35d28d3`.

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | VERIFIED |
| Traceability (weakest scenario) | COMPLIANT (11/11) |
| Verification-gap audit | none open |
| TDD evidence (weakest ticket) | OK |
| Test quality audit | none |
| Code-quality gates (weakest gate) | PASS (coverage/mutation/lint/trivy N/A) |
| Security audit | SUGGESTION (dompurify LOW, not reachable) |
| Verification scope (composite) | COMPLETE |

**FLOOR:** Security audit — SUGGESTION. A suggestion does not cap the gate below PASS (no CRITICAL, no non-answer dimension). PASS-eligible.

## Findings

### CRITICAL (must fix before merge)

- None. The two CRITICAL rows from the first render (HEAD `ffd83337` did not build; TICKET-04 endpoints unregistered) closed at `0bc899a6`.

### WARNING (should fix before archive)

- None.

### SUGGESTION (nice to have)

- dompurify 3.4.15 → 3.4.16 (GHSA-p98j-92pf-mc4p, LOW, not reachable: `MermaidBlock.tsx` does not use `IN_PLACE` or `addHook`). W010.
- L3 not run: no browser walk of `/mnemonic/graph` against `skillgrid serve`. The classification floor is L2, which passed. W011.

## Gate Decision

**Verdict:** PASS

**Reasoning:** Re-verified after `0bc899a6` registered `GET /prototypes` and `GET /security/trivy` and restored the build. G7 is 1/1, the UI suite is 122/122, `build:check` is under budget, and both Go builds exit 0. `d35d28d3` adds the mounted graph test and the `apiGet` / `apiOrigin` tests the first render had marked PRESENT_BEHAVIOR_UNVERIFIED. The remaining dompurify note is LOW and not reachable, and `fail_on` is empty, so it does not move the verdict.

## Human Override

> A human decision always overrides this machine verdict.
> An epic that fails its criteria with no human decision is recorded as **not accepted** — never as silently accepted.

**Human decision (fill in):** <accept / accept-with-open-items / reject> — <name> — <date>

<!-- reflect completes this half at archive time -->
