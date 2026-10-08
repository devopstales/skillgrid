# ADR Review Manifest

- **Change:** `2026-10-07-replace-opencode-hooks-with-plugins`
- **Status:** in review
- **Review date:** 2026-10-08

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0008-yaml-dependency.md` — the `yaml` package is the only third-party dependency in the mnemonic module; it is used by the installer to read/write harness config files. This change **extends** its use: the installer now writes 5 plugin entries instead of the single `opencode-yaml-hooks` entry. No new dependency is introduced.
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — fail-open floors hold: the mnemonic HTTP server starts with the facts routes registered; a down store never blocks the server (the facts routes return 500 on store error, the server itself is unaffected). The native TS plugins use the same base URL resolution and fail-closed policy gate as the retired shell hooks.
- `.skillgrid/artifacts/04-adr-0027-clm-context-language-model.md` — Go owns state/decisions, Node owns the request path. The 5 native TS plugins are the Node-side request path for opencode/Kilo, replacing the shell-era hooks. The shell hooks called the same HTTP endpoints; the plugins call the same endpoints via `fetch`. No change to the Go-side state ownership.
- `.skillgrid/artifacts/04-adr-0031-go-version-floor.md` — Go 1.22+ minimum. The installer changes are pure Go within the existing module; no version floor change.
- Locked constraint "No new dependencies without an ADR" — satisfied: no new third-party dependency. The 5 TS plugins use only `@opencode-ai/plugin` (resolved at install time via `~/.config/opencode/plugin/package.json`, already present) and the platform `fetch` API. The Go facts routes use only the existing `mnemonic/internal/facts` package and the standard `net/http` mux.

## New Durable ADRs Created

- `.skillgrid/artifacts/04-adr-0032-opencode-kilo-native-plugins.md` — opencode and Kilo harnesses load 5 native TypeScript plugins (compaction, events, squad, memory, codeindex) from a single source directory `plugins/opencode/`; Kilo's installer copies each file at install time. The shell-era hooks (4 `.sh` scripts, 2 `hooks.yaml` files, 2 `skillgrid-checkpoint.ts` copies) are retired. The `opencode-yaml-hooks` plugin registration is dropped from harness config via `dropRetiredPlugins`. Shared workers (`tool-call-capture.js`, `stop-tests.js`, `gate-stop.js`) are preserved for Cursor and the git-hook Stop gates. Facts HTTP routes (`POST /facts`, `POST /facts/search`, `POST /facts/{id}/forget`, `POST /facts/{id}/decay`, `POST /facts/decay-all`) wrap the existing `mnemonic/internal/facts.Store`.

## Supersessions

- None. ADR-0008 (yaml dependency), ADR-0016 (second brain), ADR-0027 (CLM), and ADR-0031 (Go floor) remain in force. ADR-0032 adds the native-plugin harness architecture on top of them.
