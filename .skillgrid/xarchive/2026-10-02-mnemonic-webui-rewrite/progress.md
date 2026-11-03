# SDD ledger — plan: .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/tasks.md

Mode: as-built execution. The code landed as `aa0aa791` (now re-parented, same tree) before the blueprint/tasks existed; every ticket is verified against HEAD in a clean detached worktree (`/tmp/sg-webui-verify`, `skillgrid-ui/node_modules` symlinked to the main tree) because the main tree carries ~110 dirty files from other sessions.

## Pre-flight plan scan

| Pair / item | Finding | Ruling |
|-------------|---------|--------|
| T01 vs T02/T03/T04 | T01 verifies shell files; T02-T04 verify feature folders. Disjoint reads, no writes. | proceed |
| T05 vs dirty tree | The only ticket that writes. Its three files' dirty content was diff-inspected and is exactly the fix. | proceed (pathspec commit) |
| T04 precondition | Half satisfied mid-run: the spikes→prototypes rename was committed by another session (`ef32d593`, formerly `5e502b86`) but `server.go` / `app.tsx` / `AppLayout.tsx` / `PrototypesPage.tsx` / `docs_prototypes.go` / `security.go` stayed uncommitted. | halt T04 at precondition |
| T06 at HEAD | Depends on HEAD building; see T06. | blocked |

## Baselines

- Pre-rename HEAD used for T01-T03/T05 RED: `d5e01985`.
- Post-rename HEAD used for T05 GREEN and T06: `ffd83337` (`fix(ui): keep briefing task refs linked under MarkdownView`).
- History note: all commits of this change after `d5e01985` were rewritten once (`commit-tree`, identical trees/authors/dates) to drop a `Co-authored-by: Cursor` trailer the shell integration injects past the `commit-msg` guard. Old→new: `e6f30090→69bd07e9`, `82e14ed3→9b2ad1e1`, `98f37339→355ee588`, `5e502b86→ef32d593` (other session's, re-parented only), `603b9014→982c97b0`, `f5a753a8→f54bda85`, `a853c641→ffd83337`.

## Tasks

- TICKET-01 (TASK-032): complete — verified at `d5e01985`
  - G1 `npx vitest run src/components/layout` → 4 passed
  - G2 route grep → 5
  - G3 `--color-accent: #6366f1` present in `src/styles/index.css`; no green accent
  - G8 `127.0.0.1:7438` outside `apiBase.ts` → only the display label in the AppLayout footer; no `fetch(` site
  - RED not re-run (pre-`aa0aa791` state, per tasks.md acceptance-first note)

- TICKET-02 (TASK-034): complete — verified at `d5e01985`
  - G4 three matching lines in `GraphPage.tsx` (d3 force, `limit=500`, Node Inspector)
  - G5 Sigma/graphology refs → 0; d3 import → 1
  - `npm run build:check` → "Bundle-size budget OK." 298.2 kB / 400 kB, `vendor-d3` chunk emitted
  - door check (blueprint hypothesis) held

- TICKET-03 (TASK-035): complete — verified at `d5e01985`
  - G6 `rg -c '"GET /(context|context/compaction|web/status|web/search)"' server.go` → 4
  - `npx vitest run src/features/settings src/features/swagger` → 9 passed
  - `SwaggerPage` iframe src is `apiUrl('/swagger/')`
  - Finding during this ticket: `/security/trivy` is not registered at HEAD → spec revised, endpoint moved into TICKET-04 (`982c97b0`, `f54bda85`)

- TICKET-04 (TASK-036): blocked — halted at precondition
  - `git status --short | rg -c '^R  .skillgrid/spikes/'` → 0 (rename committed by `ef32d593`)
  - `git diff --quiet -- skillgrid-cli/internal/mnemonic/http/server.go` → exit 1 (still dirty; carries the prototypes + trivy routes mixed with policy/events/usage routes and CORS middleware)
  - G7 at HEAD: `/security/trivy` → 0, prototypes fetch path → 0 (RED, as the ticket predicts)
  - Owner: the session holding the trivy bridge + rename completion. Files that make HEAD consistent: `server.go`, `skillgrid-ui/src/app.tsx`, `AppLayout.tsx`, `features/prototypes/PrototypesPage.tsx`, untracked `docs/docs_prototypes.go(+_test)`, `docs/security.go(+_test)`.

- TICKET-05 (TASK-033): complete — commit `ffd83337`
  - RED at `d5e01985`: `npm test` → 87/88, `PlansPage.linkedTasks.test.tsx` misses `/tracker?task=012`
  - Fix: `linkifyTaskRefsMarkdown` in `taskLinks.ts`; `PlanDetail` feeds the briefing through it; new case in `taskLinks.test.ts`
  - GREEN at `ffd83337` (clean worktree): `npx vitest run src/features/plans` → 2 files, 11 passed
  - Pathspec-limited commit of exactly the three files; trailer stripped (see Baselines)

- TICKET-06 (TASK-037): blocked — HEAD does not build, cause outside this change
  - At `ffd83337` (clean worktree):
    - `go build ./...` → `internal/mnemonic/http/server.go:224:54: s.handlePrototypes undefined`, `:225:48: s.handlePrototype undefined`
    - `tsc -b` → `src/app.tsx(102,102): TS2339 Property 'PrototypesPage' does not exist`, `src/app.tsx(120,36): TS2307 Cannot find module './features/spikes/SpikesPage'`
  - Root cause: `ef32d593` ("rename spike to prototype and prune retired surfaces", committed from staged files only) deleted `http/prototypes.go` and `features/spikes/SpikesPage.tsx` while the files that re-point the callers stayed unstaged. Same six files as TICKET-04's blocker.
  - G10/G11 cannot be re-run meaningfully until those land. Last green floor for this change's own code: `d5e01985` + the T05 fix (npm test 88/88 equivalent via plans suite, build:check OK, `go build` both tags OK, `go vet` OK).

## Open items for QA

1. TICKET-04 endpoints (Security, Prototypes) — blocked, external owner.
2. HEAD unbuildable since `ef32d593` — not introduced by this change; fixed by committing the six files above. Verify by re-running TICKET-06 after that lands.

## Re-verification

- TICKET-04: G7 = 1 then 1 at 0bc899a6 (path=/prototypes). Nav label Prototypes.
- TICKET-06: clean worktree npm test 110, build:check 298.7 kB OK, go build and -tags ui exit 0. Main tree after d35d28d3: 122 passed.
- Characterization tests d35d28d3: GraphPage.test.tsx, api.test.ts, apiBase.test.ts.
- Gate re-rendered PASS. Open suggestions: W010, W011.
