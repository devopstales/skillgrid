# Tasks: compact-search-output — Token-Efficient Search Output

> **STATUS:** `stalled` (2026-09-17, draft) — Flagged stalled 2026-09-21: `FormatCompact` + `context`/`unfold` params not yet implemented in the hybrid/MCP layers.
>
> **For agentic workers:** REQUIRED SUB-SKILL: use subagent-execution (or simple-execution) to implement step-by-step.

**Goal:** Add a `context` boolean parameter to `code_hybrid_search` and `code_search` MCP tools. When `context: true`, return token-efficient results (name + file + lines + signature, no source bodies). Target: ≤250 tokens for 10 hits vs ~2,000+ with bodies.

**Spec:** `.skillgrid/specs/2026-09-17-compact-search-output/briefing.md`

**Research:** Five production tools validate the pattern (mnemo 48★, context-mode 20K★, headroom 72K★, CTX 338★, mcp-injector 2★). See `.skillgrid/specs/2026-09-17-sqlite-ai-code-indexing/findings.md` §7.

---

## Delivery Strategy

| Field | Value |
|-------|-------|
| Estimated changed lines | ~400 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | n/a |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: n/a
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | `FormatCompact` in hybrid layer + `context`/`unfold` param on MCP tools | PR 1 | `go test ./internal/mnemonic/hybrid/ -run TestCompactFormat` + `go test ./internal/mnemonic/mcp/ -run TestCompactSearch` | Hybrid search with seeded FTS + vector cache; MCP server with seeded store | Revert `FormatCompact` in rank.go + the `context`/`unfold` param wiring in tools_code_hybrid.go + tools_code_search.go |

## Tickets

### TICKET-01 — `FormatCompact` in the hybrid layer (deterministic)

- **Scope:** Add a `FormatCompact(hits []HybridHit) string` function in `hybrid/rank.go` that formats each hit as a single line: symbol hits as `name (file:line_start-line_end) — signature`, chunk hits as `file:line_start-line_end — first 80 chars`. No source bodies. Return the lines joined by `\n`. The function must be deterministic: same input → same output, no timestamps, no map iteration order, no rand. Hits arrive already sorted by RRF rank, so ordering is stable.
- **Acceptance:**
  - `FormatCompact` on 5 symbol hits returns 5 lines, each matching `name (file:line_start-line_end) — signature`
  - `FormatCompact` on 5 chunk hits returns 5 lines, each matching `file:line_start-line_end — first 80 chars`
  - `FormatCompact` on 10 mixed hits produces ≤250 tokens (verified by `len(output)/4` heuristic)
  - `FormatCompact` does not include any source body text
  - `FormatCompact` is deterministic: calling it twice on the same input produces byte-identical output
  - `go test ./internal/mnemonic/hybrid/ -run TestCompactFormat` passes
  - `go test ./internal/mnemonic/hybrid/ -run TestCompactFormatDeterministic` passes
- **SATISFIES:** compact-format-symbol, compact-format-chunk, compact-token-budget, compact-no-body, compact-deterministic
- **Files:** `hybrid/rank.go` (new `FormatCompact` function), `hybrid/rank_test.go` (new test)
- **Size:** ~150 (S)
- **Blocks:** TICKET-02
- **Blocked by:** none

### TICKET-02 — `context` + `unfold` parameters on `code_hybrid_search` and `code_search`

- **Scope:** Add a `context` boolean parameter and an optional `unfold` string array parameter to the `code_hybrid_search` and `code_search` MCP tool handlers. When `context: true`, call `FormatCompact` on the results. If `unfold` is non-empty, match the flagged paths/globs against the hit files and replace the compact line with the full source for the matched hits (all other hits remain compact). When `context: false` or omitted, return the current full format (unchanged).
- **Acceptance:**
  - `code_hybrid_search` with `context: true` returns compact format (name + file + lines + signature, no body)
  - `code_hybrid_search` with `context: false` (or omitted) returns the current full format (unchanged)
  - `code_search` with `context: true` returns the same compact format
  - `code_search` with `context: false` (or omitted) returns the current full format (unchanged)
  - Compact response for 10 hits is ≤250 tokens
  - Full response for 10 hits is unchanged from current behavior (regression test)
  - `code_hybrid_search` with `context: true` and `unfold: ["src/auth/handler.go"]` returns full source for that file, compact format for all other hits
  - `unfold` with a glob pattern (`**/*_test.go`) matches multiple files
  - `go test ./internal/mnemonic/mcp/ -run TestCompactSearch` passes
  - `go test ./internal/mnemonic/mcp/ -run TestCompactSearchUnfold` passes
