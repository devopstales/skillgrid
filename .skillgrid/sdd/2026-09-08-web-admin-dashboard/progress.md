# SDD ledger — plan: .skillgrid/specs/2026-09-08-web-admin-dashboard/tasks.md (Phase 1: embed-pipeline-shell)

## Preflight

- Plan: `.skillgrid/specs/2026-09-08-web-admin-dashboard/tasks.md`, Phase 1 section (tasks 1.1–1.6).
- Spec (binding authority): `.skillgrid/specs/2026-09-08-web-admin-dashboard/briefing.md` + `acceptance.feature` (@phase-1 scenarios).
- Worktree: none — user consented to work in place on `release/2`.
- Baseline: `go test ./...` in skillgrid-cli → 1 pre-existing FAIL `TestSkillgridEvalSelfCorpus` (cmd/skillgrid): the test's fixture git repo is on branch `main`, and the repo's global pre-commit hook (core.hooksPath=dotfiles template) refuses commits on protected branches. Unrelated to this change; `internal/mnemonic/http/...` is green.

## Rulings

- Ruling: commit directly on `release/2` — user chose "work in place on release/2"; the repo's global hooksPath shim (gitleaks-only) does not enforce the skillgrid protected-ref guard, and release/2 is the delivery branch for this change. Cost if wrong: phase commits sit on the release branch (revertable per work-unit commit).
- Ruling: plan's `ui.go` (Modify) is a stale path — `ui.go` was deleted in a7afd106 (vanilla dashboard removal). The Phase 1 SPA-serve code goes in NEW `embed.go` (+ test `embed_test.go`), with route registration added in `server.go` (`registerRoutes`). Tasks 1.3/1.4 are merged into that one RED task. Cost if wrong: reviewer flags file-layout drift vs plan text; plan's own Files list already creates embed.go.
- Ruling: the old swagger assets were deleted in a7afd106, so `/swagger/` + `/openapi.yaml` are currently DEAD — the DoD says "still load", which requires restoring them: `ui/openapi.yaml` restored from a7afd106 (1990 lines, documents all existing routes) and a local swagger-ui bundle restored under `ui/swagger/`, both committed (no CDN at runtime — Global Constraint). Cost if wrong: ~2MB of static assets in the repo (same as before a7afd106).
- Ruling: `GET /` serves `index.html`; the mux is not a fallback mux — API-prefix 404s come from explicit catch-all 404-JSON routes for API prefixes, and a final catch-all serves the SPA `index.html`. This satisfies "SPA fallback serves index.html for non-API/non-asset routes; assets + API routes are not shadowed". Cost if wrong: catch-all ordering must be audited per new route group in later phases.
- Ruling: Vite `base: './'` + relative asset refs so the embedded SPA works regardless of mount prefix; content-hash filenames per plan.
- Ruling: shadcn/ui is deferred — Phase 1 needs plain layout components (sidebar, page shell); shadcn init lands in the first phase that actually uses a shadcn component (avoids ~30 unused files). Cost if wrong: Phase 2+ pays a small one-off init step.
- Ruling: task-brief/review-package scripts expect "Task N" headings; tasks.md uses `## N-slug` + numbered checkboxes, so briefs are hand-extracted to the workspace (script limitation, documented).

## Preflight conflict scan

