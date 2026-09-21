# Change: compact-search-output — Token-Efficient Search Output

> **STATUS:** `draft` (2026-09-17)
>
> **For agentic workers:** REQUIRED: follow `.agents/skills/_shared/conventions/sdd-structure.md`. This file is WHY + HOW (intent + plan). Spec phase instantiates `tasks.md` + `acceptance.feature` from the Step Blueprint and per-step WHAT below.

**Goal:** Add a `context` boolean parameter to `code_hybrid_search` and `code_search` MCP tools. When `context: true`, the response returns a token-efficient format (symbol name + file + line range + one-line signature, no source bodies) instead of verbatim source. Target: ≤250 tokens for a 10-hit response vs the current ~2,000+ tokens.

**Architecture:** A new `FormatCompact` function in `hybrid/rank.go` that takes the existing `[]HybridHit` slice and returns a compact string per hit (name + file:lines + signature, no body). The function must be deterministic (same input → same output, no timestamps, stable ordering) so the LLM's KV prompt cache hits on repeated queries. The MCP tool handlers (`tools_code_hybrid.go`, `tools_code_search.go`) accept the new `context` parameter and an optional `unfold` array (file paths or globs to serve at full resolution within the compact response). When `context: true`, they call `FormatCompact` when set, falling back to the current full format otherwise. When `unfold` is non-empty, the flagged files are served at full resolution in the same response, saving a `code_read` round-trip.

**Tech stack:** Go (`skillgrid-cli`), hybrid search layer (`internal/mnemonic/hybrid/`), MCP tools (`internal/mnemonic/mcp/`).

**Research:** Five production tools validate the compact output pattern. (1) Pilan-AI/mnemo `--context` flag (48★, v1.3.1) — outputs 5 results in ~250 tokens vs ~2,000 tokens for full output. (2) mksglu/context-mode (20,331★, ELv2, 17 platforms) — intercepts all tool output at the MCP boundary and sandboxes it (315 KB → 5.4 KB, 98% reduction). (3) headroomlabs-ai/headroom (72,104★, Apache-2.0) — AST-aware CodeCompressor trims code source (21% on 100 code search results) with CCR (cache original, retrieve on demand). (4) context-hub/generator / CTX (338★, MIT, PHP) — `php-signature` modifier extracts API surfaces (signatures, no bodies) for context documents, proving "signature-only context" is a shipped, practical pattern. (5) foldwork-dev/mcp-injector (2★, Go) — AST body folding (strip bodies, keep signatures) with 72-89% reduction on real codebases, `unfolded_files` for selective drill-in in one response, and canonical determinism for KV cache hits. All five confirm that token-efficient output is the #1 agent-context pain point. See `.skillgrid/specs/2026-09-17-sqlite-ai-code-indexing/findings.md` §7.

**Ticket:** none

**Depends on:** none (independent of the health_check/index_status spec; can run in parallel)

---

## Goal

The agent gets a cheap "what exists where?" answer first, then drills into specific files with `code_read` only when needed. Currently, `code_hybrid_search` and `code_search` return verbatim source bodies that can run thousands of tokens — the agent pays for every one, even when it only needs to know which file and line range to read next.

## Out of scope / Non-Goals

- Changing the search algorithm or ranking (RRF k=60, 3-leg fusion unchanged)
- Adding a new MCP tool (this is a new parameter on existing tools, not a new tool)
- Compact output for `code_explore` (different response shape — grouped-by-file source; can be a follow-up)
- Compact output for `code_read` (already returns only the requested line range)
- Real tokenizer for the budget check (a `len(json)/4` heuristic is sufficient for the test)

## Definition of Done

This change is done only when **all** of the following are true:

1. `code_hybrid_search` with `context: true` returns hits in compact format (name + file + lines + signature, no body)
2. `code_hybrid_search` with `context: false` (or omitted) returns the current full format (unchanged — regression test passes)
3. `code_search` with `context: true` returns the same compact format
4. Compact response for 10 synthetic hits is ≤250 tokens (verified by token count in test)
5. Full response for 10 synthetic hits is unchanged from current behavior (regression test)
6. `FormatCompact` is deterministic: calling it twice on the same input produces byte-identical output (KV cache requirement)
7. `unfold` parameter: when `context: true` and `unfold` is non-empty, the flagged files are served at full resolution within the compact response; all other hits remain compact
8. `go test ./internal/mnemonic/mcp/ -run TestCompactSearch` passes
9. `go test ./internal/mnemonic/hybrid/ -run TestCompactFormat` passes
10. No changes to the search algorithm, ranking, or FTS/vector legs

