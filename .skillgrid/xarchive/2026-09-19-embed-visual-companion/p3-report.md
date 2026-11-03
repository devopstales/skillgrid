# P3 Report — Visual Companion: prototype serve route + sandboxed iframe (TICKET-05 + TICKET-06)

## Status

DONE — both tickets implemented TDD-style (RED commit → GREEN commit), all verification passing.

## Commits

| Commit | Subject |
|--------|---------|
| `4c884d2` | test(http): RED acceptance tests for prototype serve + decision card iframe (TICKET-05) |
| `1086a2b` | feat(http): prototype serve route (.skillgrid/prototype) + sandboxed iframe in decision card (TICKET-06) |

(Not pushed, per protocol.)

## Files changed

- `skillgrid-cli/internal/mnemonic/http/prototype_test.go` (new, RED) — seeded `.skillgrid/prototype/demo/a.html` + out-of-root `secret.html`; 200+`text/html`+`nosniff` happy path, traversal/absolute id table, 404 unknown.
- `skillgrid-cli/internal/mnemonic/http/prototype.go` (new) — `prototypeRoot()`, `prototypeFile()` (stitchFile guard shape, root = `.skillgrid/prototype/`), `handlePrototypeDecision` (400 invalid / 404 unknown / 200 + `text/html; charset=utf-8` + `X-Content-Type-Options: nosniff`).
- `skillgrid-cli/internal/mnemonic/http/server.go` — registers `GET /prototype/{id...}` in `registerGraphRoutes()`, right after the plural `/prototypes` routes and before `registerUIRoutes()` (SPA fallback).
- `skillgrid-ui/src/features/decisions/DecisionCard.test.tsx` — RED iframe case added (visual set → iframe with `src=/prototype/demo/a.html`, `sandbox="allow-scripts"`, no `allow-same-origin`; the pre-existing unset case already asserted no iframe).
- `skillgrid-ui/src/features/decisions/DecisionCard.tsx` — the "visual available (renders in P3)" placeholder is now a sandboxed `<iframe src={/prototype/${content.visual}} sandbox="allow-scripts">` in a bordered container, rendered only when `content.visual` is set.

## TDD evidence

### RED (commit 4c884d2)

Go — `go test -count=1 ./internal/mnemonic/http/ -run TestPrototype -v`:

```
--- FAIL: TestPrototypeServeAndTraversal (0.02s)
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http	1.756s
```

Failure mode: the unregistered `/prototype/...` path fell through to the SPA
shell fallback (`handleShellPage` → 200 index.html), and bare `..` ids 307-
redirected — i.e. no handler, behavior assertions all red. Package compiled.

UI — `npx vitest run src/features/decisions`:

```
FAIL  DecisionCard > renders the sandboxed iframe for a decision with visual
AssertionError: expected null not to be null
     56|     expect(iframe).not.toBeNull()
 Test Files  1 failed | 1 passed (2)
      Tests  1 failed | 8 passed (9)
```

### GREEN (commit 1086a2b)

```
cd skillgrid-cli && go build ./... && go test -count=1 ./internal/mnemonic/http/...
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http	26.807s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/docs	0.825s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/tracker	15.757s
```

All P1 decision tests included (whole package green).

```
cd skillgrid-ui && npx vitest run src/features/decisions
Test Files  2 passed (2)
     Tests  9 passed (9)
```

```
cd skillgrid-ui && npm run build:check
Bundle-size budget OK.
```

## Decisions

1. **Commit subjects narrowed to single scope.** The repo's `commit-msg` hook
   (`~/.aiskillgrid/git-hooks` → `.agents/hooks/checkpoint-state.sh guard-msg`)
   validates `type(scope):` with scope regex `[a-z0-9._-]+` — the verbatim
   brief subjects `test(http+ui):` / `feat(http+ui):` are rejected (`+` not
   allowed; prior P1/P2 commits used single scopes). Used `test(http):` and
   `feat(http):`; the UI portion is named in the description.
2. **Bare `..` path segments are 307, not 400, at this route.** The Go 1.22
   `http.ServeMux` clean-path rule redirects `/prototype/../a.html` (and any
   literal `..` segment) to the cleaned path *before* the handler runs — the
   redirect target lands outside the `/prototype` subtree, so the secret file
   is never served. The `%2e%2e`/`%2F`-encoded forms reach the handler and get
   400 from `prototypeFile` (verified with a standalone mux probe). The test
   table asserts the exact status per case (307/400) plus the invariant "no
   200 ever contains SECRET", mirroring the existing `prototypes_test.go`
   threat assertions. This does not invent a new traversal check — the
   `stitchFile` guard shape is reused verbatim for the `.skillgrid/prototype/`
   root.
3. **Inline iframe instead of importing `SandboxPreview`.** The brief
   sanctioned "SandboxPreview or a minimal inline `<iframe sandbox=
   allow-scripts>`". `SandboxPreview` is `srcDoc`-based (the .stitch gallery
   fetches HTML then inlines it) and device-width framed; the companion
   route serves by URL, so an inline `src`-based iframe with the same
   sandbox contract (`sandbox="allow-scripts"`, no `allow-same-origin`) was
   the smaller fit.
4. **No version-bump cache-busting on the iframe src.** There is no existing
   version-bump pattern at any `SandboxPreview` usage (the .stitch gallery
   re-fetches HTML on selection change; no `?v=` anywhere in
   `features/prototypes/`), so there was nothing to reuse. The card re-renders
   on navigation; a re-authored variant served at the same path is loaded by
   the browser's normal iframe navigation. Noted here in case a later variant
   hot-reload need arises.

## Concerns

- `server.go` was committed whole (pre-existing handoff hunks ride along,
  acceptable per protocol). Other dirty files (handoff work, docs, mcp) were
  left unstaged.
- The 307-vs-400 nuance for bare `..` ids is a mux behavior, not a handler
  rejection. If the route is ever moved behind a proxy/CDN that follows
  redirects without the mux, the guard still holds (encoded forms 400; the
  cleaned redirect targets don't exist under the root → 404), but the status
  code shape would differ. Low risk: the invariant (no escape) holds either way.