| Tasks | Shared file/interface | Produces vs consumes | Found |
|---|---|---|---|
| 1.1 (scaffold) → 1.2 (shell) → 1.5 (stubs) | skillgrid-ui/** | scaffold defines package.json/vite config; shell consumes it | clean |
| 1.3/1.4 (embed + fallback) | embed.go, server.go, embed_test.go | 1.3 creates embed.go + route wiring; 1.4 extends the same tests/handlers | MERGED (ruling above) |
| 1.6 (Taskfile/.gitignore) | Taskfile.yml, .gitignore | consumes 1.1 outDir | clean |
| 1.2 ↔ 1.5 | src/features/* | 1.2 creates placeholder routes; 1.5 verifies stub behavior (tsc) | clean |

## Task ownership

- Task 1.1: Owns: `skillgrid-ui/**` (package.json, vite.config.ts, tsconfig, index.html, src/**) — Needs: none
- Task 1.2: Owns: `skillgrid-ui/src/app/**`, `skillgrid-ui/src/components/layout/**`, `skillgrid-ui/src/features/**` — Needs: 1.1
- Task 1.3+1.4: Owns: `skillgrid-cli/internal/mnemonic/http/embed.go`, `embed_test.go`, `server.go` (registration only), `skillgrid-cli/internal/mnemonic/http/ui/**` (openapi.yaml, swagger/) — Needs: 1.1 (dist must exist)
- Task 1.5: Owns: `skillgrid-ui/src/features/**` (stub polish), verification only — Needs: 1.2, 1.3+1.4
- Task 1.6: Owns: `Taskfile.yml`, `.gitignore` — Needs: 1.1

## Apply log

### Phase 1 — embed-pipeline-shell (PASS, 2026-09-16)

Commits (on `release/2`, work in place per ruling):
- `6b835e2` chore(ui): track skillgrid-ui source tree in git
- `fe94784` feat(ui): scaffold Vite/React/TS/Tailwind SPA (shell + 9 stub routes, dark theme, project selector)
- `a893b1e` feat(ui): restore embedded openapi.yaml + swagger-ui assets
- `3df0ad0` feat(ui): embed SPA shell via go:embed with swagger/openapi + SPA fallback (`embed.go`, `embed_test.go`, `server.go` wiring, `serve --help` text fix)
- `2540426` feat(ui): Taskfile ui:build + build:all for the go:embed pipeline

Tasks 1.1–1.6 all `[x]`; Phase 1 DoD all `[x]`; Verification Verdict `PASS`.

Evidence:
- `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase1_'` → PASS (`TestPhase1_Embed`, `TestPhase1_SPAFallback`).
- `go test ./skillgrid-cli/internal/mnemonic/http/...` → PASS (full package, 26.9s).
- `cd skillgrid-ui && npx tsc --noEmit && npm run build` → PASS (tsc clean; `ui/dist/index.html` + `assets/` content-hashed).
- Live smoke (`skillgrid serve` :7438 + curl): `/` 200 SPA shell; `/tracker` 200 SPA fallback; `/docs` 200 SPA; `/openapi.yaml` 200 yaml; `/swagger/` 200 + `/swagger/swagger-ui.css` 200 css; `/memory/nope` 404 JSON; `/projects` 200 JSON; `/assets/index-*.js` 200 js; `/docs/changes` 200 API.
- `task ui:build` + `task build:all` → PASS.

Implementation notes (deviations from plan text, all justified):
- `embed.go` (new) instead of plan's `ui.go` Modify — `ui.go` was deleted in a7afd106 (see Rulings).
- Go 1.22 `ServeMux` pattern conflicts resolved: `GET /tracker` + `GET /docs` need EXACT routes because the mux clean-path rule 307-redirects bare `/tracker`→`/tracker/` before the `{rest...}` wildcard fires. `GET /` handled by the final `GET /{rest...}` fallback (no explicit `GET /` needed). `/swagger` vs `/swagger/{file}` coexist (exact + wildcard); `/swagger/` exact serves the index.
- API-prefix 404 catch-alls (`GET /<prefix>/{rest...}` for sessions, memory, observations, code, web, projects, prompts, relations, context, health, tracker, docs) keep unknown API tails as 404 JSON; the final `GET /{rest...}` serves the SPA shell for everything else.
- Test `firstDistJS` walks `fs.Sub(uiDistFS, "ui/dist/assets")` so returned paths are relative to that root — the request URL is `/assets/<name>` and the readback path is `ui/dist/assets/<name>`.
- Test uses `/memory/nope` (not `/mnemonic/nope`) for the API-prefix 404 case: `/mnemonic` has no registered API prefix in this codebase (that's a future phase), so it correctly falls through to the SPA shell.

Pre-existing WIP preserved (stashed for the test run, restored after): `skillgrid-cli/internal/mnemonic/hybrid/{rank.go,vectorcache.go}`, `skillgrid-cli/internal/mnemonic/codeindex/indexer.go`, `docs/user-guide/05-memory-and-indexing.md`, untracked `skillgrid-cli/cmd/vecbench/`. Unrelated to this change; `hybrid/` does not compile until that WIP lands (rank.go:461 `loadSymbolCache` undefined) — blocked full-module `go test ./...` during the run, http package tested in isolation.
