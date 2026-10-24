# P2 Report — Visual Companion (Mnemonic Decision Bridge) — TICKET-03 + TICKET-04

## Status: DONE

P2 implemented the `/decisions` dashboard view that consumes the P1 backend
decision bridge (`GET /mnemonic/decisions` + `POST /mnemonic/decisions/{id}/answer`).
Two commits, strict TDD (RED → GREEN), `build:check` passing.

- RED commit:   `b7e39a4` — `test(ui): RED acceptance tests for decisions view (TICKET-03)`
- Impl commit:  `76ec335` — `feat(ui): decisions view with answer round-trip (TICKET-04)`

---

## TDD evidence

### RED run (after b7e39a4, before implementation)

The two new test files compile (vitest transforms them) but fail on the missing
components/route — the expected RED for "route/component not yet present":

```
 FAIL  src/features/decisions/DecisionCard.test.tsx [ src/features/decisions/DecisionCard.test.tsx ]
Error: Failed to resolve import "./DecisionCard" from "src/features/decisions/DecisionCard.test.tsx". Does the file exist?
 FAIL  src/features/decisions/DecisionsPage.test.tsx [ src/features/decisions/DecisionsPage.test.tsx ]
Error: Failed to resolve import "./DecisionsPage" from "src/features/decisions/DecisionsPage.test.tsx". Does the file exist?
 Test Files  2 failed (2)
      Tests  no tests
```

Tests compile (the failure is import-resolution, not a type/syntax error) — the
RED "tests compile, fail on missing route/components" criterion is met.

### GREEN run (after 76ec335)

Decisions tests:
```
 Test Files  2 passed (2)
      Tests  8 passed (8)
```

Full UI suite (the jsdom env change affects the pre-existing pure-logic tests,
so I re-ran everything):
```
 Test Files  5 passed (5)
      Tests  31 passed (31)
```

### Build check

`npm run build:check` = `tsc -b && vite build && node scripts/check-bundle-size.mjs` → **OK**.

- `tsc -b` clean (all of `src` typechecks, including the new `.tsx` tests).
- `vite build` produces its own lazy chunk: `dist/assets/DecisionsPage-BB_MUzOW.js`
  (confirmed the Decisions view is its own code-split chunk, per the global constraint).
- Bundle budget: `295.6 kB  index (budget 400 kB) — OK` / `Bundle-size budget OK.`

---

## Files changed

### New (TICKET-04)
- `skillgrid-ui/src/features/decisions/api.ts` — `fetchDecisions(state)`,
  `answerDecision(id, {optionId, note})`, `DecisionApiError`, `DecisionRow`
  (alias of `Decision`). Project resolution via `currentProjectName()`.
- `skillgrid-ui/src/features/decisions/DecisionsPage.tsx` — default-exported route
  component: pending inbox + answered/all tabs, 4s polling `setInterval` (no WS/SSE),
  answer → POST then reload.
- `skillgrid-ui/src/features/decisions/DecisionCard.tsx` — `DecisionCard` + exported
  `Decision` / `DecisionContent` / `DecisionOption` types. Lettered options,
  recommended highlighted, optional note, Submit. Re-answer allowed.
- `skillgrid-ui/src/features/decisions/DecisionsPage.test.tsx` (TICKET-03, refined in T04)
- `skillgrid-ui/src/features/decisions/DecisionCard.test.tsx` (TICKET-03)

### Modified
- `skillgrid-ui/src/app.tsx` — added `decisionsRoute` (lazy) + registered it in
  `routeTree.children`. The pre-existing in-flight `handoff` route hunk lands in the
  same commit (expected, single size:exception).
- `skillgrid-ui/src/components/layout/AppLayout.tsx` — added `{ to: '/decisions', label: 'Decisions' }`
  to `STANDALONE_ITEMS`. Pre-existing handoff nav hunk also lands (expected).
- `skillgrid-ui/vite.config.ts` — added vitest `test` block: `environment: 'jsdom'`
  (RED commit) + `globals: true` (impl commit), and `/// <reference types="vitest/config" />`
  at the top.
- `skillgrid-ui/src/vite-env.d.ts` — added `/// <reference types="vitest" />` so the
  test files typecheck under `tsc -b`.
- `skillgrid-ui/package.json` / `package-lock.json` — added devDeps
  `jsdom`, `@testing-library/react`, `@testing-library/dom` (RED commit).

---

## Decisions made

