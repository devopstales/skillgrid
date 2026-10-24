# Sessions as the Single "What Happened" UI

**Change:** 2026-09-21-sessions-activity-unification
**Class:** standard (UI consolidation + 2 small route changes) · **Verification floor:** L3 (Go tests + vitest + live serve)

## Problem
Four surfaces answer "what happened" and overlap:
- **Activity** (`/activity`, `features/activity`) — live observations feed (SSE) + stats/filters/health/alerts.
- **Handoff Hub** (`/handoff`, `features/handoff`) — 3 tabs: change-snapshot log (git), a *duplicate* activity tab, handoffs (checkpoints + refs).
- **Sessions** (`/mnemonic/sessions`, `features/mnemonic/SessionsPage`) — session list (left) + hash-chained audit trail (right); session selection is nearly no-op.
- The audit trail itself.

Every observation carries `session_id`, and `GET /activity/events` already returns `sessionId`. The data is already session-scoped — the UI just doesn't use it.

## Goal
One **Sessions** page at `/mnemonic/sessions` that is the single home for "what happened":
- Left: session list (unchanged).
- Right: **session detail** with a header + 4 tabs. Nothing selected → global view.

### Tabs
1. **Activity** *(default)* — the live feed (`EventCard` + SSE). **Scoped to the selected session** via a new `GET /sessions/{id}/activity?limit=N`; global feed when nothing selected. Existing `Filters`/`StatsBar`/`AlertBanner`/`AgentHealth` carry over, applied to the scoped feed.
2. **Changes** — moved from Handoff Hub: latest-snapshot header + `ChangeSnapshot` table + checkpoint markers (from `fetchSnapshots`/`fetchHubStatus` + `event: snapshot` SSE).
3. **Handoffs** — moved from Handoff Hub: `handoff_refs` + open checkpoints.
4. **Audit** *(secondary)* — the hash-chained `observation_versions` table + chain-validity badge (moved out of the default view).

### Removed
- `/activity` + `/handoff` **nav items and routes** and their page components. The `EventCard`/`Filters`/`StatsBar`/`AlertBanner`/`AgentHealth` + `api.ts` are **moved** into a new `features/sessions/` feature; `HandoffHubPage`'s `ChangesTab`/`HandoffsTab` + `handoff/api.ts` move too. The **API routes + SSE stay** (plugin/API consumers).
- Handoff Hub's duplicate `activity` tab — deleted.

## Backend (small)
1. **Rename** `GET /activity/snapshots` → `GET /handoff/snapshots` (hard, no alias). Consumers: `server.go:185` (route), `handoff.go:10` (doc), `activity.go:206` (comment), `handoff_test.go:46` (test). UI consumer `features/handoff/api.ts` is moved in B2.
2. **New** `GET /sessions/{id}/activity?limit=N` — server-side session-scoped feed. `handleMnemonicActivityEvents` gains an optional `session` filter (path `{id}` or `?session_id=`) adding `AND o.session_id = ?` to the observations query. No new table, no migration.

## Files
### New (UI)
- `skillgrid-ui/src/features/sessions/` — `api.ts` (merged activity+handoff+mnemonic-sessions fetches), `ActivityPane.tsx`, `ChangesPane.tsx`, `HandoffsPane.tsx`, `AuditTab.tsx`.

### Modified (UI)
- `skillgrid-ui/src/features/mnemonic/SessionsPage.tsx` — rebuild as the 4-tab activity home.
- `skillgrid-ui/src/app.tsx` — drop `/activity` + `/handoff` routes.
- `skillgrid-ui/src/components/layout/AppLayout.tsx` — remove the 2 nav items.

### Moved
- `features/activity/{EventCard,Filters,StatsBar,AlertBanner,AgentHealth,api}.tsx` → `features/sessions/`.
- `features/handoff/{api.ts, ChangesTab, HandoffsTab}` → `features/sessions/`.

### Deleted
- `features/activity/ActivityPage.tsx`, `features/handoff/HandoffHubPage.tsx` (+ `ActivityTab`), the 2 nav entries, the 2 routes.

### Backend (Go)
- `server.go` (rename route + register new route), `activity.go` (optional session filter), `handoff.go` (doc), `handoff_test.go` (rename), new `sessions_activity_test.go`.

### Docs
- `docs/user-guide/09-serve-dashboard.md` — page/route map.

## Threat Matrix (carried to spec/tasks)
| Threat | Mitigation |
|---|---|
| Route rename breaks a live consumer | hard rename; only 3 in-tree Go refs + the moved UI fetch; SSE + `/handoff/status` untouched; Go test asserts new path 200 + old 404 |
| Session filter leaks global rows | `AND o.session_id = ?` only when `session` present; test: scoped returns only that session, empty returns none |
| Removed routes still referenced | grep shows only `app.tsx` + `AppLayout.tsx`; B3 deletes both; build:check catches a dangling import |
| Moved components import stale relative paths | B2 rewrites `./api` / `../activity/api` → local `./api`; vitest + tsc catch it |
| SSE double-subscribed (activity + snapshot on same stream) | one `EventSource` on `/activity/stream`, multiplex `activity` + `snapshot` events |

## Rollback
Each task is an independent, revertable commit. Revert B3 → B2 → B1 in order to restore the four-surface layout; the backend rename is self-contained (revert `server.go` + test).

## Delivery
Single PR on `release/2` (size:exception — one cohesive UI consolidation + 2 route changes). No data migration, no new trust boundary.
