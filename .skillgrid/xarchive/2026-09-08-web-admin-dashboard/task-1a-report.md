# Task 1 (slice A) report — Phase 1 `embed-pipeline-shell`, tasks 1.1 + 1.2

## What was implemented

- **`.gitignore` fix**: replaced `skillgrid-ui/` (whole-dir ignore) with `skillgrid-ui/node_modules/` + `skillgrid-ui/dist/` so the source tree is tracked.
- **`skillgrid-ui/` scaffold (task 1.1)**: Vite 6.4.3, React 19.1, TypeScript 5.8 (strict), Tailwind CSS v4 (via `@tailwindcss/vite`, CSS-first `@theme` — no `tailwind.config.ts` needed in v4), TanStack Router (data router, `createRouter`/`RouterProvider`).
  - `vite.config.ts`: `base: './'`, `build.outDir: '../skillgrid-cli/internal/mnemonic/http/ui/dist'`, `emptyOutDir: true`, content-hash `entryFileNames`/`chunkFileNames`/`assetFileNames` under `assets/`, dev `server.proxy` `/api` → `http://127.0.0.1:7438`.
  - **Deviation (noted per instructions)**: plan text said proxy to `:8080`; used `127.0.0.1:7438` (the real serve port, per controller).
  - No shadcn/ui (deferred per ruling); no `@fontsource` — system font stack only (`Inter, system-ui, ...` / `"JetBrains Mono", ui-monospace, ...`), declared as Tailwind theme tokens.
- **App shell (task 1.2)**:
  - TanStack Router with **all 13 routes**: `/` (redirect → `/tracker`), `/tracker`, `/mnemonic/{graph,files,memories,sessions,search}`, `/docs`, `/plans`, `/activity`, `/git`, `/prototypes`, `/settings`, plus a `/__404__` catch-all stub (so no route is ever a blank page or hard 404).
  - Sidebar nav exactly per spec: Tracker, Mnemonic group (Graph, Files, Memories, Sessions, Search), Docs, Plans, Activity, Git, Prototypes, Settings. Active-state highlighting via TanStack `useLocation`.
  - Dark theme tokens in `src/index.css` `@theme`: bg `#0a0a0b`, card `#141416`, border `#27272a`, accent `#6366f1`.
  - Every stub page = `StubPage` (view name + "Coming in Phase N" + disabled button) — Tracker→2, Docs→3, Graph→4, Files/Memories/Sessions/Search→5, Plans/Activity/Git→6, Prototypes→7, Settings→"Coming in a later phase".
  - **Project selector**: top-bar `<select>` backed by `GET /projects` (`{"projects": [...]}`), persisted to `localStorage["skillgrid.project"]` (read/write wrapped in try/catch); on fetch failure the select renders **disabled with placeholder "no server"**. This is the only data fetch in the slice.

## Verification results

| Check | Command | Result |
|-------|---------|--------|
| Typecheck | `cd skillgrid-ui && npx tsc --noEmit` | PASS (no output) |
| Install | `cd skillgrid-ui && npm ci` | PASS (esbuild postinstall ran, `npx esbuild --version` → 0.25.12) |
| Build | `cd skillgrid-ui && npm run build` | PASS — `index.html` 0.41 kB, `assets/index-vc0Oj3_I.css` 9.73 kB, `assets/index-DB2thT4n.js` 311.29 kB at `skillgrid-cli/internal/mnemonic/http/ui/dist/` |
| CDN refs | `grep -rE '(src|href)="\./?https?://' skillgrid-cli/internal/mnemonic/http/ui/dist/` | empty (exit 1) |
| Go side | `go test ./internal/mnemonic/http/...` (run from `skillgrid-cli/` — module root) | PASS: `ok .../http 12.876s`, `ok .../http/docs (cached)`, `ok .../http/tracker (cached)` |
| Smoke | `vite preview` + curl | `/tracker` → 200 SPA, hashed JS asset → 200 `text/javascript`, `/docs` fallback → 200 index.html |

Note: the task text's `go test ./skillgrid-cli/internal/mnemonic/http/...` fails when run from the repo root because the Go module root is `skillgrid-cli/` — same package, run from inside the module, all green.

## Refinement (4-pass)

