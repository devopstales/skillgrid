# ADR Review Manifest — skill-pi

> **Change:** `2026-10-09-skill-pi`
> **Date:** 2026-10-09
> **Status:** complete

## In-force ADRs reviewed

All 30 in-force ADRs (0001–0036, excluding removed 0014/0015) were reviewed against the skill-pi design. No conflicts found. The permission layer design is additive: it extends skill-pi (a new agent) without modifying any in-force decision.

## ADRs created by this change

| ADR | File | Summary |
|-----|------|---------|
| ADR-0037 | `.skillgrid/artifacts/04-adr-0037-skill-pi-permission-extension.md` | skill-pi permission layer: 3-tier config, Tree-sitter bash matching, tool hiding, diff preview, hybrid approvals |
| ADR-0038 | `.skillgrid/artifacts/04-adr-0038-skill-pi-ticketing-backlog-adapter.md` | skill-pi ticketing: `ticket` tool (10 actions) + `/tickets` command backed by the `backlog` MCP server over stdio; no vendored tk, no direct file I/O |

## Terms added

Four new terms added to `.skillgrid/artifacts/02-technical-terms.md`:

- **Permission Extension** — the skill-pi Pi extension that gates tool calls
- **Tool Hiding** — the pre-start mechanism where deny/ask tools are hidden from the agent
- **Approval Persistence** — the in-prompt approval flow (allow once / always / session / reject / reject with feedback)
- **Ticket Extension** — the skill-pi `ticket` tool + `/tickets` command backed by the `backlog` MCP server (Backlog.md adapter)

## Notes

- ADR-0037 records two new dependencies for the skill-pi package: Tree-sitter (already used in the mnemonic code index) and `@pierre/diffs` (new). Per the "no new dependencies without an ADR" rule in AGENTS.md.
- The existing constraint in findings.md ("Pi runs with the launching user's permissions, no built-in permission system") is a description of the current state, not a locked decision. ADR-0037 adds the permission layer as a new capability without superseding any prior ADR.
