# P2 Review Verdict — TICKET-03 + TICKET-04 (Decisions dashboard view)

Reviewed: b7e39a4 (RED) + 76ec335 (impl). Method: read full diff, opened changed files,
ran the suite myself (8/8 decisions, 31/31 full), `tsc -b` clean, confirmed dist chunks,
cross-checked server auth + kanban convention.

## Spec compliance

| Constraint | Status | Evidence |
|---|---|---|
| Lazy chunk; build:check passes | ✅ | `app.tsx:103` `lazyPage(() => import('./features/decisions/DecisionsPage'))` — dynamic import via shared `lazyPage` (app.tsx:11); `dist/assets/DecisionsPage-BB_MUzOW.js` present; `tsc -b` exit 0 (verified) |
| Polling, not push (4s) + refresh on answer | ✅ | `DecisionsPage.tsx:12` `REFRESH_MS = 4000`; `:31` `setInterval`; `:38-40` `onAnswer` → `answerDecision` then `load()`. No WS/SSE anywhere in feature |
| Card: question + lettered options + recommended highlight | ✅ | `DecisionCard.tsx:93` lettered badge `String.fromCharCode(65+i)`; `:102` `recommended` class + badge. Test asserts `className` contains `recommended` |
| Answer = option + optional note → POST; on success state=answered + stored answer | ✅ | `DecisionCard.tsx:69-73` `onAnswer(id,{optionId,note})`; `api.ts:65-77` POST `…/answer` body `{optionId,note}`; answered state renders note + updatedBy (`DecisionCard.tsx:129-136`), "your choice" badge. Verified by DecisionsPage.test "shows a stored answer" |
| `content.visual` in parsed type; no iframe; no crash when set | ✅ | `DecisionCard.tsx:26` `visual?: string`; `:71` `hasVisual`; `:138-142` dashed placeholder, no `<iframe>`; test asserts `querySelector('iframe') === null` for non-visual. P3 owns the iframe (p3-brief TICKET-05/06) |
| Styling matches design system; no new UI dep (test-infra OK, minimal) | ✅ | Tokens `accent/edge/card/bg` exist in `index.css:4-7`; rest is zinc (repo convention). devDeps = `jsdom`, `@testing-library/react`, `@testing-library/dom` only — minimal standard set for first component tests; `globals:true` justified (auto-cleanup). `package.json` +3 lines |
| Nav "Decisions" + `/decisions` route | ✅ | `AppLayout.tsx:18` `{ to: '/decisions', label: 'Decisions' }`; `app.tsx:99-103` + registered in `routeTree` `:142` |
| app.tsx/AppLayout pre-dirty handoff hunks: no conflict with decisions | ✅ (decision) | `git diff 7cc50f4..76ec335` shows the handoff route/nav hunks ride in the same commit. They do not touch the decisions route/nav (adjacent, non-overlapping). Decisions feature files are self-contained; decisions route/nav are not blocked by the handoff feature. |
| Answer POST uses `localStorage['skillgrid.httpToken']` → Bearer — matches server? | ✅ | Server `requireWriteAuth` (`server.go:254-266`) reads `Authorization`, expects exactly `"Bearer " + token` (token = `SKILLGRID_HTTP_TOKEN` env, `server.go:41`). Route gated at `decisions.go:73`. UI `api.ts:9,29` reads the same `skillgrid.httpToken` key and sets `Authorization: Bearer <token>`, byte-identical to the canonical `kanban/api.ts:11,32`. Convention matches. |
| TDD: RED compiles + fails on missing components; GREEN passes | ✅ | See TDD evidence below. Verified GREEN by re-running: 8/8 decisions, 31/31 full. |

Token-convention answer (explicit request): **it matches.** The implementer's claim is correct — the memory write *helpers* don't attach a token, but the decisions answer *route* is wrapped in `requireWriteAuth` (decisions.go:73), which reads the Bearer header. Mirroring kanban/tracker (`skillgrid.httpToken` → `Authorization: Bearer`) is exactly what the server validates.

## Quality findings

**LOW — No end-to-end test at the fetch boundary for the answer POST** — `DecisionsPage.test.tsx:30` mocks `api.answerDecision` (function-level spy), so the real `fetch` POST (URL, `Bearer` header from `skillgrid.httpToken`, JSON body, `!res.ok` → `DecisionApiError`) is never exercised. — Why: a regression in `answerDecision`'s URL/header/body (the token wiring is the one non-obvious bit) would not be caught by the current suite; the card test only proves the `onAnswer` prop is called with the right shape. — Fix (suggested, not blocking): one test that spies `global.fetch`, calls `answerDecision`, and asserts `url === '/mnemonic/decisions/1/answer?project=…'`, method POST, body `{optionId,note}`, and — with a token in localStorage — the `Authorization: Bearer` header.

**LOW — Unverified live round-trip; no real store/httptest in the UI layer** — `p2-report.md` "Concerns" concedes no live-dashboard check and no server in the loop. — Why: the plan's runtime harness is "dashboard UI + real store"; P2 covers the UI with vitest/jsdom (fetch mocked) and the store logic lives in P1's httptest. There is a gap: nothing in P2 proves the UI's request shape actually round-trips against the P1 server. — Fix: acceptable as a P2 slice (P1 httptest + P2 component tests bracket the contract), but note it — a single Playwright/agent-browser smoke (or a P1 httptest that the UI `api.ts` is pointed at) would close it.

