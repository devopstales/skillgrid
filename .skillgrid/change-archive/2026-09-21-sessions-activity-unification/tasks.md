# Tasks — 2026-09-21-sessions-activity-unification

## Delivery Strategy
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Medium (one cohesive PR; ~400-500 lines across UI move + 2 route changes)

Global constraints (apply to every task):
- TDD on (repo standard). Every behavior change has a failing test first.
- Go test cmd: `cd skillgrid-cli && go test ./internal/mnemonic/http/...`
- UI test cmd: `cd skillgrid-ui && npx vitest run` · typecheck/build: `npm run build:check`
- Do NOT rename the SSE `/activity/stream` or `event: activity` / `event: snapshot` event names — only the REST snapshot path moves.
- Keep the `/handoff/status` route unchanged.
- The new session-scoped activity endpoint reuses `handleMnemonicActivityEvents`'s response shape (`{project, events, limit}`) so the UI needs no new type.

---

## B1 — Go backend: rename + session-scoped activity
- [ ] B1.1 Rename `GET /activity/snapshots` → `GET /handoff/snapshots` (hard, no alias)
  - `server.go:185` route; `handoff.go:10` doc; `activity.go:206` comment; `handoff_test.go:46` test.
  - RED: test GET `/handoff/snapshots` returns 200 with a snapshot; test GET `/activity/snapshots` now 404.
  - `Owns:` skillgrid-cli/internal/mnemonic/http/{server.go,handoff.go,activity.go,handoff_test.go}
  - `Needs:` —

- [ ] B1.2 New `GET /sessions/{id}/activity?limit=N` (server-side session-scoped feed)
  - `handleMnemonicActivityEvents` gains an optional `session` param (path `{id}` or `?session_id=`) → `AND o.session_id = ?` when present. Register `GET /sessions/{id}/activity` in `server.go` (near `/sessions/{id}/summary`, line ~92).
  - RED: seed 2 sessions, assert scoped returns only that session's events; a session with none returns `events: []`; global (no id) still returns all.
  - `Owns:` skillgrid-cli/internal/mnemonic/http/{activity.go,server.go,sessions_activity_test.go}
  - `Needs:` —

## B2 — UI: new features/sessions feature + rebuild SessionsPage
- [ ] B2.1 Create `features/sessions/api.ts` (merged) + move shared components
  - Move `activity/{EventCard,Filters,StatsBar,AlertBanner,AgentHealth}.tsx` and `handoff/api.ts` types into `features/sessions/`. New `api.ts` exports: activity (fetchActivityEvents w/ optional `sessionId`, fetchActivityStats, openActivityStream), handoff (fetchSnapshots→now `/handoff/snapshots`, fetchHubStatus, openSnapshotStream), sessions (fetchSessions, fetchAudit, new fetchSessionActivity(id), new fetchSessionSummary(id)).
  - `Owns:` skillgrid-ui/src/features/sessions/{api.ts,EventCard.tsx,Filters.tsx,StatsBar.tsx,AlertBanner.tsx,AgentHealth.tsx}
  - `Needs:` —

- [ ] B2.2 Panes: ActivityPane, ChangesPane, HandoffsPane, AuditTab
  - `ActivityPane` = current `ActivityPage` body (feed + stats + filters + alerts + health), param `sessionId?: string` (scope fetch + SSE filter). `ChangesPane`/`HandoffsPane` = moved `ChangesTab`/`HandoffsTab`. `AuditTab` = current `AuditTable` + chain badge.
  - RED: ActivityPane with `sessionId` fetches the scoped endpoint; ChangesPane/HandoffsPane render from moved APIs.
  - `Owns:` skillgrid-ui/src/features/sessions/{ActivityPane.tsx,ChangesPane.tsx,HandoffsPane.tsx,AuditTab.tsx}
  - `Needs:` B2.1

- [ ] B2.3 Rebuild `SessionsPage` as the 4-tab activity home
  - Left session list (unchanged). Right: header (selected session summary via `fetchSessionSummary`, else global title) + tabs [Activity(default) | Changes | Handoffs | Audit]. Activity tab passes `sessionId`; global when none selected.
  - RED: selecting a session switches Activity to scoped; no selection → global; Audit tab shows chain badge.
  - `Owns:` skillgrid-ui/src/features/mnemonic/SessionsPage.tsx
  - `Needs:` B2.2

## B3 — UI: remove old routes/nav, delete old features, docs
- [ ] B3.1 Remove `/activity` + `/handoff` routes (`app.tsx`) and nav items (`AppLayout.tsx`); delete `ActivityPage.tsx`, `HandoffHubPage.tsx` (+`ActivityTab`).
  - RED: a route/render test asserts `/activity` and `/handoff` no longer resolve and the nav items are gone; build:check passes (no dangling import).
  - `Owns:` skillgrid-ui/src/{app.tsx,components/layout/AppLayout.tsx,features/activity/ActivityPage.tsx,features/handoff/HandoffHubPage.tsx}
  - `Needs:` B2.3

- [ ] B3.2 Update `docs/user-guide/09-serve-dashboard.md` page/route map (Sessions = activity home; `/activity`+`/handoff` pages removed; `/activity/snapshots`→`/handoff/snapshots`; new `/sessions/{id}/activity`).
  - `Owns:` docs/user-guide/09-serve-dashboard.md
  - `Needs:` B3.1

## Gates
- G1 (integration): `go test ./internal/mnemonic/http/...` green AND `npx vitest run` green AND `npm run build:check` green.
- G2 (live): `skillgrid serve` → `/mnemonic/sessions`: pick a session → Activity scopes; Changes/Handoffs/Audit tabs render; `/activity` + `/handoff` nav gone; old snapshot path 404s.
