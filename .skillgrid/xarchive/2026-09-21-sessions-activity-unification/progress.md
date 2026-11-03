# Ledger — 2026-09-21-sessions-activity-unification

BASE (branch start): 17caa72 (release/2)
Workspace: .skillgrid/sdd/2026-09-21-sessions-activity-unification

## Plan
- Tasks: 3 batches (B1 Go backend, B2 UI feature+rebuild, B3 remove+docs). Sequential, each its own commit.
- Delivery: size:exception single PR (one cohesive UI consolidation + 2 route changes).
- Ruling: hard rename /activity/snapshots→/handoff/snapshots (no alias) — user decision 2026-09-21.
- Ruling: add server-side GET /sessions/{id}/activity (clean scoping) — user approved "yes".
- Ruling: audit trail = secondary "Audit" tab (not default); Handoff Hub merged in (changes+handoffs kept, duplicate activity tab dropped) — user decisions 2026-09-21.

## QA gate
- Verdict: **PASS** (see qa-report.md). P0 100%, P1 100%. Full go suite: 28 ok / 9 FAIL — all pre-existing git-hook pkgs (cmd, embedder, project, store, service), none in the touched http pkg. UI vitest 37/37, tsc clean, build:check OK (index 295.1kB < 400kB). Security: no new deps, no secrets.

## Final two-axis review
- **Spec: 0 Critical/Important.** All 5 spec items MET (item 5 PARTIAL only on openapi.yaml propagation).
- **Standards: 0 Critical, 2 Important** (double SSE subscription; silent hub/audit/summary error swallow), 5 Minor.
- Fix wave (commit f28bf54): single page-owned SSE stream (registerActivity channel); hub+audit error banners; dark zinc tokens on Changes/Handoffs panes; "offline" label; dropped "one activity home" copy; openapi.yaml + ChangeSnapshot + TestPhase6_OpenAPI updated.
- Deferred (Minor, logged): handler copy-paste extraction (YAGNI); global-feed + snapshot-SSE unit tests (live-gate covered).
- Post-fix re-verify: Go http ok, UI 37/37, tsc clean, build:check OK, live :7438 (openapi 2 matches, SPA 200, scoped endpoint scoped, renamed 200, old → HTML).

## Commits
- 3d0a818 B1 (Go rename + scoped endpoint)
- 19a657e B2+B3 (UI feature + rebuild + remove routes/nav + docs)
- f28bf54 final-review fix wave

## Pre-flight
- Go http pkg baseline: PASS (`go test ./internal/mnemonic/http/...`).
- Server already running on :7438 (will restart after Go build for live check).
- Pre-existing unrelated WIP left untouched: .agents/skills/{resume,ship,work-unit-commits}, docs/user-guide/04-hooks.md, skills-lock.json, .opencode/, site/*, fk_probe_test.go.

## Task log

### B3 — remove old routes/nav + docs + live gate (commit 19a657e)
- Removed /activity + /handoff routes (app.tsx) and nav items (AppLayout.tsx); deleted ActivityPage + HandoffHubPage; updated 09-serve-dashboard.md.
- Live gate G2 on :7438 (rebuilt binary): /sessions/{id}/activity scoped ✓, /handoff/snapshots 200 ✓, /activity/snapshots → SPA HTML ✓, /handoff/status unchanged ✓, SPA serves /mnemonic/sessions with new SessionsPage chunk and no stale ActivityPage/HandoffHub chunks ✓.
- Work Unit Evidence:
  - Focused test: `npx vitest run` → 37/37 pass (incl. 5 new SessionsPage consolidation tests); `npm run build:check` → tsc clean + index 295.1kB < 400kB
  - Runtime harness: live serve :7438 → 4 route checks pass (above)
  - Rollback boundary: 18 files (12 in features/sessions, SessionsPage rebuild, app.tsx, AppLayout.tsx, docs); revert 19a657e restores old 4-surface layout

### B1 — Go backend (commit 3d0a818)
- Rename `/activity/snapshots`→`/handoff/snapshots` (server.go, handoff.go, activity.go, handoff_test.go) + new `GET /sessions/{id}/activity` (activity.go handler + server.go route + sessions_activity_test.go).
- RED verified (new path unmarshal-fail on SPA HTML; scoped route 404) → GREEN.
- Ruling: old-path "gone" check = body no longer parses as JSON (the SPA fallback serves shell HTML 200 for any unmatched GET, so 404 was the wrong assertion).
- Work Unit Evidence:
  - Focused test: `go test ./internal/mnemonic/http/ -run 'TestHandoffSnapshotsEndpoint|TestSessionScopedActivity'` → PASS
  - Runtime harness: full `go test ./internal/mnemonic/http/...` → ok (no regression)
  - Rollback boundary: 5 files (server.go, activity.go, handoff.go, handoff_test.go, sessions_activity_test.go); revert 3d0a818 restores old route + removes the new handler
