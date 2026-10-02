# Locked Constraints

Project-wide boundaries locked by the user. These override per-change decisions and are the hard limits a change must respect. A constraint is locked only when the user says so — inferred limits belong in `.skillgrid/ASSUMPTIONS.md` or an ADR, not here.

The `### Rules` section of the agent config block (`AGENTS.md`) is rendered from this file, one bullet per constraint. `state.yaml` `constraints_ref` points here.

## Locked

- Go 1.22+ minimum to build.
- No new dependencies without an ADR.
- Trivy is advisory-only — findings are reported, never blocking the QA gate.
- Serial development: one change at a time, no parallel branches.
- Conventional commits only; no AI-attribution trailers (commit-msg hook enforces).
- Spec-zone changes commit before code-zone changes (pre-commit zone guard enforces).
- Session-inject uses a two-layer mechanism: (1) auto-prepend a slim token-capped L1 summary only on resume (not fresh sessions), and (2) an on-demand `mem_inject_session` tool for deeper BM25/semantic retrieval. See `.skillgrid/artifacts/07-mnemonic-tool-surface.md`.
- Session-inject privacy: tag-by-default — secrets, full local paths, and `private`-tagged content are auto-excluded from injection; project-relative paths, commit SHAs, tool names, and task IDs are included by default.
- Session-inject scope: project-scoped by default; cross-project injection only when the agent explicitly passes `all_projects: true` (reuses the existing `mem_search` flag).
- Session-inject selection is hybrid (BM25 + semantic, RRF-fused) by default. The vector leg degrades to BM25-only when no embedder is active (reuses the existing degrade-to-Null design); BM25-only is a degraded state, not the target model.

## Notes

- A constraint that is later unlocked gets moved to a `## Unlocked (historical)` section below with the date — it is not deleted, so the boundary history stays readable.

## Unlocked (historical)

- (none yet)
