# Windows — cross-change defect register

> Open WARNING/SUGGESTION findings that survived their change's QA gate.
> Appended by `skillgrid:qa` at gate render; closed by the fix's commit.

| ID | Change | Severity | Finding | Owner | Date | Status |
|---|---|---|---|---|---|---|
| W001 | 2026-11-08-mnemonic-mcp-split | WARNING | 4 pre-existing v1 suite failures in `skillgrid-cli/internal/install` (TestMemoryAndPersonasBoundary, TestCollectNPMPackages, TestInstallNPMBatchAllowScripts, TestPrePushBlocksMalformedCloses) — identical at baseline f30aea50; out of the mcp-split footprint (v1 untouched per R1) — 3 roots: opencode `tui.json` ensure dropped by the V2-logo refactor (source regression, fresh-home install crash), stale `opencode-ai` vs `@opencode/cli` test expectations, developer's global `core.hooksPath` leaking into throwaway test git repos (now hermetic) | qa-auto | 2026-11-08 | closed (0b9e65cd, 2026-11-08) |
| W002 | 2026-11-08-mnemonic-mcp-split | SUGGESTION | `state-drift-check.mjs` cannot parse the multi-segment state.yaml format (strict YAML; pre-existing — pre- and post-change content fail identically at line 9, column 26) — the drift guard is dead against the current format; make the format strict-YAML or the parser lenient | qa-auto | 2026-11-08 | open |
