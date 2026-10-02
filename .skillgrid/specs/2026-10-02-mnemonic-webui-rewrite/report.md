# Report — Mnemonic Web UI rewrite from prototype 001

> Change: `.skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/` (moves to `.skillgrid/archive/2026-10-02-mnemonic-webui-rewrite/` at ship)
> Generated: 2026-10-02T15:45:00+02:00 (qa)
> Gate: FAIL
> Effective tier: T2 (briefing `Tier:` line) · Classification: standard → applied floor L2 (full UI suite + build:check + Go embed build). T2 selects L2+L3; L3 (runtime harness) was not run because the integrated tree does not build.
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
| 6 | Bundle under 400 kB with `vendor-d3` chunk | `scripts/check-bundle-size.mjs` | integration | `npm run build:check` | P0 | pr | covered at `d5e01985` (298.2 kB); **fails at HEAD** — `tsc -b` error, external (see Findings) |
| 7 | Observe panels hit registered routes | Go mux ↔ `apiGet` paths | integration | gate G6 (= 4) + `src/features/settings`, `src/features/swagger` suites | P0 | pr | covered (4; 9 passed) |
| 8 | Security + Prototypes panels hit registered routes | Go mux ↔ `apiGet` paths | integration | gate G7 | P0 | pr | **pending** — 0 / 0 at HEAD; TICKET-04 blocked |
| 9 | API origin relative in prod | `lib/apiBase.ts` | unit | gate G8 | P1 | pr | covered |
| 10 | Briefing `#NNN` becomes `/tracker?task=NNN` under MarkdownView | `PlansPage` DOM | unit | `src/features/plans/PlansPage.linkedTasks.test.tsx`, `taskLinks.test.ts::emits a markdown link…` | P1 | pr | covered (11 passed) |
| 11 | Full UI suite green | vitest | unit | `npm test` | P0 | pr | covered (18 files, 89 passed at HEAD) |
| 12 | Go builds with and without `-tags ui` | `go build` | integration | gate G11 | P0 | pr | **pending** — `go build ./...` fails at HEAD, external |

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
| Truth | D3 graph loads ≤500 nodes with inspector | G4 3 lines, G5 0/1, `vendor-d3` chunk at `d5e01985` | PRESENT_BEHAVIOR_UNVERIFIED (no test mounts `GraphPage`; static evidence only) |
| Truth | New panels fetch live endpoints | Compaction/Web Cache/Telemetry: G6 = 4, settings+swagger 9 passed. Security/Prototypes: G7 = 0/0 | UNVERIFIED (Security, Prototypes) |
| Truth | `npm test` + `build:check` pass; Go embed builds | `npm test` 89/89 at HEAD; `build:check` FAIL at HEAD (`tsc -b`); `go build` FAIL at HEAD | UNVERIFIED at HEAD (last green `d5e01985`) |
| Artifact | `GraphPage.tsx`, `apiBase.ts`, `index.css`, `AppLayout.tsx` | present and imported from `app.tsx` | VERIFIED |
| Key Link | `PlanDetail` → `linkifyTaskRefsMarkdown` → `MarkdownView` | `PlansPage.linkedTasks.test.tsx` renders the anchor | VERIFIED |
| Data Flow | `apiGet` injects `project` and uses relative origin | G8 + reading `lib/apiBase.ts` (`apiOrigin()` returns `''` outside `import.meta.env.DEV`); no vitest test covers `apiGet` directly | PRESENT_BEHAVIOR_UNVERIFIED |

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| nav groups render in mockup order | `AppLayout.test.tsx` (#1) | yes | pass |
| demoted routes remain reachable | G2 (#2) | yes | pass |
| index.css carries the indigo surface tokens | G3 (#3) | yes | pass |
| graph page requests at most 500 nodes | G4 (#4) | yes | pass |
| Sigma stack removed | G5 (#5) | yes | pass |
| Observe panels hit registered routes | G6 + settings/swagger suites (#7) | yes | pass |
| Security and Prototypes panels hit registered routes | G7 (#8) | yes | **fail** (0 / 0) |
| apiBase is relative in production | G8 (#9) | yes | pass |
| briefing #012 becomes a tracker link | #10 | yes | pass |
| full UI suite and bundle budget pass | #11 + #6 | yes | **fail** at HEAD (`tsc -b`; `npm test` alone 89/89) |
| Go embed builds | #12 | yes | **fail** at HEAD |

**Coverage:** 8/11 scenarios covered by a test that ran and passed at HEAD. The three failures share one cause outside this change plus TICKET-04's open endpoints.

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| HEAD does not compile | `skillgrid-cli/internal/mnemonic/http/server.go:224-225`; `skillgrid-ui/src/app.tsx:102,120` | regression (external) | `ef32d593` deleted `http/prototypes.go` and `features/spikes/SpikesPage.tsx` while the callers' updates stayed uncommitted | any `go build` / `tsc -b` |
| Security + Prototypes endpoints unregistered | `server.go` (no `"GET /security/trivy"`, no prototypes route) | missing-adoption | G7 = 0 / 0 | `/system/security` and `/project/prototypes` render `ErrorState` |
| GraphPage has no mounted test | `src/features/mnemonic/GraphPage.tsx` | broken-verification (static-only) | G4 is a source-text grep | changing `limit: 500` to `limit: 50` would not fail a test |

## TDD Evidence Audit

| Task (from tasks.md) | SATISFIES Scenario | RED Evidence | GREEN Evidence | Verdict |
|----------------------|--------------------|--------------|----------------|---------|
| TICKET-01 | nav-matches-mockup, theme-tokens, relative-api-urls | pre-`aa0aa791` state, not re-run (as-built, declared in tasks.md) | G1-G3, G8 at `d5e01985` | OK (as-built exemption recorded) |
| TICKET-02 | d3-code-graph | same | G4, G5, build:check at `d5e01985` | OK (as-built exemption recorded) |
| TICKET-03 | panels-fetch-live-endpoints (Observe) | same | G6 = 4, 9 passed | OK (as-built exemption recorded) |
| TICKET-04 | panels-fetch-live-endpoints (Security, Prototypes) | G7 = 0 / 0 at HEAD | none | MISSING_GREEN (blocked) |
| TICKET-05 | briefing-task-refs-linkified | `npm test` 87/88 at `d5e01985` | 11 passed at `ffd83337` | OK |
| TICKET-06 | verification-floor | n/a | G10/G11 FAIL at HEAD | MISSING_GREEN (blocked, external) |

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| G4 (`GraphPage` grep) | test-claimed-path | source-text assertion; no render — flagged as SUGGESTION, not a vitest test |

No vitest test in the changed set violates a contract; `PlansPage.linkedTasks.test.tsx` renders real `PlanDetail` with a realistic plan shape.

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
| Lint | not configured (`go vet` inline) | errors only | `go vet`: 1 pre-existing (`memory/budget.go:114`, W005) + build error in `http` | FAIL (build) |
| Typecheck | `tsc -b` via `build:check`; Go compilation | errors only | 2 TS errors (`app.tsx:102,120`); 2 Go errors (`server.go:224,225`) | FAIL |
| P0 pass rate | plan rows #4,#6,#7,#8,#11,#12 | 100 | 3/6 = 50% | FAIL |
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

**Stale-verification:** `STALE: none` — every count in this report was produced at `ffd83337` except where `d5e01985` is named explicitly as the last green baseline.

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | UNVERIFIED (Security/Prototypes endpoints; floor at HEAD); also PRESENT_BEHAVIOR_UNVERIFIED for the GraphPage and apiGet data-flow rows |
| Traceability (weakest scenario) | FAILING (3 scenarios) |
| Verification-gap audit | CRITICAL (HEAD does not compile) |
| TDD evidence (weakest ticket) | MISSING_GREEN (TICKET-04, TICKET-06) |
| Test quality audit | SUGGESTION |
| Code-quality gates (weakest gate) | FAIL (typecheck/build, P0 50%) |
| Security audit | SUGGESTION |
| Verification scope (composite) | COMPLETE |

**FLOOR:** Code-quality gates — FAIL. The gate cannot render higher than FAIL.

## Findings

### CRITICAL (must fix before merge)

- HEAD `ffd83337` does not build: `go build ./...` → `server.go:224:54 s.handlePrototypes undefined`, `:225:48 s.handlePrototype undefined`; `tsc -b` → `app.tsx(102) PrototypesPage export missing`, `app.tsx(120) ./features/spikes/SpikesPage not found`. Introduced by `ef32d593` (staged-only commit of the spikes→prototypes rename from another session), not by this change. Fix: commit `server.go`, `docs/docs_prototypes.go(+_test)`, `docs/security.go(+_test)`, `src/app.tsx`, `AppLayout.tsx`, `features/prototypes/PrototypesPage.tsx` from the main working tree.
- TICKET-04: `/security/trivy` and the Prototypes fetch path are not registered on the Go mux (G7 = 0 / 0); `/system/security` and `/project/prototypes` render `ErrorState`. Same six files unblock it.

### WARNING (should fix before archive)

- None.

### SUGGESTION (nice to have)

- `GraphPage` has no mounted test; G4 is a source-text grep. Add a vitest render asserting the `limit=500` request and the "Node Inspector" panel.
- `lib/api.ts` / `lib/apiBase.ts` have no direct test; add a unit test asserting `apiOrigin()` is `''` outside DEV and that `apiGet` appends `project`.
- dompurify 3.4.15 → 3.4.16 (GHSA-p98j-92pf-mc4p, LOW, not exercised: no `IN_PLACE`/`addHook` in `src/`).

## Gate Decision

**Verdict:** FAIL

**Reasoning:** This change's own code is green — `npm test` 89/89 at HEAD, the TICKET-05 regression fixed and tested, G1–G6/G8/G9 pass — but the gate rests on the integrated tree, and the integrated tree neither type-checks nor compiles since `ef32d593`, which also leaves TICKET-04's two endpoints unregistered. Fix-loop cap not consumed: the fix is a commit of six files owned by another in-flight session, which cannot be made from this change without sweeping that work, so the gate escalates to the human.

**Path to re-verification:** after those six files land, re-run G7, G10, G11 (TICKET-04 and TICKET-06) in a clean worktree; expected outcome PASS with the two SUGGESTIONs open.

## Human Override

> A human decision always overrides this machine verdict.
> An epic that fails its criteria with no human decision is recorded as **not accepted** — never as silently accepted.

**Human decision (fill in):** <accept / accept-with-open-items / reject> — <name> — <date>

<!-- reflect completes this half at archive time -->