1. **Test infrastructure (minimum, noted as allowed).** The UI had NO React component
   tests — all pre-existing tests were pure-logic unit tests running under the default
   node env, and neither `jsdom` nor `@testing-library/*` were installed. To render
   and act on the React components the brief's tests require, I added the minimum:
   devDeps `jsdom`, `@testing-library/react`, `@testing-library/dom`; a `test` block in
   `vite.config.ts` (`environment: 'jsdom'`, `globals: true`); and a vitest client
   type reference. `globals: true` is required for testing-library's automatic
   `afterEach(cleanup)` (it resolves the cleanup hook via a global import). The
   `/// <reference types="vitest/config" />` + `vitest` reference make the `test`
   key and test globals typecheck under the project's strict `tsc -b`. All pre-existing
   tests still pass under jsdom (31/31).

2. **Auth token on the write.** The brief says the answer POST is "write-gated when
   SKILLGRID_HTTP_TOKEN is set — the dashboard passes the token as Bearer exactly like
   the existing memory write routes." I checked the actual code: the memory/mnemonic
   `write()` helper does NOT currently attach a token — the token-gated pattern lives
   in the **kanban/tracker** write routes (`skillgrid-ui/src/features/kanban/api.ts`),
   which read `localStorage['skillgrid.httpToken']` and set
   `Authorization: Bearer <token>`. I mirrored that exact convention (same storage key,
   same header) in `decisions/api.ts` and attached it only to the mutating
   `answerDecision` request, leaving the read path open. This is the closest existing
   "authenticated POST" and matches the brief's intent.

3. **Interaction model (smallest reasonable, brief-compatible).** The blueprint
   sketch had the card call `onAnswer` on option *click*; the brief's acceptance and
   task description call for "pick an option + optional note → submit → POST". I
   implemented the brief's model: options are selectable (`role="radio"`,
   `aria-checked`), and a single **Submit** button POSTs the effective choice
   (selected option, else the stored answer, else the recommended default) with the
   note. Re-answer is allowed on an answered decision (the button relabels "Update
   answer") and re-submits; the backend appends a version. This satisfies both the
   "clicking a card opens the detail with option selection + note + submit" and
   "a re-answer is allowed" constraints while keeping the card a single self-contained
   detail pane (the inbox list is the card list — no separate detail route was
   required by the brief's file list).

4. **`visual` handling (P3-ready, no iframe now).** `DecisionContent.visual?: string`
   is part of the parsed type. In this task the card renders NO `<iframe>`; when
   `content.visual` is set it shows a dashed "Visual prototype available (renders in
   P3)" placeholder. The "no iframe for a decision without visual" assertion is
   covered by a dedicated test; P3 adds the sandboxed iframe to `DecisionCard`.

5. **Polling cadence.** The blueprint Task 3 specifies `setInterval(load, 4000)`; I use
   `REFRESH_MS = 4000`, re-fetch on every answer submit, and no WebSocket/SSE.

---

## Verification checklist

- [x] `cd skillgrid-ui && npx vitest run src/features/decisions` → 8 passed (2 files)
- [x] `cd skillgrid-ui && npx vitest run` (full) → 31 passed (5 files)
- [x] `cd skillgrid-ui && npm run build:check` → tsc + vite + bundle budget OK
- [x] Decisions view is a lazy chunk (`DecisionsPage-*.js`)
- [x] Answer POSTs to `/mnemonic/decisions/{id}/answer` with `{optionId, note}`
- [x] On success the card shows `state=answered` + the stored answer
- [x] Re-answer re-submits (UI just re-submits; backend appends a version)
- [x] A decision without `visual` renders no iframe
- [x] Nav "Decisions" + `/decisions` route added alongside the pre-dirty handoff hunk

---

## Concerns

- **Handoff hunks piggyback.** As instructed, `app.tsx` and `AppLayout.tsx` were
  pre-dirty with the unrelated in-flight handoff feature; both files are committed
  whole (including the handoff route/nav), landing them in this commit. This is the
  expected single size:exception. No other pre-existing dirty file was staged.
- **No live-browser verification.** Verified via unit/component tests + build; I did
  not open the live dashboard (no running server was required by the brief, and the
  "do not build after the final commit" constraint was respected — the last
  `build:check` ran before the impl commit was finalised).
- **Answerer identity.** The backend records `updatedBy = user:<answerer>` and reads
  `?answerer=` from the query (defaulting to `unknown`). The UI does not currently
  send an `answerer` param, so stored answers will be attributed to `user:unknown`.
  This is backend-default behavior and not exercised by the P2 acceptance criteria
  (which only assert the stored answer + state). If attribution matters, a small
  follow-up (e.g. persist an answerer name in Settings) would set it. Not blocking P2.
- **Token not in Settings for decisions.** The `httpToken` is shared with the
  tracker/kanban write path via the same `localStorage` key, so a user who set the
  token for writes already has it; no new Settings UI was required by the brief.
