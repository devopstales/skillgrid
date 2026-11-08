# ADR Review Manifest

- **Change:** `2026-11-08-mnemonic-mcp-split`
- **Status:** in review
- **Review date:** 2026-11-08

## In-Force ADRs Reviewed

- `.skillgrid/projectwiki/decisions/04-adr-0052-mnemonic-v2-module-extraction.md` — the module this change renames: the extraction decision stays in force; the directory + module path move (`mnemonic-v2/` → `mnemonic/`, `github.com/devopstales/skillgrid/mnemonic`).
- `.skillgrid/projectwiki/decisions/04-adr-0005-trust-boundary-mcp-spawn-model.md` — the MCP-spawn trust boundary is unchanged: still a stdio process spawned by the agent harness. The split creates three spawn points (one per family) with the same trust semantics; no data-ownership change.
- `.skillgrid/projectwiki/decisions/04-adr-0031-go-version-floor.md` — Go 1.22+ floor holds; the rename + tree change use no new language feature.
- `.skillgrid/projectwiki/decisions/04-adr-0048-hybrid-sot.md` — the `.skillgrid/` Markdown bundle is the SoT for the project wiki: this change writes its ADR into `decisions/`, its LOCKED row into `ASSUMPTIONS.md`, and re-indexes via `projectwiki index` in the same edit.
- Locked constraint "No new dependencies without an ADR" — satisfied: no new Go module dependency (the split reuses the existing mcp-go + the registration seams it introduces).

## New Durable ADRs Created

- `.skillgrid/projectwiki/decisions/04-adr-0061-mcp-family-split.md` — MCP family split: three single-purpose stdio servers (`mnemonic memory mcp` / `code mcp` / `projectwiki mcp`, one mounted per client, initialize + instructions baseline-modeled on engram / openwiki / gitnexus+CodeGraph); family-first CLI tree (mem→memory, index→code, per-family `mcp` verbs, `serve` + bare `mcp` removed, no aliases); module rename `mnemonic-v2/` → `mnemonic/` (server version 0.2.0); serial stdio (`WithWorkerPoolSize(1)`); code response contract (`code_search` per-hit blast-radius + snippet, `code_explore` composite primary tool, dedup floor); store unchanged (lease + busy_timeout keep parallel family processes safe).

## Supersessions

- ADR-0061 supersedes the combined-server design decision of 2026-10-31-mnemonic-v2-code-index (blueprint T16 + spec `008-transport` §3, built on the memory change's D-PROFILE two-profile mechanism) — a design decision, not a numbered ADR (the `supersedes:` field carries the precedent wording from ADR-0033).
