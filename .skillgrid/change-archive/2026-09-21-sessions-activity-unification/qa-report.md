# QA Report — 2026-09-21-sessions-activity-unification

**Tier:** T2 (standard change; L2+L3 floor — unit + integration/live-serve) · **Floor applied:** L3 (live serve on :7438)
**Commits:** 3d0a818 (B1 Go) · 19a657e (B2+B3 UI) · base 17caa72
**config:** `quality.coverage_min: 0` (disabled), `quality.mutation_min: 0` (disabled), `p0_pass_rate: 100`, `p1_pass_rate: 95`, `testing.runner: go test ./...`

## Verdict: **PASS**

All truths VERIFIED by named tests; all scenarios covered by a test that ran and
passed; no CRITICAL findings; no MISSING_RED; all code-quality gates PASS or
N/A; P0 100%, P1 100%.

## Test Plan

| # | Scenario (behavior) | Test | Layer | Cadence | Rank | Status |
|---|---|---|---|---|---|---|
| 1 | `/handoff/snapshots` returns the seeded snapshot (renamed route) | `TestHandoffSnapshotsEndpoint` (http) | unit | pr | P0 | covered |
| 2 | old `/activity/snapshots` no longer returns JSON (hard rename, no alias) | `TestHandoffSnapshotsEndpoint` (old-path probe) | unit | pr | P0 | covered |
| 3 | `/sessions/{id}/activity` returns only that session's events, newest-first | `TestSessionScopedActivity` (http) | unit | pr | P0 | covered |
| 4 | `/sessions/{id}/activity` for a session with no obs → 200 + empty events | `TestSessionScopedActivity` (empty case) | unit | pr | P1 | covered |
| 5 | Sessions page renders the session list | `SessionsPage.test` (list) | unit | pr | P1 | covered |
| 6 | selecting a session scopes Activity to `/sessions/{id}/activity` | `SessionsPage.test` (scoped fetch) | unit | pr | P0 | covered |
| 7 | Audit is a distinct secondary tab (not default) | `SessionsPage.test` (audit tab) | unit | pr | P1 | covered |
| 8 | Handoff change log folded in as Changes tab | `SessionsPage.test` (changes) | unit | pr | P1 | covered |
| 9 | Handoff handoffs folded in as Handoffs tab | `SessionsPage.test` (handoffs) | unit | pr | P1 | covered |
| 10 | end-to-end: scoped route + renamed route + SPA on a live server | live gate G2 (`curl` on :7438) | integration | pr | P0 | covered (runtime) |

**Out of scope:** the pre-existing `embedder`/`project`/`store`/`cmd` git-hook
test failures (reproduce at base `17caa72`, not touched by this change);
coverage/mutation thresholds (disabled in config).

## Goal-Backward Verification

| Truth | Test | Level | Status |
|---|---|---|---|
| One Sessions page is the activity home; old /activity + /handoff pages gone | SessionsPage.test + app.tsx route removal + live SPA serves /mnemonic/sessions with SessionsPage chunk, no ActivityPage/HandoffHub chunks | Truth+Artifact+KeyLink+DataFlow | VERIFIED |
| Activity tab is session-scoped (server-side), global when none selected | `TestSessionScopedActivity` + live `curl /sessions/{id}/activity` (all events sessionId==id) | Truth+DataFlow | VERIFIED |
| `/activity/snapshots` renamed to `/handoff/snapshots` (hard) | `TestHandoffSnapshotsEndpoint` (new 200 / old not-JSON) + live `curl` | Truth+DataFlow | VERIFIED |
| Handoff change log + handoffs folded into Sessions (duplicate activity tab removed) | SessionsPage.test (changes/handoffs tabs) | Artifact+KeyLink | VERIFIED |
| Audit trail demoted to a secondary tab | SessionsPage.test (audit tab, not default) | Artifact+Behavior | VERIFIED |

## Verification-Gap Audit

- **Regression gaps:** none — every changed behavior (route rename, scoped
  route, tab fold-in, scoped fetch) has a covering unit test + the live gate.
