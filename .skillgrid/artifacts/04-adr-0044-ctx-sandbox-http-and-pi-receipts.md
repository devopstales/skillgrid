# ADR-0044: Context-harness sandbox HTTP routes + Pi tool-output receipts

---
status: "accepted"
supersedes: none
date: 2026-10-16
---

**Supersedes:** nothing (extends ADR-0043 and the context-harness CLI surface)

## Context

The context harness (ADR-0026/0027/0028) already stores oversized tool output
in a per-project sandbox (`tool_outputs` + FTS) and fuses it with indexed code
chunks via two-leg RRF (`ctx search`). But that capability was reachable
**only through the CLI** (`mnemonic ctx search/stats/purge`) — no HTTP routes,
no MCP tools, no native extension tools. Meanwhile the Pi extension
(ADR-0043) registered read-only memory tools: oversized `read`/`bash`/`grep`
results still flooded the model context unmediated.

The pattern proven by `pi-context` (spences10/my-pi): a `tool_result` handler
stores oversized output and **replaces the model-visible content with a
compact receipt** carrying the retrieval path — the model never sees the raw
flood, and a search tool recovers what it needs snippet-first.

## Decision

**Expose the context-harness sandbox over HTTP, and give the Pi extension
receipt-based tool-output overflow + `ctx_*` tools.**

- **Five HTTP routes** on the `mnemonic serve` mux (`internal/http/ctx_sandbox.go`),
  thin transports over the existing `context_harness` package (no new storage,
  no new dependency):
  - `POST /ctx/capture` (write-auth) — stores one tool output
    (`StoreToolOutput`, `StripPrivate` applied) and returns `{sandbox_id, gated,
    reason, pointer, summary}`. The gate decision is *reported*, not enforced:
    the client already gated on the mirrored 4096-byte threshold, and the
    receipt is useful even for slightly-below-threshold output the client chose
    to send.
  - `GET /ctx/search?query=&limit=` — two-leg RRF fusion (`CtxSearch`). Fail-open:
    a store error returns an empty result set + note, not a 500 (ADR-0016).
  - `GET /ctx/get/{id}` — one full sandboxed output by primary key (the
    retrieval path receipts point at).
  - `GET /ctx/stats` — sandbox rows/bytes + indexed-chunk count.
    **Loopback-only** (403 otherwise), matching the existing ctx observability
    routes: this is a storage-metric surface, not a data surface.
  - `DELETE /ctx/sessions/{id}` (write-auth) — `PurgeSession` (fail-open: an
    absent table purges zero rows).

- **Pi extension** (`plugins/pi/mnemonic/ctx-sandbox.ts`, wired in `index.ts`):
  - `tool_result` overflow handler: text content above 4096 bytes (mirrored
    `SandboxThreshold`, hard cap 200KB shipped) is captured to
    `POST /ctx/capture` and the model-visible content is **replaced** by a
    receipt: `[skillgrid-ctx] <tool> output stored server-side (N bytes,
    sandbox_id K). Retrieve with ctx_search … or ctx_get {sandbox_id: K}.`
    + an input summary (pi-context's `summarize_tool_input` shape). Marker guard
    (`[skillgrid-ctx]` in content → skip) prevents re-indexing receipts.
  - Four tools: `ctx_search` (snippet-first FTS), `ctx_get` (full text by
    sandbox_id), `ctx_stats`, `ctx_purge` (session-scoped).
  - `promptSnippet` added to **all 20** Pi tools — without it Pi omits custom
    tools from the system-prompt "Available tools" section.
  - **Skip list:** the `ctx_*` tools never index their own output (recursion
    guard, the pi-context convention).
  - Fail-open everywhere: a dead server, missing session, or timeout keeps the
    original tool content and never blocks the agent.

- **Install:** `ctx-sandbox.ts` added to `piExtensionFiles`
  (`mnemonic/internal/setup/pi.go`) and the extension `package.json` `files`
  array. No setup change otherwise.

## Consequences

- **Positive:** Pi sessions get the same oversized-output discipline as the
  CLI harness: the context flood is replaced by a ~150-byte receipt with a
  working retrieval path. The sandbox becomes reachable from any HTTP client
  (opencode/kilo plugins and future agents can adopt the same pattern without
  Go changes). `ctx_search` gives the model recall of what a tool *output*
  said, which memory search cannot answer (observations are curated, the
  sandbox is raw output).
- **Negative:** `tool_result` content replacement is aggressive — if the
  capture POST succeeds but the Pi process dies before the model sees the
  receipt, the raw output exists only in the store (recoverable via
  `ctx_search`/`ctx_get`). `ctx_stats` is loopback-only, so a remote Pi client
  against a non-loopback `SKILLGRID_MNEMONIC_HTTP_URL` gets 403 on stats only.
  The `tool_outputs` table has no TTL on these routes (CLI `ctx purge` +
  `DELETE /ctx/sessions/{id}` own retention).
- **One-way gate:** if a second harness (opencode/kilo) adopts the receipt
  pattern, the shared 4096-byte threshold + marker + skip-list convention
  belongs in a shared plugin helper file, not duplicated per harness.
