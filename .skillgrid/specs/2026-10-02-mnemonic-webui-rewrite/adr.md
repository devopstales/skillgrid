# ADR Review Manifest

- **Change:** `2026-10-02-mnemonic-webui-rewrite`
- **Status:** completed
- **Review date:** 2026-10-02

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0005-trust-boundary-mcp-spawn-model.md` — MCP process is the trust boundary; the SPA reads only through the Go HTTP read bridges, no new write surface.
- `.skillgrid/artifacts/04-adr-0013-repo-source-of-truth.md` — repo artifacts win over session memory; this manifest and the blueprint were written against the committed code, not the chat.
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — the Second Brain chat panel from mockup v1 is out of this change; the nav drops it rather than wiring an unverified FTS path.
- `.skillgrid/artifacts/04-adr-0017-d3-force-graph.md` — the decision this change implements: D3 force graph in the SPA, Sigma/graphology removed. Authored with commit `aa0aa791`.
- `.skillgrid/artifacts/04-adr-0019-decisions-are-files.md` — one file per decision; this manifest holds pointers only.

## New Durable ADRs Created

- None in this execution pass. ADR-0017 already exists and is in force (row 0017 of the `### In-force set` table).

## Supersessions

- None.

## Locked constraints touched

- "No new dependencies without an ADR" (`ASSUMPTIONS.md § Locked constraints`) — `d3` + `@types/d3` are covered by ADR-0017.