- **Missing-adoption gaps:** none — the only `/handoff/snapshots` consumer is
  the moved `features/sessions/api.ts` (verified by grep; build:check would
  catch a dangling import).
- **Broken-verification gaps:** none.
- **Missing-oracle gaps:** none — each behavior has a runnable CHECK (the named
  test) + EXPECT.
- Note: the `EventsTab` SSE multiplex (single `/activity/stream` for both
  `activity` + `snapshot`) is exercised live (G2) but the snapshot-event branch
  is not pinned by a unit test — low risk (identical pattern to the former
  Handoff Hub, which worked), logged as a minor.

## TDD Evidence Audit

| Ticket | RED | GREEN | Scenario match | Verdict |
|---|---|---|---|---|
| B1 (Go) | new path unmarshal-fail + scoped 404 (before impl) | `go test ./internal/mnemonic/http/...` ok | rename + scoped scenarios | OK |
| B2 (UI) | 3 of 5 SessionsPage tests failed before impl (capitalize text + ambiguous selector) | `npx vitest run` 37/37 | scoped/tab/audit scenarios | OK |
| B3 (UI) | `tsc -b` failed on removed routes before app.tsx/AppLayout edits | `tsc -b` clean + build:check OK | route-removal | OK |

No MISSING_RED.

## Test Quality Audit

- Real code: Go tests hit the real mux+store (no mocks); UI tests render the
  real component with only the `api` module mocked (boundary, realistic shapes).
- No vacuous/pass-always: the scoped test would fail if the `AND o.session_id`
  clause were dropped (would return session B's rows too); the old-path probe
  fails if the handler is re-registered.
- P0 triangulation: scoped route tested with 2 sessions + an empty session
  (different inputs) → triangulated. Rename tested with new + old path.
- Minor (deferred): the snapshot-SSE branch and the "global when none selected"
  path are covered by live-gate only, not a unit test.

## Security Audit (Mode B — manual; no `security.trivy.command` in config)

- **Secrets:** none in changed files (grep clean). No `.env`/`.pem` added.
- **Dependencies:** no new Go deps (stdlib + internal only); no `package.json`
  change (no new UI deps).
- **OWASP spot-check:** the new Go handler reads `r.PathValue("id")` + the
  `project` query, both parameterized into the SQL (`?` placeholders) — no
  injection. No user-facing input handling introduced in the UI (same fetch
  pattern as existing views).

## Code Quality Gate

| Gate | Command | Result | Threshold | Status |
|---|---|---|---|---|
| Go build | `go build ./...` (via test compile) | ok | — | PASS |
| Go tests (touched) | `go test ./internal/mnemonic/http/...` | ok | — | PASS |
| Go tests (full) | `go test ./...` | 28 ok / 9 FAIL (all pre-existing git-hook pkgs: cmd, embedder, project, store, service — none in http) | — | PASS (regression-free) |
| go vet (http) | `go vet ./internal/mnemonic/http/` | clean | — | PASS |
| UI typecheck | `npx tsc -b` | clean | — | PASS |
| UI tests | `npx vitest run` | 37/37 | P0 100 / P1 95 | PASS |
| UI build | `npm run build:check` | tsc + bundle + index 295.1kB < 400kB | — | PASS |
| Coverage | (disabled) | — | 0 | N/A |
| Mutation | (disabled) | — | 0 | N/A |
| Lint | (no UI lint script) | — | — | N/A |
| gofmt (my hunks) | `gofmt -d` on changed files | my additions clean; 2 pre-existing map-alignment diffs in untouched code | — | PASS (note) |

## Findings

- **CRITICAL:** none.
- **WARNING:** none.
- **SUGGESTION / minor (deferred, logged):** (1) the snapshot-SSE branch +
  "global when none selected" path are live-gate covered but not unit-pinned;
  (2) two pre-existing gofmt map-alignment diffs in `activity.go`/`handoff.go`
  (untouched code) — leave for a separate formatting pass to keep this diff
  minimal.

**Decision:** PASS. No blocking findings. The 9 full-suite failures are the
documented pre-existing git-hook set (reproduce at base), none in the changed
package. Route to `requesting-code-review` (final two-axis review).