## Testing Strategy

- **Unit test (hybrid layer):** `TestCompactFormat` — seed 10 synthetic `HybridHit` values (5 symbol hits, 5 chunk hits), call `FormatCompact`, assert each line matches the expected `name (file:line_start-line_end) — signature` pattern, assert total token count ≤250 via `len(json)/4` heuristic.
- **Unit test (hybrid layer):** `TestCompactFormatDeterministic` — seed 10 synthetic `HybridHit` values, call `FormatCompact` twice, assert the two outputs are byte-identical (KV cache requirement). No timestamps, no map-iteration-order dependence.
- **Unit test (hybrid layer):** `TestCompactFormatFullUnchanged` — seed the same 10 hits, call the existing full-format function, assert the output is byte-identical to the pre-change behavior (regression).
- **Integration test (MCP layer):** `TestCompactSearch` — call `code_hybrid_search` and `code_search` with `context: true` on a seeded store, assert the response is in compact format and ≤250 tokens. Call with `context: false`, assert full format is unchanged.
- **Integration test (MCP layer):** `TestCompactSearchUnfold` — call `code_hybrid_search` with `context: true` and `unfold: ["src/auth/handler.go"]` on a seeded store. Assert the response contains the full source for `src/auth/handler.go` and compact format for all other hits.

## Compact Format Design