**LOW — Handoff feature is committed-incomplete (pre-existing, out of scope, but worth flagging to the orchestrator)** — `app.tsx:94-97` + `AppLayout.tsx:17` (landed in 76ec335) reference `./features/handoff/HandoffHubPage`, which is **untracked** (`git status` → `?? skillgrid-ui/src/features/handoff/`; `git log --all` shows it was never committed; `git cat-file` → blob not in db). It builds only because those files sit in the working tree. — Why: a clean checkout of 76ec335 (e.g. CI, a fresh clone, or a future `git worktree`) would fail to resolve that import. This is the "pre-dirty handoff feature" the orchestrator said to treat as out of scope, so it does not gate P2's decisions work — but it's a real dangling dependency now frozen into the committed app shell. — Fix (for the handoff owner, not this ticket): commit `skillgrid-ui/src/features/handoff/` (plus the `??` Go handoff files) or drop the handoff route/nav hunks from this commit.

YAGNI: clean. No speculative features — tabs (pending/answered/all) match the blueprint; re-answer allowed (backend appends a version per 7cc50f4 R6); `DecisionRow` is a thin alias of `Decision` (one line, not speculative).

Test hygiene: real render + action assertions, not "renders without crashing." `DecisionCard.test.tsx` asserts recommended highlight, absence of iframe, and exact `onAnswer` call args (option + note) on submit and re-answer. `DecisionsPage.test.tsx` asserts the inbox render, default `fetchDecisions('pending')` call, the 4s interval refetch, and the stored-answer display. The only boundary gap is the fetch-level POST (finding 1).

Lazy wiring: confirmed dynamic import (not static) — `DecisionsPage-BB_MUzOW.js` is a separate chunk in dist.

## TDD evidence

- **RED?** ✅ — b7e39a4 adds only the two test files + devDeps + `environment: 'jsdom'` (no impl). `git show --stat b7e39a4` confirms no `*.tsx` component. Report shows both test files fail with `Failed to resolve import "./DecisionCard"/"./DecisionsPage"` → `Test Files 2 failed (2)`, `Tests no tests`. That is a compile/transform failure on the missing component (not a syntax error) → "tests compile, fail on missing component" criterion met. I did not re-run RED (would require a dirty checkout to strip the impl), but the commit content + reported output are consistent.
- **GREEN?** ✅ (verified by me) — `npx vitest run src/features/decisions` → `2 passed (2) / 8 passed (8)`; `npx vitest run` (full) → `5 passed (5) / 31 passed (31)`. `tsc -b` exit 0.
- **Process clean?** ✅ — Strict RED→GREEN ordering, conventional commits, atomic (RED = tests only; impl = feature + wiring). The impl commit also *refines* `DecisionsPage.test.tsx` (+45/-13) to assert the real behavior — legitimate TDD, not test-gaming. One blemish (not TDD, see findings): the handoff hunks piggyback in the impl commit.

## Work Unit Evidence check

- **Focused test (cmd + result):** ✅ — Report + checklist give `npx vitest run src/features/decisions` (8 passed) and full (31 passed); I reproduced both.
- **Runtime harness (plan: dashboard UI + real store):** ⚠️ Partial — Dashboard UI is covered by component tests under jsdom (fetch mocked); the real store (decisions bridge read/answer) is covered by P1's httptest, not re-exercised here. The UI↔server contract (request shape + Bearer) is verified by code cross-check (matches `requireWriteAuth`) but not by a live/httptest round-trip in P2. Noted as finding 2.
- **Rollback boundary present in report?** ⚠️ Partial — The boundary exists structurally: two independently-revertable commits (revert 76ec335 → decisions UI + wiring gone; revert b7e39a4 → tests + test-infra gone), and the decisions feature is self-contained (reverting 76ec335's *decisions* hunks leaves a clean tree). However the report's "Concerns" section documents scope/verification caveats but does **not** explicitly name the rollback/revert target; and the `[skillgrid-context]` commit block (Task/Decisions/Remaining/Tried) required by the work-unit-commits protocol is **absent** from both commit bodies (subject only). The repo's own `.skillgrid/sdd/checkpoint.json` is also stale (last_commit 72dd264, from an unrelated task). These are process gaps, not correctness issues.

## Verdict: **APPROVED**

All nine global constraints satisfied with file:line evidence; TDD evidence genuine (RED content verified, GREEN reproduced); test suite is meaningful and the one test gap (fetch-boundary POST) is low-severity and easy to add. No CRITICAL or HIGH findings. The three LOW items are: (1) add a fetch-boundary POST test, (2) note the no-live-store gap, (3) flag the committed-but-untracked handoff feature to its owner. None block merging the decisions work.

Counts by severity: CRITICAL 0 · HIGH 0 · MEDIUM 0 · LOW 3.
Worst finding: handoff route/nav committed in 76ec335 points at `./features/handoff/HandoffHubPage`, which is untracked (never committed) — builds only in this dirty working tree; a clean checkout of the commit fails to resolve the import (out of P2 scope, flag to handoff owner).
