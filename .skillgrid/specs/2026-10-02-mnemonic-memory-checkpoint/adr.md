# ADR Review Manifest

- **Change:** `2026-10-02-mnemonic-memory-checkpoint`
- **Status:** completed
- **Review date:** 2026-10-02

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0005-trust-boundary-mcp-spawn-model.md` — the agent writes memory only through the MCP tools it already has; the claim route adds a read of session state and a claim timestamp, no new write surface for content.
- `.skillgrid/artifacts/04-adr-0011-observations-are-bitemporal.md` — checkpoint observations go through `mem_save`, so the Add/Update/Noop classification and the bi-temporal columns apply unchanged.
- `.skillgrid/artifacts/04-adr-0013-repo-source-of-truth.md` — the Memory Index is a convenience view of the store; the committed spec artifacts remain authoritative for project state.
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — the index uses the existing `mem_get_observation` / `mem_timeline` / `mem_search` surface; no new query tools.
- `.skillgrid/artifacts/04-adr-0018-mem-search-additive-signals.md` — the index does not rank by search score; it is recency plus pinned-first, so the additive-signal contract is untouched.
- `.skillgrid/artifacts/04-adr-0019-decisions-are-files.md` — this manifest holds pointers only.
- `.skillgrid/artifacts/04-adr-0020-sdd-ledger-owns-execution.md` — checkpoint observations are session memory, not execution coordination; the ledger is not written by the checkpoint.
- `.skillgrid/artifacts/04-adr-0021-pre-tool-policy-fail-open.md` — the same fail-open posture: no server, timeout, or malformed answer means no prompt.

## New Durable ADRs Created

- `.skillgrid/artifacts/04-adr-0022-host-agent-memory-checkpoint.md` — the host agent is the memory observer; checkpoints are gated by the server (min events, cooldown, loop guard); the external-LLM seams stay unwired.

## Supersessions

- None.

## Locked constraints touched

- "No new dependencies without an ADR" — none added. The OpenCode/Kilo plugin uses the plugin API the harness already loads; the server uses stdlib.
- "Serial development: one change at a time" — the user overrode it for this change on 2026-10-02 ("start execution right after the spec is committed") while `2026-10-02-mnemonic-webui-rewrite` is at QA (blocked, external).
- "Session-inject privacy: tag-by-default" — extended: Private Spans are removed at store time, not only at inject time.