Modeled on five production patterns:
- **Pilan-AI/mnemo** (48★): per-tool `--context` flag that outputs `path:line — one-line summary` per hit, no bodies. Our approach — a `context` boolean parameter on existing tools — follows this pattern.
- **mksglu/context-mode** (20K★): intercept-and-abstract sandbox that keeps raw data out of context at the MCP boundary (315 KB → 5.4 KB, 98%). Our approach is less aggressive (per-tool parameter, not a global sandbox) but solves the same problem at a smaller scope. The 98% reduction is context-mode's target; our 87.5% (2,000→250 tokens) is real but less dramatic because we compact an already-formatted result set, not raw tool output.
- **headroomlabs-ai/headroom** (72K★): AST-aware CodeCompressor that trims *within* the source (keeps signatures + control flow, drops boilerplate) — 21% on 100 code search results. Our approach is more aggressive (replace the source with a *reference* to the source — name + file + lines + signature) but less sophisticated (we don't keep parts of the body). The two are complementary: headroom trims within the source; we replace the source with a pointer. headroom's 21% is the reality check: code is already dense, so *compression* yields less than *replacement*. Our 87.5% is higher because we do replacement, not compression. The CCR pattern (headroom caches the original, agent calls `headroom_retrieve` for full text) is the same as our drill-in (agent calls `code_read` with the file + line range from the compact response).
- **context-hub/generator / CTX** (338★): `php-signature` modifier that extracts API surfaces (class signatures, method signatures, interfaces) and drops the bodies. This proves "signature-only context" is a *shipped, practical pattern*, not just a theoretical one. CTX pre-generates a static document (config-driven); our compact format is per-query and on-demand (query-driven). Different timing, same transformation. CTX is PHP-specific (needs a modifier per language); our `FormatCompact` is language-agnostic (uses the signature that gotreesitter already extracted, 30+ languages).
- **foldwork-dev/mcp-injector** (2★): AST body folding (strip function bodies, preserve signatures) — "strips out function bodies while preserving signatures" is exactly our `FormatCompact`. 72-89% reduction on real Go/Python/JS codebases (Django 89.3%, Tokio 72.2%, Gin 75.8%) validates our 87.5% target. Two patterns we adopt: (1) `unfolded_files` — an array of paths/globs served at full resolution within the compact response, saving a `code_read` round-trip. (2) Canonical determinism — byte-identical outputs across runs so the LLM's KV prompt cache hits. Our `FormatCompact` must be deterministic (same input → same output, no timestamps, stable ordering).

Our equivalent:

| Hit type | Compact format | Example |
|----------|---------------|---------|
| Symbol | `name (file:line_start-line_end) — signature` | `CalculateTotal (src/cart.go:42-58) — func (c *Cart) CalculateTotal() float64` |
| Chunk | `file:line_start-line_end — first 80 chars` | `src/auth.go:112-134 — func (s *Session) Validate(token string) error { if s.ex` |

**Token budget:** ~25 tokens/hit for symbols, ~20 tokens/hit for chunks. 10 hits ≈ 250 tokens vs ~2,000+ tokens with source bodies.

**Drill-in path (two modes):**
1. **Separate call (default):** the agent calls `code_read` with the file + line range from the compact response — it only pays for the source it actually needs.
2. **Inline unfold (from mcp-injector):** the agent passes `unfold: ["path1", "glob2"]` in the same search call. The flagged files are served at full resolution within the compact response; all other hits remain compact. Saves a `code_read` round-trip when the agent already knows which files it needs.

## Slicing Notes

- **Why a separate spec:** this is a different concern from the health_check/index_status spec. The status tools are about "is my index fresh enough to trust?" (index state). The compact output is about "how do I get a cheap answer from search?" (response format). Different files touched, different test scenarios, different acceptance criteria. Keeping them separate means each spec can be executed independently and each PR stays under the 400-line budget.

- **Why a parameter, not a new tool:** adding a `context` boolean to the existing tools is cheaper than a new `code_hybrid_search_compact` tool. The agent already knows to call `code_hybrid_search`; it just passes `context: true` when it wants the cheap answer. A new tool would double the tool list and require the agent to learn when to use which.

- **Why the format is name + file + lines + signature, not just name + file:** the line range is what the agent needs to call `code_read` next. Without it, the agent would have to call `code_search` again (full format) to get the line range — defeating the purpose. The signature (one line) disambiguates overloads without paying for the body.

- **"Think in Code" as a future direction (from context-mode, 20K★):** context-mode's most radical pattern — the agent writes a script that does the analysis and returns only the result — maps to a `code_query` tool that takes a structured question ("how many functions call CalculateTotal?", "list all exported functions in package auth") and returns only the answer (a count, a list of names, yes/no), not the source. This is the next step beyond compact output: not "here's the source, cheaper" but "here's the answer, no source at all." Not part of this spec; a candidate for a follow-up spec when the compact output is shipped and the agent's context budget is further constrained.

- **AST-aware compression as a future direction (from headroom, 72K★):** headroom's CodeCompressor keeps *parts* of the function body (the interesting lines — control flow, key statements) and drops the rest, using the AST to decide what's interesting. Our compact format drops the body entirely. If "name + signature" alone is not enough (the agent needs to see the first few lines or the control-flow skeleton to decide whether to drill in), headroom's AST-aware compression is the pattern to follow: keep the interesting lines, drop the boilerplate. Not part of this spec; a candidate for a follow-up when the agent's context budget is further constrained and "name + signature" alone is not enough to make the drill-in decision.

- **Determinism is a hard requirement (from mcp-injector, 2★):** mcp-injector guarantees byte-identical outputs across runs to maximize Anthropic's KV prompt cache hits. Our `FormatCompact` must be equally deterministic: same `[]HybridHit` input → same string output, every time. No timestamps, no `map` iteration order (Go maps are non-deterministic), no `rand`. The hits arrive already sorted by RRF rank, so the ordering is stable. If we ever add a `map[string]` intermediate, we must sort the keys before formatting. The test `TestCompactFormatDeterministic` calls the function twice and asserts byte-identity.

- **`unfold` is a cheap enhancement, not a new tool (from mcp-injector, 2★):** mcp-injector's `get_project_map` takes `unfolded_files` — an array of paths/globs served at full resolution within the compressed response. Our `unfold` parameter does the same: the agent passes `unfold: ["src/auth/handler.go", "**/*_test.go"]` and those files come back with full source while everything else stays compact. It's a slice parameter on the existing tool, not a new tool. The implementation is: after `FormatCompact`, iterate the `unfold` list, match against the hit files, and replace the compact line with the full source for the matched hits. ~30 lines of code, no new MCP tool.