1. **Implement complete**: both 1.1 and 1.2 covered; all routes, sidebar, theme, selector, redirect, 404 stub present.
2. **Expert re-read**: verified route paths match spec 1:1, nav order matches spec, `StubPage` shows view + phase label, no unused imports (tsc `noUnusedLocals` clean), TanStack `Register` module augmentation for typed `to`.
3. **Defect hunt**: found & fixed (a) `Stub` defined in `app.tsx` then also imported from `./components/Stub` → removed the local copy; (b) `NavLink` originally used `window.location.pathname` for active state (wrong under history navigation) → switched to TanStack `useLocation`; (c) localStorage read/write wrapped in try/catch (private-mode browsers); (d) fetch failure path sets `unavailable` → disabled select with "no server" placeholder; (e) stored project no longer in the fetched list → falls back to first project; (f) StrictMode double-effect handled with `cancelled` flag.
4. **Polish**: consistent `zinc` grays over the token palette, `color-scheme: dark`, focus ring on the select, aria-current on active nav link. Final outcome: all re-checks (tsc, build, grep, go test) re-run green after fixes.

## Lazy (ladder verdict)

Skipped, all per ruling: **shadcn/ui** (deferred — no component needs it yet), **@fontsource** (system stacks allowed; keeps deps minimal), **Tailwind config file** (v4 is CSS-first `@theme`), **Go embed / Taskfile / swagger restore** (1.3–1.6, next slice). Nothing skipped without a ruling or a reason.

## Checkpoint (commits)

| SHA | Subject | Context block |
|-----|---------|---------------|
| `6b835e2` | `chore(ui): track skillgrid-ui source tree in git` | Task 1.1 · Decisions: track source, ignore only deps/dist · Remaining: scaffold + app shell · Tried: n/a |
| `fe94784` | `feat(ui): scaffold Vite/React/TS/Tailwind SPA` | Task 1.1 · Decisions: shadcn deferred; `base './'`; hashed assets; proxy → 7438 (plan said 8080); system fonts · Remaining: 1.2 (landed in same commit, see below), then 1.3/1.4 embed + 1.6 Taskfile · Tried: n/a |

`bash .agents/hooks/checkpoint-state.sh snapshot` run after **each** commit (2 runs) and once more at the end — `CHECKPOINT_WRITTEN .skillgrid/sdd/checkpoint.json` every time. No files under `.skillgrid/` committed.

Note on atomicity: the instruction called for one commit for the scaffold configs and one for the app shell (`src/**`). The scaffold commit was made after the source existed, so it contains both; the `.gitignore` commit is atomic and independently revertable on its own. Flagged here rather than splitting a commit that has already passed the hooks.

## Files changed

- `.gitignore` (modified)
- `skillgrid-ui/package.json`, `package-lock.json`, `tsconfig.json`, `tsconfig.app.json`, `tsconfig.node.json`, `vite.config.ts`, `index.html` (new)
- `skillgrid-ui/src/main.tsx`, `app.tsx`, `index.css`, `vite-env.d.ts` (new)
- `skillgrid-ui/src/components/Stub.tsx`, `StubPage.tsx`, `layout/AppLayout.tsx`, `layout/ProjectSelector.tsx` (new)
- `skillgrid-ui/src/lib/projects.ts` (new)
- `skillgrid-ui/src/features/{tracker,mnemonic,docs,plans,activity,git,prototypes,settings}/*Page.tsx` (12 new)
- Generated (untracked, not committed): `skillgrid-cli/internal/mnemonic/http/ui/dist/{index.html,assets/*}` — consumed by the next slice's `//go:embed all:ui/dist`.

## Self-review findings

- Pre-existing working-tree noise (not mine, left untouched): `docs/user-guide/05-memory-and-indexing.md` modified (unstaged) — predates this slice.
- Two recent repo commits (`1df2f85`, `84481d9`) lack `[skillgrid-context]` blocks — pre-existing, not caused here.
- `npm ci` emits `allow-scripts` warnings (esbuild postinstall) — harmless; esbuild binary verified working.
- No other defects found in the 4-pass.

## Concerns

1. **Scaffold commit is not split** into configs-vs-shell as the protocol suggested (see Checkpoint note). If the controller wants the split, `fe94784` can be reset and re-split — cheap since nothing depends on it yet.
2. Go-side serving of `dist/` is the next slice: until 1.3 lands, the binary still has no `ui/` dir embedded, which is expected and green (`go test` passes).
3. The repo-root `go test ./skillgrid-cli/...` form needs `cd skillgrid-cli` first (module root) — noted so the controller's harness runs it correctly.
