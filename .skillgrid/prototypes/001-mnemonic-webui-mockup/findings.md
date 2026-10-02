# Prototype 001 Findings

## CORS Fix
- **What:** Added `corsMiddleware` to `skillgrid-cli/internal/mnemonic/http/server.go` Handler() method
- **Why:** Browser-based UIs on any origin were blocked by the Go HTTP server (no Access-Control-Allow-Origin headers)
- **Where:** `server.go:259-277` — 15-line middleware wrapping the mux
- **Learned:** Origin-echo pattern (reflects request Origin) is safe for local dev. Standard methods + Content-Type/Authorization headers. OPTIONS preflight returns 204.

## FTS5 Search Gap
- **What:** `/mnemonic/search?q=<any>&project=skillgrid` returns `{"results":[],"total":0}` for all tested queries
- **Why:** The FTS5 index may not be populated, or the search code path differs from the MCP `mem_search` tool
- **Where:** `skillgrid-cli/internal/mnemonic/http/` — `handleMnemonicSearch` handler
- **Learned:** MCP `mem_search` works fine (returns results). The HTTP endpoint may use a different search path. Needs investigation before the Second Brain feature can work end-to-end.

## Graph Data Volume
- **What:** `/mnemonic/graph/data?limit=500` returns 500 nodes + 2,110 edges = 365KB JSON
- **Why:** Full graph is 4,819 nodes / 15,926 edges — too large for client-side D3 without limits
- **Where:** `skillgrid-cli/internal/mnemonic/http/mnemonic_graph.go`
- **Learned:** D3 v7 handles 500 nodes fine (~3s initial load). The `limit` param is essential for usability.
