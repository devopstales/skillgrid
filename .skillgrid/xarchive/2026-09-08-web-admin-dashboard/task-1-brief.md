# Task 1 — Phase 1: embed-pipeline-shell (Vite/React SPA + go:embed)

Extracted from `.skillgrid/specs/2026-09-08-web-admin-dashboard/tasks.md` (section `## 1-embed-pipeline-shell`) with controller rulings appended. The plan file is the binding spec; where a ruling below adjusts the plan's file layout, the ruling wins (it is recorded in the ledger `.skillgrid/sdd/2026-09-08-web-admin-dashboard/progress.md`).

## Phase Goal

The embed pipeline works end-to-end and `GET /` serves the SPA shell with every view as a stub — the foundation every later phase builds on.

## Out of scope / Non-Goals (phase)

- Any view content beyond disabled stubs (belongs to Phases 2–7)
- Any new backend data route (only the SPA + preserved swagger/openapi)
- OpenAPI changes beyond the SPA + preserved swagger

## Phase Definition of Done

- [ ] `npm run build` produces `ui/dist/index.html` + `ui/dist/assets/`
- [ ] `go build` embeds `ui/dist/` and `skillgrid serve` serves the SPA at `/`
- [ ] SPA fallback serves `index.html` for non-API/non-asset routes; assets + API routes are not shadowed
- [ ] `/swagger/` + `/openapi.yaml` still load
- [ ] Sidebar renders with all routes as labeled disabled stubs, dark theme applied
- [ ] `tsc --noEmit` clean; no external CDN assets
- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@phase-1` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] No Global Constraint violated

## Files (per plan)

- Create: `skillgrid-ui/` (Vite + React + TS scaffold: `package.json`, `vite.config.ts`, `tailwind.config.ts`, `index.html`, `src/app/`)
- Create: `skillgrid-ui/src/features/*/` (placeholder route components for all 9 views)
- Create: `skillgrid-cli/internal/mnemonic/http/embed.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui.go` → **RULING: `ui.go` no longer exists (deleted in a7afd106). The SPA-serve code goes in NEW `embed.go` (+ `embed_test.go`), with route registration added in `server.go`. Do not create `ui.go`.**
- Modify: `Taskfile.yml`
- Modify: `.gitignore` (add `ui/dist/`)

## Interfaces

- Consumes: existing server (`skillgrid serve` :7438), existing `GET /projects` for the project selector
- Produces: the embedded SPA shell (sidebar + router + stubs + dark theme) that Phases 2–7 extend one view each; `//go:embed all:ui/dist`

## Tasks

- [ ] 1.1 `[AFK]` Scaffold `skillgrid-ui/` (Vite + React 19 + TS + Tailwind v4 + shadcn/ui); `vite.config.ts` `build.outDir` → `../skillgrid-cli/internal/mnemonic/http/ui/dist`, `emptyOutDir: true`, content-hash `entryFileNames`/`chunkFileNames`/`assetFileNames`; dev `server.proxy` `/api` → `:8080`
  - **RULING: shadcn/ui is DEFERRED** — Phase 1 needs plain layout components; run `shadcn init` in the first phase that actually uses a shadcn component. Do not add unused shadcn files. (Plan keeps the tech stack; this only defers the init ceremony.)
  - **RULING: set `base: './'` in vite config** so the embedded SPA works at any mount prefix (acceptance scenarios request `/tracker`, `/docs`, etc. — the client router must not depend on absolute `/assets/...` paths).
- [ ] 1.2 `[AFK]` App shell: TanStack Router with placeholder routes for all 9 views; sidebar nav (Tracker, Mnemonic▸Graph/Files/Memories/Sessions/Search, Docs, Plans, Activity, Git, Prototypes, Settings); dark theme (`#0a0a0b`/`#141416`/`#27272a`/`#6366f1`); Inter + JetBrains Mono; project selector persists to localStorage
  - Fonts: self-host or use system font stacks — **no external CDN at runtime (Global Constraint: offline binary)**. Inter/JetBrains Mono via `@fontsource` npm packages (bundled into assets) or plain system stacks; no `<link href="https://...">` in the built output.
  - Views list (9): tracker, mnemonic/graph, mnemonic/files, mnemonic/memories, mnemonic/sessions, mnemonic/search, docs, plans, activity, git, prototypes, settings — every route exists in the router; not-yet-built views render a labeled disabled stub ("Coming in Phase N"), never a blank page or 404.
- [ ] 1.3 `[RED]` Create `embed.go`: `//go:embed all:ui/dist`, `//go:embed ui/openapi.yaml`, `//go:embed all:ui/swagger`
  - **RULING (merged with 1.4):** `ui/openapi.yaml` and `ui/swagger/` were DELETED in a7afd106. Restore them under `skillgrid-cli/internal/mnemonic/http/ui/`: `openapi.yaml` from commit a7afd106 (1990 lines, documents all existing routes — run `git show a7afd106:skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml`); the swagger-ui static bundle (swagger-ui-bundle.js, swagger-ui-standalone-preset.js, swagger-ui.css, index.css, index.html, swagger-initializer.js, favicon-*.png) from a7afd106's `ui/swagger/`. Commit them (no CDN at runtime). `swagger-initializer.js` must point `url` at `/openapi.yaml` (as before, the old code rewrote the petstore URL at serve time — either pre-bake the URL in the committed file or keep the runtime rewrite; pre-baking is simpler).
  - [ ] 1.3.a Write failing test (`embed_test.go`, package `http`): `GET /` returns the SPA `index.html` (assert `<div id="root">` or equivalent + `<!DOCTYPE html>`); a real hashed asset path from `ui/dist/assets/` returns 200 with non-HTML content (not `index.html`); `GET /mnemonic/nope` (an API prefix) returns 404 JSON (`{"error":...}`), not `index.html`.
    - SATISFIES: `GET / serves the embedded SPA shell`, `Hashed JS and CSS assets are served from the binary`, `API-prefix routes are not shadowed by the SPA fallback` (@phase-1)
  - [ ] 1.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase1_Embed` — Expected: FAIL
  - [ ] 1.3.c Minimal implementation: `embed.go` with the three embeds + route registration (see 1.4.c behavior) + server.go wiring.
  - [ ] 1.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase1_Embed` — Expected: PASS
  - [ ] 1.3.e Commit — `feat(ui): go:embed Vite dist + SPA fallback`
- [ ] 1.4 `[RED]` Threat: SPA fallback does not shadow API/asset routes
  - [ ] 1.4.a Write failing test: `GET /openapi.yaml` → 200 with `application/yaml` and `openapi:` in body; `GET /swagger/` (and `/swagger/index.html`) → 200 html; a non-asset, non-API path (e.g. `/tracker`) → 200 `index.html` (client router); a known API route prefix that does not exist (e.g. `/projects/xyz`) → 404 JSON, not `index.html`.
    - SATISFIES: `swagger and openapi are preserved`, `SPA fallback serves index.html for non-API, non-asset routes` (@phase-1)
  - [ ] 1.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase1_SPAFallback` — Expected: FAIL
  - [ ] 1.4.c Minimal implementation: route rules — (1) exact GET routes for `/openapi.yaml` and `/swagger/`+`/swagger/{file...}`; (2) all existing API routes registered FIRST (they are); (3) explicit 404-JSON catch-alls for API prefixes (`/sessions`, `/memory`, `/observations`, `/code`, `/web`, `/projects`, `/tracker`, `/docs`, `/prompts`, `/relations`, `/context`, `/health`); (4) `GET /` → index.html; (5) a final catch-all → index.html (SPA fallback). Verify with a test that an existing API route (`GET /projects`) still returns JSON, not index.html.
  - [ ] 1.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/http/... -run TestPhase1_SPAFallback` — Expected: PASS
  - [ ] 1.4.e Commit — `feat(ui): SPA fallback preserves API + asset routes`
- [ ] 1.5 `[AFK]` Future views render as labeled disabled stubs, never dead links; no external CDN assets (offline check) — `Run: cd skillgrid-ui && npx tsc --noEmit` — Expected: PASS
  - Offline check: `grep -rE 'https?://' skillgrid-cli/internal/mnemonic/http/ui/dist/ | grep -v 'xmlns\|w3.org\|schema.org'` must not yield any `<script src>`/`<link href>` external asset refs (SVG xmlns namespaces are fine).
- [ ] 1.6 `[AFK]` Taskfile `ui:build` (`dir: skillgrid-ui`, `npm ci && npm run build`) + `build:all` (`deps: [ui:build]`, `go build -o dist/skillgrid ./skillgrid-cli/cmd/skillgrid`); `.gitignore` adds `ui/dist/` — `Run: go test ./skillgrid-cli/internal/mnemonic/http/...` — Expected: PASS
  - `.gitignore` note: `ui/dist/` must be ignored (the build output is regenerated by `task ui:build` before `go build` via `build:all`). Note: `//go:embed all:ui/dist` REQUIRES the directory to exist at `go build` time — `build:all` guarantees ordering; a plain `go build` without a prior `ui:build` will fail (that is intended and must be documented in the Taskfile desc).
  - Also ignore `skillgrid-ui/node_modules/` (add if not present).

## Verification (fill in at completion)

Verdict: `PENDING`

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/http/... -run 'TestPhase1_'` | PASS | — | embed + SPA fallback |
| SPA build | `cd skillgrid-ui && npm ci && npx tsc --noEmit && npm run build` | PASS | — | produces `ui/dist/` |
| Acceptance `@phase-1` / `@p0` | manual smoke: `skillgrid serve` + browser — sidebar + stubs + dark theme + swagger link | PENDING | — | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/http/...` | PASS | — | full package green |
| Rollback boundary | `git revert` + `skillgrid serve` — old viewer still works | PENDING | N/A (additive) | new `skillgrid-ui/` + `embed.go` + `ui/` static; revert restores pre-phase state |
| Global Constraints | — | held | — | offline, 127.0.1 unchanged, no `tools_*.go` touched |

## Commit

When phase DoD is met: `feat(ui): Vite/React SPA shell + go:embed pipeline (Phase 1)`

## Global Constraints (inherited — do not violate)

- MCP tool names, signatures, return shapes, error codes frozen — no `tools_*.go` touched
- SPA build is deterministic + offline-capable — no external CDN at runtime; all assets from the binary (Sigma.js-from-CDN allowed only inside the exported graph HTML file — Phase 4, not now)
- Server stays bound to `127.0.0.1` by default; no new serving env vars (except `SKILLGRID_TRACKER`)
- Each phase adds its view; not-yet-built views render as disabled stubs, never dead links
- Each backend phase documents its routes in `openapi.yaml` as it lands
- Path traversal on served repo files → 400/404 (docs phase concern, but the SPA asset server must not serve `..` paths either)

## Environment notes

- Work in `/Users/paladm/git/ai-test/skillgrid-v2` (in place on branch `release/2`, user-consented).
- Go module root: `skillgrid-cli/` (module `github.com/devopstales/skillgrid/skillgrid-cli`, go 1.26).
- Node v24, npm 11 available.
- Baseline: `internal/mnemonic/http/...` tests green. One pre-existing unrelated failure in `cmd/skillgrid` (`TestSkillgridEvalSelfCorpus` — fixture commits on protected branch `main` vs the global pre-commit hook).
- The old embedded dashboard (deleted in a7afd106) is the design reference: `git show a7afd106~1:skillgrid-cli/internal/mnemonic/http/ui/app.js` etc.
- Commit protocol: conventional subject, no AI-attribution trailer, `[skillgrid-context]` block (Task / Decisions / Remaining / Tried) in the body, then `bash .agents/hooks/checkpoint-state.sh snapshot`. Commit spec-zone (`.skillgrid/specs/`) changes before code-zone changes, never in the same commit.
- Mark phase checkboxes `[x]` in `tasks.md` as you go (spec-zone edit — commit separately from code).