- **SATISFIES:** compact-search-hybrid, compact-search-fts, compact-token-budget-mcp, full-search-unchanged, compact-unfold-single, compact-unfold-glob
- **Files:** `mcp/tools_code_hybrid.go` (add `context` + `unfold` params, call `FormatCompact` + unfold logic when set), `mcp/tools_code_search.go` (add `context` + `unfold` params, call `FormatCompact` + unfold logic when set), `mcp/tools_code_hybrid_test.go` (new test)
- **Size:** ~250 (S)
- **Blocks:** none
- **Blocked by:** TICKET-01 (needs `FormatCompact` to exist)

## Dependency Graph

```mermaid
graph LR
    T01[TICKET-01: FormatCompact] --> T02[TICKET-02: context param on MCP tools]
```

## Execution Order

- **Wave 1:** TICKET-01 (FormatCompact — establishes the compact format function in the hybrid layer)
- **Wave 2:** TICKET-02 (context param — wires FormatCompact into the MCP tool handlers)

> **Acceptance-first (BDD is always on):** a ticket's failing acceptance
> scenario (its `SATISFIES` scenario) is written and confirmed RED *before*
> the implementation that makes it green.

## Slicing Notes

- **Why TICKET-01 blocks TICKET-02:** TICKET-02 calls `FormatCompact`, which
  TICKET-01 creates. Without the format function, the tool handlers have
  nothing to call. Splitting means the format logic is tested in isolation
  (hybrid layer, no MCP dependency) before it's wired into the tool handlers.

- **Why not a single ticket:** the format function and the parameter wiring
  are in different packages (`hybrid` vs `mcp`) with different test harnesses.
  The hybrid-layer test seeds `[]HybridHit` values directly; the MCP test
  calls the tool handler end-to-end. Keeping them separate means each test
  file is independent and each ticket's acceptance criteria are focused.

- **Compact format design (from mnemo + CTX + mcp-injector):** mnemo's
  `--context` mode outputs `path:line — one-line summary` per hit, no bodies.
  CTX's `php-signature` modifier extracts API surfaces (signatures, no bodies).
  mcp-injector's AST body folding "strips out function bodies while preserving
  signatures" — exactly our `FormatCompact`. Our equivalent is
  `name (file:line_start-line_end) — signature` for symbol hits,
  `file:line_start-line_end — first 80 chars` for chunk hits. The agent gets
  "what exists where" in ~25 tokens/hit vs ~200+ tokens/hit with source bodies.
  The drill-in path is `code_read` with the file + line range, or the inline
  `unfold` parameter for a single-response drill-in.

- **Determinism (from mcp-injector):** mcp-injector guarantees byte-identical
  outputs across runs to maximize Anthropic's KV prompt cache hits. Our
  `FormatCompact` must be equally deterministic: same input → same output,
  every time. No timestamps, no map iteration order (Go maps are non-
  deterministic), no rand. Hits arrive already sorted by RRF rank, so ordering
  is stable. If we ever add a `map[string]` intermediate, we must sort the
  keys before formatting. The test `TestCompactFormatDeterministic` calls the
  function twice and asserts byte-identity.

- **`unfold` parameter (from mcp-injector's `unfolded_files`):** mcp-injector's
  `get_project_map` takes `unfolded_files` — an array of paths/globs served at
  full resolution within the compressed response. Our `unfold` parameter does
  the same: the agent passes `unfold: ["src/auth/handler.go", "**/*_test.go"]`
  and those files come back with full source while everything else stays
  compact. It's a slice parameter on the existing tool, not a new tool. The
  implementation: after `FormatCompact`, iterate the `unfold` list, match
  against the hit files (exact path or glob), and replace the compact line
  with the full source for the matched hits. ~30 lines of code.

- **Token budget verification:** the test asserts the compact output for
  10 synthetic hits is ≤250 tokens. Token count is estimated with a
  simple `len(output)/4` heuristic (≈1 token per 4 chars for English/code) —
  good enough for the budget check, not a real tokenizer. Note: the `unfold`
  parameter intentionally exceeds the 250-token budget for the unfolded files
  (that's the point — the agent opted in to full source for those files).
