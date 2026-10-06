# Context Harness owns the session context lifecycle

---
status: "accepted"
supersedes: none
date: 2026-10-06
---

## Context and Problem Statement

Mnemonic's `session_inject` package is a pure retrieval-and-render engine: it feeds a session's prior context into the current one via Layer-1 auto-prepend on resume and Layer-2 on-demand hybrid retrieval. It does not capture anything. Large tool output (a `grep -r` across the repo, a `Read` of a 2,000-line file, a `WebFetch` of a dense article) still flows into the agent's context raw, which is the exact context-bloat problem context-mode v1.0.169 was built to solve.

The natural question is where the new context capabilities live. Options: a sibling package next to `session_inject`, a new top-level component, or absorption. Absorption is the cleanest: `session_inject` is already the seam that owns "what context does this session get." Adding capture, structured query, routing, and the `ctx` CLI as siblings would split the context lifecycle across packages that all need the same FTS store and the same token-cost model. A single owner keeps the lifecycle coherent and gives the `ctx` CLI one place to talk to.

The absorption is a boundary change: `session_inject`'s public functions (`AutoPrepend`, `HybridRetrieve`, `RenderContextBlock`, `EstimateTokens`, `ObservationInjectable`) are imported by `loop_cmd.go`, `secondbrain/ask.go`, and the `mem_inject_session` MCP tool. They must not churn.

## Considered Options

- New top-level component `context_harness` alongside `session_inject`, with `session_inject` unchanged as a peer
- Absorb `session_inject` into `context_harness`: the new package owns retrieval, render, capture, query, routing, and the `ctx` CLI; `session_inject`'s functions move but keep their signatures
- Absorb with a thin `context_harness` facade that delegates to `session_inject` unchanged (no file moves)

Chosen option: "Absorb `session_inject` into `context_harness`," because the context lifecycle (retrieve, render, capture, query, route, CLI) is one concern, and splitting it across packages forces every new capability to import the retrieval engine rather than own it. The public functions move to `context_harness` but keep their signatures so the importers (`loop_cmd.go`, `secondbrain/ask.go`, `mcp/tools_session_inject.go`) only change their import path, not their call sites.

- The `mem_inject_session` MCP tool is preserved verbatim: same name, same output shape, same behavior. The tool's handler changes its import from `session_inject` to `context_harness`; the tool contract does not change.
- `context_harness` owns four new capabilities: (1) intercept-and-abstract capture into a session-scoped FTS5 `tool_outputs` sandbox store, (2) a structured `ctx_query` over the code index (counts, lists, existence checks), (3) a `Context Routing` block injected by `skillgrid prime`, and (4) the `ctx` CLI (`stats`, `index`, `search`, `purge`).
- The capture path gates on **actual** output size in the existing PostToolUse capture seam (`tool-call-capture.js` → `POST /sessions/{id}/tool-calls` → `http/toolcalls.go`), not on a PreToolUse prediction. The current route carries only `content_preview` (truncated to 500 chars); the addition is: when actual output > threshold (default ~4KB) and no `SKILLGRID_CTX_BYPASS`, store the full output to `tool_outputs` and return a 200-char summary + `ctx_search` pointer to the agent. Bypass forces the full text through.
- The `tool_outputs` table is created by migration `050_tool_outputs.sql` (session-scoped, purged at session end).
- `Context Routing` is a ~80-word tool map in the SessionStart block (via `skillgrid prime`), mapping intent to the right tool: counts/lists → `ctx_query`, retrieve from captured output → `ctx_search`, proactive index → `ctx index`, memory → `mem_*`. It is advisory, not blocking.

### Consequences

- Good, because the context lifecycle is one owner: retrieval, render, capture, query, routing, and CLI all share the FTS store, the token-cost model, and the privacy filters.
- Good, because `session_inject`'s public contract is preserved: importers change their import path, not their call sites. The `mem_inject_session` MCP tool is unchanged.
- Good, because the capture gate is on actual output, not prediction — no false-positive blocks on small outputs, no false negatives on large ones.
- Bad, because the absorption moves ~250 LOC and its tests into a new package; the import churn in `loop_cmd.go`, `secondbrain/ask.go`, and `mcp/tools_session_inject.go` is mechanical but non-zero.
- Bad, because `context_harness` becomes a wide package (retrieval + render + capture + query + routing + CLI). The internal package boundary must stay clean: the `ctx` CLI and the MCP tool talk to the harness, not to the FTS store directly.
